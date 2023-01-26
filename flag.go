package mainer

import (
	"encoding"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/caarlos0/env/v6"
)

// Parser implements a command-line flags parser that uses struct tags to
// configure supported flags and returns any error it encounters, without
// printing anything automatically. It can optionally read flag values from
// environment variables first, with the command-line flags used to override
// them.
//
// The struct tag to specify flags is `flag`, while the one to specify
// environment variables is `env`. See the env/v6 package for full details on
// struct tags configuration and decoding support:
// https://github.com/caarlos0/env.
//
// Flag parsing uses the stdlib's flag package internally, and as such shares
// the same behaviour regarding short and long flags. However, it does
// support mixing order of flag arguments and non-flag ones.
type Parser struct {
	// EnvVars indicates if environment variables are used to read flag values.
	EnvVars bool

	// EnvPrefix is the prefix to use in front of each flag's environment
	// variable name. If it is empty, the name of the program (as read from the
	// args slice at index 0) is used, all uppercase and with dashes replaced
	// with underscores. Set it to "-" to disable any prefix.
	EnvPrefix string
}

// Parse parses args into v, using struct tags to detect flags. Note that the
// args slice should start with the program name (as is the case for `os.Args`,
// which is typically used). The tag must be named "flag" and multiple flags
// may be set for the same field using a comma-separated list.
//
// v must be a pointer to a struct and the flags must be defined on exported
// fields with a type of string, int/int64, uint/uint64, float64, bool or
// time.Duration or with a type that directly implements
// encoding.TextMarshaler/TextUnmarshaler (both interfaces must be satisfied),
// or on a type T that implements those interfaces on *T (a pointer to the
// type).
//
// If Parser.EnvVars is true, flag values are initialized from corresponding
// environment variables first, as defined by the github.com/caarlos0/env/v6
// package (which is used for environment parsing).
//
// Flags and arguments can be interspersed, but flag parsing stops if it
// encounters the "--" value; all subsequent values are treated as arguments.
//
// After parsing, if v implements a Validate method that returns an error, it
// is called and any non-nil error is returned as error.
//
// If v has a SetArgs([]string) method, it is called with the list of non-flag
// arguments (a slice of strings) that respects the provided order.
//
// If v has a SetFlags(map[string]bool) method, it is called with the set of
// flags that were explicitly set by args (a map[string]bool). Note that if a
// field can be set with multiple flags, the key is canonicalized to the first
// flag defined on the field.
//
// If v has a SetFlagsCount(map[string]int) method, it is called with the map
// of flags that were explicitly set by args, and the associated value is the
// number of times the flag was provided. As for SetFlags, the key is
// canonicalized to the first flag defined on the field.
//
// It panics if v is not a pointer to a struct or if a flag is defined with an
// unsupported type.
func (p *Parser) Parse(args []string, v interface{}) error {
	if p.EnvVars {
		if err := p.parseEnvVars(args, v); err != nil {
			return err
		}
	}

	// TODO: support []string (and other types?) that collects all values set via multiple flags
	// TODO: support []string (and other types?) that collects all values via comma-separated list

	if err := p.parseFlags(args, v); err != nil {
		return err
	}

	if val, ok := v.(interface{ Validate() error }); ok {
		return val.Validate()
	}
	return nil
}

var durationType = reflect.TypeOf(time.Duration(0))

type valueSetter struct {
	flag.Value
	setter func(string) error
	isBool bool
}

func (v valueSetter) Set(s string) error {
	return v.setter(s)
}

func (v valueSetter) IsBoolFlag() bool {
	return v.isBool
}

