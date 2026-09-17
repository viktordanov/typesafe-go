package typesafe

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"
)

type SystemOneRequest struct {
	State     any // text or any JSON-encodable value
	Questions Questions
	Model     string // overrides WithModel and the client default
}

// Validate reports every problem with the request found without sending it.
func (r SystemOneRequest) Validate() error {
	_, _, err := r.encode(DefaultModel)
	return err
}

type wireQuestion struct {
	Type         string `json:"type"`
	Instructions any    `json:"instructions,omitempty"`
	Criteria     any    `json:"criteria,omitempty"`
}

func (r SystemOneRequest) encode(model string) (map[string]wireQuestion, []byte, error) {
	var errs []error
	if len(r.Questions) == 0 {
		errs = append(errs, invalid("questions", "at least one question is required"))
	}
	questions := make(map[string]wireQuestion, len(r.Questions))
	for _, name := range slices.Sorted(maps.Keys(r.Questions)) {
		path := fmt.Sprintf("questions[%q]", name)
		if strings.TrimSpace(name) == "" {
			errs = append(errs, invalid(path, "name is blank"))
		}
		switch q := r.Questions[name].(type) {
		case Noul:
			w := wireQuestion{Type: "noul", Instructions: q.Instructions}
			if q.Criteria != nil {
				w.Criteria = q.Criteria
			}
			questions[name] = w
		case Choice:
			if n := len(q.Criteria); n < 1 || n > 255 {
				errs = append(errs, invalid(path+".criteria", "needs 1 to 255 labels, got %d", n))
			}
			for label := range q.Criteria {
				if strings.TrimSpace(label) == "" {
					errs = append(errs, invalid(path+".criteria", "label %q is blank", label))
				}
			}
			questions[name] = wireQuestion{Type: "choice", Instructions: q.Instructions, Criteria: q.Criteria}
		case Score:
			if n := len(q.Criteria); n < 2 || n > 10 {
				errs = append(errs, invalid(path+".criteria", "needs 2 to 10 levels, got %d", n))
			}
			questions[name] = wireQuestion{Type: "score", Instructions: q.Instructions, Criteria: q.Criteria}
		default:
			errs = append(errs, invalid(path, "must be a Noul, Choice, or Score, got %T", q))
		}
	}

	body, err := json.Marshal(struct {
		State     any                     `json:"state"`
		Model     string                  `json:"model"`
		Questions map[string]wireQuestion `json:"questions"`
	}{r.State, model, questions})
	if err != nil {
		errs = append(errs, &ValidationError{Err: err})
	}
	return questions, body, errors.Join(errs...)
}

// SystemOne answers the request's questions. Invalid requests return errors
// matching ErrInvalidRequest without sending anything.
func (c *Client) SystemOne(ctx context.Context, request SystemOneRequest, options ...RequestOption) (*SystemOneResult, error) {
	o, optErr := applyRequestOptions(options)
	model := strings.TrimSpace(request.Model)
	if model != "" && o.model != "" && model != o.model {
		optErr = errors.Join(optErr, invalid("model", "%q conflicts with WithModel(%q)", model, o.model))
	}
	questions, body, err := request.encode(cmp.Or(model, o.model, c.defaultModel))
	if err := errors.Join(optErr, err); err != nil {
		return nil, err
	}

	resp, err := c.send(ctx, http.MethodPost, "v1/systemone", body, o)
	if err != nil {
		return nil, err
	}
	return decodeSystemOneResult(resp, questions)
}
