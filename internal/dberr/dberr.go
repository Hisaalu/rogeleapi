package dberr

import (
	"errors"
	"fmt"

	"github.com/lib/pq"

	"github.com/Hisaalu/rogele/internal/domain"
)

func Translate(err error) error {
	if err == nil {
		return nil
	}

	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		switch pqErr.Code {
		case "23505":
			return fmt.Errorf("%w: %s", domain.ErrConflict, humanReadable(pqErr))
		case "23503":
			return fmt.Errorf("%w: %s", domain.ErrInvalidInput, humanReadable(pqErr))
		case "23502":
			return fmt.Errorf("%w: %s", domain.ErrInvalidInput, humanReadable(pqErr))
		case "23514":
			return fmt.Errorf("%w: %s", domain.ErrInvalidInput, humanReadable(pqErr))
		case "22P02":
			return fmt.Errorf("%w: %s", domain.ErrInvalidInput, humanReadable(pqErr))
		}
	}

	return err
}

func humanReadable(pqErr *pq.Error) string {
	switch pqErr.Code {
	case "23505":
		return "a record with these values already exists"
	case "23503":
		if pqErr.Constraint != "" {
			return "referenced record does not exist (" + pqErr.Constraint + ")"
		}
		return "referenced record does not exist"
	case "23502":
		return "required field is missing"
	case "23514":
		return "value failed validation check"
	case "22P02":
		return "invalid value format"
	}
	return pqErr.Message
}
