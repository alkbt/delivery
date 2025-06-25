package kernel

import (
	"errors"
	"fmt"
	"math/rand/v2"

	"delivery/internal/pkg/errs"
	"delivery/internal/pkg/guard"
)

// Coordinate represents a position value on the delivery grid.
// Valid coordinates range from LocationMinX/Y to LocationMaxX/Y inclusive.
type Coordinate int8

const (
	// LocationMinX is the minimum valid X coordinate on the delivery grid.
	LocationMinX Coordinate = 1
	// LocationMinY is the minimum valid Y coordinate on the delivery grid.
	LocationMinY Coordinate = 1
	// LocationMaxX is the maximum valid X coordinate on the delivery grid.
	LocationMaxX Coordinate = 10
	// LocationMaxY is the maximum valid Y coordinate on the delivery grid.
	LocationMaxY Coordinate = 10
)

// ErrLocationIsNotConstructed is returned when attempting to use an improperly initialized Location.
// Locations must be created using NewLocation or NewRandomLocation constructors to ensure validity.
var ErrLocationIsNotConstructed = errs.NewValueIsRequiredError(
	"location must be created via NewLocation or NewRandomLocation constructors")

// Location represents a point on the delivery grid with validated coordinates.
// Location is an immutable value object that ensures coordinates are always within valid bounds.
// The zero value of Location is invalid and will fail validation - use constructors to create instances.
//
// Example:
//
//	loc, err := kernel.NewLocation(5, 7)
//	if err != nil {
//	    // Handle validation error
//	}
//	fmt.Printf("Location: %s", loc) // Output: Location(5,7)
type Location struct { //nolint:recvcheck //using for validation
	x     Coordinate
	y     Coordinate
	guard guard.ConstructorGuard
}

// NewLocation creates a new Location with the specified coordinates.
// Both x and y coordinates must be within the valid range [LocationMinX..LocationMaxX] and [LocationMinY..LocationMaxY].
// Returns an error if either coordinate is outside the valid bounds.
//
// Parameters:
//   - x: The X coordinate (must be between LocationMinX and LocationMaxX inclusive)
//   - y: The Y coordinate (must be between LocationMinY and LocationMaxY inclusive)
//
// Returns:
//   - Location: A valid location instance
//   - error: Validation error if coordinates are out of bounds
//
// Example:
//
//	loc, err := NewLocation(5, 7)
//	if err != nil {
//	    log.Fatal("Invalid coordinates:", err)
//	}
//	// loc is now ready to use
func NewLocation(x Coordinate, y Coordinate) (Location, error) {
	loc := Location{
		guard: guard.NewConstructorGuard(),
	}

	if err := errors.Join(loc.setX(x), loc.setY(y)); err != nil {
		return Location{}, err
	}

	return loc, nil
}

// NewRandomLocation creates a new Location with randomly generated coordinates.
// The coordinates are guaranteed to be within valid bounds [LocationMinX..LocationMaxX] and [LocationMinY..LocationMaxY].
// This function is useful for testing or generating random delivery locations.
//
// Returns:
//   - Location: A valid location with random coordinates
//   - error: Should never return an error since coordinates are always valid
//
// Example:
//
//	loc, err := NewRandomLocation()
//	if err != nil {
//	    log.Fatal("Unexpected error:", err) // Should never happen
//	}
//	fmt.Printf("Random location: %s", loc)
func NewRandomLocation() (Location, error) {
	x := Coordinate(rand.IntN(int(LocationMaxX-LocationMinX+1)) + int(LocationMinX)) //nolint:gosec // it's ok
	y := Coordinate(rand.IntN(int(LocationMaxY-LocationMinY+1)) + int(LocationMinY)) //nolint:gosec // it's ok
	return NewLocation(x, y)
}

// Validate checks if the Location was properly constructed using a constructor.
// The zero value of Location is invalid and will fail this validation.
// This method is primarily used internally by other methods to ensure Location integrity.
//
// Returns:
//   - error: ErrLocationIsNotConstructed if the location was not properly initialized, nil otherwise
//
// Example:
//
//	var loc Location // Zero value - invalid
//	if err := loc.Validate(); err != nil {
//	    fmt.Println("Invalid location:", err)
//	}
//
//	validLoc, _ := NewLocation(5, 7)
//	if err := validLoc.Validate(); err == nil {
//	    fmt.Println("Location is valid")
//	}
func (l Location) Validate() error {
	return l.guard.Validate(ErrLocationIsNotConstructed)
}

