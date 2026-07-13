package openapi

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type Document struct {
	Info  Info                            `json:"info"`
	Paths map[string]map[string]Operation `json:"paths"`
}

type Info struct {
	Title   string `json:"title"`
	Version string `json:"version"`
}

type Operation struct {
	OperationID string `json:"operationId"`
	Summary     string `json:"summary"`
}

type Entry struct {
	Method      string `json:"method"`
	Path        string `json:"path"`
	OperationID string `json:"operation_id,omitempty"`
	Summary     string `json:"summary,omitempty"`
}

func Load(spec []byte) (*Document, error) {
	var doc Document
	if err := json.Unmarshal(spec, &doc); err != nil {
		return nil, err
	}
	if len(doc.Paths) == 0 {
		return nil, fmt.Errorf("openapi document has no paths")
	}
	return &doc, nil
}

func (d *Document) Entries() []Entry {
	var entries []Entry
	for path, methods := range d.Paths {
		for method, operation := range methods {
			entries = append(entries, Entry{
				Method:      strings.ToUpper(method),
				Path:        path,
				OperationID: operation.OperationID,
				Summary:     operation.Summary,
			})
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Path == entries[j].Path {
			return entries[i].Method < entries[j].Method
		}
		return entries[i].Path < entries[j].Path
	})
	return entries
}
