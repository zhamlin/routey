package routey_test

import (
	"errors"
	"path"
	"testing"

	"github.com/zhamlin/routey"
	"github.com/zhamlin/routey/internal"
	"github.com/zhamlin/routey/internal/test"
)

type testHandlerInput struct {
	Value      int
	QueryValue routey.Query[int]
}

func testHandler(testHandlerInput) {}

func compareErrors(t *testing.T, err routey.HandlerError, want string) {
	t.Helper()

	got := err.Error()
	const tabSize = 4
	test.VisuallyMatch(t, got, want, tabSize)
}

func getFnInfo(fn any) internal.FnInfo {
	i := internal.GetFnInfo(fn)
	_, i.File = path.Split(i.File)
	return i
}

func TestCreateHandlerErrMsg(t *testing.T) {
	err := routey.HandlerError{
		Pattern:    "/",
		Handler:    getFnInfo(testHandler),
		Err:        errors.New("error text"),
		CallerSkip: 5,
	}

	want := `
error text

route: /
|> .:0

function: testHandler
| func(routey_test.testHandlerInput)
|> error_test.go:18
`
	compareErrors(t, err, want)
}
