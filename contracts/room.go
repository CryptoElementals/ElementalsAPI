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

// RoomContractMetaData contains all meta data concerning the RoomContract contract.
var RoomContractMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_player1\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_player2\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_roundTimeout\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_totalRound\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"_creator\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_initialHP\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"round\",\"type\":\"uint256\"}],\"name\":\"startANewRound\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"player\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"cards\",\"type\":\"string\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"salt\",\"type\":\"string\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"round\",\"type\":\"uint256\"}],\"name\":\"submitCards\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"player\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"cardsHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"round\",\"type\":\"uint256\"}],\"name\":\"submitCardsHash\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"StartANewRound\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"_cards\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"_salt\",\"type\":\"string\"},{\"internalType\":\"uint256\",\"name\":\"_round\",\"type\":\"uint256\"}],\"name\":\"SubmitCards\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_cardsHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"_round\",\"type\":\"uint256\"}],\"name\":\"SubmitCardsHash\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"cardsHash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"creator\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"currentRound\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"initialHP\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"launchTime\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"player1\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"player2\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"roundTimeout\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"totalRound\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// RoomContractABI is the input ABI used to generate the binding from.
// Deprecated: Use RoomContractMetaData.ABI instead.
var RoomContractABI = RoomContractMetaData.ABI

// RoomContract is an auto generated Go binding around an Ethereum contract.
type RoomContract struct {
	RoomContractCaller     // Read-only binding to the contract
	RoomContractTransactor // Write-only binding to the contract
	RoomContractFilterer   // Log filterer for contract events
}

// RoomContractCaller is an auto generated read-only Go binding around an Ethereum contract.
type RoomContractCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RoomContractTransactor is an auto generated write-only Go binding around an Ethereum contract.
type RoomContractTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RoomContractFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type RoomContractFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RoomContractSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type RoomContractSession struct {
	Contract     *RoomContract     // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// RoomContractCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type RoomContractCallerSession struct {
	Contract *RoomContractCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts       // Call options to use throughout this session
}

// RoomContractTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type RoomContractTransactorSession struct {
	Contract     *RoomContractTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts       // Transaction auth options to use throughout this session
}

// RoomContractRaw is an auto generated low-level Go binding around an Ethereum contract.
type RoomContractRaw struct {
	Contract *RoomContract // Generic contract binding to access the raw methods on
}

// RoomContractCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type RoomContractCallerRaw struct {
	Contract *RoomContractCaller // Generic read-only contract binding to access the raw methods on
}

// RoomContractTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type RoomContractTransactorRaw struct {
	Contract *RoomContractTransactor // Generic write-only contract binding to access the raw methods on
}

