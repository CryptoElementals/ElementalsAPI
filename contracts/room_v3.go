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

// RoomV3ContractMetaData contains all meta data concerning the RoomV3Contract contract.
var RoomV3ContractMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_player1Id\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_player2Id\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"_player1_tmp\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"_player2_tmp\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_totalRound\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_totalCardIndex\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_initialHP\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_gameId\",\"type\":\"uint256\"}],\"name\":\"RoomCreated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"cardIndex\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"round\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"gameId\",\"type\":\"uint256\"}],\"name\":\"startANewTurn\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"playerId\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"player\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"card\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"salt\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"round\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"cardIndex\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"gameId\",\"type\":\"uint256\"}],\"name\":\"submitCard\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"playerId\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"player\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"cardHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"round\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"cardIndex\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"gameId\",\"type\":\"uint256\"}],\"name\":\"submitCardHash\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_newManager\",\"type\":\"address\"}],\"name\":\"addManager\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint8[]\",\"name\":\"taskIndexes\",\"type\":\"uint8[]\"},{\"internalType\":\"bytes[]\",\"name\":\"tasks\",\"type\":\"bytes[]\"}],\"name\":\"batchSubmitTasks\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"gameId\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"gameIdIndex\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_gameId\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_round\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_cardIndex\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_playerIndex\",\"type\":\"uint256\"}],\"name\":\"getCard\",\"outputs\":[{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_gameId\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_round\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_cardIndex\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_playerIndex\",\"type\":\"uint256\"}],\"name\":\"getCardHash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"managerIndex\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"roomData\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"currentRound\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"currentCardIndex\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"totalRound\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"totalCardIndex\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"creator\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"player1Id\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"player2Id\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"player1Temp\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"player2Temp\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"launchTime\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"roundTimeout\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"initialHP\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// RoomV3ContractABI is the input ABI used to generate the binding from.
// Deprecated: Use RoomV3ContractMetaData.ABI instead.
var RoomV3ContractABI = RoomV3ContractMetaData.ABI

// RoomV3Contract is an auto generated Go binding around an Ethereum contract.
type RoomV3Contract struct {
	RoomV3ContractCaller     // Read-only binding to the contract
	RoomV3ContractTransactor // Write-only binding to the contract
	RoomV3ContractFilterer   // Log filterer for contract events
}

// RoomV3ContractCaller is an auto generated read-only Go binding around an Ethereum contract.
type RoomV3ContractCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RoomV3ContractTransactor is an auto generated write-only Go binding around an Ethereum contract.
type RoomV3ContractTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RoomV3ContractFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type RoomV3ContractFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RoomV3ContractSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type RoomV3ContractSession struct {
	Contract     *RoomV3Contract   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// RoomV3ContractCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type RoomV3ContractCallerSession struct {
	Contract *RoomV3ContractCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// RoomV3ContractTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type RoomV3ContractTransactorSession struct {
	Contract     *RoomV3ContractTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// RoomV3ContractRaw is an auto generated low-level Go binding around an Ethereum contract.
type RoomV3ContractRaw struct {
	Contract *RoomV3Contract // Generic contract binding to access the raw methods on
}

// RoomV3ContractCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type RoomV3ContractCallerRaw struct {
	Contract *RoomV3ContractCaller // Generic read-only contract binding to access the raw methods on
}

// RoomV3ContractTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type RoomV3ContractTransactorRaw struct {
	Contract *RoomV3ContractTransactor // Generic write-only contract binding to access the raw methods on
}

