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
	ABI: "[{\"type\":\"constructor\",\"inputs\":[],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"initialize_address_chunk\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"initialize_storage_chunk\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"num_address_initialized\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint160\",\"internalType\":\"uint160\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"num_storage_deleted\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"num_storage_initialized\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"num_storage_slots_needed\",\"inputs\":[{\"name\":\"config\",\"type\":\"tuple\",\"internalType\":\"structSimulatorConfig\",\"components\":[{\"name\":\"load_accounts\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"update_accounts\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"delete_accounts\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"create_accounts\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"load_storage\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"update_storage\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"delete_storage\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"create_storage\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"precompiles\",\"type\":\"tuple[]\",\"internalType\":\"structPrecompileConfig[]\",\"components\":[{\"name\":\"precompile_address\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"num_calls\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]}]}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"run\",\"inputs\":[{\"name\":\"config\",\"type\":\"tuple\",\"internalType\":\"structSimulatorConfig\",\"components\":[{\"name\":\"load_accounts\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"update_accounts\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"delete_accounts\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"create_accounts\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"load_storage\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"update_storage\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"delete_storage\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"create_storage\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"precompiles\",\"type\":\"tuple[]\",\"internalType\":\"structPrecompileConfig[]\",\"components\":[{\"name\":\"precompile_address\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"num_calls\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]}]}],\"outputs\":[],\"stateMutability\":\"nonpayable\"}]",
	Bin: "0x60806040526127106001555f60025f6101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055506127106003556127106004556108de806100635f395ff3fe608060405234801561000f575f5ffd5b506004361061007b575f3560e01c8063c1d61ec411610059578063c1d61ec4146100b1578063e2b5a25c146100cd578063e41c3c69146100eb578063ee2bb32b1461011b5761007b565b80633594dea61461007f57806339aa1ab9146100895780634e83a9d5146100a7575b5f5ffd5b610087610139565b005b610091610282565b60405161009e9190610590565b60405180910390f35b6100af610288565b005b6100cb60048036038101906100c691906105d4565b6102ed565b005b6100d5610520565b6040516100e29190610649565b60405180910390f35b610105600480360381019061010091906105d4565b610545565b6040516101129190610590565b60405180910390f35b61012361056e565b6040516101309190610590565b60405180910390f35b5f60025f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1690505f606460025f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1661018d919061068f565b90505f5f8390505b8273ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff16101561020d578073ffffffffffffffffffffffffffffffffffffffff166108fc600190811502906040515f60405180830381858888f1935050505091508080600101915050610195565b50606460025f8282829054906101000a900473ffffffffffffffffffffffffffffffffffffffff1661023f919061068f565b92506101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550505050565b60035481565b5f60015490505f606460015461029e91906106d6565b90505f8290505b818110156102cf57805f5f8381526020019081526020015f208190555080806001019150506102a5565b50606460015f8282546102e291906106d6565b925050819055505050565b6001548160a00135826080013560045461030791906106d6565b61031191906106d6565b1115610352576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161034990610789565b60405180910390fd5b5f60045490505b816080013560045461036b91906106d6565b81101561037f578080600101915050610359565b50806080013560045f82825461039591906106d6565b925050819055505f60045490505b8160a001356004546103b591906106d6565b8110156103cc5780815580806001019150506103a3565b508060a0013560045f8282546103e291906106d6565b925050819055505f60015490505b8160e0013560015461040291906106d6565b8110156104195780815580806001019150506103f0565b508060e0013560015f82825461042f91906106d6565b925050819055505f60035490505b8160c0013560035461044f91906106d6565b811015610466575f8155808060010191505061043d565b508060c0013560035f82825461047c91906106d6565b925050819055505f5f90505b8180610100019061049991906107b3565b905081101561051c5761050f828061010001906104b691906107b3565b838181106104c7576104c6610815565b5b9050604002015f0160208101906104de919061087d565b838061010001906104ef91906107b3565b84818110610500576104ff610815565b5b90506040020160200135610574565b8080600101915050610488565b5050565b60025f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b5f8160a00135826080013560045461055d91906106d6565b61056791906106d6565b9050919050565b60015481565b5050565b5f819050919050565b61058a81610578565b82525050565b5f6020820190506105a35f830184610581565b92915050565b5f5ffd5b5f5ffd5b5f5ffd5b5f61012082840312156105cb576105ca6105b1565b5b81905092915050565b5f602082840312156105e9576105e86105a9565b5b5f82013567ffffffffffffffff811115610606576106056105ad565b5b610612848285016105b5565b91505092915050565b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b6106438161061b565b82525050565b5f60208201905061065c5f83018461063a565b92915050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b5f6106998261061b565b91506106a48361061b565b9250828201905073ffffffffffffffffffffffffffffffffffffffff8111156106d0576106cf610662565b5b92915050565b5f6106e082610578565b91506106eb83610578565b925082820190508082111561070357610702610662565b5b92915050565b5f82825260208201905092915050565b7f4e6f7420656e6f7567682073746f7261676520736c6f747320746f206c6f61645f8201527f2f75706461746500000000000000000000000000000000000000000000000000602082015250565b5f610773602783610709565b915061077e82610719565b604082019050919050565b5f6020820190508181035f8301526107a081610767565b9050919050565b5f5ffd5b5f5ffd5b5f5ffd5b5f5f833560016020038436030381126107cf576107ce6107a7565b5b80840192508235915067ffffffffffffffff8211156107f1576107f06107ab565b5b60208301925060408202360383131561080d5761080c6107af565b5b509250929050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52603260045260245ffd5b5f61084c8261061b565b9050919050565b61085c81610842565b8114610866575f5ffd5b50565b5f8135905061087781610853565b92915050565b5f60208284031215610892576108916105a9565b5b5f61089f84828501610869565b9150509291505056fea264697066735822122069a9552f9ef31be597a5876feb077aa89467acfb11ed0609d58a31087123a1f264736f6c634300081e0033",
}

