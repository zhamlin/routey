package routey

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/zhamlin/routey/internal"
	"github.com/zhamlin/routey/internal/color"
	"github.com/zhamlin/routey/internal/structs"
)

// ErrorConfig contains options used to modify the generated errors.
type ErrorConfig struct {
	// Whether or not to include color in the error messages.
	Colored bool
	// The amount of callers to skip when finding the caller of a func
	// that produced an error.
	CallerSkip int

	// Whether or not to stop after the first extractor error.
	CollectAll bool
}

func (e ErrorConfig) color() structs.Colors {
	if e.Colored {
		return coloredErrors
	}
	return structs.NoErrorColors
}

type HandlerError struct {
	Pattern    string
	Handler    internal.FnInfo
	Err        error
	CallerSkip int
}

func (h HandlerError) Unwrap() error {
	return h.Err
}

func (h HandlerError) Error() string {
	return createHandlerErrMsg(h, errorParams{
		Caller: internal.GetCaller(h.CallerSkip + 1),
		Colors: structs.NoErrorColors,
	})
}

func (h HandlerError) ErrorWithColor(c structs.Colors) string {
	return createHandlerErrMsg(h, errorParams{
		Caller: internal.GetCaller(h.CallerSkip + 1),
		Colors: c,
	})
}

// getParentAndBase returns the parent directory and base filename from a path.
// "/foo/bar/file.go" returns "bar/file.go".
func getParentAndBase(path string) string {
	// Clean the path to handle any . or .. and ensure consistent separators
	path = filepath.Clean(path)

	// Get the directory and base name
	dir := filepath.Dir(path)
	base := filepath.Base(path)

	// Get the parent directory name
	parent := filepath.Base(dir)

	// Handle special cases
	if parent == "." || parent == "/" || parent == filepath.VolumeName(parent) {
		return base
	}

	return filepath.Join(parent, base)
}

var coloredErrors = structs.Colors{
	Error: color.Red,
	Reset: color.Reset,
}

type errorParams struct {
	Caller internal.CallerInfo
	Colors structs.Colors
}

type coloredError struct {
	err    error
	colors structs.Colors
}

func (e coloredError) Error() string {
	return errorString(e.err, e.colors)
}

func (e coloredError) Unwrap() error {
	return e.err
}

func errorString(err error, colors structs.Colors) string {
	var coloredError interface {
		ErrorWithColor(structs.Colors) string
	}

	if errors.As(err, &coloredError) {
		return coloredError.ErrorWithColor(colors)
	}
	return err.Error()
}

func createHandlerErrMsg(err HandlerError, params errorParams) string {
	msg := &strings.Builder{}

	writeError(msg, err.Err, params.Colors)
	writeRouteInfo(msg, err, params.Caller)

	if err.Handler.Type != nil {
		writeHandlerInfo(msg, err.Handler)
	}

	return msg.String()
}

func writeError(msg *strings.Builder, err error, colors structs.Colors) {
	m := errorString(err, colors)
	fmt.Fprintln(msg, strings.TrimSuffix(m, "\n"))
	fmt.Fprintln(msg)
}

func writeRouteInfo(msg *strings.Builder, err HandlerError, caller internal.CallerInfo) {
	fmt.Fprintln(msg, "route: "+err.Pattern)
	file := getParentAndBase(caller.File)
	fmt.Fprintf(msg, "|> %s:%d\n", file, caller.Line)
	fmt.Fprintln(msg)
}

func writeHandlerInfo(msg *strings.Builder, handler internal.FnInfo) {
	fmt.Fprintf(msg, "function: %s\n", handler.Name)
	fmt.Fprintf(msg, "| %s\n", handler.Type.String())

	file := getParentAndBase(handler.File)
	fmt.Fprintf(msg, "|> %s:%d\n", file, handler.Line)
}
