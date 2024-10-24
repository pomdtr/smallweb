//go:build go1.22

// Package api provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/oapi-codegen/oapi-codegen/v2 version v2.4.0 DO NOT EDIT.
package api

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/oapi-codegen/runtime"
)

// App defines model for App.
type App struct {
	Manifest *map[string]interface{} `json:"manifest,omitempty"`
	Name     string                  `json:"name"`
	Url      string                  `json:"url"`
}

// CommandOutput defines model for CommandOutput.
type CommandOutput struct {
	Code    int    `json:"code"`
	Stderr  string `json:"stderr"`
	Stdout  string `json:"stdout"`
	Success bool   `json:"success"`
}

// GetConsoleLogsParams defines parameters for GetConsoleLogs.
type GetConsoleLogsParams struct {
	// App Filter logs by app
	App *string `form:"app,omitempty" json:"app,omitempty"`
}

// GetHttpLogsParams defines parameters for GetHttpLogs.
type GetHttpLogsParams struct {
	// Host Filter logs by host
	Host *string `form:"host,omitempty" json:"host,omitempty"`
}

// RunAppJSONBody defines parameters for RunApp.
type RunAppJSONBody struct {
	Args []string `json:"args"`
}

// RunAppJSONRequestBody defines body for RunApp for application/json ContentType.
type RunAppJSONRequestBody RunAppJSONBody

// RequestEditorFn  is the function signature for the RequestEditor callback function
type RequestEditorFn func(ctx context.Context, req *http.Request) error

// Doer performs HTTP requests.
//
// The standard http.Client implements this interface.
type HttpRequestDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client which conforms to the OpenAPI3 specification for this service.
type Client struct {
	// The endpoint of the server conforming to this interface, with scheme,
	// https://api.deepmap.com for example. This can contain a path relative
	// to the server, such as https://api.deepmap.com/dev-test, and all the
	// paths in the swagger spec will be appended to the server.
	Server string

	// Doer for performing requests, typically a *http.Client with any
	// customized settings, such as certificate chains.
	Client HttpRequestDoer

	// A list of callbacks for modifying requests which are generated before sending over
	// the network.
	RequestEditors []RequestEditorFn
}

// ClientOption allows setting custom parameters during construction
type ClientOption func(*Client) error

// Creates a new Client, with reasonable defaults
func NewClient(server string, opts ...ClientOption) (*Client, error) {
	// create a client with sane default values
	client := Client{
		Server: server,
	}
	// mutate client and add all optional params
	for _, o := range opts {
		if err := o(&client); err != nil {
			return nil, err
		}
	}
	// ensure the server URL always has a trailing slash
	if !strings.HasSuffix(client.Server, "/") {
		client.Server += "/"
	}
	// create httpClient, if not already present
	if client.Client == nil {
		client.Client = &http.Client{}
	}
	return &client, nil
}

// WithHTTPClient allows overriding the default Doer, which is
// automatically created using http.Client. This is useful for tests.
func WithHTTPClient(doer HttpRequestDoer) ClientOption {
	return func(c *Client) error {
		c.Client = doer
		return nil
	}
}

// WithRequestEditorFn allows setting up a callback function, which will be
// called right before sending the request. This can be used to mutate the request.
func WithRequestEditorFn(fn RequestEditorFn) ClientOption {
	return func(c *Client) error {
		c.RequestEditors = append(c.RequestEditors, fn)
		return nil
	}
}

// The interface specification for the client above.
type ClientInterface interface {
	// GetApps request
	GetApps(ctx context.Context, reqEditors ...RequestEditorFn) (*http.Response, error)

	// GetApp request
	GetApp(ctx context.Context, app string, reqEditors ...RequestEditorFn) (*http.Response, error)

	// GetConsoleLogs request
	GetConsoleLogs(ctx context.Context, params *GetConsoleLogsParams, reqEditors ...RequestEditorFn) (*http.Response, error)

	// GetHttpLogs request
	GetHttpLogs(ctx context.Context, params *GetHttpLogsParams, reqEditors ...RequestEditorFn) (*http.Response, error)

	// RunAppWithBody request with any body
	RunAppWithBody(ctx context.Context, app string, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*http.Response, error)

	RunApp(ctx context.Context, app string, body RunAppJSONRequestBody, reqEditors ...RequestEditorFn) (*http.Response, error)
}

