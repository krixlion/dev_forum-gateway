package server

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/krixlion/dev_forum-entity/pkg/entity"
	pb "github.com/krixlion/dev_forum-entity/pkg/grpc/v1"
	"github.com/krixlion/dev_forum-entity/pkg/helpers/gentest"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func Test_entityFromPB(t *testing.T) {
	id := gentest.RandomString(3)
	userId := gentest.RandomString(3)
	body := gentest.RandomString(3)
	title := gentest.RandomString(3)

	testCases := []struct {
		desc string
		arg  *pb.Entity
		want entity.Entity
	}{
		{
			desc: "Test if works on simple random data",
			arg: &pb.Entity{
				Id:        id,
				UserId:    userId,
				Title:     title,
				Body:      body,
				CreatedAt: timestamppb.New(time.Time{}),
				UpdatedAt: timestamppb.New(time.Time{}),
			},
			want: entity.Entity{
				Id:        id,
				UserId:    userId,
				Title:     title,
				Body:      body,
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			got := EntityFromPB(tC.arg)

			if !cmp.Equal(got, tC.want, cmpopts.IgnoreUnexported(pb.Entity{})) {
				t.Errorf("Entitys are not equal:\n got = %+v\n want = %+v\n", got, tC.want)
				return
			}
		})
	}
}
