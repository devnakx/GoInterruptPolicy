// //go:build debug

package main

import (
	"encoding/json"
)

func PrettyPrint(data any) string {
	b, _ := json.MarshalIndent(data, "", "  ")
	return string(b)
}
