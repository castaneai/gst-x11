#!/bin/bash
export DISPLAY=:99
Xvfb "${DISPLAY}" -screen 0 640x480x24 &
x11vnc -display WAIT"${DISPLAY}" -shared -forever &
"$@"
