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

// WalletManagerContractMetaData contains all meta data concerning the WalletManagerContract contract.
var WalletManagerContractMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"admin\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"enabled\",\"type\":\"bool\"}],\"name\":\"AdminSet\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"oldOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnerTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"walletIndex\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"wallet\",\"type\":\"address\"}],\"name\":\"WalletAdded\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"walletIndex\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"oldAddress\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newAddress\",\"type\":\"address\"}],\"name\":\"WalletAddressUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"walletIndex\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"isActive\",\"type\":\"bool\"}],\"name\":\"WalletRevoked\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"walletIndex\",\"type\":\"uint256\"}],\"name\":\"activateWallet\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"position\",\"type\":\"uint256\"}],\"name\":\"activeWalletAt\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"walletIndex\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"activeWalletCount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"activeWalletIndexList\",\"outputs\":[{\"internalType\":\"uint256[]\",\"name\":\"\",\"type\":\"uint256[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"activeWalletList\",\"outputs\":[{\"internalType\":\"address[]\",\"name\":\"\",\"type\":\"address[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"admin\",\"type\":\"address\"}],\"name\":\"addAdmin\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"wallet\",\"type\":\"address\"}],\"name\":\"addWallet\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"walletIndex\",\"type\":\"uint256\"}],\"name\":\"deactivateWallet\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"admin\",\"type\":\"address\"}],\"name\":\"getAdmin\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getOwner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"playerId\",\"type\":\"uint256\"}],\"name\":\"getWalletAddr\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"wallet\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"walletIndex\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"wallet\",\"type\":\"address\"}],\"name\":\"getWalletByAddress\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"walletIndex\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isActive\",\"type\":\"bool\"},{\"internalType\":\"bool\",\"name\":\"isCurrent\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"walletIndex\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"historyIndex\",\"type\":\"uint256\"}],\"name\":\"getWalletHistory\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"playerId\",\"type\":\"uint256\"}],\"name\":\"getWalletIndexForPlayerId\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"walletIndex\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"walletIndex\",\"type\":\"uint256\"}],\"name\":\"getWalletSlot\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"exists\",\"type\":\"bool\"},{\"internalType\":\"bool\",\"name\":\"isActive\",\"type\":\"bool\"},{\"internalType\":\"address\",\"name\":\"currentAddress\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"historyLength\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"isAdmin\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"wallet\",\"type\":\"address\"}],\"name\":\"isWalletActive\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"wallet\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"playerId\",\"type\":\"uint256\"}],\"name\":\"isWalletAddressForPlayerId\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"walletIndex\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"playerId\",\"type\":\"uint256\"}],\"name\":\"isWalletIndexForPlayerId\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"admin\",\"type\":\"address\"}],\"name\":\"removeAdmin\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"newWallets\",\"type\":\"address[]\"}],\"name\":\"scaleWallets\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"walletIndex\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"newWallet\",\"type\":\"address\"}],\"name\":\"setNewWallet\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"walletIndex\",\"type\":\"uint256\"}],\"name\":\"startWallet\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"walletIndex\",\"type\":\"uint256\"}],\"name\":\"stopWallet\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"totalWallets\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"walletIndex\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isActive\",\"type\":\"bool\"}],\"name\":\"updateWallet\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"wallet\",\"type\":\"address\"}],\"name\":\"walletIndexByAddress\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"found\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"walletIndex\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// WalletManagerContractABI is the input ABI used to generate the binding from.
// Deprecated: Use WalletManagerContractMetaData.ABI instead.
var WalletManagerContractABI = WalletManagerContractMetaData.ABI

// WalletManagerContract is an auto generated Go binding around an Ethereum contract.
type WalletManagerContract struct {
	WalletManagerContractCaller     // Read-only binding to the contract
	WalletManagerContractTransactor // Write-only binding to the contract
	WalletManagerContractFilterer   // Log filterer for contract events
}

// WalletManagerContractCaller is an auto generated read-only Go binding around an Ethereum contract.
type WalletManagerContractCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// WalletManagerContractTransactor is an auto generated write-only Go binding around an Ethereum contract.
type WalletManagerContractTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// WalletManagerContractFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type WalletManagerContractFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// WalletManagerContractSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type WalletManagerContractSession struct {
	Contract     *WalletManagerContract // Generic contract binding to set the session for
	CallOpts     bind.CallOpts          // Call options to use throughout this session
	TransactOpts bind.TransactOpts      // Transaction auth options to use throughout this session
}

// WalletManagerContractCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type WalletManagerContractCallerSession struct {
	Contract *WalletManagerContractCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                // Call options to use throughout this session
}

// WalletManagerContractTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type WalletManagerContractTransactorSession struct {
	Contract     *WalletManagerContractTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                // Transaction auth options to use throughout this session
}

// WalletManagerContractRaw is an auto generated low-level Go binding around an Ethereum contract.
type WalletManagerContractRaw struct {
	Contract *WalletManagerContract // Generic contract binding to access the raw methods on
}

// WalletManagerContractCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type WalletManagerContractCallerRaw struct {
	Contract *WalletManagerContractCaller // Generic read-only contract binding to access the raw methods on
}

// WalletManagerContractTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type WalletManagerContractTransactorRaw struct {
	Contract *WalletManagerContractTransactor // Generic write-only contract binding to access the raw methods on
}

