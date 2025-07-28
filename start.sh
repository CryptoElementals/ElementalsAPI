#!/bin/bash

BIN_DIR="bin"
API_SERVER="ele-apiserver"
API_CONFIG="config.yaml"

nohup ./$BIN_DIR/$API_SERVER run --config $API_CONFIG > /dev/null 2>&1 &
echo "$API_SERVER started with config $API_CONFIG."