// SimulatorABI is the input ABI used to generate the binding from.
// Deprecated: Use SimulatorMetaData.ABI instead.
var SimulatorABI = SimulatorMetaData.ABI

// SimulatorBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use SimulatorMetaData.Bin instead.
var SimulatorBin = SimulatorMetaData.Bin

// DeploySimulator deploys a new Ethereum contract, binding an instance of Simulator to it.
func DeploySimulator(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Simulator, error) {
	parsed, err := SimulatorMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(SimulatorBin), backend)
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

// NumAddressInitialized is a free data retrieval call binding the contract method 0xe2b5a25c.
//
// Solidity: function num_address_initialized() view returns(uint160)
func (_Simulator *SimulatorCaller) NumAddressInitialized(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Simulator.contract.Call(opts, &out, "num_address_initialized")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// NumAddressInitialized is a free data retrieval call binding the contract method 0xe2b5a25c.
//
// Solidity: function num_address_initialized() view returns(uint160)
func (_Simulator *SimulatorSession) NumAddressInitialized() (*big.Int, error) {
	return _Simulator.Contract.NumAddressInitialized(&_Simulator.CallOpts)
}

// NumAddressInitialized is a free data retrieval call binding the contract method 0xe2b5a25c.
//
// Solidity: function num_address_initialized() view returns(uint160)
func (_Simulator *SimulatorCallerSession) NumAddressInitialized() (*big.Int, error) {
	return _Simulator.Contract.NumAddressInitialized(&_Simulator.CallOpts)
}

// NumStorageDeleted is a free data retrieval call binding the contract method 0x39aa1ab9.
//
// Solidity: function num_storage_deleted() view returns(uint256)
func (_Simulator *SimulatorCaller) NumStorageDeleted(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Simulator.contract.Call(opts, &out, "num_storage_deleted")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// NumStorageDeleted is a free data retrieval call binding the contract method 0x39aa1ab9.
//
// Solidity: function num_storage_deleted() view returns(uint256)
func (_Simulator *SimulatorSession) NumStorageDeleted() (*big.Int, error) {
	return _Simulator.Contract.NumStorageDeleted(&_Simulator.CallOpts)
}

// NumStorageDeleted is a free data retrieval call binding the contract method 0x39aa1ab9.
//
// Solidity: function num_storage_deleted() view returns(uint256)
func (_Simulator *SimulatorCallerSession) NumStorageDeleted() (*big.Int, error) {
	return _Simulator.Contract.NumStorageDeleted(&_Simulator.CallOpts)
}

// NumStorageInitialized is a free data retrieval call binding the contract method 0xee2bb32b.
//
// Solidity: function num_storage_initialized() view returns(uint256)
func (_Simulator *SimulatorCaller) NumStorageInitialized(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Simulator.contract.Call(opts, &out, "num_storage_initialized")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// NumStorageInitialized is a free data retrieval call binding the contract method 0xee2bb32b.
//
// Solidity: function num_storage_initialized() view returns(uint256)
func (_Simulator *SimulatorSession) NumStorageInitialized() (*big.Int, error) {
	return _Simulator.Contract.NumStorageInitialized(&_Simulator.CallOpts)
}

// NumStorageInitialized is a free data retrieval call binding the contract method 0xee2bb32b.
//
// Solidity: function num_storage_initialized() view returns(uint256)
func (_Simulator *SimulatorCallerSession) NumStorageInitialized() (*big.Int, error) {
	return _Simulator.Contract.NumStorageInitialized(&_Simulator.CallOpts)
}

// NumStorageSlotsNeeded is a free data retrieval call binding the contract method 0xe41c3c69.
//
// Solidity: function num_storage_slots_needed((uint256,uint256,uint256,uint256,uint256,uint256,uint256,uint256,(address,uint256)[]) config) view returns(uint256)
func (_Simulator *SimulatorCaller) NumStorageSlotsNeeded(opts *bind.CallOpts, config SimulatorConfig) (*big.Int, error) {
	var out []interface{}
	err := _Simulator.contract.Call(opts, &out, "num_storage_slots_needed", config)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// NumStorageSlotsNeeded is a free data retrieval call binding the contract method 0xe41c3c69.
//
// Solidity: function num_storage_slots_needed((uint256,uint256,uint256,uint256,uint256,uint256,uint256,uint256,(address,uint256)[]) config) view returns(uint256)
func (_Simulator *SimulatorSession) NumStorageSlotsNeeded(config SimulatorConfig) (*big.Int, error) {
	return _Simulator.Contract.NumStorageSlotsNeeded(&_Simulator.CallOpts, config)
}

// NumStorageSlotsNeeded is a free data retrieval call binding the contract method 0xe41c3c69.
//
// Solidity: function num_storage_slots_needed((uint256,uint256,uint256,uint256,uint256,uint256,uint256,uint256,(address,uint256)[]) config) view returns(uint256)
func (_Simulator *SimulatorCallerSession) NumStorageSlotsNeeded(config SimulatorConfig) (*big.Int, error) {
	return _Simulator.Contract.NumStorageSlotsNeeded(&_Simulator.CallOpts, config)
}

// InitializeAddressChunk is a paid mutator transaction binding the contract method 0x3594dea6.
//
// Solidity: function initialize_address_chunk() returns()
func (_Simulator *SimulatorTransactor) InitializeAddressChunk(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Simulator.contract.Transact(opts, "initialize_address_chunk")
}

// InitializeAddressChunk is a paid mutator transaction binding the contract method 0x3594dea6.
//
// Solidity: function initialize_address_chunk() returns()
func (_Simulator *SimulatorSession) InitializeAddressChunk() (*types.Transaction, error) {
	return _Simulator.Contract.InitializeAddressChunk(&_Simulator.TransactOpts)
}

// InitializeAddressChunk is a paid mutator transaction binding the contract method 0x3594dea6.
//
// Solidity: function initialize_address_chunk() returns()
func (_Simulator *SimulatorTransactorSession) InitializeAddressChunk() (*types.Transaction, error) {
	return _Simulator.Contract.InitializeAddressChunk(&_Simulator.TransactOpts)
}

// InitializeStorageChunk is a paid mutator transaction binding the contract method 0x4e83a9d5.
//
// Solidity: function initialize_storage_chunk() returns()
func (_Simulator *SimulatorTransactor) InitializeStorageChunk(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Simulator.contract.Transact(opts, "initialize_storage_chunk")
}

// InitializeStorageChunk is a paid mutator transaction binding the contract method 0x4e83a9d5.
//
// Solidity: function initialize_storage_chunk() returns()
func (_Simulator *SimulatorSession) InitializeStorageChunk() (*types.Transaction, error) {
	return _Simulator.Contract.InitializeStorageChunk(&_Simulator.TransactOpts)
}

// InitializeStorageChunk is a paid mutator transaction binding the contract method 0x4e83a9d5.
//
// Solidity: function initialize_storage_chunk() returns()
func (_Simulator *SimulatorTransactorSession) InitializeStorageChunk() (*types.Transaction, error) {
	return _Simulator.Contract.InitializeStorageChunk(&_Simulator.TransactOpts)
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
