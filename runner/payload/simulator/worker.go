package simulator

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"strconv"
	"time"

	"github.com/base/base-bench/runner/network/mempool"
	benchtypes "github.com/base/base-bench/runner/network/types"
	"github.com/base/base-bench/runner/payload/simulator/abi"
	"github.com/base/base-bench/runner/payload/simulator/simulatorstats"
	"github.com/base/base-bench/runner/payload/worker"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"
)

const defaultNumCallers = 1

type Bytecode struct {
	Object string `json:"object"`
}

type Contract struct {
	Bytecode Bytecode `json:"bytecode"`
}

type SimulatorPayloadDefinition = simulatorstats.StatsConfig

type simulatorPayloadWorker struct {
	log          log.Logger
	gasUsedCache map[common.Hash]uint64

	params  benchtypes.RunParams
	chainID *big.Int
	client  *ethclient.Client

	prefundedAccount *ecdsa.PrivateKey
	prefundAmount    *big.Int

	mempool *mempool.StaticWorkloadMempool

	contractAddr common.Address

	// scaleFactor is the factor by which to scale the numCallsPerBlock to match the gas limit
	scaleFactor float64

	payloadParams   *simulatorstats.Stats
	actualNumConfig *simulatorstats.Stats
	numCalls        uint64
	contractBackend *backendWithTrackedNonce

	// Caller accounts for distributing transactions across multiple senders
	callerKeys     []*ecdsa.PrivateKey
	callerAddrs    []common.Address
	transactors    []*bind.TransactOpts
	callTransactor *bind.CallOpts
	currCallerIdx  int

	// setupTransactor is used for deployment and initialization (uses prefunded account)
	setupTransactor *bind.TransactOpts

	numCallsPerBlock uint64
	numCallers       int
}

// backendWithTrackedNonce wraps a ContractBackend and tracks nonces locally
// for multiple addresses, avoiding repeated RPC calls for nonce fetching.
type backendWithTrackedNonce struct {
	bind.ContractBackend
	nonces map[common.Address]uint64
}

func newBackendWithTrackedNonce(backend bind.ContractBackend, addresses []common.Address) (*backendWithTrackedNonce, error) {
	nonces := make(map[common.Address]uint64, len(addresses))
	for _, addr := range addresses {
		nonce, err := backend.PendingNonceAt(context.Background(), addr)
		if err != nil {
			return nil, err
		}
		nonces[addr] = nonce
	}

	return &backendWithTrackedNonce{
		ContractBackend: backend,
		nonces:          nonces,
	}, nil
}

func (t *backendWithTrackedNonce) incrementNonce(addr common.Address) {
	t.nonces[addr]++
}

func (t *backendWithTrackedNonce) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	if nonce, ok := t.nonces[account]; ok {
		return nonce, nil
	}
	return t.ContractBackend.PendingNonceAt(ctx, account)
}

var _ bind.ContractBackend = &backendWithTrackedNonce{}

