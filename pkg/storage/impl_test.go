package storage_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/krixlion/dev-forum_entity/pkg/entity"
	"github.com/krixlion/dev-forum_entity/pkg/helpers/gentest"
	"github.com/krixlion/dev-forum_entity/pkg/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_Get(t *testing.T) {
	type args struct {
		ctx context.Context
		id  string
	}

	testCases := []struct {
		desc    string
		query   mockQuery
		args    args
		want    entity.Entity
		wantErr bool
	}{
		{
			desc: "Test if method is invoked",
			args: args{
				ctx: context.Background(),
				id:  "",
			},
			want: entity.Entity{},
			query: func() mockQuery {
				m := mockQuery{new(mock.Mock)}
				m.On("Get", mock.Anything, mock.AnythingOfType("string")).Return(entity.Entity{}, nil).Once()
				return m
			}(),
		},
		{
			desc: "Test if method forwards an error",
			args: args{
				ctx: context.Background(),
				id:  "",
			},
			want:    entity.Entity{},
			wantErr: true,
			query: func() mockQuery {
				m := mockQuery{new(mock.Mock)}
				m.On("Get", mock.Anything, mock.AnythingOfType("string")).Return(entity.Entity{}, errors.New("test err")).Once()
				return m
			}(),
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			db := storage.NewStorage(mockCmd{}, tC.query, nulls.NullLogger{})
			got, err := db.Get(tC.args.ctx, tC.args.id)
			if (err != nil) != tC.wantErr {
				t.Errorf("storage.Get():\n error = %+v\n wantErr = %+v\n", err, tC.wantErr)
				return
			}

			if !cmp.Equal(got, tC.want) {
				t.Errorf("storage.Get():\n got = %+v\n want = %+v\n", got, tC.want)
				return
			}
			assert.True(t, tC.query.AssertCalled(t, "Get", mock.Anything, tC.args.id))
		})
	}
}
func Test_GetMultiple(t *testing.T) {
	type args struct {
		ctx    context.Context
		offset string
		limit  string
	}

	testCases := []struct {
		desc    string
		query   mockQuery
		args    args
		want    []entity.Entity
		wantErr bool
	}{
		{
			desc: "Test if method is invoked",
			args: args{
				ctx:    context.Background(),
				limit:  "",
				offset: "",
			},
			want: []entity.Entity{},
			query: func() mockQuery {
				m := mockQuery{new(mock.Mock)}
				m.On("GetMultiple", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]entity.Entity{}, nil).Once()
				return m
			}(),
		},
		{
			desc: "Test if method forwards an error",
			args: args{
				ctx:    context.Background(),
				limit:  "",
				offset: "",
			},
			want:    []entity.Entity{},
			wantErr: true,
			query: func() mockQuery {
				m := mockQuery{new(mock.Mock)}
				m.On("GetMultiple", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return([]entity.Entity{}, errors.New("test err")).Once()
				return m
			}(),
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			db := storage.NewStorage(mockCmd{}, tC.query, nulls.NullLogger{})
			got, err := db.GetMultiple(tC.args.ctx, tC.args.offset, tC.args.limit)
			if (err != nil) != tC.wantErr {
				t.Errorf("storage.GetMultiple():\n error = %+v\n wantErr = %+v\n", err, tC.wantErr)
				return
			}

			if !cmp.Equal(got, tC.want, cmpopts.EquateEmpty()) {
				t.Errorf("storage.GetMultiple():\n got = %+v\n want = %+v\n", got, tC.want)
				return
			}

			assert.True(t, tC.query.AssertCalled(t, "GetMultiple", mock.Anything, tC.args.offset, tC.args.limit))
		})
	}
}
func Test_Create(t *testing.T) {
	type args struct {
		ctx    context.Context
		Entity entity.Entity
	}

	testCases := []struct {
		desc    string
		cmd     mockCmd
		args    args
		wantErr bool
	}{
		{
			desc: "Test if method is invoked",
			args: args{
				ctx:    context.Background(),
				Entity: entity.Entity{},
			},

			cmd: func() mockCmd {
				m := mockCmd{new(mock.Mock)}
				m.On("Create", mock.Anything, mock.AnythingOfType("entity.Entity")).Return(nil).Once()
				return m
			}(),
		},
		{
			desc: "Test if an error is forwarded",
			args: args{
				ctx:    context.Background(),
				Entity: entity.Entity{},
			},
			wantErr: true,
			cmd: func() mockCmd {
				m := mockCmd{new(mock.Mock)}
				m.On("Create", mock.Anything, mock.AnythingOfType("entity.Entity")).Return(errors.New("test err")).Once()
				return m
			}(),
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			db := storage.NewStorage(tC.cmd, mockQuery{}, nulls.NullLogger{})
			err := db.Create(tC.args.ctx, tC.args.Entity)
			if (err != nil) != tC.wantErr {
				t.Errorf("storage.Create():\n error = %+v\n wantErr = %+v\n", err, tC.wantErr)
				return
			}
			assert.True(t, tC.cmd.AssertCalled(t, "Create", mock.Anything, tC.args.Entity))
		})
	}
}
func Test_Update(t *testing.T) {
	type args struct {
		ctx    context.Context
		Entity entity.Entity
	}

	testCases := []struct {
		desc    string
		cmd     mockCmd
		args    args
		wantErr bool
	}{
		{
			desc: "Test if method is invoked",
			args: args{
				ctx:    context.Background(),
				Entity: entity.Entity{},
			},

			cmd: func() mockCmd {
				m := mockCmd{new(mock.Mock)}
				m.On("Update", mock.Anything, mock.AnythingOfType("entity.Entity")).Return(nil).Once()
				return m
			}(),
		},
		{
			desc: "Test if error is forwarded",
			args: args{
				ctx:    context.Background(),
				Entity: entity.Entity{},
			},
			wantErr: true,
			cmd: func() mockCmd {
				m := mockCmd{new(mock.Mock)}
				m.On("Update", mock.Anything, mock.AnythingOfType("entity.Entity")).Return(errors.New("test err")).Once()
				return m
			}(),
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			db := storage.NewStorage(tC.cmd, mockQuery{}, nulls.NullLogger{})
			err := db.Update(tC.args.ctx, tC.args.Entity)
			if (err != nil) != tC.wantErr {
				t.Errorf("storage.Update():\n error = %+v\n wantErr = %+v\n", err, tC.wantErr)
				return
			}
			assert.True(t, tC.cmd.AssertCalled(t, "Update", mock.Anything, tC.args.Entity))
		})
	}
}
func Test_Delete(t *testing.T) {
	type args struct {
		ctx context.Context
		id  string
	}

	testCases := []struct {
		desc    string
		cmd     mockCmd
		args    args
		wantErr bool
	}{
		{
			desc: "Test if method is invoked",
			args: args{
				ctx: context.Background(),
				id:  "",
			},

			cmd: func() mockCmd {
				m := mockCmd{new(mock.Mock)}
				m.On("Delete", mock.Anything, mock.AnythingOfType("string")).Return(nil).Once()
				return m
			}(),
		},
		{
			desc: "Test if error is forwarded",
			args: args{
				ctx: context.Background(),
				id:  "",
			},
			wantErr: true,
			cmd: func() mockCmd {
				m := mockCmd{new(mock.Mock)}
				m.On("Delete", mock.Anything, mock.AnythingOfType("string")).Return(errors.New("test err")).Once()
				return m
			}(),
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			db := storage.NewStorage(tC.cmd, mockQuery{}, nulls.NullLogger{})
			err := db.Delete(tC.args.ctx, tC.args.id)
			if (err != nil) != tC.wantErr {
				t.Errorf("storage.Delete():\n error = %+v\n wantErr = %+v\n", err, tC.wantErr)
				return
			}
			assert.True(t, tC.cmd.AssertCalled(t, "Delete", mock.Anything, tC.args.id))
			assert.True(t, tC.cmd.AssertExpectations(t))
		})
	}
}