func (c *Client) GetApps(ctx context.Context, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewGetAppsRequest(c.Server)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) GetApp(ctx context.Context, app string, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewGetAppRequest(c.Server, app)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) GetConsoleLogs(ctx context.Context, params *GetConsoleLogsParams, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewGetConsoleLogsRequest(c.Server, params)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) GetHttpLogs(ctx context.Context, params *GetHttpLogsParams, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewGetHttpLogsRequest(c.Server, params)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) RunAppWithBody(ctx context.Context, app string, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewRunAppRequestWithBody(c.Server, app, contentType, body)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) RunApp(ctx context.Context, app string, body RunAppJSONRequestBody, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewRunAppRequest(c.Server, app, body)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

// NewGetAppsRequest generates requests for GetApps
func NewGetAppsRequest(server string) (*http.Request, error) {
	var err error

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/v0/apps")
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", queryURL.String(), nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

// NewGetAppRequest generates requests for GetApp
func NewGetAppRequest(server string, app string) (*http.Request, error) {
	var err error

	var pathParam0 string

	pathParam0, err = runtime.StyleParamWithLocation("simple", false, "app", runtime.ParamLocationPath, app)
	if err != nil {
		return nil, err
	}

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/v0/apps/%s", pathParam0)
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", queryURL.String(), nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

// NewGetConsoleLogsRequest generates requests for GetConsoleLogs
func NewGetConsoleLogsRequest(server string, params *GetConsoleLogsParams) (*http.Request, error) {
	var err error

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/v0/logs/console")
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	if params != nil {
		queryValues := queryURL.Query()

		if params.App != nil {

			if queryFrag, err := runtime.StyleParamWithLocation("form", true, "app", runtime.ParamLocationQuery, *params.App); err != nil {
				return nil, err
			} else if parsed, err := url.ParseQuery(queryFrag); err != nil {
				return nil, err
			} else {
				for k, v := range parsed {
					for _, v2 := range v {
						queryValues.Add(k, v2)
					}
				}
			}

		}

		queryURL.RawQuery = queryValues.Encode()
	}

	req, err := http.NewRequest("GET", queryURL.String(), nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

// NewGetHttpLogsRequest generates requests for GetHttpLogs
func NewGetHttpLogsRequest(server string, params *GetHttpLogsParams) (*http.Request, error) {
	var err error

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/v0/logs/http")
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	if params != nil {
		queryValues := queryURL.Query()

		if params.Host != nil {

			if queryFrag, err := runtime.StyleParamWithLocation("form", true, "host", runtime.ParamLocationQuery, *params.Host); err != nil {
				return nil, err
			} else if parsed, err := url.ParseQuery(queryFrag); err != nil {
				return nil, err
			} else {
				for k, v := range parsed {
					for _, v2 := range v {
						queryValues.Add(k, v2)
					}
				}
			}

		}

		queryURL.RawQuery = queryValues.Encode()
	}

	req, err := http.NewRequest("GET", queryURL.String(), nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

// NewRunAppRequest calls the generic RunApp builder with application/json body
func NewRunAppRequest(server string, app string, body RunAppJSONRequestBody) (*http.Request, error) {
	var bodyReader io.Reader
	buf, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	bodyReader = bytes.NewReader(buf)
	return NewRunAppRequestWithBody(server, app, "application/json", bodyReader)
}

// NewRunAppRequestWithBody generates requests for RunApp with any type of body
func NewRunAppRequestWithBody(server string, app string, contentType string, body io.Reader) (*http.Request, error) {
	var err error

	var pathParam0 string

	pathParam0, err = runtime.StyleParamWithLocation("simple", false, "app", runtime.ParamLocationPath, app)
	if err != nil {
		return nil, err
	}

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/v0/run/%s", pathParam0)
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", queryURL.String(), body)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", contentType)

	return req, nil
}

func (c *Client) applyEditors(ctx context.Context, req *http.Request, additionalEditors []RequestEditorFn) error {
	for _, r := range c.RequestEditors {
		if err := r(ctx, req); err != nil {
			return err
		}
	}
	for _, r := range additionalEditors {
		if err := r(ctx, req); err != nil {
			return err
		}
	}
	return nil
}

// ClientWithResponses builds on ClientInterface to offer response payloads
type ClientWithResponses struct {
	ClientInterface
}

// NewClientWithResponses creates a new ClientWithResponses, which wraps
// Client with return type handling
func NewClientWithResponses(server string, opts ...ClientOption) (*ClientWithResponses, error) {
	client, err := NewClient(server, opts...)
	if err != nil {
		return nil, err
	}
	return &ClientWithResponses{client}, nil
}

// WithBaseURL overrides the baseURL.
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) error {
		newBaseURL, err := url.Parse(baseURL)
		if err != nil {
			return err
		}
		c.Server = newBaseURL.String()
		return nil
	}
}

// ClientWithResponsesInterface is the interface specification for the client with responses above.
type ClientWithResponsesInterface interface {
	// GetAppsWithResponse request
	GetAppsWithResponse(ctx context.Context, reqEditors ...RequestEditorFn) (*GetAppsResponse, error)

	// GetAppWithResponse request
	GetAppWithResponse(ctx context.Context, app string, reqEditors ...RequestEditorFn) (*GetAppResponse, error)

	// GetConsoleLogsWithResponse request
	GetConsoleLogsWithResponse(ctx context.Context, params *GetConsoleLogsParams, reqEditors ...RequestEditorFn) (*GetConsoleLogsResponse, error)

	// GetHttpLogsWithResponse request
	GetHttpLogsWithResponse(ctx context.Context, params *GetHttpLogsParams, reqEditors ...RequestEditorFn) (*GetHttpLogsResponse, error)

	// RunAppWithBodyWithResponse request with any body
	RunAppWithBodyWithResponse(ctx context.Context, app string, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*RunAppResponse, error)

	RunAppWithResponse(ctx context.Context, app string, body RunAppJSONRequestBody, reqEditors ...RequestEditorFn) (*RunAppResponse, error)
}

type GetAppsResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *[]App
}

// Status returns HTTPResponse.Status
func (r GetAppsResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r GetAppsResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

type GetAppResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *App
}

// Status returns HTTPResponse.Status
func (r GetAppResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r GetAppResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

type GetConsoleLogsResponse struct {
	Body         []byte
	HTTPResponse *http.Response
}

// Status returns HTTPResponse.Status
func (r GetConsoleLogsResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r GetConsoleLogsResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

type GetHttpLogsResponse struct {
	Body         []byte
	HTTPResponse *http.Response
}

// Status returns HTTPResponse.Status
func (r GetHttpLogsResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r GetHttpLogsResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

type RunAppResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *CommandOutput
}

// Status returns HTTPResponse.Status
func (r RunAppResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r RunAppResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

// GetAppsWithResponse request returning *GetAppsResponse
func (c *ClientWithResponses) GetAppsWithResponse(ctx context.Context, reqEditors ...RequestEditorFn) (*GetAppsResponse, error) {
	rsp, err := c.GetApps(ctx, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseGetAppsResponse(rsp)
}

// GetAppWithResponse request returning *GetAppResponse
func (c *ClientWithResponses) GetAppWithResponse(ctx context.Context, app string, reqEditors ...RequestEditorFn) (*GetAppResponse, error) {
	rsp, err := c.GetApp(ctx, app, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseGetAppResponse(rsp)
}

// GetConsoleLogsWithResponse request returning *GetConsoleLogsResponse
func (c *ClientWithResponses) GetConsoleLogsWithResponse(ctx context.Context, params *GetConsoleLogsParams, reqEditors ...RequestEditorFn) (*GetConsoleLogsResponse, error) {
	rsp, err := c.GetConsoleLogs(ctx, params, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseGetConsoleLogsResponse(rsp)
}

// GetHttpLogsWithResponse request returning *GetHttpLogsResponse
func (c *ClientWithResponses) GetHttpLogsWithResponse(ctx context.Context, params *GetHttpLogsParams, reqEditors ...RequestEditorFn) (*GetHttpLogsResponse, error) {
	rsp, err := c.GetHttpLogs(ctx, params, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseGetHttpLogsResponse(rsp)
}

// RunAppWithBodyWithResponse request with arbitrary body returning *RunAppResponse
func (c *ClientWithResponses) RunAppWithBodyWithResponse(ctx context.Context, app string, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*RunAppResponse, error) {
	rsp, err := c.RunAppWithBody(ctx, app, contentType, body, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseRunAppResponse(rsp)
}

func (c *ClientWithResponses) RunAppWithResponse(ctx context.Context, app string, body RunAppJSONRequestBody, reqEditors ...RequestEditorFn) (*RunAppResponse, error) {
	rsp, err := c.RunApp(ctx, app, body, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseRunAppResponse(rsp)
}

// ParseGetAppsResponse parses an HTTP response from a GetAppsWithResponse call
func ParseGetAppsResponse(rsp *http.Response) (*GetAppsResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &GetAppsResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	switch {
	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 200:
		var dest []App
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON200 = &dest

	}

	return response, nil
}

// ParseGetAppResponse parses an HTTP response from a GetAppWithResponse call
func ParseGetAppResponse(rsp *http.Response) (*GetAppResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &GetAppResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	switch {
	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 200:
		var dest App
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON200 = &dest

	}

	return response, nil
}

// ParseGetConsoleLogsResponse parses an HTTP response from a GetConsoleLogsWithResponse call
func ParseGetConsoleLogsResponse(rsp *http.Response) (*GetConsoleLogsResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &GetConsoleLogsResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	return response, nil
}

// ParseGetHttpLogsResponse parses an HTTP response from a GetHttpLogsWithResponse call
func ParseGetHttpLogsResponse(rsp *http.Response) (*GetHttpLogsResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &GetHttpLogsResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	return response, nil
}

// ParseRunAppResponse parses an HTTP response from a RunAppWithResponse call
func ParseRunAppResponse(rsp *http.Response) (*RunAppResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &RunAppResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	switch {
	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 200:
		var dest CommandOutput
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON200 = &dest

	case rsp.StatusCode == 200:
		// Content-type (text/plain) unsupported

	}

	return response, nil
}

// ServerInterface represents all server handlers.
type ServerInterface interface {

	// (GET /v0/apps)
	GetApps(w http.ResponseWriter, r *http.Request)

	// (GET /v0/apps/{app})
	GetApp(w http.ResponseWriter, r *http.Request, app string)

	// (GET /v0/logs/console)
	GetConsoleLogs(w http.ResponseWriter, r *http.Request, params GetConsoleLogsParams)

	// (GET /v0/logs/http)
	GetHttpLogs(w http.ResponseWriter, r *http.Request, params GetHttpLogsParams)

	// (POST /v0/run/{app})
	RunApp(w http.ResponseWriter, r *http.Request, app string)
}

// ServerInterfaceWrapper converts contexts to parameters.
type ServerInterfaceWrapper struct {
	Handler            ServerInterface
	HandlerMiddlewares []MiddlewareFunc
	ErrorHandlerFunc   func(w http.ResponseWriter, r *http.Request, err error)
}

type MiddlewareFunc func(http.Handler) http.Handler

// GetApps operation middleware
func (siw *ServerInterfaceWrapper) GetApps(w http.ResponseWriter, r *http.Request) {

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.GetApps(w, r)
	}))

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler.ServeHTTP(w, r)
}

// GetApp operation middleware
func (siw *ServerInterfaceWrapper) GetApp(w http.ResponseWriter, r *http.Request) {

	var err error

	// ------------- Path parameter "app" -------------
	var app string

	err = runtime.BindStyledParameterWithOptions("simple", "app", r.PathValue("app"), &app, runtime.BindStyledParameterOptions{ParamLocation: runtime.ParamLocationPath, Explode: false, Required: true})
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "app", Err: err})
		return
	}

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.GetApp(w, r, app)
	}))

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler.ServeHTTP(w, r)
}

// GetConsoleLogs operation middleware
func (siw *ServerInterfaceWrapper) GetConsoleLogs(w http.ResponseWriter, r *http.Request) {

	var err error

	// Parameter object where we will unmarshal all parameters from the context
	var params GetConsoleLogsParams

	// ------------- Optional query parameter "app" -------------

	err = runtime.BindQueryParameter("form", true, false, "app", r.URL.Query(), &params.App)
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "app", Err: err})
		return
	}

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.GetConsoleLogs(w, r, params)
	}))

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler.ServeHTTP(w, r)
}

