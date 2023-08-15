package server_test

import (
	"context"
	"errors"
	"log"
	"net"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/krixlion/def-forum_proto/entity_service/pb"
	"github.com/stretchr/testify/mock"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

// setUpServer initializes and runs in the background a gRPC
// server allowing only for local calls for testing.
// Returns a client to interact with the server.
// The server is shutdown when ctx.Done() receives.
func setUpServer(ctx context.Context, mock storage.CQRStorage) pb.EntityServiceClient {
	// bufconn allows the server to call itself
	// great for testing across whole infrastructure
	lis := bufconn.Listen(1024 * 1024)
	bufDialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}

	s := grpc.NewServer()
	server := server.EntityServer{
		Storage: mock,
	}
	pb.RegisterEntityServiceServer(s, server)
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("Server exited with error: %v", err)
		}
	}()

	go func() {
		<-ctx.Done()
		s.Stop()
	}()

	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to dial bufnet: %v", err)
	}

	client := pb.NewEntityServiceClient(conn)
	return client
}

func Test_Get(t *testing.T) {
	v := gentest.RandomEntity(2, 5)
	Entity := &pb.Entity{
		Id:     v.Id,
		UserId: v.UserId,
		Title:  v.Title,
		Body:   v.Body,
	}

	testCases := []struct {
		desc    string
		arg     *pb.GetEntityRequest
		want    *pb.GetEntityResponse
		wantErr bool
		storage storage.CQRStorage
	}{
		{
			desc: "Test if response is returned properly on simple request",
			arg: &pb.GetEntityRequest{
				EntityId: Entity.Id,
			},
			want: &pb.GetEntityResponse{
				Entity: Entity,
			},
			storage: func() (m mockStorage) {
				m.On("Get", mock.Anything, mock.AnythingOfType("string")).Return(v, nil).Times(1)
				return
			}(),
		},
		{
			desc: "Test if error is returned properly on storage error",
			arg: &pb.GetEntityRequest{
				EntityId: "",
			},
			want:    nil,
			wantErr: true,
			storage: func() (m mockStorage) {
				m.On("Get", mock.Anything, mock.AnythingOfType("string")).Return(entity.Entity{}, errors.New("test err")).Times(1)
				return
			}(),
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			ctx, shutdown := context.WithCancel(context.Background())
			defer shutdown()

			client := setUpServer(ctx, tC.storage)

			ctx, cancel := context.WithTimeout(ctx, time.Second)
			defer cancel()

			getResponse, err := client.Get(ctx, tC.arg)
			if (err != nil) != tC.wantErr {
				t.Errorf("Failed to Get Entity, err: %v", err)
				return
			}

			// Equals false if both are nil or they point to the same memory address
			// so be sure to use seperate structs when providing args in order to prevent SEGV.
			if getResponse != tC.want {
				if !cmp.Equal(getResponse.Entity, tC.want.Entity, cmpopts.IgnoreUnexported(pb.Entity{})) {
					t.Errorf("Entitys are not equal:\n Got = %+v\n, want = %+v\n", getResponse.Entity, tC.want.Entity)
					return
				}
			}
		})
	}
}

func Test_Create(t *testing.T) {
	v := gentest.RandomEntity(2, 5)
	Entity := &pb.Entity{
		Id:     v.Id,
		UserId: v.UserId,
		Title:  v.Title,
		Body:   v.Body,
	}

	testCases := []struct {
		desc     string
		arg      *pb.CreateEntityRequest
		dontWant *pb.CreateEntityResponse
		wantErr  bool
		storage  storage.CQRStorage
	}{
		{
			desc: "Test if response is returned properly on simple request",
			arg: &pb.CreateEntityRequest{
				Entity: Entity,
			},
			dontWant: &pb.CreateEntityResponse{
				Id: Entity.Id,
			},
			storage: func() (m mockStorage) {
				m.On("Create", mock.Anything, mock.AnythingOfType("entity.Entity")).Return(nil).Times(1)
				return
			}(),
		},
		{
			desc: "Test if error is returned properly on storage error",
			arg: &pb.CreateEntityRequest{
				Entity: Entity,
			},
			dontWant: nil,
			wantErr:  true,
			storage: func() (m mockStorage) {
				m.On("Create", mock.Anything, mock.AnythingOfType("entity.Entity")).Return(errors.New("test err")).Times(1)
				return
			}(),
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			ctx, shutdown := context.WithCancel(context.Background())
			defer shutdown()
			client := setUpServer(ctx, tC.storage)

			createResponse, err := client.Create(ctx, tC.arg)
			if (err != nil) != tC.wantErr {
				t.Errorf("Failed to Get Entity, err: %v", err)
				return
			}

			// Equals false if both are nil or they point to the same memory address
			// so be sure to use seperate structs when providing args in order to prevent SEGV.
			if createResponse != tC.dontWant {
				if cmp.Equal(createResponse.Id, tC.dontWant.Id) {
					t.Errorf("Entity IDs was not reassigned:\n Got = %+v\n want = %+v\n", createResponse.Id, tC.dontWant.Id)
					return
				}
				if _, err := uuid.FromString(createResponse.Id); err != nil {
					t.Errorf("Entity ID is not correct UUID:\n ID = %+v\n err = %+v", createResponse.Id, err)
					return
				}
			}
		})
	}
}