func Test_CatchUp(t *testing.T) {
	testCases := []struct {
		desc   string
		arg    event.Event
		query  mockQuery
		method string
	}{
		{
			desc: "Test if Update method is invoked on EntityUpdated event",
			arg: event.Event{
				Type: event.EntityUpdated,
				Body: gentest.RandomJSONEntity(2, 3),
			},
			method: "Update",
			query: func() mockQuery {
				m := mockQuery{new(mock.Mock)}
				m.On("Update", mock.Anything, mock.AnythingOfType("entity.Entity")).Return(nil).Once()
				return m
			}(),
		},
		{
			desc: "Test if Create method is invoked on EntityCreated event",
			arg: event.Event{
				Type: event.EntityCreated,
				Body: gentest.RandomJSONEntity(2, 3),
			},
			method: "Create",
			query: func() mockQuery {
				m := mockQuery{new(mock.Mock)}
				m.On("Create", mock.Anything, mock.AnythingOfType("entity.Entity")).Return(nil).Once()
				return m
			}(),
		},
		{
			desc: "Test if Delete method is invoked on EntityDeleted event",
			arg: event.Event{
				Type: event.EntityDeleted,
				Body: func() []byte {
					id, err := json.Marshal(gentest.RandomString(5))
					if err != nil {
						t.Fatalf("Failed to marshal random ID to JSON. Error: %+v", err)
					}
					return id
				}(),
			},
			method: "Delete",
			query: func() mockQuery {
				m := mockQuery{new(mock.Mock)}
				m.On("Delete", mock.Anything, mock.AnythingOfType("string")).Return(nil).Once()
				return m
			}(),
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			db := storage.NewStorage(mockCmd{}, tC.query, nulls.NullLogger{})
			db.CatchUp(tC.arg)

			switch tC.method {
			case "Delete":
				var id string
				err := json.Unmarshal(tC.arg.Body, &id)
				if err != nil {
					t.Errorf("Failed to unmarshal random JSON ID. Error: %+v", err)
					return
				}

				assert.True(t, tC.query.AssertCalled(t, tC.method, mock.Anything, id))

			default:
				var Entity entity.Entity
				err := json.Unmarshal(tC.arg.Body, &Entity)
				if err != nil {
					t.Errorf("Failed to unmarshal random JSON Entity. Error: %+v", err)
					return
				}

				assert.True(t, tC.query.AssertCalled(t, tC.method, mock.Anything, Entity))
			}

			assert.True(t, tC.query.AssertExpectations(t))
		})
	}
}
