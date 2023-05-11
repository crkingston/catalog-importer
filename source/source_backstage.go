package source

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	kitlog "github.com/go-kit/kit/log"
	"github.com/go-ozzo/ozzo-validation/is"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/pkg/errors"
)

type SourceBackstage struct {
	Endpoint string     `json:"endpoint"` // https://backstage.company.io/api/catalog/entities
	Token    Credential `json:"token"`
}

func (s SourceBackstage) Validate() error {
	return validation.ValidateStruct(&s,
		validation.Field(&s.Endpoint,
			validation.Required.Error("must provide an endpoint for fetching Backstage entries"),
			is.URL,
		),
	)
}

func (s SourceBackstage) String() string {
	return fmt.Sprintf("backstage (endpoint=%s)", s.Endpoint)
}

func (s SourceBackstage) Load(ctx context.Context, logger kitlog.Logger) ([]*SourceEntry, error) {
	client := cleanhttp.DefaultClient()

	var (
		limit  = 100
		offset = 0
	)

	entries := []*SourceEntry{}
	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.Endpoint+fmt.Sprintf("?limit=%d&offset=%d", limit, offset), nil)
		if err != nil {
			return nil, errors.Wrap(err, "building Backstage URL")
		}
		if string(s.Token) != "" {
			req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", string(s.Token)))
		}

		resp, err := client.Do(req)
		if err == nil && resp.StatusCode != http.StatusOK {
			err = fmt.Errorf("received error from Backstage: %s", resp.Status)
		}
		if err != nil {
			return nil, errors.Wrap(err, "fetching Backstage entries")
		}

		page := []json.RawMessage{}
		if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
			return nil, errors.Wrap(err, "parsing Backstage entries")
		}

		if len(page) == 0 {
			return entries, nil
		}

		for idx := range page {
			entries = append(entries, &SourceEntry{
				Origin:  s.String(),
				Content: page[idx],
			})
		}

		offset += len(page)
	}
}