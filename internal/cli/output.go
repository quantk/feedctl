package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"feedctl/internal/app"
	"feedctl/internal/config"
)

type options struct {
	JSON bool
}

type errorObject struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Path    string `json:"path,omitempty"`
	Field   string `json:"field,omitempty"`
}

type envelope struct {
	OK     bool          `json:"ok"`
	Action string        `json:"action,omitempty"`
	DryRun bool          `json:"dry_run,omitempty"`
	Data   any           `json:"data,omitempty"`
	Errors []errorObject `json:"errors,omitempty"`
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func writeSuccess(w io.Writer, action string, dryRun bool, data any) error {
	return writeJSON(w, envelope{OK: true, Action: action, DryRun: dryRun, Data: data})
}

func writeError(w io.Writer, err error) error {
	var validationErrors config.ValidationErrors
	if errors.As(err, &validationErrors) {
		objects := make([]errorObject, 0, len(validationErrors))
		for _, ve := range validationErrors {
			objects = append(objects, errorObject{Code: ve.Code, Message: ve.Message, Path: ve.Path, Field: ve.Field})
		}
		return writeJSON(w, envelope{OK: false, Errors: objects})
	}
	var validationError config.ValidationError
	if errors.As(err, &validationError) {
		return writeJSON(w, envelope{OK: false, Errors: []errorObject{{Code: validationError.Code, Message: validationError.Message, Path: validationError.Path, Field: validationError.Field}}})
	}
	code := "error"
	message := err.Error()
	var appErr app.Error
	if errors.As(err, &appErr) {
		code = appErr.Code
		if appErr.Message != "" {
			message = appErr.Message
		}
	}
	return writeJSON(w, envelope{OK: false, Errors: []errorObject{{Code: code, Message: message}}})
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var appErr app.Error
	if errors.As(err, &appErr) {
		switch appErr.Code {
		case "source-not-found", "item-not-found":
			return 4
		case "invalid-url", "invalid-source-id", "invalid-duration", "duplicate-source-id", "forbidden-runtime-field":
			return 2
		case "unsupported-source-type":
			return 3
		}
	}
	return 1
}

func plainBool(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func splitTags(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func printLines(w io.Writer, lines ...string) error {
	for _, line := range lines {
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}
	return nil
}
