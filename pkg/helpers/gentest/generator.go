package gentest

import (
	"encoding/json"
	"math/rand"

	"github.com/gofrs/uuid"
)

func RandomString(length int) string {
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	v := make([]rune, length)
	for i := range v {
		v[i] = letters[rand.Intn(len(letters))]
	}
	return string(v)
}

// RandomEntity panics on hardware error.
// It should be used ONLY for testing.
func RandomEntity(titleLen, bodyLen int) entity.Entity {
	id := uuid.Must(uuid.NewV4())
	userId := uuid.Must(uuid.NewV4())

	return entity.Entity{
		Id:     id.String(),
		UserId: userId.String(),
		Title:  RandomString(titleLen),
		Body:   RandomString(bodyLen),
	}
}

// RandomEntity returns a random Entity marshaled
// to JSON and panics on error.
// It should be used ONLY for testing.
func RandomJSONEntity(titleLen, bodyLen int) []byte {
	Entity := RandomEntity(titleLen, bodyLen)
	json, err := json.Marshal(Entity)
	if err != nil {
		panic(err)
	}
	return json
}
