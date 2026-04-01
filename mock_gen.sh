#! /bin/bash

mockgen -destination=room_server/worker/testing/mock_handler.go -package=testing github.com/CryptoElementals/common/room_server/worker EventHandler
mockgen -destination=room_server/worker/testing/mock_game_creator.go -package=testing github.com/CryptoElementals/common/lobby_server/worker/queue GameCreator
mockgen -destination=room_server/worker/testing/mock_contract_client.go -package=testing github.com/CryptoElementals/common/room_server/worker/game ContractClient
mockgen -destination=room_server/worker/testing/mock_server.go -package=testing github.com/CryptoElementals/common/rpc/server  ChainRequestHandler,PlayerRequestHandler,GamePhaseHandler