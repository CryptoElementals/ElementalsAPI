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

// TokenCollectorContractMetaData contains all meta data concerning the TokenCollectorContract contract.
var TokenCollectorContractMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"contractIERC20\",\"name\":\"token_\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalWallets_\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"currentWalletIndex_\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"token\",\"type\":\"address\"}],\"name\":\"SafeERC20FailedOperation\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"admin\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"enabled\",\"type\":\"bool\"}],\"name\":\"AdminSet\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"admin\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"Collected\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"minAmount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"maxAmount\",\"type\":\"uint256\"}],\"name\":\"DepositAmountRangeUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"playerId\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"newCredited\",\"type\":\"uint256\"}],\"name\":\"Deposited\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"oldOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnerTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"hash\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"computedHash\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"operator\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"}],\"name\":\"WithdrawalSignatureInvalid\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"operator\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"Withdrawn\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"uint256[]\",\"name\":\"playerIds\",\"type\":\"uint256[]\"},{\"internalType\":\"uint256[]\",\"name\":\"amounts\",\"type\":\"uint256[]\"},{\"internalType\":\"bytes[]\",\"name\":\"signatures\",\"type\":\"bytes[]\"}],\"name\":\"batchWithdraw\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"collect\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"}],\"name\":\"collectAll\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"credited\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"depositAddr\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"currentWalletIndex\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"playerId\",\"type\":\"uint256\"}],\"name\":\"deposit\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"playerId\",\"type\":\"uint256\"},{\"internalType\":\"uint8\",\"name\":\"v\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"r\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"s\",\"type\":\"bytes32\"}],\"name\":\"depositWithPermit\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"isAdmin\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"maxDepositAmount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"minDepositAmount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bool\",\"name\":\"isActive_\",\"type\":\"bool\"}],\"name\":\"setActive\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"admin\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"enabled\",\"type\":\"bool\"}],\"name\":\"setAdmin\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"walletIndex\",\"type\":\"uint256\"}],\"name\":\"setCurrentWalletIndex\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"minAmount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxAmount\",\"type\":\"uint256\"}],\"name\":\"setDepositAmountRange\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"totalWallets_\",\"type\":\"uint256\"}],\"name\":\"setTotalWallets\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"token\",\"outputs\":[{\"internalType\":\"contractIERC20\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"totalWallets\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"playerId\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"signature\",\"type\":\"bytes\"}],\"name\":\"withdraw\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// TokenCollectorContractABI is the input ABI used to generate the binding from.
// Deprecated: Use TokenCollectorContractMetaData.ABI instead.
var TokenCollectorContractABI = TokenCollectorContractMetaData.ABI

// TokenCollectorContract is an auto generated Go binding around an Ethereum contract.
type TokenCollectorContract struct {
	TokenCollectorContractCaller     // Read-only binding to the contract
	TokenCollectorContractTransactor // Write-only binding to the contract
	TokenCollectorContractFilterer   // Log filterer for contract events
}

// TokenCollectorContractCaller is an auto generated read-only Go binding around an Ethereum contract.
type TokenCollectorContractCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TokenCollectorContractTransactor is an auto generated write-only Go binding around an Ethereum contract.
type TokenCollectorContractTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TokenCollectorContractFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type TokenCollectorContractFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TokenCollectorContractSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type TokenCollectorContractSession struct {
	Contract     *TokenCollectorContract // Generic contract binding to set the session for
	CallOpts     bind.CallOpts           // Call options to use throughout this session
	TransactOpts bind.TransactOpts       // Transaction auth options to use throughout this session
}

// TokenCollectorContractCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type TokenCollectorContractCallerSession struct {
	Contract *TokenCollectorContractCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                 // Call options to use throughout this session
}

// TokenCollectorContractTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type TokenCollectorContractTransactorSession struct {
	Contract     *TokenCollectorContractTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                 // Transaction auth options to use throughout this session
}

// TokenCollectorContractRaw is an auto generated low-level Go binding around an Ethereum contract.
type TokenCollectorContractRaw struct {
	Contract *TokenCollectorContract // Generic contract binding to access the raw methods on
}

// TokenCollectorContractCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type TokenCollectorContractCallerRaw struct {
	Contract *TokenCollectorContractCaller // Generic read-only contract binding to access the raw methods on
}

// TokenCollectorContractTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type TokenCollectorContractTransactorRaw struct {
	Contract *TokenCollectorContractTransactor // Generic write-only contract binding to access the raw methods on
}

