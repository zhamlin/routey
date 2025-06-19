package jsonschema

import "github.com/sv-tools/openapi"

type Format string

const (
	FormatInt32               Format = openapi.Int32Format
	FormatInt64               Format = openapi.Int64Format
	FormatFloat               Format = openapi.FloatFormat
	FormatDouble              Format = openapi.DoubleFormat
	FormatPassword            Format = openapi.PasswordFormat
	FormatDateTime            Format = openapi.DateTimeFormat
	FormatTime                Format = openapi.TimeFormat
	FormatDate                Format = openapi.DateFormat
	FormatDuration            Format = openapi.DurationFormat
	FormatEmail               Format = openapi.EmailFormat
	FormatIDNEmail            Format = openapi.IDNEmailFormat
	FormatHostname            Format = openapi.HostnameFormat
	FormatIDNHostname         Format = openapi.IDNHostnameFormat
	FormatIPv4                Format = openapi.IPv4Format
	FormatIPv6                Format = openapi.IPv6Format
	FormatUUID                Format = openapi.UUIDFormat
	FormatURI                 Format = openapi.URIFormat
	FormatURIReference        Format = openapi.URIReferenceFormat
	FormatIRI                 Format = openapi.IRIFormat
	FormatIRIReference        Format = openapi.IRIReferenceFormat
	FormatURITemplate         Format = openapi.URITemplateFormat
	FormatJsonPointer         Format = openapi.JsonPointerFormat
	FormatRelativeJsonPointer Format = openapi.RelativeJsonPointerFormat
	FormatRegex               Format = openapi.RegexFormat
)
