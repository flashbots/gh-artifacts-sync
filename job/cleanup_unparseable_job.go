package job

import (
	"fmt"
	"math/rand/v2"
)

const TypeCleanupUnparseableJob = "cleanup-unparseable-job"

type CleanupUnparseableJob struct {
	Meta *Meta `json:"meta"`
}

func NewCleanupUnparseableJob(path string, err error) *CleanupUnparseableJob {
	return &CleanupUnparseableJob{
		Meta: &Meta{
			Type:          TypeCleanupUnparseableJob,
			ID:            fmt.Sprintf("%s-%d", TypeCleanupUnparseableJob, rand.Int64()),
			persistedPath: path,
		},
	}
}

func (j *CleanupUnparseableJob) meta() *Meta {
	return j.Meta
}