func NewSimulatorPayloadWorker(ctx context.Context, log log.Logger, elRPCURL string, params benchtypes.RunParams, prefundedPrivateKey ecdsa.PrivateKey, prefundAmount *big.Int, genesis *core.Genesis, payloadParams interface{}) (worker.Worker, error) {
	mempool := mempool.NewStaticWorkloadMempool(log, genesis.Config.ChainID)

	client, err := ethclient.Dial(elRPCURL)
	if err != nil {
		return nil, err
	}

	chainID := genesis.Config.ChainID

	if payloadParams == nil {
		return nil, errors.New("Simulator payload params are required")
	}

	simulatorParams, ok := payloadParams.(*SimulatorPayloadDefinition)
	if !ok {
		return nil, errors.New("Simulator payload params are not valid")
	}

	numCallers := defaultNumCallers
	if simulatorParams.NumCallers != nil && *simulatorParams.NumCallers > 0 {
		numCallers = *simulatorParams.NumCallers
	}

	// Generate caller accounts deterministically from prefunded account
	callerKeys, callerAddrs := generateCallerAccounts(&prefundedPrivateKey, numCallers)

	// Create transactors for each caller
	transactors := make([]*bind.TransactOpts, numCallers)

	for i, key := range callerKeys {
		transactor, err := bind.NewKeyedTransactorWithChainID(key, chainID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create transactor for caller %d", i)
		}
		transactor.NoSend = true
		nonce, err := client.PendingNonceAt(context.Background(), crypto.PubkeyToAddress(key.PublicKey))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get nonce for caller %d", i)
		}
		transactor.Nonce = big.NewInt(int64(nonce))

		transactor.GasFeeCap = new(big.Int).Mul(big.NewInt(1e9), big.NewInt(1))
		transactor.GasTipCap = big.NewInt(1)

		transactors[i] = transactor
	}

	// Initialize nonce tracker with prefunded account (used for deployment/setup)
	// Caller accounts will be added after funding in Setup
	contractBackend, err := newBackendWithTrackedNonce(client, []common.Address{
		crypto.PubkeyToAddress(prefundedPrivateKey.PublicKey),
	})
	if err != nil {
		return nil, err
	}

	// Setup transactor for deployment and initialization
	setupTransactor, err := bind.NewKeyedTransactorWithChainID(&prefundedPrivateKey, chainID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create setup transactor")
	}
	setupTransactor.NoSend = true

	callTransactor := &bind.CallOpts{
		From:    crypto.PubkeyToAddress(prefundedPrivateKey.PublicKey),
		Context: context.Background(),
	}

	scaleFactor := 1.0
	if simulatorParams.AvgGasUsed != nil && simulatorParams.CallsPerBlock != nil && *simulatorParams.CallsPerBlock != "fill" {
		scaleFactor = float64(params.GasLimit) / float64(*simulatorParams.AvgGasUsed)
	}

	t := &simulatorPayloadWorker{
		log:              log,
		client:           client,
		mempool:          mempool,
		params:           params,
		chainID:          chainID,
		prefundedAccount: &prefundedPrivateKey,
		prefundAmount:    prefundAmount,
		payloadParams:    simulatorParams.ToStats(),
		contractBackend:  contractBackend,
		callerKeys:       callerKeys,
		callerAddrs:      callerAddrs,
		transactors:      transactors,
		setupTransactor:  setupTransactor,
		callTransactor:   callTransactor,
		scaleFactor:      scaleFactor,
		actualNumConfig:  simulatorstats.NewStats(),
		numCallers:       numCallers,
		gasUsedCache:     make(map[common.Hash]uint64),
	}

	return t, nil
}

// generateCallerAccounts derives caller accounts deterministically from the prefunded account.
// If numCallers is 1, it returns the prefunded account itself.
func generateCallerAccounts(prefundedKey *ecdsa.PrivateKey, numCallers int) ([]*ecdsa.PrivateKey, []common.Address) {
	if numCallers == 1 {
		return []*ecdsa.PrivateKey{prefundedKey}, []common.Address{crypto.PubkeyToAddress(prefundedKey.PublicKey)}
	}

	// Use deterministic random source seeded from prefunded key
	seed := int64(prefundedKey.D.Uint64())
	src := rand.New(rand.NewSource(seed))

	keys := make([]*ecdsa.PrivateKey, numCallers)
	addrs := make([]common.Address, numCallers)
	for i := 0; i < numCallers; i++ {
		key, err := ecdsa.GenerateKey(crypto.S256(), src)
		if err != nil {
			panic(fmt.Sprintf("failed to generate caller key: %v", err))
		}
		keys[i] = key
		addrs[i] = crypto.PubkeyToAddress(key.PublicKey)
	}
	return keys, addrs
}

func (t *simulatorPayloadWorker) Mempool() mempool.FakeMempool {
	return t.mempool
}

func (t *simulatorPayloadWorker) Stop(ctx context.Context) error {
	// TODO: Implement
	return nil
}

func (t *simulatorPayloadWorker) mineAndConfirm(ctx context.Context, txs []*types.Transaction) error {
	t.mempool.AddTransactions(txs)

	receipt, err := t.waitForReceipt(ctx, txs[len(txs)-1].Hash())
	if err != nil {
		return errors.Wrap(err, "failed to wait for receipt")
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		return fmt.Errorf("receipt status not successful: %d", receipt.Status)
	}

	return nil
}

