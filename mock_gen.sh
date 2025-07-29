#! /bin/bash

mockgen -destination=room_server/worker/testing/mock_handler.go -package=testing github.com/CryptoElementals/common/room_server/worker EventHandler
mockgen -destination=room_server/worker/testing/mock_game_creator.go -package=testing github.com/CryptoElementals/common/room_server/worker/queue GameCreator
mockgen -destination=room_server/worker/testing/mock_contract_client.go -package=testing github.com/CryptoElementals/common/room_server/worker/game ContractClient
mockgen -destination=room_server/worker/testing/mock_getters.go -package=testing github.com/CryptoElementals/common/room_server/worker/player GameInfoGetter,Queuer