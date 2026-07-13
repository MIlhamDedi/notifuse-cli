package input

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

func JSON(inline string, file string, stdin io.Reader) ([]byte, error) {
	if inline != "" && file != "" {
		return nil, fmt.Errorf("choose only one of --body-json or --body-file")
	}
	var body []byte
	var err error
	switch {
	case inline != "":
		body = []byte(inline)
	case file == "-":
		body, err = io.ReadAll(stdin)
	case file != "":
		body, err = os.ReadFile(file)
	default:
		return nil, fmt.Errorf("request body is required; pass --body-json or --body-file")
	}
	if err != nil {
		return nil, err
	}
	trimmed := bytes.TrimSpace(body)
	if !json.Valid(trimmed) {
		return nil, fmt.Errorf("request body is not valid JSON")
	}
	return trimmed, nil
}

func Object(body []byte) (map[string]any, error) {
	var object map[string]any
	if err := json.Unmarshal(body, &object); err != nil {
		return nil, err
	}
	return object, nil
}
