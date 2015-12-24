#!/bin/bash
DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
cd $DIR

IND_HOME=${IND_HOME-/etc/indispenso}

# Auto create foldder
mkdir -p ${IND_HOME}

CONF_FILE="${IND_HOME}/indispenso.yaml"

# This will start the server of indispenso and will generate a unique token the first time
if [ ! -f "${CONF_FILE}" ]; then
	echo "Server has no secure token yet, generating token"
	TOKEN=`openssl rand -base64 128 | tr -d '\n'`
	echo "Your token is:"
	echo "$TOKEN"

	# Update conf file
	touch ${CONF_FILE}
	cat ${CONF_FILE} | grep -v "secure_token" > ${CONF_FILE}.bak
	echo "token: $TOKEN" >> ${CONF_FILE}.bak
	echo "serverEnabled: true" >> ${CONF_FILE}.bak
	mv ${CONF_FILE}.bak ${CONF_FILE}
fi

# Start server
./indispenso "$@"
