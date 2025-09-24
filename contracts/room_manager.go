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

// RoomManagerContractMetaData contains all meta data concerning the RoomManagerContract contract.
var RoomManagerContractMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"_player1\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"_player2\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"_player1_tmp\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"_player2_tmp\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_totalRound\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"_roomAddress\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_initialHP\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_gameId\",\"type\":\"uint256\"}],\"name\":\"RoomCreated\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_player1\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_player2\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_player1_tmp\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_player2_tmp\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_roundTimeout\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_totalRound\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_initialHP\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_gameId\",\"type\":\"uint256\"}],\"name\":\"CreateRoom\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"Rooms\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_newManager\",\"type\":\"address\"}],\"name\":\"addManager\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_creator\",\"type\":\"address\"}],\"name\":\"listRooms\",\"outputs\":[{\"internalType\":\"address[]\",\"name\":\"\",\"type\":\"address[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"managerIndex\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// RoomManagerContractABI is the input ABI used to generate the binding from.
// Deprecated: Use RoomManagerContractMetaData.ABI instead.
var RoomManagerContractABI = RoomManagerContractMetaData.ABI

// RoomManagerContract is an auto generated Go binding around an Ethereum contract.
type RoomManagerContract struct {
	RoomManagerContractCaller     // Read-only binding to the contract
	RoomManagerContractTransactor // Write-only binding to the contract
	RoomManagerContractFilterer   // Log filterer for contract events
}

// RoomManagerContractCaller is an auto generated read-only Go binding around an Ethereum contract.
type RoomManagerContractCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RoomManagerContractTransactor is an auto generated write-only Go binding around an Ethereum contract.
type RoomManagerContractTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RoomManagerContractFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type RoomManagerContractFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RoomManagerContractSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type RoomManagerContractSession struct {
	Contract     *RoomManagerContract // Generic contract binding to set the session for
	CallOpts     bind.CallOpts        // Call options to use throughout this session
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// RoomManagerContractCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type RoomManagerContractCallerSession struct {
	Contract *RoomManagerContractCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts              // Call options to use throughout this session
}

// RoomManagerContractTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type RoomManagerContractTransactorSession struct {
	Contract     *RoomManagerContractTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts              // Transaction auth options to use throughout this session
}

// RoomManagerContractRaw is an auto generated low-level Go binding around an Ethereum contract.
type RoomManagerContractRaw struct {
	Contract *RoomManagerContract // Generic contract binding to access the raw methods on
}

// RoomManagerContractCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type RoomManagerContractCallerRaw struct {
	Contract *RoomManagerContractCaller // Generic read-only contract binding to access the raw methods on
}

// RoomManagerContractTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type RoomManagerContractTransactorRaw struct {
	Contract *RoomManagerContractTransactor // Generic write-only contract binding to access the raw methods on
}

