package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

const maxRequestBodySize = 1 << 20 // 1 MiB

type errorResponse struct {
	Error string `json:"error"`
	Field string `json:"field,omitempty"`
}

func decodeJSON(
	w http.ResponseWriter,
	r *http.Request,
	destination any,
) error {
	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		maxRequestBodySize,
	)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("decode JSON: %w", err)
	}

	var extra any

	err := decoder.Decode(&extra)
	if !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New(
				"request body must contain a single JSON value",
			)
		}

		return fmt.Errorf("decode trailing JSON: %w", err)
	}

	return nil
}

func writeJSON(
	w http.ResponseWriter,
	status int,
	value any,
) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal JSON response: %w", err)
	}

	payload = append(payload, '\n')

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if _, err := w.Write(payload); err != nil {
		return fmt.Errorf("write JSON response: %w", err)
	}

	return nil
}

func writeError(
	w http.ResponseWriter,
	status int,
	message string,
	field string,
) error {
	return writeJSON(
		w,
		status,
		errorResponse{
			Error: message,
			Field: field,
		},
	)
}
