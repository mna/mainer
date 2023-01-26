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
	S string        `flag:"s,string,long-string"`
	I int           `flag:"i,int"`
	B bool          `flag:"b"`
	H bool          `flag:"h,help"`
	T time.Duration `flag:"t"`
	N int

	I64 int64   `flag:"i64"`
	U64 uint64  `flag:"u64"`
	F64 float64 `flag:"f64"`

	Spaced string `flag:", sp , spaced "`

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
		args []string // args only, the 0-index is automatically added in test
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
				flags: map[string]bool{"i": true},
			},
		},
		{
			args: []string{"--int", "20", "-i", "10"},
			want: &F{
				I:     10,
				flags: map[string]bool{"i": true},
			},
		},
		{
			args: []string{"-s", "a", "--string", "b", "-long-string", "c"},
			want: &F{
				S:     "c",
				flags: map[string]bool{"s": true},
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
				flags: map[string]bool{"b": true, "i": true, "s": true},
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
		{
			args: []string{"-sp", "hello"},
			want: &F{
				Spaced: "hello",
				flags:  map[string]bool{"sp": true},
			},
		},
		{
			args: []string{"--spaced", "hello"},
			want: &F{
				Spaced: "hello",
				flags:  map[string]bool{"sp": true},
			},
		},
		{
			args: []string{"--i64", "-123", "-u64", "456", "-f64", "3.1415"},
			want: &F{
				I64:   -123,
				U64:   456,
				F64:   3.1415,
				flags: map[string]bool{"i64": true, "u64": true, "f64": true},
			},
		},
		{
			args: []string{"-u64", "-1"},
			want: &F{},
			err:  `invalid value "-1" for flag -u64`,
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

type Fc struct {
	S string `flag:"string,s"`
	I int    `flag:"int,i"`
	B bool   `flag:"b"`

	counts map[string]int
}

var equalsFc = qt.CmpEquals(cmp.AllowUnexported(Fc{}))

func (f *Fc) SetFlagsCount(flags map[string]int) {
	f.counts = flags
}

func TestParseFlagsCount(t *testing.T) {
	c := qt.New(t)

	cases := []struct {
		args []string // args only, the 0-index is automatically added in test
		want *Fc
		err  string
	}{
		{
			want: &Fc{},
		},
		{
			args: []string{"toto"},
			want: &Fc{},
		},
		{
			args: []string{"-h"},
			want: &Fc{},
			err:  "not defined: -h",
		},
		{
			args: []string{"-help"},
			want: &Fc{},
			err:  "not defined: -help",
		},
		{
			args: []string{"-s", "a"},
			want: &Fc{
				S:      "a",
				counts: map[string]int{"string": 1},
			},
		},
		{
			args: []string{"-b", "-b", "-b"},
			want: &Fc{
				B:      true,
				counts: map[string]int{"b": 3},
			},
		},
		{
			args: []string{"-b", "-i", "1", "--int", "2", "a", "-s", "x", "b", "-i", "3", "c", "--string", "y"},
			want: &Fc{
				B:      true,
				I:      3,
				S:      "y",
				counts: map[string]int{"b": 1, "int": 3, "string": 2},
			},
		},
	}

	var p Parser
	for _, tc := range cases {
		c.Run(strings.Join(tc.args, " "), func(c *qt.C) {
			var fc Fc
			args := append([]string{""}, tc.args...)
			err := p.Parse(args, &fc)

			if tc.err != "" {
				c.Assert(err, qt.IsNotNil)
				c.Assert(err.Error(), qt.Contains, tc.err)
				return
			}

			c.Assert(err, qt.IsNil)
			c.Assert(&fc, equalsFc, tc.want)
		})
	}
}

func TestParseDefaultsSet(t *testing.T) {
	c := qt.New(t)

	f := F{
		I:      1,
		I64:    2,
		U64:    3,
		F64:    4.0,
		B:      true,
		S:      "s",
		Spaced: "sp",
		N:      5,
		T:      time.Hour,
	}

	f2 := f
	f2.I = 1000
	f2.flags = map[string]bool{"i": true}

	var p Parser
	err := p.Parse([]string{"", "-i", "1000"}, &f)
	c.Assert(err, qt.IsNil)
	c.Assert(f, equalsF, f2)
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
	c.Assert(err.Error(), qt.Contains, "not defined: -zz")
}

func TestParseDuplicateFlagName(t *testing.T) {
	c := qt.New(t)

	type F struct {
		X bool `flag:"x"`
		Y int  `flag:"x"`
	}
	var p Parser
	f := F{}
	c.Assert(func() {
		_ = p.Parse([]string{"-x", "1"}, &f)
	}, qt.PanicMatches, `flag redefined: x`)
}

func TestParseDuplicateAltFlagName(t *testing.T) {
	c := qt.New(t)

	type F struct {
		X bool `flag:"x,long-x"`
		Y bool `flag:"y,long-x"`
	}
	var p Parser
	f := F{}
	c.Assert(func() {
		_ = p.Parse([]string{"-x", "1"}, &f)
	}, qt.PanicMatches, `flag redefined: long-x`)
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
	Addr    string `flag:"addr" env:"ADDR"`
	DB      string `flag:"db" env:"DB"`
	Help    bool   `flag:"h,help"`
	Version bool   `flag:"v,version"`
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
					key = strings.ToUpper(prefixFromProgramName(progName)) + key
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

type reverseVal string

func (r *reverseVal) UnmarshalText(t []byte) error {
	for i := len(t)/2 - 1; i >= 0; i-- {
		opp := len(t) - 1 - i
		t[i], t[opp] = t[opp], t[i]
	}
	*r = reverseVal(t)
	return nil
}

func (r reverseVal) MarshalText() ([]byte, error) {
	return []byte(r), nil
}

func TestTextUnmarshalerFlagValue(t *testing.T) {
	c := qt.New(t)

	type F struct {
		V reverseVal `flag:"reverse"`
	}
	var (
		f F
		p Parser
	)
	err := p.Parse([]string{"", "-reverse", "hello"}, &f)
	c.Assert(err, qt.IsNil)
	c.Assert(string(f.V), qt.Equals, "olleh")
}

func TestTextUnmarshalerFlagPtr(t *testing.T) {
	c := qt.New(t)

	type F struct {
		V *reverseVal `flag:"reverse"`
	}
	var p Parser
	f := F{V: new(reverseVal)}
	err := p.Parse([]string{"", "-reverse", "hello"}, &f)
	c.Assert(err, qt.IsNil)
	c.Assert(string(*f.V), qt.Equals, "olleh")
}
