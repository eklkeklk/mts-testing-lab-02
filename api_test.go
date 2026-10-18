package apitest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

const (
	defaultPlaygroundURL = "https://playground.nileslabs.com/api/v1"
	defaultGraphQLURL    = "https://graphql.postman-echo.com/graphql"
	defaultPassword      = "Password@123"
)

type apiClient struct {
	baseURL  string
	http     *http.Client
	identity string
}

type apiResponse struct {
	method      string
	url         string
	status      int
	requestBody []byte
	bearer      bool
	body        []byte
}

func newAPIClient(t *testing.T, baseURL string) *apiClient {
	t.Helper()

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("create cookie jar: %v", err)
	}

	return &apiClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 20 * time.Second, Jar: jar},
	}
}

func playgroundClient(t *testing.T) *apiClient {
	return newAPIClient(t, envOrDefault("PLAYGROUND_BASE_URL", defaultPlaygroundURL))
}

func graphQLClient(t *testing.T) *apiClient {
	return newAPIClient(t, envOrDefault("GRAPHQL_URL", defaultGraphQLURL))
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}

	return fallback
}

func (c *apiClient) request(t *testing.T, method, path string, payload any, token string) apiResponse {
	t.Helper()

	var body io.Reader
	var requestBody []byte

	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal %s %s request: %v", method, path, err)
		}

		requestBody = encoded
		body = bytes.NewReader(encoded)
	}

	requestURL := c.baseURL + path

	req, err := http.NewRequest(method, requestURL, body)
	if err != nil {
		t.Fatalf("create %s %s request: %v", method, requestURL, err)
	}

	req.Header.Set("Accept", "application/json")

	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if c.identity != "" {
		req.Header.Set("X-Playground-Identity", c.identity)
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, requestURL, err)
	}

	defer resp.Body.Close()

	if identity := resp.Header.Get("X-Playground-Identity"); identity != "" {
		c.identity = identity
	}

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		t.Fatalf("read %s %s response: %v", method, requestURL, err)
	}

	return apiResponse{
		method: method, url: requestURL, status: resp.StatusCode,
		requestBody: requestBody, bearer: token != "", body: responseBody,
	}
}

func (r apiResponse) expectStatus(t *testing.T, expected int) {
	t.Helper()

	if r.status != expected {
		r.failf(t, "HTTP %d, want %d", r.status, expected)
	}
}

func (r apiResponse) decode(t *testing.T, destination any) {
	t.Helper()

	if err := json.Unmarshal(r.body, destination); err != nil {
		r.failf(t, "invalid JSON: %v", err)
	}
}

func (r apiResponse) failf(t *testing.T, format string, args ...any) {
	t.Helper()

	requestBody := "<empty>"

	if len(r.requestBody) != 0 {
		requestBody = safeJSON(r.requestBody)
	}

	t.Fatalf("%s; request: %s %s, body=%s, bearer=%t; response: HTTP %d, body=%s",
		fmt.Sprintf(format, args...), r.method, r.url, requestBody, r.bearer, r.status, r.safeBody())
}

func (r apiResponse) safeBody() string {
	return safeJSON(r.body)
}

func safeJSON(body []byte) string {
	var value any

	if json.Unmarshal(body, &value) != nil {
		return string(body)
	}

	encoded, err := json.Marshal(redact(value))
	if err != nil {
		return "[unavailable]"
	}

	return string(encoded)
}

func redact(value any) any {
	switch v := value.(type) {
	case map[string]any:
		for key, item := range v {
			lower := strings.ToLower(key)

			if strings.Contains(lower, "token") || strings.Contains(lower, "password") {
				v[key] = "[redacted]"
			} else {
				v[key] = redact(item)
			}
		}
	case []any:
		for index, item := range v {
			v[index] = redact(item)
		}
	}
	return value
}

func (r apiResponse) requireID(t *testing.T, raw json.RawMessage) string {
	t.Helper()

	var id string

	if json.Unmarshal(raw, &id) == nil && strings.TrimSpace(id) != "" {
		return id
	}

	var numeric json.Number

	if json.Unmarshal(raw, &numeric) == nil && numeric.String() != "" && numeric.String() != "0" {
		return numeric.String()
	}

	r.failf(t, "response has no usable id: %s", raw)

	return ""
}

