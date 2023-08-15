package server

import (
	"github.com/krixlion/dev_forum-entity/pkg/entity"
	pb "github.com/krixlion/dev_forum-entity/pkg/grpc/v1"
)

func entityFromPB(v *pb.Entity) entity.Entity {
	return entity.Entity{
		Id:        v.GetId(),
		UserId:    v.GetUserId(),
		Title:     v.GetTitle(),
		Body:      v.GetBody(),
		CreatedAt: v.GetCreatedAt().AsTime(),
		UpdatedAt: v.GetUpdatedAt().AsTime(),
	}
}
