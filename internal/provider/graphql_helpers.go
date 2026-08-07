package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"golang.org/x/oauth2"
)

type graphQLError struct {
	Message string `json:"message"`
}

type graphQLResponse[T any] struct {
	Data   T              `json:"data"`
	Errors []graphQLError `json:"errors"`
}

func newConfiguredHTTPClient(config *LogScaleConfig) *http.Client {
	src := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: config.APIToken})
	return oauth2.NewClient(context.Background(), src)
}

func doGraphQLRequest(ctx context.Context, client *http.Client, endpoint string, query string, variables map[string]interface{}) ([]byte, error) {
	payload := map[string]interface{}{
		"query": query,
	}
	if variables != nil {
		payload["variables"] = variables
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("could not marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("could not create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("could not execute HTTP request: %w", err)
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read response body: %w", err)
	}

	return body, nil
}

func executeGraphQL[T any](ctx context.Context, client *http.Client, endpoint string, query string, variables map[string]interface{}, out *T) error {
	body, err := doGraphQLRequest(ctx, client, endpoint, query, variables)
	if err != nil {
		return err
	}

	var gqlResp graphQLResponse[T]
	if err := json.Unmarshal(body, &gqlResp); err != nil {
		return fmt.Errorf("could not unmarshal response: %w\nBody: %s", err, body)
	}

	if len(gqlResp.Errors) > 0 {
		return fmt.Errorf("GraphQL error: %s", formatGraphQLErrors(gqlResp.Errors))
	}

	*out = gqlResp.Data
	return nil
}

// executeGraphQLLenient is like executeGraphQL but treats GraphQL errors as
// warnings when the response also contains data. Used for mutations where the
// API may return ancillary permission errors alongside a successfully created
// resource (e.g. "Manage cluster not allowed" on executionInfo access even
// though the create itself succeeded). Returns the data and any warning
// messages; returns a hard error only when no data was returned at all.
func executeGraphQLLenient[T any](ctx context.Context, client *http.Client, endpoint string, query string, variables map[string]interface{}, out *T) ([]string, error) {
	body, err := doGraphQLRequest(ctx, client, endpoint, query, variables)
	if err != nil {
		return nil, err
	}

	var gqlResp graphQLResponse[T]
	if err := json.Unmarshal(body, &gqlResp); err != nil {
		return nil, fmt.Errorf("could not unmarshal response: %w\nBody: %s", err, body)
	}

	var warnings []string
	for _, e := range gqlResp.Errors {
		if e.Message != "" {
			warnings = append(warnings, e.Message)
		}
	}

	*out = gqlResp.Data
	return warnings, nil
}

func formatGraphQLErrors(errors []graphQLError) string {
	messages := make([]string, 0, len(errors))
	for _, gqlErr := range errors {
		if gqlErr.Message != "" {
			messages = append(messages, gqlErr.Message)
		}
	}

	return strings.Join(messages, "; ")
}
