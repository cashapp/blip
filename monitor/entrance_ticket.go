package monitor

import (
	"fmt"
	"sync"
)

// Creates entrance tickets to coordinate the execution
// of code based on the order of the ticket creation. Can
// we used to synchronize asynchronous routines in the
// order they were issued tickets.
type EntranceTicketFactory struct {
	tickets    []*EntranceTicket
	mx         sync.Mutex
	running    bool
	stopChan   chan struct{}
	signalChan chan struct{}
	doneChan   chan struct{}
}

// Controls the entrance into a section of code, based on the order
// the tickets were created.
type EntranceTicket struct {
	enterChan chan struct{}
	exitChan  chan struct{}
	mx        sync.Mutex
	done      bool
}

func NewEntranceTicketFactory() *EntranceTicketFactory {
	return &EntranceTicketFactory{
		doneChan:   nil,
		signalChan: make(chan struct{}, 1),
		stopChan:   make(chan struct{}),
	}
}

func (f *EntranceTicketFactory) Start() {
	f.mx.Lock()
	defer f.mx.Unlock()
	if !f.running {
		f.doneChan = make(chan struct{})
		f.running = true
		go f.monitor()
	}
}

func (f *EntranceTicketFactory) Stop() <-chan struct{} {
	f.mx.Lock()
	defer f.mx.Unlock()
	if f.running {
		close(f.stopChan)
		f.running = false
		return f.doneChan
	}

	return f.doneChan
}

func (f *EntranceTicketFactory) monitor() {
	for {
	WAIT_FOR_TICKET:
		select {
		case <-f.stopChan:
			goto EXIT
		case <-f.signalChan:
			for {
				f.mx.Lock()
				nextTicket := f.tickets[0]
				close(nextTicket.enterChan)
				f.mx.Unlock()
				select {
				case <-f.stopChan:
					goto EXIT
				case <-nextTicket.exitChan:
					f.mx.Lock()
					f.tickets = f.tickets[1:]

					if len(f.tickets) == 0 {
						f.mx.Unlock()
						goto WAIT_FOR_TICKET
					}

					f.mx.Unlock()
				}
			}
		}
	}

EXIT:
	f.mx.Lock()
	defer f.mx.Unlock()
	for _, ticket := range f.tickets {
		select {
		case <-ticket.enterChan:
		default:
			close(ticket.enterChan)
		}
	}

	f.tickets = []*EntranceTicket{}

	close(f.doneChan)
}

// Generates a new ticket. The ticket will allow entry into a section
// of code once all tickets that were issue prior to the current one
// have indicated that they have exited.
func (f *EntranceTicketFactory) NewTicket() (*EntranceTicket, error) {
	f.mx.Lock()
	defer f.mx.Unlock()
	if !f.running {
		return nil, fmt.Errorf("Factory not running")
	}

	newTicket := EntranceTicket{
		enterChan: make(chan struct{}),
		exitChan:  make(chan struct{}),
	}
	f.tickets = append(f.tickets, &newTicket)

	if len(f.tickets) == 1 {
		f.signalChan <- struct{}{}
	}

	return &newTicket, nil
}

// Returns a channel to wait on. When the code can be
// entered the channel will stop blocking
func (t *EntranceTicket) Enter() <-chan struct{} {
	return t.enterChan
}

// Signals that the ticket holder has exited the synchronized section of code
func (t *EntranceTicket) Exit() {
	t.mx.Lock()
	defer t.mx.Unlock()
	if !t.done {
		t.done = true
		close(t.exitChan)
	}
}
