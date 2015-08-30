#!/bin/bash

# Update repo
git pull

# Build
../build.sh

# Start the server
../start_server.sh &

# Start another few clients
../indispenso --server=false --hostname=client-one &
../indispenso --server=false --hostname=client-two &
../indispenso --server=false --hostname=client-three &

# Make sure we wait (without wasting cpu cycles), we read "nothing" :)
cat
