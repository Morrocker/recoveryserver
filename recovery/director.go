package recovery

import (
	"sync"
	"time"

	"github.com/clonercl/blockserver/blocks"
	"github.com/recoveryserver/utils"
)

// Director orders and decides which recoveries should be executed next
type Director struct {
	Run bool

	Order         int
	LegacyStores  []blocks.MasterStore
	CurrentStores []blocks.MasterStore
	Recoveries    map[string]*Recovery
	Lock          sync.Mutex
}

// StartWorkers continually tries to start workers for each recovery added
func (d *Director) StartWorkers() {
	for {
		d.Lock.Lock()
		for key, recover := range d.Recoveries {
			if recover.Status == Queue {
				go d.Recoveries[key].Run()
			}
		}
		d.Lock.Unlock()
		time.Sleep(10 * time.Second)
	}
}

// AddRecovery adds the given recovery data to create a new entry on the Recoveries map
func (d *Director) AddRecovery(r Data) (hash string) {
	hash = utils.RandString(8)
	d.Recoveries[hash] = &Recovery{Info: r}
	return
}

// Stop sets Run to false
func (d *Director) Stop() {
	d.Run = false
}

// Start sets Run to true
func (d *Director) Start() {
	d.Run = true
}
