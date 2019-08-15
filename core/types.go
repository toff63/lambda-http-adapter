package core

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/aws/aws-lambda-go/events"
)

// TimeoutResponse returns a dafault Gateway Timeout (504) response
func TimeoutResponse() events.ALBTargetGroupResponse {
	return events.ALBTargetGroupResponse{StatusCode: http.StatusGatewayTimeout, StatusDescription: strconv.Itoa(http.StatusGatewayTimeout)}
}

// NewLoggedError generates a new error and logs it to stdout
func NewLoggedError(format string, a ...interface{}) error {
	err := fmt.Errorf(format, a...)
	fmt.Println(err.Error())
	return err

}
