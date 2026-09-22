package service

import (
	"context"
	"strings"
)

const (
	ModelAPIAnthropicMessages  = "anthropic-messages"
	ModelAPIOpenAICompletions  = "openai-completions"
	ModelAPIOpenAIResponses    = "openai-responses"
	ModelAPIGoogleGenerativeAI = "google-generative-ai"
)

// EnhancedModel describes the effective model capabilities exposed by the
// Sub2API gateway. Optional capability fields are omitted when the persisted
// upstream metadata cannot establish them for every account that may serve the
// public model alias.
type EnhancedModel struct {
	ID                       string   `json:"id"`
	DisplayName              string   `json:"display_name"`
	SupportedAPIs            []string `json:"supported_apis,omitempty"`
	InputModalities          []string `json:"input_modalities,omitempty"`
	ContextWindow            int64    `json:"context_window,omitempty"`
	MaxOutputTokens          int64    `json:"max_output_tokens,omitempty"`
	Reasoning                *bool    `json:"reasoning,omitempty"`
	DefaultReasoningLevel    string   `json:"default_reasoning_level,omitempty"`
	SupportedReasoningLevels []string `json:"supported_reasoning_levels,omitempty"`
}

// BuildEnhancedModelsCatalog enriches a caller-selected visible model list
// with the conservative capability intersection already used by the Codex
// manifest builder.
func (s *GatewayService) BuildEnhancedModelsCatalog(
	ctx context.Context,
	group *Group,
	platformOverride string,
	modelIDs []string,
) []EnhancedModel {
	effectivePlatform := strings.TrimSpace(platformOverride)
	if effectivePlatform == "" && group != nil {
		effectivePlatform = group.Platform
	}

	var accounts []Account
	var compositeRoutes []CompositeModelRoute
	compositeRoutesAvailable := true
	if s != nil && s.accountRepo != nil && group != nil {
		_, catalog, err := loadCodexGroupCatalogAccounts(ctx, s.accountRepo, group.ID)
		if err == nil {
			accounts = catalog
		}
		if effectivePlatform == PlatformComposite && s.compositeResolver != nil && s.compositeResolver.repo != nil {
			routes, routeErr := s.compositeResolver.repo.ListByGroup(ctx, group.ID, false)
			if routeErr != nil {
				compositeRoutesAvailable = false
			} else {
				compositeRoutes = routes
			}
		}
	}

	models := make([]EnhancedModel, 0, len(modelIDs))
	seen := make(map[string]struct{}, len(modelIDs))
	for _, rawID := range modelIDs {
		modelID := strings.TrimSpace(rawID)
		if modelID == "" {
			continue
		}
		if _, exists := seen[modelID]; exists {
			continue
		}
		seen[modelID] = struct{}{}

		model := EnhancedModel{
			ID:            modelID,
			DisplayName:   modelID,
			SupportedAPIs: s.enhancedModelSupportedAPIs(ctx, group, effectivePlatform, modelID),
		}
		if metadata, ok := groupCodexModelMetadata(
			effectivePlatform,
			modelID,
			accounts,
			group,
			compositeRoutes,
			compositeRoutesAvailable,
		); ok {
			if name := strings.TrimSpace(metadata.DisplayName); name != "" {
				model.DisplayName = name
			}
			model.InputModalities = append([]string(nil), metadata.InputModalities...)
			model.ContextWindow = metadata.ContextWindow
			model.MaxOutputTokens = metadata.MaxOutputTokens
			model.Reasoning = cloneBool(metadata.Reasoning)
			model.DefaultReasoningLevel = metadata.DefaultReasoningLevel
			model.SupportedReasoningLevels = append([]string(nil), metadata.SupportedReasoningLevels...)
		}
		s.applyEnhancedModelPricingOverrideFallback(&model)
		models = append(models, model)
	}
	return models
}

func (s *GatewayService) applyEnhancedModelPricingOverrideFallback(model *EnhancedModel) {
	if model == nil || s == nil || s.billingService == nil || s.billingService.pricingService == nil {
		return
	}
	pricing := s.billingService.pricingService.GetIdentifiedModelPricing(model.ID)
	if pricing == nil || !pricing.CapabilityOverrideApplied {
		return
	}
	if len(model.InputModalities) == 0 {
		model.InputModalities = append([]string(nil), pricing.SupportedModalities...)
	}
	if model.ContextWindow <= 0 {
		model.ContextWindow = pricing.MaxInputTokens
	}
	if model.MaxOutputTokens <= 0 {
		model.MaxOutputTokens = pricing.MaxOutputTokens
	}
	if model.Reasoning == nil {
		model.Reasoning = cloneBool(pricing.SupportsReasoning)
	}
	if model.DefaultReasoningLevel == "" {
		model.DefaultReasoningLevel = pricing.DefaultReasoningLevel
	}
	if len(model.SupportedReasoningLevels) == 0 {
		model.SupportedReasoningLevels = append([]string(nil), pricing.SupportedReasoningLevels...)
	}
}

func cloneBool(value *bool) *bool {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func (s *GatewayService) enhancedModelSupportedAPIs(
	ctx context.Context,
	group *Group,
	platform string,
	modelID string,
) []string {
	if group != nil && group.ClaudeCodeOnly {
		return []string{ModelAPIAnthropicMessages}
	}
	if platform == PlatformComposite {
		if s == nil || s.compositeResolver == nil || group == nil {
			return nil
		}
		endpoints := []struct {
			endpoint string
			api      string
		}{
			{CompositeRouteEndpointResponses, ModelAPIOpenAIResponses},
			{CompositeRouteEndpointChatCompletions, ModelAPIOpenAICompletions},
			{CompositeRouteEndpointMessages, ModelAPIAnthropicMessages},
			{CompositeRouteEndpointGemini, ModelAPIGoogleGenerativeAI},
		}
		apis := make([]string, 0, len(endpoints))
		for _, candidate := range endpoints {
			decision, err := s.compositeResolver.Resolve(ctx, group.ID, modelID, candidate.endpoint)
			if err == nil && decision.Matched {
				apis = append(apis, candidate.api)
			}
		}
		return apis
	}

	apis := []string{
		ModelAPIOpenAIResponses,
		ModelAPIOpenAICompletions,
		ModelAPIAnthropicMessages,
	}
	if platform == PlatformGemini || platform == PlatformAntigravity {
		apis = append(apis, ModelAPIGoogleGenerativeAI)
	}
	return apis
}
