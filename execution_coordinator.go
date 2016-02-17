package main

import (
	"math"
	"sync"
)

// This will coordinate the execution of strategies
// @author Robin Verlangen

type ExecutionCoordinator struct {
	Active map[string]*ExecutionCoordinatorEntry
	mux    sync.RWMutex
}

type ExecutionCoordinatorEntry struct {
	Id        string // Consensus request id
	cmds      []*PendingClientCmd
	strategy  *ExecutionStrategy
	iteration int // starts at 0, first started iteration will update this to 1
	mux       sync.RWMutex
}

type PendingClientCmd struct {
	Client *RegisteredClient
	Cmd    *Cmd
}

// Execute the callbacks if the entire list of commands is
func (ece *ExecutionCoordinatorEntry) ExecuteCallbacks() {
	cr := server.consensus.Get(ece.Id)
	for _, cb := range cr.Callbacks {
		go cb(cr)
	}
}

// Called after a command has finished, see if there is more work to start
func (ece *ExecutionCoordinatorEntry) Next() {
	if conf.Debug {
		log.Println("Next")
	}

	// Lock
	ece.mux.Lock()
	defer ece.mux.Unlock()

	// Is all work from this batch done?
	var allFinished bool = true
	if conf.Debug {
		log.Printf("Current batch %d", ece.iteration)
	}

	// Iterate
	server.clientsMux.RLock()
outer:
	for _, client := range server.clients {
		client.mux.RLock()
		defer client.mux.RUnlock()
		for _, cmd := range client.DispatchedCmds {
			if cmd.ConsensusRequestId == ece.Id && cmd.ExecutionIterationId == ece.iteration {
				if conf.Debug {
					log.Printf("%s was started in the previous iteration %v", cmd.Id, cmd)
				}
				if cmd.State != "finished" {
					allFinished = false
					break outer
				}
			}
		}
	}
	server.clientsMux.RUnlock()

	// Done? Do we have any work left?
	if len(ece.cmds) == 0 {
		if allFinished {
			// All is done, execute the callbacks
			ece.ExecuteCallbacks()
		}
		if conf.Debug {
			log.Printf("No additional work to start for consensus request %s", ece.Id)
		}
		return
	}

	// Can we start something new?
	if allFinished == false {
		if conf.Debug {
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
		// One then the rest
		if ece.iteration == 0 {
			// One to start
			cmdsToStart = 1
		} else {
			// The rest
			cmdsToStart = len(ece.cmds)
		}
		break

	case RollingExecutionStrategy:
		// One by one
		cmdsToStart = 1
		break

	case ExponentialRollingExecutionStrategy:
		// 1, 2, 4, 8, 16, 32 etc
		cmdsToStart = int(math.Pow(2, float64(ece.iteration)))
		if cmdsToStart > len(ece.cmds) {
			cmdsToStart = len(ece.cmds)
		}
		break

	default:
		panic("Not yet supported")
	}

	// Start command(s)
	if conf.Debug {
		log.Printf("Starting %d cmds for consensus request %s", cmdsToStart, ece.Id)
	}
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
