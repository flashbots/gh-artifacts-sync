package job

const TypeCleanupUnparseableJob = "cleanup-unparseable-job"

type CleanupUnparseableJob struct {
	Meta *Meta `json:"meta"`
}

func NewCleanupUnparseableJob(path string, err error) *CleanupUnparseableJob {
	return &CleanupUnparseableJob{
		Meta: &Meta{
			Type:          TypeCleanupUnparseableJob,
			persistedPath: path,
		},
	}
}

func (j *CleanupUnparseableJob) meta() *Meta {
	return j.Meta
}
