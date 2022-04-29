package grpcerrors

import (
	"testing"

	"github.com/magiconair/properties/assert"
	"google.golang.org/grpc/codes"
)

func TestCode(t *testing.T) {
	wantCode := codes.Internal
	got := Code(wantCode)
	assert.Equal(t, wantCode, got.Code)
}
