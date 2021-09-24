package gogrep

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"regexp"
	"sync"
)

type (
	// Grepper provides an interface for grep.
	Grepper interface {
		// Grep greps source by regex.
		// The results are not guaranteed to be in order in which lines appear.
		Grep(ctx context.Context, regex string, source io.Reader) (<-chan Result, error)
	}
	// Result is a result of Grep.
	Result interface {
		// Text returns the matched string.
		// It is valid when Err() returns nil.
		Text() string
		// Err returns an error that Grep got.
		Err() error
	}
	// Config provides Grepper configuration.
	Config struct {
		threads          int
		resultBufferSize int
	}
)

type grepper struct {
	config *Config
}

const (
	grepResultBufferSize = 1000
	grepChunkSize        = 100
	grepMaxGoroutines    = 4
)

func newConfig() *Config {
	return &Config{
		threads:          grepMaxGoroutines,
		resultBufferSize: grepResultBufferSize,
	}
}

// New returns a new Grepper.
func New(opt ...Option) Grepper {
	c := newConfig()
	for _, o := range opt {
		o(c)
	}
	return &grepper{
		config: c,
	}
}

func (s *grepper) Grep(ctx context.Context, regex string, source io.Reader) (<-chan Result, error) {
	// Already canceled
	if isDone(ctx) {
		return nil, wrapErr(ctx.Err(), "grepper")
	}
	// Check regex
	r, err := regexp.Compile(regex)
	if err != nil {
		return nil, wrapErr(err, "grepper cannot compile regex %s", regex)
	}
	// Launch workers that do grep strings
	var (
		wg       sync.WaitGroup
		requestC = make(chan []string, grepMaxGoroutines*2)
		resultC  = make(chan Result, s.config.resultBufferSize)
	)
	wg.Add(s.config.threads)
	for i := 0; i < s.config.threads; i++ {
		go func() {
			defer wg.Done()
			s.grep(requestC, resultC, r)
		}()
	}
	// Client worker
	go func() {
		var (
			iCtx, cancel = context.WithCancel(ctx)
			sc           = bufio.NewScanner(source)
			buf          []string
		)
		defer cancel()
		// Split input strings by chunk size
		for sc.Scan() {
			buf = append(buf, sc.Text())
			if len(buf) < grepChunkSize {
				continue
			}
			if isDone(iCtx) {
				// Cancel client
				break
			}
			requestC <- buf // Send data to workers
			buf = nil       // Reset buffer
		}
		if isDone(iCtx) {
			resultC <- newErrResult(iCtx.Err())
		} else if len(buf) > 0 {
			requestC <- buf
		}
		close(requestC) // Requests are exhausted
		wg.Wait()       // Results from workers are exhausted
		if err := sc.Err(); err != nil {
			resultC <- newErrResult(wrapErr(err, "grepper got error from scanner"))
		}
		close(resultC)
	}()
	return resultC, nil
}

// grep selects the strings that match with the regexp.
func (s *grepper) grep(requestC <-chan []string, resultC chan<- Result, r *regexp.Regexp) {
	for lines := range requestC {
		for _, line := range lines {
			if r.MatchString(line) {
				resultC <- newResult(line)
			}
		}
	}
}

type result struct {
	text string
	err  error
}

func newResult(text string) Result  { return &result{text: text} }
func newErrResult(err error) Result { return &result{err: err} }

func (s *result) Text() string { return s.text }
func (s *result) Err() error   { return s.err }

/* Utilities */

// isDone returns true if context has already canceled.
func isDone(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

// wrapErr wraps an error.
func wrapErr(err error, format string, v ...interface{}) error {
	return fmt.Errorf("%s %w", fmt.Sprintf(format, v...), err)
}

/* Options */

type (
	// Option provides Grepper configuration.
	Option func(*Config)
)

// WithThreads sets the number of grep workers.
// Not positive number is ignored.
func WithThreads(threads int) Option {
	return func(c *Config) {
		if threads > 0 {
			c.threads = threads
		}
	}
}

// WithResultBufferSize sets the buffer size of the result channel.
// Not positive number is ignored.
func WithResultBufferSize(resultBufferSize int) Option {
	return func(c *Config) {
		if resultBufferSize > 0 {
			c.resultBufferSize = resultBufferSize
		}
	}
}
