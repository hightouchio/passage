package stats

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_joinTags(t *testing.T) {
	tags := convertTags(mergeTags([]Tags{
		Tags{
			"a": 1,
			"b": 2,
		},
		Tags{
			"b": 3,
			"c": "hello",
		},
		Tags{
			"a": "world",
			"d": 5.5,
		},
	}))

	assert.Equal(t, tags, []string{"a:world", "b:3", "c:hello", "d:5.5"})
}
