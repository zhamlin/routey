package param

import (
	"strings"

	"github.com/zhamlin/routey/internal/stringz"
)

// Namer takes a parameter name and style returning a new name.
type Namer func(name, style string) string

func NamerCapitals(name, _ string) string {
	chunks := stringz.SplitByCapitals(name)
	for i := range chunks {
		chunks[i] = strings.ToLower(chunks[i])
	}

	return strings.Join(chunks, "_")
}
