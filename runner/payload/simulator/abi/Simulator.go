// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package abi

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// PrecompileConfig is an auto generated low-level Go binding around an user-defined struct.
type PrecompileConfig struct {
	PrecompileAddress common.Address
	NumCalls          *big.Int
}

// SimulatorConfig is an auto generated low-level Go binding around an user-defined struct.
type SimulatorConfig struct {
	LoadAccounts   *big.Int
	UpdateAccounts *big.Int
	DeleteAccounts *big.Int
	CreateAccounts *big.Int
	LoadStorage    *big.Int
	UpdateStorage  *big.Int
	DeleteStorage  *big.Int
	CreateStorage  *big.Int
	Precompiles    []PrecompileConfig
}

// SimulatorMetaData contains all meta data concerning the Simulator contract.
var SimulatorMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"_storage_slots_to_initialize\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_addresses_to_initialize\",\"type\":\"uint160\",\"internalType\":\"uint160\"},{\"name\":\"_storage_chunk_size\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_addresses_chunk_size\",\"type\":\"uint160\",\"internalType\":\"uint160\"}],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"fully_initialized\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"initialize_address_chunk\",\"inputs\":[{\"name\":\"chunk_index\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"initialize_storage_chunk\",\"inputs\":[{\"name\":\"chunk_index\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"num_address_chunks\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"num_storage_chunks\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"run\",\"inputs\":[{\"name\":\"config\",\"type\":\"tuple\",\"internalType\":\"structSimulatorConfig\",\"components\":[{\"name\":\"load_accounts\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"update_accounts\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"delete_accounts\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"create_accounts\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"load_storage\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"update_storage\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"delete_storage\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"create_storage\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"precompiles\",\"type\":\"tuple[]\",\"internalType\":\"structPrecompileConfig[]\",\"components\":[{\"name\":\"precompile_address\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"num_calls\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]}]}],\"outputs\":[],\"stateMutability\":\"nonpayable\"}]",
	Bin: "0x60806040525f6005555f6006555f6007555f600855604051610c70380380610c708339818101604052810190610035919061014b565b835f819055508260015f6101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550816002819055508060035f6101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550505050506101af565b5f80fd5b5f819050919050565b6100e1816100cf565b81146100eb575f80fd5b50565b5f815190506100fc816100d8565b92915050565b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b61012a81610102565b8114610134575f80fd5b50565b5f8151905061014581610121565b92915050565b5f805f8060808587031215610163576101626100cb565b5b5f610170878288016100ee565b945050602061018187828801610137565b9350506040610192878288016100ee565b92505060606101a387828801610137565b91505092959194509250565b610ab4806101bc5f395ff3fe608060405234801561000f575f80fd5b5060043610610060575f3560e01c8063359e53e3146100645780634e125b0c1461008257806363b1845f146100a0578063746d124a146100bc578063c1d61ec4146100d8578063d845ebb1146100f4575b5f80fd5b61006c610112565b60405161007991906105f7565b60405180910390f35b61008a610127565b604051610097919061062a565b60405180910390f35b6100ba60048036038101906100b59190610675565b610177565b005b6100d660048036038101906100d19190610675565b6101e9565b005b6100f260048036038101906100ed91906106c3565b610329565b005b6100fc610571565b60405161010991906105f7565b60405180910390f35b5f6002545f546101229190610764565b905090565b5f8054600554148015610172575060015f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16600654145b905090565b5f600254826101869190610794565b90505f6002548261019791906107d5565b90505f8290505b818110156101c9578060045f8381526020019081526020015f2081905550808060010191505061019e565b5060025460055f8282546101dd91906107d5565b92505081905550505050565b5f60035f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff16826102179190610827565b90505f60035f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff16826102479190610868565b90505f8290505b8173ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff1610156102d4578073ffffffffffffffffffffffffffffffffffffffff166108fc600190811502906040515f60405180830381858888f193505050501580156102c6573d5f803e3d5ffd5b50808060010191505061024e565b5060035f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1660065f82825461031d91906107d5565b92505081905550505050565b6005548160a00135826080013560085461034391906107d5565b61034d91906107d5565b111561038e576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016103859061092f565b60405180910390fd5b5f60085490505b81608001356008546103a791906107d5565b8110156103c2575f6008549050508080600101915050610395565b50806080013560085f8282546103d891906107d5565b925050819055505f60085490505b8160a001356008546103f891906107d5565b811015610416575f60085490508081555080806001019150506103e6565b508060a0013560085f82825461042c91906107d5565b925050819055505f60055490505b8160e0013560055461044c91906107d5565b811015610468575f81905080815550808060010191505061043a565b508060e0013560055f82825461047e91906107d5565b925050819055505f60075490505b8160c0013560075461049e91906107d5565b8110156104ba575f8190505f815550808060010191505061048c565b508060c0013560075f8282546104d091906107d5565b925050819055505f5b818061010001906104ea9190610959565b905081101561056d57610560828061010001906105079190610959565b83818110610518576105176109bb565b5b9050604002015f01602081019061052f9190610a23565b838061010001906105409190610959565b84818110610551576105506109bb565b5b905060400201602001356105db565b80806001019150506104d9565b5050565b5f60035f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1660015f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff166105c09190610a4e565b73ffffffffffffffffffffffffffffffffffffffff16905090565b5050565b5f819050919050565b6105f1816105df565b82525050565b5f60208201905061060a5f8301846105e8565b92915050565b5f8115159050919050565b61062481610610565b82525050565b5f60208201905061063d5f83018461061b565b92915050565b5f80fd5b5f80fd5b610654816105df565b811461065e575f80fd5b50565b5f8135905061066f8161064b565b92915050565b5f6020828403121561068a57610689610643565b5b5f61069784828501610661565b91505092915050565b5f80fd5b5f61012082840312156106ba576106b96106a0565b5b81905092915050565b5f602082840312156106d8576106d7610643565b5b5f82013567ffffffffffffffff8111156106f5576106f4610647565b5b610701848285016106a4565b91505092915050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601260045260245ffd5b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b5f61076e826105df565b9150610779836105df565b9250826107895761078861070a565b5b828204905092915050565b5f61079e826105df565b91506107a9836105df565b92508282026107b7816105df565b915082820484148315176107ce576107cd610737565b5b5092915050565b5f6107df826105df565b91506107ea836105df565b925082820190508082111561080257610801610737565b5b92915050565b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f61083182610808565b915061083c83610808565b925082820261084a81610808565b9150828204841483151761086157610860610737565b5b5092915050565b5f61087282610808565b915061087d83610808565b9250828201905073ffffffffffffffffffffffffffffffffffffffff8111156108a9576108a8610737565b5b92915050565b5f82825260208201905092915050565b7f4e6f7420656e6f7567682073746f7261676520736c6f747320746f206c6f61645f8201527f2f75706461746500000000000000000000000000000000000000000000000000602082015250565b5f6109196027836108af565b9150610924826108bf565b604082019050919050565b5f6020820190508181035f8301526109468161090d565b9050919050565b5f80fd5b5f80fd5b5f80fd5b5f80833560016020038436030381126109755761097461094d565b5b80840192508235915067ffffffffffffffff82111561099757610996610951565b5b6020830192506040820236038313156109b3576109b2610955565b5b509250929050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52603260045260245ffd5b5f6109f282610808565b9050919050565b610a02816109e8565b8114610a0c575f80fd5b50565b5f81359050610a1d816109f9565b92915050565b5f60208284031215610a3857610a37610643565b5b5f610a4584828501610a0f565b91505092915050565b5f610a5882610808565b9150610a6383610808565b925082610a7357610a7261070a565b5b82820490509291505056fea26469706673582212202d69a4afdb0f60cb6529cab140ab23b9df8f170e5a59ea2c805826fcc5960a8c64736f6c63430008190033",
}

