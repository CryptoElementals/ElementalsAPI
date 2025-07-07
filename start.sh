#!/bin/bash

BIN_DIR="bin"
APP_NAME="beast-royale-server"
CONFIG_FILE="config.yaml"

nohup ./$BIN_DIR/$APP_NAME -config $CONFIG_FILE > stdout.log 2> stderr.log &
echo $! > $BIN_DIR/server.pid
echo "Server started. PID: $(cat $BIN_DIR/server.pid)" 