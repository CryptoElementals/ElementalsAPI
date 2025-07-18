#! /bin/bash

go_pkg="contract"

abigen --abi ./room.abi --pkg $go_pkg --type RoomContract --out ./room.go
abigen --abi ./player_state.abi --pkg $go_pkg --type PlayerStateContract --out ./player_state.go
abigen --abi ./room_manager.abi --pkg $go_pkg --type RoomManagerContract --out ./room_manager.go