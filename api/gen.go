//go:build go1.22

// Package api provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/oapi-codegen/oapi-codegen/v2 version v2.4.0 DO NOT EDIT.
package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/oapi-codegen/runtime"
)

// Defines values for ConsoleLogLevel.
const (
	ConsoleLogLevelDEBUG ConsoleLogLevel = "DEBUG"
	ConsoleLogLevelERROR ConsoleLogLevel = "ERROR"
	ConsoleLogLevelINFO  ConsoleLogLevel = "INFO"
	ConsoleLogLevelWARN  ConsoleLogLevel = "WARN"
)

// Defines values for ConsoleLogType.
const (
	Stderr ConsoleLogType = "stderr"
	Stdout ConsoleLogType = "stdout"
)

// Defines values for CronLogLevel.
const (
	CronLogLevelDEBUG CronLogLevel = "DEBUG"
	CronLogLevelERROR CronLogLevel = "ERROR"
	CronLogLevelINFO  CronLogLevel = "INFO"
	CronLogLevelWARN  CronLogLevel = "WARN"
)

// Defines values for HttpLogLevel.
const (
	DEBUG   HttpLogLevel = "DEBUG"
	ERROR   HttpLogLevel = "ERROR"
	INFO    HttpLogLevel = "INFO"
	WARNING HttpLogLevel = "WARNING"
)

// Defines values for HttpLogRequestMethod.
const (
	DELETE  HttpLogRequestMethod = "DELETE"
	GET     HttpLogRequestMethod = "GET"
	HEAD    HttpLogRequestMethod = "HEAD"
	OPTIONS HttpLogRequestMethod = "OPTIONS"
	PATCH   HttpLogRequestMethod = "PATCH"
	POST    HttpLogRequestMethod = "POST"
	PUT     HttpLogRequestMethod = "PUT"
)

// App defines model for App.
type App struct {
	Name string `json:"name"`
	Url  string `json:"url"`
}

// Config defines model for Config.
type Config struct {
	Cert          *string            `json:"cert,omitempty"`
	CustomDomains *map[string]string `json:"customDomains,omitempty"`
	Dir           *string            `json:"dir,omitempty"`
	Domain        *string            `json:"domain,omitempty"`
	Editor        *string            `json:"editor,omitempty"`
	Email         *string            `json:"email,omitempty"`
	Env           *map[string]string `json:"env,omitempty"`
	Host          *string            `json:"host,omitempty"`
	Key           *string            `json:"key,omitempty"`
	Port          *int               `json:"port,omitempty"`
	Shell         *string            `json:"shell,omitempty"`
}

// ConsoleLog defines model for ConsoleLog.
type ConsoleLog struct {
	// App The name of the application
	App string `json:"app"`

	// Level The log level
	Level ConsoleLogLevel `json:"level"`

	// Msg The log message
	Msg string `json:"msg"`

	// Text The standard error of the command
	Text string `json:"text"`

	// Time The timestamp of the log entry
	Time time.Time      `json:"time"`
	Type ConsoleLogType `json:"type"`
}

// ConsoleLogLevel The log level
type ConsoleLogLevel string

// ConsoleLogType defines model for ConsoleLog.Type.
type ConsoleLogType string

// CronLog defines model for CronLog.
type CronLog struct {
	// App The name of the application running the cron job
	App string `json:"app"`

	// Args The arguments passed to the cron job
	Args []string `json:"args"`

	// Duration The duration of the cron job execution in milliseconds
	Duration int `json:"duration"`

	// ExitCode The exit code of the cron job
	ExitCode int `json:"exit_code"`

	// Id A unique identifier for the cron job, typically in the format 'app:job'
	Id string `json:"id"`

	// Job The name of the cron job
	Job string `json:"job"`

	// Level The log level
	Level CronLogLevel `json:"level"`

	// Msg The log message, typically including the exit code
	Msg string `json:"msg"`

	// Schedule The schedule of the cron job
	Schedule string `json:"schedule"`

	// Time The timestamp of the log entry
	Time time.Time `json:"time"`

	// Type The type of log entry, always 'cron' for this schema
	Type interface{} `json:"type"`
}

// CronLogLevel The log level
type CronLogLevel string

