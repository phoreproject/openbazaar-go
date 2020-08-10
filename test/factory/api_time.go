package factory

import (
	"time"

	"github.com/phoreproject/pm-go/repo"
)

func NewAPITime(t time.Time) *repo.APITime {
	return repo.NewAPITime(t)
}
