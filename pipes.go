package script

import (
	"bytes"
	"io"
	"regexp"
	"strconv"
)

// Pipe represents a pipe object with an associated ReadAutoCloser.
type Pipe struct {
	Reader ReadAutoCloser
	err    error
	async  bool
}

// NewPipe returns a pointer to a new empty pipe, with streaming turned off.
func NewPipe() *Pipe {
	return &Pipe{ReadAutoCloser{}, nil, false}
}

// Close closes the pipe's associated reader. This is always safe to do, because
// pipes created from a non-closable source will have an `ioutil.NopCloser` to
// call.
func (p *Pipe) Close() error {
	if p == nil {
		return nil
	}
	return p.Reader.Close()
}

// Error returns the last error returned by any pipe operation, or nil otherwise.
func (p *Pipe) Error() error {
	if p == nil {
		return nil
	}
	return p.err
}

var exitStatusPattern = regexp.MustCompile(`exit status (\d+)$`)

// ExitStatus returns the integer exit status of a previous command, if the
// pipe's error status is set, and if the error matches the pattern "exit status
// %d". Otherwise, it returns zero.
func (p *Pipe) ExitStatus() int {
	perr := p.Error()
	if perr == nil {
		return 0
	}
	match := exitStatusPattern.FindStringSubmatch(perr.Error())
	if len(match) < 2 {
		return 0
	}
	status, err := strconv.Atoi(match[1])
	if err != nil {
		// This seems unlikely, but...
		return 0
	}
	return status
}

// Read reads up to len(b) bytes from the data source into b. It returns the
// number of bytes read and any error encountered. At end of file, or on a nil
// pipe, Read returns 0, io.EOF.
func (p *Pipe) Read(b []byte) (int, error) {
	if p == nil {
		return 0, io.EOF
	}
	return p.Reader.Read(b)
}

// SetError sets the pipe's error status to the specified error.
func (p *Pipe) SetError(err error) {
	if p != nil {
		p.err = err
	}
}

// WithReader takes an io.Reader, and associates the pipe with that reader. If
// necessary, the reader will be automatically closed once it has been
// completely read.
func (p *Pipe) WithReader(r io.Reader) *Pipe {
	if p == nil {
		return nil
	}
	p.Reader = NewReadAutoCloser(r)
	return p
}

// WithError sets the pipe's error status to the specified error and returns the
// modified pipe.
func (p *Pipe) WithError(err error) *Pipe {
	p.SetError(err)
	return p
}

// WithError sets the pipe's error status to the specified error and returns the
// modified pipe.
func (p *Pipe) withAsync(async bool) *Pipe {
	p.async = async
	return p
}

// Stream lets the pipeline after it be executed in the streaming mode.
// Like a Unix pipe, functions start simultaneously, each consuming the
// output of its predecessor instantly. Function will stop when it detects
// an error either in itself or from its predecessor. One can call
// Synchronize() or any sink function to disable the streaming.
func Stream() *Pipe {
	return NewPipe().withAsync(true)
}

// Stream lets the pipeline after it be executed in the streaming mode.
// Like a Unix pipe, functions start simultaneously, each consuming the
// output of its predecessor instantly. Function will stop when it detects
// an error either in itself or from its predecessor. One can call
// Synchronize() or any sink function to disable the streaming.
func (p *Pipe) Stream() *Pipe {
	if p == nil {
		return p
	}
	return p.withAsync(true)
}

// Synchronize turns off streaming mode
func (p *Pipe) Synchronize() *Pipe {
	if p == nil || p.Error() != nil || !p.async {
		return p
	}
	w := bytes.Buffer{}
	io.Copy(&w, p.Reader)
	return NewPipe().WithReader(bytes.NewReader(w.Bytes())).withAsync(false)
}
