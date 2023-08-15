package server

import (
	"context"

	pb "github.com/krixlion/dev_forum-entity/pkg/grpc/v1"
	"google.golang.org/grpc"
)

func (s EntityServer) ValidateRequestInterceptor() grpc.UnaryServerInterceptor {

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		switch info.FullMethod {
		case "/entityService/Create":
			return s.validateCreate(ctx, req.(*pb.CreateEntityRequest), handler)
		case "/entityService/Update":
			return s.validateUpdate(ctx, req.(*pb.UpdateEntityRequest), handler)
		case "/entityService/Delete":
			return s.validateDelete(ctx, req.(*pb.DeleteEntityRequest), handler)
		default:
			return handler(ctx, req)
		}
	}
}

func (s EntityServer) validateCreate(ctx context.Context, req *pb.CreateEntityRequest, handler grpc.UnaryHandler) (interface{}, error) {
	return handler(ctx, req)
}

func (s EntityServer) validateUpdate(ctx context.Context, req *pb.UpdateEntityRequest, handler grpc.UnaryHandler) (interface{}, error) {
	return handler(ctx, req)
}

func (s EntityServer) validateDelete(ctx context.Context, req *pb.DeleteEntityRequest, handler grpc.UnaryHandler) (interface{}, error) {
	return handler(ctx, req)
}
