package mainer

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
	"github.com/google/go-cmp/cmp"
)

type F struct {
	S     string        `flag:"s,string,long-string"`
	I     int           `flag:"i,int"`
	B     bool          `flag:"b"`
	H     bool          `flag:"h,help"`
	T     time.Duration `flag:"t"`
	N     int
	args  []string
	flags map[string]bool
}

var equalsF = qt.CmpEquals(cmp.AllowUnexported(F{}))

func (f *F) SetArgs(args []string) {
	f.args = args
}

func (f *F) SetFlags(flags map[string]bool) {
	f.flags = flags
}

func TestParseFlags(t *testing.T) {
	c := qt.New(t)

	cases := []struct {
		args []string
		want *F
		err  string
	}{
		{
			want: &F{},
		},
		{
			args: []string{"toto"},
			want: &F{
				args: []string{"toto"},
			},
		},
		{
			args: []string{"-h"},
			want: &F{
				H:     true,
				flags: map[string]bool{"h": true},
			},
		},
		{
			args: []string{"-i", "10", "--int", "20"},
			want: &F{
				I:     20,
				flags: map[string]bool{"i": true, "int": true},
			},
		},
		{
			args: []string{"-i", "10", "--int", "20"},
			want: &F{
				I:     20,
				flags: map[string]bool{"i": true, "int": true},
			},
		},
		{
			args: []string{"-s", "a", "--string", "b", "-long-string", "c"},
			want: &F{
				S:     "c",
				flags: map[string]bool{"s": true, "string": true, "long-string": true},
			},
		},
		{
			args: []string{"-b", "--b", "-b"},
			want: &F{
				B:     true,
				flags: map[string]bool{"b": true},
			},
		},
		{
			args: []string{"-b", "-int", "1", "-string", "a", "arg1", "arg2"},
			want: &F{
				B:     true,
				I:     1,
				S:     "a",
				args:  []string{"arg1", "arg2"},
				flags: map[string]bool{"b": true, "int": true, "string": true},
			},
		},
		{
			args: []string{"-n", "1"},
			want: &F{},
			err:  "not defined: -n",
		},
		{
			args: []string{"-t", "3s"},
			want: &F{
				T:     3 * time.Second,
				flags: map[string]bool{"t": true},
			},
		},
		{
			args: []string{"-t", "nope"},
			want: &F{},
			err:  "invalid value",
		},
		{
			args: []string{"arg1", "-i", "1", "arg2", "-b"},
			want: &F{
				I:     1,
				B:     true,
				args:  []string{"arg1", "arg2"},
				flags: map[string]bool{"i": true, "b": true},
			},
		},
		{
			args: []string{"arg1", "-z", "arg2"},
			want: &F{},
			err:  "not defined: -z",
		},
		{
			args: []string{"arg1", "--", "-i", "2"},
			want: &F{
				args: []string{"arg1", "-i", "2"},
			},
		},
	}

	var p Parser
	for _, tc := range cases {
		c.Run(strings.Join(tc.args, " "), func(c *qt.C) {
			var f F
			args := append([]string{""}, tc.args...)
			err := p.Parse(args, &f)

			if tc.err != "" {
				c.Assert(err, qt.IsNotNil)
				c.Assert(err.Error(), qt.Contains, tc.err)
				return
			}

			c.Assert(err, qt.IsNil)
			c.Assert(&f, equalsF, tc.want)
		})
	}
}

func TestParseNoFlag(t *testing.T) {
	c := qt.New(t)

	type F struct {
		V int
	}
	var p Parser

	f := F{V: 4}
	err := p.Parse([]string{"", "x"}, &f)
	c.Assert(err, qt.IsNil)
	c.Assert(f.V, qt.Equals, 4)
}

type noFlagSetArgs struct {
	args []string
}

func (n *noFlagSetArgs) SetArgs(args []string) {
	n.args = args
}

