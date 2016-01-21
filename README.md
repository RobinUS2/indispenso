indispenso [![Build Status](https://travis-ci.org/RobinUS2/indispenso.svg?branch=master)](https://travis-ci.org/RobinUS2/indispenso)
========

Distribute, manage, regulate, arrange. Simple &amp; secure management based on consensus.

## Building
This project requires Go 1.2 or later to compile. 

	$ go get github.com/RobinUS2/indispenso
	$ go test
	$ go build

If this completes without errors you have a indispenso binary.

## Configuring

### Compatibility

If you are using previous version of indispenso you need to change configurations. For reference see tables below:

Flags:

 New version  | Old version
------------- | -------------
 - | auto-tag
 - | server-port
 - | client-port
config  (c) | -
serverEnabled (s)  | disable-server (use oposite value)
debug (d) | debug
home (p) | -
endpointUri (e) | seed
token (t) | -
hostname (i) | hostname
help (h) | -

Configuration :

 New version  | Old version   | Backward compatible<br />(auto translate)
------------- | ------------- | :---------------------:
 token  | secure_token | YES
 hostname | - | NO
 useAutoTag | - | NO
 tagsList | tags | NO
 serverEnabled | server_enabled | YES
 endpointURI | seed | YES
 serverPort | - | NO
 sslCertFile | cert_file | NO
 sslPrivateKeyFile | private_key_file | NO
 autoGenerateCert | auto_generate_cert | NO
 clientPort | - | NO
 debug | - | NO


### Home directory

Home directory is location of all indispenso configuration files. By default is located in ```/etc/indispenso```
If you want to change it, you can use environmental variable named ```$IND_HOME``` or by passing command line parameter:
 
    $ indispenso -p /home/user
    
or
    
    $ indispenso --home="/home/user"


### Flags

You can run indispenso with set of flags that configure application, below:

    $ ./indispenso -h
    Usage of indispenso:
      -c, --config="": Config file location default is /etc/indispenso/indispenso.{json,toml,yaml,yml,properties,props,prop}
      -d, --debug[=false]: Enable debug mode
      -e, --endpointUri="": URI of server interface, used by client
      -h, --help[=false]: Print help message
      -p, --home="/etc/indispenso/": Home directory where all config files are located
      -i, --hostname="localhost": Hostname that is use to identify itself
      -s, --serverEnabled[=false]: Define if server module should be started or not
      -t, --token="": Secret token


### SSL configuration

Indispenso serves WebUI using SSL secure connection, so it needs private key and certificate pair. 
It will automatically generate self signed certificate during startup if it not present, 
if you want generate own self signed certificate, this code snippet will be useful:

    $ openssl genrsa -out key.pem 2048
    $ openssl req -new -x509 -key key.pem -out cert.pem -days 365 -subj "/C=NL/ST=Indispenso/L=Indispenso/O=Indispenso/OU=IT/CN=ssl.indispenso.org"

This certificate and private key should be located in home directory of indispenso. Names of this files should be ```key.pem``` 
for private key and ```cert.pem``` for certificate file. You can change required filenames in main config file.

## Running
By default this application is running in client only mode, to enable server mode use -s/--serverEnable flag.

This application is designed with minimal setup and maintenance in mind. All you need is one or multiple seed nodes, and a shared secret.

	$ ./indispenso -e "indispenso-seed.my-company.com" -t "my_secret_of_at_least_32_characters"

In order to ensure consistency across nodes this system relies on the system clock. Make sure you install something like [_ntpd_](http://en.wikipedia.org/wiki/Ntpd) to keep your clocks in sync.

To get help just use one of this commands:
    
    $ indispenso -h
    
or
    
    $ indispenso --help
    

## Goals
- Easy management of servers, applications and infrastructure
- Secure access and granular permission control
- Decentralized and simple deployment
- One single binary that contains all functionality
- Simple job template management
- Consensus of people can start any job

## Background
This project is developed as there are a lot of companies that rely on a handful of people to manage critical infrastructure.
Most companies who run critical infrastructure dislike the [_Bus factor_](http://en.wikipedia.org/wiki/Bus_factor).
This is where indispenso comes in and allows people to take actions based upon consenus. 

One can for example reboot a service on a server based on a pre-defined template.

## Implementation
There are 3 key user roles within indispenso:

| Name | Manage templates | Request job | Approve request |
|------|------------------|-------------|-----------------|
| Administrator | x | x | x |
| Requester |  | x | x |
| Approver |  |  | x |

## Example use cases
- Manage and issue commands across cluster(s) of servers
- Restart a service on production cluster of servers if two or more developers agree
- Approve a push or change to production servers by IT management
- Allow for a limited, template based management of servers and code to non-technical people
- Role based server management, eg. interns can only restart services, but cannot install new software
- Mobile interface for common server cluster tasks
- Be able to fix things while on the go, without having to use SSH
- Never retype any (bash)commands
- Never make costly mistakes by using wrong parameters
- All changes are logged and available for audit immediately, without having to consolidate and filter logs from multiple servers
- Time based access to servers; after a specified time, access is revoked for an intern or a freelancer
- Overview of who has access to which servers
- Access to servers by two factor authentication, without adding new private keys or modifing configuration
- Issue commands on staging, check if the results are desired and then replay the commands on production

## Status
Project development has recently started. Goals are being drafted and background is explained.
