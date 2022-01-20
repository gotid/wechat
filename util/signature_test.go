package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSignature(t *testing.T) {
	abc := "a9993e364706816aba3e25717850c26c9cd0d89d"
	assert.Equal(t, abc, Signature("a", "b", "c"))
}
