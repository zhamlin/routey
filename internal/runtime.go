package internal

import (
	"reflect"
	"regexp"
	"runtime"
	"strings"
)

var packageRE = regexp.MustCompile("github.com/zhamlin/routey.*")

func inPackage(filename string) bool {
	return packageRE.MatchString(filename)
}

type CallerInfo struct {
	File string
	Line int
}

func GetCaller(skip int) CallerInfo {
	const maxChecks = 10

	// skip this func
	skip++

	for i := range maxChecks {
		pc, file, line, ok := runtime.Caller(skip + i)
		if !ok {
			continue
		}

		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}

		inPackge := inPackage(fn.Name())
		inLocalTestFile := inPackge && strings.Contains(file, "_test")

		foundCaller := !inPackge || inLocalTestFile
		if foundCaller {
			return CallerInfo{
				File: file,
				Line: line,
			}
		}
	}

	return CallerInfo{}
}

type FnInfo struct {
	Name    string
	Pkg     string
	File    string
	Line    int
	Type    reflect.Type
	Args    []reflect.Type
	Returns []reflect.Type
}

func getArgsAndReturns(t reflect.Type) ([]reflect.Type, []reflect.Type) {
	in := make([]reflect.Type, t.NumIn())
	out := make([]reflect.Type, t.NumOut())

	for i := range t.NumIn() {
		in[i] = t.In(i)
	}

	for i := range t.NumOut() {
		out[i] = t.Out(i)
	}

	return in, out
}

func GetFnInfo(fn any) FnInfo {
	v := reflect.ValueOf(fn)
	if v.Kind() != reflect.Func {
		panic("GetFnInfo expects a function, got: " + v.Kind().String())
	}

	pc := v.Pointer()
	runtimeFn := runtime.FuncForPC(pc)
	file, line := runtimeFn.FileLine(pc)
	strs := strings.Split(runtimeFn.Name(), ".")
	in, out := getArgsAndReturns(v.Type())

	return FnInfo{
		Name:    strs[len(strs)-1],
		Pkg:     strs[len(strs)-2],
		File:    file,
		Line:    line,
		Args:    in,
		Returns: out,
		Type:    v.Type(),
	}
}
