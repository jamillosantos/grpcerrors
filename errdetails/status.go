package errdetails

import "google.golang.org/grpc/codes"

type CodeDetail struct {
	Code codes.Code
}

func (c *CodeDetail) GRPCCode() codes.Code {
	return c.Code
}
