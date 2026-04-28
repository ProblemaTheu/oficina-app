package valueobject

import "errors"

type Document struct {
	Value string
}

func NewDocument(value string) (*Document, error) {
	if len(value) < 11 {
		return nil, errors.New("documento inválido")
	}

	return &Document{Value: value}, nil
}