// X returns the X coordinate of the location.
// The returned coordinate is guaranteed to be within valid bounds [LocationMinX..LocationMaxX]
// for properly constructed Location instances.
//
// Returns:
//   - Coordinate: The X coordinate value
//
// Example:
//
//	loc, _ := NewLocation(5, 7)
//	x := loc.X() // x will be 5
func (l Location) X() Coordinate {
	return l.x
}

// Y returns the Y coordinate of the location.
// The returned coordinate is guaranteed to be within valid bounds [LocationMinY..LocationMaxY]
// for properly constructed Location instances.
//
// Returns:
//   - Coordinate: The Y coordinate value
//
// Example:
//
//	loc, _ := NewLocation(5, 7)
//	y := loc.Y() // y will be 7
func (l Location) Y() Coordinate {
	return l.y
}

// String returns a human-readable string representation of the Location.
// The format is "Location(x,y)" which is useful for debugging and logging.
// This method implements the fmt.Stringer interface.
//
// Returns:
//   - string: String representation in the format "Location(x,y)"
//
// Example:
//
//	loc, _ := NewLocation(5, 7)
//	fmt.Println(loc.String()) // Output: Location(5,7)
//	fmt.Println(loc)          // Output: Location(5,7) (automatic String() call)
func (l Location) String() string {
	return fmt.Sprintf("Location(%d,%d)", l.x, l.y)
}

// IsEqual compares two locations for equality.
// Two locations are considered equal if they have the same X and Y coordinates.
// Both locations must be properly constructed (pass validation) for the comparison to succeed.
//
// Parameters:
//   - other: The Location to compare with
//
// Returns:
//   - bool: true if locations are equal, false otherwise
//   - error: Validation error if either location is improperly constructed
//
// Example:
//
//	loc1, _ := NewLocation(5, 7)
//	loc2, _ := NewLocation(5, 7)
//	loc3, _ := NewLocation(3, 4)
//
//	equal, err := loc1.IsEqual(loc2)
//	// equal = true, err = nil
//
//	equal, err = loc1.IsEqual(loc3)
//	// equal = false, err = nil
func (l Location) IsEqual(other Location) (bool, error) {
	if err := errors.Join(l.Validate(), other.Validate()); err != nil {
		return false, err
	}

	return l == other, nil
}

// Distance calculates the Manhattan distance between two locations.
// Manhattan distance is the sum of the absolute differences of their coordinates: |x1-x2| + |y1-y2|.
// This represents the shortest path distance when movement is restricted to horizontal and vertical steps.
// Both locations must be properly constructed (pass validation) for the calculation to succeed.
//
// Parameters:
//   - other: The Location to calculate distance to
//
// Returns:
//   - int: The Manhattan distance between the two locations
//   - error: Validation error if either location is improperly constructed
//
// Example:
//
//	loc1, _ := NewLocation(1, 1)
//	loc2, _ := NewLocation(4, 5)
//
//	distance, err := loc1.Distance(loc2)
//	// distance = 7 (|1-4| + |1-5| = 3 + 4 = 7), err = nil
//
//	// Distance is symmetric
//	distance2, _ := loc2.Distance(loc1)
//	// distance2 = 7 (same as distance)
func (l Location) Distance(other Location) (int, error) {
	if err := errors.Join(l.Validate(), other.Validate()); err != nil {
		return 0, err
	}

	dx := abs(l.x - other.x)
	dy := abs(l.y - other.y)
	return int(dx + dy), nil
}

// setX sets the x coordinate with validation.
// Note: We intentionally use a pointer receiver here while other methods use value receivers.
// Although mixing receiver types is generally not recommended, in this case we use pointer
// receivers for these private setters to enable self-encapsulated validation of business
// requirements during object construction.
func (l *Location) setX(x Coordinate) error {
	if x < LocationMinX || x > LocationMaxX {
		return errs.NewValueIsOutOfRangeError("x", x, LocationMinX, LocationMaxX)
	}

	l.x = x
	return nil
}

// setY sets the y coordinate with validation.
// Note: We intentionally use a pointer receiver here while other methods use value receivers.
// Although mixing receiver types is generally not recommended, in this case we use pointer
// receivers for these private setters to enable self-encapsulated validation of business
// requirements during object construction.
func (l *Location) setY(y Coordinate) error {
	if y < LocationMinY || y > LocationMaxY {
		return errs.NewValueIsOutOfRangeError("y", y, LocationMinY, LocationMaxY)
	}

	l.y = y
	return nil
}

func abs(x Coordinate) Coordinate {
	if x < 0 {
		return -x
	}
	return x
}
