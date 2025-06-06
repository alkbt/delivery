package kernel_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"delivery/internal/core/domain/model/kernel"
	"delivery/internal/pkg/errs"
)

func TestNewLocation(t *testing.T) {
	tests := []struct {
		name    string
		x       kernel.Coordinate
		y       kernel.Coordinate
		wantErr bool
		errType error
	}{
		{
			name:    "valid location",
			x:       5,
			y:       5,
			wantErr: false,
		},
		{
			name:    "valid location at min bounds",
			x:       kernel.LocationMinX,
			y:       kernel.LocationMinY,
			wantErr: false,
		},
		{
			name:    "valid location at max bounds",
			x:       kernel.LocationMaxX,
			y:       kernel.LocationMaxY,
			wantErr: false,
		},
		{
			name:    "invalid x too small",
			x:       kernel.LocationMinX - 1,
			y:       5,
			wantErr: true,
			errType: errs.NewValueIsOutOfRangeError("x", kernel.Coordinate(kernel.LocationMinX-1), kernel.LocationMinX, kernel.LocationMaxX),
		},
		{
			name:    "invalid x too large",
			x:       kernel.LocationMaxX + 1,
			y:       5,
			wantErr: true,
			errType: errs.NewValueIsOutOfRangeError("x", kernel.Coordinate(kernel.LocationMaxX+1), kernel.LocationMinX, kernel.LocationMaxX),
		},
		{
			name:    "invalid y too small",
			x:       5,
			y:       kernel.LocationMinY - 1,
			wantErr: true,
			errType: errs.NewValueIsOutOfRangeError("y", kernel.Coordinate(kernel.LocationMinY-1), kernel.LocationMinY, kernel.LocationMaxY),
		},
		{
			name:    "invalid y too large",
			x:       5,
			y:       kernel.LocationMaxY + 1,
			wantErr: true,
			errType: errs.NewValueIsOutOfRangeError("y", kernel.Coordinate(kernel.LocationMaxY+1), kernel.LocationMinY, kernel.LocationMaxY),
		},
		{
			name:    "both x and y invalid",
			x:       kernel.LocationMinX - 1,
			y:       kernel.LocationMaxY + 1,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc, err := kernel.NewLocation(tt.x, tt.y)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Zero(t, loc)
				if tt.errType != nil {
					assert.ErrorAs(t, err, &tt.errType)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.x, loc.X())
				assert.Equal(t, tt.y, loc.Y())
				assert.NoError(t, loc.Validate())
			}
		})
	}
}

func TestNewRandomLocation(t *testing.T) {
	for range 100 {
		loc, err := kernel.NewRandomLocation()
		require.NoError(t, err)

		assert.NoError(t, loc.Validate())

		assert.GreaterOrEqual(t, loc.X(), kernel.LocationMinX)
		assert.LessOrEqual(t, loc.X(), kernel.LocationMaxX)
		assert.GreaterOrEqual(t, loc.Y(), kernel.LocationMinY)
		assert.LessOrEqual(t, loc.Y(), kernel.LocationMaxY)
	}
}

func TestLocation_Validate(t *testing.T) {
	t.Run("valid location", func(t *testing.T) {
		loc, err := kernel.NewLocation(5, 5)
		require.NoError(t, err)
		assert.NoError(t, loc.Validate())
	})

	t.Run("zero value location", func(t *testing.T) {
		var loc kernel.Location
		err := loc.Validate()
		assert.Error(t, err)
		assert.Equal(t, kernel.ErrLocationIsNotConstructed, err)
	})
}

func TestLocation_Getters(t *testing.T) {
	loc, err := kernel.NewLocation(3, 7)
	require.NoError(t, err)

	assert.Equal(t, kernel.Coordinate(3), loc.X())
	assert.Equal(t, kernel.Coordinate(7), loc.Y())
}

func TestLocation_String(t *testing.T) {
	tests := []struct {
		name string
		x, y kernel.Coordinate
		want string
	}{
		{
			name: "basic location",
			x:    3,
			y:    7,
			want: "Location(3,7)",
		},
		{
			name: "min bounds",
			x:    kernel.LocationMinX,
			y:    kernel.LocationMinY,
			want: "Location(1,1)",
		},
		{
			name: "max bounds",
			x:    kernel.LocationMaxX,
			y:    kernel.LocationMaxY,
			want: "Location(10,10)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc, err := kernel.NewLocation(tt.x, tt.y)
			require.NoError(t, err)
			assert.Equal(t, tt.want, loc.String())
		})
	}
}

