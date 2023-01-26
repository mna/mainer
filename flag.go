package mainer

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
)

// Parser implements a command-line flags parser that uses struct tags to
// configure supported flags and returns any error it encounters, without
// printing anything automatically. It can optionally read flag values from
// environment variables first, with the command-line flags used to override
// them.
//
// The struct tag to specify flags is `flag`, while the one to specify
// environment variables is `envconfig`. See the envconfig package for full
// details on struct tags configuration and decoding support:
// https://github.com/kelseyhightower/envconfig.
//
// Flag parsing uses the stdlib's flag package internally, and as such shares
// the same behaviour regarding short and long flags. However, it does
// support mixing order of flag arguments and non-flag ones.
type Parser struct {
	// EnvVars indicates if environment variables are used to read flag values.
	EnvVars bool

	// EnvPrefix is the prefix to use in front of each flag's environment
	// variable name. If it is empty, the name of the program (as read from the
	// args slice at index 0) is used, with dashes replaced with underscores.
	// Set it to "-" to disable any prefix.
	EnvPrefix string
}

// Parse parses args into v, using struct tags to detect flags.  The tag must
// be named "flag" and multiple flags may be set for the same field using a
// comma-separated list. v must be a pointer to a struct and the flags must be
// defined on fields with a type of string, int/int64, uint/uint64, float64,
// bool or time.Duration. If Parser.EnvVars is true, flag values are
// initialized from corresponding environment variables first.
//
// After parsing, if v implements a Validate method that returns an error, it
// is called and any non-nil error is returned as error.
//
// If v has a SetArgs([]string) method, it is called with the list of non-flag
// arguments (a slice of strings).
//
// If v has a SetFlags(map[string]bool) method, it is called with the set of
// flags that were explicitly set by args (a map[string]bool). Note that if
// a field can be set with multiple flags, the actual flag used on the command
// line will be set as key.
//
// It panics if v is not a pointer to a struct or if a flag is defined with an
// unsupported type.
func (p *Parser) Parse(args []string, v interface{}) error {
	if p.EnvVars {
		if err := p.parseEnvVars(args, v); err != nil {
			return err
		}
	}

	if err := p.parseFlags(args, v); err != nil {
		return err
	}

	if val, ok := v.(interface{ Validate() error }); ok {
		return val.Validate()
	}
	return nil
}

var durationType = reflect.TypeOf(time.Duration(0))

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
	for i := 0; i < count; i++ {
		fld := val.Field(i)
		typ := str.Field(i)
		names := strings.Split(typ.Tag.Get("flag"), ",")

		for _, nm := range names {
			nm = strings.TrimSpace(nm)
			if nm == "" {
				continue
			}

			// check for well-known types first, as their underlying type might be a
			// basic kind (so it must be checked before the basic kinds are
			// processed).
			switch typ.Type {
			case durationType:
				fs.DurationVar(fld.Addr().Interface().(*time.Duration), nm, fld.Interface().(time.Duration), "")
			default:
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

	var (
		nonFlags []string
		flagSet  map[string]bool
	)
	args = args[1:]
	for len(args) > 0 {
		if err := fs.Parse(args); err != nil {
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

		if _, ok := v.(interface{ SetFlags(map[string]bool) }); ok {
			fs.Visit(func(fl *flag.Flag) {
				if flagSet == nil {
					flagSet = make(map[string]bool)
				}
				flagSet[fl.Name] = true
			})
		}
	}

	if sa, ok := v.(interface{ SetArgs([]string) }); ok {
		sa.SetArgs(nonFlags)
	}
	if sf, ok := v.(interface{ SetFlags(map[string]bool) }); ok {
		sf.SetFlags(flagSet)
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
	return envconfig.Process(prefix, v)
}

func prefixFromProgramName(name string) string {
	name = filepath.Base(name)
	ext := filepath.Ext(name)
	if ext != "" {
		name = strings.TrimSuffix(name, ext)
	}
	return strings.ReplaceAll(name, "-", "_")
}