// GetHttpLogs operation middleware
func (siw *ServerInterfaceWrapper) GetHttpLogs(w http.ResponseWriter, r *http.Request) {

	var err error

	// Parameter object where we will unmarshal all parameters from the context
	var params GetHttpLogsParams

	// ------------- Optional query parameter "host" -------------

	err = runtime.BindQueryParameter("form", true, false, "host", r.URL.Query(), &params.Host)
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "host", Err: err})
		return
	}

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.GetHttpLogs(w, r, params)
	}))

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler.ServeHTTP(w, r)
}

// RunApp operation middleware
func (siw *ServerInterfaceWrapper) RunApp(w http.ResponseWriter, r *http.Request) {

	var err error

	// ------------- Path parameter "app" -------------
	var app string

	err = runtime.BindStyledParameterWithOptions("simple", "app", r.PathValue("app"), &app, runtime.BindStyledParameterOptions{ParamLocation: runtime.ParamLocationPath, Explode: false, Required: true})
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "app", Err: err})
		return
	}

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.RunApp(w, r, app)
	}))

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler.ServeHTTP(w, r)
}

type UnescapedCookieParamError struct {
	ParamName string
	Err       error
}

func (e *UnescapedCookieParamError) Error() string {
	return fmt.Sprintf("error unescaping cookie parameter '%s'", e.ParamName)
}

