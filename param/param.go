package param

import (
	"net/http"
)

// Config contains things used to parse params.
type Config struct {
	// Determines how params are parsed.
	Parser Parser
	// Allows modifying of param names from the structs field name.
	Namer Namer
}

// Pather is the interface implemented by an object that can
// return the value of a path parameter from a http request.
type Pather interface {
	Param(name string, r *http.Request) string
}

// Opts contains information get and parse a param.
type Opts struct {
	Name    string
	Default string
	Parser  Parser
	Pather  Pather
}

func (o Opts) PathValue(name string, r *http.Request) string {
	return o.Pather.Param(name, r)
}

func (o Opts) Parse(value any, params []string) error {
	if l := len(params); l == 0 && o.Default != "" {
		params = []string{o.Default}
	} else if l == 0 {
		return nil
	}
	return o.Parser(value, params)
}
