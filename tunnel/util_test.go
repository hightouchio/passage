package tunnel

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_FilterTunnelsById(t *testing.T) {
	tests := []struct {
		name string

		mode     string
		filter   []uuid.UUID
		original []Tunnel
		expected []Tunnel
	}{
		{
			mode: "whitelist",
			name: "empty",

			filter: []uuid.UUID{},
			original: []Tunnel{
				&NormalTunnel{ID: uuid.MustParse("00000000-0000-0000-0000-000000000000")},
				&NormalTunnel{ID: uuid.MustParse("00000000-0000-0000-0000-000000000001")},
				&NormalTunnel{ID: uuid.MustParse("00000000-0000-0000-0000-000000000002")},
			},
			expected: []Tunnel{},
		},

		{
			mode: "whitelist",
			name: "filtered",
			filter: []uuid.UUID{
				uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},
			original: []Tunnel{
				&NormalTunnel{ID: uuid.MustParse("00000000-0000-0000-0000-000000000000")},
				&NormalTunnel{ID: uuid.MustParse("00000000-0000-0000-0000-000000000001")},
				&NormalTunnel{ID: uuid.MustParse("00000000-0000-0000-0000-000000000002")},
			},
			expected: []Tunnel{
				&NormalTunnel{ID: uuid.MustParse("00000000-0000-0000-0000-000000000001")},
			},
		},

		{
			mode: "blacklist",
			name: "empty",

			filter: []uuid.UUID{},
			original: []Tunnel{
				&NormalTunnel{ID: uuid.MustParse("00000000-0000-0000-0000-000000000000")},
				&NormalTunnel{ID: uuid.MustParse("00000000-0000-0000-0000-000000000001")},
				&NormalTunnel{ID: uuid.MustParse("00000000-0000-0000-0000-000000000002")},
			},
			expected: []Tunnel{
				&NormalTunnel{ID: uuid.MustParse("00000000-0000-0000-0000-000000000000")},
				&NormalTunnel{ID: uuid.MustParse("00000000-0000-0000-0000-000000000001")},
				&NormalTunnel{ID: uuid.MustParse("00000000-0000-0000-0000-000000000002")},
			},
		},

		{
			mode: "blacklist",
			name: "filtered",
			filter: []uuid.UUID{
				uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},
			original: []Tunnel{
				&NormalTunnel{ID: uuid.MustParse("00000000-0000-0000-0000-000000000000")},
				&NormalTunnel{ID: uuid.MustParse("00000000-0000-0000-0000-000000000001")},
				&NormalTunnel{ID: uuid.MustParse("00000000-0000-0000-0000-000000000002")},
			},
			expected: []Tunnel{
				&NormalTunnel{ID: uuid.MustParse("00000000-0000-0000-0000-000000000000")},
				&NormalTunnel{ID: uuid.MustParse("00000000-0000-0000-0000-000000000002")},
			},
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s %s", test.mode, test.name), func(t *testing.T) {
			originalListFunc := func(ctx context.Context) ([]Tunnel, error) {
				return test.original, nil
			}

			// Generate the new list func
			listFunc, err := FilterTunnelsByIds(originalListFunc, test.mode, test.filter)
			assert.NoError(t, err)

			// Filter the tunnels by ID
			filtered, err := listFunc(context.Background())
			assert.NoError(t, err)

			// Assert that the filtered tunnels match what we expect
			assert.EqualValues(t, test.expected, filtered)
		})
	}
}
