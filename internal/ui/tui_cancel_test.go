package ui

import (
"context"
"sync"
"testing"
"time"

tea "github.com/charmbracelet/bubbletea"
"github.com/liup215/gline/internal/ui/bridge"
)

func TestCancelChConcurrentAccess(t *testing.T) {
	m := New(nil)
// ensure event channel and forwarding goroutine for tests to avoid blocking
m.eventCh = make(chan bridge.AgentEvent, 64)
m.done = make(chan struct{})
go func() {
for {
select {
case <-m.eventCh:
// drop events
case <-m.done:
return
}
}
}()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, cancel := context.WithCancel(context.Background())
			select {
			case m.cancelCh <- cancel:
			default:
				// consume old and replace
				select {
				case old := <-m.cancelCh:
					if old != nil {
						old()
					}
				default:
				}
				m.cancelCh <- cancel
			}
		}()
	}
	wg.Wait()
}

func TestEscInterruptAskFollowupQuestion(t *testing.T) {
	m := New(nil)
// ensure event channel and forwarding goroutine for tests to avoid blocking
m.eventCh = make(chan bridge.AgentEvent, 64)
m.done = make(chan struct{})
go func() {
for {
select {
case <-m.eventCh:
// drop events
case <-m.done:
return
}
}
}()

	// prepare pendingReply and simulate AskFollowupQuestion path
	reply := make(chan string, 1)
	m.pendingReply = reply
	m.isProcessing = true

	// simulate user pressing Esc: directly call handler
	cmds := handleKeyMsg(m, teaKeyEsc())
	// ensure pendingReply closed and cleared
	if m.pendingReply != nil {
		t.Fatalf("pendingReply should be nil after Esc, got non-nil")
	}

	// sending on closed channel should panic if not handled; reading should be closed
	_, ok := <-reply
	if ok {
		t.Fatalf("reply channel should be closed")
	}

	_ = cmds
}

func TestNoCancelFnDataRace(t *testing.T) {
m := New(nil)
// ensure event channel and forwarding goroutine for tests to avoid blocking
m.eventCh = make(chan bridge.AgentEvent, 64)
m.done = make(chan struct{})
go func() {
for {
select {
case <-m.eventCh:
// drop events
case <-m.done:
return
}
}
}()

// Goroutine 1: send cancel
go func() {
_, cancel := context.WithCancel(context.Background())
select {
case m.cancelCh <- cancel:
default:
select {
case old := <-m.cancelCh:
if old != nil {
old()
}
default:
}
m.cancelCh <- cancel
}
}()

	// Goroutine 2: receive and call cancel
	go func() {
		time.Sleep(10 * time.Millisecond)
		select {
		case c := <-m.cancelCh:
			if c != nil {
				c()
			}
		default:
		}
	}()

	time.Sleep(50 * time.Millisecond)
}

// helper to construct a tea.KeyMsg for Esc without importing tea in tests
func teaKeyEsc() tea.KeyMsg {
return tea.KeyMsg{Type: tea.KeyEsc}
}
