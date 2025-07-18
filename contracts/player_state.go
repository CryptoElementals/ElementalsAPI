// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package contract

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

// PlayerStatePlayerStorage is an auto generated low-level Go binding around an user-defined struct.
type PlayerStatePlayerStorage struct {
	Credits     *big.Int
	TokenAmount *big.Int
}

// PlayerStateContractMetaData contains all meta data concerning the PlayerStateContract contract.
var PlayerStateContractMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"manager\",\"type\":\"address\"}],\"name\":\"AddManager\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"player\",\"type\":\"address\"}],\"name\":\"AddPlayerIfInexist\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"propertyName\",\"type\":\"string\"}],\"name\":\"AddProperty\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"player\",\"type\":\"address\"}],\"name\":\"Get\",\"outputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"Credits\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"TokenAmount\",\"type\":\"uint256\"}],\"internalType\":\"structPlayerState.PlayerStorage\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"player\",\"type\":\"address\"}],\"name\":\"GetCredits\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"player\",\"type\":\"address\"}],\"name\":\"GetTokenAmount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"Manager\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"ManagerIndex\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"PlayerData\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"Credits\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"TokenAmount\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"PlayerExist\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"name\":\"PlayerProperties\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"player\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_credits\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_tokenAmount\",\"type\":\"uint256\"},{\"internalType\":\"string\",\"name\":\"_propertyName\",\"type\":\"string\"},{\"internalType\":\"uint256\",\"name\":\"_propertyAmount\",\"type\":\"uint256\"}],\"name\":\"Set\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"player\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"credits\",\"type\":\"uint256\"}],\"name\":\"SetCredits\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"player\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"propertyName\",\"type\":\"string\"},{\"internalType\":\"uint256\",\"name\":\"propertyAmount\",\"type\":\"uint256\"}],\"name\":\"SetProperty\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"player\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"tokenAmount\",\"type\":\"uint256\"}],\"name\":\"SetTokenAmount\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// PlayerStateContractABI is the input ABI used to generate the binding from.
// Deprecated: Use PlayerStateContractMetaData.ABI instead.
var PlayerStateContractABI = PlayerStateContractMetaData.ABI

// PlayerStateContract is an auto generated Go binding around an Ethereum contract.
type PlayerStateContract struct {
	PlayerStateContractCaller     // Read-only binding to the contract
	PlayerStateContractTransactor // Write-only binding to the contract
	PlayerStateContractFilterer   // Log filterer for contract events
}

// PlayerStateContractCaller is an auto generated read-only Go binding around an Ethereum contract.
type PlayerStateContractCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PlayerStateContractTransactor is an auto generated write-only Go binding around an Ethereum contract.
type PlayerStateContractTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PlayerStateContractFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type PlayerStateContractFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PlayerStateContractSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type PlayerStateContractSession struct {
	Contract     *PlayerStateContract // Generic contract binding to set the session for
	CallOpts     bind.CallOpts        // Call options to use throughout this session
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// PlayerStateContractCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type PlayerStateContractCallerSession struct {
	Contract *PlayerStateContractCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts              // Call options to use throughout this session
}

// PlayerStateContractTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type PlayerStateContractTransactorSession struct {
	Contract     *PlayerStateContractTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts              // Transaction auth options to use throughout this session
}

// PlayerStateContractRaw is an auto generated low-level Go binding around an Ethereum contract.
type PlayerStateContractRaw struct {
	Contract *PlayerStateContract // Generic contract binding to access the raw methods on
}

// PlayerStateContractCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type PlayerStateContractCallerRaw struct {
	Contract *PlayerStateContractCaller // Generic read-only contract binding to access the raw methods on
}

// PlayerStateContractTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type PlayerStateContractTransactorRaw struct {
	Contract *PlayerStateContractTransactor // Generic write-only contract binding to access the raw methods on
}

