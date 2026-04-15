#! /bin/bash

mockgen -destination=room_server/worker/testing/mock_handler.go -package=testing github.com/CryptoElementals/common/room_server/worker EventHandler
mockgen -destination=room_server/worker/testing/mock_game_creator.go -package=testing github.com/CryptoElementals/common/lobby_server/worker/queue GameCreator
# RoomChain mock is maintained as room_server/worker/testing/mock_room_chain.go (gomock version skew)
mockgen -destination=room_server/worker/testing/mock_server.go -package=testing github.com/CryptoElementals/common/rpc/server  ChainRequestHandler,PlayerRequestHandler