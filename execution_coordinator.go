package main

import (
	"sync"
)

// This will coordinate the execution of strategies
// @author Robin Verlangen

type ExecutionCoordinator struct {
	Active map[string]*ExecutionCoordinatorEntry
	mux    sync.RWMutex
}

type ExecutionCoordinatorEntry struct {
	Id        string
	cmds      []*PendingClientCmd
	strategy  *ExecutionStrategy
	iteration int // starts at 0, first started iteration will update this to 1
	mux       sync.RWMutex
}

type PendingClientCmd struct {
	Client *RegisteredClient
	Cmd    *Cmd
}

func (ece *ExecutionCoordinatorEntry) Next() {
	log.Println("Next")
	// Lock
	ece.mux.Lock()
	defer ece.mux.Unlock()

	// Done? Do we have any work left?
	if len(ece.cmds) == 0 {
		log.Printf("Execution for consensus request %s is done, no more work", ece.Id)
		return
	}

	// Is all work from this batch done?
	if debug {
		log.Printf("Current batch %d", ece.iteration)
	}
	server.clientsMux.RLock()
	var allFinished bool = true
	for _, client := range server.clients {
		for _, cmd := range client.DispatchedCmds {
			if cmd.ConsensusRequestId == ece.Id && cmd.ExecutionIterationId == ece.iteration {
				if debug {
					log.Printf("%s was started in the previous iteration %v", cmd.Id, cmd)
					if cmd.State != "finisehd" {
						allFinished = false
						break
					}
				}
			}
		}
	}
	server.clientsMux.RUnlock()
	if allFinished == false {
		if debug {
			log.Printf("Still work being executed for request %s", ece.Id)
		}
		return
	}

	// How many will we start?
	var cmdsToStart = 0
	switch ece.strategy.Strategy {
	case SimpleExecutionStrategy:
		// All at once
		cmdsToStart = len(ece.cmds)
		break
	case OneTestExecutionStrategy:
		if ece.iteration == 0 {
			// One to start
			cmdsToStart = 1
		} else {
			// The rest
			cmdsToStart = len(ece.cmds)
		}
	default:
		panic("Not yet supported")
	}

	// Start command(s)
	for i := 0; i < cmdsToStart; i++ {
		// Get element
		var cmd PendingClientCmd = *ece.cmds[len(ece.cmds)-1]

		go func(cmd PendingClientCmd) {
			// Submit to client
			log.Printf("Starting cmd %s for consensus request %s", cmd.Cmd.Id, ece.Id)

			c := *cmd.Cmd
			c.ExecutionIterationId = ece.iteration
			cmd.Client.Submit(&c)
		}(cmd)

		// Remove element
		ece.cmds = ece.cmds[:len(ece.cmds)-1]
	}

	// Increment iteration counter
	ece.iteration++
}

func (e *ExecutionCoordinator) Get(consensusRequestId string) *ExecutionCoordinatorEntry {
	e.mux.RLock()
	defer e.mux.RUnlock()
	return e.Active[consensusRequestId]
}

func (e *ExecutionCoordinator) Add(consensusRequestId string, strategy *ExecutionStrategy, cmds []*PendingClientCmd) {
	e.mux.Lock()
	defer e.mux.Unlock()
	entry := newExecutionCoordinatorEntry()
	entry.Id = consensusRequestId
	entry.cmds = cmds
	entry.strategy = strategy
	e.Active[consensusRequestId] = entry
}

func newExecutionCoordinator() *ExecutionCoordinator {
	return &ExecutionCoordinator{
		Active: make(map[string]*ExecutionCoordinatorEntry),
	}
}

func newExecutionCoordinatorEntry() *ExecutionCoordinatorEntry {
	return &ExecutionCoordinatorEntry{}
}
