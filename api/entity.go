package api

import "encoding/json"

type Entity struct {
	ID          string          `json:"id"`
	Data        json.RawMessage `json:"data,omitempty"`
	Properties  []TypeValue     `json:"properties,omitempty"`
	Constraints []TypeValue     `json:"constraints,omitempty"`
}

type TypeValue struct {
	Type  string          `json:"type"`
	Value json.RawMessage `json:"value"`
}
