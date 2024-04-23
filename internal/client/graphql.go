package client

import (
	"errors"
	"strings"
)

type graphQLRequest struct {
	operationName string
	query         string
	variables     map[string]interface{}
}

func (g *graphQLRequest) build() map[string]interface{} {
	return map[string]interface{}{
		"operationName": g.operationName,
		"query":         g.query,
		"variables":     g.variables,
	}
}

type Error struct {
	Message string   `json:"message"`
	Path    []string `json:"path"`
}

func (e *Error) Error() string {
	return e.Message + " " + strings.Join(e.Path, ".")
}

func coalesceErrors[T any](resp Response[T], err error) error {
	if err != nil {
		return err
	}
	if len(resp.Errors) > 0 {
		errs := make([]error, len(resp.Errors))
		for idx, e := range resp.Errors {
			errs[idx] = e
		}
		return errors.Join(errs...)
	}
	if resp.Data == nil {
		return errNotFound
	}
	return nil
}
