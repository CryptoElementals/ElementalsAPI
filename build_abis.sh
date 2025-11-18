#! /bin/bash

go_pkg="contract"

abigen --abi ./contracts/room.abi --pkg $go_pkg --type RoomContract --out ./contracts/room.go
abigen --abi ./contracts/RoomV2.abi --pkg $go_pkg --type RoomV2Contract --out ./contracts/room_v2.go
abigen --abi ./contracts/player_state.abi --pkg $go_pkg --type PlayerStateContract --out ./contracts/player_state.go
abigen --abi ./contracts/room_manager.abi --pkg $go_pkg --type RoomManagerContract --out ./contracts/room_manager.go