func (t *simulatorPayloadWorker) deployContract(ctx context.Context) (*abi.Simulator, error) {
	prefundAddr := crypto.PubkeyToAddress(t.prefundedAccount.PublicKey)

	contractAddr, contractDeploymentTx, err := t.createDeployTx(t.prefundedAccount)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create contract deployment transaction")
	}
	t.contractBackend.incrementNonce(prefundAddr)

	t.log.Debug("Contract address", "address", contractAddr.Hex())
	t.contractAddr = *contractAddr

	simulator, err := abi.NewSimulator(t.contractAddr, t.contractBackend)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create simulator transactor")
	}

	if err := t.mineAndConfirm(ctx, []*types.Transaction{contractDeploymentTx}); err != nil {
		return nil, errors.Wrap(err, "failed to mine and confirm contract deployment")
	}

	return simulator, nil
}

// testForBlocks runs the test over 5 blocks and collects max tx gas usage
func (t *simulatorPayloadWorker) testForBlocks(ctx context.Context, simulator *abi.Simulator) error {
	prefundAddr := crypto.PubkeyToAddress(t.prefundedAccount.PublicKey)

	// estimate storage slot usage
	contractConfig, err := t.payloadParams.Mul(float64(t.params.NumBlocks)).ToConfig()
	if err != nil {
		return errors.Wrap(err, "failed to convert payload params to config")
	}

	storageSlotsNeeded, err := simulator.NumStorageSlotsNeeded(t.callTransactor, *contractConfig)
	if err != nil {
		return errors.Wrap(err, "failed to estimate storage slot usage")
	}

	currentStorageSlots, err := simulator.NumStorageInitialized(t.callTransactor)
	if err != nil {
		return errors.Wrap(err, "failed to get current storage slots")
	}

	accountSlotsNeeded, err := simulator.NumAccountsNeeded(t.callTransactor, *contractConfig)
	if err != nil {
		return errors.Wrap(err, "failed to estimate account slot usage")
	}

	currentAccounts, err := simulator.NumAddressInitialized(t.callTransactor)
	if err != nil {
		return errors.Wrap(err, "failed to get current accounts")
	}

	sendCalls := make([]*types.Transaction, 0)

	storageChunks := uint64(math.Ceil(float64(storageSlotsNeeded.Int64()-currentStorageSlots.Int64()) / 100))
	log.Info("Initializing test storage chunks", "storageChunks", storageChunks)
	for i := uint64(0); i < storageChunks; i++ {
		storageChunkTx, err := simulator.InitializeStorageChunk(t.setupTransactor)
		if err != nil {
			return errors.Wrap(err, "failed to initialize storage chunk")
		}
		t.contractBackend.incrementNonce(prefundAddr)

		sendCalls = append(sendCalls, storageChunkTx)
	}

	accountChunks := uint64(math.Ceil(float64(accountSlotsNeeded.Int64()-currentAccounts.Int64()) / 100))
	log.Info("Initializing test account chunks", "accountChunks", accountChunks)
	for i := uint64(0); i < accountChunks; i++ {
		accountChunkTx, err := simulator.InitializeAddressChunk(t.setupTransactor)
		if err != nil {
			return errors.Wrap(err, "failed to initialize account chunk")
		}
		t.contractBackend.incrementNonce(prefundAddr)

		sendCalls = append(sendCalls, accountChunkTx)
	}

	if len(sendCalls) > 0 {
		if err := t.mineAndConfirm(ctx, sendCalls); err != nil {
			return errors.Wrap(err, "failed to mine and confirm storage chunk initialization")
		}
	}

	contractConfig, err = t.payloadParams.ToConfig()
	if err != nil {
		return errors.Wrap(err, "failed to convert payload params to config")
	}

	log.Info("Estimating gas for test run", "run", contractConfig)

	tx, err := simulator.Run(t.setupTransactor, *contractConfig)
	if err != nil {
		return errors.Wrap(err, "failed to run contract")
	}

	gas := tx.Gas()

	// max num calls per block is the gas limit divided by the gas used per call (we'll estimate that here)
	t.numCallsPerBlock = calcNumCalls(gas, t.params.GasLimit, buffer)

	// if the user specifies calls per block, use that if it's under the max
	if t.payloadParams.CallsPerBlock != "fill" {
		f, err := strconv.ParseUint(t.payloadParams.CallsPerBlock, 10, 64)
		if err != nil {
			t.log.Warn("failed to parse calls per block", "err", err, "callsPerBlock", t.payloadParams.CallsPerBlock)
		}

		// callsperblock is the max number of calls per block
		if err == nil && f < t.numCallsPerBlock {
			t.numCallsPerBlock = f
		}
	}

	t.log.Info("Calculated num calls per block", "numCalls", t.numCallsPerBlock, "gas", gas, "gasLimit", t.params.GasLimit, "buffer", buffer)

	configForAllBlocks, err := t.payloadParams.Mul(float64(t.numCallsPerBlock) * float64(t.params.NumBlocks) * t.scaleFactor * 1.05).ToConfig()
	if err != nil {
		return errors.Wrap(err, "failed to convert payload params to config")
	}
	t.log.Info("Calculated config for all blocks", "config", configForAllBlocks)

	storageSlotsNeeded, err = simulator.NumStorageSlotsNeeded(t.callTransactor, *configForAllBlocks)
	if err != nil {
		return errors.Wrap(err, "failed to estimate storage slot usage")
	}

	numExistingStorageSlots, err := simulator.NumStorageInitialized(t.callTransactor)
	if err != nil {
		return errors.Wrap(err, "failed to get number of existing storage slots")
	}

	accountSlotsNeeded, err = simulator.NumAccountsNeeded(t.callTransactor, *configForAllBlocks)
	if err != nil {
		return errors.Wrap(err, "failed to estimate account slot usage")
	}

	currentAccounts, err = simulator.NumAddressInitialized(t.callTransactor)
	if err != nil {
		return errors.Wrap(err, "failed to get current accounts")
	}

	sendCalls = make([]*types.Transaction, 0)

	accountChunks = uint64(math.Ceil(float64(accountSlotsNeeded.Int64()-currentAccounts.Int64()) / 100))
	log.Info("Initializing test account chunks", "accountChunks", accountChunks)
	for i := uint64(0); i < accountChunks; i++ {
		accountChunkTx, err := simulator.InitializeAddressChunk(t.setupTransactor)
		if err != nil {
			return errors.Wrap(err, "failed to initialize account chunk")
		}
		t.contractBackend.incrementNonce(prefundAddr)

		sendCalls = append(sendCalls, accountChunkTx)
	}

	t.log.Info("Setting up storage", "numExistingStorageSlots", numExistingStorageSlots, "storageSlotsNeeded", storageSlotsNeeded)

	additionalStorage := uint64(math.Ceil(float64(storageSlotsNeeded.Int64()-numExistingStorageSlots.Int64()) / 100))
	for i := uint64(0); i < additionalStorage; i++ {
		storageChunkTx, err := simulator.InitializeStorageChunk(t.setupTransactor)
		if err != nil {
			return errors.Wrap(err, "failed to initialize storage chunk")
		}
		t.contractBackend.incrementNonce(prefundAddr)

		sendCalls = append(sendCalls, storageChunkTx)
	}

	if len(sendCalls) > 0 {
		if err := t.mineAndConfirm(ctx, sendCalls); err != nil {
			return errors.Wrap(err, "failed to mine and confirm storage chunk initialization")
		}
	}

	return nil
}

