package queries_test

import (
	"testing"

	"delivery/internal/core/application/usecases/queries"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGetAllCouriersQuery_Valid(t *testing.T) {
	query := queries.NewGetAllCouriersQuery()
	err := query.Validate()
	require.NoError(t, err)
}

func TestGetAllCouriersQuery_NotConstructedViaConstructor(t *testing.T) {
	query := queries.GetAllCouriersQuery{}
	err := query.Validate()
	require.Error(t, err)
	assert.ErrorIs(t, err, queries.ErrGetAllCouriersQueryIsNotConstructed)
}
