package main

// @author Robin Verlangen
// The execution stratey of a command

type ExecutionStrategyType int

type ExecutionStrategy struct {
	Strategy ExecutionStrategyType
}

func (e *ExecutionStrategy) Execute(c *ConsensusRequest) bool {
	// Template
	template := c.Template()

	// Get all clients
	for _, clientId := range c.ClientIds {
		// Get client
		client := server.GetClient(clientId)
		if client == nil {
			log.Printf("Client %s not found for request %s", clientId, c.Id)
			continue
		}

		// We do not check whether we have an auth token here so the client can pickup commands after registration

		// Create command instance
		cmd := newCmd(template.Command, template.Timeout)
		cmd.TemplateId = c.Template().Id
		cmd.ClientId = client.ClientId
		cmd.RequestUserId = c.RequestUserId
		cmd.Sign(client)

		// Start
		client.Submit(cmd)
	}

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
