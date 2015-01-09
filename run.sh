#!/bin/bash

if [ "${WEAVE_LAUNCH}" = "**None**" ]; then
    echo "WEAVE_LAUNCH is **None**. Do not run weave launch"
else
    if [ ! -f /.weave_launched ]; then
        echo "Runing weave launch ${WEAVE_LAUNCH}"
        /weave launch ${WEAVE_LAUNCH}
    fi
    touch ./weave_launched
fi

python monitor.py
