package color

// Color represents an ANSI Color code.
type Color string

const (
	Reset   Color = "\033[0m"
	Red     Color = "\033[31m"
	Green   Color = "\033[32m"
	Yellow  Color = "\033[33m"
	Blue    Color = "\033[34m"
	Magenta Color = "\033[35m"
	Cyan    Color = "\033[36m"
)