func TestParseNoFlagSetArgs(t *testing.T) {
	c := qt.New(t)

	var p Parser
	f := noFlagSetArgs{}
	err := p.Parse([]string{"", "x"}, &f)
	c.Assert(err, qt.IsNil)
	c.Assert(f.args, qt.DeepEquals, []string{"x"})
}

func TestParseArgsError(t *testing.T) {
	c := qt.New(t)

	type F struct {
		X bool `flag:"x"`
	}
	var p Parser
	f := F{}
	err := p.Parse([]string{"", "-zz"}, &f)
	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Contains, "-zz")
}

func TestParseNotStructPointer(t *testing.T) {
	c := qt.New(t)

	var (
		i int
		p Parser
	)
	c.Assert(func() {
		_ = p.Parse([]string{"-h"}, i)
	}, qt.PanicMatches, `reflect:.+`)
}

func TestParseUnsupportedFlagType(t *testing.T) {
	c := qt.New(t)

	type F struct {
		C *bool `flag:"c"`
	}
	var (
		f F
		p Parser
	)
	c.Assert(func() {
		_ = p.Parse([]string{"", "-h"}, &f)
	}, qt.PanicMatches, `unsupported.+`)
}

type E struct {
	Addr    string `flag:"addr"`
	DB      string `flag:"db"`
	Help    bool   `flag:"h,help" ignored:"true"`
	Version bool   `flag:"v,version" ignored:"true"`
}

func (e *E) Validate() error {
	if e.Help || e.Version {
		return nil
	}
	if e.Addr == "" {
		return errors.New("address must be set")
	}
	if e.DB == "" {
		return errors.New("db must be set")
	}
	return nil
}

func TestParseEnvVars(t *testing.T) {
	c := qt.New(t)

	const progName = "/tmp/go-build903761289/b001/exe/mainer-test"

	p := Parser{
		EnvVars: true,
	}

	cases := []struct {
		env    string // prefix-less Key:val pairs, space-separated
		args   string // space-separated, index 0 added automatically
		want   E
		errMsg string // error must contain that errMsg
	}{
		{
			"",
			"",
			E{},
			"address must be set",
		},
		{
			"ADDR::1234 DB:x",
			"",
			E{Addr: ":1234", DB: "x"},
			"",
		},
		{
			"",
			"-addr :2345 -db v",
			E{Addr: ":2345", DB: "v"},
			"",
		},
		{
			"ADDR::1234",
			"-addr :2345 -db x",
			E{Addr: ":2345", DB: "x"},
			"",
		},
		{
			"HELP:true",
			"-addr :2345",
			E{Addr: ":2345"},
			"db must be set",
		},
		{
			"VERSION:1",
			"-addr :2345 -db x",
			E{Addr: ":2345", DB: "x"},
			"",
		},
		{
			"",
			"-help",
			E{Help: true},
			"",
		},
		{
			"",
			"-v",
			E{Version: true},
			"",
		},
		{
			"",
			"-z",
			E{},
			"flag provided but not defined: -z",
		},
	}
	for _, tc := range cases {
		c.Run(fmt.Sprintf("%s|%s", tc.env, tc.args), func(c *qt.C) {
			// set env vars
			if tc.env != "" {
				envPairs := strings.Split(tc.env, " ")
				for _, pair := range envPairs {
					ix := strings.Index(pair, ":")
					c.Assert(ix >= 0, qt.IsTrue, qt.Commentf("%s: missing colon", pair))

					key, val := pair[:ix], pair[ix+1:]
					key = strings.ToUpper(prefixFromProgramName(progName)) + "_" + key
					c.Setenv(key, val)
				}
			}

			// parse args
			args := []string{progName}
			if tc.args != "" {
				args = append(args, strings.Split(tc.args, " ")...)
			}

			var e E
			err := p.Parse(args, &e)
			if tc.errMsg != "" {
				c.Assert(err, qt.IsNotNil)
				c.Assert(err.Error(), qt.Contains, tc.errMsg)
			} else {
				c.Assert(err, qt.IsNil)
			}

			c.Assert(e, qt.DeepEquals, tc.want)
		})
	}
}