// NewTokenCollectorContract creates a new instance of TokenCollectorContract, bound to a specific deployed contract.
func NewTokenCollectorContract(address common.Address, backend bind.ContractBackend) (*TokenCollectorContract, error) {
	contract, err := bindTokenCollectorContract(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &TokenCollectorContract{TokenCollectorContractCaller: TokenCollectorContractCaller{contract: contract}, TokenCollectorContractTransactor: TokenCollectorContractTransactor{contract: contract}, TokenCollectorContractFilterer: TokenCollectorContractFilterer{contract: contract}}, nil
}

// NewTokenCollectorContractCaller creates a new read-only instance of TokenCollectorContract, bound to a specific deployed contract.
func NewTokenCollectorContractCaller(address common.Address, caller bind.ContractCaller) (*TokenCollectorContractCaller, error) {
	contract, err := bindTokenCollectorContract(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &TokenCollectorContractCaller{contract: contract}, nil
}

// NewTokenCollectorContractTransactor creates a new write-only instance of TokenCollectorContract, bound to a specific deployed contract.
func NewTokenCollectorContractTransactor(address common.Address, transactor bind.ContractTransactor) (*TokenCollectorContractTransactor, error) {
	contract, err := bindTokenCollectorContract(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &TokenCollectorContractTransactor{contract: contract}, nil
}

// NewTokenCollectorContractFilterer creates a new log filterer instance of TokenCollectorContract, bound to a specific deployed contract.
func NewTokenCollectorContractFilterer(address common.Address, filterer bind.ContractFilterer) (*TokenCollectorContractFilterer, error) {
	contract, err := bindTokenCollectorContract(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &TokenCollectorContractFilterer{contract: contract}, nil
}

// bindTokenCollectorContract binds a generic wrapper to an already deployed contract.
func bindTokenCollectorContract(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := TokenCollectorContractMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_TokenCollectorContract *TokenCollectorContractRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _TokenCollectorContract.Contract.TokenCollectorContractCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_TokenCollectorContract *TokenCollectorContractRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TokenCollectorContract.Contract.TokenCollectorContractTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_TokenCollectorContract *TokenCollectorContractRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _TokenCollectorContract.Contract.TokenCollectorContractTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_TokenCollectorContract *TokenCollectorContractCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _TokenCollectorContract.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_TokenCollectorContract *TokenCollectorContractTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TokenCollectorContract.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_TokenCollectorContract *TokenCollectorContractTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _TokenCollectorContract.Contract.contract.Transact(opts, method, params...)
}

// Credited is a free data retrieval call binding the contract method 0x7b5c88ba.
//
// Solidity: function credited(uint256 ) view returns(address depositAddr, uint256 amount)
func (_TokenCollectorContract *TokenCollectorContractCaller) Credited(opts *bind.CallOpts, arg0 *big.Int) (struct {
	DepositAddr common.Address
	Amount      *big.Int
}, error) {
	var out []interface{}
	err := _TokenCollectorContract.contract.Call(opts, &out, "credited", arg0)

	outstruct := new(struct {
		DepositAddr common.Address
		Amount      *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.DepositAddr = *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	outstruct.Amount = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// Credited is a free data retrieval call binding the contract method 0x7b5c88ba.
//
// Solidity: function credited(uint256 ) view returns(address depositAddr, uint256 amount)
func (_TokenCollectorContract *TokenCollectorContractSession) Credited(arg0 *big.Int) (struct {
	DepositAddr common.Address
	Amount      *big.Int
}, error) {
	return _TokenCollectorContract.Contract.Credited(&_TokenCollectorContract.CallOpts, arg0)
}

// Credited is a free data retrieval call binding the contract method 0x7b5c88ba.
//
// Solidity: function credited(uint256 ) view returns(address depositAddr, uint256 amount)
func (_TokenCollectorContract *TokenCollectorContractCallerSession) Credited(arg0 *big.Int) (struct {
	DepositAddr common.Address
	Amount      *big.Int
}, error) {
	return _TokenCollectorContract.Contract.Credited(&_TokenCollectorContract.CallOpts, arg0)
}

// CurrentWalletIndex is a free data retrieval call binding the contract method 0x6a2e651b.
//
// Solidity: function currentWalletIndex() view returns(uint256)
func (_TokenCollectorContract *TokenCollectorContractCaller) CurrentWalletIndex(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _TokenCollectorContract.contract.Call(opts, &out, "currentWalletIndex")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// CurrentWalletIndex is a free data retrieval call binding the contract method 0x6a2e651b.
//
// Solidity: function currentWalletIndex() view returns(uint256)
func (_TokenCollectorContract *TokenCollectorContractSession) CurrentWalletIndex() (*big.Int, error) {
	return _TokenCollectorContract.Contract.CurrentWalletIndex(&_TokenCollectorContract.CallOpts)
}

// CurrentWalletIndex is a free data retrieval call binding the contract method 0x6a2e651b.
//
// Solidity: function currentWalletIndex() view returns(uint256)
func (_TokenCollectorContract *TokenCollectorContractCallerSession) CurrentWalletIndex() (*big.Int, error) {
	return _TokenCollectorContract.Contract.CurrentWalletIndex(&_TokenCollectorContract.CallOpts)
}

// IsAdmin is a free data retrieval call binding the contract method 0x24d7806c.
//
// Solidity: function isAdmin(address ) view returns(bool)
func (_TokenCollectorContract *TokenCollectorContractCaller) IsAdmin(opts *bind.CallOpts, arg0 common.Address) (bool, error) {
	var out []interface{}
	err := _TokenCollectorContract.contract.Call(opts, &out, "isAdmin", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsAdmin is a free data retrieval call binding the contract method 0x24d7806c.
//
// Solidity: function isAdmin(address ) view returns(bool)
func (_TokenCollectorContract *TokenCollectorContractSession) IsAdmin(arg0 common.Address) (bool, error) {
	return _TokenCollectorContract.Contract.IsAdmin(&_TokenCollectorContract.CallOpts, arg0)
}

// IsAdmin is a free data retrieval call binding the contract method 0x24d7806c.
//
// Solidity: function isAdmin(address ) view returns(bool)
func (_TokenCollectorContract *TokenCollectorContractCallerSession) IsAdmin(arg0 common.Address) (bool, error) {
	return _TokenCollectorContract.Contract.IsAdmin(&_TokenCollectorContract.CallOpts, arg0)
}

// MaxDepositAmount is a free data retrieval call binding the contract method 0x8ed83271.
//
// Solidity: function maxDepositAmount() view returns(uint256)
func (_TokenCollectorContract *TokenCollectorContractCaller) MaxDepositAmount(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _TokenCollectorContract.contract.Call(opts, &out, "maxDepositAmount")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MaxDepositAmount is a free data retrieval call binding the contract method 0x8ed83271.
//
// Solidity: function maxDepositAmount() view returns(uint256)
func (_TokenCollectorContract *TokenCollectorContractSession) MaxDepositAmount() (*big.Int, error) {
	return _TokenCollectorContract.Contract.MaxDepositAmount(&_TokenCollectorContract.CallOpts)
}

// MaxDepositAmount is a free data retrieval call binding the contract method 0x8ed83271.
//
// Solidity: function maxDepositAmount() view returns(uint256)
func (_TokenCollectorContract *TokenCollectorContractCallerSession) MaxDepositAmount() (*big.Int, error) {
	return _TokenCollectorContract.Contract.MaxDepositAmount(&_TokenCollectorContract.CallOpts)
}

// MinDepositAmount is a free data retrieval call binding the contract method 0x645006ca.
//
// Solidity: function minDepositAmount() view returns(uint256)
func (_TokenCollectorContract *TokenCollectorContractCaller) MinDepositAmount(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _TokenCollectorContract.contract.Call(opts, &out, "minDepositAmount")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MinDepositAmount is a free data retrieval call binding the contract method 0x645006ca.
//
// Solidity: function minDepositAmount() view returns(uint256)
func (_TokenCollectorContract *TokenCollectorContractSession) MinDepositAmount() (*big.Int, error) {
	return _TokenCollectorContract.Contract.MinDepositAmount(&_TokenCollectorContract.CallOpts)
}

// MinDepositAmount is a free data retrieval call binding the contract method 0x645006ca.
//
// Solidity: function minDepositAmount() view returns(uint256)
func (_TokenCollectorContract *TokenCollectorContractCallerSession) MinDepositAmount() (*big.Int, error) {
	return _TokenCollectorContract.Contract.MinDepositAmount(&_TokenCollectorContract.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_TokenCollectorContract *TokenCollectorContractCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _TokenCollectorContract.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_TokenCollectorContract *TokenCollectorContractSession) Owner() (common.Address, error) {
	return _TokenCollectorContract.Contract.Owner(&_TokenCollectorContract.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_TokenCollectorContract *TokenCollectorContractCallerSession) Owner() (common.Address, error) {
	return _TokenCollectorContract.Contract.Owner(&_TokenCollectorContract.CallOpts)
}

// Token is a free data retrieval call binding the contract method 0xfc0c546a.
//
// Solidity: function token() view returns(address)
func (_TokenCollectorContract *TokenCollectorContractCaller) Token(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _TokenCollectorContract.contract.Call(opts, &out, "token")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Token is a free data retrieval call binding the contract method 0xfc0c546a.
//
// Solidity: function token() view returns(address)
func (_TokenCollectorContract *TokenCollectorContractSession) Token() (common.Address, error) {
	return _TokenCollectorContract.Contract.Token(&_TokenCollectorContract.CallOpts)
}

// Token is a free data retrieval call binding the contract method 0xfc0c546a.
//
// Solidity: function token() view returns(address)
func (_TokenCollectorContract *TokenCollectorContractCallerSession) Token() (common.Address, error) {
	return _TokenCollectorContract.Contract.Token(&_TokenCollectorContract.CallOpts)
}

// TotalWallets is a free data retrieval call binding the contract method 0x977d2c45.
//
// Solidity: function totalWallets() view returns(uint256)
func (_TokenCollectorContract *TokenCollectorContractCaller) TotalWallets(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _TokenCollectorContract.contract.Call(opts, &out, "totalWallets")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalWallets is a free data retrieval call binding the contract method 0x977d2c45.
//
// Solidity: function totalWallets() view returns(uint256)
func (_TokenCollectorContract *TokenCollectorContractSession) TotalWallets() (*big.Int, error) {
	return _TokenCollectorContract.Contract.TotalWallets(&_TokenCollectorContract.CallOpts)
}

// TotalWallets is a free data retrieval call binding the contract method 0x977d2c45.
//
// Solidity: function totalWallets() view returns(uint256)
func (_TokenCollectorContract *TokenCollectorContractCallerSession) TotalWallets() (*big.Int, error) {
	return _TokenCollectorContract.Contract.TotalWallets(&_TokenCollectorContract.CallOpts)
}

// BatchWithdraw is a paid mutator transaction binding the contract method 0x1d26155a.
//
// Solidity: function batchWithdraw(uint256[] playerIds, uint256[] amounts, bytes[] signatures) returns()
func (_TokenCollectorContract *TokenCollectorContractTransactor) BatchWithdraw(opts *bind.TransactOpts, playerIds []*big.Int, amounts []*big.Int, signatures [][]byte) (*types.Transaction, error) {
	return _TokenCollectorContract.contract.Transact(opts, "batchWithdraw", playerIds, amounts, signatures)
}

// BatchWithdraw is a paid mutator transaction binding the contract method 0x1d26155a.
//
// Solidity: function batchWithdraw(uint256[] playerIds, uint256[] amounts, bytes[] signatures) returns()
func (_TokenCollectorContract *TokenCollectorContractSession) BatchWithdraw(playerIds []*big.Int, amounts []*big.Int, signatures [][]byte) (*types.Transaction, error) {
	return _TokenCollectorContract.Contract.BatchWithdraw(&_TokenCollectorContract.TransactOpts, playerIds, amounts, signatures)
}

// BatchWithdraw is a paid mutator transaction binding the contract method 0x1d26155a.
//
// Solidity: function batchWithdraw(uint256[] playerIds, uint256[] amounts, bytes[] signatures) returns()
func (_TokenCollectorContract *TokenCollectorContractTransactorSession) BatchWithdraw(playerIds []*big.Int, amounts []*big.Int, signatures [][]byte) (*types.Transaction, error) {
	return _TokenCollectorContract.Contract.BatchWithdraw(&_TokenCollectorContract.TransactOpts, playerIds, amounts, signatures)
}

// Collect is a paid mutator transaction binding the contract method 0x6b8357ac.
//
// Solidity: function collect(address to, uint256 amount) returns()
func (_TokenCollectorContract *TokenCollectorContractTransactor) Collect(opts *bind.TransactOpts, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _TokenCollectorContract.contract.Transact(opts, "collect", to, amount)
}

// Collect is a paid mutator transaction binding the contract method 0x6b8357ac.
//
// Solidity: function collect(address to, uint256 amount) returns()
func (_TokenCollectorContract *TokenCollectorContractSession) Collect(to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _TokenCollectorContract.Contract.Collect(&_TokenCollectorContract.TransactOpts, to, amount)
}

// Collect is a paid mutator transaction binding the contract method 0x6b8357ac.
//
// Solidity: function collect(address to, uint256 amount) returns()
func (_TokenCollectorContract *TokenCollectorContractTransactorSession) Collect(to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _TokenCollectorContract.Contract.Collect(&_TokenCollectorContract.TransactOpts, to, amount)
}

// CollectAll is a paid mutator transaction binding the contract method 0xd657c9e7.
//
// Solidity: function collectAll(address to) returns(uint256 amount)
func (_TokenCollectorContract *TokenCollectorContractTransactor) CollectAll(opts *bind.TransactOpts, to common.Address) (*types.Transaction, error) {
	return _TokenCollectorContract.contract.Transact(opts, "collectAll", to)
}

// CollectAll is a paid mutator transaction binding the contract method 0xd657c9e7.
//
// Solidity: function collectAll(address to) returns(uint256 amount)
func (_TokenCollectorContract *TokenCollectorContractSession) CollectAll(to common.Address) (*types.Transaction, error) {
	return _TokenCollectorContract.Contract.CollectAll(&_TokenCollectorContract.TransactOpts, to)
}

// CollectAll is a paid mutator transaction binding the contract method 0xd657c9e7.
//
// Solidity: function collectAll(address to) returns(uint256 amount)
func (_TokenCollectorContract *TokenCollectorContractTransactorSession) CollectAll(to common.Address) (*types.Transaction, error) {
	return _TokenCollectorContract.Contract.CollectAll(&_TokenCollectorContract.TransactOpts, to)
}

// Deposit is a paid mutator transaction binding the contract method 0xe2bbb158.
//
// Solidity: function deposit(uint256 amount, uint256 playerId) returns()
func (_TokenCollectorContract *TokenCollectorContractTransactor) Deposit(opts *bind.TransactOpts, amount *big.Int, playerId *big.Int) (*types.Transaction, error) {
	return _TokenCollectorContract.contract.Transact(opts, "deposit", amount, playerId)
}

// Deposit is a paid mutator transaction binding the contract method 0xe2bbb158.
//
// Solidity: function deposit(uint256 amount, uint256 playerId) returns()
func (_TokenCollectorContract *TokenCollectorContractSession) Deposit(amount *big.Int, playerId *big.Int) (*types.Transaction, error) {
	return _TokenCollectorContract.Contract.Deposit(&_TokenCollectorContract.TransactOpts, amount, playerId)
}

// Deposit is a paid mutator transaction binding the contract method 0xe2bbb158.
//
// Solidity: function deposit(uint256 amount, uint256 playerId) returns()
func (_TokenCollectorContract *TokenCollectorContractTransactorSession) Deposit(amount *big.Int, playerId *big.Int) (*types.Transaction, error) {
	return _TokenCollectorContract.Contract.Deposit(&_TokenCollectorContract.TransactOpts, amount, playerId)
}

// DepositWithPermit is a paid mutator transaction binding the contract method 0x515bc323.
//
// Solidity: function depositWithPermit(uint256 amount, uint256 deadline, uint256 playerId, uint8 v, bytes32 r, bytes32 s) returns()
func (_TokenCollectorContract *TokenCollectorContractTransactor) DepositWithPermit(opts *bind.TransactOpts, amount *big.Int, deadline *big.Int, playerId *big.Int, v uint8, r [32]byte, s [32]byte) (*types.Transaction, error) {
	return _TokenCollectorContract.contract.Transact(opts, "depositWithPermit", amount, deadline, playerId, v, r, s)
}

// DepositWithPermit is a paid mutator transaction binding the contract method 0x515bc323.
//
// Solidity: function depositWithPermit(uint256 amount, uint256 deadline, uint256 playerId, uint8 v, bytes32 r, bytes32 s) returns()
func (_TokenCollectorContract *TokenCollectorContractSession) DepositWithPermit(amount *big.Int, deadline *big.Int, playerId *big.Int, v uint8, r [32]byte, s [32]byte) (*types.Transaction, error) {
	return _TokenCollectorContract.Contract.DepositWithPermit(&_TokenCollectorContract.TransactOpts, amount, deadline, playerId, v, r, s)
}

// DepositWithPermit is a paid mutator transaction binding the contract method 0x515bc323.
//
// Solidity: function depositWithPermit(uint256 amount, uint256 deadline, uint256 playerId, uint8 v, bytes32 r, bytes32 s) returns()
func (_TokenCollectorContract *TokenCollectorContractTransactorSession) DepositWithPermit(amount *big.Int, deadline *big.Int, playerId *big.Int, v uint8, r [32]byte, s [32]byte) (*types.Transaction, error) {
	return _TokenCollectorContract.Contract.DepositWithPermit(&_TokenCollectorContract.TransactOpts, amount, deadline, playerId, v, r, s)
}

// SetActive is a paid mutator transaction binding the contract method 0xacec338a.
//
// Solidity: function setActive(bool isActive_) returns()
func (_TokenCollectorContract *TokenCollectorContractTransactor) SetActive(opts *bind.TransactOpts, isActive_ bool) (*types.Transaction, error) {
	return _TokenCollectorContract.contract.Transact(opts, "setActive", isActive_)
}

// SetActive is a paid mutator transaction binding the contract method 0xacec338a.
//
// Solidity: function setActive(bool isActive_) returns()
func (_TokenCollectorContract *TokenCollectorContractSession) SetActive(isActive_ bool) (*types.Transaction, error) {
	return _TokenCollectorContract.Contract.SetActive(&_TokenCollectorContract.TransactOpts, isActive_)
}

// SetActive is a paid mutator transaction binding the contract method 0xacec338a.
//
// Solidity: function setActive(bool isActive_) returns()
func (_TokenCollectorContract *TokenCollectorContractTransactorSession) SetActive(isActive_ bool) (*types.Transaction, error) {
	return _TokenCollectorContract.Contract.SetActive(&_TokenCollectorContract.TransactOpts, isActive_)
}

// SetAdmin is a paid mutator transaction binding the contract method 0x4b0bddd2.
//
// Solidity: function setAdmin(address admin, bool enabled) returns()
func (_TokenCollectorContract *TokenCollectorContractTransactor) SetAdmin(opts *bind.TransactOpts, admin common.Address, enabled bool) (*types.Transaction, error) {
	return _TokenCollectorContract.contract.Transact(opts, "setAdmin", admin, enabled)
}

// SetAdmin is a paid mutator transaction binding the contract method 0x4b0bddd2.
//
// Solidity: function setAdmin(address admin, bool enabled) returns()
func (_TokenCollectorContract *TokenCollectorContractSession) SetAdmin(admin common.Address, enabled bool) (*types.Transaction, error) {
	return _TokenCollectorContract.Contract.SetAdmin(&_TokenCollectorContract.TransactOpts, admin, enabled)
}

// SetAdmin is a paid mutator transaction binding the contract method 0x4b0bddd2.
//
// Solidity: function setAdmin(address admin, bool enabled) returns()
func (_TokenCollectorContract *TokenCollectorContractTransactorSession) SetAdmin(admin common.Address, enabled bool) (*types.Transaction, error) {
	return _TokenCollectorContract.Contract.SetAdmin(&_TokenCollectorContract.TransactOpts, admin, enabled)
}

// SetCurrentWalletIndex is a paid mutator transaction binding the contract method 0xf92b63aa.
//
// Solidity: function setCurrentWalletIndex(uint256 walletIndex) returns()
func (_TokenCollectorContract *TokenCollectorContractTransactor) SetCurrentWalletIndex(opts *bind.TransactOpts, walletIndex *big.Int) (*types.Transaction, error) {
	return _TokenCollectorContract.contract.Transact(opts, "setCurrentWalletIndex", walletIndex)
}

// SetCurrentWalletIndex is a paid mutator transaction binding the contract method 0xf92b63aa.
//
// Solidity: function setCurrentWalletIndex(uint256 walletIndex) returns()
func (_TokenCollectorContract *TokenCollectorContractSession) SetCurrentWalletIndex(walletIndex *big.Int) (*types.Transaction, error) {
	return _TokenCollectorContract.Contract.SetCurrentWalletIndex(&_TokenCollectorContract.TransactOpts, walletIndex)
}

// SetCurrentWalletIndex is a paid mutator transaction binding the contract method 0xf92b63aa.
//
// Solidity: function setCurrentWalletIndex(uint256 walletIndex) returns()
func (_TokenCollectorContract *TokenCollectorContractTransactorSession) SetCurrentWalletIndex(walletIndex *big.Int) (*types.Transaction, error) {
	return _TokenCollectorContract.Contract.SetCurrentWalletIndex(&_TokenCollectorContract.TransactOpts, walletIndex)
}

// SetDepositAmountRange is a paid mutator transaction binding the contract method 0xab3c7531.
//
// Solidity: function setDepositAmountRange(uint256 minAmount, uint256 maxAmount) returns()
func (_TokenCollectorContract *TokenCollectorContractTransactor) SetDepositAmountRange(opts *bind.TransactOpts, minAmount *big.Int, maxAmount *big.Int) (*types.Transaction, error) {
	return _TokenCollectorContract.contract.Transact(opts, "setDepositAmountRange", minAmount, maxAmount)
}

// SetDepositAmountRange is a paid mutator transaction binding the contract method 0xab3c7531.
//
// Solidity: function setDepositAmountRange(uint256 minAmount, uint256 maxAmount) returns()
func (_TokenCollectorContract *TokenCollectorContractSession) SetDepositAmountRange(minAmount *big.Int, maxAmount *big.Int) (*types.Transaction, error) {
	return _TokenCollectorContract.Contract.SetDepositAmountRange(&_TokenCollectorContract.TransactOpts, minAmount, maxAmount)
}

// SetDepositAmountRange is a paid mutator transaction binding the contract method 0xab3c7531.
//
// Solidity: function setDepositAmountRange(uint256 minAmount, uint256 maxAmount) returns()
func (_TokenCollectorContract *TokenCollectorContractTransactorSession) SetDepositAmountRange(minAmount *big.Int, maxAmount *big.Int) (*types.Transaction, error) {
	return _TokenCollectorContract.Contract.SetDepositAmountRange(&_TokenCollectorContract.TransactOpts, minAmount, maxAmount)
}

// SetTotalWallets is a paid mutator transaction binding the contract method 0x8b1fd5c1.
//
// Solidity: function setTotalWallets(uint256 totalWallets_) returns()
func (_TokenCollectorContract *TokenCollectorContractTransactor) SetTotalWallets(opts *bind.TransactOpts, totalWallets_ *big.Int) (*types.Transaction, error) {
	return _TokenCollectorContract.contract.Transact(opts, "setTotalWallets", totalWallets_)
}

// SetTotalWallets is a paid mutator transaction binding the contract method 0x8b1fd5c1.
//
// Solidity: function setTotalWallets(uint256 totalWallets_) returns()
func (_TokenCollectorContract *TokenCollectorContractSession) SetTotalWallets(totalWallets_ *big.Int) (*types.Transaction, error) {
	return _TokenCollectorContract.Contract.SetTotalWallets(&_TokenCollectorContract.TransactOpts, totalWallets_)
}

// SetTotalWallets is a paid mutator transaction binding the contract method 0x8b1fd5c1.
//
// Solidity: function setTotalWallets(uint256 totalWallets_) returns()
func (_TokenCollectorContract *TokenCollectorContractTransactorSession) SetTotalWallets(totalWallets_ *big.Int) (*types.Transaction, error) {
	return _TokenCollectorContract.Contract.SetTotalWallets(&_TokenCollectorContract.TransactOpts, totalWallets_)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_TokenCollectorContract *TokenCollectorContractTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _TokenCollectorContract.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_TokenCollectorContract *TokenCollectorContractSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _TokenCollectorContract.Contract.TransferOwnership(&_TokenCollectorContract.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_TokenCollectorContract *TokenCollectorContractTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _TokenCollectorContract.Contract.TransferOwnership(&_TokenCollectorContract.TransactOpts, newOwner)
}

// Withdraw is a paid mutator transaction binding the contract method 0x744fb6ca.
//
// Solidity: function withdraw(uint256 playerId, uint256 amount, bytes signature) returns(bool)
func (_TokenCollectorContract *TokenCollectorContractTransactor) Withdraw(opts *bind.TransactOpts, playerId *big.Int, amount *big.Int, signature []byte) (*types.Transaction, error) {
	return _TokenCollectorContract.contract.Transact(opts, "withdraw", playerId, amount, signature)
}

// Withdraw is a paid mutator transaction binding the contract method 0x744fb6ca.
//
// Solidity: function withdraw(uint256 playerId, uint256 amount, bytes signature) returns(bool)
func (_TokenCollectorContract *TokenCollectorContractSession) Withdraw(playerId *big.Int, amount *big.Int, signature []byte) (*types.Transaction, error) {
	return _TokenCollectorContract.Contract.Withdraw(&_TokenCollectorContract.TransactOpts, playerId, amount, signature)
}

// Withdraw is a paid mutator transaction binding the contract method 0x744fb6ca.
//
// Solidity: function withdraw(uint256 playerId, uint256 amount, bytes signature) returns(bool)
func (_TokenCollectorContract *TokenCollectorContractTransactorSession) Withdraw(playerId *big.Int, amount *big.Int, signature []byte) (*types.Transaction, error) {
	return _TokenCollectorContract.Contract.Withdraw(&_TokenCollectorContract.TransactOpts, playerId, amount, signature)
}

// TokenCollectorContractAdminSetIterator is returned from FilterAdminSet and is used to iterate over the raw logs and unpacked data for AdminSet events raised by the TokenCollectorContract contract.
type TokenCollectorContractAdminSetIterator struct {
	Event *TokenCollectorContractAdminSet // Event containing the contract specifics and raw log

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
func (it *TokenCollectorContractAdminSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TokenCollectorContractAdminSet)
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
		it.Event = new(TokenCollectorContractAdminSet)
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
func (it *TokenCollectorContractAdminSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TokenCollectorContractAdminSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TokenCollectorContractAdminSet represents a AdminSet event raised by the TokenCollectorContract contract.
type TokenCollectorContractAdminSet struct {
	Admin   common.Address
	Enabled bool
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterAdminSet is a free log retrieval operation binding the contract event 0xe68d2c359a771606c400cf8b87000cf5864010363d6a736e98f5047b7bbe18e9.
//
// Solidity: event AdminSet(address indexed admin, bool enabled)
func (_TokenCollectorContract *TokenCollectorContractFilterer) FilterAdminSet(opts *bind.FilterOpts, admin []common.Address) (*TokenCollectorContractAdminSetIterator, error) {

	var adminRule []interface{}
	for _, adminItem := range admin {
		adminRule = append(adminRule, adminItem)
	}

	logs, sub, err := _TokenCollectorContract.contract.FilterLogs(opts, "AdminSet", adminRule)
	if err != nil {
		return nil, err
	}
	return &TokenCollectorContractAdminSetIterator{contract: _TokenCollectorContract.contract, event: "AdminSet", logs: logs, sub: sub}, nil
}

// WatchAdminSet is a free log subscription operation binding the contract event 0xe68d2c359a771606c400cf8b87000cf5864010363d6a736e98f5047b7bbe18e9.
//
// Solidity: event AdminSet(address indexed admin, bool enabled)
func (_TokenCollectorContract *TokenCollectorContractFilterer) WatchAdminSet(opts *bind.WatchOpts, sink chan<- *TokenCollectorContractAdminSet, admin []common.Address) (event.Subscription, error) {

	var adminRule []interface{}
	for _, adminItem := range admin {
		adminRule = append(adminRule, adminItem)
	}

	logs, sub, err := _TokenCollectorContract.contract.WatchLogs(opts, "AdminSet", adminRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TokenCollectorContractAdminSet)
				if err := _TokenCollectorContract.contract.UnpackLog(event, "AdminSet", log); err != nil {
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
func (_TokenCollectorContract *TokenCollectorContractFilterer) ParseAdminSet(log types.Log) (*TokenCollectorContractAdminSet, error) {
	event := new(TokenCollectorContractAdminSet)
	if err := _TokenCollectorContract.contract.UnpackLog(event, "AdminSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TokenCollectorContractCollectedIterator is returned from FilterCollected and is used to iterate over the raw logs and unpacked data for Collected events raised by the TokenCollectorContract contract.
type TokenCollectorContractCollectedIterator struct {
	Event *TokenCollectorContractCollected // Event containing the contract specifics and raw log

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
func (it *TokenCollectorContractCollectedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TokenCollectorContractCollected)
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
		it.Event = new(TokenCollectorContractCollected)
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
func (it *TokenCollectorContractCollectedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TokenCollectorContractCollectedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TokenCollectorContractCollected represents a Collected event raised by the TokenCollectorContract contract.
type TokenCollectorContractCollected struct {
	Admin  common.Address
	To     common.Address
	Amount *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterCollected is a free log retrieval operation binding the contract event 0x484decdc1e9549e1866295f6f86c889ded3f7de410e7488a7a415978589dc8fd.
//
// Solidity: event Collected(address indexed admin, address indexed to, uint256 amount)
func (_TokenCollectorContract *TokenCollectorContractFilterer) FilterCollected(opts *bind.FilterOpts, admin []common.Address, to []common.Address) (*TokenCollectorContractCollectedIterator, error) {

	var adminRule []interface{}
	for _, adminItem := range admin {
		adminRule = append(adminRule, adminItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _TokenCollectorContract.contract.FilterLogs(opts, "Collected", adminRule, toRule)
	if err != nil {
		return nil, err
	}
	return &TokenCollectorContractCollectedIterator{contract: _TokenCollectorContract.contract, event: "Collected", logs: logs, sub: sub}, nil
}

// WatchCollected is a free log subscription operation binding the contract event 0x484decdc1e9549e1866295f6f86c889ded3f7de410e7488a7a415978589dc8fd.
//
// Solidity: event Collected(address indexed admin, address indexed to, uint256 amount)
func (_TokenCollectorContract *TokenCollectorContractFilterer) WatchCollected(opts *bind.WatchOpts, sink chan<- *TokenCollectorContractCollected, admin []common.Address, to []common.Address) (event.Subscription, error) {

	var adminRule []interface{}
	for _, adminItem := range admin {
		adminRule = append(adminRule, adminItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _TokenCollectorContract.contract.WatchLogs(opts, "Collected", adminRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TokenCollectorContractCollected)
				if err := _TokenCollectorContract.contract.UnpackLog(event, "Collected", log); err != nil {
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

// ParseCollected is a log parse operation binding the contract event 0x484decdc1e9549e1866295f6f86c889ded3f7de410e7488a7a415978589dc8fd.
//
// Solidity: event Collected(address indexed admin, address indexed to, uint256 amount)
func (_TokenCollectorContract *TokenCollectorContractFilterer) ParseCollected(log types.Log) (*TokenCollectorContractCollected, error) {
	event := new(TokenCollectorContractCollected)
	if err := _TokenCollectorContract.contract.UnpackLog(event, "Collected", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TokenCollectorContractDepositAmountRangeUpdatedIterator is returned from FilterDepositAmountRangeUpdated and is used to iterate over the raw logs and unpacked data for DepositAmountRangeUpdated events raised by the TokenCollectorContract contract.
type TokenCollectorContractDepositAmountRangeUpdatedIterator struct {
	Event *TokenCollectorContractDepositAmountRangeUpdated // Event containing the contract specifics and raw log

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
func (it *TokenCollectorContractDepositAmountRangeUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TokenCollectorContractDepositAmountRangeUpdated)
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
		it.Event = new(TokenCollectorContractDepositAmountRangeUpdated)
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
func (it *TokenCollectorContractDepositAmountRangeUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TokenCollectorContractDepositAmountRangeUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TokenCollectorContractDepositAmountRangeUpdated represents a DepositAmountRangeUpdated event raised by the TokenCollectorContract contract.
type TokenCollectorContractDepositAmountRangeUpdated struct {
	MinAmount *big.Int
	MaxAmount *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterDepositAmountRangeUpdated is a free log retrieval operation binding the contract event 0x0706621053d284ae183f9d874c7cbcf50f783d7bd66a59cbb4ceb314038dfd2c.
//
// Solidity: event DepositAmountRangeUpdated(uint256 minAmount, uint256 maxAmount)
func (_TokenCollectorContract *TokenCollectorContractFilterer) FilterDepositAmountRangeUpdated(opts *bind.FilterOpts) (*TokenCollectorContractDepositAmountRangeUpdatedIterator, error) {

	logs, sub, err := _TokenCollectorContract.contract.FilterLogs(opts, "DepositAmountRangeUpdated")
	if err != nil {
		return nil, err
	}
	return &TokenCollectorContractDepositAmountRangeUpdatedIterator{contract: _TokenCollectorContract.contract, event: "DepositAmountRangeUpdated", logs: logs, sub: sub}, nil
}

// WatchDepositAmountRangeUpdated is a free log subscription operation binding the contract event 0x0706621053d284ae183f9d874c7cbcf50f783d7bd66a59cbb4ceb314038dfd2c.
//
// Solidity: event DepositAmountRangeUpdated(uint256 minAmount, uint256 maxAmount)
func (_TokenCollectorContract *TokenCollectorContractFilterer) WatchDepositAmountRangeUpdated(opts *bind.WatchOpts, sink chan<- *TokenCollectorContractDepositAmountRangeUpdated) (event.Subscription, error) {

	logs, sub, err := _TokenCollectorContract.contract.WatchLogs(opts, "DepositAmountRangeUpdated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TokenCollectorContractDepositAmountRangeUpdated)
				if err := _TokenCollectorContract.contract.UnpackLog(event, "DepositAmountRangeUpdated", log); err != nil {
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

// ParseDepositAmountRangeUpdated is a log parse operation binding the contract event 0x0706621053d284ae183f9d874c7cbcf50f783d7bd66a59cbb4ceb314038dfd2c.
//
// Solidity: event DepositAmountRangeUpdated(uint256 minAmount, uint256 maxAmount)
func (_TokenCollectorContract *TokenCollectorContractFilterer) ParseDepositAmountRangeUpdated(log types.Log) (*TokenCollectorContractDepositAmountRangeUpdated, error) {
	event := new(TokenCollectorContractDepositAmountRangeUpdated)
	if err := _TokenCollectorContract.contract.UnpackLog(event, "DepositAmountRangeUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TokenCollectorContractDepositedIterator is returned from FilterDeposited and is used to iterate over the raw logs and unpacked data for Deposited events raised by the TokenCollectorContract contract.
type TokenCollectorContractDepositedIterator struct {
	Event *TokenCollectorContractDeposited // Event containing the contract specifics and raw log

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
func (it *TokenCollectorContractDepositedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TokenCollectorContractDeposited)
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
		it.Event = new(TokenCollectorContractDeposited)
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
func (it *TokenCollectorContractDepositedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TokenCollectorContractDepositedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TokenCollectorContractDeposited represents a Deposited event raised by the TokenCollectorContract contract.
type TokenCollectorContractDeposited struct {
	From        common.Address
	PlayerId    *big.Int
	Amount      *big.Int
	NewCredited *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterDeposited is a free log retrieval operation binding the contract event 0x91ede45f04a37a7c170f5c1207df3b6bc748dc1e04ad5e917a241d0f52feada3.
//
// Solidity: event Deposited(address indexed from, uint256 indexed playerId, uint256 amount, uint256 newCredited)
func (_TokenCollectorContract *TokenCollectorContractFilterer) FilterDeposited(opts *bind.FilterOpts, from []common.Address, playerId []*big.Int) (*TokenCollectorContractDepositedIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var playerIdRule []interface{}
	for _, playerIdItem := range playerId {
		playerIdRule = append(playerIdRule, playerIdItem)
	}

	logs, sub, err := _TokenCollectorContract.contract.FilterLogs(opts, "Deposited", fromRule, playerIdRule)
	if err != nil {
		return nil, err
	}
	return &TokenCollectorContractDepositedIterator{contract: _TokenCollectorContract.contract, event: "Deposited", logs: logs, sub: sub}, nil
}

// WatchDeposited is a free log subscription operation binding the contract event 0x91ede45f04a37a7c170f5c1207df3b6bc748dc1e04ad5e917a241d0f52feada3.
//
// Solidity: event Deposited(address indexed from, uint256 indexed playerId, uint256 amount, uint256 newCredited)
func (_TokenCollectorContract *TokenCollectorContractFilterer) WatchDeposited(opts *bind.WatchOpts, sink chan<- *TokenCollectorContractDeposited, from []common.Address, playerId []*big.Int) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var playerIdRule []interface{}
	for _, playerIdItem := range playerId {
		playerIdRule = append(playerIdRule, playerIdItem)
	}

	logs, sub, err := _TokenCollectorContract.contract.WatchLogs(opts, "Deposited", fromRule, playerIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TokenCollectorContractDeposited)
				if err := _TokenCollectorContract.contract.UnpackLog(event, "Deposited", log); err != nil {
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

// ParseDeposited is a log parse operation binding the contract event 0x91ede45f04a37a7c170f5c1207df3b6bc748dc1e04ad5e917a241d0f52feada3.
//
// Solidity: event Deposited(address indexed from, uint256 indexed playerId, uint256 amount, uint256 newCredited)
func (_TokenCollectorContract *TokenCollectorContractFilterer) ParseDeposited(log types.Log) (*TokenCollectorContractDeposited, error) {
	event := new(TokenCollectorContractDeposited)
	if err := _TokenCollectorContract.contract.UnpackLog(event, "Deposited", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TokenCollectorContractOwnerTransferredIterator is returned from FilterOwnerTransferred and is used to iterate over the raw logs and unpacked data for OwnerTransferred events raised by the TokenCollectorContract contract.
type TokenCollectorContractOwnerTransferredIterator struct {
	Event *TokenCollectorContractOwnerTransferred // Event containing the contract specifics and raw log

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
func (it *TokenCollectorContractOwnerTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TokenCollectorContractOwnerTransferred)
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
		it.Event = new(TokenCollectorContractOwnerTransferred)
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
func (it *TokenCollectorContractOwnerTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TokenCollectorContractOwnerTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TokenCollectorContractOwnerTransferred represents a OwnerTransferred event raised by the TokenCollectorContract contract.
type TokenCollectorContractOwnerTransferred struct {
	OldOwner common.Address
	NewOwner common.Address
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterOwnerTransferred is a free log retrieval operation binding the contract event 0x8934ce4adea8d9ce0d714d2c22b86790e41b7731c84b926fbbdc1d40ff6533c9.
//
// Solidity: event OwnerTransferred(address indexed oldOwner, address indexed newOwner)
func (_TokenCollectorContract *TokenCollectorContractFilterer) FilterOwnerTransferred(opts *bind.FilterOpts, oldOwner []common.Address, newOwner []common.Address) (*TokenCollectorContractOwnerTransferredIterator, error) {

	var oldOwnerRule []interface{}
	for _, oldOwnerItem := range oldOwner {
		oldOwnerRule = append(oldOwnerRule, oldOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _TokenCollectorContract.contract.FilterLogs(opts, "OwnerTransferred", oldOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &TokenCollectorContractOwnerTransferredIterator{contract: _TokenCollectorContract.contract, event: "OwnerTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnerTransferred is a free log subscription operation binding the contract event 0x8934ce4adea8d9ce0d714d2c22b86790e41b7731c84b926fbbdc1d40ff6533c9.
//
// Solidity: event OwnerTransferred(address indexed oldOwner, address indexed newOwner)
func (_TokenCollectorContract *TokenCollectorContractFilterer) WatchOwnerTransferred(opts *bind.WatchOpts, sink chan<- *TokenCollectorContractOwnerTransferred, oldOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var oldOwnerRule []interface{}
	for _, oldOwnerItem := range oldOwner {
		oldOwnerRule = append(oldOwnerRule, oldOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _TokenCollectorContract.contract.WatchLogs(opts, "OwnerTransferred", oldOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TokenCollectorContractOwnerTransferred)
				if err := _TokenCollectorContract.contract.UnpackLog(event, "OwnerTransferred", log); err != nil {
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
func (_TokenCollectorContract *TokenCollectorContractFilterer) ParseOwnerTransferred(log types.Log) (*TokenCollectorContractOwnerTransferred, error) {
	event := new(TokenCollectorContractOwnerTransferred)
	if err := _TokenCollectorContract.contract.UnpackLog(event, "OwnerTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TokenCollectorContractWithdrawalSignatureInvalidIterator is returned from FilterWithdrawalSignatureInvalid and is used to iterate over the raw logs and unpacked data for WithdrawalSignatureInvalid events raised by the TokenCollectorContract contract.
type TokenCollectorContractWithdrawalSignatureInvalidIterator struct {
	Event *TokenCollectorContractWithdrawalSignatureInvalid // Event containing the contract specifics and raw log

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
func (it *TokenCollectorContractWithdrawalSignatureInvalidIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TokenCollectorContractWithdrawalSignatureInvalid)
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
		it.Event = new(TokenCollectorContractWithdrawalSignatureInvalid)
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
func (it *TokenCollectorContractWithdrawalSignatureInvalidIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TokenCollectorContractWithdrawalSignatureInvalidIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TokenCollectorContractWithdrawalSignatureInvalid represents a WithdrawalSignatureInvalid event raised by the TokenCollectorContract contract.
type TokenCollectorContractWithdrawalSignatureInvalid struct {
	Hash         [32]byte
	ComputedHash [32]byte
	Operator     common.Address
	Success      bool
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterWithdrawalSignatureInvalid is a free log retrieval operation binding the contract event 0x42eb2a24e8bf8db6e66f20fbab8b3c8cfa78c982e1c07c2ba5cbe0c0f1b8b8a2.
//
// Solidity: event WithdrawalSignatureInvalid(bytes32 indexed hash, bytes32 indexed computedHash, address indexed operator, bool success)
func (_TokenCollectorContract *TokenCollectorContractFilterer) FilterWithdrawalSignatureInvalid(opts *bind.FilterOpts, hash [][32]byte, computedHash [][32]byte, operator []common.Address) (*TokenCollectorContractWithdrawalSignatureInvalidIterator, error) {

	var hashRule []interface{}
	for _, hashItem := range hash {
		hashRule = append(hashRule, hashItem)
	}
	var computedHashRule []interface{}
	for _, computedHashItem := range computedHash {
		computedHashRule = append(computedHashRule, computedHashItem)
	}
	var operatorRule []interface{}
	for _, operatorItem := range operator {
		operatorRule = append(operatorRule, operatorItem)
	}

	logs, sub, err := _TokenCollectorContract.contract.FilterLogs(opts, "WithdrawalSignatureInvalid", hashRule, computedHashRule, operatorRule)
	if err != nil {
		return nil, err
	}
	return &TokenCollectorContractWithdrawalSignatureInvalidIterator{contract: _TokenCollectorContract.contract, event: "WithdrawalSignatureInvalid", logs: logs, sub: sub}, nil
}

// WatchWithdrawalSignatureInvalid is a free log subscription operation binding the contract event 0x42eb2a24e8bf8db6e66f20fbab8b3c8cfa78c982e1c07c2ba5cbe0c0f1b8b8a2.
//
// Solidity: event WithdrawalSignatureInvalid(bytes32 indexed hash, bytes32 indexed computedHash, address indexed operator, bool success)
func (_TokenCollectorContract *TokenCollectorContractFilterer) WatchWithdrawalSignatureInvalid(opts *bind.WatchOpts, sink chan<- *TokenCollectorContractWithdrawalSignatureInvalid, hash [][32]byte, computedHash [][32]byte, operator []common.Address) (event.Subscription, error) {

	var hashRule []interface{}
	for _, hashItem := range hash {
		hashRule = append(hashRule, hashItem)
	}
	var computedHashRule []interface{}
	for _, computedHashItem := range computedHash {
		computedHashRule = append(computedHashRule, computedHashItem)
	}
	var operatorRule []interface{}
	for _, operatorItem := range operator {
		operatorRule = append(operatorRule, operatorItem)
	}

	logs, sub, err := _TokenCollectorContract.contract.WatchLogs(opts, "WithdrawalSignatureInvalid", hashRule, computedHashRule, operatorRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TokenCollectorContractWithdrawalSignatureInvalid)
				if err := _TokenCollectorContract.contract.UnpackLog(event, "WithdrawalSignatureInvalid", log); err != nil {
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

// ParseWithdrawalSignatureInvalid is a log parse operation binding the contract event 0x42eb2a24e8bf8db6e66f20fbab8b3c8cfa78c982e1c07c2ba5cbe0c0f1b8b8a2.
//
// Solidity: event WithdrawalSignatureInvalid(bytes32 indexed hash, bytes32 indexed computedHash, address indexed operator, bool success)
func (_TokenCollectorContract *TokenCollectorContractFilterer) ParseWithdrawalSignatureInvalid(log types.Log) (*TokenCollectorContractWithdrawalSignatureInvalid, error) {
	event := new(TokenCollectorContractWithdrawalSignatureInvalid)
	if err := _TokenCollectorContract.contract.UnpackLog(event, "WithdrawalSignatureInvalid", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TokenCollectorContractWithdrawnIterator is returned from FilterWithdrawn and is used to iterate over the raw logs and unpacked data for Withdrawn events raised by the TokenCollectorContract contract.
type TokenCollectorContractWithdrawnIterator struct {
	Event *TokenCollectorContractWithdrawn // Event containing the contract specifics and raw log

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
func (it *TokenCollectorContractWithdrawnIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TokenCollectorContractWithdrawn)
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
		it.Event = new(TokenCollectorContractWithdrawn)
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
func (it *TokenCollectorContractWithdrawnIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TokenCollectorContractWithdrawnIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TokenCollectorContractWithdrawn represents a Withdrawn event raised by the TokenCollectorContract contract.
type TokenCollectorContractWithdrawn struct {
	Operator common.Address
	To       common.Address
	Amount   *big.Int
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterWithdrawn is a free log retrieval operation binding the contract event 0xd1c19fbcd4551a5edfb66d43d2e337c04837afda3482b42bdf569a8fccdae5fb.
//
// Solidity: event Withdrawn(address indexed operator, address indexed to, uint256 amount)
func (_TokenCollectorContract *TokenCollectorContractFilterer) FilterWithdrawn(opts *bind.FilterOpts, operator []common.Address, to []common.Address) (*TokenCollectorContractWithdrawnIterator, error) {

	var operatorRule []interface{}
	for _, operatorItem := range operator {
		operatorRule = append(operatorRule, operatorItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _TokenCollectorContract.contract.FilterLogs(opts, "Withdrawn", operatorRule, toRule)
	if err != nil {
		return nil, err
	}
	return &TokenCollectorContractWithdrawnIterator{contract: _TokenCollectorContract.contract, event: "Withdrawn", logs: logs, sub: sub}, nil
}

// WatchWithdrawn is a free log subscription operation binding the contract event 0xd1c19fbcd4551a5edfb66d43d2e337c04837afda3482b42bdf569a8fccdae5fb.
//
// Solidity: event Withdrawn(address indexed operator, address indexed to, uint256 amount)
func (_TokenCollectorContract *TokenCollectorContractFilterer) WatchWithdrawn(opts *bind.WatchOpts, sink chan<- *TokenCollectorContractWithdrawn, operator []common.Address, to []common.Address) (event.Subscription, error) {

	var operatorRule []interface{}
	for _, operatorItem := range operator {
		operatorRule = append(operatorRule, operatorItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _TokenCollectorContract.contract.WatchLogs(opts, "Withdrawn", operatorRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TokenCollectorContractWithdrawn)
				if err := _TokenCollectorContract.contract.UnpackLog(event, "Withdrawn", log); err != nil {
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

// ParseWithdrawn is a log parse operation binding the contract event 0xd1c19fbcd4551a5edfb66d43d2e337c04837afda3482b42bdf569a8fccdae5fb.
//
// Solidity: event Withdrawn(address indexed operator, address indexed to, uint256 amount)
func (_TokenCollectorContract *TokenCollectorContractFilterer) ParseWithdrawn(log types.Log) (*TokenCollectorContractWithdrawn, error) {
	event := new(TokenCollectorContractWithdrawn)
	if err := _TokenCollectorContract.contract.UnpackLog(event, "Withdrawn", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
