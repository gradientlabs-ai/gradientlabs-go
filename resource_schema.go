package client

import "encoding/json"

// Schema of data related to a resource.
type Schema struct {
	// Raw is the raw schema data.
	// - inferred from the data payloads
	// - updated asynchronously as data is fetched
	// - can be unmarshalled into a jsonschema.Schema (use Parse method)
	Raw json.RawMessage `json:"raw"`

	// Attributes is a list of attributes derived from the schema. Only leaf properties are included in this list,
	// container objects and arrays themselves are not included as separate attributes.
	//
	// For example, given a schema with:
	//   {
	//     "properties": {
	//       "name": { "type": "string" },
	//       "address": {
	//         "type": "object",
	//         "properties": {
	//           "street": { "type": "string" },
	//           "city": { "type": "string" }
	//         }
	//       },
	//       "skills": {
	//         "type": "array",
	//         "items": {
	//           "type": "object",
	//           "properties": {
	//             "name": { "type": "string" },
	//             "level": { "type": "string" }
	//           }
	//         }
	//       }
	//     }
	//   }
	//
	// The resulting attributes would be:
	//   - $.name (string)
	//   - $.address.street (string)
	//   - $.address.city (string)
	//   - $.skills[*].name (string)
	//   - $.skills[*].level (string)
	//
	// Note that $.address and $.skills are not included as separate attributes.
	// Attributes are sorted alphabetically by JSONPath.
	Attributes []Attribute `json:"attributes"`
}

type Attribute struct {
	// Path expression to read this attribute from a resource payload.
	Path string `json:"path"`

	// Type describes the data type of the attribute.
	Type AttributeType `json:"type"`

	// Cardinality determines how many values would be returned when applying the
	// given JSONPath. Only attributes with a cardinality of "one" can be used as
	// tool parameter values.
	Cardinality AttributeCardinality `json:"cardinality"`

	// Description is an optional human-readable description of the attribute.
	Description string `json:"description"`

	// IsTopLevel is true if the attribute is a top-level field (direct child of the root).
	// The API sends it as `is_root`.
	IsTopLevel bool `json:"is_root"`

	// Name is the name of the field, e.g.:
	//   - "id" for `$.id`
	//   - "quantity" for `$.items[*].quantity`
	Name string `json:"name"`
}

// AttributeCardinality determines how many values would be returned when
// applying an attribute's JSONPath to a resource payload.
type AttributeCardinality string

const (
	// AttributeCardinalityOne means the attribute represents one value (e.g.
	// `$.id` or `$.name`).
	AttributeCardinalityOne AttributeCardinality = "one"

	// AttributeCardinalityMany means the attribute represents many values (e.g.
	// `$.items[*].quantity`).
	AttributeCardinalityMany AttributeCardinality = "many"
)

// AttributeType describes the data type of the attribute.
type AttributeType string

const (
	// AttributeTypeString means the attribute is a string.
	AttributeTypeString AttributeType = "string"

	// AttributeTypeDate means the attribute is a date in the string form:
	// YYYY-MM-DD.
	AttributeTypeDate AttributeType = "date"

	// AttributeTypeTimestamp means the attribute is a timestamp in the RFC3339 form.
	AttributeTypeTimestamp AttributeType = "timestamp"

	// AttributeTypeBoolean means the attribute is a boolean value.
	AttributeTypeBoolean AttributeType = "boolean"

	// AttributeTypeNumber means the attribute is a numeric value. JSON doesn't
	// differentiate between integers and floating point numbers, so it's up to
	// the user to interpret this.
	AttributeTypeNumber AttributeType = "number"

	// AttributeTypeArray means the attribute is an array of primitive values.
	//
	// Note: an array of primitive values will still have a cardinality of "one"
	// because there is one value available (the array itself).
	AttributeTypeArray AttributeType = "array"

	// AttributeTypeComplex means the attribute could be many types, which we don't
	// support yet.
	AttributeTypeComplex AttributeType = "complex"
)
