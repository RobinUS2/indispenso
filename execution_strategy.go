package main

// @author Robin Verlangen
// The execution stratey of a command

type ExecutionStrategyType int

type ExecutionStrategy struct {
	Strategy ExecutionStrategyType
}

type ClientCmd struct {
	Client *RegisteredClient
	Cmd    *Cmd
}

// Execute a request
func (e *ExecutionStrategy) Execute(c *ConsensusRequest) bool {
	// Template
	template := c.Template()

	// Create list of commands for clients
	var clientCmds []*ClientCmd = make([]*ClientCmd, 0)

	// Assemble commands
	for _, clientId := range c.ClientIds {
		// Get client
		client := server.GetClient(clientId)
		if client == nil {
			log.Printf("Client %s not found for request %s", clientId, c.Id)
			continue
		}

		// Create command instance
		cmd := newCmd(template.Command, template.Timeout)
		cmd.ConsensusRequestId = c.Id
		cmd.TemplateId = c.Template().Id
		cmd.ClientId = client.ClientId
		cmd.RequestUserId = c.RequestUserId
		cmd.Sign(client)
		clientCmd := &ClientCmd{
			Client: client,
			Cmd:    cmd,
		}

		// Add to list
		clientCmds = append(clientCmds, clientCmd)
	}

	// Based on strategy
	var res bool = false
	switch e.Strategy {
	case SimpleExecutionStrategy:
		res = e._executeSimple(c, clientCmds)
		break

	case OneTestExecutionStrategy:
	case RollingUpgradeExecutionStrategy:
	case ExponentialRollingUpgradeExecutionStrategy:
		// @todo register with execution coordinator
		res = e._executePhased(c, clientCmds)
		break
	default:
		panic("Not supported")
	}
	return res
}

// Simple: all at once
func (e *ExecutionStrategy) _executeSimple(c *ConsensusRequest, cmds []*ClientCmd) bool {
	// Get all clients
	for _, cmd := range cmds {
		// Start
		cmd.Client.Submit(cmd.Cmd)
	}

	// Done
	return true
}

// This will all start with a single one, and then perform more on completion
func (e *ExecutionStrategy) _executePhased(c *ConsensusRequest, cmds []*ClientCmd) bool {
	// Start one
	log.Println("one test")
	cmds[0].Client.Submit(cmds[0].Cmd)

	// Done
	return true
}

const (
	SimpleExecutionStrategy ExecutionStrategyType = iota
	OneTestExecutionStrategy
	RollingUpgradeExecutionStrategy
	ExponentialRollingUpgradeExecutionStrategy
)

func newExecutionStrategy(strategy ExecutionStrategyType) *ExecutionStrategy {
	return &ExecutionStrategy{
		Strategy: strategy,
	}
}
