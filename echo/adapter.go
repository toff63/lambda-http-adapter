// Package echoadapter add Echo support for the library.
// Uses the core package behind the scenes and exposes the New method to
// get a new instance and Proxy method to send request to the echo.Echo
// Adapted from https://github.com/awslabs/aws-lambda-go-api-proxy/blob/19825165bd2fce09ee70ddbd1d4a9a1f710d64a4/echo/adapter.go
// to support ALB
package echoadapter

import (
	"context"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/labstack/echo"
	"github.com/toff63/lambda-http-adapter/core"
)

// EchoLambda makes it easy to send API Gateway proxy events to a echo.Echo.
// The library transforms the proxy event into an HTTP request and then
// creates a proxy response object from the http.ResponseWriter
type EchoLambda struct {
	core.RequestAccessor

	Echo *echo.Echo
}

// New creates a new instance of the EchoLambda object.
// Receives an initialized *echo.Echo object - normally created with echo.New().
// It returns the initialized instance of the EchoLambda object.
func New(e *echo.Echo) *EchoLambda {
	return &EchoLambda{Echo: e}

}

// ProxyWithContext receives context and an API Gateway proxy event,
// transforms them into an http.Request object, and sends it to the echo.Echo for routing.
// It returns a proxy response object generated from the http.ResponseWriter.
func (e *EchoLambda) ProxyWithContext(ctx context.Context, req events.ALBTargetGroupRequest) (events.ALBTargetGroupResponse, error) {
	echoRequest, err := e.EventToRequestWithContext(ctx, req)
	return e.proxyInternal(echoRequest, err)
}

func (e *EchoLambda) proxyInternal(req *http.Request, err error) (events.ALBTargetGroupResponse, error) {
	if err != nil {
		return core.TimeoutResponse(), core.NewLoggedError("Could not convert proxy event to request: %v", err)
	}

	respWriter := core.NewProxyResponseWriter()
	e.Echo.ServeHTTP(http.ResponseWriter(respWriter), req)

	proxyResponse, err := respWriter.GetProxyResponse()
	if err != nil {
		return core.TimeoutResponse(), core.NewLoggedError("Error while generating proxy response: %v", err)

	}

	return proxyResponse, nil

}
