[![Go Reference](https://pkg.go.dev/badge/github.com/mna/mainer.svg)](https://pkg.go.dev/github.com/mna/mainer)
[![Build Status](https://github.com/mna/mainer/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/mna/mainer/actions)

# mainer

Package mainer defines types relevant to command entrypoint implementation.
It includes the `Stdio` struct that abstracts the I/O and working directory of
a command, and the `Parser` struct that implements a simple flag parser with
support for struct tags and environment variables-based argument values.

## Installation

    $ go get github.com/mna/mainer

## Description

The [code documentation](https://pkg.go.dev/github.com/mna/mainer) is the
canonical source for documentation.

A typical main entrypoint looks like this, where the cmd struct could be
implemented in a distinct package so the main package is minimal (just
the main function at the end) and the command implementation is easily
testable:

```go
type cmd struct {
  Help    bool   `flag:"h,help"`
  Version bool   `flag:"v,version"`
  FooBar  string `flag:"foo-bar" env:"FOO_BAR"`
}

func (c *cmd) Validate() error {
  // the struct may implement a Validate method, and if it does the
  // Parser will call it once the flags are stored in the fields.
  return nil
}

func (c *cmd) Main(args []string, stdio mainer.Stdio) mainer.ExitCode {
  // parse the flags, using env var <CMD>_FOO_BAR for the --foo-bar
  // flag if the flag is not set (where <CMD> defaults to the base name
  // of the executable, in uppercase and without extension).
  p := &mainer.Parser{EnvVars: true}
  if err := p.Parse(args, c); err != nil {
    fmt.Fprintln(stdio.Stderr, err)
    return mainer.InvalidArgs
  }

  // execute the command...
  return mainer.Success
}

func main() {
  var c cmd
  os.Exit(int(c.Main(os.Args, mainer.CurrentStdio())))
}
```

## Breaking changes

### v0.3

* Use `github.com/caarlos0/env/v6` as environment-variable parsing package instead of `github.com/kelseyhightower/envconfig`.
* Flag names are trimmed of any leading and trailing spaces (e.g. `flag:" h , hello "` now define the flags "h" and "hello").
* `SetFlags` now reports set flags using a canonical flag name (the first flag defined on the field). Which of the various flag aliases was used should not matter (if it does, define distinct fields instead).

## License

The [BSD 3-Clause license](http://opensource.org/licenses/BSD-3-Clause).

