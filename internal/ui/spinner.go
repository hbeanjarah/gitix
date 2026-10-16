package ui

import (
	"fmt"
	"io"
	"runtime"
	"sync"
	"time"
)

// frames animate the spinner. The legacy Windows console has no braille glyphs.
var frames = func() []string {
	if runtime.GOOS == "windows" {
		return []string{"|", "/", "-", "\\"}
	}
	return []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
}()

// Spinner animates one status line ("⠹ Fetching origin…") until Stop erases it.
// It is meant for terminals: callers skip it when the output is not one.
type Spinner struct {
	w     io.Writer
	paint Painter
	now   func() time.Time
	start time.Time
	mu    sync.Mutex
	text  string
	frame int
	done  chan struct{}
	wg    sync.WaitGroup
}

// StartSpinner draws text right away, then animates it until Stop.
func StartSpinner(w io.Writer, paint Painter, text string) *Spinner {
	s := startSpinner(w, paint, text, time.Now)
	s.wg.Add(1)
	go s.animate()
	return s
}

func startSpinner(w io.Writer, paint Painter, text string, now func() time.Time) *Spinner {
	s := &Spinner{w: w, paint: paint, now: now, start: now(), text: text, done: make(chan struct{})}
	s.draw()
	return s
}

// Set replaces the text and redraws at once, e.g. "Analyzing branches  18/42".
func (s *Spinner) Set(text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.text = text
	s.drawLocked()
}

// Stop ends the animation and erases the line.
func (s *Spinner) Stop() {
	close(s.done)
	s.wg.Wait()
	s.mu.Lock()
	defer s.mu.Unlock()
	fmt.Fprint(s.w, "\r\x1b[2K")
}

func (s *Spinner) animate() {
	defer s.wg.Done()
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-s.done:
			return
		case <-tick.C:
			s.mu.Lock()
			s.frame = (s.frame + 1) % len(frames)
			s.drawLocked()
			s.mu.Unlock()
		}
	}
}

func (s *Spinner) draw() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.drawLocked()
}

// drawLocked rewrites the line; past 3 seconds it adds the elapsed time.
func (s *Spinner) drawLocked() {
	line := s.paint.Cyan(frames[s.frame]) + " " + s.text
	if elapsed := s.now().Sub(s.start); elapsed >= 3*time.Second {
		line += s.paint.Dim(fmt.Sprintf(" %ds", int(elapsed.Seconds())))
	}
	fmt.Fprint(s.w, "\r\x1b[2K"+line)
}
