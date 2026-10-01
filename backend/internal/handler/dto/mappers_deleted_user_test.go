package dto

import (
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestUserFromServiceShallow_MapsDeletedAt(t *testing.T) {
	ts := time.Date(2026, 5, 28, 10, 0, 0, 0, time.UTC)

	deleted := UserFromServiceShallow(&service.User{ID: 1, Email: "d@test.com", DeletedAt: &ts})
	require.NotNil(t, deleted.DeletedAt)
	require.Equal(t, ts, *deleted.DeletedAt)

	active := UserFromServiceShallow(&service.User{ID: 2, Email: "a@test.com"})
	require.Nil(t, active.DeletedAt, "active user must have nil DeletedAt")
}

func TestUserFromServiceShallow_ExposesEffectiveDiscountMultiplier(t *testing.T) {
	discounted := UserFromServiceShallow(&service.User{ID: 1, DiscountMultiplier: 0.8})
	require.InDelta(t, 0.8, discounted.DiscountMultiplier, 1e-12)

	legacy := UserFromServiceShallow(&service.User{ID: 2})
	require.InDelta(t, 1.0, legacy.DiscountMultiplier, 1e-12)
}