// NewRoomV3Contract creates a new instance of RoomV3Contract, bound to a specific deployed contract.
func NewRoomV3Contract(address common.Address, backend bind.ContractBackend) (*RoomV3Contract, error) {
	contract, err := bindRoomV3Contract(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &RoomV3Contract{RoomV3ContractCaller: RoomV3ContractCaller{contract: contract}, RoomV3ContractTransactor: RoomV3ContractTransactor{contract: contract}, RoomV3ContractFilterer: RoomV3ContractFilterer{contract: contract}}, nil
}

// NewRoomV3ContractCaller creates a new read-only instance of RoomV3Contract, bound to a specific deployed contract.
func NewRoomV3ContractCaller(address common.Address, caller bind.ContractCaller) (*RoomV3ContractCaller, error) {
	contract, err := bindRoomV3Contract(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &RoomV3ContractCaller{contract: contract}, nil
}

// NewRoomV3ContractTransactor creates a new write-only instance of RoomV3Contract, bound to a specific deployed contract.
func NewRoomV3ContractTransactor(address common.Address, transactor bind.ContractTransactor) (*RoomV3ContractTransactor, error) {
	contract, err := bindRoomV3Contract(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &RoomV3ContractTransactor{contract: contract}, nil
}

// NewRoomV3ContractFilterer creates a new log filterer instance of RoomV3Contract, bound to a specific deployed contract.
func NewRoomV3ContractFilterer(address common.Address, filterer bind.ContractFilterer) (*RoomV3ContractFilterer, error) {
	contract, err := bindRoomV3Contract(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &RoomV3ContractFilterer{contract: contract}, nil
}

// bindRoomV3Contract binds a generic wrapper to an already deployed contract.
func bindRoomV3Contract(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := RoomV3ContractMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_RoomV3Contract *RoomV3ContractRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _RoomV3Contract.Contract.RoomV3ContractCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_RoomV3Contract *RoomV3ContractRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _RoomV3Contract.Contract.RoomV3ContractTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_RoomV3Contract *RoomV3ContractRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _RoomV3Contract.Contract.RoomV3ContractTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_RoomV3Contract *RoomV3ContractCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _RoomV3Contract.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_RoomV3Contract *RoomV3ContractTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _RoomV3Contract.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_RoomV3Contract *RoomV3ContractTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _RoomV3Contract.Contract.contract.Transact(opts, method, params...)
}

// GameId is a free data retrieval call binding the contract method 0x168aa23a.
//
// Solidity: function gameId(uint256 ) view returns(uint256)
func (_RoomV3Contract *RoomV3ContractCaller) GameId(opts *bind.CallOpts, arg0 *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _RoomV3Contract.contract.Call(opts, &out, "gameId", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GameId is a free data retrieval call binding the contract method 0x168aa23a.
//
// Solidity: function gameId(uint256 ) view returns(uint256)
func (_RoomV3Contract *RoomV3ContractSession) GameId(arg0 *big.Int) (*big.Int, error) {
	return _RoomV3Contract.Contract.GameId(&_RoomV3Contract.CallOpts, arg0)
}

// GameId is a free data retrieval call binding the contract method 0x168aa23a.
//
// Solidity: function gameId(uint256 ) view returns(uint256)
func (_RoomV3Contract *RoomV3ContractCallerSession) GameId(arg0 *big.Int) (*big.Int, error) {
	return _RoomV3Contract.Contract.GameId(&_RoomV3Contract.CallOpts, arg0)
}

// GameIdIndex is a free data retrieval call binding the contract method 0xa3157b47.
//
// Solidity: function gameIdIndex(uint256 ) view returns(uint256)
func (_RoomV3Contract *RoomV3ContractCaller) GameIdIndex(opts *bind.CallOpts, arg0 *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _RoomV3Contract.contract.Call(opts, &out, "gameIdIndex", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GameIdIndex is a free data retrieval call binding the contract method 0xa3157b47.
//
// Solidity: function gameIdIndex(uint256 ) view returns(uint256)
func (_RoomV3Contract *RoomV3ContractSession) GameIdIndex(arg0 *big.Int) (*big.Int, error) {
	return _RoomV3Contract.Contract.GameIdIndex(&_RoomV3Contract.CallOpts, arg0)
}

// GameIdIndex is a free data retrieval call binding the contract method 0xa3157b47.
//
// Solidity: function gameIdIndex(uint256 ) view returns(uint256)
func (_RoomV3Contract *RoomV3ContractCallerSession) GameIdIndex(arg0 *big.Int) (*big.Int, error) {
	return _RoomV3Contract.Contract.GameIdIndex(&_RoomV3Contract.CallOpts, arg0)
}

// GetCard is a free data retrieval call binding the contract method 0x833c8d3f.
//
// Solidity: function getCard(uint256 _gameId, uint256 _round, uint256 _cardIndex, uint256 _playerIndex) view returns(bytes)
func (_RoomV3Contract *RoomV3ContractCaller) GetCard(opts *bind.CallOpts, _gameId *big.Int, _round *big.Int, _cardIndex *big.Int, _playerIndex *big.Int) ([]byte, error) {
	var out []interface{}
	err := _RoomV3Contract.contract.Call(opts, &out, "getCard", _gameId, _round, _cardIndex, _playerIndex)

	if err != nil {
		return *new([]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([]byte)).(*[]byte)

	return out0, err

}

// GetCard is a free data retrieval call binding the contract method 0x833c8d3f.
//
// Solidity: function getCard(uint256 _gameId, uint256 _round, uint256 _cardIndex, uint256 _playerIndex) view returns(bytes)
func (_RoomV3Contract *RoomV3ContractSession) GetCard(_gameId *big.Int, _round *big.Int, _cardIndex *big.Int, _playerIndex *big.Int) ([]byte, error) {
	return _RoomV3Contract.Contract.GetCard(&_RoomV3Contract.CallOpts, _gameId, _round, _cardIndex, _playerIndex)
}

// GetCard is a free data retrieval call binding the contract method 0x833c8d3f.
//
// Solidity: function getCard(uint256 _gameId, uint256 _round, uint256 _cardIndex, uint256 _playerIndex) view returns(bytes)
func (_RoomV3Contract *RoomV3ContractCallerSession) GetCard(_gameId *big.Int, _round *big.Int, _cardIndex *big.Int, _playerIndex *big.Int) ([]byte, error) {
	return _RoomV3Contract.Contract.GetCard(&_RoomV3Contract.CallOpts, _gameId, _round, _cardIndex, _playerIndex)
}

// GetCardHash is a free data retrieval call binding the contract method 0xa942de03.
//
// Solidity: function getCardHash(uint256 _gameId, uint256 _round, uint256 _cardIndex, uint256 _playerIndex) view returns(bytes32)
func (_RoomV3Contract *RoomV3ContractCaller) GetCardHash(opts *bind.CallOpts, _gameId *big.Int, _round *big.Int, _cardIndex *big.Int, _playerIndex *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _RoomV3Contract.contract.Call(opts, &out, "getCardHash", _gameId, _round, _cardIndex, _playerIndex)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetCardHash is a free data retrieval call binding the contract method 0xa942de03.
//
// Solidity: function getCardHash(uint256 _gameId, uint256 _round, uint256 _cardIndex, uint256 _playerIndex) view returns(bytes32)
func (_RoomV3Contract *RoomV3ContractSession) GetCardHash(_gameId *big.Int, _round *big.Int, _cardIndex *big.Int, _playerIndex *big.Int) ([32]byte, error) {
	return _RoomV3Contract.Contract.GetCardHash(&_RoomV3Contract.CallOpts, _gameId, _round, _cardIndex, _playerIndex)
}

// GetCardHash is a free data retrieval call binding the contract method 0xa942de03.
//
// Solidity: function getCardHash(uint256 _gameId, uint256 _round, uint256 _cardIndex, uint256 _playerIndex) view returns(bytes32)
func (_RoomV3Contract *RoomV3ContractCallerSession) GetCardHash(_gameId *big.Int, _round *big.Int, _cardIndex *big.Int, _playerIndex *big.Int) ([32]byte, error) {
	return _RoomV3Contract.Contract.GetCardHash(&_RoomV3Contract.CallOpts, _gameId, _round, _cardIndex, _playerIndex)
}

// ManagerIndex is a free data retrieval call binding the contract method 0x573d44fa.
//
// Solidity: function managerIndex(address ) view returns(uint256)
func (_RoomV3Contract *RoomV3ContractCaller) ManagerIndex(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _RoomV3Contract.contract.Call(opts, &out, "managerIndex", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ManagerIndex is a free data retrieval call binding the contract method 0x573d44fa.
//
// Solidity: function managerIndex(address ) view returns(uint256)
func (_RoomV3Contract *RoomV3ContractSession) ManagerIndex(arg0 common.Address) (*big.Int, error) {
	return _RoomV3Contract.Contract.ManagerIndex(&_RoomV3Contract.CallOpts, arg0)
}

// ManagerIndex is a free data retrieval call binding the contract method 0x573d44fa.
//
// Solidity: function managerIndex(address ) view returns(uint256)
func (_RoomV3Contract *RoomV3ContractCallerSession) ManagerIndex(arg0 common.Address) (*big.Int, error) {
	return _RoomV3Contract.Contract.ManagerIndex(&_RoomV3Contract.CallOpts, arg0)
}

// RoomData is a free data retrieval call binding the contract method 0xe193dff8.
//
// Solidity: function roomData(uint256 ) view returns(uint256 currentRound, uint256 currentCardIndex, uint256 totalRound, uint256 totalCardIndex, address creator, uint256 player1Id, uint256 player2Id, address player1Temp, address player2Temp, uint256 launchTime, uint256 roundTimeout, uint256 initialHP)
func (_RoomV3Contract *RoomV3ContractCaller) RoomData(opts *bind.CallOpts, arg0 *big.Int) (struct {
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
	err := _RoomV3Contract.contract.Call(opts, &out, "roomData", arg0)

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
func (_RoomV3Contract *RoomV3ContractSession) RoomData(arg0 *big.Int) (struct {
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
	return _RoomV3Contract.Contract.RoomData(&_RoomV3Contract.CallOpts, arg0)
}

// RoomData is a free data retrieval call binding the contract method 0xe193dff8.
//
// Solidity: function roomData(uint256 ) view returns(uint256 currentRound, uint256 currentCardIndex, uint256 totalRound, uint256 totalCardIndex, address creator, uint256 player1Id, uint256 player2Id, address player1Temp, address player2Temp, uint256 launchTime, uint256 roundTimeout, uint256 initialHP)
func (_RoomV3Contract *RoomV3ContractCallerSession) RoomData(arg0 *big.Int) (struct {
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
	return _RoomV3Contract.Contract.RoomData(&_RoomV3Contract.CallOpts, arg0)
}

// AddManager is a paid mutator transaction binding the contract method 0x2d06177a.
//
// Solidity: function addManager(address _newManager) returns()
func (_RoomV3Contract *RoomV3ContractTransactor) AddManager(opts *bind.TransactOpts, _newManager common.Address) (*types.Transaction, error) {
	return _RoomV3Contract.contract.Transact(opts, "addManager", _newManager)
}

// AddManager is a paid mutator transaction binding the contract method 0x2d06177a.
//
// Solidity: function addManager(address _newManager) returns()
func (_RoomV3Contract *RoomV3ContractSession) AddManager(_newManager common.Address) (*types.Transaction, error) {
	return _RoomV3Contract.Contract.AddManager(&_RoomV3Contract.TransactOpts, _newManager)
}

// AddManager is a paid mutator transaction binding the contract method 0x2d06177a.
//
// Solidity: function addManager(address _newManager) returns()
func (_RoomV3Contract *RoomV3ContractTransactorSession) AddManager(_newManager common.Address) (*types.Transaction, error) {
	return _RoomV3Contract.Contract.AddManager(&_RoomV3Contract.TransactOpts, _newManager)
}

// BatchSubmitTasks is a paid mutator transaction binding the contract method 0xae478660.
//
// Solidity: function batchSubmitTasks(uint8[] taskIndexes, bytes[] tasks) returns()
func (_RoomV3Contract *RoomV3ContractTransactor) BatchSubmitTasks(opts *bind.TransactOpts, taskIndexes []uint8, tasks [][]byte) (*types.Transaction, error) {
	return _RoomV3Contract.contract.Transact(opts, "batchSubmitTasks", taskIndexes, tasks)
}

// BatchSubmitTasks is a paid mutator transaction binding the contract method 0xae478660.
//
// Solidity: function batchSubmitTasks(uint8[] taskIndexes, bytes[] tasks) returns()
func (_RoomV3Contract *RoomV3ContractSession) BatchSubmitTasks(taskIndexes []uint8, tasks [][]byte) (*types.Transaction, error) {
	return _RoomV3Contract.Contract.BatchSubmitTasks(&_RoomV3Contract.TransactOpts, taskIndexes, tasks)
}

// BatchSubmitTasks is a paid mutator transaction binding the contract method 0xae478660.
//
// Solidity: function batchSubmitTasks(uint8[] taskIndexes, bytes[] tasks) returns()
func (_RoomV3Contract *RoomV3ContractTransactorSession) BatchSubmitTasks(taskIndexes []uint8, tasks [][]byte) (*types.Transaction, error) {
	return _RoomV3Contract.Contract.BatchSubmitTasks(&_RoomV3Contract.TransactOpts, taskIndexes, tasks)
}

// RoomV3ContractRoomCreatedIterator is returned from FilterRoomCreated and is used to iterate over the raw logs and unpacked data for RoomCreated events raised by the RoomV3Contract contract.
type RoomV3ContractRoomCreatedIterator struct {
	Event *RoomV3ContractRoomCreated // Event containing the contract specifics and raw log

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
func (it *RoomV3ContractRoomCreatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(RoomV3ContractRoomCreated)
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
		it.Event = new(RoomV3ContractRoomCreated)
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
func (it *RoomV3ContractRoomCreatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *RoomV3ContractRoomCreatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// RoomV3ContractRoomCreated represents a RoomCreated event raised by the RoomV3Contract contract.
type RoomV3ContractRoomCreated struct {
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
func (_RoomV3Contract *RoomV3ContractFilterer) FilterRoomCreated(opts *bind.FilterOpts) (*RoomV3ContractRoomCreatedIterator, error) {

	logs, sub, err := _RoomV3Contract.contract.FilterLogs(opts, "RoomCreated")
	if err != nil {
		return nil, err
	}
	return &RoomV3ContractRoomCreatedIterator{contract: _RoomV3Contract.contract, event: "RoomCreated", logs: logs, sub: sub}, nil
}

// WatchRoomCreated is a free log subscription operation binding the contract event 0x103b46b7007b0717433baa83443c5aa46c84bb502276b1508d3ba3ecef4aac6a.
//
// Solidity: event RoomCreated(uint256 _player1Id, uint256 _player2Id, address _player1_tmp, address _player2_tmp, uint256 _totalRound, uint256 _totalCardIndex, uint256 _initialHP, uint256 _gameId)
func (_RoomV3Contract *RoomV3ContractFilterer) WatchRoomCreated(opts *bind.WatchOpts, sink chan<- *RoomV3ContractRoomCreated) (event.Subscription, error) {

	logs, sub, err := _RoomV3Contract.contract.WatchLogs(opts, "RoomCreated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(RoomV3ContractRoomCreated)
				if err := _RoomV3Contract.contract.UnpackLog(event, "RoomCreated", log); err != nil {
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
func (_RoomV3Contract *RoomV3ContractFilterer) ParseRoomCreated(log types.Log) (*RoomV3ContractRoomCreated, error) {
	event := new(RoomV3ContractRoomCreated)
	if err := _RoomV3Contract.contract.UnpackLog(event, "RoomCreated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// RoomV3ContractStartANewTurnIterator is returned from FilterStartANewTurn and is used to iterate over the raw logs and unpacked data for StartANewTurn events raised by the RoomV3Contract contract.
type RoomV3ContractStartANewTurnIterator struct {
	Event *RoomV3ContractStartANewTurn // Event containing the contract specifics and raw log

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
func (it *RoomV3ContractStartANewTurnIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(RoomV3ContractStartANewTurn)
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
		it.Event = new(RoomV3ContractStartANewTurn)
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
func (it *RoomV3ContractStartANewTurnIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *RoomV3ContractStartANewTurnIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// RoomV3ContractStartANewTurn represents a StartANewTurn event raised by the RoomV3Contract contract.
type RoomV3ContractStartANewTurn struct {
	CardIndex *big.Int
	Round     *big.Int
	GameId    *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterStartANewTurn is a free log retrieval operation binding the contract event 0x882606b8d795d9c8d0f6dc344b6b949ddc29ced7c5512e714aff5ef6ee5e9cf8.
//
// Solidity: event startANewTurn(uint256 cardIndex, uint256 round, uint256 gameId)
func (_RoomV3Contract *RoomV3ContractFilterer) FilterStartANewTurn(opts *bind.FilterOpts) (*RoomV3ContractStartANewTurnIterator, error) {

	logs, sub, err := _RoomV3Contract.contract.FilterLogs(opts, "startANewTurn")
	if err != nil {
		return nil, err
	}
	return &RoomV3ContractStartANewTurnIterator{contract: _RoomV3Contract.contract, event: "startANewTurn", logs: logs, sub: sub}, nil
}

// WatchStartANewTurn is a free log subscription operation binding the contract event 0x882606b8d795d9c8d0f6dc344b6b949ddc29ced7c5512e714aff5ef6ee5e9cf8.
//
// Solidity: event startANewTurn(uint256 cardIndex, uint256 round, uint256 gameId)
func (_RoomV3Contract *RoomV3ContractFilterer) WatchStartANewTurn(opts *bind.WatchOpts, sink chan<- *RoomV3ContractStartANewTurn) (event.Subscription, error) {

	logs, sub, err := _RoomV3Contract.contract.WatchLogs(opts, "startANewTurn")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(RoomV3ContractStartANewTurn)
				if err := _RoomV3Contract.contract.UnpackLog(event, "startANewTurn", log); err != nil {
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
func (_RoomV3Contract *RoomV3ContractFilterer) ParseStartANewTurn(log types.Log) (*RoomV3ContractStartANewTurn, error) {
	event := new(RoomV3ContractStartANewTurn)
	if err := _RoomV3Contract.contract.UnpackLog(event, "startANewTurn", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// RoomV3ContractSubmitCardIterator is returned from FilterSubmitCard and is used to iterate over the raw logs and unpacked data for SubmitCard events raised by the RoomV3Contract contract.
type RoomV3ContractSubmitCardIterator struct {
	Event *RoomV3ContractSubmitCard // Event containing the contract specifics and raw log

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
func (it *RoomV3ContractSubmitCardIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(RoomV3ContractSubmitCard)
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
		it.Event = new(RoomV3ContractSubmitCard)
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
func (it *RoomV3ContractSubmitCardIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *RoomV3ContractSubmitCardIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// RoomV3ContractSubmitCard represents a SubmitCard event raised by the RoomV3Contract contract.
type RoomV3ContractSubmitCard struct {
	PlayerId  *big.Int
	Player    common.Address
	Card      *big.Int
	Salt      [32]byte
	Round     *big.Int
	CardIndex *big.Int
	GameId    *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterSubmitCard is a free log retrieval operation binding the contract event 0x8a8f5a4d2262edd29529747b4822394276e61b8c2f4f4278d33bb8e395daf053.
//
// Solidity: event submitCard(uint256 playerId, address player, uint256 card, bytes32 salt, uint256 round, uint256 cardIndex, uint256 gameId)
func (_RoomV3Contract *RoomV3ContractFilterer) FilterSubmitCard(opts *bind.FilterOpts) (*RoomV3ContractSubmitCardIterator, error) {

	logs, sub, err := _RoomV3Contract.contract.FilterLogs(opts, "submitCard")
	if err != nil {
		return nil, err
	}
	return &RoomV3ContractSubmitCardIterator{contract: _RoomV3Contract.contract, event: "submitCard", logs: logs, sub: sub}, nil
}

// WatchSubmitCard is a free log subscription operation binding the contract event 0x8a8f5a4d2262edd29529747b4822394276e61b8c2f4f4278d33bb8e395daf053.
//
// Solidity: event submitCard(uint256 playerId, address player, uint256 card, bytes32 salt, uint256 round, uint256 cardIndex, uint256 gameId)
func (_RoomV3Contract *RoomV3ContractFilterer) WatchSubmitCard(opts *bind.WatchOpts, sink chan<- *RoomV3ContractSubmitCard) (event.Subscription, error) {

	logs, sub, err := _RoomV3Contract.contract.WatchLogs(opts, "submitCard")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(RoomV3ContractSubmitCard)
				if err := _RoomV3Contract.contract.UnpackLog(event, "submitCard", log); err != nil {
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

// ParseSubmitCard is a log parse operation binding the contract event 0x8a8f5a4d2262edd29529747b4822394276e61b8c2f4f4278d33bb8e395daf053.
//
// Solidity: event submitCard(uint256 playerId, address player, uint256 card, bytes32 salt, uint256 round, uint256 cardIndex, uint256 gameId)
func (_RoomV3Contract *RoomV3ContractFilterer) ParseSubmitCard(log types.Log) (*RoomV3ContractSubmitCard, error) {
	event := new(RoomV3ContractSubmitCard)
	if err := _RoomV3Contract.contract.UnpackLog(event, "submitCard", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// RoomV3ContractSubmitCardHashIterator is returned from FilterSubmitCardHash and is used to iterate over the raw logs and unpacked data for SubmitCardHash events raised by the RoomV3Contract contract.
type RoomV3ContractSubmitCardHashIterator struct {
	Event *RoomV3ContractSubmitCardHash // Event containing the contract specifics and raw log

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
func (it *RoomV3ContractSubmitCardHashIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(RoomV3ContractSubmitCardHash)
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
		it.Event = new(RoomV3ContractSubmitCardHash)
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
func (it *RoomV3ContractSubmitCardHashIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *RoomV3ContractSubmitCardHashIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// RoomV3ContractSubmitCardHash represents a SubmitCardHash event raised by the RoomV3Contract contract.
type RoomV3ContractSubmitCardHash struct {
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
func (_RoomV3Contract *RoomV3ContractFilterer) FilterSubmitCardHash(opts *bind.FilterOpts) (*RoomV3ContractSubmitCardHashIterator, error) {

	logs, sub, err := _RoomV3Contract.contract.FilterLogs(opts, "submitCardHash")
	if err != nil {
		return nil, err
	}
	return &RoomV3ContractSubmitCardHashIterator{contract: _RoomV3Contract.contract, event: "submitCardHash", logs: logs, sub: sub}, nil
}

// WatchSubmitCardHash is a free log subscription operation binding the contract event 0xf97708df45e82dc78722b3d12db0f7b5ed10df426b0f8069be9cc7e6aa08baf4.
//
// Solidity: event submitCardHash(uint256 playerId, address player, bytes32 cardHash, uint256 round, uint256 cardIndex, uint256 gameId)
func (_RoomV3Contract *RoomV3ContractFilterer) WatchSubmitCardHash(opts *bind.WatchOpts, sink chan<- *RoomV3ContractSubmitCardHash) (event.Subscription, error) {

	logs, sub, err := _RoomV3Contract.contract.WatchLogs(opts, "submitCardHash")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(RoomV3ContractSubmitCardHash)
				if err := _RoomV3Contract.contract.UnpackLog(event, "submitCardHash", log); err != nil {
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
func (_RoomV3Contract *RoomV3ContractFilterer) ParseSubmitCardHash(log types.Log) (*RoomV3ContractSubmitCardHash, error) {
	event := new(RoomV3ContractSubmitCardHash)
	if err := _RoomV3Contract.contract.UnpackLog(event, "submitCardHash", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
