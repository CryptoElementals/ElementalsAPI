#!/bin/bash

BIN_DIR="bin"
APP_NAME="beast-royale-server"
CONFIG_FILE="config.yaml"

nohup ./$BIN_DIR/$APP_NAME run --config $CONFIG_FILE > /dev/null 2>&1 &
echo "Server started." 