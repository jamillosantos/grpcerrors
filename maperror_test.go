package grpcerrors

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/protobuf/proto" // nolint
	jerrors "github.com/jamillosantos/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	jerrdetails "github.com/jamillosantos/errors/errdetails"
)

func Test_reasonToProto(t *testing.T) {
	t.Run("should return nil when nil is given", func(t *testing.T) {
		got := reasonToProto(nil)
		assert.Nil(t, got)
	})

	t.Run("should return a errdetails.ErrorInfo", func(t *testing.T) {
		givenReason := jerrdetails.Reason{
			Reason: "reason",
			Domain: "domain",
		}
		got := reasonToProto(&givenReason)
		assert.Equal(t, &errdetails.ErrorInfo{
			Reason: givenReason.Reason,
			Domain: givenReason.Domain,
		}, got)
	})
}

func Test_fieldValidationsToProto(t *testing.T) {
	t.Run("should return nil when nil is given", func(t *testing.T) {
		got := fieldValidationsToProto(nil)
		assert.Nil(t, got)
	})

	t.Run("should return a errdetails.BadRequest", func(t *testing.T) {
		givenReason := jerrdetails.FieldViolations{
			Violations: []*jerrdetails.FieldViolation{
				{
					Field:     "field1",
					Violation: "violation1",
				},
				{
					Field:     "field2",
					Violation: "violation2",
				},
			},
		}
		got := fieldValidationsToProto(&givenReason)
		assert.Equal(t, &errdetails.BadRequest{
			FieldViolations: []*errdetails.BadRequest_FieldViolation{
				{
					Field:       "field1",
					Description: "violation1",
				},
				{
					Field:       "field2",
					Description: "violation2",
				},
			},
		}, got)
	})
}

type testErrDetailer struct {
}

func (t *testErrDetailer) ErrorDetail() proto.Message {
	return &errdetails.RequestInfo{
		RequestId: "123",
	}
}

func Test_detailsToProto(t *testing.T) {
	givenReason := jerrdetails.Reason{
		Reason: "reason",
		Domain: "domain",
	}
	givenViolations := jerrdetails.FieldViolations{
		Violations: []*jerrdetails.FieldViolation{
			{
				Field:     "field1",
				Violation: "value1",
			},
		},
	}

	wantRequestInfo := &errdetails.RequestInfo{
		RequestId: "123",
	}
	wantReason := &errdetails.ErrorInfo{
		Reason: givenReason.Reason,
		Domain: givenReason.Domain,
	}
	wantBadRequest := &errdetails.BadRequest{
		FieldViolations: []*errdetails.BadRequest_FieldViolation{
			{
				Field:       "field1",
				Description: "value1",
			},
		},
	}
	wantDetails := []proto.Message{
		wantRequestInfo,
		wantReason,
		wantReason,
		wantBadRequest,
		wantBadRequest,
	}
	gotDetails := detailsToProto([]interface{}{
		&testErrDetailer{},
		givenReason,
		&givenReason,
		givenViolations,
		&givenViolations,
	})
	require.Len(t, gotDetails, 5)
	assert.True(t, proto.Equal(wantDetails[0], gotDetails[0]), "failed matching 0")
	assert.True(t, proto.Equal(wantDetails[1], gotDetails[1]), "failed matching 1")
	assert.True(t, proto.Equal(wantDetails[2], gotDetails[2]), "failed matching 2")
	assert.True(t, proto.Equal(wantDetails[3], gotDetails[3]), "failed matching 3")
	assert.True(t, proto.Equal(wantDetails[4], gotDetails[4]), "failed matching 4")
}

