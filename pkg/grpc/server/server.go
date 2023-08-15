package server

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/krixlion/def-forum_proto/entity_service/pb"
	"github.com/krixlion/dev-forum_Entity/pkg/entity"
	"github.com/krixlion/dev-forum_Entity/pkg/logging"
	"github.com/krixlion/dev-forum_Entity/pkg/storage"
	"github.com/krixlion/dev_forum-lib/event"
	"github.com/krixlion/dev_forum-lib/event/dispatcher"

	"github.com/gofrs/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type EntityServer struct {
	pb.UnimplementedEntityServiceServer
	storage    storage.CQRStorage
	logger     logging.Logger
	dispatcher *dispatcher.Dispatcher
}

func NewEntityServer(storage storage.CQRStorage, logger logging.Logger, dispatcher *dispatcher.Dispatcher) EntityServer {
	return EntityServer{
		storage:    storage,
		logger:     logger,
		dispatcher: dispatcher,
	}
}

func (s EntityServer) Close() error {
	var errMsg string

	err := s.storage.Close()
	if err != nil {
		errMsg = fmt.Sprintf("%s, failed to close storage: %s", errMsg, err)
	}

	if errMsg != "" {
		return errors.New(errMsg)
	}

	return nil
}

func (s EntityServer) Create(ctx context.Context, req *pb.CreateEntityRequest) (*pb.CreateEntityResponse, error) {
	entity := entity.EntityFromPB(req.GetEntity())
	id, err := uuid.NewV4()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	// Assign new UUID to entity about to be created.
	entity.Id = id.String()

	if err := s.storage.Create(ctx, entity); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	s.dispatcher.Publish(event.MakeEvent(event.EntityAggregate, event.EntityCreated, entity))

	return &pb.CreateEntityResponse{
		Id: id.String(),
	}, nil
}

func (s EntityServer) Delete(ctx context.Context, req *pb.DeleteEntityRequest) (*pb.DeleteEntityResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	id := req.GetEntityId()

	if err := s.storage.Delete(ctx, id); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	s.dispatcher.Publish(event.MakeEvent(event.EntityAggregate, event.EntityDeleted, id))

	return &pb.DeleteEntityResponse{}, nil
}

func (s EntityServer) Update(ctx context.Context, req *pb.UpdateEntityRequest) (*pb.UpdateEntityResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	entity := entity.EntityFromPB(req.GetEntity())

	if err := s.storage.Update(ctx, entity); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	s.dispatcher.Publish(event.MakeEvent(event.EntityAggregate, event.EntityUpdated, entity))

	return &pb.UpdateEntityResponse{}, nil
}

func (s EntityServer) Get(ctx context.Context, req *pb.GetEntityRequest) (*pb.GetEntityResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	entity, err := s.storage.Get(ctx, req.GetEntityId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to get entity: %v", err)
	}

	return &pb.GetEntityResponse{
		Entity: &pb.Entity{
			Id:     entity.Id,
			UserId: entity.UserId,
			Title:  entity.Title,
			Body:   entity.Body,
		},
	}, err
}

func (s EntityServer) GetStream(req *pb.GetEntitysRequest, stream pb.EntityService_GetStreamServer) error {
	ctx, cancel := context.WithTimeout(stream.Context(), time.Second*10)
	defer cancel()

	Entitys, err := s.storage.GetMultiple(ctx, req.GetOffset(), req.GetLimit())
	if err != nil {
		return err
	}

	for _, v := range Entitys {
		select {
		case <-ctx.Done():
			return nil
		default:
			Entity := pb.Entity{
				Id:     v.Id,
				UserId: v.UserId,
				Title:  v.Title,
				Body:   v.Body,
			}

			if err := stream.Send(&Entity); err != nil {
				return err
			}
		}
	}
	return nil
}
