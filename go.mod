module github.com/zhamlin/routey

go 1.24

replace github.com/sv-tools/openapi v1.1.0 => github.com/zhamlin/go-openapi v0.0.0-20250612073337-718e9eb6ac95

require (
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.2
	github.com/sv-tools/openapi v1.1.0
)

// test dependencies
require github.com/nsf/jsondiff v0.0.0-20230430225905-43f6cf3098c1

require golang.org/x/text v0.14.0 // indirect
