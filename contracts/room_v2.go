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

// RoomV2ContractMetaData contains all meta data concerning the RoomV2Contract contract.
var RoomV2ContractMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_player1Id\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_player2Id\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"_player1_tmp\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"_player2_tmp\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_totalRound\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_totalCardIndex\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_initialHP\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_gameId\",\"type\":\"uint256\"}],\"name\":\"RoomCreated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"cardIndex\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"round\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"gameId\",\"type\":\"uint256\"}],\"name\":\"startANewTurn\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"playerId\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"player\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"card\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"salt\",\"type\":\"string\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"round\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"cardIndex\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"gameId\",\"type\":\"uint256\"}],\"name\":\"submitCard\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"playerId\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"player\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"cardHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"round\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"cardIndex\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"gameId\",\"type\":\"uint256\"}],\"name\":\"submitCardHash\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_player1Id\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_player2Id\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"_player1_tmp\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_player2_tmp\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_roundTimeout\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_totalRound\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_totalCardIndex\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_initialHP\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_gameId\",\"type\":\"uint256\"}],\"name\":\"CreateRoom\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_gameId\",\"type\":\"uint256\"}],\"name\":\"StartANewTurn\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_newManager\",\"type\":\"address\"}],\"name\":\"addManager\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256[]\",\"name\":\"_gameIds\",\"type\":\"uint256[]\"},{\"internalType\":\"uint256[]\",\"name\":\"_cards\",\"type\":\"uint256[]\"},{\"internalType\":\"string[]\",\"name\":\"_salts\",\"type\":\"string[]\"},{\"internalType\":\"uint256[]\",\"name\":\"_cardIndexes\",\"type\":\"uint256[]\"},{\"internalType\":\"uint256[]\",\"name\":\"_rounds\",\"type\":\"uint256[]\"},{\"internalType\":\"bytes[]\",\"name\":\"_signatures\",\"type\":\"bytes[]\"}],\"name\":\"batchSubmitCards\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256[]\",\"name\":\"_gameIds\",\"type\":\"uint256[]\"},{\"internalType\":\"bytes32[]\",\"name\":\"_cardsHashes\",\"type\":\"bytes32[]\"},{\"internalType\":\"uint256[]\",\"name\":\"_cardIndexes\",\"type\":\"uint256[]\"},{\"internalType\":\"uint256[]\",\"name\":\"_rounds\",\"type\":\"uint256[]\"},{\"internalType\":\"bytes[]\",\"name\":\"_signatures\",\"type\":\"bytes[]\"}],\"name\":\"batchSubmitCardsHash\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"gameId\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"gameIdIndex\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"managerIndex\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"roomData\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"currentRound\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"currentCardIndex\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"totalRound\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"totalCardIndex\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"creator\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"player1Id\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"player2Id\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"player1Temp\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"player2Temp\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"launchTime\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"roundTimeout\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"initialHP\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// RoomV2ContractABI is the input ABI used to generate the binding from.
// Deprecated: Use RoomV2ContractMetaData.ABI instead.
var RoomV2ContractABI = RoomV2ContractMetaData.ABI

// RoomV2Contract is an auto generated Go binding around an Ethereum contract.
type RoomV2Contract struct {
	RoomV2ContractCaller     // Read-only binding to the contract
	RoomV2ContractTransactor // Write-only binding to the contract
	RoomV2ContractFilterer   // Log filterer for contract events
}

// RoomV2ContractCaller is an auto generated read-only Go binding around an Ethereum contract.
type RoomV2ContractCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RoomV2ContractTransactor is an auto generated write-only Go binding around an Ethereum contract.
type RoomV2ContractTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RoomV2ContractFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type RoomV2ContractFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RoomV2ContractSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type RoomV2ContractSession struct {
	Contract     *RoomV2Contract   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// RoomV2ContractCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type RoomV2ContractCallerSession struct {
	Contract *RoomV2ContractCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// RoomV2ContractTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type RoomV2ContractTransactorSession struct {
	Contract     *RoomV2ContractTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// RoomV2ContractRaw is an auto generated low-level Go binding around an Ethereum contract.
type RoomV2ContractRaw struct {
	Contract *RoomV2Contract // Generic contract binding to access the raw methods on
}

// RoomV2ContractCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type RoomV2ContractCallerRaw struct {
	Contract *RoomV2ContractCaller // Generic read-only contract binding to access the raw methods on
}

// RoomV2ContractTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type RoomV2ContractTransactorRaw struct {
	Contract *RoomV2ContractTransactor // Generic write-only contract binding to access the raw methods on
}

