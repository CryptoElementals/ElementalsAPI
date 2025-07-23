#! /bin/bash

mockgen -destination=room_server/worker/testing/mock_getters.go -package=testing github.com/CryptoElementals/common/room_server/worker/player GameInfoGetter,QueueInfoGetter
mockgen -destination=room_server/worker/testing/mock_handler.go -package=testing github.com/CryptoElementals/common/room_server/worker EventHandler
mockgen -destination=room_server/worker/testing/mock_server.go -package=testing github.com/CryptoElementals/common/rpc/server PlayerManager,GameRequestHandler,ChainRequestHandler,PlayerRequestHandler