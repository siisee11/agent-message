package cmd

import (
	"encoding/json"
	"fmt"
	"io"
)

func writeJSON(w io.Writer, payload any) error {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(payload)
}

func writeTextOrJSON(rt *Runtime, plain string, payload any) error {
	if rt != nil && rt.JSONOutput {
		return writeJSON(rt.Stdout, payload)
	}
	_, err := fmt.Fprintln(rt.Stdout, plain)
	return err
}
