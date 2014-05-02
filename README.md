indispenso
========

Distribute, manage, regulate, arrange. Simple &amp; secure management based on consensus.

## Building
This project requires Go 1.2 or later to compile. 

	$ go get github.com/RobinUS2/indispenso
	$ go test
	$ go build

If this completes without errors you have a indispenso binary.

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
- Manage and issue commands across cluster of servers
- Restart a service on production cluster of servers if two or more developers agree
- Aprove a push or change to production servers by IT management
- Allow for a limited, template based management of servers and code to non-technical people
- Role based server management, eg. interns can only restart services, but cannot install new software
- Mobile interface for common server cluster tasks
- Be able to fix things while on the go, without having to use SSH
- Never retype any (bash)commands
- Never make costly mistakes by using wrong parameters
- All changes are logged and available for audit immediately, without having to consolidate and filter logs from multiple servers

## Status
Project development has recently started. Goals are being drafted and background is explained.
