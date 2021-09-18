package main

import (
	"context"
	"fmt"
	"sync"
)

// dataInfo struct sent over secondary channel
type dataInfo struct {
	FailMsg bool
	Count   int
}

// data struct sent over secondary channel
type data struct {
	Name string
	Msg  string
}

// Configuration main struct that is having operations
// based off its data
type Configuration struct {
	When   string
	Fail   string
	Count  int
	Ack    chan data
	Update chan dataInfo
}

// main this creates a pool of workers that are a go routine for
// processing a configuration. The configuration has two secondary
// channels that are used for processing state information of the
// main configuration struct.
func main() {
	ackCh := make(chan data)
	updateCh := make(chan dataInfo)
	ctx, cancel := context.WithTimeout(context.Background(), 10)
	defer cancel()

	config := &Configuration{
		When:   "now",
		Fail:   "never",
		Count:  1,
		Ack:    ackCh,
		Update: updateCh,
	}

	errConfig := &Configuration{
		When:   "issue",
		Fail:   "now",
		Count:  1,
		Ack:    ackCh,
		Update: updateCh,
	}

	fmt.Print("Starting go routine\n")
	go config.handleState(ctx, config, ackCh, updateCh)

	var wg sync.WaitGroup
	reboots := make(chan Configuration, 4)
	for workerID := 1; workerID <= 5; workerID++ {
		fmt.Printf("Creating worker: %d\n", workerID)
		wg.Add(1)
		go config.TriggerAck(workerID, reboots, &wg)
	}

	// Induce errros by changing the values here
	for i := 0; i < 5; i++ {
		if i == 0 || i == 2 {
			reboots <- *errConfig
		} else {
			reboots <- *config
		}
	}
	close(reboots)
	wg.Wait()
}

// TriggerAck this is where you are do the processing of the main channels data
// calling the secondary channels if there is something that requires a change
// in state
func (c *Configuration) TriggerAck(workId int, configs <-chan Configuration, wg *sync.WaitGroup) {
	defer wg.Done()

	for r := range configs {
		if r.When == "issue" {
			r.Update <- dataInfo{
				FailMsg: true,
				Count:   workId,
			}
		} else {
			r.Ack <- data{
				Name: "data",
				Msg:  "nope",
			}
		}
	}
}

// handleState is used to handle the State of the process, this is where you are listing for data going into
// the various channels that you are responsible for. Secondary channels for handling information
func (c *Configuration) handleState(ctx context.Context, config *Configuration, ackCh chan data, updateCh chan dataInfo) {
	for {
		select {
		case <-ackCh:
			fmt.Printf("\nNew Count: [%d]", config.Count)
			config.Count += 1
		case <-updateCh:
			config.Fail = "updateCh"
			fmt.Printf("\nFailure/WorkerId: [%s]/[%d] ", config.Fail, config.Count)
			<-ctx.Done()
		}
	}
}
