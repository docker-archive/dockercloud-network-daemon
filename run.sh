#!/bin/bash
set -e

echo "Testing if docker binary is available"
docker version > /dev/null

if docker ps | awk '{print $2}' | grep -q -F 'zettio/weave'; then
	echo "Weave router is already running"
else
	if [ "${WEAVE_LAUNCH}" = "**None**" ]; then
		echo "WEAVE_LAUNCH is **None**. Not running 'weave launch'"
	else
		echo "Running: weave launch ${WEAVE_LAUNCH}"
		/weave launch ${WEAVE_LAUNCH}
	fi
fi

echo "Starting peer discovery daemon"
exec python -u /monitor.py