type post struct {
	ID     json.RawMessage `json:"id"`
	Title  string          `json:"title"`
	Body   string          `json:"body"`
	UserID int             `json:"user_id"`
}

func TestListPosts(t *testing.T) {
	client := playgroundClient(t)
	response := client.request(t, http.MethodGet, "/posts?_limit=3", nil, "")
	response.expectStatus(t, http.StatusOK)

	var result struct {
		Data       []post `json:"data"`
		Pagination struct {
			Page  int `json:"page"`
			Limit int `json:"limit"`
			Total int `json:"total"`
		} `json:"pagination"`
	}

	response.decode(t, &result)

	if len(result.Data) == 0 || len(result.Data) > 3 {
		response.failf(t, "expected 1–3 posts, got %d", len(result.Data))
	}

	if result.Pagination.Page != 1 || result.Pagination.Limit != 3 || result.Pagination.Total < len(result.Data) {
		response.failf(t, "unexpected pagination: page=%d limit=%d total=%d", result.Pagination.Page, result.Pagination.Limit, result.Pagination.Total)
	}

	for _, item := range result.Data {
		response.requireID(t, item.ID)
	}
}

func TestGetUserByID(t *testing.T) {
	client := playgroundClient(t)
	response := client.request(t, http.MethodGet, "/users/1", nil, "")
	response.expectStatus(t, http.StatusOK)

	var user struct {
		ID       json.RawMessage `json:"id"`
		Username string          `json:"username"`
		Name     string          `json:"name"`
		Email    string          `json:"email"`
	}

	response.decode(t, &user)

	if response.requireID(t, user.ID) != "1" || !strings.EqualFold(user.Username, "Bret") ||
		strings.TrimSpace(user.Name) == "" || strings.TrimSpace(user.Email) == "" {
		response.failf(t, "unexpected user: id=%s username=%q name=%q email=%q", user.ID, user.Username, user.Name, user.Email)
	}
}

func TestPostCreatePatchGet(t *testing.T) {
	client := playgroundClient(t)
	originalTitle := fmt.Sprintf("Go API lab %d", time.Now().UnixNano())
	originalBody := "Body must survive a title-only PATCH"

	createdResponse := client.request(t, http.MethodPost, "/posts", map[string]any{
		"title": originalTitle, "body": originalBody, "user_id": 1,
	}, "")

	createdResponse.expectStatus(t, http.StatusCreated)

	var created post

	createdResponse.decode(t, &created)
	id := createdResponse.requireID(t, created.ID)

	if created.Title != originalTitle || created.Body != originalBody || created.UserID != 1 {
		createdResponse.failf(t, "created post differs from request: title=%q body=%q user_id=%d", created.Title, created.Body, created.UserID)
	}

	updatedTitle := originalTitle + " updated"
	resourcePath := "/posts/" + url.PathEscape(id)
	patchedResponse := client.request(t, http.MethodPatch, resourcePath, map[string]string{"title": updatedTitle}, "")
	patchedResponse.expectStatus(t, http.StatusOK)

	var patched post

	patchedResponse.decode(t, &patched)

	if patchedResponse.requireID(t, patched.ID) != id || patched.Title != updatedTitle {
		patchedResponse.failf(t, "PATCH returned wrong post: id=%s title=%q", patched.ID, patched.Title)
	}

	gotResponse := client.request(t, http.MethodGet, resourcePath, nil, "")
	gotResponse.expectStatus(t, http.StatusOK)

	var got post

	gotResponse.decode(t, &got)

	if gotResponse.requireID(t, got.ID) != id || got.Title != updatedTitle || got.Body != originalBody || got.UserID != 1 {
		gotResponse.failf(t, 
			"GET did not preserve the patch and original fields: id=%s title=%q body=%q user_id=%d", 
			got.ID, got.Title, got.Body, got.UserID,
		)
	}
}

