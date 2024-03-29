// Package adapter provides utility methods that help convert events
// into an http.Request and http.ResponseWriter
// The code below is adapted from https://github.com/awslabs/aws-lambda-go-api-proxy/blob/19825165bd2fce09ee70ddbd1d4a9a1f710d64a4/core/request.go
package core

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambdacontext"
)

// CustomHostVariable is the name of the environment variable that contains
// the custom hostname for the request. If this variable is not set the framework
// reverts to `DefaultServerAddress`. The value for a custom host should include
// a protocol: http://my-custom.host.com
const CustomHostVariable = "GO_API_HOST"

// DefaultServerAddress is prepended to the path of each incoming reuqest
const DefaultServerAddress = "https://aws-serverless-go-api.com"

// RequestAccessor objects give access to custom ALB properties
// in the request.
type RequestAccessor struct {
	stripBasePath string
}

// StripBasePath instructs the RequestAccessor object that the given base
// path should be removed from the request path before sending it to the
// framework for routing. This is used when API Gateway is configured with
// base path mappings in custom domain names.
// TODO check if this is still needed.
func (r *RequestAccessor) StripBasePath(basePath string) string {
	if strings.Trim(basePath, " ") == "" {
		r.stripBasePath = ""
		return ""

	}

	newBasePath := basePath
	if !strings.HasPrefix(newBasePath, "/") {
		newBasePath = "/" + newBasePath

	}

	if strings.HasSuffix(newBasePath, "/") {
		newBasePath = newBasePath[:len(newBasePath)-1]

	}
	r.stripBasePath = newBasePath
	return newBasePath

}

// EventToRequestWithContext converts an ALB event and context into an http.Request object.
// Returns the populated http request with lambda context, stage variables and ALBTargetGroupRequestContext as part of its context.
// Access those using GetALBContextFromContext and GetRuntimeContextFromContext functions in this package.
func (r *RequestAccessor) EventToRequestWithContext(ctx context.Context, req events.ALBTargetGroupRequest) (*http.Request, error) {
	httpRequest, err := r.EventToRequest(req)
	if err != nil {
		log.Println(err)
		return nil, err

	}
	return addToContext(ctx, httpRequest, req), nil

}

// EventToRequest converts an ALB event into an http.Request object.
// Returns the populated request maintaining headers
func (r *RequestAccessor) EventToRequest(req events.ALBTargetGroupRequest) (*http.Request, error) {
	decodedBody := []byte(req.Body)
	if req.IsBase64Encoded {
		base64Body, err := base64.StdEncoding.DecodeString(req.Body)
		if err != nil {
			return nil, err
		}
		decodedBody = base64Body
	}

	path := req.Path
	if r.stripBasePath != "" && len(r.stripBasePath) > 1 {
		if strings.HasPrefix(path, r.stripBasePath) {
			path = strings.Replace(path, r.stripBasePath, "", 1)

		}

	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path

	}
	serverAddress := DefaultServerAddress
	if customAddress, ok := os.LookupEnv(CustomHostVariable); ok {
		serverAddress = customAddress

	}
	path = serverAddress + path

	if len(req.MultiValueQueryStringParameters) > 0 {
		queryString := ""
		for q, l := range req.MultiValueQueryStringParameters {
			for _, v := range l {
				if queryString != "" {
					queryString += "&"

				}
				queryString += url.QueryEscape(q) + "=" + url.QueryEscape(v)

			}

		}
		path += "?" + queryString

	} else if len(req.QueryStringParameters) > 0 {
		// Support `QueryStringParameters` for backward compatibility.
		// https://github.com/awslabs/aws-lambda-go-api-proxy/issues/37
		queryString := ""
		for q := range req.QueryStringParameters {
			if queryString != "" {
				queryString += "&"

			}
			queryString += url.QueryEscape(q) + "=" + url.QueryEscape(req.QueryStringParameters[q])

		}
		path += "?" + queryString

	}

	httpRequest, err := http.NewRequest(
		strings.ToUpper(req.HTTPMethod),
		path,
		bytes.NewReader(decodedBody),
	)

	if err != nil {
		fmt.Printf("Could not convert request %s:%s to http.Request\n", req.HTTPMethod, req.Path)
		log.Println(err)
		return nil, err

	}
	for h := range req.Headers {
		httpRequest.Header.Add(h, req.Headers[h])

	}
	return httpRequest, nil
}

func addToContext(ctx context.Context, req *http.Request, albRequest events.ALBTargetGroupRequest) *http.Request {
	lc, _ := lambdacontext.FromContext(ctx)
	rc := requestContext{lambdaContext: lc, albContext: albRequest.RequestContext}
	ctx = context.WithValue(ctx, ctxKey{}, rc)
	return req.WithContext(ctx)

}

// GetALBContextFromContext retrieve ALBTargetGroupRequestContext from context.Context
func GetALBContextFromContext(ctx context.Context) (events.ALBTargetGroupRequestContext, bool) {
	v, ok := ctx.Value(ctxKey{}).(requestContext)
	return v.albContext, ok

}

// GetRuntimeContextFromContext retrieve Lambda Runtime Context from context.Context
func GetRuntimeContextFromContext(ctx context.Context) (*lambdacontext.LambdaContext, bool) {
	v, ok := ctx.Value(ctxKey{}).(requestContext)
	return v.lambdaContext, ok

}

type ctxKey struct{}

type requestContext struct {
	lambdaContext *lambdacontext.LambdaContext
	albContext    events.ALBTargetGroupRequestContext
}
