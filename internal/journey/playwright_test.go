package journey

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func fakeFactory(t *testing.T) *PlaywrightSessionFactory {
	t.Helper()
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not installed")
	}
	return NewPlaywrightSessionFactory("node", filepath.Join("testdata", "fake-worker.mjs"))
}

func TestPlaywrightSessionAgainstFakeWorker(t *testing.T) {
	f := fakeFactory(t)
	ctx := context.Background()

	sess, err := f.Open(ctx, SessionOptions{})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer sess.Close()

	st, err := sess.Perform(ctx, Step{Action: ActionNavigate, Value: "https://acme.test/"})
	if err != nil {
		t.Fatalf("Perform navigate: %v", err)
	}
	if st.Status != 200 {
		t.Errorf("status = %d", st.Status)
	}
	if st.Title != "Fake 1" {
		t.Errorf("title = %q", st.Title)
	}
	events := dataLayerEvents(st.DataLayer)
	if len(events) != 1 || events[0] != "page_view" {
		t.Errorf("events = %v", events)
	}
	if len(st.Cookies) != 1 || st.Cookies[0].Name != "sid" {
		t.Errorf("cookies = %+v", st.Cookies)
	}

	st2, err := sess.Perform(ctx, Step{Action: ActionClick, Selector: "#buy"})
	if err != nil {
		t.Fatalf("Perform click: %v", err)
	}
	if !strings.Contains(st2.FinalURL, "/product/1") {
		t.Errorf("finalURL = %q", st2.FinalURL)
	}
	if ev := dataLayerEvents(st2.DataLayer); len(ev) != 2 {
		t.Errorf("events after click = %v", ev)
	}

	png, err := sess.Screenshot(ctx)
	if err != nil {
		t.Fatalf("Screenshot: %v", err)
	}
	if string(png) != "PNG" {
		t.Errorf("screenshot = %q", png)
	}
}

func TestPlaywrightWorkerErrorPropagates(t *testing.T) {
	f := fakeFactory(t)
	ctx := context.Background()
	sess, err := f.Open(ctx, SessionOptions{})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer sess.Close()

	_, err = sess.Perform(ctx, Step{Action: "explode"})
	if err == nil {
		t.Fatal("expected worker error")
	}
	if !strings.Contains(err.Error(), "simulated step failure") {
		t.Errorf("error = %v", err)
	}
}

func TestPlaywrightMissingCommand(t *testing.T) {
	f := NewPlaywrightSessionFactory()
	if _, err := f.Open(context.Background(), SessionOptions{}); err == nil {
		t.Fatal("expected error for empty command")
	}
}