// NewRoomContract creates a new instance of RoomContract, bound to a specific deployed contract.
func NewRoomContract(address common.Address, backend bind.ContractBackend) (*RoomContract, error) {
	contract, err := bindRoomContract(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &RoomContract{RoomContractCaller: RoomContractCaller{contract: contract}, RoomContractTransactor: RoomContractTransactor{contract: contract}, RoomContractFilterer: RoomContractFilterer{contract: contract}}, nil
}

// NewRoomContractCaller creates a new read-only instance of RoomContract, bound to a specific deployed contract.
func NewRoomContractCaller(address common.Address, caller bind.ContractCaller) (*RoomContractCaller, error) {
	contract, err := bindRoomContract(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &RoomContractCaller{contract: contract}, nil
}

// NewRoomContractTransactor creates a new write-only instance of RoomContract, bound to a specific deployed contract.
func NewRoomContractTransactor(address common.Address, transactor bind.ContractTransactor) (*RoomContractTransactor, error) {
	contract, err := bindRoomContract(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &RoomContractTransactor{contract: contract}, nil
}

// NewRoomContractFilterer creates a new log filterer instance of RoomContract, bound to a specific deployed contract.
func NewRoomContractFilterer(address common.Address, filterer bind.ContractFilterer) (*RoomContractFilterer, error) {
	contract, err := bindRoomContract(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &RoomContractFilterer{contract: contract}, nil
}

// bindRoomContract binds a generic wrapper to an already deployed contract.
func bindRoomContract(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := RoomContractMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_RoomContract *RoomContractRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _RoomContract.Contract.RoomContractCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_RoomContract *RoomContractRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _RoomContract.Contract.RoomContractTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_RoomContract *RoomContractRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _RoomContract.Contract.RoomContractTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_RoomContract *RoomContractCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _RoomContract.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_RoomContract *RoomContractTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _RoomContract.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_RoomContract *RoomContractTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _RoomContract.Contract.contract.Transact(opts, method, params...)
}

// CardsHash is a free data retrieval call binding the contract method 0xdd337a96.
//
// Solidity: function cardsHash(uint256 , uint256 ) view returns(bytes32)
func (_RoomContract *RoomContractCaller) CardsHash(opts *bind.CallOpts, arg0 *big.Int, arg1 *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _RoomContract.contract.Call(opts, &out, "cardsHash", arg0, arg1)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// CardsHash is a free data retrieval call binding the contract method 0xdd337a96.
//
// Solidity: function cardsHash(uint256 , uint256 ) view returns(bytes32)
func (_RoomContract *RoomContractSession) CardsHash(arg0 *big.Int, arg1 *big.Int) ([32]byte, error) {
	return _RoomContract.Contract.CardsHash(&_RoomContract.CallOpts, arg0, arg1)
}

// CardsHash is a free data retrieval call binding the contract method 0xdd337a96.
//
// Solidity: function cardsHash(uint256 , uint256 ) view returns(bytes32)
func (_RoomContract *RoomContractCallerSession) CardsHash(arg0 *big.Int, arg1 *big.Int) ([32]byte, error) {
	return _RoomContract.Contract.CardsHash(&_RoomContract.CallOpts, arg0, arg1)
}

// Creator is a free data retrieval call binding the contract method 0x02d05d3f.
//
// Solidity: function creator() view returns(address)
func (_RoomContract *RoomContractCaller) Creator(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _RoomContract.contract.Call(opts, &out, "creator")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Creator is a free data retrieval call binding the contract method 0x02d05d3f.
//
// Solidity: function creator() view returns(address)
func (_RoomContract *RoomContractSession) Creator() (common.Address, error) {
	return _RoomContract.Contract.Creator(&_RoomContract.CallOpts)
}

// Creator is a free data retrieval call binding the contract method 0x02d05d3f.
//
// Solidity: function creator() view returns(address)
func (_RoomContract *RoomContractCallerSession) Creator() (common.Address, error) {
	return _RoomContract.Contract.Creator(&_RoomContract.CallOpts)
}

// CurrentRound is a free data retrieval call binding the contract method 0x8a19c8bc.
//
// Solidity: function currentRound() view returns(uint256)
func (_RoomContract *RoomContractCaller) CurrentRound(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _RoomContract.contract.Call(opts, &out, "currentRound")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// CurrentRound is a free data retrieval call binding the contract method 0x8a19c8bc.
//
// Solidity: function currentRound() view returns(uint256)
func (_RoomContract *RoomContractSession) CurrentRound() (*big.Int, error) {
	return _RoomContract.Contract.CurrentRound(&_RoomContract.CallOpts)
}

// CurrentRound is a free data retrieval call binding the contract method 0x8a19c8bc.
//
// Solidity: function currentRound() view returns(uint256)
func (_RoomContract *RoomContractCallerSession) CurrentRound() (*big.Int, error) {
	return _RoomContract.Contract.CurrentRound(&_RoomContract.CallOpts)
}

// InitialHP is a free data retrieval call binding the contract method 0x2fab3323.
//
// Solidity: function initialHP() view returns(uint256)
func (_RoomContract *RoomContractCaller) InitialHP(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _RoomContract.contract.Call(opts, &out, "initialHP")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// InitialHP is a free data retrieval call binding the contract method 0x2fab3323.
//
// Solidity: function initialHP() view returns(uint256)
func (_RoomContract *RoomContractSession) InitialHP() (*big.Int, error) {
	return _RoomContract.Contract.InitialHP(&_RoomContract.CallOpts)
}

// InitialHP is a free data retrieval call binding the contract method 0x2fab3323.
//
// Solidity: function initialHP() view returns(uint256)
func (_RoomContract *RoomContractCallerSession) InitialHP() (*big.Int, error) {
	return _RoomContract.Contract.InitialHP(&_RoomContract.CallOpts)
}

// LaunchTime is a free data retrieval call binding the contract method 0x790ca413.
//
// Solidity: function launchTime() view returns(uint256)
func (_RoomContract *RoomContractCaller) LaunchTime(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _RoomContract.contract.Call(opts, &out, "launchTime")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// LaunchTime is a free data retrieval call binding the contract method 0x790ca413.
//
// Solidity: function launchTime() view returns(uint256)
func (_RoomContract *RoomContractSession) LaunchTime() (*big.Int, error) {
	return _RoomContract.Contract.LaunchTime(&_RoomContract.CallOpts)
}

// LaunchTime is a free data retrieval call binding the contract method 0x790ca413.
//
// Solidity: function launchTime() view returns(uint256)
func (_RoomContract *RoomContractCallerSession) LaunchTime() (*big.Int, error) {
	return _RoomContract.Contract.LaunchTime(&_RoomContract.CallOpts)
}

// Player1 is a free data retrieval call binding the contract method 0xd30895e4.
//
// Solidity: function player1() view returns(address)
func (_RoomContract *RoomContractCaller) Player1(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _RoomContract.contract.Call(opts, &out, "player1")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Player1 is a free data retrieval call binding the contract method 0xd30895e4.
//
// Solidity: function player1() view returns(address)
func (_RoomContract *RoomContractSession) Player1() (common.Address, error) {
	return _RoomContract.Contract.Player1(&_RoomContract.CallOpts)
}

// Player1 is a free data retrieval call binding the contract method 0xd30895e4.
//
// Solidity: function player1() view returns(address)
func (_RoomContract *RoomContractCallerSession) Player1() (common.Address, error) {
	return _RoomContract.Contract.Player1(&_RoomContract.CallOpts)
}

// Player2 is a free data retrieval call binding the contract method 0x59a5f12d.
//
// Solidity: function player2() view returns(address)
func (_RoomContract *RoomContractCaller) Player2(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _RoomContract.contract.Call(opts, &out, "player2")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Player2 is a free data retrieval call binding the contract method 0x59a5f12d.
//
// Solidity: function player2() view returns(address)
func (_RoomContract *RoomContractSession) Player2() (common.Address, error) {
	return _RoomContract.Contract.Player2(&_RoomContract.CallOpts)
}

// Player2 is a free data retrieval call binding the contract method 0x59a5f12d.
//
// Solidity: function player2() view returns(address)
func (_RoomContract *RoomContractCallerSession) Player2() (common.Address, error) {
	return _RoomContract.Contract.Player2(&_RoomContract.CallOpts)
}

// RoundTimeout is a free data retrieval call binding the contract method 0x53c298d8.
//
// Solidity: function roundTimeout() view returns(uint256)
func (_RoomContract *RoomContractCaller) RoundTimeout(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _RoomContract.contract.Call(opts, &out, "roundTimeout")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// RoundTimeout is a free data retrieval call binding the contract method 0x53c298d8.
//
// Solidity: function roundTimeout() view returns(uint256)
func (_RoomContract *RoomContractSession) RoundTimeout() (*big.Int, error) {
	return _RoomContract.Contract.RoundTimeout(&_RoomContract.CallOpts)
}

// RoundTimeout is a free data retrieval call binding the contract method 0x53c298d8.
//
// Solidity: function roundTimeout() view returns(uint256)
func (_RoomContract *RoomContractCallerSession) RoundTimeout() (*big.Int, error) {
	return _RoomContract.Contract.RoundTimeout(&_RoomContract.CallOpts)
}

// TotalRound is a free data retrieval call binding the contract method 0xd0421a28.
//
// Solidity: function totalRound() view returns(uint256)
func (_RoomContract *RoomContractCaller) TotalRound(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _RoomContract.contract.Call(opts, &out, "totalRound")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalRound is a free data retrieval call binding the contract method 0xd0421a28.
//
// Solidity: function totalRound() view returns(uint256)
func (_RoomContract *RoomContractSession) TotalRound() (*big.Int, error) {
	return _RoomContract.Contract.TotalRound(&_RoomContract.CallOpts)
}

// TotalRound is a free data retrieval call binding the contract method 0xd0421a28.
//
// Solidity: function totalRound() view returns(uint256)
func (_RoomContract *RoomContractCallerSession) TotalRound() (*big.Int, error) {
	return _RoomContract.Contract.TotalRound(&_RoomContract.CallOpts)
}

// StartANewRound is a paid mutator transaction binding the contract method 0xf1871164.
//
// Solidity: function StartANewRound() returns()
func (_RoomContract *RoomContractTransactor) StartANewRound(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _RoomContract.contract.Transact(opts, "StartANewRound")
}

// StartANewRound is a paid mutator transaction binding the contract method 0xf1871164.
//
// Solidity: function StartANewRound() returns()
func (_RoomContract *RoomContractSession) StartANewRound() (*types.Transaction, error) {
	return _RoomContract.Contract.StartANewRound(&_RoomContract.TransactOpts)
}

// StartANewRound is a paid mutator transaction binding the contract method 0xf1871164.
//
// Solidity: function StartANewRound() returns()
func (_RoomContract *RoomContractTransactorSession) StartANewRound() (*types.Transaction, error) {
	return _RoomContract.Contract.StartANewRound(&_RoomContract.TransactOpts)
}

// SubmitCards is a paid mutator transaction binding the contract method 0x263616a1.
//
// Solidity: function SubmitCards(string _cards, string _salt, uint256 _round) returns()
func (_RoomContract *RoomContractTransactor) SubmitCards(opts *bind.TransactOpts, _cards string, _salt string, _round *big.Int) (*types.Transaction, error) {
	return _RoomContract.contract.Transact(opts, "SubmitCards", _cards, _salt, _round)
}

// SubmitCards is a paid mutator transaction binding the contract method 0x263616a1.
//
// Solidity: function SubmitCards(string _cards, string _salt, uint256 _round) returns()
func (_RoomContract *RoomContractSession) SubmitCards(_cards string, _salt string, _round *big.Int) (*types.Transaction, error) {
	return _RoomContract.Contract.SubmitCards(&_RoomContract.TransactOpts, _cards, _salt, _round)
}

// SubmitCards is a paid mutator transaction binding the contract method 0x263616a1.
//
// Solidity: function SubmitCards(string _cards, string _salt, uint256 _round) returns()
func (_RoomContract *RoomContractTransactorSession) SubmitCards(_cards string, _salt string, _round *big.Int) (*types.Transaction, error) {
	return _RoomContract.Contract.SubmitCards(&_RoomContract.TransactOpts, _cards, _salt, _round)
}

// SubmitCardsHash is a paid mutator transaction binding the contract method 0x2e2ebd23.
//
// Solidity: function SubmitCardsHash(bytes32 _cardsHash, uint256 _round) returns()
func (_RoomContract *RoomContractTransactor) SubmitCardsHash(opts *bind.TransactOpts, _cardsHash [32]byte, _round *big.Int) (*types.Transaction, error) {
	return _RoomContract.contract.Transact(opts, "SubmitCardsHash", _cardsHash, _round)
}

// SubmitCardsHash is a paid mutator transaction binding the contract method 0x2e2ebd23.
//
// Solidity: function SubmitCardsHash(bytes32 _cardsHash, uint256 _round) returns()
func (_RoomContract *RoomContractSession) SubmitCardsHash(_cardsHash [32]byte, _round *big.Int) (*types.Transaction, error) {
	return _RoomContract.Contract.SubmitCardsHash(&_RoomContract.TransactOpts, _cardsHash, _round)
}

// SubmitCardsHash is a paid mutator transaction binding the contract method 0x2e2ebd23.
//
// Solidity: function SubmitCardsHash(bytes32 _cardsHash, uint256 _round) returns()
func (_RoomContract *RoomContractTransactorSession) SubmitCardsHash(_cardsHash [32]byte, _round *big.Int) (*types.Transaction, error) {
	return _RoomContract.Contract.SubmitCardsHash(&_RoomContract.TransactOpts, _cardsHash, _round)
}

// RoomContractStartANewRoundIterator is returned from FilterStartANewRound and is used to iterate over the raw logs and unpacked data for StartANewRound events raised by the RoomContract contract.
type RoomContractStartANewRoundIterator struct {
	Event *RoomContractStartANewRound // Event containing the contract specifics and raw log

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
func (it *RoomContractStartANewRoundIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(RoomContractStartANewRound)
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
		it.Event = new(RoomContractStartANewRound)
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
func (it *RoomContractStartANewRoundIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *RoomContractStartANewRoundIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// RoomContractStartANewRound represents a StartANewRound event raised by the RoomContract contract.
type RoomContractStartANewRound struct {
	Round *big.Int
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterStartANewRound is a free log retrieval operation binding the contract event 0x2725d08e118f50552c873a88cc6bb7d51493108e351080ca153086d4fcfd74a7.
//
// Solidity: event startANewRound(uint256 round)
func (_RoomContract *RoomContractFilterer) FilterStartANewRound(opts *bind.FilterOpts) (*RoomContractStartANewRoundIterator, error) {

	logs, sub, err := _RoomContract.contract.FilterLogs(opts, "startANewRound")
	if err != nil {
		return nil, err
	}
	return &RoomContractStartANewRoundIterator{contract: _RoomContract.contract, event: "startANewRound", logs: logs, sub: sub}, nil
}

// WatchStartANewRound is a free log subscription operation binding the contract event 0x2725d08e118f50552c873a88cc6bb7d51493108e351080ca153086d4fcfd74a7.
//
// Solidity: event startANewRound(uint256 round)
func (_RoomContract *RoomContractFilterer) WatchStartANewRound(opts *bind.WatchOpts, sink chan<- *RoomContractStartANewRound) (event.Subscription, error) {

	logs, sub, err := _RoomContract.contract.WatchLogs(opts, "startANewRound")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(RoomContractStartANewRound)
				if err := _RoomContract.contract.UnpackLog(event, "startANewRound", log); err != nil {
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

// ParseStartANewRound is a log parse operation binding the contract event 0x2725d08e118f50552c873a88cc6bb7d51493108e351080ca153086d4fcfd74a7.
//
// Solidity: event startANewRound(uint256 round)
func (_RoomContract *RoomContractFilterer) ParseStartANewRound(log types.Log) (*RoomContractStartANewRound, error) {
	event := new(RoomContractStartANewRound)
	if err := _RoomContract.contract.UnpackLog(event, "startANewRound", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// RoomContractSubmitCardsIterator is returned from FilterSubmitCards and is used to iterate over the raw logs and unpacked data for SubmitCards events raised by the RoomContract contract.
type RoomContractSubmitCardsIterator struct {
	Event *RoomContractSubmitCards // Event containing the contract specifics and raw log

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
func (it *RoomContractSubmitCardsIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(RoomContractSubmitCards)
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
		it.Event = new(RoomContractSubmitCards)
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
func (it *RoomContractSubmitCardsIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *RoomContractSubmitCardsIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// RoomContractSubmitCards represents a SubmitCards event raised by the RoomContract contract.
type RoomContractSubmitCards struct {
	Player common.Address
	Cards  string
	Salt   string
	Round  *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterSubmitCards is a free log retrieval operation binding the contract event 0xb57778228a8340017474c84eeb5d3b73cf06543b74edf776a78011b593e28779.
//
// Solidity: event submitCards(address player, string cards, string salt, uint256 round)
func (_RoomContract *RoomContractFilterer) FilterSubmitCards(opts *bind.FilterOpts) (*RoomContractSubmitCardsIterator, error) {

	logs, sub, err := _RoomContract.contract.FilterLogs(opts, "submitCards")
	if err != nil {
		return nil, err
	}
	return &RoomContractSubmitCardsIterator{contract: _RoomContract.contract, event: "submitCards", logs: logs, sub: sub}, nil
}

// WatchSubmitCards is a free log subscription operation binding the contract event 0xb57778228a8340017474c84eeb5d3b73cf06543b74edf776a78011b593e28779.
//
// Solidity: event submitCards(address player, string cards, string salt, uint256 round)
func (_RoomContract *RoomContractFilterer) WatchSubmitCards(opts *bind.WatchOpts, sink chan<- *RoomContractSubmitCards) (event.Subscription, error) {

	logs, sub, err := _RoomContract.contract.WatchLogs(opts, "submitCards")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(RoomContractSubmitCards)
				if err := _RoomContract.contract.UnpackLog(event, "submitCards", log); err != nil {
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

// ParseSubmitCards is a log parse operation binding the contract event 0xb57778228a8340017474c84eeb5d3b73cf06543b74edf776a78011b593e28779.
//
// Solidity: event submitCards(address player, string cards, string salt, uint256 round)
func (_RoomContract *RoomContractFilterer) ParseSubmitCards(log types.Log) (*RoomContractSubmitCards, error) {
	event := new(RoomContractSubmitCards)
	if err := _RoomContract.contract.UnpackLog(event, "submitCards", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// RoomContractSubmitCardsHashIterator is returned from FilterSubmitCardsHash and is used to iterate over the raw logs and unpacked data for SubmitCardsHash events raised by the RoomContract contract.
type RoomContractSubmitCardsHashIterator struct {
	Event *RoomContractSubmitCardsHash // Event containing the contract specifics and raw log

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
func (it *RoomContractSubmitCardsHashIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(RoomContractSubmitCardsHash)
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
		it.Event = new(RoomContractSubmitCardsHash)
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
func (it *RoomContractSubmitCardsHashIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *RoomContractSubmitCardsHashIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// RoomContractSubmitCardsHash represents a SubmitCardsHash event raised by the RoomContract contract.
type RoomContractSubmitCardsHash struct {
	Player    common.Address
	CardsHash [32]byte
	Round     *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterSubmitCardsHash is a free log retrieval operation binding the contract event 0x47d16da20fbc7182e199bf669bb55176ef3a0c4fb2288b85cbe394932e024f3d.
//
// Solidity: event submitCardsHash(address player, bytes32 cardsHash, uint256 round)
func (_RoomContract *RoomContractFilterer) FilterSubmitCardsHash(opts *bind.FilterOpts) (*RoomContractSubmitCardsHashIterator, error) {

	logs, sub, err := _RoomContract.contract.FilterLogs(opts, "submitCardsHash")
	if err != nil {
		return nil, err
	}
	return &RoomContractSubmitCardsHashIterator{contract: _RoomContract.contract, event: "submitCardsHash", logs: logs, sub: sub}, nil
}

// WatchSubmitCardsHash is a free log subscription operation binding the contract event 0x47d16da20fbc7182e199bf669bb55176ef3a0c4fb2288b85cbe394932e024f3d.
//
// Solidity: event submitCardsHash(address player, bytes32 cardsHash, uint256 round)
func (_RoomContract *RoomContractFilterer) WatchSubmitCardsHash(opts *bind.WatchOpts, sink chan<- *RoomContractSubmitCardsHash) (event.Subscription, error) {

	logs, sub, err := _RoomContract.contract.WatchLogs(opts, "submitCardsHash")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(RoomContractSubmitCardsHash)
				if err := _RoomContract.contract.UnpackLog(event, "submitCardsHash", log); err != nil {
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

// ParseSubmitCardsHash is a log parse operation binding the contract event 0x47d16da20fbc7182e199bf669bb55176ef3a0c4fb2288b85cbe394932e024f3d.
//
// Solidity: event submitCardsHash(address player, bytes32 cardsHash, uint256 round)
func (_RoomContract *RoomContractFilterer) ParseSubmitCardsHash(log types.Log) (*RoomContractSubmitCardsHash, error) {
	event := new(RoomContractSubmitCardsHash)
	if err := _RoomContract.contract.UnpackLog(event, "submitCardsHash", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
