package web

import (
	"bytes"
	"fmt"
	"math"
	"math/rand/v2"
	"testing"
)

func TestHomepage(t *testing.T) {
	t.Parallel()

	inputs := []uint{0, math.MaxUint, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	inputs = append(inputs, uint(rand.Int64N(math.MaxInt64)))

	var buffer bytes.Buffer
	for _, tc := range inputs {
		t.Run(fmt.Sprintf("TestHomepage%d", tc), func(t *testing.T) {
			if err := homepage(tc).Render(&buffer); err != nil {
				t.Errorf("homepage render: %v", err)
			}
		})
	}
}
