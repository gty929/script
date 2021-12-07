package script

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"
)

// Some tests require monkeying with stdout. Make this concurrency-safe.
var stdoutM sync.Mutex

func TestWithReader(t *testing.T) {
	t.Parallel()
	want := "Hello, world."
	p := NewPipe().WithReader(strings.NewReader(want))
	got, err := p.String()
	if err != nil {
		t.Error(err)
	}
	if got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestError(t *testing.T) {
	t.Parallel()
	p := File("testdata/nonexistent.txt")
	if p.Error() == nil {
		t.Error("want error status reading nonexistent file, but got nil")
	}
	defer func() {
		// Reading an erroneous pipe should not panic.
		if r := recover(); r != nil {
			t.Errorf("panic reading erroneous pipe: %v", r)
		}
	}()
	_, err := p.String()
	if err != p.Error() {
		t.Error(err)
	}
	_, err = p.CountLines()
	if err != p.Error() {
		t.Error(err)
	}
	e := errors.New("fake error")
	p.SetError(e)
	if p.Error() != e {
		t.Errorf("want %v when setting pipe error, got %v", e, p.Error())
	}
}

func TestExitStatus(t *testing.T) {
	t.Parallel()
	tcs := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"bogus", 0},
		{"exit status bogus", 0},
		{"exit status 127", 127},
		{"exit status 1", 1},
		{"exit status 0", 0},
		{"exit status 1 followed by junk", 0},
	}
	for _, tc := range tcs {
		p := NewPipe()
		p.SetError(fmt.Errorf(tc.input))
		got := p.ExitStatus()
		if got != tc.want {
			t.Errorf("input %q: want %d, got %d", tc.input, tc.want, got)
		}
	}
	got := NewPipe().ExitStatus()
	if got != 0 {
		t.Errorf("want 0, got %d", got)
	}
}

func TestStream(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	n := 0
	round := 5
	plusOneAndDoubleLine := func(s string, b *strings.Builder) {
		time.Sleep(10 * time.Millisecond)
		mu.Lock()
		n++
		mu.Unlock()
		b.WriteRune('\n')
		b.WriteRune('\n')
	}
	timesTwo := func(s string, b *strings.Builder) {
		mu.Lock()
		n *= 2
		mu.Unlock()
		b.WriteRune('\n')
	}
	err := Stream().Exec("bash -c 'yes 1 | head -n 5; wait'").EachLine(plusOneAndDoubleLine).EachLine(timesTwo).Wait()
	if err != nil {
		t.Errorf("Got unexpected error %q", err)
	}
	want := 0
	for i := 0; i < round; i++ {
		want++
		want *= 4
	}
	if n != want {
		t.Errorf("want n = %d, got %d", want, n)
	}
}

func TestStreamFilter(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	n := 0
	round := 5
	plusOneAndDoubleLine := func(s string, b *strings.Builder) {
		time.Sleep(10 * time.Millisecond)
		mu.Lock()
		n++
		mu.Unlock()
		b.WriteRune('\n')
		b.WriteRune('\n')
	}
	timesTwo := func(s string, b *strings.Builder) {
		mu.Lock()
		n *= 2
		mu.Unlock()
		b.WriteRune('\n')
	}
	Slice(make([]string, round)).Stream().EachLine(plusOneAndDoubleLine).EachLine(timesTwo).Wait()
	want := 0
	for i := 0; i < round; i++ {
		want++
		want *= 4
	}
	if n != want {
		t.Errorf("want n = %d, got %d", want, n)
	}
}

func TestSynchronize(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	n := 0
	round := 5
	plusOne := func(s string, b *strings.Builder) {
		time.Sleep(10 * time.Millisecond)
		mu.Lock()
		n++
		mu.Unlock()
		b.WriteRune('\n')
	}
	timesTwo := func(s string, b *strings.Builder) {
		mu.Lock()
		n *= 2
		mu.Unlock()
		b.WriteRune('\n')
	}
	p := Slice(make([]string, round)).Stream().EachLine(plusOne).EachLine(timesTwo).Synchronize().EachLine(timesTwo)
	if p.err != nil {
		t.Errorf("unexpected error value: %v", p.err)
	}
	want := 0
	for i := 0; i < round; i++ {
		want++
		want *= 2
	}
	for i := 0; i < round; i++ {
		want *= 2
	}
	if n != want {
		t.Errorf("want n = %d, got %d", want, n)
	}
}

// doMethodsOnPipe calls every kind of method on the supplied pipe and
// tries to trigger a panic.
func doMethodsOnPipe(t *testing.T, p *Pipe, kind string) {
	var action string
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("panic: %s on %s pipe", action, kind)
		}
	}()
	defer os.Remove("testdata/bogus.txt")
	action = "AppendFile()"
	p.AppendFile("testdata/bogus.txt")
	action = "Basename()"
	p.Basename()
	action = "Bytes()"
	p.Bytes()
	action = "Close()"
	p.Close()
	action = "Column()"
	p.Column(2)
	action = "Concat()"
	p.Concat()
	action = "CountLines()"
	p.CountLines()
	action = "Dirname()"
	p.Dirname()
	action = "EachLine()"
	p.EachLine(func(string, *strings.Builder) {})
	action = "Error()"
	p.Error()
	action = "Exec()"
	p.Exec("bogus")
	action = "ExecForEach()"
	p.ExecForEach("bogus")
	action = "ExitStatus()"
	p.ExitStatus()
	action = "First()"
	p.First(1)
	action = "Freq()"
	p.Freq()
	action = "Join()"
	p.Join()
	action = "Last()"
	p.Last(1)
	action = "Match()"
	p.Match("foo")
	action = "MatchRegexp()"
	p.MatchRegexp(regexp.MustCompile(".*"))
	action = "Read()"
	p.Read([]byte{})
	action = "Replace()"
	p.Replace("old", "new")
	action = "ReplaceRegexp()"
	p.ReplaceRegexp(regexp.MustCompile(".*"), "")
	action = "SetError()"
	p.SetError(nil)
	action = "SHA256Sums()"
	p.SHA256Sums()
	action = "SHA256Sum()"
	p.SHA256Sum()
	action = "Slice()"
	p.Slice()
	action = "Stdout()"
	// Ensure we don't clash with TestStdout
	stdoutM.Lock()
	defer stdoutM.Unlock()
	p.Stdout()
	action = "Stream()"
	p.Stream()
	action = "String()"
	p.String()
	action = "Synchronize()"
	p.Synchronize()
	action = "WithError()"
	p.WithError(nil)
	action = "WithReader()"
	p.WithReader(strings.NewReader(""))
	action = "WriteFile()"
	p.WriteFile("testdata/bogus.txt")
}

func TestNilPipes(t *testing.T) {
	t.Parallel()
	doMethodsOnPipe(t, nil, "nil")
}

func TestZeroPipes(t *testing.T) {
	t.Parallel()
	doMethodsOnPipe(t, &Pipe{}, "zero")
}

func TestNewPipes(t *testing.T) {
	t.Parallel()
	doMethodsOnPipe(t, NewPipe(), "new")
}

func TestPipeIsReader(t *testing.T) {
	t.Parallel()
	var p io.Reader = NewPipe()
	_, err := ioutil.ReadAll(p)
	if err != nil {
		t.Error(err)
	}
}
