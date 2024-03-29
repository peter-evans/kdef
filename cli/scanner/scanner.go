// Package scanner implements a scanner with methods to prompt for user input.
package scanner

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// New creates a new scanner.
func New() *Scanner {
	s := &Scanner{
		s: bufio.NewScanner(os.Stdin),
	}
	s.cond = sync.NewCond(&s.mu)
	go s.scan()
	return s
}

// Scanner represents an active scanner.
type Scanner struct {
	s     *bufio.Scanner
	mu    sync.Mutex
	cond  *sync.Cond
	lines []string
}

// PromptLine prompts the user for a single line of input.
func (s *Scanner) PromptLine(prompt string, defaultValue string) string {
	if l := s.line(prompt); len(l) > 0 {
		return l
	}
	return defaultValue
}

// PromptMultiline prompts the user for multiline input.
func (s *Scanner) PromptMultiline(prompt string, defaultValue []string) []string {
	var lines []string
	for {
		l := s.line(prompt)
		if len(l) == 0 {
			break
		}

		for _, addr := range strings.Split(l, ",") {
			addr = strings.TrimSpace(addr)
			if len(addr) == 0 {
				continue
			}
			lines = append(lines, addr)
		}
	}

	if len(lines) > 0 {
		return lines
	}
	return defaultValue
}

// PromptYesNo prompts the user for yes/no input.
func (s *Scanner) PromptYesNo(prompt string, defaultValue bool) bool {
	if l := s.line(prompt); len(l) > 0 {
		switch l {
		case "Yes", "yes", "Y", "y":
			return true
		default:
			return false
		}
	} else {
		return defaultValue
	}
}

func (s *Scanner) scan() {
	last := time.Now()
	for s.s.Scan() {
		line := s.s.Text()
		if len(line) == 0 && time.Since(last) < 50*time.Millisecond {
			last = time.Now()
			continue
		}
		last = time.Now()

		s.mu.Lock()
		s.lines = append(s.lines, line)
		if len(s.lines) > 10 {
			exit("too much unhandled input, exiting")
		}
		s.mu.Unlock()
		s.cond.Broadcast()
	}
	if s.s.Err() != nil {
		exit("scanner error: %v, exiting", s.s.Err())
	}
	exit("scanner received EOF, exiting")
}

func exit(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg+"\n", args...)
	os.Exit(1)
}

func (s *Scanner) line(prompt string) string {
	fmt.Printf("%s ", prompt)

	done := make(chan struct{})
	var line string
	go func() {
		defer close(done)
		s.mu.Lock()
		defer s.mu.Unlock()

		for len(s.lines) == 0 {
			s.cond.Wait()
		}
		line = s.lines[0]
		s.lines = s.lines[1:]
	}()

	<-done
	return line
}
