package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestBuildEnhancedModelsCatalogUsesConservativeMetadata(t *testing.T) {
	t.Parallel()

	const groupID int64 = 901
	newAccount := func(id int64, contextWindow, maxOutputTokens int64, modalities []string) Account {
		account := Account{
			ID: id, Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
			Status: StatusActive, Schedulable: true,
			Credentials: map[string]any{"model_mapping": map[string]any{"public-model": "upstream-model"}},
		}
		account.SetUpstreamModelMetadataSnapshot(UpstreamModelMetadataSnapshot{Models: map[string]UpstreamModelMetadata{
			"upstream-model": {
				ID: "upstream-model", DisplayName: "Provider Model",
				InputModalities: modalities, ContextWindow: contextWindow,
				MaxOutputTokens: maxOutputTokens,
			},
		}})
		return account
	}
	svc := &GatewayService{accountRepo: codexModelsVisibilityAccountRepo{byGroup: map[int64][]Account{
		groupID: {
			newAccount(1, 128_000, 16_384, []string{"text", "image"}),
			newAccount(2, 64_000, 8_192, []string{"text"}),
		},
	}}}

	models := svc.BuildEnhancedModelsCatalog(
		context.Background(),
		&Group{ID: groupID, Platform: PlatformOpenAI},
		"",
		[]string{"public-model"},
	)

	require.Equal(t, []EnhancedModel{{
		ID: "public-model", DisplayName: "public-model",
		SupportedAPIs:   []string{ModelAPIOpenAIResponses, ModelAPIOpenAICompletions, ModelAPIAnthropicMessages},
		InputModalities: []string{"text"}, ContextWindow: 64_000, MaxOutputTokens: 8_192,
	}}, models)
}

func TestBuildEnhancedModelsCatalogAddsGeminiNativeAPI(t *testing.T) {
	t.Parallel()

	models := (*GatewayService)(nil).BuildEnhancedModelsCatalog(
		context.Background(),
		&Group{ID: 902, Platform: PlatformGemini},
		"",
		[]string{"gemini-model"},
	)

	require.Equal(t, []string{
		ModelAPIOpenAIResponses,
		ModelAPIOpenAICompletions,
		ModelAPIAnthropicMessages,
		ModelAPIGoogleGenerativeAI,
	}, models[0].SupportedAPIs)
	require.Empty(t, models[0].InputModalities)
	require.Zero(t, models[0].ContextWindow)
	require.Zero(t, models[0].MaxOutputTokens)
}

func TestBuildEnhancedModelsCatalogHonorsClaudeCodeOnly(t *testing.T) {
	t.Parallel()

	models := (*GatewayService)(nil).BuildEnhancedModelsCatalog(
		context.Background(),
		&Group{ID: 903, Platform: PlatformOpenAI, ClaudeCodeOnly: true},
		"",
		[]string{"claude-model"},
	)

	require.Equal(t, []string{ModelAPIAnthropicMessages}, models[0].SupportedAPIs)
}

func TestBuildEnhancedModelsCatalogUsesPricingOverrideAsCapabilityFallback(t *testing.T) {
	t.Parallel()

	reasoning := true
	pricingService := &PricingService{pricingData: map[string]*LiteLLMModelPricing{
		"custom-model": {
			MaxInputTokens:            128_000,
			MaxOutputTokens:           8_192,
			SupportsReasoning:         &reasoning,
			DefaultReasoningLevel:     "medium",
			SupportedReasoningLevels:  []string{"low", "medium", "high"},
			SupportedModalities:       []string{"text"},
			CapabilityOverrideApplied: true,
		},
	}}
	svc := &GatewayService{billingService: NewBillingService(&config.Config{}, pricingService)}

	models := svc.BuildEnhancedModelsCatalog(
		context.Background(),
		&Group{ID: 904, Platform: PlatformOpenAI},
		"",
		[]string{"custom-model"},
	)

	require.Len(t, models, 1)
	model := models[0]
	require.Equal(t, []string{"text"}, model.InputModalities)
	require.EqualValues(t, 128_000, model.ContextWindow)
	require.EqualValues(t, 8_192, model.MaxOutputTokens)
	require.NotNil(t, model.Reasoning)
	require.True(t, *model.Reasoning)
	require.Equal(t, "medium", model.DefaultReasoningLevel)
	require.Equal(t, []string{"low", "medium", "high"}, model.SupportedReasoningLevels)
}

func TestBuildEnhancedModelsCatalogIgnoresNonOverridePricingCapabilities(t *testing.T) {
	t.Parallel()

	reasoning := true
	pricingService := &PricingService{pricingData: map[string]*LiteLLMModelPricing{
		"catalog-model": {
			MaxInputTokens:    128_000,
			SupportsReasoning: &reasoning,
		},
	}}
	svc := &GatewayService{billingService: NewBillingService(&config.Config{}, pricingService)}

	models := svc.BuildEnhancedModelsCatalog(
		context.Background(),
		&Group{ID: 905, Platform: PlatformOpenAI},
		"",
		[]string{"catalog-model"},
	)

	require.Len(t, models, 1)
	require.Zero(t, models[0].ContextWindow)
	require.Nil(t, models[0].Reasoning)
}
