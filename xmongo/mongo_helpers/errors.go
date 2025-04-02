package mongo_helpers

import (
	"errors"

	"github.com/raphoester/x/xerrs"
	"go.mongodb.org/mongo-driver/mongo"
)

func MapErr(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, mongo.ErrNoDocuments) {
		return xerrs.ErrNotFound
	}

	if mongo.IsDuplicateKeyError(err) {
		return xerrs.ErrConflict
	}

	return err
}
