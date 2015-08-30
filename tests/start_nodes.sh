#!/bin/bash

# Update repo
git pull

# Build
../build.sh

# Start the server
../start_server.sh
