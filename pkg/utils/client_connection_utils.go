package utils

import (
	"errors"

	"github.com/sk25469/scribble_backend/pkg/model"
)

func Remove(s []model.ClientInfo, client model.ClientInfo) ([]model.ClientInfo, error) {
	idx := -1
	for index, val := range s {
		if val == client {
			idx = index
			break
		}
	}

	if idx == -1 {
		return s, errors.New("ID doesn't exists")
	}

	s[idx] = s[len(s)-1]
	return s[:len(s)-1], nil
}
