package monitor

import (
	"sync"
	"testing"

	"github.com/go-test/deep"
)

func TestEntranceTicket(t *testing.T) {
	values := []int{}
	expectedValues := []int{0, 1, 2, 3}
	tickets := []*EntranceTicket{}
	var wg sync.WaitGroup
	factory := NewEntranceTicketFactory()
	factory.Start()
	defer factory.Stop()

	for i := 0; i < len(expectedValues); i++ {
		ticket, err := factory.NewTicket()
		if err != nil {
			t.Fatal(err)
		}

		tickets = append(tickets, ticket)
	}

	for i := 3; i >= 0; i-- {
		wg.Add(1)
		go func(value int, ti *EntranceTicket) {
			defer wg.Done()
			<-ti.Enter()
			values = append(values, value)

			ti.Exit()
		}(i, tickets[i])
	}

	wg.Wait()

	if diff := deep.Equal(expectedValues, values); diff != nil {
		t.Error(diff)
	}
}
