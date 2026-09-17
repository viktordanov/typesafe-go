package typesafe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

type Model struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ReleaseDate string `json:"release_date"`
}

// ListModels returns the models available to the account.
func (c *Client) ListModels(ctx context.Context, options ...RequestOption) ([]Model, error) {
	o, err := applyRequestOptions(options)
	if o.model != "" {
		err = errors.Join(err, invalid("options", "WithModel does not apply to ListModels"))
	}
	if err != nil {
		return nil, err
	}

	resp, err := c.send(ctx, http.MethodGet, "v1/models", nil, o)
	if err != nil {
		return nil, err
	}
	var body struct{ Models []Model }
	if err := json.Unmarshal(resp.body, &body); err != nil {
		return nil, resp.invalid("", err)
	}
	if body.Models == nil {
		return nil, resp.invalid("models", errors.New("is missing"))
	}
	for i, m := range body.Models {
		if m.Name == "" {
			return nil, resp.invalid(fmt.Sprintf("models[%d].name", i), errors.New("is missing"))
		}
	}
	return body.Models, nil
}
