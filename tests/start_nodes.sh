#!/bin/bash

# Update repo
git pull

# Build
../build.sh

# Start the server
../start_server.sh --debug=true --client-port=-1 >>/out.log 2>&1 &
sleep 3

# Start another few clients
../indispenso --seed="https://127.0.0.1:897/" --hostname="client-one" --debug=true --disable-server=true >>/out.log 2>&1 &
../indispenso --seed="https://127.0.0.1:897/" --hostname="client-two" --debug=true --disable-server=true --client-port=-1 >>/out.log 2>&1 &

# Make sure we wait (without wasting cpu cycles), we read "nothing" :)
tail -f /out.log
