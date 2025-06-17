package utils

func Map[S any, D any](src []S, m func(S) D) []D {
	if src == nil {
		return nil
	}
	dst := make([]D, 0, len(src))
	for _, s := range src {
		dst = append(dst, m(s))
	}
	return dst
}
