package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeGroupModelPricingPreservesOfficialPriceMultiplier(t *testing.T) {
	multiplier := 1.75

	pricing, err := normalizeGroupModelPricing(PlatformOpenAI, []ChannelModelPricing{{
		Models:                  []string{"gpt-5.1"},
		BillingMode:             BillingModeToken,
		OfficialPriceMultiplier: &multiplier,
	}})

	require.NoError(t, err)
	require.Len(t, pricing, 1)
	require.NotNil(t, pricing[0].OfficialPriceMultiplier)
	require.Equal(t, multiplier, *pricing[0].OfficialPriceMultiplier)
}

func TestNormalizeGroupModelPricingRejectsInvalidOfficialPriceMultiplier(t *testing.T) {
	multiplier := 0.0

	_, err := normalizeGroupModelPricing(PlatformOpenAI, []ChannelModelPricing{{
		Models:                  []string{"gpt-5.1"},
		BillingMode:             BillingModeToken,
		OfficialPriceMultiplier: &multiplier,
	}})

	require.ErrorContains(t, err, "official price multiplier must be greater than zero")
}