// SimulatorABI is the input ABI used to generate the binding from.
// Deprecated: Use SimulatorMetaData.ABI instead.
var SimulatorABI = SimulatorMetaData.ABI

// SimulatorBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use SimulatorMetaData.Bin instead.
var SimulatorBin = SimulatorMetaData.Bin

// DeploySimulator deploys a new Ethereum contract, binding an instance of Simulator to it.
func DeploySimulator(auth *bind.TransactOpts, backend bind.ContractBackend, _storage_slots_to_initialize *big.Int, _addresses_to_initialize *big.Int, _storage_chunk_size *big.Int, _addresses_chunk_size *big.Int) (common.Address, *types.Transaction, *Simulator, error) {
	parsed, err := SimulatorMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(SimulatorBin), backend, _storage_slots_to_initialize, _addresses_to_initialize, _storage_chunk_size, _addresses_chunk_size)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Simulator{SimulatorCaller: SimulatorCaller{contract: contract}, SimulatorTransactor: SimulatorTransactor{contract: contract}, SimulatorFilterer: SimulatorFilterer{contract: contract}}, nil
}

// Simulator is an auto generated Go binding around an Ethereum contract.
type Simulator struct {
	SimulatorCaller     // Read-only binding to the contract
	SimulatorTransactor // Write-only binding to the contract
	SimulatorFilterer   // Log filterer for contract events
}

