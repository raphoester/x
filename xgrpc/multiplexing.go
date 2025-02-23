package xgrpc

import (
	"net/http"
	"strings"

	"github.com/improbable-eng/grpc-web/go/grpcweb" // TODO: this works, but it's deprecated :/
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
)

func Multiplex(
	grpcServer *grpc.Server,
	httpMux *http.ServeMux,
) *http.Server {
	wrappedGRPC := grpcweb.WrapServer(
		grpcServer,
		grpcweb.WithAllowedRequestHeaders([]string{"*"}),
		grpcweb.WithOriginFunc(func(origin string) bool {
			return true // allow all origins (adjust in production)
		}),
	)

	return &http.Server{
		Handler: h2c.NewHandler(
			http.HandlerFunc(
				func(
					writer http.ResponseWriter,
					request *http.Request,
				) {
					if wrappedGRPC.IsGrpcWebRequest(request) || wrappedGRPC.IsAcceptableGrpcCorsRequest(request) {
						wrappedGRPC.ServeHTTP(writer, request)
						return
					}

					// standard gRPC
					if request.ProtoMajor == 2 &&
						strings.Contains(request.Header.Get("Content-Type"), "application/grpc") {
						grpcServer.ServeHTTP(writer, request)
						return
					}

					httpMux.ServeHTTP(writer, request)
				},
			),
			&http2.Server{},
		),
	}
}
