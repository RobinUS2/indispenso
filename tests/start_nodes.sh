#!/bin/bash

# Update repo
git pull

# Build
../build.sh
TOKEN=`cat ../server.token`
echo "Token from file $TOKEN"

# Start the server
../start_server.sh &
sleep 1

# Start another few clients
../indispenso --seed="https://localhost:897/" --hostname=client-one &
../indispenso --seed="https://localhost:897/" --hostname=client-two &

# Make sure we wait (without wasting cpu cycles), we read "nothing" :)
cat
