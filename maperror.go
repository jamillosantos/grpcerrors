package grpcerrors

import (
	"context"
	"errors"

	"github.com/golang/protobuf/proto" //nolint
	jerrdetails "github.com/jamillosantos/errors/errdetails"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type grpcCoder interface {
	GRPCCode() codes.Code
}

type errorDetailer interface {
	ErrorDetail() proto.Message
}

func MapErrorToStatus(originalError error) (*status.Status, error) {
	// Check if err is already a Status...
	if st, ok := status.FromError(originalError); ok {
		return st, nil
	}

	var (
		code       = codes.Unknown
		codeSet    bool
		message    string
		messageSet bool

		errDetails = make([]proto.Message, 0)
	)

	err := originalError
	for err != nil {
		// When we have an error in the chain that can set the code of the status.Status.
		if coder, ok := err.(grpcCoder); !codeSet && ok {
			code = coder.GRPCCode()
			codeSet = true
		}

		if messenger, ok := err.(interface {
			GRPCMessage() string
		}); !messageSet && ok {
			message = messenger.GRPCMessage()
			messageSet = true
		}

		if detailsGetter, ok := err.(interface {
			Details() []interface{}
		}); !messageSet && ok {
			ds := detailsGetter.Details()
			for _, d := range ds {
				// When we get a error detail that can set the code of the status.Status
				if dd, ok := d.(grpcCoder); !codeSet && ok {
					code = dd.GRPCCode()
					codeSet = true
				}
			}

			d := detailsToProto(ds)
			if len(d) > 0 {
				errDetails = append(errDetails, d...)
			}
		}

		if errDetailer, ok := err.(errorDetailer); ok {
			if d := errDetailer.ErrorDetail(); d != nil {
				errDetails = append(errDetails, d)
			}
		}

		err = errors.Unwrap(err)
	}

	if !messageSet {
		message = originalError.Error()
	}

	st := status.New(code, message)
	if len(errDetails) > 0 {
		return st.WithDetails(errDetails...)
	}
	return st, nil
}

func detailsToProto(details []interface{}) []proto.Message {
	r := make([]proto.Message, 0)
	for _, errDetail := range details {
		if errDetail == nil {
			continue
		}
		switch d := errDetail.(type) {
		case errorDetailer:
			if dd := d.ErrorDetail(); d != nil {
				r = append(r, dd)
			}
		case jerrdetails.FieldViolations:
			r = append(r, fieldValidationsToProto(&d))
		case *jerrdetails.FieldViolations: // nolint
			r = append(r, fieldValidationsToProto(d))
		case jerrdetails.Reason:
			r = append(r, reasonToProto(&d))
		case *jerrdetails.Reason: // nolint
			r = append(r, reasonToProto(d))
		}
	}
	return r
}

func reasonToProto(d *jerrdetails.Reason) proto.Message {
	if d == nil {
		return nil
	}
	return &errdetails.ErrorInfo{
		Reason: d.Reason,
		Domain: d.Domain,
	}
}

func fieldValidationsToProto(d *jerrdetails.FieldViolations) proto.Message {
	if d == nil {
		return nil
	}
	r := &errdetails.BadRequest{
		FieldViolations: make([]*errdetails.BadRequest_FieldViolation, len(d.Violations)),
	}
	for i, v := range d.Violations {
		r.FieldViolations[i] = &errdetails.BadRequest_FieldViolation{
			Field:       v.Field,
			Description: v.Violation,
		}
	}
	return r
}

// MapErrorInterceptor returns a new unary server interceptors that performs request rate limiting.
func MapErrorInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		resp, err := handler(ctx, req)
		if err != nil {
			st, err := MapErrorToStatus(err)
			if err != nil {
				// TODO Add callback to proper report this error.
				return resp, st.Err()
			}
			return resp, st.Err()
		}
		return resp, nil
	}
}
