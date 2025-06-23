package utils

import (
	"fmt"
	"strings"
)

func Join[T fmt.Stringer](elems []T, sep string) string {
	const maxInt = int(^uint(0) >> 1)

	switch len(elems) {
	case 0:
		return ""
	case 1:
		return elems[0].String()
	}

	var n int
	if len(sep) > 0 {
		if len(sep) >= maxInt/(len(elems)-1) {
			panic("strings: Join output length overflow")
		}
		n += len(sep) * (len(elems) - 1)
	}
	for _, elem := range elems {
		if len(elem.String()) > maxInt-n {
			panic("strings: Join output length overflow")
		}
		n += len(elem.String())
	}

	var b strings.Builder
	b.Grow(n)
	b.WriteString(elems[0].String())
	for _, s := range elems[1:] {
		b.WriteString(sep)
		b.WriteString(s.String())
	}
	return b.String()
}