const buffer = 1e6 // 1M gas buffer to start

func calcNumCalls(gasPerTx uint64, gasLimit uint64, buffer uint64) uint64 {
	return (gasLimit - buffer) / gasPerTx
}

func (t *simulatorPayloadWorker) Setup(ctx context.Context) error {
	// check balance > prefundAmount
	balance, err := t.client.BalanceAt(ctx, crypto.PubkeyToAddress(t.prefundedAccount.PublicKey), nil)
	log.Info("Prefunded account balance", "balance", balance.String())
	if err != nil {
		return errors.Wrap(err, "failed to fetch prefunded account balance")
	}

	if balance.Cmp(t.prefundAmount) < 0 {
		return fmt.Errorf("prefunded account balance %s is less than prefund amount %s", balance.String(), t.prefundAmount.String())
	}

	simulator, err := t.deployContract(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to deploy contract")
	}

	err = t.testForBlocks(ctx, simulator)
	if err != nil {
		return errors.Wrap(err, "failed to test for blocks")
	}

	// Fund caller accounts if using multiple callers
	if err := t.fundCallerAccounts(ctx); err != nil {
		return errors.Wrap(err, "failed to fund caller accounts")
	}

	return nil
}

// fundCallerAccounts distributes funds from the prefunded account to caller accounts
// and initializes nonce tracking for all callers.
func (t *simulatorPayloadWorker) fundCallerAccounts(ctx context.Context) error {
	prefundAddr := crypto.PubkeyToAddress(t.prefundedAccount.PublicKey)

	// If using just the prefunded account, no funding transfer needed
	if t.numCallers == 1 && t.callerAddrs[0] == prefundAddr {
		t.log.Info("Using single caller (prefunded account)", "address", prefundAddr.Hex())
		pendingNonce, err := t.client.PendingNonceAt(ctx, prefundAddr)
		if err != nil {
			return errors.Wrap(err, "failed to get pending nonce for prefunded account")
		}
		t.transactors[0].Nonce = big.NewInt(int64(pendingNonce))
		return nil
	}

	t.log.Info("Funding caller accounts", "numCallers", t.numCallers)

	// Get current balance of prefunded account
	balance, err := t.client.BalanceAt(ctx, prefundAddr, nil)
	if err != nil {
		return errors.Wrap(err, "failed to get prefunded account balance")
	}

	// Reserve some funds for gas costs (21000 gas per transfer * gas price)
	gasCostPerTransfer := new(big.Int).Mul(big.NewInt(21000), big.NewInt(1e9)) // 21000 * 1 gwei
	totalGasCost := new(big.Int).Mul(gasCostPerTransfer, big.NewInt(int64(t.numCallers)))

	// Distribute remaining balance equally among callers
	distributableBalance := new(big.Int).Sub(balance, totalGasCost)
	if distributableBalance.Sign() <= 0 {
		return fmt.Errorf("insufficient balance to fund caller accounts: balance=%s, gasCost=%s",
			balance.String(), totalGasCost.String())
	}

	perCallerAmount := new(big.Int).Div(distributableBalance, big.NewInt(int64(t.numCallers)))
	t.log.Info("Funding each caller", "amount", perCallerAmount.String(), "numCallers", t.numCallers)

	// Create funding transactions
	fundingTxs := make([]*types.Transaction, 0, t.numCallers)
	nonce := t.contractBackend.nonces[prefundAddr]

	signer := types.NewPragueSigner(t.chainID)
	for _, callerAddr := range t.callerAddrs {
		// Skip if caller is the prefunded account
		if callerAddr == prefundAddr {
			continue
		}

		txData := &types.DynamicFeeTx{
			ChainID:   t.chainID,
			Nonce:     nonce,
			To:        &callerAddr,
			Gas:       21000,
			GasFeeCap: new(big.Int).Mul(big.NewInt(1e9), big.NewInt(1)), // 1 gwei
			GasTipCap: big.NewInt(2),
			Value:     perCallerAmount,
		}
		tx := types.MustSignNewTx(t.prefundedAccount, signer, txData)
		fundingTxs = append(fundingTxs, tx)
		nonce++
	}

	if len(fundingTxs) == 0 {
		return nil
	}

	// Mine and confirm funding transactions
	if err := t.mineAndConfirm(ctx, fundingTxs); err != nil {
		return errors.Wrap(err, "failed to mine and confirm funding transactions")
	}

	// Update nonce tracker for prefunded account
	t.contractBackend.nonces[prefundAddr] = nonce

	// Initialize nonce tracking for all caller accounts
	for _, callerAddr := range t.callerAddrs {
		if callerAddr == prefundAddr {
			continue
		}
		callerNonce, err := t.client.PendingNonceAt(ctx, callerAddr)
		if err != nil {
			return errors.Wrapf(err, "failed to get nonce for caller %s", callerAddr.Hex())
		}
		t.contractBackend.nonces[callerAddr] = callerNonce
	}

	t.log.Info("Successfully funded caller accounts", "numCallers", t.numCallers)
	return nil
}