// NewWalletManagerContract creates a new instance of WalletManagerContract, bound to a specific deployed contract.
func NewWalletManagerContract(address common.Address, backend bind.ContractBackend) (*WalletManagerContract, error) {
	contract, err := bindWalletManagerContract(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &WalletManagerContract{WalletManagerContractCaller: WalletManagerContractCaller{contract: contract}, WalletManagerContractTransactor: WalletManagerContractTransactor{contract: contract}, WalletManagerContractFilterer: WalletManagerContractFilterer{contract: contract}}, nil
}

// NewWalletManagerContractCaller creates a new read-only instance of WalletManagerContract, bound to a specific deployed contract.
func NewWalletManagerContractCaller(address common.Address, caller bind.ContractCaller) (*WalletManagerContractCaller, error) {
	contract, err := bindWalletManagerContract(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &WalletManagerContractCaller{contract: contract}, nil
}

// NewWalletManagerContractTransactor creates a new write-only instance of WalletManagerContract, bound to a specific deployed contract.
func NewWalletManagerContractTransactor(address common.Address, transactor bind.ContractTransactor) (*WalletManagerContractTransactor, error) {
	contract, err := bindWalletManagerContract(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &WalletManagerContractTransactor{contract: contract}, nil
}

// NewWalletManagerContractFilterer creates a new log filterer instance of WalletManagerContract, bound to a specific deployed contract.
func NewWalletManagerContractFilterer(address common.Address, filterer bind.ContractFilterer) (*WalletManagerContractFilterer, error) {
	contract, err := bindWalletManagerContract(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &WalletManagerContractFilterer{contract: contract}, nil
}

// bindWalletManagerContract binds a generic wrapper to an already deployed contract.
func bindWalletManagerContract(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := WalletManagerContractMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_WalletManagerContract *WalletManagerContractRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _WalletManagerContract.Contract.WalletManagerContractCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_WalletManagerContract *WalletManagerContractRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _WalletManagerContract.Contract.WalletManagerContractTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_WalletManagerContract *WalletManagerContractRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _WalletManagerContract.Contract.WalletManagerContractTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_WalletManagerContract *WalletManagerContractCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _WalletManagerContract.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_WalletManagerContract *WalletManagerContractTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _WalletManagerContract.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_WalletManagerContract *WalletManagerContractTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _WalletManagerContract.Contract.contract.Transact(opts, method, params...)
}

// ActiveWalletAt is a free data retrieval call binding the contract method 0x29ab0e60.
//
// Solidity: function activeWalletAt(uint256 position) view returns(uint256 walletIndex)
func (_WalletManagerContract *WalletManagerContractCaller) ActiveWalletAt(opts *bind.CallOpts, position *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _WalletManagerContract.contract.Call(opts, &out, "activeWalletAt", position)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ActiveWalletAt is a free data retrieval call binding the contract method 0x29ab0e60.
//
// Solidity: function activeWalletAt(uint256 position) view returns(uint256 walletIndex)
func (_WalletManagerContract *WalletManagerContractSession) ActiveWalletAt(position *big.Int) (*big.Int, error) {
	return _WalletManagerContract.Contract.ActiveWalletAt(&_WalletManagerContract.CallOpts, position)
}

// ActiveWalletAt is a free data retrieval call binding the contract method 0x29ab0e60.
//
// Solidity: function activeWalletAt(uint256 position) view returns(uint256 walletIndex)
func (_WalletManagerContract *WalletManagerContractCallerSession) ActiveWalletAt(position *big.Int) (*big.Int, error) {
	return _WalletManagerContract.Contract.ActiveWalletAt(&_WalletManagerContract.CallOpts, position)
}

// ActiveWalletCount is a free data retrieval call binding the contract method 0xf1b78c4a.
//
// Solidity: function activeWalletCount() view returns(uint256)
func (_WalletManagerContract *WalletManagerContractCaller) ActiveWalletCount(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _WalletManagerContract.contract.Call(opts, &out, "activeWalletCount")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ActiveWalletCount is a free data retrieval call binding the contract method 0xf1b78c4a.
//
// Solidity: function activeWalletCount() view returns(uint256)
func (_WalletManagerContract *WalletManagerContractSession) ActiveWalletCount() (*big.Int, error) {
	return _WalletManagerContract.Contract.ActiveWalletCount(&_WalletManagerContract.CallOpts)
}

// ActiveWalletCount is a free data retrieval call binding the contract method 0xf1b78c4a.
//
// Solidity: function activeWalletCount() view returns(uint256)
func (_WalletManagerContract *WalletManagerContractCallerSession) ActiveWalletCount() (*big.Int, error) {
	return _WalletManagerContract.Contract.ActiveWalletCount(&_WalletManagerContract.CallOpts)
}

// ActiveWalletIndexList is a free data retrieval call binding the contract method 0xfa9cd5da.
//
// Solidity: function activeWalletIndexList() view returns(uint256[])
func (_WalletManagerContract *WalletManagerContractCaller) ActiveWalletIndexList(opts *bind.CallOpts) ([]*big.Int, error) {
	var out []interface{}
	err := _WalletManagerContract.contract.Call(opts, &out, "activeWalletIndexList")

	if err != nil {
		return *new([]*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new([]*big.Int)).(*[]*big.Int)

	return out0, err

}

// ActiveWalletIndexList is a free data retrieval call binding the contract method 0xfa9cd5da.
//
// Solidity: function activeWalletIndexList() view returns(uint256[])
func (_WalletManagerContract *WalletManagerContractSession) ActiveWalletIndexList() ([]*big.Int, error) {
	return _WalletManagerContract.Contract.ActiveWalletIndexList(&_WalletManagerContract.CallOpts)
}

// ActiveWalletIndexList is a free data retrieval call binding the contract method 0xfa9cd5da.
//
// Solidity: function activeWalletIndexList() view returns(uint256[])
func (_WalletManagerContract *WalletManagerContractCallerSession) ActiveWalletIndexList() ([]*big.Int, error) {
	return _WalletManagerContract.Contract.ActiveWalletIndexList(&_WalletManagerContract.CallOpts)
}

// ActiveWalletList is a free data retrieval call binding the contract method 0x8221a99a.
//
// Solidity: function activeWalletList() view returns(address[])
func (_WalletManagerContract *WalletManagerContractCaller) ActiveWalletList(opts *bind.CallOpts) ([]common.Address, error) {
	var out []interface{}
	err := _WalletManagerContract.contract.Call(opts, &out, "activeWalletList")

	if err != nil {
		return *new([]common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new([]common.Address)).(*[]common.Address)

	return out0, err

}

// ActiveWalletList is a free data retrieval call binding the contract method 0x8221a99a.
//
// Solidity: function activeWalletList() view returns(address[])
func (_WalletManagerContract *WalletManagerContractSession) ActiveWalletList() ([]common.Address, error) {
	return _WalletManagerContract.Contract.ActiveWalletList(&_WalletManagerContract.CallOpts)
}

// ActiveWalletList is a free data retrieval call binding the contract method 0x8221a99a.
//
// Solidity: function activeWalletList() view returns(address[])
func (_WalletManagerContract *WalletManagerContractCallerSession) ActiveWalletList() ([]common.Address, error) {
	return _WalletManagerContract.Contract.ActiveWalletList(&_WalletManagerContract.CallOpts)
}

// GetAdmin is a free data retrieval call binding the contract method 0x64efb22b.
//
// Solidity: function getAdmin(address admin) view returns(bool)
func (_WalletManagerContract *WalletManagerContractCaller) GetAdmin(opts *bind.CallOpts, admin common.Address) (bool, error) {
	var out []interface{}
	err := _WalletManagerContract.contract.Call(opts, &out, "getAdmin", admin)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// GetAdmin is a free data retrieval call binding the contract method 0x64efb22b.
//
// Solidity: function getAdmin(address admin) view returns(bool)
func (_WalletManagerContract *WalletManagerContractSession) GetAdmin(admin common.Address) (bool, error) {
	return _WalletManagerContract.Contract.GetAdmin(&_WalletManagerContract.CallOpts, admin)
}

// GetAdmin is a free data retrieval call binding the contract method 0x64efb22b.
//
// Solidity: function getAdmin(address admin) view returns(bool)
func (_WalletManagerContract *WalletManagerContractCallerSession) GetAdmin(admin common.Address) (bool, error) {
	return _WalletManagerContract.Contract.GetAdmin(&_WalletManagerContract.CallOpts, admin)
}

// GetOwner is a free data retrieval call binding the contract method 0x893d20e8.
//
// Solidity: function getOwner() view returns(address)
func (_WalletManagerContract *WalletManagerContractCaller) GetOwner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _WalletManagerContract.contract.Call(opts, &out, "getOwner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetOwner is a free data retrieval call binding the contract method 0x893d20e8.
//
// Solidity: function getOwner() view returns(address)
func (_WalletManagerContract *WalletManagerContractSession) GetOwner() (common.Address, error) {
	return _WalletManagerContract.Contract.GetOwner(&_WalletManagerContract.CallOpts)
}

// GetOwner is a free data retrieval call binding the contract method 0x893d20e8.
//
// Solidity: function getOwner() view returns(address)
func (_WalletManagerContract *WalletManagerContractCallerSession) GetOwner() (common.Address, error) {
	return _WalletManagerContract.Contract.GetOwner(&_WalletManagerContract.CallOpts)
}

// GetWalletAddr is a free data retrieval call binding the contract method 0x677e4e8f.
//
// Solidity: function getWalletAddr(uint256 playerId) view returns(address wallet, uint256 walletIndex)
func (_WalletManagerContract *WalletManagerContractCaller) GetWalletAddr(opts *bind.CallOpts, playerId *big.Int) (struct {
	Wallet      common.Address
	WalletIndex *big.Int
}, error) {
	var out []interface{}
	err := _WalletManagerContract.contract.Call(opts, &out, "getWalletAddr", playerId)

	outstruct := new(struct {
		Wallet      common.Address
		WalletIndex *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Wallet = *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	outstruct.WalletIndex = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// GetWalletAddr is a free data retrieval call binding the contract method 0x677e4e8f.
//
// Solidity: function getWalletAddr(uint256 playerId) view returns(address wallet, uint256 walletIndex)
func (_WalletManagerContract *WalletManagerContractSession) GetWalletAddr(playerId *big.Int) (struct {
	Wallet      common.Address
	WalletIndex *big.Int
}, error) {
	return _WalletManagerContract.Contract.GetWalletAddr(&_WalletManagerContract.CallOpts, playerId)
}

// GetWalletAddr is a free data retrieval call binding the contract method 0x677e4e8f.
//
// Solidity: function getWalletAddr(uint256 playerId) view returns(address wallet, uint256 walletIndex)
func (_WalletManagerContract *WalletManagerContractCallerSession) GetWalletAddr(playerId *big.Int) (struct {
	Wallet      common.Address
	WalletIndex *big.Int
}, error) {
	return _WalletManagerContract.Contract.GetWalletAddr(&_WalletManagerContract.CallOpts, playerId)
}

// GetWalletByAddress is a free data retrieval call binding the contract method 0x66179d7f.
//
// Solidity: function getWalletByAddress(address wallet) view returns(uint256 walletIndex, bool isActive, bool isCurrent)
func (_WalletManagerContract *WalletManagerContractCaller) GetWalletByAddress(opts *bind.CallOpts, wallet common.Address) (struct {
	WalletIndex *big.Int
	IsActive    bool
	IsCurrent   bool
}, error) {
	var out []interface{}
	err := _WalletManagerContract.contract.Call(opts, &out, "getWalletByAddress", wallet)

	outstruct := new(struct {
		WalletIndex *big.Int
		IsActive    bool
		IsCurrent   bool
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.WalletIndex = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.IsActive = *abi.ConvertType(out[1], new(bool)).(*bool)
	outstruct.IsCurrent = *abi.ConvertType(out[2], new(bool)).(*bool)

	return *outstruct, err

}

// GetWalletByAddress is a free data retrieval call binding the contract method 0x66179d7f.
//
// Solidity: function getWalletByAddress(address wallet) view returns(uint256 walletIndex, bool isActive, bool isCurrent)
func (_WalletManagerContract *WalletManagerContractSession) GetWalletByAddress(wallet common.Address) (struct {
	WalletIndex *big.Int
	IsActive    bool
	IsCurrent   bool
}, error) {
	return _WalletManagerContract.Contract.GetWalletByAddress(&_WalletManagerContract.CallOpts, wallet)
}

// GetWalletByAddress is a free data retrieval call binding the contract method 0x66179d7f.
//
// Solidity: function getWalletByAddress(address wallet) view returns(uint256 walletIndex, bool isActive, bool isCurrent)
func (_WalletManagerContract *WalletManagerContractCallerSession) GetWalletByAddress(wallet common.Address) (struct {
	WalletIndex *big.Int
	IsActive    bool
	IsCurrent   bool
}, error) {
	return _WalletManagerContract.Contract.GetWalletByAddress(&_WalletManagerContract.CallOpts, wallet)
}

// GetWalletHistory is a free data retrieval call binding the contract method 0x75725972.
//
// Solidity: function getWalletHistory(uint256 walletIndex, uint256 historyIndex) view returns(address)
func (_WalletManagerContract *WalletManagerContractCaller) GetWalletHistory(opts *bind.CallOpts, walletIndex *big.Int, historyIndex *big.Int) (common.Address, error) {
	var out []interface{}
	err := _WalletManagerContract.contract.Call(opts, &out, "getWalletHistory", walletIndex, historyIndex)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetWalletHistory is a free data retrieval call binding the contract method 0x75725972.
//
// Solidity: function getWalletHistory(uint256 walletIndex, uint256 historyIndex) view returns(address)
func (_WalletManagerContract *WalletManagerContractSession) GetWalletHistory(walletIndex *big.Int, historyIndex *big.Int) (common.Address, error) {
	return _WalletManagerContract.Contract.GetWalletHistory(&_WalletManagerContract.CallOpts, walletIndex, historyIndex)
}

// GetWalletHistory is a free data retrieval call binding the contract method 0x75725972.
//
// Solidity: function getWalletHistory(uint256 walletIndex, uint256 historyIndex) view returns(address)
func (_WalletManagerContract *WalletManagerContractCallerSession) GetWalletHistory(walletIndex *big.Int, historyIndex *big.Int) (common.Address, error) {
	return _WalletManagerContract.Contract.GetWalletHistory(&_WalletManagerContract.CallOpts, walletIndex, historyIndex)
}

// GetWalletIndexForPlayerId is a free data retrieval call binding the contract method 0x74258796.
//
// Solidity: function getWalletIndexForPlayerId(uint256 playerId) view returns(uint256 walletIndex)
func (_WalletManagerContract *WalletManagerContractCaller) GetWalletIndexForPlayerId(opts *bind.CallOpts, playerId *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _WalletManagerContract.contract.Call(opts, &out, "getWalletIndexForPlayerId", playerId)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetWalletIndexForPlayerId is a free data retrieval call binding the contract method 0x74258796.
//
// Solidity: function getWalletIndexForPlayerId(uint256 playerId) view returns(uint256 walletIndex)
func (_WalletManagerContract *WalletManagerContractSession) GetWalletIndexForPlayerId(playerId *big.Int) (*big.Int, error) {
	return _WalletManagerContract.Contract.GetWalletIndexForPlayerId(&_WalletManagerContract.CallOpts, playerId)
}

// GetWalletIndexForPlayerId is a free data retrieval call binding the contract method 0x74258796.
//
// Solidity: function getWalletIndexForPlayerId(uint256 playerId) view returns(uint256 walletIndex)
func (_WalletManagerContract *WalletManagerContractCallerSession) GetWalletIndexForPlayerId(playerId *big.Int) (*big.Int, error) {
	return _WalletManagerContract.Contract.GetWalletIndexForPlayerId(&_WalletManagerContract.CallOpts, playerId)
}

// GetWalletSlot is a free data retrieval call binding the contract method 0x7fc85899.
//
// Solidity: function getWalletSlot(uint256 walletIndex) view returns(bool exists, bool isActive, address currentAddress, uint256 historyLength)
func (_WalletManagerContract *WalletManagerContractCaller) GetWalletSlot(opts *bind.CallOpts, walletIndex *big.Int) (struct {
	Exists         bool
	IsActive       bool
	CurrentAddress common.Address
	HistoryLength  *big.Int
}, error) {
	var out []interface{}
	err := _WalletManagerContract.contract.Call(opts, &out, "getWalletSlot", walletIndex)

	outstruct := new(struct {
		Exists         bool
		IsActive       bool
		CurrentAddress common.Address
		HistoryLength  *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Exists = *abi.ConvertType(out[0], new(bool)).(*bool)
	outstruct.IsActive = *abi.ConvertType(out[1], new(bool)).(*bool)
	outstruct.CurrentAddress = *abi.ConvertType(out[2], new(common.Address)).(*common.Address)
	outstruct.HistoryLength = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// GetWalletSlot is a free data retrieval call binding the contract method 0x7fc85899.
//
// Solidity: function getWalletSlot(uint256 walletIndex) view returns(bool exists, bool isActive, address currentAddress, uint256 historyLength)
func (_WalletManagerContract *WalletManagerContractSession) GetWalletSlot(walletIndex *big.Int) (struct {
	Exists         bool
	IsActive       bool
	CurrentAddress common.Address
	HistoryLength  *big.Int
}, error) {
	return _WalletManagerContract.Contract.GetWalletSlot(&_WalletManagerContract.CallOpts, walletIndex)
}

// GetWalletSlot is a free data retrieval call binding the contract method 0x7fc85899.
//
// Solidity: function getWalletSlot(uint256 walletIndex) view returns(bool exists, bool isActive, address currentAddress, uint256 historyLength)
func (_WalletManagerContract *WalletManagerContractCallerSession) GetWalletSlot(walletIndex *big.Int) (struct {
	Exists         bool
	IsActive       bool
	CurrentAddress common.Address
	HistoryLength  *big.Int
}, error) {
	return _WalletManagerContract.Contract.GetWalletSlot(&_WalletManagerContract.CallOpts, walletIndex)
}

// IsAdmin is a free data retrieval call binding the contract method 0x24d7806c.
//
// Solidity: function isAdmin(address ) view returns(bool)
func (_WalletManagerContract *WalletManagerContractCaller) IsAdmin(opts *bind.CallOpts, arg0 common.Address) (bool, error) {
	var out []interface{}
	err := _WalletManagerContract.contract.Call(opts, &out, "isAdmin", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsAdmin is a free data retrieval call binding the contract method 0x24d7806c.
//
// Solidity: function isAdmin(address ) view returns(bool)
func (_WalletManagerContract *WalletManagerContractSession) IsAdmin(arg0 common.Address) (bool, error) {
	return _WalletManagerContract.Contract.IsAdmin(&_WalletManagerContract.CallOpts, arg0)
}

// IsAdmin is a free data retrieval call binding the contract method 0x24d7806c.
//
// Solidity: function isAdmin(address ) view returns(bool)
func (_WalletManagerContract *WalletManagerContractCallerSession) IsAdmin(arg0 common.Address) (bool, error) {
	return _WalletManagerContract.Contract.IsAdmin(&_WalletManagerContract.CallOpts, arg0)
}

// IsWalletActive is a free data retrieval call binding the contract method 0xd0ed7b91.
//
// Solidity: function isWalletActive(address wallet) view returns(bool)
func (_WalletManagerContract *WalletManagerContractCaller) IsWalletActive(opts *bind.CallOpts, wallet common.Address) (bool, error) {
	var out []interface{}
	err := _WalletManagerContract.contract.Call(opts, &out, "isWalletActive", wallet)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsWalletActive is a free data retrieval call binding the contract method 0xd0ed7b91.
//
// Solidity: function isWalletActive(address wallet) view returns(bool)
func (_WalletManagerContract *WalletManagerContractSession) IsWalletActive(wallet common.Address) (bool, error) {
	return _WalletManagerContract.Contract.IsWalletActive(&_WalletManagerContract.CallOpts, wallet)
}

// IsWalletActive is a free data retrieval call binding the contract method 0xd0ed7b91.
//
// Solidity: function isWalletActive(address wallet) view returns(bool)
func (_WalletManagerContract *WalletManagerContractCallerSession) IsWalletActive(wallet common.Address) (bool, error) {
	return _WalletManagerContract.Contract.IsWalletActive(&_WalletManagerContract.CallOpts, wallet)
}

// IsWalletAddressForPlayerId is a free data retrieval call binding the contract method 0x11d3b4a7.
//
// Solidity: function isWalletAddressForPlayerId(address wallet, uint256 playerId) view returns(bool)
func (_WalletManagerContract *WalletManagerContractCaller) IsWalletAddressForPlayerId(opts *bind.CallOpts, wallet common.Address, playerId *big.Int) (bool, error) {
	var out []interface{}
	err := _WalletManagerContract.contract.Call(opts, &out, "isWalletAddressForPlayerId", wallet, playerId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsWalletAddressForPlayerId is a free data retrieval call binding the contract method 0x11d3b4a7.
//
// Solidity: function isWalletAddressForPlayerId(address wallet, uint256 playerId) view returns(bool)
func (_WalletManagerContract *WalletManagerContractSession) IsWalletAddressForPlayerId(wallet common.Address, playerId *big.Int) (bool, error) {
	return _WalletManagerContract.Contract.IsWalletAddressForPlayerId(&_WalletManagerContract.CallOpts, wallet, playerId)
}

// IsWalletAddressForPlayerId is a free data retrieval call binding the contract method 0x11d3b4a7.
//
// Solidity: function isWalletAddressForPlayerId(address wallet, uint256 playerId) view returns(bool)
func (_WalletManagerContract *WalletManagerContractCallerSession) IsWalletAddressForPlayerId(wallet common.Address, playerId *big.Int) (bool, error) {
	return _WalletManagerContract.Contract.IsWalletAddressForPlayerId(&_WalletManagerContract.CallOpts, wallet, playerId)
}

// IsWalletIndexForPlayerId is a free data retrieval call binding the contract method 0xfa190583.
//
// Solidity: function isWalletIndexForPlayerId(uint256 walletIndex, uint256 playerId) view returns(bool)
func (_WalletManagerContract *WalletManagerContractCaller) IsWalletIndexForPlayerId(opts *bind.CallOpts, walletIndex *big.Int, playerId *big.Int) (bool, error) {
	var out []interface{}
	err := _WalletManagerContract.contract.Call(opts, &out, "isWalletIndexForPlayerId", walletIndex, playerId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsWalletIndexForPlayerId is a free data retrieval call binding the contract method 0xfa190583.
//
// Solidity: function isWalletIndexForPlayerId(uint256 walletIndex, uint256 playerId) view returns(bool)
func (_WalletManagerContract *WalletManagerContractSession) IsWalletIndexForPlayerId(walletIndex *big.Int, playerId *big.Int) (bool, error) {
	return _WalletManagerContract.Contract.IsWalletIndexForPlayerId(&_WalletManagerContract.CallOpts, walletIndex, playerId)
}

// IsWalletIndexForPlayerId is a free data retrieval call binding the contract method 0xfa190583.
//
// Solidity: function isWalletIndexForPlayerId(uint256 walletIndex, uint256 playerId) view returns(bool)
func (_WalletManagerContract *WalletManagerContractCallerSession) IsWalletIndexForPlayerId(walletIndex *big.Int, playerId *big.Int) (bool, error) {
	return _WalletManagerContract.Contract.IsWalletIndexForPlayerId(&_WalletManagerContract.CallOpts, walletIndex, playerId)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_WalletManagerContract *WalletManagerContractCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _WalletManagerContract.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_WalletManagerContract *WalletManagerContractSession) Owner() (common.Address, error) {
	return _WalletManagerContract.Contract.Owner(&_WalletManagerContract.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_WalletManagerContract *WalletManagerContractCallerSession) Owner() (common.Address, error) {
	return _WalletManagerContract.Contract.Owner(&_WalletManagerContract.CallOpts)
}

// TotalWallets is a free data retrieval call binding the contract method 0x977d2c45.
//
// Solidity: function totalWallets() view returns(uint256)
func (_WalletManagerContract *WalletManagerContractCaller) TotalWallets(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _WalletManagerContract.contract.Call(opts, &out, "totalWallets")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalWallets is a free data retrieval call binding the contract method 0x977d2c45.
//
// Solidity: function totalWallets() view returns(uint256)
func (_WalletManagerContract *WalletManagerContractSession) TotalWallets() (*big.Int, error) {
	return _WalletManagerContract.Contract.TotalWallets(&_WalletManagerContract.CallOpts)
}

// TotalWallets is a free data retrieval call binding the contract method 0x977d2c45.
//
// Solidity: function totalWallets() view returns(uint256)
func (_WalletManagerContract *WalletManagerContractCallerSession) TotalWallets() (*big.Int, error) {
	return _WalletManagerContract.Contract.TotalWallets(&_WalletManagerContract.CallOpts)
}

// WalletIndexByAddress is a free data retrieval call binding the contract method 0x94a34f3c.
//
// Solidity: function walletIndexByAddress(address wallet) view returns(bool found, uint256 walletIndex)
func (_WalletManagerContract *WalletManagerContractCaller) WalletIndexByAddress(opts *bind.CallOpts, wallet common.Address) (struct {
	Found       bool
	WalletIndex *big.Int
}, error) {
	var out []interface{}
	err := _WalletManagerContract.contract.Call(opts, &out, "walletIndexByAddress", wallet)

	outstruct := new(struct {
		Found       bool
		WalletIndex *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Found = *abi.ConvertType(out[0], new(bool)).(*bool)
	outstruct.WalletIndex = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// WalletIndexByAddress is a free data retrieval call binding the contract method 0x94a34f3c.
//
// Solidity: function walletIndexByAddress(address wallet) view returns(bool found, uint256 walletIndex)
func (_WalletManagerContract *WalletManagerContractSession) WalletIndexByAddress(wallet common.Address) (struct {
	Found       bool
	WalletIndex *big.Int
}, error) {
	return _WalletManagerContract.Contract.WalletIndexByAddress(&_WalletManagerContract.CallOpts, wallet)
}

// WalletIndexByAddress is a free data retrieval call binding the contract method 0x94a34f3c.
//
// Solidity: function walletIndexByAddress(address wallet) view returns(bool found, uint256 walletIndex)
func (_WalletManagerContract *WalletManagerContractCallerSession) WalletIndexByAddress(wallet common.Address) (struct {
	Found       bool
	WalletIndex *big.Int
}, error) {
	return _WalletManagerContract.Contract.WalletIndexByAddress(&_WalletManagerContract.CallOpts, wallet)
}

// ActivateWallet is a paid mutator transaction binding the contract method 0x773eaf64.
//
// Solidity: function activateWallet(uint256 walletIndex) returns()
func (_WalletManagerContract *WalletManagerContractTransactor) ActivateWallet(opts *bind.TransactOpts, walletIndex *big.Int) (*types.Transaction, error) {
	return _WalletManagerContract.contract.Transact(opts, "activateWallet", walletIndex)
}

// ActivateWallet is a paid mutator transaction binding the contract method 0x773eaf64.
//
// Solidity: function activateWallet(uint256 walletIndex) returns()
func (_WalletManagerContract *WalletManagerContractSession) ActivateWallet(walletIndex *big.Int) (*types.Transaction, error) {
	return _WalletManagerContract.Contract.ActivateWallet(&_WalletManagerContract.TransactOpts, walletIndex)
}

// ActivateWallet is a paid mutator transaction binding the contract method 0x773eaf64.
//
// Solidity: function activateWallet(uint256 walletIndex) returns()
func (_WalletManagerContract *WalletManagerContractTransactorSession) ActivateWallet(walletIndex *big.Int) (*types.Transaction, error) {
	return _WalletManagerContract.Contract.ActivateWallet(&_WalletManagerContract.TransactOpts, walletIndex)
}

// AddAdmin is a paid mutator transaction binding the contract method 0x70480275.
//
// Solidity: function addAdmin(address admin) returns()
func (_WalletManagerContract *WalletManagerContractTransactor) AddAdmin(opts *bind.TransactOpts, admin common.Address) (*types.Transaction, error) {
	return _WalletManagerContract.contract.Transact(opts, "addAdmin", admin)
}

// AddAdmin is a paid mutator transaction binding the contract method 0x70480275.
//
// Solidity: function addAdmin(address admin) returns()
func (_WalletManagerContract *WalletManagerContractSession) AddAdmin(admin common.Address) (*types.Transaction, error) {
	return _WalletManagerContract.Contract.AddAdmin(&_WalletManagerContract.TransactOpts, admin)
}

// AddAdmin is a paid mutator transaction binding the contract method 0x70480275.
//
// Solidity: function addAdmin(address admin) returns()
func (_WalletManagerContract *WalletManagerContractTransactorSession) AddAdmin(admin common.Address) (*types.Transaction, error) {
	return _WalletManagerContract.Contract.AddAdmin(&_WalletManagerContract.TransactOpts, admin)
}

// AddWallet is a paid mutator transaction binding the contract method 0xefeb5f1f.
//
// Solidity: function addWallet(address wallet) returns()
func (_WalletManagerContract *WalletManagerContractTransactor) AddWallet(opts *bind.TransactOpts, wallet common.Address) (*types.Transaction, error) {
	return _WalletManagerContract.contract.Transact(opts, "addWallet", wallet)
}

// AddWallet is a paid mutator transaction binding the contract method 0xefeb5f1f.
//
// Solidity: function addWallet(address wallet) returns()
func (_WalletManagerContract *WalletManagerContractSession) AddWallet(wallet common.Address) (*types.Transaction, error) {
	return _WalletManagerContract.Contract.AddWallet(&_WalletManagerContract.TransactOpts, wallet)
}

// AddWallet is a paid mutator transaction binding the contract method 0xefeb5f1f.
//
// Solidity: function addWallet(address wallet) returns()
func (_WalletManagerContract *WalletManagerContractTransactorSession) AddWallet(wallet common.Address) (*types.Transaction, error) {
	return _WalletManagerContract.Contract.AddWallet(&_WalletManagerContract.TransactOpts, wallet)
}

// DeactivateWallet is a paid mutator transaction binding the contract method 0x4a0c2173.
//
// Solidity: function deactivateWallet(uint256 walletIndex) returns()
func (_WalletManagerContract *WalletManagerContractTransactor) DeactivateWallet(opts *bind.TransactOpts, walletIndex *big.Int) (*types.Transaction, error) {
	return _WalletManagerContract.contract.Transact(opts, "deactivateWallet", walletIndex)
}

// DeactivateWallet is a paid mutator transaction binding the contract method 0x4a0c2173.
//
// Solidity: function deactivateWallet(uint256 walletIndex) returns()
func (_WalletManagerContract *WalletManagerContractSession) DeactivateWallet(walletIndex *big.Int) (*types.Transaction, error) {
	return _WalletManagerContract.Contract.DeactivateWallet(&_WalletManagerContract.TransactOpts, walletIndex)
}

// DeactivateWallet is a paid mutator transaction binding the contract method 0x4a0c2173.
//
// Solidity: function deactivateWallet(uint256 walletIndex) returns()
func (_WalletManagerContract *WalletManagerContractTransactorSession) DeactivateWallet(walletIndex *big.Int) (*types.Transaction, error) {
	return _WalletManagerContract.Contract.DeactivateWallet(&_WalletManagerContract.TransactOpts, walletIndex)
}

// RemoveAdmin is a paid mutator transaction binding the contract method 0x1785f53c.
//
// Solidity: function removeAdmin(address admin) returns()
func (_WalletManagerContract *WalletManagerContractTransactor) RemoveAdmin(opts *bind.TransactOpts, admin common.Address) (*types.Transaction, error) {
	return _WalletManagerContract.contract.Transact(opts, "removeAdmin", admin)
}

// RemoveAdmin is a paid mutator transaction binding the contract method 0x1785f53c.
//
// Solidity: function removeAdmin(address admin) returns()
func (_WalletManagerContract *WalletManagerContractSession) RemoveAdmin(admin common.Address) (*types.Transaction, error) {
	return _WalletManagerContract.Contract.RemoveAdmin(&_WalletManagerContract.TransactOpts, admin)
}

// RemoveAdmin is a paid mutator transaction binding the contract method 0x1785f53c.
//
// Solidity: function removeAdmin(address admin) returns()
func (_WalletManagerContract *WalletManagerContractTransactorSession) RemoveAdmin(admin common.Address) (*types.Transaction, error) {
	return _WalletManagerContract.Contract.RemoveAdmin(&_WalletManagerContract.TransactOpts, admin)
}

// ScaleWallets is a paid mutator transaction binding the contract method 0xbef901c8.
//
// Solidity: function scaleWallets(address[] newWallets) returns()
func (_WalletManagerContract *WalletManagerContractTransactor) ScaleWallets(opts *bind.TransactOpts, newWallets []common.Address) (*types.Transaction, error) {
	return _WalletManagerContract.contract.Transact(opts, "scaleWallets", newWallets)
}

// ScaleWallets is a paid mutator transaction binding the contract method 0xbef901c8.
//
// Solidity: function scaleWallets(address[] newWallets) returns()
func (_WalletManagerContract *WalletManagerContractSession) ScaleWallets(newWallets []common.Address) (*types.Transaction, error) {
	return _WalletManagerContract.Contract.ScaleWallets(&_WalletManagerContract.TransactOpts, newWallets)
}

// ScaleWallets is a paid mutator transaction binding the contract method 0xbef901c8.
//
// Solidity: function scaleWallets(address[] newWallets) returns()
func (_WalletManagerContract *WalletManagerContractTransactorSession) ScaleWallets(newWallets []common.Address) (*types.Transaction, error) {
	return _WalletManagerContract.Contract.ScaleWallets(&_WalletManagerContract.TransactOpts, newWallets)
}

// SetNewWallet is a paid mutator transaction binding the contract method 0x10b8c8a5.
//
// Solidity: function setNewWallet(uint256 walletIndex, address newWallet) returns()
func (_WalletManagerContract *WalletManagerContractTransactor) SetNewWallet(opts *bind.TransactOpts, walletIndex *big.Int, newWallet common.Address) (*types.Transaction, error) {
	return _WalletManagerContract.contract.Transact(opts, "setNewWallet", walletIndex, newWallet)
}

// SetNewWallet is a paid mutator transaction binding the contract method 0x10b8c8a5.
//
// Solidity: function setNewWallet(uint256 walletIndex, address newWallet) returns()
func (_WalletManagerContract *WalletManagerContractSession) SetNewWallet(walletIndex *big.Int, newWallet common.Address) (*types.Transaction, error) {
	return _WalletManagerContract.Contract.SetNewWallet(&_WalletManagerContract.TransactOpts, walletIndex, newWallet)
}

// SetNewWallet is a paid mutator transaction binding the contract method 0x10b8c8a5.
//
// Solidity: function setNewWallet(uint256 walletIndex, address newWallet) returns()
func (_WalletManagerContract *WalletManagerContractTransactorSession) SetNewWallet(walletIndex *big.Int, newWallet common.Address) (*types.Transaction, error) {
	return _WalletManagerContract.Contract.SetNewWallet(&_WalletManagerContract.TransactOpts, walletIndex, newWallet)
}

// StartWallet is a paid mutator transaction binding the contract method 0x7bb8998f.
//
// Solidity: function startWallet(uint256 walletIndex) returns()
func (_WalletManagerContract *WalletManagerContractTransactor) StartWallet(opts *bind.TransactOpts, walletIndex *big.Int) (*types.Transaction, error) {
	return _WalletManagerContract.contract.Transact(opts, "startWallet", walletIndex)
}

// StartWallet is a paid mutator transaction binding the contract method 0x7bb8998f.
//
// Solidity: function startWallet(uint256 walletIndex) returns()
func (_WalletManagerContract *WalletManagerContractSession) StartWallet(walletIndex *big.Int) (*types.Transaction, error) {
	return _WalletManagerContract.Contract.StartWallet(&_WalletManagerContract.TransactOpts, walletIndex)
}

// StartWallet is a paid mutator transaction binding the contract method 0x7bb8998f.
//
// Solidity: function startWallet(uint256 walletIndex) returns()
func (_WalletManagerContract *WalletManagerContractTransactorSession) StartWallet(walletIndex *big.Int) (*types.Transaction, error) {
	return _WalletManagerContract.Contract.StartWallet(&_WalletManagerContract.TransactOpts, walletIndex)
}

// StopWallet is a paid mutator transaction binding the contract method 0x02740f1b.
//
// Solidity: function stopWallet(uint256 walletIndex) returns()
func (_WalletManagerContract *WalletManagerContractTransactor) StopWallet(opts *bind.TransactOpts, walletIndex *big.Int) (*types.Transaction, error) {
	return _WalletManagerContract.contract.Transact(opts, "stopWallet", walletIndex)
}

// StopWallet is a paid mutator transaction binding the contract method 0x02740f1b.
//
// Solidity: function stopWallet(uint256 walletIndex) returns()
func (_WalletManagerContract *WalletManagerContractSession) StopWallet(walletIndex *big.Int) (*types.Transaction, error) {
	return _WalletManagerContract.Contract.StopWallet(&_WalletManagerContract.TransactOpts, walletIndex)
}

// StopWallet is a paid mutator transaction binding the contract method 0x02740f1b.
//
// Solidity: function stopWallet(uint256 walletIndex) returns()
func (_WalletManagerContract *WalletManagerContractTransactorSession) StopWallet(walletIndex *big.Int) (*types.Transaction, error) {
	return _WalletManagerContract.Contract.StopWallet(&_WalletManagerContract.TransactOpts, walletIndex)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_WalletManagerContract *WalletManagerContractTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _WalletManagerContract.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_WalletManagerContract *WalletManagerContractSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _WalletManagerContract.Contract.TransferOwnership(&_WalletManagerContract.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_WalletManagerContract *WalletManagerContractTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _WalletManagerContract.Contract.TransferOwnership(&_WalletManagerContract.TransactOpts, newOwner)
}

// UpdateWallet is a paid mutator transaction binding the contract method 0x628d46da.
//
// Solidity: function updateWallet(uint256 walletIndex, bool isActive) returns()
func (_WalletManagerContract *WalletManagerContractTransactor) UpdateWallet(opts *bind.TransactOpts, walletIndex *big.Int, isActive bool) (*types.Transaction, error) {
	return _WalletManagerContract.contract.Transact(opts, "updateWallet", walletIndex, isActive)
}

// UpdateWallet is a paid mutator transaction binding the contract method 0x628d46da.
//
// Solidity: function updateWallet(uint256 walletIndex, bool isActive) returns()
func (_WalletManagerContract *WalletManagerContractSession) UpdateWallet(walletIndex *big.Int, isActive bool) (*types.Transaction, error) {
	return _WalletManagerContract.Contract.UpdateWallet(&_WalletManagerContract.TransactOpts, walletIndex, isActive)
}

// UpdateWallet is a paid mutator transaction binding the contract method 0x628d46da.
//
// Solidity: function updateWallet(uint256 walletIndex, bool isActive) returns()
func (_WalletManagerContract *WalletManagerContractTransactorSession) UpdateWallet(walletIndex *big.Int, isActive bool) (*types.Transaction, error) {
	return _WalletManagerContract.Contract.UpdateWallet(&_WalletManagerContract.TransactOpts, walletIndex, isActive)
}

// WalletManagerContractAdminSetIterator is returned from FilterAdminSet and is used to iterate over the raw logs and unpacked data for AdminSet events raised by the WalletManagerContract contract.
type WalletManagerContractAdminSetIterator struct {
	Event *WalletManagerContractAdminSet // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *WalletManagerContractAdminSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(WalletManagerContractAdminSet)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(WalletManagerContractAdminSet)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *WalletManagerContractAdminSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *WalletManagerContractAdminSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// WalletManagerContractAdminSet represents a AdminSet event raised by the WalletManagerContract contract.
type WalletManagerContractAdminSet struct {
	Admin   common.Address
	Enabled bool
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterAdminSet is a free log retrieval operation binding the contract event 0xe68d2c359a771606c400cf8b87000cf5864010363d6a736e98f5047b7bbe18e9.
//
// Solidity: event AdminSet(address indexed admin, bool enabled)
func (_WalletManagerContract *WalletManagerContractFilterer) FilterAdminSet(opts *bind.FilterOpts, admin []common.Address) (*WalletManagerContractAdminSetIterator, error) {

	var adminRule []interface{}
	for _, adminItem := range admin {
		adminRule = append(adminRule, adminItem)
	}

	logs, sub, err := _WalletManagerContract.contract.FilterLogs(opts, "AdminSet", adminRule)
	if err != nil {
		return nil, err
	}
	return &WalletManagerContractAdminSetIterator{contract: _WalletManagerContract.contract, event: "AdminSet", logs: logs, sub: sub}, nil
}

// WatchAdminSet is a free log subscription operation binding the contract event 0xe68d2c359a771606c400cf8b87000cf5864010363d6a736e98f5047b7bbe18e9.
//
// Solidity: event AdminSet(address indexed admin, bool enabled)
func (_WalletManagerContract *WalletManagerContractFilterer) WatchAdminSet(opts *bind.WatchOpts, sink chan<- *WalletManagerContractAdminSet, admin []common.Address) (event.Subscription, error) {

	var adminRule []interface{}
	for _, adminItem := range admin {
		adminRule = append(adminRule, adminItem)
	}

	logs, sub, err := _WalletManagerContract.contract.WatchLogs(opts, "AdminSet", adminRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(WalletManagerContractAdminSet)
				if err := _WalletManagerContract.contract.UnpackLog(event, "AdminSet", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseAdminSet is a log parse operation binding the contract event 0xe68d2c359a771606c400cf8b87000cf5864010363d6a736e98f5047b7bbe18e9.
//
// Solidity: event AdminSet(address indexed admin, bool enabled)
func (_WalletManagerContract *WalletManagerContractFilterer) ParseAdminSet(log types.Log) (*WalletManagerContractAdminSet, error) {
	event := new(WalletManagerContractAdminSet)
	if err := _WalletManagerContract.contract.UnpackLog(event, "AdminSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// WalletManagerContractOwnerTransferredIterator is returned from FilterOwnerTransferred and is used to iterate over the raw logs and unpacked data for OwnerTransferred events raised by the WalletManagerContract contract.
type WalletManagerContractOwnerTransferredIterator struct {
	Event *WalletManagerContractOwnerTransferred // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *WalletManagerContractOwnerTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(WalletManagerContractOwnerTransferred)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(WalletManagerContractOwnerTransferred)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *WalletManagerContractOwnerTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *WalletManagerContractOwnerTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// WalletManagerContractOwnerTransferred represents a OwnerTransferred event raised by the WalletManagerContract contract.
type WalletManagerContractOwnerTransferred struct {
	OldOwner common.Address
	NewOwner common.Address
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterOwnerTransferred is a free log retrieval operation binding the contract event 0x8934ce4adea8d9ce0d714d2c22b86790e41b7731c84b926fbbdc1d40ff6533c9.
//
// Solidity: event OwnerTransferred(address indexed oldOwner, address indexed newOwner)
func (_WalletManagerContract *WalletManagerContractFilterer) FilterOwnerTransferred(opts *bind.FilterOpts, oldOwner []common.Address, newOwner []common.Address) (*WalletManagerContractOwnerTransferredIterator, error) {

	var oldOwnerRule []interface{}
	for _, oldOwnerItem := range oldOwner {
		oldOwnerRule = append(oldOwnerRule, oldOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _WalletManagerContract.contract.FilterLogs(opts, "OwnerTransferred", oldOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &WalletManagerContractOwnerTransferredIterator{contract: _WalletManagerContract.contract, event: "OwnerTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnerTransferred is a free log subscription operation binding the contract event 0x8934ce4adea8d9ce0d714d2c22b86790e41b7731c84b926fbbdc1d40ff6533c9.
//
// Solidity: event OwnerTransferred(address indexed oldOwner, address indexed newOwner)
func (_WalletManagerContract *WalletManagerContractFilterer) WatchOwnerTransferred(opts *bind.WatchOpts, sink chan<- *WalletManagerContractOwnerTransferred, oldOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var oldOwnerRule []interface{}
	for _, oldOwnerItem := range oldOwner {
		oldOwnerRule = append(oldOwnerRule, oldOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _WalletManagerContract.contract.WatchLogs(opts, "OwnerTransferred", oldOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(WalletManagerContractOwnerTransferred)
				if err := _WalletManagerContract.contract.UnpackLog(event, "OwnerTransferred", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseOwnerTransferred is a log parse operation binding the contract event 0x8934ce4adea8d9ce0d714d2c22b86790e41b7731c84b926fbbdc1d40ff6533c9.
//
// Solidity: event OwnerTransferred(address indexed oldOwner, address indexed newOwner)
func (_WalletManagerContract *WalletManagerContractFilterer) ParseOwnerTransferred(log types.Log) (*WalletManagerContractOwnerTransferred, error) {
	event := new(WalletManagerContractOwnerTransferred)
	if err := _WalletManagerContract.contract.UnpackLog(event, "OwnerTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// WalletManagerContractWalletAddedIterator is returned from FilterWalletAdded and is used to iterate over the raw logs and unpacked data for WalletAdded events raised by the WalletManagerContract contract.
type WalletManagerContractWalletAddedIterator struct {
	Event *WalletManagerContractWalletAdded // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *WalletManagerContractWalletAddedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(WalletManagerContractWalletAdded)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(WalletManagerContractWalletAdded)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *WalletManagerContractWalletAddedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *WalletManagerContractWalletAddedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// WalletManagerContractWalletAdded represents a WalletAdded event raised by the WalletManagerContract contract.
type WalletManagerContractWalletAdded struct {
	WalletIndex *big.Int
	Wallet      common.Address
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterWalletAdded is a free log retrieval operation binding the contract event 0xf0306f4e8668d23ebcc691f3a47412cb3e3bc7a86b5be28204cf62853914ff15.
//
// Solidity: event WalletAdded(uint256 indexed walletIndex, address wallet)
func (_WalletManagerContract *WalletManagerContractFilterer) FilterWalletAdded(opts *bind.FilterOpts, walletIndex []*big.Int) (*WalletManagerContractWalletAddedIterator, error) {

	var walletIndexRule []interface{}
	for _, walletIndexItem := range walletIndex {
		walletIndexRule = append(walletIndexRule, walletIndexItem)
	}

	logs, sub, err := _WalletManagerContract.contract.FilterLogs(opts, "WalletAdded", walletIndexRule)
	if err != nil {
		return nil, err
	}
	return &WalletManagerContractWalletAddedIterator{contract: _WalletManagerContract.contract, event: "WalletAdded", logs: logs, sub: sub}, nil
}

// WatchWalletAdded is a free log subscription operation binding the contract event 0xf0306f4e8668d23ebcc691f3a47412cb3e3bc7a86b5be28204cf62853914ff15.
//
// Solidity: event WalletAdded(uint256 indexed walletIndex, address wallet)
func (_WalletManagerContract *WalletManagerContractFilterer) WatchWalletAdded(opts *bind.WatchOpts, sink chan<- *WalletManagerContractWalletAdded, walletIndex []*big.Int) (event.Subscription, error) {

	var walletIndexRule []interface{}
	for _, walletIndexItem := range walletIndex {
		walletIndexRule = append(walletIndexRule, walletIndexItem)
	}

	logs, sub, err := _WalletManagerContract.contract.WatchLogs(opts, "WalletAdded", walletIndexRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(WalletManagerContractWalletAdded)
				if err := _WalletManagerContract.contract.UnpackLog(event, "WalletAdded", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseWalletAdded is a log parse operation binding the contract event 0xf0306f4e8668d23ebcc691f3a47412cb3e3bc7a86b5be28204cf62853914ff15.
//
// Solidity: event WalletAdded(uint256 indexed walletIndex, address wallet)
func (_WalletManagerContract *WalletManagerContractFilterer) ParseWalletAdded(log types.Log) (*WalletManagerContractWalletAdded, error) {
	event := new(WalletManagerContractWalletAdded)
	if err := _WalletManagerContract.contract.UnpackLog(event, "WalletAdded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// WalletManagerContractWalletAddressUpdatedIterator is returned from FilterWalletAddressUpdated and is used to iterate over the raw logs and unpacked data for WalletAddressUpdated events raised by the WalletManagerContract contract.
type WalletManagerContractWalletAddressUpdatedIterator struct {
	Event *WalletManagerContractWalletAddressUpdated // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *WalletManagerContractWalletAddressUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(WalletManagerContractWalletAddressUpdated)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(WalletManagerContractWalletAddressUpdated)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *WalletManagerContractWalletAddressUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *WalletManagerContractWalletAddressUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// WalletManagerContractWalletAddressUpdated represents a WalletAddressUpdated event raised by the WalletManagerContract contract.
type WalletManagerContractWalletAddressUpdated struct {
	WalletIndex *big.Int
	OldAddress  common.Address
	NewAddress  common.Address
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterWalletAddressUpdated is a free log retrieval operation binding the contract event 0x6644506d845ee53d9dbd59292c8e99200d9c182548a14d1e597c89d9fec9cd40.
//
// Solidity: event WalletAddressUpdated(uint256 indexed walletIndex, address indexed oldAddress, address indexed newAddress)
func (_WalletManagerContract *WalletManagerContractFilterer) FilterWalletAddressUpdated(opts *bind.FilterOpts, walletIndex []*big.Int, oldAddress []common.Address, newAddress []common.Address) (*WalletManagerContractWalletAddressUpdatedIterator, error) {

	var walletIndexRule []interface{}
	for _, walletIndexItem := range walletIndex {
		walletIndexRule = append(walletIndexRule, walletIndexItem)
	}
	var oldAddressRule []interface{}
	for _, oldAddressItem := range oldAddress {
		oldAddressRule = append(oldAddressRule, oldAddressItem)
	}
	var newAddressRule []interface{}
	for _, newAddressItem := range newAddress {
		newAddressRule = append(newAddressRule, newAddressItem)
	}

	logs, sub, err := _WalletManagerContract.contract.FilterLogs(opts, "WalletAddressUpdated", walletIndexRule, oldAddressRule, newAddressRule)
	if err != nil {
		return nil, err
	}
	return &WalletManagerContractWalletAddressUpdatedIterator{contract: _WalletManagerContract.contract, event: "WalletAddressUpdated", logs: logs, sub: sub}, nil
}

// WatchWalletAddressUpdated is a free log subscription operation binding the contract event 0x6644506d845ee53d9dbd59292c8e99200d9c182548a14d1e597c89d9fec9cd40.
//
// Solidity: event WalletAddressUpdated(uint256 indexed walletIndex, address indexed oldAddress, address indexed newAddress)
func (_WalletManagerContract *WalletManagerContractFilterer) WatchWalletAddressUpdated(opts *bind.WatchOpts, sink chan<- *WalletManagerContractWalletAddressUpdated, walletIndex []*big.Int, oldAddress []common.Address, newAddress []common.Address) (event.Subscription, error) {

	var walletIndexRule []interface{}
	for _, walletIndexItem := range walletIndex {
		walletIndexRule = append(walletIndexRule, walletIndexItem)
	}
	var oldAddressRule []interface{}
	for _, oldAddressItem := range oldAddress {
		oldAddressRule = append(oldAddressRule, oldAddressItem)
	}
	var newAddressRule []interface{}
	for _, newAddressItem := range newAddress {
		newAddressRule = append(newAddressRule, newAddressItem)
	}

	logs, sub, err := _WalletManagerContract.contract.WatchLogs(opts, "WalletAddressUpdated", walletIndexRule, oldAddressRule, newAddressRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(WalletManagerContractWalletAddressUpdated)
				if err := _WalletManagerContract.contract.UnpackLog(event, "WalletAddressUpdated", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseWalletAddressUpdated is a log parse operation binding the contract event 0x6644506d845ee53d9dbd59292c8e99200d9c182548a14d1e597c89d9fec9cd40.
//
// Solidity: event WalletAddressUpdated(uint256 indexed walletIndex, address indexed oldAddress, address indexed newAddress)
func (_WalletManagerContract *WalletManagerContractFilterer) ParseWalletAddressUpdated(log types.Log) (*WalletManagerContractWalletAddressUpdated, error) {
	event := new(WalletManagerContractWalletAddressUpdated)
	if err := _WalletManagerContract.contract.UnpackLog(event, "WalletAddressUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// WalletManagerContractWalletRevokedIterator is returned from FilterWalletRevoked and is used to iterate over the raw logs and unpacked data for WalletRevoked events raised by the WalletManagerContract contract.
type WalletManagerContractWalletRevokedIterator struct {
	Event *WalletManagerContractWalletRevoked // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *WalletManagerContractWalletRevokedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(WalletManagerContractWalletRevoked)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(WalletManagerContractWalletRevoked)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *WalletManagerContractWalletRevokedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *WalletManagerContractWalletRevokedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// WalletManagerContractWalletRevoked represents a WalletRevoked event raised by the WalletManagerContract contract.
type WalletManagerContractWalletRevoked struct {
	WalletIndex *big.Int
	IsActive    bool
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterWalletRevoked is a free log retrieval operation binding the contract event 0x4c4d4cd6c6b7a73a4f87d8df4cfdc6345f8c581f8edf19e5ccec3d7335d7ef4c.
//
// Solidity: event WalletRevoked(uint256 indexed walletIndex, bool isActive)
func (_WalletManagerContract *WalletManagerContractFilterer) FilterWalletRevoked(opts *bind.FilterOpts, walletIndex []*big.Int) (*WalletManagerContractWalletRevokedIterator, error) {

	var walletIndexRule []interface{}
	for _, walletIndexItem := range walletIndex {
		walletIndexRule = append(walletIndexRule, walletIndexItem)
	}

	logs, sub, err := _WalletManagerContract.contract.FilterLogs(opts, "WalletRevoked", walletIndexRule)
	if err != nil {
		return nil, err
	}
	return &WalletManagerContractWalletRevokedIterator{contract: _WalletManagerContract.contract, event: "WalletRevoked", logs: logs, sub: sub}, nil
}

// WatchWalletRevoked is a free log subscription operation binding the contract event 0x4c4d4cd6c6b7a73a4f87d8df4cfdc6345f8c581f8edf19e5ccec3d7335d7ef4c.
//
// Solidity: event WalletRevoked(uint256 indexed walletIndex, bool isActive)
func (_WalletManagerContract *WalletManagerContractFilterer) WatchWalletRevoked(opts *bind.WatchOpts, sink chan<- *WalletManagerContractWalletRevoked, walletIndex []*big.Int) (event.Subscription, error) {

	var walletIndexRule []interface{}
	for _, walletIndexItem := range walletIndex {
		walletIndexRule = append(walletIndexRule, walletIndexItem)
	}

	logs, sub, err := _WalletManagerContract.contract.WatchLogs(opts, "WalletRevoked", walletIndexRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(WalletManagerContractWalletRevoked)
				if err := _WalletManagerContract.contract.UnpackLog(event, "WalletRevoked", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseWalletRevoked is a log parse operation binding the contract event 0x4c4d4cd6c6b7a73a4f87d8df4cfdc6345f8c581f8edf19e5ccec3d7335d7ef4c.
//
// Solidity: event WalletRevoked(uint256 indexed walletIndex, bool isActive)
func (_WalletManagerContract *WalletManagerContractFilterer) ParseWalletRevoked(log types.Log) (*WalletManagerContractWalletRevoked, error) {
	event := new(WalletManagerContractWalletRevoked)
	if err := _WalletManagerContract.contract.UnpackLog(event, "WalletRevoked", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
