package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

func JSON(writer io.Writer, body []byte, pretty bool) error {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		_, err := fmt.Fprintln(writer, "{}")
		return err
	}
	if !pretty {
		_, err := writer.Write(append(trimmed, '\n'))
		return err
	}
	var value any
	if err := json.Unmarshal(trimmed, &value); err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	_, err = writer.Write(append(encoded, '\n'))
	return err
}