func TestLoginTokenAndCurrentUser(t *testing.T) {
	client := playgroundClient(t)
	loginResponse := client.request(t, http.MethodPost, "/auth/login", map[string]string{
		"username": "Bret", "password": envOrDefault("PLAYGROUND_PASSWORD", defaultPassword),
	}, "")
	loginResponse.expectStatus(t, http.StatusOK)

	var login struct {
		AccessToken string `json:"access_token"`
	}

	loginResponse.decode(t, &login)

	if strings.TrimSpace(login.AccessToken) == "" {
		loginResponse.failf(t, "POST /auth/login returned an empty access_token")
	}

	meResponse := client.request(t, http.MethodGet, "/auth/me", nil, login.AccessToken)
	meResponse.expectStatus(t, http.StatusOK)

	var me struct {
		ID       json.RawMessage `json:"id"`
		Username string          `json:"username"`
	}

	meResponse.decode(t, &me)

	if !strings.EqualFold(me.Username, "Bret") {
		meResponse.failf(t, "GET /auth/me returned username %q, want Bret", me.Username)
	}

	meResponse.requireID(t, me.ID)
}

func TestAuthMeRejectsMissingAndInvalidToken(t *testing.T) {
	for _, testCase := range []struct {
		name  string
		token string
	}{
		{name: "missing"},
		{name: "invalid", token: "invalid-token"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			client := playgroundClient(t)
			response := client.request(t, http.MethodGet, "/auth/me", nil, testCase.token)
			response.expectStatus(t, http.StatusUnauthorized)

			var result struct {
				Error string `json:"error"`
			}

			response.decode(t, &result)

			if !strings.Contains(strings.ToLower(result.Error), "unauthorized") {
				response.failf(t, "expected authorization error, got %q", result.Error)
			}
		})
	}
}

type graphQLError struct {
	Message string `json:"message"`
}

func TestGraphQLHelloQuery(t *testing.T) {
	client := graphQLClient(t)
	response := client.request(t, http.MethodPost, "", map[string]string{
		"query": `query Hello { hello(person: { name: "Ivan", age: 35 }) }`,
	}, "")
	response.expectStatus(t, http.StatusOK)

	var result struct {
		Data struct {
			Hello string `json:"hello"`
		} `json:"data"`
		Errors []graphQLError `json:"errors"`
	}

	response.decode(t, &result)

	if len(result.Errors) != 0 || result.Data.Hello != "Hello Ivan" {
		response.failf(t, "unexpected result: data.hello=%q errors=%v", result.Data.Hello, result.Errors)
	}
}

func TestGraphQLCreatePersonMutation(t *testing.T) {
	client := graphQLClient(t)
	response := client.request(t, http.MethodPost, "", map[string]string{
		"query": `mutation CreatePerson { createPerson(person: { name: "Student", age: 20 }) { id name age } }`,
	}, "")
	response.expectStatus(t, http.StatusOK)

	var result struct {
		Data struct {
			CreatePerson struct {
				ID   json.RawMessage `json:"id"`
				Name string          `json:"name"`
				Age  int             `json:"age"`
			} `json:"createPerson"`
		} `json:"data"`
		Errors []graphQLError `json:"errors"`
	}

	response.decode(t, &result)

	if len(result.Errors) != 0 {
		response.failf(t, "CreatePerson returned GraphQL errors: %v", result.Errors)
	}

	person := result.Data.CreatePerson
	response.requireID(t, person.ID)

	if person.Name != "Student" || person.Age != 20 {
		response.failf(t, "CreatePerson returned name=%q age=%d, want Student/20", person.Name, person.Age)
	}
}

func TestGraphQLInvalidFieldReturnsErrors(t *testing.T) {
	client := graphQLClient(t)
	response := client.request(t, http.MethodPost, "", map[string]string{
		"query": `query InvalidField { noSuchField }`,
	}, "")
	response.expectStatus(t, http.StatusOK)

	var result struct {
		Data   json.RawMessage `json:"data"`
		Errors []graphQLError  `json:"errors"`
	}

	response.decode(t, &result)

	if len(result.Errors) == 0 || !strings.Contains(result.Errors[0].Message, "noSuchField") {
		response.failf(t, "expected validation error for noSuchField")
	}
	
	if len(result.Data) != 0 && string(result.Data) != "null" {
		response.failf(t, "invalid query unexpectedly returned data: %s", result.Data)
	}
}
