package storage

import (
	"context"
	"io"

)

type CQRStorage interface {
	Getter
	Writer
	CatchUp(event.Event)
}

type Storage interface {
	Getter
	Writer
}

type Getter interface {
	io.Closer
	Get(ctx context.Context, id string) (entity.Entity, error)
	GetMultiple(ctx context.Context, offset, limit string) ([]entity.Entity, error)
}

type Writer interface {
	io.Closer
	Create(context.Context, entity.Entity) error
	Update(context.Context, entity.Entity) error
	Delete(ctx context.Context, id string) error
}

type Eventstore interface {
	event.Consumer
	Writer
}
