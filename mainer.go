// Package mainer defines types relevant to command entrypoint
// implementation. It includes the Stdio struct that abstracts the I/O and
// working directory of a command, and the Parser struct that implements a
// simple flag parser with support for struct tags and environment
// variables-based argument values.
//
// A typical main entrypoint looks like this, where the cmd struct could be
// implemented in a distinct package so the main package is minimal (just
// the main function at the end):
//
//	 type cmd struct {
//	   Help    bool   `flag:"h,help"`
//	   Version bool   `flag:"v,version"`
//	   FooBar  string `flag:"foo-bar" env:"FOO_BAR"`
//	 }
//
//	 func (c *cmd) Validate() error {
//	   // the struct may implement a Validate method, and if it does the
//	   // Parser will call it once the flags are stored in the fields.
//	   return nil
//	 }
//
//	 func (c *cmd) Main(args []string, stdio mainer.Stdio) mainer.ExitCode {
//	   // parse the flags, using env var <CMD>_FOO_BAR for the --foo-bar
//	   // flag if the flag is not set (where <CMD> defaults to the base name
//	   // of the executable, in uppercase and without extension).
//		 p := &mainer.Parser{EnvVars: true}
//		 if err := p.Parse(args, c); err != nil {
//		 	 fmt.Fprintln(stdio.Stderr, err)
//		 	 return mainer.InvalidArgs
//		 }
//
//	   // execute the command...
//	   return mainer.Success
//	 }
//
//	 func main() {
//	   var c cmd
//	   os.Exit(int(c.Main(os.Args, mainer.CurrentStdio())))
//	 }
package mainer

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
)

// ExitCode is the type of a process exit code.
type ExitCode int

// List of pre-defined exit codes.
const (
	Success ExitCode = iota
	Failure
	InvalidArgs
)

// CurrentStdio returns the Stdio for the current process. Its Cwd
// field reflects the working directory at the time of the call.
func CurrentStdio() Stdio {
	cwd, err := os.Getwd()
	if err != nil {
		panic(fmt.Sprintf("failed to get current working directory: %s", err))
	}
	return Stdio{
		Cwd:    cwd,
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}

// Stdio defines the OS abstraction for standard I/O.
type Stdio struct {
	// Cwd is the current working directory.
	Cwd string

	// Stdin is the standard input reader.
	Stdin io.Reader

	// Stdout is the standard output writer.
	Stdout io.Writer

	// Stderr is the standard error writer.
	Stderr io.Writer
}

// Mainer defines the method to implement for a type that
// implements a Main entrypoint of a command.
type Mainer interface {
	Main([]string, Stdio) ExitCode
}

// CancelOnSignal returns a context that is canceled when the process receives
// one of the specified signals.
func CancelOnSignal(ctx context.Context, signals ...os.Signal) context.Context {
	if len(signals) == 0 {
		return ctx
	}

	ctx, cancel := context.WithCancel(ctx)

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, signals...)
	go func() {
		<-ch
		cancel()
	}()

	return ctx
}