func TestLocation_IsEqual(t *testing.T) {
	tests := []struct {
		name    string
		loc1    kernel.Location
		loc2    kernel.Location
		want    bool
		wantErr bool
	}{
		{
			name:    "equal locations",
			loc1:    mustNewLocation(t, 5, 5),
			loc2:    mustNewLocation(t, 5, 5),
			want:    true,
			wantErr: false,
		},
		{
			name:    "different x",
			loc1:    mustNewLocation(t, 3, 5),
			loc2:    mustNewLocation(t, 5, 5),
			want:    false,
			wantErr: false,
		},
		{
			name:    "different y",
			loc1:    mustNewLocation(t, 5, 3),
			loc2:    mustNewLocation(t, 5, 5),
			want:    false,
			wantErr: false,
		},
		{
			name:    "both different",
			loc1:    mustNewLocation(t, 3, 3),
			loc2:    mustNewLocation(t, 5, 5),
			want:    false,
			wantErr: false,
		},
		{
			name:    "first location invalid",
			loc1:    kernel.Location{},
			loc2:    mustNewLocation(t, 5, 5),
			want:    false,
			wantErr: true,
		},
		{
			name:    "second location invalid",
			loc1:    mustNewLocation(t, 5, 5),
			loc2:    kernel.Location{},
			want:    false,
			wantErr: true,
		},
		{
			name:    "both locations invalid",
			loc1:    kernel.Location{},
			loc2:    kernel.Location{},
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.loc1.IsEqual(tt.loc2)

			if tt.wantErr {
				assert.Error(t, err)
				assert.False(t, got)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestLocation_Distance(t *testing.T) {
	tests := []struct {
		name    string
		loc1    kernel.Location
		loc2    kernel.Location
		want    kernel.Coordinate
		wantErr bool
	}{
		{
			name:    "same location",
			loc1:    mustNewLocation(t, 5, 5),
			loc2:    mustNewLocation(t, 5, 5),
			want:    0,
			wantErr: false,
		},
		{
			name:    "positive distance",
			loc1:    mustNewLocation(t, 7, 8),
			loc2:    mustNewLocation(t, 3, 4),
			want:    8, // (7-3) + (8-4) = 4 + 4 = 8
			wantErr: false,
		},
		{
			name:    "distance is symmetric",
			loc1:    mustNewLocation(t, 3, 4),
			loc2:    mustNewLocation(t, 7, 8),
			want:    8, // Manhattan distance: |3-7| + |4-8| = 4 + 4 = 8
			wantErr: false,
		},
		{
			name:    "mixed coordinates distance",
			loc1:    mustNewLocation(t, 8, 3),
			loc2:    mustNewLocation(t, 2, 9),
			want:    12, // Manhattan distance: |8-2| + |3-9| = 6 + 6 = 12
			wantErr: false,
		},
		{
			name:    "maximum distance corner to corner",
			loc1:    mustNewLocation(t, 1, 1),
			loc2:    mustNewLocation(t, 10, 10),
			want:    18, // Manhattan distance: |1-10| + |1-10| = 9 + 9 = 18
			wantErr: false,
		},
		{
			name:    "maximum distance corner to corner reversed",
			loc1:    mustNewLocation(t, 10, 10),
			loc2:    mustNewLocation(t, 1, 1),
			want:    18, // Manhattan distance: |10-1| + |10-1| = 9 + 9 = 18
			wantErr: false,
		},
		{
			name:    "first location invalid",
			loc1:    kernel.Location{},
			loc2:    mustNewLocation(t, 5, 5),
			want:    0,
			wantErr: true,
		},
		{
			name:    "second location invalid",
			loc1:    mustNewLocation(t, 5, 5),
			loc2:    kernel.Location{},
			want:    0,
			wantErr: true,
		},
		{
			name:    "both locations invalid",
			loc1:    kernel.Location{},
			loc2:    kernel.Location{},
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.loc1.Distance(tt.loc2)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, kernel.Coordinate(0), got)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestLocation_DistanceProperties(t *testing.T) {
	t.Run("distance symmetry", func(t *testing.T) {
		// Test that distance(A, B) == distance(B, A)
		for x1 := kernel.LocationMinX; x1 <= kernel.LocationMaxX; x1++ {
			for y1 := kernel.LocationMinY; y1 <= kernel.LocationMaxY; y1++ {
				for x2 := kernel.LocationMinX; x2 <= kernel.LocationMaxX; x2++ {
					for y2 := kernel.LocationMinY; y2 <= kernel.LocationMaxY; y2++ {
						loc1 := mustNewLocation(t, x1, y1)
						loc2 := mustNewLocation(t, x2, y2)
						
						dist1, err1 := loc1.Distance(loc2)
						require.NoError(t, err1)
						
						dist2, err2 := loc2.Distance(loc1)
						require.NoError(t, err2)
						
						assert.Equal(t, dist1, dist2, "Distance should be symmetric for %v and %v", loc1, loc2)
					}
				}
			}
		}
	})

	t.Run("distance identity", func(t *testing.T) {
		// Test that distance(A, A) == 0
		for x := kernel.LocationMinX; x <= kernel.LocationMaxX; x++ {
			for y := kernel.LocationMinY; y <= kernel.LocationMaxY; y++ {
				loc := mustNewLocation(t, x, y)
				dist, err := loc.Distance(loc)
				require.NoError(t, err)
				assert.Equal(t, kernel.Coordinate(0), dist, "Distance from location to itself should be 0")
			}
		}
	})

	t.Run("triangle inequality", func(t *testing.T) {
		// Test that distance(A, C) <= distance(A, B) + distance(B, C)
		testCases := []struct {
			name string
			a, b, c kernel.Coordinate
		}{
			{"diagonal", 1, 5, 10},
			{"horizontal", 3, 6, 9},
			{"vertical", 2, 4, 8},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				locA := mustNewLocation(t, tc.a, tc.a)
				locB := mustNewLocation(t, tc.b, tc.b)
				locC := mustNewLocation(t, tc.c, tc.c)

				distAC, err := locA.Distance(locC)
				require.NoError(t, err)

				distAB, err := locA.Distance(locB)
				require.NoError(t, err)

				distBC, err := locB.Distance(locC)
				require.NoError(t, err)

				assert.LessOrEqual(t, distAC, distAB+distBC, "Triangle inequality should hold")
			})
		}
	})
}

func TestLocation_EdgeCases(t *testing.T) {
	t.Run("boundary coordinates", func(t *testing.T) {
		corners := []struct {
			name string
			x, y kernel.Coordinate
		}{
			{"bottom-left", kernel.LocationMinX, kernel.LocationMinY},
			{"bottom-right", kernel.LocationMaxX, kernel.LocationMinY},
			{"top-left", kernel.LocationMinX, kernel.LocationMaxY},
			{"top-right", kernel.LocationMaxX, kernel.LocationMaxY},
		}

		for _, corner := range corners {
			t.Run(corner.name, func(t *testing.T) {
				loc, err := kernel.NewLocation(corner.x, corner.y)
				require.NoError(t, err)
				assert.NoError(t, loc.Validate())
				assert.Equal(t, corner.x, loc.X())
				assert.Equal(t, corner.y, loc.Y())
			})
		}
	})

	t.Run("maximum possible distance", func(t *testing.T) {
		corner1 := mustNewLocation(t, kernel.LocationMinX, kernel.LocationMinY)
		corner2 := mustNewLocation(t, kernel.LocationMaxX, kernel.LocationMaxY)
		
		expectedMaxDistance := (kernel.LocationMaxX - kernel.LocationMinX) + (kernel.LocationMaxY - kernel.LocationMinY)
		
		dist, err := corner1.Distance(corner2)
		require.NoError(t, err)
		assert.Equal(t, expectedMaxDistance, dist)
	})
}

func FuzzNewLocation(f *testing.F) {
	// Add seed corpus
	f.Add(int8(1), int8(1))
	f.Add(int8(10), int8(10))
	f.Add(int8(5), int8(5))
	f.Add(int8(0), int8(11))  // Invalid values

	f.Fuzz(func(t *testing.T, x, y int8) {
		loc, err := kernel.NewLocation(kernel.Coordinate(x), kernel.Coordinate(y))
		
		if x >= int8(kernel.LocationMinX) && x <= int8(kernel.LocationMaxX) &&
		   y >= int8(kernel.LocationMinY) && y <= int8(kernel.LocationMaxY) {
			// Should succeed
			require.NoError(t, err)
			assert.Equal(t, kernel.Coordinate(x), loc.X())
			assert.Equal(t, kernel.Coordinate(y), loc.Y())
			assert.NoError(t, loc.Validate())
		} else {
			// Should fail
			assert.Error(t, err)
			assert.Zero(t, loc)
		}
	})
}

func mustNewLocation(t *testing.T, x, y kernel.Coordinate) kernel.Location {
	t.Helper()
	loc, err := kernel.NewLocation(x, y)
	require.NoError(t, err)
	return loc
}
