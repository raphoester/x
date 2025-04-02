package xgrpc

import (
	"context"
	"errors"

	"github.com/raphoester/x/xerrs"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func UnaryErrorTransformer() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {

		resp, err := handler(ctx, req)
		if err == nil {
			return resp, nil
		}

		return nil, transformError(err)
	}
}

func StreamErrorTransformer() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		err := handler(srv, stream)
		return transformError(err)
	}
}

var errorMatches = map[error]codes.Code{
	xerrs.ErrNotFound:  codes.NotFound,
	xerrs.ErrConflict:  codes.AlreadyExists,
	xerrs.ErrForbidden: codes.PermissionDenied,
}

func transformError(err error) error {
	if err == nil {
		return nil
	}

	for e, s := range errorMatches {
		if errors.Is(err, e) {
			return status.Error(s, err.Error())
		}
	}

	return err
}
