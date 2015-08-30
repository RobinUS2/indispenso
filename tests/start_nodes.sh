#!/bin/bash

# Update repo
git pull

# Build
../build.sh
TOKEN=`cat ../server.token`
echo "Token from file $TOKEN"

# Start the server
../start_server.sh &

# Start another few clients
../indispenso --seed="https://localhost:897/" --hostname=client-one &

# Make sure we wait (without wasting cpu cycles), we read "nothing" :)
cat
