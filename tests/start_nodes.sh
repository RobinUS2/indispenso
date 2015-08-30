#!/bin/bash

# Update repo
git pull

# Build
../build.sh
TOKEN=`cat ../server.token`
echo "Token from file $TOKEN"

# Start the server
../start_server.sh --debug=true &
sleep 1

# Start another few clients
../indispenso --seed="https://127.0.0.1:897/" --hostname="client-one" --debug=true &
../indispenso --seed="https://127.0.0.1:897/" --hostname="client-two" --debug=true &

# Make sure we wait (without wasting cpu cycles), we read "nothing" :)
cat
