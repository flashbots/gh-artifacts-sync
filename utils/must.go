package utils

func MustString(str *string) string {
	if str == nil {
		return ""
	}
	return *str
}