// NewRoomV2Contract creates a new instance of RoomV2Contract, bound to a specific deployed contract.
func NewRoomV2Contract(address common.Address, backend bind.ContractBackend) (*RoomV2Contract, error) {
	contract, err := bindRoomV2Contract(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &RoomV2Contract{RoomV2ContractCaller: RoomV2ContractCaller{contract: contract}, RoomV2ContractTransactor: RoomV2ContractTransactor{contract: contract}, RoomV2ContractFilterer: RoomV2ContractFilterer{contract: contract}}, nil
}

// NewRoomV2ContractCaller creates a new read-only instance of RoomV2Contract, bound to a specific deployed contract.
func NewRoomV2ContractCaller(address common.Address, caller bind.ContractCaller) (*RoomV2ContractCaller, error) {
	contract, err := bindRoomV2Contract(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &RoomV2ContractCaller{contract: contract}, nil
}

// NewRoomV2ContractTransactor creates a new write-only instance of RoomV2Contract, bound to a specific deployed contract.
func NewRoomV2ContractTransactor(address common.Address, transactor bind.ContractTransactor) (*RoomV2ContractTransactor, error) {
	contract, err := bindRoomV2Contract(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &RoomV2ContractTransactor{contract: contract}, nil
}

// NewRoomV2ContractFilterer creates a new log filterer instance of RoomV2Contract, bound to a specific deployed contract.
func NewRoomV2ContractFilterer(address common.Address, filterer bind.ContractFilterer) (*RoomV2ContractFilterer, error) {
	contract, err := bindRoomV2Contract(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &RoomV2ContractFilterer{contract: contract}, nil
}

// bindRoomV2Contract binds a generic wrapper to an already deployed contract.
func bindRoomV2Contract(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := RoomV2ContractMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_RoomV2Contract *RoomV2ContractRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _RoomV2Contract.Contract.RoomV2ContractCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_RoomV2Contract *RoomV2ContractRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _RoomV2Contract.Contract.RoomV2ContractTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_RoomV2Contract *RoomV2ContractRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _RoomV2Contract.Contract.RoomV2ContractTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_RoomV2Contract *RoomV2ContractCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _RoomV2Contract.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_RoomV2Contract *RoomV2ContractTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _RoomV2Contract.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_RoomV2Contract *RoomV2ContractTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _RoomV2Contract.Contract.contract.Transact(opts, method, params...)
}

// GameId is a free data retrieval call binding the contract method 0x168aa23a.
//
// Solidity: function gameId(uint256 ) view returns(uint256)
func (_RoomV2Contract *RoomV2ContractCaller) GameId(opts *bind.CallOpts, arg0 *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _RoomV2Contract.contract.Call(opts, &out, "gameId", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GameId is a free data retrieval call binding the contract method 0x168aa23a.
//
// Solidity: function gameId(uint256 ) view returns(uint256)
func (_RoomV2Contract *RoomV2ContractSession) GameId(arg0 *big.Int) (*big.Int, error) {
	return _RoomV2Contract.Contract.GameId(&_RoomV2Contract.CallOpts, arg0)
}

// GameId is a free data retrieval call binding the contract method 0x168aa23a.
//
// Solidity: function gameId(uint256 ) view returns(uint256)
func (_RoomV2Contract *RoomV2ContractCallerSession) GameId(arg0 *big.Int) (*big.Int, error) {
	return _RoomV2Contract.Contract.GameId(&_RoomV2Contract.CallOpts, arg0)
}

// GameIdIndex is a free data retrieval call binding the contract method 0xa3157b47.
//
// Solidity: function gameIdIndex(uint256 ) view returns(uint256)
func (_RoomV2Contract *RoomV2ContractCaller) GameIdIndex(opts *bind.CallOpts, arg0 *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _RoomV2Contract.contract.Call(opts, &out, "gameIdIndex", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GameIdIndex is a free data retrieval call binding the contract method 0xa3157b47.
//
// Solidity: function gameIdIndex(uint256 ) view returns(uint256)
func (_RoomV2Contract *RoomV2ContractSession) GameIdIndex(arg0 *big.Int) (*big.Int, error) {
	return _RoomV2Contract.Contract.GameIdIndex(&_RoomV2Contract.CallOpts, arg0)
}

// GameIdIndex is a free data retrieval call binding the contract method 0xa3157b47.
//
// Solidity: function gameIdIndex(uint256 ) view returns(uint256)
func (_RoomV2Contract *RoomV2ContractCallerSession) GameIdIndex(arg0 *big.Int) (*big.Int, error) {
	return _RoomV2Contract.Contract.GameIdIndex(&_RoomV2Contract.CallOpts, arg0)
}

// ManagerIndex is a free data retrieval call binding the contract method 0x573d44fa.
//
// Solidity: function managerIndex(address ) view returns(uint256)
func (_RoomV2Contract *RoomV2ContractCaller) ManagerIndex(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _RoomV2Contract.contract.Call(opts, &out, "managerIndex", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ManagerIndex is a free data retrieval call binding the contract method 0x573d44fa.
//
// Solidity: function managerIndex(address ) view returns(uint256)
func (_RoomV2Contract *RoomV2ContractSession) ManagerIndex(arg0 common.Address) (*big.Int, error) {
	return _RoomV2Contract.Contract.ManagerIndex(&_RoomV2Contract.CallOpts, arg0)
}

// ManagerIndex is a free data retrieval call binding the contract method 0x573d44fa.
//
// Solidity: function managerIndex(address ) view returns(uint256)
func (_RoomV2Contract *RoomV2ContractCallerSession) ManagerIndex(arg0 common.Address) (*big.Int, error) {
	return _RoomV2Contract.Contract.ManagerIndex(&_RoomV2Contract.CallOpts, arg0)
}

// RoomData is a free data retrieval call binding the contract method 0xe193dff8.
//
// Solidity: function roomData(uint256 ) view returns(uint256 currentRound, uint256 currentCardIndex, uint256 totalRound, uint256 totalCardIndex, address creator, uint256 player1Id, uint256 player2Id, address player1Temp, address player2Temp, uint256 launchTime, uint256 roundTimeout, uint256 initialHP)
func (_RoomV2Contract *RoomV2ContractCaller) RoomData(opts *bind.CallOpts, arg0 *big.Int) (struct {
	CurrentRound     *big.Int
	CurrentCardIndex *big.Int
	TotalRound       *big.Int
	TotalCardIndex   *big.Int
	Creator          common.Address
	Player1Id        *big.Int
	Player2Id        *big.Int
	Player1Temp      common.Address
	Player2Temp      common.Address
	LaunchTime       *big.Int
	RoundTimeout     *big.Int
	InitialHP        *big.Int
}, error) {
	var out []interface{}
	err := _RoomV2Contract.contract.Call(opts, &out, "roomData", arg0)

	outstruct := new(struct {
		CurrentRound     *big.Int
		CurrentCardIndex *big.Int
		TotalRound       *big.Int
		TotalCardIndex   *big.Int
		Creator          common.Address
		Player1Id        *big.Int
		Player2Id        *big.Int
		Player1Temp      common.Address
		Player2Temp      common.Address
		LaunchTime       *big.Int
		RoundTimeout     *big.Int
		InitialHP        *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.CurrentRound = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.CurrentCardIndex = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.TotalRound = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.TotalCardIndex = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)
	outstruct.Creator = *abi.ConvertType(out[4], new(common.Address)).(*common.Address)
	outstruct.Player1Id = *abi.ConvertType(out[5], new(*big.Int)).(**big.Int)
	outstruct.Player2Id = *abi.ConvertType(out[6], new(*big.Int)).(**big.Int)
	outstruct.Player1Temp = *abi.ConvertType(out[7], new(common.Address)).(*common.Address)
	outstruct.Player2Temp = *abi.ConvertType(out[8], new(common.Address)).(*common.Address)
	outstruct.LaunchTime = *abi.ConvertType(out[9], new(*big.Int)).(**big.Int)
	outstruct.RoundTimeout = *abi.ConvertType(out[10], new(*big.Int)).(**big.Int)
	outstruct.InitialHP = *abi.ConvertType(out[11], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// RoomData is a free data retrieval call binding the contract method 0xe193dff8.
//
// Solidity: function roomData(uint256 ) view returns(uint256 currentRound, uint256 currentCardIndex, uint256 totalRound, uint256 totalCardIndex, address creator, uint256 player1Id, uint256 player2Id, address player1Temp, address player2Temp, uint256 launchTime, uint256 roundTimeout, uint256 initialHP)
func (_RoomV2Contract *RoomV2ContractSession) RoomData(arg0 *big.Int) (struct {
	CurrentRound     *big.Int
	CurrentCardIndex *big.Int
	TotalRound       *big.Int
	TotalCardIndex   *big.Int
	Creator          common.Address
	Player1Id        *big.Int
	Player2Id        *big.Int
	Player1Temp      common.Address
	Player2Temp      common.Address
	LaunchTime       *big.Int
	RoundTimeout     *big.Int
	InitialHP        *big.Int
}, error) {
	return _RoomV2Contract.Contract.RoomData(&_RoomV2Contract.CallOpts, arg0)
}

// RoomData is a free data retrieval call binding the contract method 0xe193dff8.
//
// Solidity: function roomData(uint256 ) view returns(uint256 currentRound, uint256 currentCardIndex, uint256 totalRound, uint256 totalCardIndex, address creator, uint256 player1Id, uint256 player2Id, address player1Temp, address player2Temp, uint256 launchTime, uint256 roundTimeout, uint256 initialHP)
func (_RoomV2Contract *RoomV2ContractCallerSession) RoomData(arg0 *big.Int) (struct {
	CurrentRound     *big.Int
	CurrentCardIndex *big.Int
	TotalRound       *big.Int
	TotalCardIndex   *big.Int
	Creator          common.Address
	Player1Id        *big.Int
	Player2Id        *big.Int
	Player1Temp      common.Address
	Player2Temp      common.Address
	LaunchTime       *big.Int
	RoundTimeout     *big.Int
	InitialHP        *big.Int
}, error) {
	return _RoomV2Contract.Contract.RoomData(&_RoomV2Contract.CallOpts, arg0)
}

// CreateRoom is a paid mutator transaction binding the contract method 0x7a9df21e.
//
// Solidity: function CreateRoom(uint256 _player1Id, uint256 _player2Id, address _player1_tmp, address _player2_tmp, uint256 _roundTimeout, uint256 _totalRound, uint256 _totalCardIndex, uint256 _initialHP, uint256 _gameId) returns()
func (_RoomV2Contract *RoomV2ContractTransactor) CreateRoom(opts *bind.TransactOpts, _player1Id *big.Int, _player2Id *big.Int, _player1_tmp common.Address, _player2_tmp common.Address, _roundTimeout *big.Int, _totalRound *big.Int, _totalCardIndex *big.Int, _initialHP *big.Int, _gameId *big.Int) (*types.Transaction, error) {
	return _RoomV2Contract.contract.Transact(opts, "CreateRoom", _player1Id, _player2Id, _player1_tmp, _player2_tmp, _roundTimeout, _totalRound, _totalCardIndex, _initialHP, _gameId)
}

// CreateRoom is a paid mutator transaction binding the contract method 0x7a9df21e.
//
// Solidity: function CreateRoom(uint256 _player1Id, uint256 _player2Id, address _player1_tmp, address _player2_tmp, uint256 _roundTimeout, uint256 _totalRound, uint256 _totalCardIndex, uint256 _initialHP, uint256 _gameId) returns()
func (_RoomV2Contract *RoomV2ContractSession) CreateRoom(_player1Id *big.Int, _player2Id *big.Int, _player1_tmp common.Address, _player2_tmp common.Address, _roundTimeout *big.Int, _totalRound *big.Int, _totalCardIndex *big.Int, _initialHP *big.Int, _gameId *big.Int) (*types.Transaction, error) {
	return _RoomV2Contract.Contract.CreateRoom(&_RoomV2Contract.TransactOpts, _player1Id, _player2Id, _player1_tmp, _player2_tmp, _roundTimeout, _totalRound, _totalCardIndex, _initialHP, _gameId)
}

// CreateRoom is a paid mutator transaction binding the contract method 0x7a9df21e.
//
// Solidity: function CreateRoom(uint256 _player1Id, uint256 _player2Id, address _player1_tmp, address _player2_tmp, uint256 _roundTimeout, uint256 _totalRound, uint256 _totalCardIndex, uint256 _initialHP, uint256 _gameId) returns()
func (_RoomV2Contract *RoomV2ContractTransactorSession) CreateRoom(_player1Id *big.Int, _player2Id *big.Int, _player1_tmp common.Address, _player2_tmp common.Address, _roundTimeout *big.Int, _totalRound *big.Int, _totalCardIndex *big.Int, _initialHP *big.Int, _gameId *big.Int) (*types.Transaction, error) {
	return _RoomV2Contract.Contract.CreateRoom(&_RoomV2Contract.TransactOpts, _player1Id, _player2Id, _player1_tmp, _player2_tmp, _roundTimeout, _totalRound, _totalCardIndex, _initialHP, _gameId)
}

// StartANewTurn is a paid mutator transaction binding the contract method 0x934f92db.
//
// Solidity: function StartANewTurn(uint256 _gameId) returns()
func (_RoomV2Contract *RoomV2ContractTransactor) StartANewTurn(opts *bind.TransactOpts, _gameId *big.Int) (*types.Transaction, error) {
	return _RoomV2Contract.contract.Transact(opts, "StartANewTurn", _gameId)
}

// StartANewTurn is a paid mutator transaction binding the contract method 0x934f92db.
//
// Solidity: function StartANewTurn(uint256 _gameId) returns()
func (_RoomV2Contract *RoomV2ContractSession) StartANewTurn(_gameId *big.Int) (*types.Transaction, error) {
	return _RoomV2Contract.Contract.StartANewTurn(&_RoomV2Contract.TransactOpts, _gameId)
}

// StartANewTurn is a paid mutator transaction binding the contract method 0x934f92db.
//
// Solidity: function StartANewTurn(uint256 _gameId) returns()
func (_RoomV2Contract *RoomV2ContractTransactorSession) StartANewTurn(_gameId *big.Int) (*types.Transaction, error) {
	return _RoomV2Contract.Contract.StartANewTurn(&_RoomV2Contract.TransactOpts, _gameId)
}

// AddManager is a paid mutator transaction binding the contract method 0x2d06177a.
//
// Solidity: function addManager(address _newManager) returns()
func (_RoomV2Contract *RoomV2ContractTransactor) AddManager(opts *bind.TransactOpts, _newManager common.Address) (*types.Transaction, error) {
	return _RoomV2Contract.contract.Transact(opts, "addManager", _newManager)
}

// AddManager is a paid mutator transaction binding the contract method 0x2d06177a.
//
// Solidity: function addManager(address _newManager) returns()
func (_RoomV2Contract *RoomV2ContractSession) AddManager(_newManager common.Address) (*types.Transaction, error) {
	return _RoomV2Contract.Contract.AddManager(&_RoomV2Contract.TransactOpts, _newManager)
}

// AddManager is a paid mutator transaction binding the contract method 0x2d06177a.
//
// Solidity: function addManager(address _newManager) returns()
func (_RoomV2Contract *RoomV2ContractTransactorSession) AddManager(_newManager common.Address) (*types.Transaction, error) {
	return _RoomV2Contract.Contract.AddManager(&_RoomV2Contract.TransactOpts, _newManager)
}

// BatchSubmitCards is a paid mutator transaction binding the contract method 0x824eb3c2.
//
// Solidity: function batchSubmitCards(uint256[] _gameIds, uint256[] _cards, string[] _salts, uint256[] _cardIndexes, uint256[] _rounds, bytes[] _signatures) returns()
func (_RoomV2Contract *RoomV2ContractTransactor) BatchSubmitCards(opts *bind.TransactOpts, _gameIds []*big.Int, _cards []*big.Int, _salts []string, _cardIndexes []*big.Int, _rounds []*big.Int, _signatures [][]byte) (*types.Transaction, error) {
	return _RoomV2Contract.contract.Transact(opts, "batchSubmitCards", _gameIds, _cards, _salts, _cardIndexes, _rounds, _signatures)
}

// BatchSubmitCards is a paid mutator transaction binding the contract method 0x824eb3c2.
//
// Solidity: function batchSubmitCards(uint256[] _gameIds, uint256[] _cards, string[] _salts, uint256[] _cardIndexes, uint256[] _rounds, bytes[] _signatures) returns()
func (_RoomV2Contract *RoomV2ContractSession) BatchSubmitCards(_gameIds []*big.Int, _cards []*big.Int, _salts []string, _cardIndexes []*big.Int, _rounds []*big.Int, _signatures [][]byte) (*types.Transaction, error) {
	return _RoomV2Contract.Contract.BatchSubmitCards(&_RoomV2Contract.TransactOpts, _gameIds, _cards, _salts, _cardIndexes, _rounds, _signatures)
}

// BatchSubmitCards is a paid mutator transaction binding the contract method 0x824eb3c2.
//
// Solidity: function batchSubmitCards(uint256[] _gameIds, uint256[] _cards, string[] _salts, uint256[] _cardIndexes, uint256[] _rounds, bytes[] _signatures) returns()
func (_RoomV2Contract *RoomV2ContractTransactorSession) BatchSubmitCards(_gameIds []*big.Int, _cards []*big.Int, _salts []string, _cardIndexes []*big.Int, _rounds []*big.Int, _signatures [][]byte) (*types.Transaction, error) {
	return _RoomV2Contract.Contract.BatchSubmitCards(&_RoomV2Contract.TransactOpts, _gameIds, _cards, _salts, _cardIndexes, _rounds, _signatures)
}

// BatchSubmitCardsHash is a paid mutator transaction binding the contract method 0xad05ae71.
//
// Solidity: function batchSubmitCardsHash(uint256[] _gameIds, bytes32[] _cardsHashes, uint256[] _cardIndexes, uint256[] _rounds, bytes[] _signatures) returns()
func (_RoomV2Contract *RoomV2ContractTransactor) BatchSubmitCardsHash(opts *bind.TransactOpts, _gameIds []*big.Int, _cardsHashes [][32]byte, _cardIndexes []*big.Int, _rounds []*big.Int, _signatures [][]byte) (*types.Transaction, error) {
	return _RoomV2Contract.contract.Transact(opts, "batchSubmitCardsHash", _gameIds, _cardsHashes, _cardIndexes, _rounds, _signatures)
}

// BatchSubmitCardsHash is a paid mutator transaction binding the contract method 0xad05ae71.
//
// Solidity: function batchSubmitCardsHash(uint256[] _gameIds, bytes32[] _cardsHashes, uint256[] _cardIndexes, uint256[] _rounds, bytes[] _signatures) returns()
func (_RoomV2Contract *RoomV2ContractSession) BatchSubmitCardsHash(_gameIds []*big.Int, _cardsHashes [][32]byte, _cardIndexes []*big.Int, _rounds []*big.Int, _signatures [][]byte) (*types.Transaction, error) {
	return _RoomV2Contract.Contract.BatchSubmitCardsHash(&_RoomV2Contract.TransactOpts, _gameIds, _cardsHashes, _cardIndexes, _rounds, _signatures)
}

// BatchSubmitCardsHash is a paid mutator transaction binding the contract method 0xad05ae71.
//
// Solidity: function batchSubmitCardsHash(uint256[] _gameIds, bytes32[] _cardsHashes, uint256[] _cardIndexes, uint256[] _rounds, bytes[] _signatures) returns()
func (_RoomV2Contract *RoomV2ContractTransactorSession) BatchSubmitCardsHash(_gameIds []*big.Int, _cardsHashes [][32]byte, _cardIndexes []*big.Int, _rounds []*big.Int, _signatures [][]byte) (*types.Transaction, error) {
	return _RoomV2Contract.Contract.BatchSubmitCardsHash(&_RoomV2Contract.TransactOpts, _gameIds, _cardsHashes, _cardIndexes, _rounds, _signatures)
}

// RoomV2ContractRoomCreatedIterator is returned from FilterRoomCreated and is used to iterate over the raw logs and unpacked data for RoomCreated events raised by the RoomV2Contract contract.
type RoomV2ContractRoomCreatedIterator struct {
	Event *RoomV2ContractRoomCreated // Event containing the contract specifics and raw log

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
func (it *RoomV2ContractRoomCreatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(RoomV2ContractRoomCreated)
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
		it.Event = new(RoomV2ContractRoomCreated)
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
func (it *RoomV2ContractRoomCreatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *RoomV2ContractRoomCreatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// RoomV2ContractRoomCreated represents a RoomCreated event raised by the RoomV2Contract contract.
type RoomV2ContractRoomCreated struct {
	Player1Id      *big.Int
	Player2Id      *big.Int
	Player1Tmp     common.Address
	Player2Tmp     common.Address
	TotalRound     *big.Int
	TotalCardIndex *big.Int
	InitialHP      *big.Int
	GameId         *big.Int
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterRoomCreated is a free log retrieval operation binding the contract event 0x103b46b7007b0717433baa83443c5aa46c84bb502276b1508d3ba3ecef4aac6a.
//
// Solidity: event RoomCreated(uint256 _player1Id, uint256 _player2Id, address _player1_tmp, address _player2_tmp, uint256 _totalRound, uint256 _totalCardIndex, uint256 _initialHP, uint256 _gameId)
func (_RoomV2Contract *RoomV2ContractFilterer) FilterRoomCreated(opts *bind.FilterOpts) (*RoomV2ContractRoomCreatedIterator, error) {

	logs, sub, err := _RoomV2Contract.contract.FilterLogs(opts, "RoomCreated")
	if err != nil {
		return nil, err
	}
	return &RoomV2ContractRoomCreatedIterator{contract: _RoomV2Contract.contract, event: "RoomCreated", logs: logs, sub: sub}, nil
}

// WatchRoomCreated is a free log subscription operation binding the contract event 0x103b46b7007b0717433baa83443c5aa46c84bb502276b1508d3ba3ecef4aac6a.
//
// Solidity: event RoomCreated(uint256 _player1Id, uint256 _player2Id, address _player1_tmp, address _player2_tmp, uint256 _totalRound, uint256 _totalCardIndex, uint256 _initialHP, uint256 _gameId)
func (_RoomV2Contract *RoomV2ContractFilterer) WatchRoomCreated(opts *bind.WatchOpts, sink chan<- *RoomV2ContractRoomCreated) (event.Subscription, error) {

	logs, sub, err := _RoomV2Contract.contract.WatchLogs(opts, "RoomCreated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(RoomV2ContractRoomCreated)
				if err := _RoomV2Contract.contract.UnpackLog(event, "RoomCreated", log); err != nil {
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

// ParseRoomCreated is a log parse operation binding the contract event 0x103b46b7007b0717433baa83443c5aa46c84bb502276b1508d3ba3ecef4aac6a.
//
// Solidity: event RoomCreated(uint256 _player1Id, uint256 _player2Id, address _player1_tmp, address _player2_tmp, uint256 _totalRound, uint256 _totalCardIndex, uint256 _initialHP, uint256 _gameId)
func (_RoomV2Contract *RoomV2ContractFilterer) ParseRoomCreated(log types.Log) (*RoomV2ContractRoomCreated, error) {
	event := new(RoomV2ContractRoomCreated)
	if err := _RoomV2Contract.contract.UnpackLog(event, "RoomCreated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// RoomV2ContractStartANewTurnIterator is returned from FilterStartANewTurn and is used to iterate over the raw logs and unpacked data for StartANewTurn events raised by the RoomV2Contract contract.
type RoomV2ContractStartANewTurnIterator struct {
	Event *RoomV2ContractStartANewTurn // Event containing the contract specifics and raw log

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
func (it *RoomV2ContractStartANewTurnIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(RoomV2ContractStartANewTurn)
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
		it.Event = new(RoomV2ContractStartANewTurn)
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
func (it *RoomV2ContractStartANewTurnIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *RoomV2ContractStartANewTurnIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// RoomV2ContractStartANewTurn represents a StartANewTurn event raised by the RoomV2Contract contract.
type RoomV2ContractStartANewTurn struct {
	CardIndex *big.Int
	Round     *big.Int
	GameId    *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterStartANewTurn is a free log retrieval operation binding the contract event 0x882606b8d795d9c8d0f6dc344b6b949ddc29ced7c5512e714aff5ef6ee5e9cf8.
//
// Solidity: event startANewTurn(uint256 cardIndex, uint256 round, uint256 gameId)
func (_RoomV2Contract *RoomV2ContractFilterer) FilterStartANewTurn(opts *bind.FilterOpts) (*RoomV2ContractStartANewTurnIterator, error) {

	logs, sub, err := _RoomV2Contract.contract.FilterLogs(opts, "startANewTurn")
	if err != nil {
		return nil, err
	}
	return &RoomV2ContractStartANewTurnIterator{contract: _RoomV2Contract.contract, event: "startANewTurn", logs: logs, sub: sub}, nil
}

// WatchStartANewTurn is a free log subscription operation binding the contract event 0x882606b8d795d9c8d0f6dc344b6b949ddc29ced7c5512e714aff5ef6ee5e9cf8.
//
// Solidity: event startANewTurn(uint256 cardIndex, uint256 round, uint256 gameId)
func (_RoomV2Contract *RoomV2ContractFilterer) WatchStartANewTurn(opts *bind.WatchOpts, sink chan<- *RoomV2ContractStartANewTurn) (event.Subscription, error) {

	logs, sub, err := _RoomV2Contract.contract.WatchLogs(opts, "startANewTurn")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(RoomV2ContractStartANewTurn)
				if err := _RoomV2Contract.contract.UnpackLog(event, "startANewTurn", log); err != nil {
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

// ParseStartANewTurn is a log parse operation binding the contract event 0x882606b8d795d9c8d0f6dc344b6b949ddc29ced7c5512e714aff5ef6ee5e9cf8.
//
// Solidity: event startANewTurn(uint256 cardIndex, uint256 round, uint256 gameId)
func (_RoomV2Contract *RoomV2ContractFilterer) ParseStartANewTurn(log types.Log) (*RoomV2ContractStartANewTurn, error) {
	event := new(RoomV2ContractStartANewTurn)
	if err := _RoomV2Contract.contract.UnpackLog(event, "startANewTurn", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// RoomV2ContractSubmitCardIterator is returned from FilterSubmitCard and is used to iterate over the raw logs and unpacked data for SubmitCard events raised by the RoomV2Contract contract.
type RoomV2ContractSubmitCardIterator struct {
	Event *RoomV2ContractSubmitCard // Event containing the contract specifics and raw log

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
func (it *RoomV2ContractSubmitCardIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(RoomV2ContractSubmitCard)
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
		it.Event = new(RoomV2ContractSubmitCard)
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
func (it *RoomV2ContractSubmitCardIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *RoomV2ContractSubmitCardIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// RoomV2ContractSubmitCard represents a SubmitCard event raised by the RoomV2Contract contract.
type RoomV2ContractSubmitCard struct {
	PlayerId  *big.Int
	Player    common.Address
	Card      *big.Int
	Salt      string
	Round     *big.Int
	CardIndex *big.Int
	GameId    *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterSubmitCard is a free log retrieval operation binding the contract event 0x068155fe5488ed7b2ae50de0f63d43ae436bea1b3a455d6c212871cbf3ad7f09.
//
// Solidity: event submitCard(uint256 playerId, address player, uint256 card, string salt, uint256 round, uint256 cardIndex, uint256 gameId)
func (_RoomV2Contract *RoomV2ContractFilterer) FilterSubmitCard(opts *bind.FilterOpts) (*RoomV2ContractSubmitCardIterator, error) {

	logs, sub, err := _RoomV2Contract.contract.FilterLogs(opts, "submitCard")
	if err != nil {
		return nil, err
	}
	return &RoomV2ContractSubmitCardIterator{contract: _RoomV2Contract.contract, event: "submitCard", logs: logs, sub: sub}, nil
}

// WatchSubmitCard is a free log subscription operation binding the contract event 0x068155fe5488ed7b2ae50de0f63d43ae436bea1b3a455d6c212871cbf3ad7f09.
//
// Solidity: event submitCard(uint256 playerId, address player, uint256 card, string salt, uint256 round, uint256 cardIndex, uint256 gameId)
func (_RoomV2Contract *RoomV2ContractFilterer) WatchSubmitCard(opts *bind.WatchOpts, sink chan<- *RoomV2ContractSubmitCard) (event.Subscription, error) {

	logs, sub, err := _RoomV2Contract.contract.WatchLogs(opts, "submitCard")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(RoomV2ContractSubmitCard)
				if err := _RoomV2Contract.contract.UnpackLog(event, "submitCard", log); err != nil {
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

// ParseSubmitCard is a log parse operation binding the contract event 0x068155fe5488ed7b2ae50de0f63d43ae436bea1b3a455d6c212871cbf3ad7f09.
//
// Solidity: event submitCard(uint256 playerId, address player, uint256 card, string salt, uint256 round, uint256 cardIndex, uint256 gameId)
func (_RoomV2Contract *RoomV2ContractFilterer) ParseSubmitCard(log types.Log) (*RoomV2ContractSubmitCard, error) {
	event := new(RoomV2ContractSubmitCard)
	if err := _RoomV2Contract.contract.UnpackLog(event, "submitCard", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// RoomV2ContractSubmitCardHashIterator is returned from FilterSubmitCardHash and is used to iterate over the raw logs and unpacked data for SubmitCardHash events raised by the RoomV2Contract contract.
type RoomV2ContractSubmitCardHashIterator struct {
	Event *RoomV2ContractSubmitCardHash // Event containing the contract specifics and raw log

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
func (it *RoomV2ContractSubmitCardHashIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(RoomV2ContractSubmitCardHash)
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
		it.Event = new(RoomV2ContractSubmitCardHash)
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
func (it *RoomV2ContractSubmitCardHashIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *RoomV2ContractSubmitCardHashIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// RoomV2ContractSubmitCardHash represents a SubmitCardHash event raised by the RoomV2Contract contract.
type RoomV2ContractSubmitCardHash struct {
	PlayerId  *big.Int
	Player    common.Address
	CardHash  [32]byte
	Round     *big.Int
	CardIndex *big.Int
	GameId    *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterSubmitCardHash is a free log retrieval operation binding the contract event 0xf97708df45e82dc78722b3d12db0f7b5ed10df426b0f8069be9cc7e6aa08baf4.
//
// Solidity: event submitCardHash(uint256 playerId, address player, bytes32 cardHash, uint256 round, uint256 cardIndex, uint256 gameId)
func (_RoomV2Contract *RoomV2ContractFilterer) FilterSubmitCardHash(opts *bind.FilterOpts) (*RoomV2ContractSubmitCardHashIterator, error) {

	logs, sub, err := _RoomV2Contract.contract.FilterLogs(opts, "submitCardHash")
	if err != nil {
		return nil, err
	}
	return &RoomV2ContractSubmitCardHashIterator{contract: _RoomV2Contract.contract, event: "submitCardHash", logs: logs, sub: sub}, nil
}

// WatchSubmitCardHash is a free log subscription operation binding the contract event 0xf97708df45e82dc78722b3d12db0f7b5ed10df426b0f8069be9cc7e6aa08baf4.
//
// Solidity: event submitCardHash(uint256 playerId, address player, bytes32 cardHash, uint256 round, uint256 cardIndex, uint256 gameId)
func (_RoomV2Contract *RoomV2ContractFilterer) WatchSubmitCardHash(opts *bind.WatchOpts, sink chan<- *RoomV2ContractSubmitCardHash) (event.Subscription, error) {

	logs, sub, err := _RoomV2Contract.contract.WatchLogs(opts, "submitCardHash")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(RoomV2ContractSubmitCardHash)
				if err := _RoomV2Contract.contract.UnpackLog(event, "submitCardHash", log); err != nil {
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

// ParseSubmitCardHash is a log parse operation binding the contract event 0xf97708df45e82dc78722b3d12db0f7b5ed10df426b0f8069be9cc7e6aa08baf4.
//
// Solidity: event submitCardHash(uint256 playerId, address player, bytes32 cardHash, uint256 round, uint256 cardIndex, uint256 gameId)
func (_RoomV2Contract *RoomV2ContractFilterer) ParseSubmitCardHash(log types.Log) (*RoomV2ContractSubmitCardHash, error) {
	event := new(RoomV2ContractSubmitCardHash)
	if err := _RoomV2Contract.contract.UnpackLog(event, "submitCardHash", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