func (e *UnescapedCookieParamError) Unwrap() error {
	return e.Err
}

type UnmarshalingParamError struct {
	ParamName string
	Err       error
}

func (e *UnmarshalingParamError) Error() string {
	return fmt.Sprintf("Error unmarshaling parameter %s as JSON: %s", e.ParamName, e.Err.Error())
}

func (e *UnmarshalingParamError) Unwrap() error {
	return e.Err
}

type RequiredParamError struct {
	ParamName string
}

func (e *RequiredParamError) Error() string {
	return fmt.Sprintf("Query argument %s is required, but not found", e.ParamName)
}

type RequiredHeaderError struct {
	ParamName string
	Err       error
}

func (e *RequiredHeaderError) Error() string {
	return fmt.Sprintf("Header parameter %s is required, but not found", e.ParamName)
}

func (e *RequiredHeaderError) Unwrap() error {
	return e.Err
}

type InvalidParamFormatError struct {
	ParamName string
	Err       error
}

func (e *InvalidParamFormatError) Error() string {
	return fmt.Sprintf("Invalid format for parameter %s: %s", e.ParamName, e.Err.Error())
}

func (e *InvalidParamFormatError) Unwrap() error {
	return e.Err
}

type TooManyValuesForParamError struct {
	ParamName string
	Count     int
}

