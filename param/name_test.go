package param_test

import (
	"testing"

	"github.com/zhamlin/routey/param"
)

func TestNamerCapitals(t *testing.T) {
	got := param.NamerCapitals("lowerUpper", "")
	want := "lower_upper"

	if got != want {
		t.Errorf("wanted: %s, got: %s", want, got)
	}
}