// HttpLog defines model for HttpLog.
type HttpLog struct {
	// Level The log level
	Level HttpLogLevel `json:"level"`

	// Msg A brief description of the logged event
	Msg     string `json:"msg"`
	Request struct {
		// Headers The headers sent with the request
		Headers map[string]string `json:"headers"`

		// Host The host component of the request URL
		Host string `json:"host"`

		// Method The HTTP method used for the request
		Method HttpLogRequestMethod `json:"method"`

		// Path The path component of the request URL
		Path string `json:"path"`

		// Url The full URL of the request
		Url string `json:"url"`
	} `json:"request"`
	Response struct {
		// Bytes The number of bytes in the response body
		Bytes int `json:"bytes"`

		// Elapsed The time taken to process the request and generate the response, in seconds
		Elapsed float32 `json:"elapsed"`

		// Status The HTTP status code of the response
		Status int `json:"status"`
	} `json:"response"`

	// Time The time when the log entry was created
	Time time.Time `json:"time"`
}

// HttpLogLevel The log level
type HttpLogLevel string

// HttpLogRequestMethod The HTTP method used for the request
type HttpLogRequestMethod string

// GetV0LogsConsoleParams defines parameters for GetV0LogsConsole.
type GetV0LogsConsoleParams struct {
	// App Filter logs by app
	App *string `form:"app,omitempty" json:"app,omitempty"`
}

// GetV0LogsCronParams defines parameters for GetV0LogsCron.
type GetV0LogsCronParams struct {
	// App Filter logs by app
	App *string `form:"app,omitempty" json:"app,omitempty"`
}

// GetV0LogsHttpParams defines parameters for GetV0LogsHttp.
type GetV0LogsHttpParams struct {
	// Host Filter logs by host
	Host *string `form:"host,omitempty" json:"host,omitempty"`
}

// PostV0RunAppJSONBody defines parameters for PostV0RunApp.
type PostV0RunAppJSONBody struct {
	Args []string `json:"args"`
}

// PostV0RunAppJSONRequestBody defines body for PostV0RunApp for application/json ContentType.
type PostV0RunAppJSONRequestBody PostV0RunAppJSONBody

// ServerInterface represents all server handlers.
type ServerInterface interface {

	// (GET /v0/apps)
	GetV0Apps(w http.ResponseWriter, r *http.Request)

	// (GET /v0/apps/{app}/config)
	GetV0AppsAppConfig(w http.ResponseWriter, r *http.Request, app string)

	// (GET /v0/apps/{app}/env)
	GetV0AppsAppEnv(w http.ResponseWriter, r *http.Request, app string)

	// (GET /v0/config)
	GetV0Config(w http.ResponseWriter, r *http.Request)

	// (GET /v0/logs/console)
	GetV0LogsConsole(w http.ResponseWriter, r *http.Request, params GetV0LogsConsoleParams)

	// (GET /v0/logs/cron)
	GetV0LogsCron(w http.ResponseWriter, r *http.Request, params GetV0LogsCronParams)

	// (GET /v0/logs/http)
	GetV0LogsHttp(w http.ResponseWriter, r *http.Request, params GetV0LogsHttpParams)

	// (POST /v0/run/{app})
	PostV0RunApp(w http.ResponseWriter, r *http.Request, app string)
}

// ServerInterfaceWrapper converts contexts to parameters.
type ServerInterfaceWrapper struct {
	Handler            ServerInterface
	HandlerMiddlewares []MiddlewareFunc
	ErrorHandlerFunc   func(w http.ResponseWriter, r *http.Request, err error)
}

type MiddlewareFunc func(http.Handler) http.Handler

// GetV0Apps operation middleware
func (siw *ServerInterfaceWrapper) GetV0Apps(w http.ResponseWriter, r *http.Request) {

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.GetV0Apps(w, r)
	}))

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler.ServeHTTP(w, r)
}

// GetV0AppsAppConfig operation middleware
func (siw *ServerInterfaceWrapper) GetV0AppsAppConfig(w http.ResponseWriter, r *http.Request) {

	var err error

	// ------------- Path parameter "app" -------------
	var app string

	err = runtime.BindStyledParameterWithOptions("simple", "app", r.PathValue("app"), &app, runtime.BindStyledParameterOptions{ParamLocation: runtime.ParamLocationPath, Explode: false, Required: true})
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "app", Err: err})
		return
	}

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.GetV0AppsAppConfig(w, r, app)
	}))

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler.ServeHTTP(w, r)
}

