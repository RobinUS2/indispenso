#!/bin/bash
DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
cd ${DIR}

IND_HOME=${IND_HOME-/etc/indispenso}

# Auto create folder
mkdir -p ${IND_HOME}

TOKEN=${IND_SERVER_TOKEN-`openssl rand -base64 128 | tr -d '\n'`}

echo -e "Server token: ${TOKEN}\n"

# Start server
./indispenso -s -t "${TOKEN}" $@
