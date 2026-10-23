package events

import (
	"testing"
	"time"

	"github.com/pageguard/pageguard/internal/domain"
)

func TestHubFanOutAndCancel(t *testing.T) {
	h := NewHub()
	runID := domain.NewID()

	ch1, cancel1 := h.Subscribe(runID)
	defer cancel1()
	ch2, cancel2 := h.Subscribe(runID)
	defer cancel2()

	if h.SubscriberCount(runID) != 2 {
		t.Fatalf("subscriber count = %d", h.SubscriberCount(runID))
	}

	h.Publish(&domain.Event{RunID: runID, Type: domain.EvtRunStarted})
	h.Publish(&domain.Event{RunID: runID, Type: domain.EvtPageCompleted})

	for _, ch := range []<-chan *domain.Event{ch1, ch2} {
		select {
		case evt := <-ch:
			if evt.Type != domain.EvtRunStarted {
				t.Errorf("first event = %s", evt.Type)
			}
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for event")
		}
		if evt := <-ch; evt.Type != domain.EvtPageCompleted || evt.ID == 0 {
			t.Errorf("second event = %+v", evt)
		}
	}

	cancel1()
	if h.SubscriberCount(runID) != 1 {
		t.Errorf("after cancel count = %d", h.SubscriberCount(runID))
	}
}

func TestHubDoesNotDeliverToOtherRuns(t *testing.T) {
	h := NewHub()
	a := domain.NewID()
	b := domain.NewID()
	ch, cancel := h.Subscribe(a)
	defer cancel()

	h.Publish(&domain.Event{RunID: b, Type: domain.EvtRunStarted})
	select {
	case evt := <-ch:
		t.Fatalf("unexpected event for run a: %+v", evt)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestEmitter(t *testing.T) {
	h := NewHub()
	run := &domain.Run{ID: domain.NewID(), Status: domain.RunQueued}
	em := NewEmitter(h, nil, run)
	ch, cancel := h.Subscribe(run.ID)
	defer cancel()

	em.SetStatus(domain.RunRunning)
	em.IncDiscovered()
	em.IncScanned()
	em.IncFailed()
	em.IncFindings(2)
	em.Complete("")

	p := em.Progress()
	if p.Stats.PagesDiscovered != 1 || p.Stats.PagesScanned != 1 || p.Stats.PagesFailed != 1 || p.Stats.Findings != 2 {
		t.Errorf("stats = %+v", p.Stats)
	}
	if p.Status != domain.RunSucceeded {
		t.Errorf("status = %s", p.Status)
	}

	// Drain and confirm we saw a completion event.
	sawComplete := false
	for len(ch) > 0 {
		evt := <-ch
		if evt.Type == domain.EvtRunCompleted {
			sawComplete = true
		}
	}
	if !sawComplete {
		t.Error("missing run.completed event")
	}
}