// NewPlayerStateContract creates a new instance of PlayerStateContract, bound to a specific deployed contract.
func NewPlayerStateContract(address common.Address, backend bind.ContractBackend) (*PlayerStateContract, error) {
	contract, err := bindPlayerStateContract(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &PlayerStateContract{PlayerStateContractCaller: PlayerStateContractCaller{contract: contract}, PlayerStateContractTransactor: PlayerStateContractTransactor{contract: contract}, PlayerStateContractFilterer: PlayerStateContractFilterer{contract: contract}}, nil
}

// NewPlayerStateContractCaller creates a new read-only instance of PlayerStateContract, bound to a specific deployed contract.
func NewPlayerStateContractCaller(address common.Address, caller bind.ContractCaller) (*PlayerStateContractCaller, error) {
	contract, err := bindPlayerStateContract(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &PlayerStateContractCaller{contract: contract}, nil
}

// NewPlayerStateContractTransactor creates a new write-only instance of PlayerStateContract, bound to a specific deployed contract.
func NewPlayerStateContractTransactor(address common.Address, transactor bind.ContractTransactor) (*PlayerStateContractTransactor, error) {
	contract, err := bindPlayerStateContract(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &PlayerStateContractTransactor{contract: contract}, nil
}

// NewPlayerStateContractFilterer creates a new log filterer instance of PlayerStateContract, bound to a specific deployed contract.
func NewPlayerStateContractFilterer(address common.Address, filterer bind.ContractFilterer) (*PlayerStateContractFilterer, error) {
	contract, err := bindPlayerStateContract(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &PlayerStateContractFilterer{contract: contract}, nil
}

// bindPlayerStateContract binds a generic wrapper to an already deployed contract.
func bindPlayerStateContract(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := PlayerStateContractMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_PlayerStateContract *PlayerStateContractRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _PlayerStateContract.Contract.PlayerStateContractCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_PlayerStateContract *PlayerStateContractRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _PlayerStateContract.Contract.PlayerStateContractTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_PlayerStateContract *PlayerStateContractRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _PlayerStateContract.Contract.PlayerStateContractTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_PlayerStateContract *PlayerStateContractCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _PlayerStateContract.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_PlayerStateContract *PlayerStateContractTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _PlayerStateContract.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_PlayerStateContract *PlayerStateContractTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _PlayerStateContract.Contract.contract.Transact(opts, method, params...)
}

// Get is a free data retrieval call binding the contract method 0xb4ab2e71.
//
// Solidity: function Get(address player) view returns((uint256,uint256))
func (_PlayerStateContract *PlayerStateContractCaller) Get(opts *bind.CallOpts, player common.Address) (PlayerStatePlayerStorage, error) {
	var out []interface{}
	err := _PlayerStateContract.contract.Call(opts, &out, "Get", player)

	if err != nil {
		return *new(PlayerStatePlayerStorage), err
	}

	out0 := *abi.ConvertType(out[0], new(PlayerStatePlayerStorage)).(*PlayerStatePlayerStorage)

	return out0, err

}

// Get is a free data retrieval call binding the contract method 0xb4ab2e71.
//
// Solidity: function Get(address player) view returns((uint256,uint256))
func (_PlayerStateContract *PlayerStateContractSession) Get(player common.Address) (PlayerStatePlayerStorage, error) {
	return _PlayerStateContract.Contract.Get(&_PlayerStateContract.CallOpts, player)
}

// Get is a free data retrieval call binding the contract method 0xb4ab2e71.
//
// Solidity: function Get(address player) view returns((uint256,uint256))
func (_PlayerStateContract *PlayerStateContractCallerSession) Get(player common.Address) (PlayerStatePlayerStorage, error) {
	return _PlayerStateContract.Contract.Get(&_PlayerStateContract.CallOpts, player)
}

// GetCredits is a free data retrieval call binding the contract method 0x01f74f79.
//
// Solidity: function GetCredits(address player) view returns(uint256)
func (_PlayerStateContract *PlayerStateContractCaller) GetCredits(opts *bind.CallOpts, player common.Address) (*big.Int, error) {
	var out []interface{}
	err := _PlayerStateContract.contract.Call(opts, &out, "GetCredits", player)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetCredits is a free data retrieval call binding the contract method 0x01f74f79.
//
// Solidity: function GetCredits(address player) view returns(uint256)
func (_PlayerStateContract *PlayerStateContractSession) GetCredits(player common.Address) (*big.Int, error) {
	return _PlayerStateContract.Contract.GetCredits(&_PlayerStateContract.CallOpts, player)
}

// GetCredits is a free data retrieval call binding the contract method 0x01f74f79.
//
// Solidity: function GetCredits(address player) view returns(uint256)
func (_PlayerStateContract *PlayerStateContractCallerSession) GetCredits(player common.Address) (*big.Int, error) {
	return _PlayerStateContract.Contract.GetCredits(&_PlayerStateContract.CallOpts, player)
}

// GetTokenAmount is a free data retrieval call binding the contract method 0xd63e8583.
//
// Solidity: function GetTokenAmount(address player) view returns(uint256)
func (_PlayerStateContract *PlayerStateContractCaller) GetTokenAmount(opts *bind.CallOpts, player common.Address) (*big.Int, error) {
	var out []interface{}
	err := _PlayerStateContract.contract.Call(opts, &out, "GetTokenAmount", player)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetTokenAmount is a free data retrieval call binding the contract method 0xd63e8583.
//
// Solidity: function GetTokenAmount(address player) view returns(uint256)
func (_PlayerStateContract *PlayerStateContractSession) GetTokenAmount(player common.Address) (*big.Int, error) {
	return _PlayerStateContract.Contract.GetTokenAmount(&_PlayerStateContract.CallOpts, player)
}

// GetTokenAmount is a free data retrieval call binding the contract method 0xd63e8583.
//
// Solidity: function GetTokenAmount(address player) view returns(uint256)
func (_PlayerStateContract *PlayerStateContractCallerSession) GetTokenAmount(player common.Address) (*big.Int, error) {
	return _PlayerStateContract.Contract.GetTokenAmount(&_PlayerStateContract.CallOpts, player)
}

// Manager is a free data retrieval call binding the contract method 0x223760e8.
//
// Solidity: function Manager(uint256 ) view returns(address)
func (_PlayerStateContract *PlayerStateContractCaller) Manager(opts *bind.CallOpts, arg0 *big.Int) (common.Address, error) {
	var out []interface{}
	err := _PlayerStateContract.contract.Call(opts, &out, "Manager", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Manager is a free data retrieval call binding the contract method 0x223760e8.
//
// Solidity: function Manager(uint256 ) view returns(address)
func (_PlayerStateContract *PlayerStateContractSession) Manager(arg0 *big.Int) (common.Address, error) {
	return _PlayerStateContract.Contract.Manager(&_PlayerStateContract.CallOpts, arg0)
}

// Manager is a free data retrieval call binding the contract method 0x223760e8.
//
// Solidity: function Manager(uint256 ) view returns(address)
func (_PlayerStateContract *PlayerStateContractCallerSession) Manager(arg0 *big.Int) (common.Address, error) {
	return _PlayerStateContract.Contract.Manager(&_PlayerStateContract.CallOpts, arg0)
}

// ManagerIndex is a free data retrieval call binding the contract method 0x2d8c8926.
//
// Solidity: function ManagerIndex(address ) view returns(uint256)
func (_PlayerStateContract *PlayerStateContractCaller) ManagerIndex(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _PlayerStateContract.contract.Call(opts, &out, "ManagerIndex", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ManagerIndex is a free data retrieval call binding the contract method 0x2d8c8926.
//
// Solidity: function ManagerIndex(address ) view returns(uint256)
func (_PlayerStateContract *PlayerStateContractSession) ManagerIndex(arg0 common.Address) (*big.Int, error) {
	return _PlayerStateContract.Contract.ManagerIndex(&_PlayerStateContract.CallOpts, arg0)
}

// ManagerIndex is a free data retrieval call binding the contract method 0x2d8c8926.
//
// Solidity: function ManagerIndex(address ) view returns(uint256)
func (_PlayerStateContract *PlayerStateContractCallerSession) ManagerIndex(arg0 common.Address) (*big.Int, error) {
	return _PlayerStateContract.Contract.ManagerIndex(&_PlayerStateContract.CallOpts, arg0)
}

// PlayerData is a free data retrieval call binding the contract method 0x5ec3e62e.
//
// Solidity: function PlayerData(address ) view returns(uint256 Credits, uint256 TokenAmount)
func (_PlayerStateContract *PlayerStateContractCaller) PlayerData(opts *bind.CallOpts, arg0 common.Address) (struct {
	Credits     *big.Int
	TokenAmount *big.Int
}, error) {
	var out []interface{}
	err := _PlayerStateContract.contract.Call(opts, &out, "PlayerData", arg0)

	outstruct := new(struct {
		Credits     *big.Int
		TokenAmount *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Credits = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.TokenAmount = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// PlayerData is a free data retrieval call binding the contract method 0x5ec3e62e.
//
// Solidity: function PlayerData(address ) view returns(uint256 Credits, uint256 TokenAmount)
func (_PlayerStateContract *PlayerStateContractSession) PlayerData(arg0 common.Address) (struct {
	Credits     *big.Int
	TokenAmount *big.Int
}, error) {
	return _PlayerStateContract.Contract.PlayerData(&_PlayerStateContract.CallOpts, arg0)
}

// PlayerData is a free data retrieval call binding the contract method 0x5ec3e62e.
//
// Solidity: function PlayerData(address ) view returns(uint256 Credits, uint256 TokenAmount)
func (_PlayerStateContract *PlayerStateContractCallerSession) PlayerData(arg0 common.Address) (struct {
	Credits     *big.Int
	TokenAmount *big.Int
}, error) {
	return _PlayerStateContract.Contract.PlayerData(&_PlayerStateContract.CallOpts, arg0)
}

// PlayerExist is a free data retrieval call binding the contract method 0x62517301.
//
// Solidity: function PlayerExist(address ) view returns(bool)
func (_PlayerStateContract *PlayerStateContractCaller) PlayerExist(opts *bind.CallOpts, arg0 common.Address) (bool, error) {
	var out []interface{}
	err := _PlayerStateContract.contract.Call(opts, &out, "PlayerExist", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// PlayerExist is a free data retrieval call binding the contract method 0x62517301.
//
// Solidity: function PlayerExist(address ) view returns(bool)
func (_PlayerStateContract *PlayerStateContractSession) PlayerExist(arg0 common.Address) (bool, error) {
	return _PlayerStateContract.Contract.PlayerExist(&_PlayerStateContract.CallOpts, arg0)
}

// PlayerExist is a free data retrieval call binding the contract method 0x62517301.
//
// Solidity: function PlayerExist(address ) view returns(bool)
func (_PlayerStateContract *PlayerStateContractCallerSession) PlayerExist(arg0 common.Address) (bool, error) {
	return _PlayerStateContract.Contract.PlayerExist(&_PlayerStateContract.CallOpts, arg0)
}

// PlayerProperties is a free data retrieval call binding the contract method 0x1aa816e6.
//
// Solidity: function PlayerProperties(address , string ) view returns(uint256)
func (_PlayerStateContract *PlayerStateContractCaller) PlayerProperties(opts *bind.CallOpts, arg0 common.Address, arg1 string) (*big.Int, error) {
	var out []interface{}
	err := _PlayerStateContract.contract.Call(opts, &out, "PlayerProperties", arg0, arg1)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// PlayerProperties is a free data retrieval call binding the contract method 0x1aa816e6.
//
// Solidity: function PlayerProperties(address , string ) view returns(uint256)
func (_PlayerStateContract *PlayerStateContractSession) PlayerProperties(arg0 common.Address, arg1 string) (*big.Int, error) {
	return _PlayerStateContract.Contract.PlayerProperties(&_PlayerStateContract.CallOpts, arg0, arg1)
}

// PlayerProperties is a free data retrieval call binding the contract method 0x1aa816e6.
//
// Solidity: function PlayerProperties(address , string ) view returns(uint256)
func (_PlayerStateContract *PlayerStateContractCallerSession) PlayerProperties(arg0 common.Address, arg1 string) (*big.Int, error) {
	return _PlayerStateContract.Contract.PlayerProperties(&_PlayerStateContract.CallOpts, arg0, arg1)
}

// AddManager is a paid mutator transaction binding the contract method 0x3630096a.
//
// Solidity: function AddManager(address manager) returns()
func (_PlayerStateContract *PlayerStateContractTransactor) AddManager(opts *bind.TransactOpts, manager common.Address) (*types.Transaction, error) {
	return _PlayerStateContract.contract.Transact(opts, "AddManager", manager)
}

// AddManager is a paid mutator transaction binding the contract method 0x3630096a.
//
// Solidity: function AddManager(address manager) returns()
func (_PlayerStateContract *PlayerStateContractSession) AddManager(manager common.Address) (*types.Transaction, error) {
	return _PlayerStateContract.Contract.AddManager(&_PlayerStateContract.TransactOpts, manager)
}

// AddManager is a paid mutator transaction binding the contract method 0x3630096a.
//
// Solidity: function AddManager(address manager) returns()
func (_PlayerStateContract *PlayerStateContractTransactorSession) AddManager(manager common.Address) (*types.Transaction, error) {
	return _PlayerStateContract.Contract.AddManager(&_PlayerStateContract.TransactOpts, manager)
}

// AddPlayerIfInexist is a paid mutator transaction binding the contract method 0x606ceece.
//
// Solidity: function AddPlayerIfInexist(address player) returns()
func (_PlayerStateContract *PlayerStateContractTransactor) AddPlayerIfInexist(opts *bind.TransactOpts, player common.Address) (*types.Transaction, error) {
	return _PlayerStateContract.contract.Transact(opts, "AddPlayerIfInexist", player)
}

// AddPlayerIfInexist is a paid mutator transaction binding the contract method 0x606ceece.
//
// Solidity: function AddPlayerIfInexist(address player) returns()
func (_PlayerStateContract *PlayerStateContractSession) AddPlayerIfInexist(player common.Address) (*types.Transaction, error) {
	return _PlayerStateContract.Contract.AddPlayerIfInexist(&_PlayerStateContract.TransactOpts, player)
}

// AddPlayerIfInexist is a paid mutator transaction binding the contract method 0x606ceece.
//
// Solidity: function AddPlayerIfInexist(address player) returns()
func (_PlayerStateContract *PlayerStateContractTransactorSession) AddPlayerIfInexist(player common.Address) (*types.Transaction, error) {
	return _PlayerStateContract.Contract.AddPlayerIfInexist(&_PlayerStateContract.TransactOpts, player)
}

// AddProperty is a paid mutator transaction binding the contract method 0x0abe3e05.
//
// Solidity: function AddProperty(string propertyName) returns()
func (_PlayerStateContract *PlayerStateContractTransactor) AddProperty(opts *bind.TransactOpts, propertyName string) (*types.Transaction, error) {
	return _PlayerStateContract.contract.Transact(opts, "AddProperty", propertyName)
}

// AddProperty is a paid mutator transaction binding the contract method 0x0abe3e05.
//
// Solidity: function AddProperty(string propertyName) returns()
func (_PlayerStateContract *PlayerStateContractSession) AddProperty(propertyName string) (*types.Transaction, error) {
	return _PlayerStateContract.Contract.AddProperty(&_PlayerStateContract.TransactOpts, propertyName)
}

// AddProperty is a paid mutator transaction binding the contract method 0x0abe3e05.
//
// Solidity: function AddProperty(string propertyName) returns()
func (_PlayerStateContract *PlayerStateContractTransactorSession) AddProperty(propertyName string) (*types.Transaction, error) {
	return _PlayerStateContract.Contract.AddProperty(&_PlayerStateContract.TransactOpts, propertyName)
}

// Set is a paid mutator transaction binding the contract method 0x34da2dd5.
//
// Solidity: function Set(address player, uint256 _credits, uint256 _tokenAmount, string _propertyName, uint256 _propertyAmount) returns()
func (_PlayerStateContract *PlayerStateContractTransactor) Set(opts *bind.TransactOpts, player common.Address, _credits *big.Int, _tokenAmount *big.Int, _propertyName string, _propertyAmount *big.Int) (*types.Transaction, error) {
	return _PlayerStateContract.contract.Transact(opts, "Set", player, _credits, _tokenAmount, _propertyName, _propertyAmount)
}

// Set is a paid mutator transaction binding the contract method 0x34da2dd5.
//
// Solidity: function Set(address player, uint256 _credits, uint256 _tokenAmount, string _propertyName, uint256 _propertyAmount) returns()
func (_PlayerStateContract *PlayerStateContractSession) Set(player common.Address, _credits *big.Int, _tokenAmount *big.Int, _propertyName string, _propertyAmount *big.Int) (*types.Transaction, error) {
	return _PlayerStateContract.Contract.Set(&_PlayerStateContract.TransactOpts, player, _credits, _tokenAmount, _propertyName, _propertyAmount)
}

// Set is a paid mutator transaction binding the contract method 0x34da2dd5.
//
// Solidity: function Set(address player, uint256 _credits, uint256 _tokenAmount, string _propertyName, uint256 _propertyAmount) returns()
func (_PlayerStateContract *PlayerStateContractTransactorSession) Set(player common.Address, _credits *big.Int, _tokenAmount *big.Int, _propertyName string, _propertyAmount *big.Int) (*types.Transaction, error) {
	return _PlayerStateContract.Contract.Set(&_PlayerStateContract.TransactOpts, player, _credits, _tokenAmount, _propertyName, _propertyAmount)
}

// SetCredits is a paid mutator transaction binding the contract method 0x502f13ba.
//
// Solidity: function SetCredits(address player, uint256 credits) returns()
func (_PlayerStateContract *PlayerStateContractTransactor) SetCredits(opts *bind.TransactOpts, player common.Address, credits *big.Int) (*types.Transaction, error) {
	return _PlayerStateContract.contract.Transact(opts, "SetCredits", player, credits)
}

// SetCredits is a paid mutator transaction binding the contract method 0x502f13ba.
//
// Solidity: function SetCredits(address player, uint256 credits) returns()
func (_PlayerStateContract *PlayerStateContractSession) SetCredits(player common.Address, credits *big.Int) (*types.Transaction, error) {
	return _PlayerStateContract.Contract.SetCredits(&_PlayerStateContract.TransactOpts, player, credits)
}

// SetCredits is a paid mutator transaction binding the contract method 0x502f13ba.
//
// Solidity: function SetCredits(address player, uint256 credits) returns()
func (_PlayerStateContract *PlayerStateContractTransactorSession) SetCredits(player common.Address, credits *big.Int) (*types.Transaction, error) {
	return _PlayerStateContract.Contract.SetCredits(&_PlayerStateContract.TransactOpts, player, credits)
}

// SetProperty is a paid mutator transaction binding the contract method 0x3dc03dc6.
//
// Solidity: function SetProperty(address player, string propertyName, uint256 propertyAmount) returns()
func (_PlayerStateContract *PlayerStateContractTransactor) SetProperty(opts *bind.TransactOpts, player common.Address, propertyName string, propertyAmount *big.Int) (*types.Transaction, error) {
	return _PlayerStateContract.contract.Transact(opts, "SetProperty", player, propertyName, propertyAmount)
}

// SetProperty is a paid mutator transaction binding the contract method 0x3dc03dc6.
//
// Solidity: function SetProperty(address player, string propertyName, uint256 propertyAmount) returns()
func (_PlayerStateContract *PlayerStateContractSession) SetProperty(player common.Address, propertyName string, propertyAmount *big.Int) (*types.Transaction, error) {
	return _PlayerStateContract.Contract.SetProperty(&_PlayerStateContract.TransactOpts, player, propertyName, propertyAmount)
}

// SetProperty is a paid mutator transaction binding the contract method 0x3dc03dc6.
//
// Solidity: function SetProperty(address player, string propertyName, uint256 propertyAmount) returns()
func (_PlayerStateContract *PlayerStateContractTransactorSession) SetProperty(player common.Address, propertyName string, propertyAmount *big.Int) (*types.Transaction, error) {
	return _PlayerStateContract.Contract.SetProperty(&_PlayerStateContract.TransactOpts, player, propertyName, propertyAmount)
}

// SetTokenAmount is a paid mutator transaction binding the contract method 0x769c9ce2.
//
// Solidity: function SetTokenAmount(address player, uint256 tokenAmount) returns()
func (_PlayerStateContract *PlayerStateContractTransactor) SetTokenAmount(opts *bind.TransactOpts, player common.Address, tokenAmount *big.Int) (*types.Transaction, error) {
	return _PlayerStateContract.contract.Transact(opts, "SetTokenAmount", player, tokenAmount)
}

// SetTokenAmount is a paid mutator transaction binding the contract method 0x769c9ce2.
//
// Solidity: function SetTokenAmount(address player, uint256 tokenAmount) returns()
func (_PlayerStateContract *PlayerStateContractSession) SetTokenAmount(player common.Address, tokenAmount *big.Int) (*types.Transaction, error) {
	return _PlayerStateContract.Contract.SetTokenAmount(&_PlayerStateContract.TransactOpts, player, tokenAmount)
}

// SetTokenAmount is a paid mutator transaction binding the contract method 0x769c9ce2.
//
// Solidity: function SetTokenAmount(address player, uint256 tokenAmount) returns()
func (_PlayerStateContract *PlayerStateContractTransactorSession) SetTokenAmount(player common.Address, tokenAmount *big.Int) (*types.Transaction, error) {
	return _PlayerStateContract.Contract.SetTokenAmount(&_PlayerStateContract.TransactOpts, player, tokenAmount)
}
