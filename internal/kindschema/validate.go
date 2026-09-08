package kindschema

import (
	"fmt"
	"net/url"
)

// ValidationError names one field-level problem with a model body.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationErrors is a list of per-field errors. An empty slice means the
// body validates. Its Error() joins the members so it satisfies error too.
type ValidationErrors []ValidationError

func (es ValidationErrors) Error() string {
	if len(es) == 0 {
		return ""
	}
	if len(es) == 1 {
		return es[0].Error()
	}
	return fmt.Sprintf("%d validation errors: %s (…and %d more)", len(es), es[0].Error(), len(es)-1)
}

// Validate walks the schema's declared fields and returns per-field errors
// for any body values that violate their declared type, required flag, enum
// membership, or number bounds. Fields not declared in the schema are left
// alone — that stays additive so existing bodies do not have to be scrubbed
// before a schema is authored.
//
// A nil/empty schema means no rules, so any body validates.
func Validate(s *Schema, body map[string]any) ValidationErrors {
	if s == nil || len(s.Fields) == 0 {
		return nil
	}
	var errs ValidationErrors
	for _, f := range s.Fields {
		v, present := body[f.Key]
		if !present || isBlank(v) {
			if f.Required {
				errs = append(errs, ValidationError{
					Field:   f.Key,
					Message: "required",
				})
			}
			continue
		}
		if e := validateOne(f, v); e != "" {
			errs = append(errs, ValidationError{Field: f.Key, Message: e})
		}
	}
	return errs
}

// isBlank reports whether v should be treated as "missing" for required-field
// purposes. nil, empty string, and empty arrays count.
func isBlank(v any) bool {
	if v == nil {
		return true
	}
	switch t := v.(type) {
	case string:
		return t == ""
	case []any:
		return len(t) == 0
	}
	return false
}

// validateOne returns an empty string on success or a short human message.
func validateOne(f Field, v any) string {
	switch f.Type {
	case TypeText, TypeURL, TypeTags:
		if f.Type == TypeURL {
			s, ok := v.(string)
			if !ok {
				return "expected a URL string"
			}
			if _, err := url.Parse(s); err != nil {
				return "invalid URL"
			}
			return ""
		}
		if _, ok := v.(string); !ok && f.Type == TypeText {
			return "expected a string"
		}
		if f.Type == TypeTags {
			if _, ok := v.([]any); !ok {
				if _, ok := v.(string); !ok {
					return "expected an array of tags or a string"
				}
			}
		}
		return ""

	case TypeNumber, TypeInteger:
		n, ok := numeric(v)
		if !ok {
			return "expected a number"
		}
		if f.Type == TypeInteger && n != float64(int64(n)) {
			return "expected a whole number"
		}
		if f.Min != nil && n < *f.Min {
			return fmt.Sprintf("must be >= %v", *f.Min)
		}
		if f.Max != nil && n > *f.Max {
			return fmt.Sprintf("must be <= %v", *f.Max)
		}
		return ""

	case TypeBoolean:
		if _, ok := v.(bool); !ok {
			return "expected true or false"
		}
		return ""

	case TypeDate:
		if _, ok := v.(string); !ok {
			return "expected an ISO date string"
		}
		return ""

	case TypeEnum:
		s, ok := v.(string)
		if !ok {
			return "expected a string"
		}
		if len(f.Values) == 0 {
			return ""
		}
		for _, allowed := range f.Values {
			if s == allowed {
				return ""
			}
		}
		return "not one of the allowed values"
	}
	return ""
}

func numeric(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	}
	return 0, false
}