func Test_Update(t *testing.T) {
	v := gentest.RandomEntity(2, 5)
	Entity := &pb.Entity{
		Id:     v.Id,
		UserId: v.UserId,
		Title:  v.Title,
		Body:   v.Body,
	}

	testCases := []struct {
		desc    string
		arg     *pb.UpdateEntityRequest
		want    *pb.UpdateEntityResponse
		wantErr bool
		storage storage.CQRStorage
	}{
		{
			desc: "Test if response is returned properly on simple request",
			arg: &pb.UpdateEntityRequest{
				Entity: Entity,
			},
			want: &pb.UpdateEntityResponse{},
			storage: func() (m mockStorage) {
				m.On("Update", mock.Anything, mock.AnythingOfType("entity.Entity")).Return(nil).Times(1)
				return
			}(),
		},
		{
			desc: "Test if error is returned properly on storage error",
			arg: &pb.UpdateEntityRequest{
				Entity: Entity,
			},
			want:    nil,
			wantErr: true,
			storage: func() (m mockStorage) {
				m.On("Update", mock.Anything, mock.AnythingOfType("entity.Entity")).Return(errors.New("test err")).Times(1)
				return
			}(),
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			ctx, shutdown := context.WithCancel(context.Background())
			defer shutdown()
			client := setUpServer(ctx, tC.storage)

			got, err := client.Update(ctx, tC.arg)
			if (err != nil) != tC.wantErr {
				t.Errorf("Failed to Update Entity, err: %v", err)
				return
			}

			// Equals false if both are nil or they point to the same memory address
			// so be sure to use seperate structs when providing args in order to prevent SEGV.
			if got != tC.want {
				if !cmp.Equal(got, tC.want, cmpopts.IgnoreUnexported(pb.UpdateEntityResponse{})) {
					t.Errorf("Wrong response:\n got = %+v\n want = %+v\n", got, tC.want)
					return
				}
			}
		})
	}
}

func Test_Delete(t *testing.T) {
	v := gentest.RandomEntity(2, 5)
	Entity := &pb.Entity{
		Id:     v.Id,
		UserId: v.UserId,
		Title:  v.Title,
		Body:   v.Body,
	}

	testCases := []struct {
		desc    string
		arg     *pb.DeleteEntityRequest
		want    *pb.DeleteEntityResponse
		wantErr bool
		storage storage.CQRStorage
	}{
		{
			desc: "Test if response is returned properly on simple request",
			arg: &pb.DeleteEntityRequest{
				EntityId: Entity.Id,
			},
			want: &pb.DeleteEntityResponse{},
			storage: func() (m mockStorage) {
				m.On("Delete", mock.Anything, mock.AnythingOfType("string")).Return(nil).Times(1)
				return
			}(),
		},
		{
			desc: "Test if error is returned properly on storage error",
			arg: &pb.DeleteEntityRequest{
				EntityId: Entity.Id,
			},
			want:    nil,
			wantErr: true,
			storage: func() (m mockStorage) {
				m.On("Delete", mock.Anything, mock.AnythingOfType("string")).Return(errors.New("test err")).Times(1)
				return
			}(),
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			ctx, shutdown := context.WithCancel(context.Background())
			defer shutdown()
			client := setUpServer(ctx, tC.storage)

			got, err := client.Delete(ctx, tC.arg)
			if (err != nil) != tC.wantErr {
				t.Errorf("Failed to Delete Entity, err: %v", err)
				return
			}

			// Equals false if both are nil or they point to the same memory address
			// so be sure to use seperate structs when providing args in order to prevent SEGV.
			if got != tC.want {
				if !cmp.Equal(got, tC.want, cmpopts.IgnoreUnexported(pb.DeleteEntityResponse{})) {
					t.Errorf("Wrong response:\n got = %+v\n want = %+v\n", got, tC.want)
					return
				}
			}
		})
	}
}

func Test_GetStream(t *testing.T) {
	var Entitys []entity.Entity
	for i := 0; i < 5; i++ {
		Entity := gentest.RandomEntity(2, 5)
		Entitys = append(Entitys, Entity)
	}

	var pbEntitys []*pb.Entity
	for _, v := range Entitys {
		pbEntity := &pb.Entity{
			Id:     v.Id,
			UserId: v.UserId,
			Title:  v.Title,
			Body:   v.Body,
		}
		pbEntitys = append(pbEntitys, pbEntity)
	}

	testCases := []struct {
		desc    string
		arg     *pb.GetEntitysRequest
		want    []*pb.Entity
		wantErr bool
		storage storage.CQRStorage
	}{
		{
			desc: "Test if response is returned properly on simple request",
			arg: &pb.GetEntitysRequest{
				Offset: "0",
				Limit:  "5",
			},
			want: pbEntitys,
			storage: func() (m mockStorage) {
				m.On("GetMultiple", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(Entitys, nil).Times(1)
				return
			}(),
		},
		{
			desc:    "Test if error is returned properly on storage error",
			arg:     &pb.GetEntitysRequest{},
			want:    nil,
			wantErr: true,
			storage: func() (m mockStorage) {
				m.On("GetMultiple", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]entity.Entity{}, errors.New("test err")).Times(1)
				return
			}(),
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			ctx, shutdown := context.WithCancel(context.Background())
			defer shutdown()
			client := setUpServer(ctx, tC.storage)

			stream, err := client.GetStream(ctx, tC.arg)
			if err != nil {
				t.Errorf("Failed to Get stream, err: %v", err)
				return
			}

			var got []*pb.Entity
			for i := 0; i < len(tC.want); i++ {
				Entity, err := stream.Recv()
				if (err != nil) != tC.wantErr {
					t.Errorf("Failed to receive Entity from stream, err: %v", err)
					return
				}
				got = append(got, Entity)
			}

			if !cmp.Equal(got, tC.want, cmpopts.IgnoreUnexported(pb.Entity{})) {
				t.Errorf("Entitys are not equal:\n Got = %+v\n want = %+v\n", got, tC.want)
				return
			}
		})
	}
}