func TestMapErrorToStatus(t *testing.T) {
	wantErr := errors.New("random error")

	//wantReason := &errdetails.ErrorInfo{
	//	Reason: "reason",
	//	Domain: "domain",
	//}

	tests := []struct {
		name        string
		err         error
		wantCode    codes.Code
		wantMessage string
		wantDetails []interface{}
	}{
		{
			"should return an unknown error when a go error is given",
			jerrors.Wrap(wantErr),
			codes.Unknown,
			wantErr.Error(),
			[]interface{}{},
		},
		{
			"should return an status ignoring nil details",
			jerrors.Wrap(wantErr, nil),
			codes.Unknown,
			wantErr.Error(),
			[]interface{}{},
		},
		{
			"should return a status with a code given a code detail",
			jerrors.Wrap(wantErr, Code(codes.FailedPrecondition)),
			codes.FailedPrecondition,
			wantErr.Error(),
			[]interface{}{},
		},
		{
			"should return a status with custom errordetailer",
			jerrors.Wrap(wantErr, &testErrDetailer{}),
			codes.Unknown,
			wantErr.Error(),
			[]interface{}{
				&errdetails.RequestInfo{
					RequestId: "123",
				},
			},
		},
		{
			"should return a status with errdetails.BadRequest",
			jerrors.Wrap(wantErr, jerrors.FieldViolations().FieldViolation("field1", "value1")),
			codes.Unknown,
			wantErr.Error(),
			[]interface{}{
				&errdetails.BadRequest{
					FieldViolations: []*errdetails.BadRequest_FieldViolation{
						{
							Field:       "field1",
							Description: "value1",
						},
					},
				},
			},
		},
		{
			"should return a status with errdetails.BadRequest (value)",
			jerrors.Wrap(wantErr, *jerrors.FieldViolations().FieldViolation("field1", "value1")),
			codes.Unknown,
			wantErr.Error(),
			[]interface{}{
				&errdetails.BadRequest{
					FieldViolations: []*errdetails.BadRequest_FieldViolation{
						{
							Field:       "field1",
							Description: "value1",
						},
					},
				},
			},
		},
		{
			"should return a status with errdetails.ErrorInfo",
			jerrors.Wrap(wantErr, jerrors.Reason("reason", "domain")),
			codes.Unknown,
			wantErr.Error(),
			[]interface{}{
				&errdetails.ErrorInfo{
					Reason: "reason",
					Domain: "domain",
				},
			},
		},
		{
			"should return a status with errdetails.ErrorInfo (value)",
			jerrors.Wrap(wantErr, *jerrors.Reason("reason", "domain")),
			codes.Unknown,
			wantErr.Error(),
			[]interface{}{
				&errdetails.ErrorInfo{
					Reason: "reason",
					Domain: "domain",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MapErrorToStatus(tt.err)
			require.NoError(t, err)
			assert.Equal(t, tt.wantCode, got.Code())
			assert.Equal(t, tt.wantMessage, got.Message())
			assert.Len(t, got.Details(), len(tt.wantDetails))
			for i, m := range got.Details() {
				require.True(t, proto.Equal(m.(proto.Message), tt.wantDetails[i].(proto.Message)), "assertion failed for item %d", i)
			}
		})
	}
}

func TestMapErrorInterceptor(t *testing.T) {
	t.Run("should return a error containing a status.Status", func(t *testing.T) {
		wantReason := "reason"
		wantDomain := "domain"
		wantCode := codes.NotFound
		wantMessage := "status message"
		_, err := MapErrorInterceptor()(context.Background(), nil, nil, func(ctx context.Context, req interface{}) (interface{}, error) {
			return nil, jerrors.Wrap(errors.New(wantMessage), Code(wantCode)).WithDetails(jerrors.Reason(wantReason, wantDomain))
		})

		require.Error(t, err)

		st, ok := status.FromError(err)
		require.Truef(t, ok, "status.Status not found")
		assert.Equal(t, wantCode, st.Code())
		assert.Equal(t, wantMessage, st.Message())

		details := st.Details()
		require.Len(t, details, 1)
		errInfo := details[0].(*errdetails.ErrorInfo)
		assert.Equal(t, wantReason, errInfo.GetReason())
		assert.Equal(t, wantDomain, errInfo.GetDomain())
	})

	t.Run("should return nil", func(t *testing.T) {
		wantResponse := "fake response"
		gotResponse, err := MapErrorInterceptor()(context.Background(), nil, nil, func(ctx context.Context, req interface{}) (interface{}, error) {
			return wantResponse, nil
		})
		require.NoError(t, err)
		assert.Equal(t, wantResponse, gotResponse)
	})
}
