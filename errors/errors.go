package storage

import (
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var ErrDetailsDiscontinuity = &errdetails.ErrorInfo{Reason: "DISCONTINUITY"}

func IsDiscontinuity(err error) bool {
	stat := status.Convert(err)
	if stat.Code() == codes.OK {
		return false
	}
	for _, detail := range stat.Details() {
		if info, ok := detail.(*errdetails.ErrorInfo); ok && info.Reason == "DISCONTINUITY" {
			return true
		}
	}
	return false
}
