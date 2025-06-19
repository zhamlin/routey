package jsonschema

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// Validator compiles json schemas and validate input against them.
type Validator struct {
	compiler *jsonschema.Compiler
	schemas  map[string]*jsonschema.Schema
}

type noopLoader struct{}

var ErrSchemaLoad = errors.New("remote schemas are not supported")

func (noopLoader) Load(string) (any, error) {
	return nil, ErrSchemaLoad
}

// NewValidator returns a new [Validator].
func NewValidator() *Validator {
	c := jsonschema.NewCompiler()
	c.DefaultDraft(jsonschema.Draft2020)
	c.UseLoader(noopLoader{})

	return &Validator{
		compiler: c,
		schemas:  map[string]*jsonschema.Schema{},
	}
}

// Add compiles and stores the schema under the given name.
func (c *Validator) Add(name, schema string) error {
	s, err := jsonschema.UnmarshalJSON(strings.NewReader(schema))
	if err != nil {
		return fmt.Errorf("jsonschema.UnmarshalJSON(%s): %w", name, err)
	}

	err = c.compiler.AddResource(name, s)
	if err != nil {
		return fmt.Errorf("compiler.AddResource(%s): %w", name, err)
	}

	return c.compile(name)
}

var ErrSchemaNotFound = errors.New("schema not found in validator")

// Validate validates the input against the compiled schema matching
// the name given.
func (c *Validator) Validate(name string, input []byte) error {
	s, has := c.schemas[name]
	if !has {
		return ErrSchemaNotFound
	}

	var v any
	if err := json.Unmarshal(input, &v); err != nil {
		return err
	}

	if err := s.Validate(v); err != nil {
		var verr *jsonschema.ValidationError
		if errors.As(err, &verr) {
			return convertError(verr)
		}

		return fmt.Errorf("validate(%s): %w", name, err)
	}

	return nil
}

func (c *Validator) compile(name string) error {
	s, err := c.compiler.Compile(name)
	if err != nil {
		return fmt.Errorf("compile(%s): %w", name, err)
	}

	c.schemas[name] = s
	return nil
}

// ValidationError represents any errors that occurred during
// validation of a json object against a schema.
type ValidationError struct {
	OriginalError *jsonschema.ValidationError

	Causes   []ValidationError
	Message  string
	Location string
}

func (ve ValidationError) String() string {
	loc := ve.Location
	if loc == "" {
		loc = "/"
	}

	loc = loc[strings.IndexByte(loc, '#')+1:]
	msg := fmt.Sprintf("[#%s]", loc)

	if ve.Message != "" {
		msg += " " + ve.Message
	}

	for _, c := range ve.Causes {
		for line := range strings.SplitSeq(c.String(), "\n") {
			msg += "\n  " + line
		}
	}

	return msg
}

func (ve ValidationError) Error() string {
	return ve.String()
}

func validationErrToErrorDetail(verr *jsonschema.ValidationError) []ValidationError {
	details := []ValidationError{}

	if len(verr.Causes) == 0 {
		details = append(details, ValidationError{
			Message:  verr.BasicOutput().Error.String(),
			Location: "/" + strings.Join(verr.InstanceLocation, "/"),
		})
	}

	for _, c := range verr.Causes {
		details = append(details, validationErrToErrorDetail(c)...)
	}

	return details
}

func convertError(e *jsonschema.ValidationError) ValidationError {
	causes := validationErrToErrorDetail(e)
	err := ValidationError{
		OriginalError: e,
		Causes:        causes,
	}

	if len(err.Causes) == 0 {
		err.Message = e.BasicOutput().Error.String()
		err.Location = "/" + strings.Join(e.InstanceLocation, "/")
	}

	return err
}