func (p *Parser) parseFlags(args []string, v interface{}) error {
	if len(args) == 0 {
		return nil
	}

	// create a FlagSet that is silent and only returns any error
	// it encounters.
	fs := flag.NewFlagSet("", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = nil

	// extract the flags from the struct
	val := reflect.ValueOf(v).Elem()
	str := val.Type()
	count := val.NumField()
	canonLookup := make(map[string]string, count) // key is flag name, value is canonical name

	for i := 0; i < count; i++ {
		fld := val.Field(i)
		typ := str.Field(i)
		names := strings.Split(typ.Tag.Get("flag"), ",")

		var canonFlag string
		for _, nm := range names {
			nm = strings.TrimSpace(nm)
			if nm == "" {
				continue
			}
			if canonFlag == "" {
				canonFlag = nm
			}
			canonLookup[nm] = canonFlag

			// check for well-known types first, as their underlying type might be a
			// basic kind (so it must be checked before the basic kinds are
			// processed).
			switch typ.Type {
			case durationType:
				fs.DurationVar(fld.Addr().Interface().(*time.Duration), nm, fld.Interface().(time.Duration), "")
			default:
				// for flag.TextVar to be supported, the type must implement both
				// TextUnmarshaler and TextMarshaler. As a convenience, if the type
				// does not implement TextUnmarshaler but a pointer to the type does,
				// support it.
				tuv, okuv := fld.Interface().(encoding.TextUnmarshaler)
				tmv, okmv := fld.Interface().(encoding.TextMarshaler)
				tup, okup := fld.Addr().Interface().(encoding.TextUnmarshaler)
				tmp, okmp := fld.Addr().Interface().(encoding.TextMarshaler)
				if okuv && okmv {
					// the field's value itself implements both
					fs.TextVar(tuv, nm, tmv, "")
					continue
				} else if (okuv || okup) && (okmv || okmp) {
					// the pointer implements the missing one, so use a pointer for the flag
					fs.TextVar(tup, nm, tmp, "")
					continue
				}

				switch fld.Kind() {
				case reflect.Bool:
					fs.BoolVar(fld.Addr().Interface().(*bool), nm, fld.Bool(), "")
				case reflect.String:
					fs.StringVar(fld.Addr().Interface().(*string), nm, fld.String(), "")
				case reflect.Int:
					fs.IntVar(fld.Addr().Interface().(*int), nm, int(fld.Int()), "")
				case reflect.Int64:
					fs.Int64Var(fld.Addr().Interface().(*int64), nm, fld.Int(), "")
				case reflect.Uint:
					fs.UintVar(fld.Addr().Interface().(*uint), nm, uint(fld.Uint()), "")
				case reflect.Uint64:
					fs.Uint64Var(fld.Addr().Interface().(*uint64), nm, fld.Uint(), "")
				case reflect.Float64:
					fs.Float64Var(fld.Addr().Interface().(*float64), nm, fld.Float(), "")
				default:
					panic(fmt.Sprintf("unsupported flag field kind: %s (%s: %s)", fld.Kind(), typ.Name, typ.Type))
				}
			}
		}
	}

	var flagsCount map[string]int
	if _, ok := v.(interface{ SetFlagsCount(map[string]int) }); ok {
		// v implements SetFlagsCount, so wrap each flag in a func that will count
		// and report the number of times it was set (under the canonical - first
		// defined - flag name).
		flagsCount = make(map[string]int)

		fs.VisitAll(func(fl *flag.Flag) {
			inner := fl.Value
			setter := valueSetter{
				Value: inner,
				setter: func(s string) error {
					flagsCount[canonLookup[fl.Name]]++
					return inner.Set(s)
				},
			}
			if bo, ok := inner.(interface{ IsBoolFlag() bool }); ok && bo.IsBoolFlag() {
				setter.isBool = true
			}
			fl.Value = setter
		})
	}

	var nonFlags []string
	args = args[1:] // skip the program name
	for len(args) > 0 {
		if err := fs.Parse(args); err != nil {
			if err == flag.ErrHelp {
				if fs.Lookup("help") == nil && sliceContains(args, "-help") {
					return errors.New("flag provided but not defined: -help")
				}
				return errors.New("flag provided but not defined: -h")
			}
			return err
		}

		args = nil
		curNonFlags := fs.Args()
		for i, nf := range curNonFlags {
			if nf == "--" {
				// ignore this one, but treat all subsequent as non-flags
				nonFlags = append(nonFlags, curNonFlags[i+1:]...)
				break
			}
			if ((strings.HasPrefix(nf, "-") && len(nf) > 1) ||
				(strings.HasPrefix(nf, "--") && len(nf) > 2)) &&
				!strings.HasPrefix(nf, "---") {

				// this is a flag, stop non-flags here
				args = curNonFlags[i:]
				break
			}
			nonFlags = append(nonFlags, nf)
		}
	}

	if sa, ok := v.(interface{ SetArgs([]string) }); ok {
		sa.SetArgs(nonFlags)
	}

	if sf, ok := v.(interface{ SetFlags(map[string]bool) }); ok {
		var flagSet map[string]bool
		fs.Visit(func(fl *flag.Flag) {
			if flagSet == nil {
				flagSet = make(map[string]bool)
			}
			flagSet[canonLookup[fl.Name]] = true
		})
		sf.SetFlags(flagSet)
	}

	if sfc, ok := v.(interface{ SetFlagsCount(map[string]int) }); ok {
		if len(flagsCount) == 0 {
			sfc.SetFlagsCount(nil)
		} else {
			sfc.SetFlagsCount(flagsCount)
		}
	}

	return nil
}

func (p *Parser) parseEnvVars(args []string, v interface{}) error {
	prefix := p.EnvPrefix

	if prefix == "" && len(args) > 0 {
		prefix = prefixFromProgramName(args[0])
	}
	if prefix == "-" {
		prefix = ""
	}
	return env.Parse(v, env.Options{Prefix: prefix})
}

func prefixFromProgramName(name string) string {
	name = filepath.Base(name)
	ext := filepath.Ext(name)
	if ext != "" {
		name = strings.TrimSuffix(name, ext)
	}
	return strings.ToUpper(strings.ReplaceAll(name, "-", "_")) + "_"
}

func sliceContains(sl []string, s string) bool {
	for _, ss := range sl {
		if ss == s {
			return true
		}
	}
	return false
}