func (e *TooManyValuesForParamError) Error() string {
	return fmt.Sprintf("Expected one value for %s, got %d", e.ParamName, e.Count)
}

// Handler creates http.Handler with routing matching OpenAPI spec.
func Handler(si ServerInterface) http.Handler {
	return HandlerWithOptions(si, StdHTTPServerOptions{})
}

// ServeMux is an abstraction of http.ServeMux.
type ServeMux interface {
	HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

type StdHTTPServerOptions struct {
	BaseURL          string
	BaseRouter       ServeMux
	Middlewares      []MiddlewareFunc
	ErrorHandlerFunc func(w http.ResponseWriter, r *http.Request, err error)
}

// HandlerFromMux creates http.Handler with routing matching OpenAPI spec based on the provided mux.
func HandlerFromMux(si ServerInterface, m ServeMux) http.Handler {
	return HandlerWithOptions(si, StdHTTPServerOptions{
		BaseRouter: m,
	})
}

func HandlerFromMuxWithBaseURL(si ServerInterface, m ServeMux, baseURL string) http.Handler {
	return HandlerWithOptions(si, StdHTTPServerOptions{
		BaseURL:    baseURL,
		BaseRouter: m,
	})
}

// HandlerWithOptions creates http.Handler with additional options
func HandlerWithOptions(si ServerInterface, options StdHTTPServerOptions) http.Handler {
	m := options.BaseRouter

	if m == nil {
		m = http.NewServeMux()
	}
	if options.ErrorHandlerFunc == nil {
		options.ErrorHandlerFunc = func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	}

	wrapper := ServerInterfaceWrapper{
		Handler:            si,
		HandlerMiddlewares: options.Middlewares,
		ErrorHandlerFunc:   options.ErrorHandlerFunc,
	}

	m.HandleFunc("GET "+options.BaseURL+"/v0/apps", wrapper.GetApps)
	m.HandleFunc("GET "+options.BaseURL+"/v0/apps/{app}", wrapper.GetApp)
	m.HandleFunc("GET "+options.BaseURL+"/v0/logs/console", wrapper.GetConsoleLogs)
	m.HandleFunc("GET "+options.BaseURL+"/v0/logs/http", wrapper.GetHttpLogs)
	m.HandleFunc("POST "+options.BaseURL+"/v0/run/{app}", wrapper.RunApp)

	return m
}

// Base64 encoded, gzipped, json marshaled Swagger object
var swaggerSpec = []string{

	"H4sIAAAAAAAC/9SVTW/bPAyA/4rB9z16dbDefOsKrCtQYEN7HHpQbMZRYYuqSHULAv/3gXLTxrGXYB89",
	"7GSBFL8ekfQWKuo8OXTCUG6BqzV2Jh0vvNePD+QxiMUk7IyzK2TRs2w8Qgm0fMBKoM/BmQ73FCzBukYV",
	"MbQz8j6HgI/RBqyh/DpYD3fv86nzS+o64+rPUXyUaWIV1fuxrRNsMKghS40hzObFUlOUeVWsKmTe0y2J",
	"WjRukvfuZj7k8OL1JfK0GnVh3YqSdyut6u4607bfcJldfLmGHJ4wsCUHJSw0HfLojLdQwvnZ4uwccvBG",
	"1im94mlRGO/TucFUjYIxYsld11DCFcqF6jVt9uR4IPZ+sRjAOUGXzIz3ra2SYfHAGnzXEHqygl0y/D/g",
	"Ckr4r3htneK5bwptmv6lXhOC2Qzl1shVsF6Gmm4sS0arLOWtajENK8wkuFfJrqxia7zvTxSXeATToWBQ",
	"P1uwGkYZwa4v1TfsP52EiPlegYftef+HwE5ymnK5QlEmR5C01HBRkWNq8RiUy+HKDTU8hTMO+tG2giFT",
	"z9lykw2YEr/HiGFzCPBvAaNKUN6xBDTdGNyKQmdER846kxI4jDQBd5fcpBpG8JJgDG8t4o+R+yTifwPb",
	"mlh+wu1Z9Q+DC9G9zqEnniF3G92bDuJjRJYPVG9+aQbHPwkTGh6tssnaP1hc402fzOe2ef+Gm2L830tP",
	"hN+l8K2xB15OvvVtdDrdWdXa2Q3T9z8CAAD//4ItGTEPCAAA",
}

// GetSwagger returns the content of the embedded swagger specification file
// or error if failed to decode
func decodeSpec() ([]byte, error) {
	zipped, err := base64.StdEncoding.DecodeString(strings.Join(swaggerSpec, ""))
	if err != nil {
		return nil, fmt.Errorf("error base64 decoding spec: %w", err)
	}
	zr, err := gzip.NewReader(bytes.NewReader(zipped))
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %w", err)
	}
	var buf bytes.Buffer
	_, err = buf.ReadFrom(zr)
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %w", err)
	}

	return buf.Bytes(), nil
}

