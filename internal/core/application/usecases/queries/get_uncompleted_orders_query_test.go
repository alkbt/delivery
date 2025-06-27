package queries_test

import (
	"testing"

	"delivery/internal/core/application/usecases/queries"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGetUncompletedOrdersQuery_Valid(t *testing.T) {
	query := queries.NewGetUncompletedOrdersQuery()
	err := query.Validate()
	require.NoError(t, err)
}

func TestGetUncompletedOrdersQuery_NotConstructedViaConstructor(t *testing.T) {
	query := queries.GetUncompletedOrdersQuery{}
	err := query.Validate()
	require.Error(t, err)
	assert.ErrorIs(t, err, queries.ErrGetUncompletedOrdersQueryIsNotConstructed)
}
