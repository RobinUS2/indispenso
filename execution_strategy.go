package main

// @author Robin Verlangen
// The execution stratey of a command

type ExecutionStrategyType int

type ExecutionStrategy struct {
	Strategy ExecutionStrategyType
}

// Execute a request
func (e *ExecutionStrategy) Execute(c *ConsensusRequest) bool {
	// Template
	template := c.Template()

	// Create list of commands for clients
	var clientCmds []*PendingClientCmd = make([]*PendingClientCmd, 0)

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
		clientCmd := &PendingClientCmd{
			Client: client,
			Cmd:    cmd,
		}

		// Add to list
		clientCmds = append(clientCmds, clientCmd)
	}

	// Register with execution coordinator
	server.executionCoordinator.Add(c.Id, e, clientCmds)

	// Next step
	server.executionCoordinator.Get(c.Id).Next()

	return true
}

const (
	SimpleExecutionStrategy             ExecutionStrategyType = iota // 0
	OneTestExecutionStrategy                                         // 1
	RollingExecutionStrategy                                         // 2
	ExponentialRollingExecutionStrategy                              // 3
)

func newExecutionStrategy(strategy ExecutionStrategyType) *ExecutionStrategy {
	return &ExecutionStrategy{
		Strategy: strategy,
	}
}