func (t *simulatorPayloadWorker) waitForReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	return retry.Do(ctx, 240, retry.Fixed(1*time.Second), func() (*types.Receipt, error) {
		receipt, err := t.client.TransactionReceipt(ctx, txHash)
		if err != nil {
			return nil, err
		}
		return receipt, nil
	})
}

func (t *simulatorPayloadWorker) sendTxs(ctx context.Context) error {
	txs := make([]*types.Transaction, 0, t.numCallers)

	gas := t.params.GasLimit - 100_000

	sendTxsStartTime := time.Now()

	for i := uint64(0); i < uint64(math.Ceil(float64(t.numCallsPerBlock)*t.scaleFactor)); i++ {
		actual := t.actualNumConfig
		expected := t.payloadParams.Mul(float64(t.numCalls+1) * t.scaleFactor)

		// Round-robin across callers
		callerIdx := t.currCallerIdx
		t.currCallerIdx = (t.currCallerIdx + 1) % t.numCallers
		blockCounts := expected.Sub(actual).Round()

		expectedGas, ok := t.gasUsedCache[blockCounts.Hash()]
		if ok {
			t.transactors[callerIdx].GasLimit = expectedGas
		} else {
			t.transactors[callerIdx].GasLimit = 0
		}

		transferTx, err := t.createCallTx(t.transactors[callerIdx], t.callerKeys[callerIdx], blockCounts)
		if err != nil {
			t.log.Error("Failed to create transfer transaction", "err", err)
			return err
		}

		t.gasUsedCache[blockCounts.Hash()] = transferTx.Gas()

		t.transactors[callerIdx].Nonce = t.transactors[callerIdx].Nonce.Add(t.transactors[callerIdx].Nonce, big.NewInt(1))

		gasUsed := transferTx.Gas()
		if gasUsed > gas {
			t.log.Warn("Gas used is greater than gas limit, stopping tx sending", "gasUsed", gasUsed, "gasLimit", t.params.GasLimit)
			break
		}

		t.contractBackend.incrementNonce(t.callerAddrs[callerIdx])

		gas -= gasUsed

		txs = append(txs, transferTx)

		t.actualNumConfig = t.actualNumConfig.Add(blockCounts)
		t.numCalls++
	}

	t.mempool.AddTransactions(txs)
	sendTxsDuration := time.Since(sendTxsStartTime)
	log.Info("Send transactions duration", "duration", sendTxsDuration, "numCalls", uint64(math.Ceil(float64(t.numCallsPerBlock)*t.scaleFactor)))
	return nil
}

