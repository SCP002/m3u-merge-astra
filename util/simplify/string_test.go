package simplify

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestName(t *testing.T) {
	assert.Exactly(t, "samplename", Name("Sample, Name!\r\n"))
	assert.NotEqual(t, "samplename2", Name("Sample Name (+2)"))
}
