package jsonschema

import "github.com/sv-tools/openapi"

type Type string

const (
	TypeString  Type = openapi.StringType
	TypeNumber  Type = openapi.NumberType
	TypeInteger Type = openapi.IntegerType
	TypeObject  Type = openapi.ObjectType
	TypeArray   Type = openapi.ArrayType
	TypeBoolean Type = openapi.BooleanType
	TypeNull    Type = openapi.NullType
)
