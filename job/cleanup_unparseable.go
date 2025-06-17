package job

const TypeCleanupUnparseable = "cleanup-unparseable"

type CleanupUnparseable struct {
	Meta *Meta `json:"meta"`
}

func NewCleanupUnparseable(path string, err error) *CleanupUnparseable {
	return &CleanupUnparseable{
		Meta: &Meta{
			Type:          TypeCleanupUnparseable,
			persistedPath: path,
		},
	}
}

func (j *CleanupUnparseable) meta() *Meta {
	return j.Meta
}
