package factory

import (
	"time"

	"github.com/phoreproject/openbazaar-go/repo"
)

func NewAPITime(t time.Time) *repo.APITime {
	return repo.NewAPITime(t)
}
