# gojsonschemalite

`gojsonschemalite` is a lightweight, treeshaked fork of `github.com/xeipuuv/gojsonschema` for the Go programming language.

## Key Differences from original `gojsonschema`

* **Draft-04 Only**: All support for Draft-06, Draft-07, and hybrid schema detection is removed.
* **No `$ref` or `$id` resolution**: Cross-referencing schemas via `$ref` is fully removed. Every schema is parsed and evaluated standalone.
* **No File or HTTP Fetching**: Removed all file/URL loading mechanisms (e.g., `file://`, `http://`).
* **Minimal Loaders**: Supports exactly two loaders:
  * `NewBytesLoader([]byte)` for raw JSON inputs.
  * `NewRawLoader(interface{})` for pre-unmarshaled Go structures, maps, or slices.
* **Zero Runtime Dependencies**: Stripped out external dependencies like `gojsonreference` and `gojsonpointer`. The library has zero third-party dependencies in production.

## Installation

```bash
go get github.com/swwind/gojsonschemalite
```

## Usage

### Example

```go
package main

import (
	"fmt"
	"github.com/swwind/gojsonschemalite"
)

func main() {
	schemaJSON := []byte(`{
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer", "minimum": 0}
		},
		"required": ["name"]
	}`)

	documentJSON := []byte(`{
		"name": "Alice",
		"age": 30
	}`)

	schemaLoader := gojsonschema.NewBytesLoader(schemaJSON)
	documentLoader := gojsonschema.NewBytesLoader(documentJSON)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		panic(err.Error())
	}

	if result.Valid() {
		fmt.Println("The document is valid")
	} else {
		fmt.Println("The document is not valid. See errors:")
		for _, desc := range result.Errors() {
			fmt.Printf("- %s\n", desc)
		}
	}
}
```

### Loaders

`gojsonschemalite` provides only two ways to load schemas and documents:

* **Raw JSON bytes**:
```go
loader := gojsonschema.NewBytesLoader([]byte(`{"type": "string"}`))
```

* **Custom Go types / objects**:
```go
data := map[string]interface{}{"name": "Alice"}
loader := gojsonschema.NewRawLoader(data)
```
