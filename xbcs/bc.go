package xbcs

import (
	"context"
	"net/http"

	"github.com/raphoester/x/xgrpc"
	"google.golang.org/grpc"
)

type Runnable interface {
	Run(ctx context.Context) error
}

type RouteDeclarer interface {
	DeclareRoutes(mux *http.ServeMux) // todo make it more flexible
}

type Option func(*BoundedContext)

func WithRunnables(r ...Runnable) Option {
	return func(bc *BoundedContext) {
		bc.runnables = append(bc.runnables, r...)
	}
}

func WithGRPCRegistrables(r ...xgrpc.Registrable) Option {
	return func(bc *BoundedContext) {
		bc.grpcRegistrables = append(bc.grpcRegistrables, r...)
	}
}

func WithRouteDeclarers(r ...RouteDeclarer) Option {
	return func(bc *BoundedContext) {
		bc.routeDeclarers = append(bc.routeDeclarers, r...)
	}
}

func WithOthers(f func() error) Option {
	return func(bc *BoundedContext) {
		bc.others = f
	}
}

func New(opts ...Option) *BoundedContext {
	bc := &BoundedContext{}

	for _, opt := range opts {
		opt(bc)
	}

	return bc
}

type BoundedContext struct {
	runnables        []Runnable
	grpcRegistrables []xgrpc.Registrable
	routeDeclarers   []RouteDeclarer
	others           func() error
}

func (bc BoundedContext) Register(
	ctx context.Context,
	grpcServer *grpc.Server,
	mux *http.ServeMux,
) error {
	for _, r := range bc.runnables {
		if err := r.Run(ctx); err != nil {
			return err
		}
	}

	for _, r := range bc.routeDeclarers {
		r.DeclareRoutes(mux)
	}

	for _, r := range bc.grpcRegistrables {
		r.Registrar()(grpcServer)
	}

	if bc.others != nil {
		return bc.others()
	}

	return nil
}
