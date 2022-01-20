package util

import (
	"crypto/sha1"
	"fmt"
	"io"
	"sort"
)

// Signature sha1 签名。
func Signature(x ...string) string {
	sort.Strings(x)
	h := sha1.New()
	for _, s := range x {
		_, _ = io.WriteString(h, s)
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}
