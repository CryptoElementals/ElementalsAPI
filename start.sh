#!/bin/bash

BIN_DIR="bin"
API_SERVER="ele-apiserver"
ROOM_SERVER="ele-roomserver"
API_CONFIG="config.yaml"
ROOM_CONFIG="config.yaml"

nohup ./$BIN_DIR/$ROOM_SERVER --config $ROOM_CONFIG > /dev/null 2>&1 &
echo "$ROOM_SERVER started with config $ROOM_CONFIG."

nohup ./$BIN_DIR/$API_SERVER --config $API_CONFIG > /dev/null 2>&1 &
echo "$API_SERVER started with config $API_CONFIG."