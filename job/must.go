package job

func mustInt64(val *int64) int64 {
	if val == nil {
		return 0
	}
	return *val
}

func mustString(val *string) string {
	if val == nil {
		return ""
	}
	return *val
}
