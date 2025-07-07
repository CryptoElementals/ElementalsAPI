#!/bin/bash

BIN_DIR="bin"
PID_FILE="$BIN_DIR/server.pid"

if [ ! -f "$PID_FILE" ]; then
  echo "No PID file found. Server may not be running."
  exit 1
fi

PID=$(cat $PID_FILE)
if kill -0 $PID 2>/dev/null; then
  kill $PID
  echo "Server (PID: $PID) stopped."
  rm -f $PID_FILE
else
  echo "Process $PID not running. Removing stale PID file."
  rm -f $PID_FILE
fi 