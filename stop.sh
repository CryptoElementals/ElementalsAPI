#!/bin/bash

APP_NAME="beast-royale-server"

# 查找所有相关进程（排除grep本身和stop.sh本身）
PIDS=$(ps aux | grep "$APP_NAME" | grep -v grep | grep -v stop.sh | awk '{print $2}')

if [ -z "$PIDS" ]; then
    echo "No $APP_NAME process found."
    exit 0
fi

echo "Killing $APP_NAME processes: $PIDS"
for PID in $PIDS; do
    kill $PID
    echo "Killed process $PID"
done

echo "All $APP_NAME processes stopped." 