func (t *simulatorPayloadWorker) createCallTx(transactor *bind.TransactOpts, fromPriv *ecdsa.PrivateKey, config *simulatorstats.Stats) (*types.Transaction, error) {
	simulator, err := abi.NewSimulatorTransactor(t.contractAddr, t.contractBackend)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create simulator transactor")
	}

	contractConfig, err := t.payloadParams.ToConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert payload params to config")
	}

	return simulator.Run(transactor, *contractConfig)
}

func (t *simulatorPayloadWorker) createDeployTx(fromPriv *ecdsa.PrivateKey) (*common.Address, *types.Transaction, error) {

	transactor, err := bind.NewKeyedTransactorWithChainID(fromPriv, t.chainID)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create transactor")
	}
	transactor.NoSend = true
	transactor.GasLimit = t.params.GasLimit / 2
	transactor.Value = new(big.Int).Div(t.prefundAmount, big.NewInt(2))

	rand64 := rand.Uint64()

	deployAddr, deployTx, _, err := abi.DeploySimulator(transactor, t.contractBackend, new(big.Int).SetUint64(rand64))
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to deploy simulator")
	}

	return &deployAddr, deployTx, nil
}

func (t *simulatorPayloadWorker) SendTxs(ctx context.Context) error {
	if err := t.sendTxs(ctx); err != nil {
		t.log.Error("Failed to send transactions", "err", err)
		return err
	}
	return nil
}