// NewRoomManagerContract creates a new instance of RoomManagerContract, bound to a specific deployed contract.
func NewRoomManagerContract(address common.Address, backend bind.ContractBackend) (*RoomManagerContract, error) {
	contract, err := bindRoomManagerContract(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &RoomManagerContract{RoomManagerContractCaller: RoomManagerContractCaller{contract: contract}, RoomManagerContractTransactor: RoomManagerContractTransactor{contract: contract}, RoomManagerContractFilterer: RoomManagerContractFilterer{contract: contract}}, nil
}

// NewRoomManagerContractCaller creates a new read-only instance of RoomManagerContract, bound to a specific deployed contract.
func NewRoomManagerContractCaller(address common.Address, caller bind.ContractCaller) (*RoomManagerContractCaller, error) {
	contract, err := bindRoomManagerContract(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &RoomManagerContractCaller{contract: contract}, nil
}

// NewRoomManagerContractTransactor creates a new write-only instance of RoomManagerContract, bound to a specific deployed contract.
func NewRoomManagerContractTransactor(address common.Address, transactor bind.ContractTransactor) (*RoomManagerContractTransactor, error) {
	contract, err := bindRoomManagerContract(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &RoomManagerContractTransactor{contract: contract}, nil
}

// NewRoomManagerContractFilterer creates a new log filterer instance of RoomManagerContract, bound to a specific deployed contract.
func NewRoomManagerContractFilterer(address common.Address, filterer bind.ContractFilterer) (*RoomManagerContractFilterer, error) {
	contract, err := bindRoomManagerContract(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &RoomManagerContractFilterer{contract: contract}, nil
}

// bindRoomManagerContract binds a generic wrapper to an already deployed contract.
func bindRoomManagerContract(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := RoomManagerContractMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_RoomManagerContract *RoomManagerContractRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _RoomManagerContract.Contract.RoomManagerContractCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_RoomManagerContract *RoomManagerContractRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _RoomManagerContract.Contract.RoomManagerContractTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_RoomManagerContract *RoomManagerContractRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _RoomManagerContract.Contract.RoomManagerContractTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_RoomManagerContract *RoomManagerContractCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _RoomManagerContract.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_RoomManagerContract *RoomManagerContractTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _RoomManagerContract.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_RoomManagerContract *RoomManagerContractTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _RoomManagerContract.Contract.contract.Transact(opts, method, params...)
}

// Rooms is a free data retrieval call binding the contract method 0xc47cc979.
//
// Solidity: function Rooms(address , uint256 ) view returns(address)
func (_RoomManagerContract *RoomManagerContractCaller) Rooms(opts *bind.CallOpts, arg0 common.Address, arg1 *big.Int) (common.Address, error) {
	var out []interface{}
	err := _RoomManagerContract.contract.Call(opts, &out, "Rooms", arg0, arg1)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Rooms is a free data retrieval call binding the contract method 0xc47cc979.
//
// Solidity: function Rooms(address , uint256 ) view returns(address)
func (_RoomManagerContract *RoomManagerContractSession) Rooms(arg0 common.Address, arg1 *big.Int) (common.Address, error) {
	return _RoomManagerContract.Contract.Rooms(&_RoomManagerContract.CallOpts, arg0, arg1)
}

// Rooms is a free data retrieval call binding the contract method 0xc47cc979.
//
// Solidity: function Rooms(address , uint256 ) view returns(address)
func (_RoomManagerContract *RoomManagerContractCallerSession) Rooms(arg0 common.Address, arg1 *big.Int) (common.Address, error) {
	return _RoomManagerContract.Contract.Rooms(&_RoomManagerContract.CallOpts, arg0, arg1)
}

// ListRooms is a free data retrieval call binding the contract method 0x48daaef4.
//
// Solidity: function listRooms(address _creator) view returns(address[])
func (_RoomManagerContract *RoomManagerContractCaller) ListRooms(opts *bind.CallOpts, _creator common.Address) ([]common.Address, error) {
	var out []interface{}
	err := _RoomManagerContract.contract.Call(opts, &out, "listRooms", _creator)

	if err != nil {
		return *new([]common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new([]common.Address)).(*[]common.Address)

	return out0, err

}

// ListRooms is a free data retrieval call binding the contract method 0x48daaef4.
//
// Solidity: function listRooms(address _creator) view returns(address[])
func (_RoomManagerContract *RoomManagerContractSession) ListRooms(_creator common.Address) ([]common.Address, error) {
	return _RoomManagerContract.Contract.ListRooms(&_RoomManagerContract.CallOpts, _creator)
}

// ListRooms is a free data retrieval call binding the contract method 0x48daaef4.
//
// Solidity: function listRooms(address _creator) view returns(address[])
func (_RoomManagerContract *RoomManagerContractCallerSession) ListRooms(_creator common.Address) ([]common.Address, error) {
	return _RoomManagerContract.Contract.ListRooms(&_RoomManagerContract.CallOpts, _creator)
}

// ManagerIndex is a free data retrieval call binding the contract method 0x573d44fa.
//
// Solidity: function managerIndex(address ) view returns(uint256)
func (_RoomManagerContract *RoomManagerContractCaller) ManagerIndex(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _RoomManagerContract.contract.Call(opts, &out, "managerIndex", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ManagerIndex is a free data retrieval call binding the contract method 0x573d44fa.
//
// Solidity: function managerIndex(address ) view returns(uint256)
func (_RoomManagerContract *RoomManagerContractSession) ManagerIndex(arg0 common.Address) (*big.Int, error) {
	return _RoomManagerContract.Contract.ManagerIndex(&_RoomManagerContract.CallOpts, arg0)
}

// ManagerIndex is a free data retrieval call binding the contract method 0x573d44fa.
//
// Solidity: function managerIndex(address ) view returns(uint256)
func (_RoomManagerContract *RoomManagerContractCallerSession) ManagerIndex(arg0 common.Address) (*big.Int, error) {
	return _RoomManagerContract.Contract.ManagerIndex(&_RoomManagerContract.CallOpts, arg0)
}

// CreateRoom is a paid mutator transaction binding the contract method 0x6f0daa9b.
//
// Solidity: function CreateRoom(address _player1, address _player2, address _player1_tmp, address _player2_tmp, uint256 _roundTimeout, uint256 _totalRound, uint256 _initialHP, uint256 _gameId) returns()
func (_RoomManagerContract *RoomManagerContractTransactor) CreateRoom(opts *bind.TransactOpts, _player1 common.Address, _player2 common.Address, _player1_tmp common.Address, _player2_tmp common.Address, _roundTimeout *big.Int, _totalRound *big.Int, _initialHP *big.Int, _gameId *big.Int) (*types.Transaction, error) {
	return _RoomManagerContract.contract.Transact(opts, "CreateRoom", _player1, _player2, _player1_tmp, _player2_tmp, _roundTimeout, _totalRound, _initialHP, _gameId)
}

// CreateRoom is a paid mutator transaction binding the contract method 0x6f0daa9b.
//
// Solidity: function CreateRoom(address _player1, address _player2, address _player1_tmp, address _player2_tmp, uint256 _roundTimeout, uint256 _totalRound, uint256 _initialHP, uint256 _gameId) returns()
func (_RoomManagerContract *RoomManagerContractSession) CreateRoom(_player1 common.Address, _player2 common.Address, _player1_tmp common.Address, _player2_tmp common.Address, _roundTimeout *big.Int, _totalRound *big.Int, _initialHP *big.Int, _gameId *big.Int) (*types.Transaction, error) {
	return _RoomManagerContract.Contract.CreateRoom(&_RoomManagerContract.TransactOpts, _player1, _player2, _player1_tmp, _player2_tmp, _roundTimeout, _totalRound, _initialHP, _gameId)
}

// CreateRoom is a paid mutator transaction binding the contract method 0x6f0daa9b.
//
// Solidity: function CreateRoom(address _player1, address _player2, address _player1_tmp, address _player2_tmp, uint256 _roundTimeout, uint256 _totalRound, uint256 _initialHP, uint256 _gameId) returns()
func (_RoomManagerContract *RoomManagerContractTransactorSession) CreateRoom(_player1 common.Address, _player2 common.Address, _player1_tmp common.Address, _player2_tmp common.Address, _roundTimeout *big.Int, _totalRound *big.Int, _initialHP *big.Int, _gameId *big.Int) (*types.Transaction, error) {
	return _RoomManagerContract.Contract.CreateRoom(&_RoomManagerContract.TransactOpts, _player1, _player2, _player1_tmp, _player2_tmp, _roundTimeout, _totalRound, _initialHP, _gameId)
}

// AddManager is a paid mutator transaction binding the contract method 0x2d06177a.
//
// Solidity: function addManager(address _newManager) returns()
func (_RoomManagerContract *RoomManagerContractTransactor) AddManager(opts *bind.TransactOpts, _newManager common.Address) (*types.Transaction, error) {
	return _RoomManagerContract.contract.Transact(opts, "addManager", _newManager)
}

// AddManager is a paid mutator transaction binding the contract method 0x2d06177a.
//
// Solidity: function addManager(address _newManager) returns()
func (_RoomManagerContract *RoomManagerContractSession) AddManager(_newManager common.Address) (*types.Transaction, error) {
	return _RoomManagerContract.Contract.AddManager(&_RoomManagerContract.TransactOpts, _newManager)
}

// AddManager is a paid mutator transaction binding the contract method 0x2d06177a.
//
// Solidity: function addManager(address _newManager) returns()
func (_RoomManagerContract *RoomManagerContractTransactorSession) AddManager(_newManager common.Address) (*types.Transaction, error) {
	return _RoomManagerContract.Contract.AddManager(&_RoomManagerContract.TransactOpts, _newManager)
}

// RoomManagerContractRoomCreatedIterator is returned from FilterRoomCreated and is used to iterate over the raw logs and unpacked data for RoomCreated events raised by the RoomManagerContract contract.
type RoomManagerContractRoomCreatedIterator struct {
	Event *RoomManagerContractRoomCreated // Event containing the contract specifics and raw log

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
func (it *RoomManagerContractRoomCreatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(RoomManagerContractRoomCreated)
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
		it.Event = new(RoomManagerContractRoomCreated)
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
func (it *RoomManagerContractRoomCreatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *RoomManagerContractRoomCreatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// RoomManagerContractRoomCreated represents a RoomCreated event raised by the RoomManagerContract contract.
type RoomManagerContractRoomCreated struct {
	Player1     common.Address
	Player2     common.Address
	Player1Tmp  common.Address
	Player2Tmp  common.Address
	TotalRound  *big.Int
	RoomAddress common.Address
	InitialHP   *big.Int
	GameId      *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterRoomCreated is a free log retrieval operation binding the contract event 0xa5bced1f977c40d8e1fce9f409acfff07cb439674b7da4b9b19531c0b27a0dfa.
//
// Solidity: event RoomCreated(address _player1, address _player2, address _player1_tmp, address _player2_tmp, uint256 _totalRound, address _roomAddress, uint256 _initialHP, uint256 _gameId)
func (_RoomManagerContract *RoomManagerContractFilterer) FilterRoomCreated(opts *bind.FilterOpts) (*RoomManagerContractRoomCreatedIterator, error) {

	logs, sub, err := _RoomManagerContract.contract.FilterLogs(opts, "RoomCreated")
	if err != nil {
		return nil, err
	}
	return &RoomManagerContractRoomCreatedIterator{contract: _RoomManagerContract.contract, event: "RoomCreated", logs: logs, sub: sub}, nil
}

// WatchRoomCreated is a free log subscription operation binding the contract event 0xa5bced1f977c40d8e1fce9f409acfff07cb439674b7da4b9b19531c0b27a0dfa.
//
// Solidity: event RoomCreated(address _player1, address _player2, address _player1_tmp, address _player2_tmp, uint256 _totalRound, address _roomAddress, uint256 _initialHP, uint256 _gameId)
func (_RoomManagerContract *RoomManagerContractFilterer) WatchRoomCreated(opts *bind.WatchOpts, sink chan<- *RoomManagerContractRoomCreated) (event.Subscription, error) {

	logs, sub, err := _RoomManagerContract.contract.WatchLogs(opts, "RoomCreated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(RoomManagerContractRoomCreated)
				if err := _RoomManagerContract.contract.UnpackLog(event, "RoomCreated", log); err != nil {
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

// ParseRoomCreated is a log parse operation binding the contract event 0xa5bced1f977c40d8e1fce9f409acfff07cb439674b7da4b9b19531c0b27a0dfa.
//
// Solidity: event RoomCreated(address _player1, address _player2, address _player1_tmp, address _player2_tmp, uint256 _totalRound, address _roomAddress, uint256 _initialHP, uint256 _gameId)
func (_RoomManagerContract *RoomManagerContractFilterer) ParseRoomCreated(log types.Log) (*RoomManagerContractRoomCreated, error) {
	event := new(RoomManagerContractRoomCreated)
	if err := _RoomManagerContract.contract.UnpackLog(event, "RoomCreated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
