package errorsgrpc

import (
	"google.golang.org/grpc/codes"

	"github.com/jamillosantos/grpcerrors/errdetails"
)

func Code(code codes.Code) *errdetails.CodeDetail {
	return &errdetails.CodeDetail{
		Code: code,
	}
}
