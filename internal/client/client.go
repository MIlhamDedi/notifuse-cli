package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type Client struct {
	baseURL string
	apiKey  string
	http    HTTPDoer
}

type Request struct {
	Method      string
	Path        string
	Query       url.Values
	Body        []byte
	ContentType string
}

type Response struct {
	Status int
	Body   []byte
}

func New(baseURL string, apiKey string, httpClient HTTPDoer) (*Client, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("empty endpoint")
	}
	if _, err := url.ParseRequestURI(baseURL); err != nil {
		return nil, fmt.Errorf("invalid endpoint %q: %w", baseURL, err)
	}
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("empty API key")
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{baseURL: baseURL, apiKey: apiKey, http: httpClient}, nil
}

func (c *Client) Do(ctx context.Context, request Request) (Response, error) {
	target, err := url.Parse(c.baseURL + request.Path)
	if err != nil {
		return Response{}, err
	}
	if len(request.Query) > 0 {
		target.RawQuery = request.Query.Encode()
	}
	var body io.Reader
	if request.Body != nil {
		body = bytes.NewReader(request.Body)
	}
	httpRequest, err := http.NewRequestWithContext(ctx, request.Method, target.String(), body)
	if err != nil {
		return Response{}, err
	}
	httpRequest.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpRequest.Header.Set("Accept", "application/json")
	if request.ContentType != "" {
		httpRequest.Header.Set("Content-Type", request.ContentType)
	}
	response, err := c.http.Do(httpRequest)
	if err != nil {
		return Response{}, err
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return Response{}, err
	}
	return Response{Status: response.StatusCode, Body: responseBody}, nil
}