var rawSpec = decodeSpecCached()

// a naive cached of a decoded swagger spec
func decodeSpecCached() func() ([]byte, error) {
	data, err := decodeSpec()
	return func() ([]byte, error) {
		return data, err
	}
}

// Constructs a synthetic filesystem for resolving external references when loading openapi specifications.
func PathToRawSpec(pathToFile string) map[string]func() ([]byte, error) {
	res := make(map[string]func() ([]byte, error))
	if len(pathToFile) > 0 {
		res[pathToFile] = rawSpec
	}

	return res
}

// GetSwagger returns the Swagger specification corresponding to the generated code
// in this file. The external references of Swagger specification are resolved.
// The logic of resolving external references is tightly connected to "import-mapping" feature.
// Externally referenced files must be embedded in the corresponding golang packages.
// Urls can be supported but this task was out of the scope.
func GetSwagger() (swagger *openapi3.T, err error) {
	resolvePath := PathToRawSpec("")

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	loader.ReadFromURIFunc = func(loader *openapi3.Loader, url *url.URL) ([]byte, error) {
		pathToFile := url.String()
		pathToFile = path.Clean(pathToFile)
		getSpec, ok := resolvePath[pathToFile]
		if !ok {
			err1 := fmt.Errorf("path not found: %s", pathToFile)
			return nil, err1
		}
		return getSpec()
	}
	var specData []byte
	specData, err = rawSpec()
	if err != nil {
		return
	}
	swagger, err = loader.LoadFromData(specData)
	if err != nil {
		return
	}
	return
}