// SimulatorCaller is an auto generated read-only Go binding around an Ethereum contract.
type SimulatorCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SimulatorTransactor is an auto generated write-only Go binding around an Ethereum contract.
type SimulatorTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SimulatorFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type SimulatorFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SimulatorSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type SimulatorSession struct {
	Contract     *Simulator        // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// SimulatorCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type SimulatorCallerSession struct {
	Contract *SimulatorCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts    // Call options to use throughout this session
}

// SimulatorTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type SimulatorTransactorSession struct {
	Contract     *SimulatorTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// SimulatorRaw is an auto generated low-level Go binding around an Ethereum contract.
type SimulatorRaw struct {
	Contract *Simulator // Generic contract binding to access the raw methods on
}

// SimulatorCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type SimulatorCallerRaw struct {
	Contract *SimulatorCaller // Generic read-only contract binding to access the raw methods on
}

// SimulatorTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type SimulatorTransactorRaw struct {
	Contract *SimulatorTransactor // Generic write-only contract binding to access the raw methods on
}

// NewSimulator creates a new instance of Simulator, bound to a specific deployed contract.
func NewSimulator(address common.Address, backend bind.ContractBackend) (*Simulator, error) {
	contract, err := bindSimulator(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Simulator{SimulatorCaller: SimulatorCaller{contract: contract}, SimulatorTransactor: SimulatorTransactor{contract: contract}, SimulatorFilterer: SimulatorFilterer{contract: contract}}, nil
}

// NewSimulatorCaller creates a new read-only instance of Simulator, bound to a specific deployed contract.
func NewSimulatorCaller(address common.Address, caller bind.ContractCaller) (*SimulatorCaller, error) {
	contract, err := bindSimulator(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &SimulatorCaller{contract: contract}, nil
}

// NewSimulatorTransactor creates a new write-only instance of Simulator, bound to a specific deployed contract.
func NewSimulatorTransactor(address common.Address, transactor bind.ContractTransactor) (*SimulatorTransactor, error) {
	contract, err := bindSimulator(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &SimulatorTransactor{contract: contract}, nil
}

// NewSimulatorFilterer creates a new log filterer instance of Simulator, bound to a specific deployed contract.
func NewSimulatorFilterer(address common.Address, filterer bind.ContractFilterer) (*SimulatorFilterer, error) {
	contract, err := bindSimulator(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &SimulatorFilterer{contract: contract}, nil
}

// bindSimulator binds a generic wrapper to an already deployed contract.
func bindSimulator(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := SimulatorMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Simulator *SimulatorRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Simulator.Contract.SimulatorCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Simulator *SimulatorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Simulator.Contract.SimulatorTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Simulator *SimulatorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Simulator.Contract.SimulatorTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Simulator *SimulatorCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Simulator.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Simulator *SimulatorTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Simulator.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Simulator *SimulatorTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Simulator.Contract.contract.Transact(opts, method, params...)
}

// FullyInitialized is a free data retrieval call binding the contract method 0x4e125b0c.
//
// Solidity: function fully_initialized() view returns(bool)
func (_Simulator *SimulatorCaller) FullyInitialized(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _Simulator.contract.Call(opts, &out, "fully_initialized")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// FullyInitialized is a free data retrieval call binding the contract method 0x4e125b0c.
//
// Solidity: function fully_initialized() view returns(bool)
func (_Simulator *SimulatorSession) FullyInitialized() (bool, error) {
	return _Simulator.Contract.FullyInitialized(&_Simulator.CallOpts)
}

// FullyInitialized is a free data retrieval call binding the contract method 0x4e125b0c.
//
// Solidity: function fully_initialized() view returns(bool)
func (_Simulator *SimulatorCallerSession) FullyInitialized() (bool, error) {
	return _Simulator.Contract.FullyInitialized(&_Simulator.CallOpts)
}

// NumAddressChunks is a free data retrieval call binding the contract method 0xd845ebb1.
//
// Solidity: function num_address_chunks() view returns(uint256)
func (_Simulator *SimulatorCaller) NumAddressChunks(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Simulator.contract.Call(opts, &out, "num_address_chunks")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// NumAddressChunks is a free data retrieval call binding the contract method 0xd845ebb1.
//
// Solidity: function num_address_chunks() view returns(uint256)
func (_Simulator *SimulatorSession) NumAddressChunks() (*big.Int, error) {
	return _Simulator.Contract.NumAddressChunks(&_Simulator.CallOpts)
}

// NumAddressChunks is a free data retrieval call binding the contract method 0xd845ebb1.
//
// Solidity: function num_address_chunks() view returns(uint256)
func (_Simulator *SimulatorCallerSession) NumAddressChunks() (*big.Int, error) {
	return _Simulator.Contract.NumAddressChunks(&_Simulator.CallOpts)
}

// NumStorageChunks is a free data retrieval call binding the contract method 0x359e53e3.
//
// Solidity: function num_storage_chunks() view returns(uint256)
func (_Simulator *SimulatorCaller) NumStorageChunks(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Simulator.contract.Call(opts, &out, "num_storage_chunks")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// NumStorageChunks is a free data retrieval call binding the contract method 0x359e53e3.
//
// Solidity: function num_storage_chunks() view returns(uint256)
func (_Simulator *SimulatorSession) NumStorageChunks() (*big.Int, error) {
	return _Simulator.Contract.NumStorageChunks(&_Simulator.CallOpts)
}

// NumStorageChunks is a free data retrieval call binding the contract method 0x359e53e3.
//
// Solidity: function num_storage_chunks() view returns(uint256)
func (_Simulator *SimulatorCallerSession) NumStorageChunks() (*big.Int, error) {
	return _Simulator.Contract.NumStorageChunks(&_Simulator.CallOpts)
}

// InitializeAddressChunk is a paid mutator transaction binding the contract method 0x746d124a.
//
// Solidity: function initialize_address_chunk(uint256 chunk_index) returns()
func (_Simulator *SimulatorTransactor) InitializeAddressChunk(opts *bind.TransactOpts, chunk_index *big.Int) (*types.Transaction, error) {
	return _Simulator.contract.Transact(opts, "initialize_address_chunk", chunk_index)
}

// InitializeAddressChunk is a paid mutator transaction binding the contract method 0x746d124a.
//
// Solidity: function initialize_address_chunk(uint256 chunk_index) returns()
func (_Simulator *SimulatorSession) InitializeAddressChunk(chunk_index *big.Int) (*types.Transaction, error) {
	return _Simulator.Contract.InitializeAddressChunk(&_Simulator.TransactOpts, chunk_index)
}

// InitializeAddressChunk is a paid mutator transaction binding the contract method 0x746d124a.
//
// Solidity: function initialize_address_chunk(uint256 chunk_index) returns()
func (_Simulator *SimulatorTransactorSession) InitializeAddressChunk(chunk_index *big.Int) (*types.Transaction, error) {
	return _Simulator.Contract.InitializeAddressChunk(&_Simulator.TransactOpts, chunk_index)
}

// InitializeStorageChunk is a paid mutator transaction binding the contract method 0x63b1845f.
//
// Solidity: function initialize_storage_chunk(uint256 chunk_index) returns()
func (_Simulator *SimulatorTransactor) InitializeStorageChunk(opts *bind.TransactOpts, chunk_index *big.Int) (*types.Transaction, error) {
	return _Simulator.contract.Transact(opts, "initialize_storage_chunk", chunk_index)
}

// InitializeStorageChunk is a paid mutator transaction binding the contract method 0x63b1845f.
//
// Solidity: function initialize_storage_chunk(uint256 chunk_index) returns()
func (_Simulator *SimulatorSession) InitializeStorageChunk(chunk_index *big.Int) (*types.Transaction, error) {
	return _Simulator.Contract.InitializeStorageChunk(&_Simulator.TransactOpts, chunk_index)
}

// InitializeStorageChunk is a paid mutator transaction binding the contract method 0x63b1845f.
//
// Solidity: function initialize_storage_chunk(uint256 chunk_index) returns()
func (_Simulator *SimulatorTransactorSession) InitializeStorageChunk(chunk_index *big.Int) (*types.Transaction, error) {
	return _Simulator.Contract.InitializeStorageChunk(&_Simulator.TransactOpts, chunk_index)
}

// Run is a paid mutator transaction binding the contract method 0xc1d61ec4.
//
// Solidity: function run((uint256,uint256,uint256,uint256,uint256,uint256,uint256,uint256,(address,uint256)[]) config) returns()
func (_Simulator *SimulatorTransactor) Run(opts *bind.TransactOpts, config SimulatorConfig) (*types.Transaction, error) {
	return _Simulator.contract.Transact(opts, "run", config)
}

// Run is a paid mutator transaction binding the contract method 0xc1d61ec4.
//
// Solidity: function run((uint256,uint256,uint256,uint256,uint256,uint256,uint256,uint256,(address,uint256)[]) config) returns()
func (_Simulator *SimulatorSession) Run(config SimulatorConfig) (*types.Transaction, error) {
	return _Simulator.Contract.Run(&_Simulator.TransactOpts, config)
}

// Run is a paid mutator transaction binding the contract method 0xc1d61ec4.
//
// Solidity: function run((uint256,uint256,uint256,uint256,uint256,uint256,uint256,uint256,(address,uint256)[]) config) returns()
func (_Simulator *SimulatorTransactorSession) Run(config SimulatorConfig) (*types.Transaction, error) {
	return _Simulator.Contract.Run(&_Simulator.TransactOpts, config)
}