// GetV0AppsAppEnv operation middleware
func (siw *ServerInterfaceWrapper) GetV0AppsAppEnv(w http.ResponseWriter, r *http.Request) {

	var err error

	// ------------- Path parameter "app" -------------
	var app string

	err = runtime.BindStyledParameterWithOptions("simple", "app", r.PathValue("app"), &app, runtime.BindStyledParameterOptions{ParamLocation: runtime.ParamLocationPath, Explode: false, Required: true})
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "app", Err: err})
		return
	}

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.GetV0AppsAppEnv(w, r, app)
	}))

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler.ServeHTTP(w, r)
}

// GetV0Config operation middleware
func (siw *ServerInterfaceWrapper) GetV0Config(w http.ResponseWriter, r *http.Request) {

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.GetV0Config(w, r)
	}))

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler.ServeHTTP(w, r)
}

// GetV0LogsConsole operation middleware
func (siw *ServerInterfaceWrapper) GetV0LogsConsole(w http.ResponseWriter, r *http.Request) {

	var err error

	// Parameter object where we will unmarshal all parameters from the context
	var params GetV0LogsConsoleParams

	// ------------- Optional query parameter "app" -------------

	err = runtime.BindQueryParameter("form", true, false, "app", r.URL.Query(), &params.App)
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "app", Err: err})
		return
	}

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.GetV0LogsConsole(w, r, params)
	}))

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler.ServeHTTP(w, r)
}

// GetV0LogsCron operation middleware
func (siw *ServerInterfaceWrapper) GetV0LogsCron(w http.ResponseWriter, r *http.Request) {

	var err error

	// Parameter object where we will unmarshal all parameters from the context
	var params GetV0LogsCronParams

	// ------------- Optional query parameter "app" -------------

	err = runtime.BindQueryParameter("form", true, false, "app", r.URL.Query(), &params.App)
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "app", Err: err})
		return
	}

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.GetV0LogsCron(w, r, params)
	}))

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler.ServeHTTP(w, r)
}

// GetV0LogsHttp operation middleware
func (siw *ServerInterfaceWrapper) GetV0LogsHttp(w http.ResponseWriter, r *http.Request) {

	var err error

	// Parameter object where we will unmarshal all parameters from the context
	var params GetV0LogsHttpParams

	// ------------- Optional query parameter "host" -------------

	err = runtime.BindQueryParameter("form", true, false, "host", r.URL.Query(), &params.Host)
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "host", Err: err})
		return
	}

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.GetV0LogsHttp(w, r, params)
	}))

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler.ServeHTTP(w, r)
}

// PostV0RunApp operation middleware
func (siw *ServerInterfaceWrapper) PostV0RunApp(w http.ResponseWriter, r *http.Request) {

	var err error

	// ------------- Path parameter "app" -------------
	var app string

	err = runtime.BindStyledParameterWithOptions("simple", "app", r.PathValue("app"), &app, runtime.BindStyledParameterOptions{ParamLocation: runtime.ParamLocationPath, Explode: false, Required: true})
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "app", Err: err})
		return
	}

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.PostV0RunApp(w, r, app)
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

	m.HandleFunc("GET "+options.BaseURL+"/v0/apps", wrapper.GetV0Apps)
	m.HandleFunc("GET "+options.BaseURL+"/v0/apps/{app}/config", wrapper.GetV0AppsAppConfig)
	m.HandleFunc("GET "+options.BaseURL+"/v0/apps/{app}/env", wrapper.GetV0AppsAppEnv)
	m.HandleFunc("GET "+options.BaseURL+"/v0/config", wrapper.GetV0Config)
	m.HandleFunc("GET "+options.BaseURL+"/v0/logs/console", wrapper.GetV0LogsConsole)
	m.HandleFunc("GET "+options.BaseURL+"/v0/logs/cron", wrapper.GetV0LogsCron)
	m.HandleFunc("GET "+options.BaseURL+"/v0/logs/http", wrapper.GetV0LogsHttp)
	m.HandleFunc("POST "+options.BaseURL+"/v0/run/{app}", wrapper.PostV0RunApp)

	return m
}
