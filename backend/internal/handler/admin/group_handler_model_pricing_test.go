package admin

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type modelPricingAdminServiceStub struct {
	*stubAdminService
	group         *service.Group
	getGroupCalls int
	updatedID     int64
	updatedInput  *service.UpdateGroupInput
}

func (s *modelPricingAdminServiceStub) GetGroup(_ context.Context, id int64) (*service.Group, error) {
	s.getGroupCalls++
	return s.group, nil
}

func (s *modelPricingAdminServiceStub) UpdateGroup(_ context.Context, id int64, input *service.UpdateGroupInput) (*service.Group, error) {
	s.updatedID = id
	s.updatedInput = input
	updated := *s.group
	if input.ModelPricing != nil {
		updated.ModelPricing = cloneGroupModelPricing(*input.ModelPricing)
	}
	return &updated, nil
}

func newModelPricingGroupRouter(svc service.AdminService, cfg *config.Config) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewGroupHandlerWithConfig(svc, nil, nil, cfg)
	r := gin.New()
	r.PUT("/groups/:id/model-pricing/:index", h.SaveModelPricingEntry)
	r.DELETE("/groups/:id/model-pricing/:index", h.DeleteModelPricingEntry)
	return r
}

func modelPricingGroupFixture() *service.Group {
	return &service.Group{
		ID:       7,
		Name:     "priced",
		Platform: service.PlatformOpenAI,
		Status:   service.StatusActive,
		ModelPricing: []service.ChannelModelPricing{
			{Platform: service.PlatformOpenAI, Models: []string{"gpt-5"}, BillingMode: service.BillingModeToken},
			{Platform: service.PlatformOpenAI, Models: []string{"gpt-4.1"}, BillingMode: service.BillingModeToken},
		},
	}
}

func performModelPricingRequest(r http.Handler, method, path, body string) *httptest.ResponseRecorder {
	res := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(res, req)
	return res
}

func requireOnlyModelPricingUpdate(t *testing.T, input *service.UpdateGroupInput) []service.ChannelModelPricing {
	t.Helper()
	require.NotNil(t, input)
	require.NotNil(t, input.ModelPricing)
	require.Equal(t, service.UpdateGroupInput{ModelPricing: input.ModelPricing}, *input)
	return *input.ModelPricing
}

func TestSaveModelPricingEntryReplacesOneEntry(t *testing.T) {
	group := modelPricingGroupFixture()
	svc := &modelPricingAdminServiceStub{stubAdminService: newStubAdminService(), group: group}
	r := newModelPricingGroupRouter(svc, nil)

	res := performModelPricingRequest(r, http.MethodPut, "/groups/7/model-pricing/0", `{"pricing":{"models":["  gpt-5.1  ",""],"billing_mode":"token","input_price":2,"official_price_multiplier":1.75}}`)

	require.Equal(t, http.StatusOK, res.Code)
	require.Equal(t, int64(7), svc.updatedID)
	pricing := requireOnlyModelPricingUpdate(t, svc.updatedInput)
	require.Len(t, pricing, 2)
	require.Equal(t, []string{"gpt-5.1"}, pricing[0].Models)
	require.NotNil(t, pricing[0].InputPrice)
	require.Equal(t, 2.0, *pricing[0].InputPrice)
	require.NotNil(t, pricing[0].OfficialPriceMultiplier)
	require.Equal(t, 1.75, *pricing[0].OfficialPriceMultiplier)
	require.Equal(t, []string{"gpt-4.1"}, pricing[1].Models)

	pricing[1].Models[0] = "changed after update"
	require.Equal(t, "gpt-4.1", group.ModelPricing[1].Models[0])
}

func TestSaveModelPricingEntryAppendsAtCurrentLength(t *testing.T) {
	group := modelPricingGroupFixture()
	svc := &modelPricingAdminServiceStub{stubAdminService: newStubAdminService(), group: group}
	r := newModelPricingGroupRouter(svc, nil)

	res := performModelPricingRequest(r, http.MethodPut, "/groups/7/model-pricing/2", `{"pricing":{"models":["o3"],"billing_mode":"token","output_price":8}}`)

	require.Equal(t, http.StatusOK, res.Code)
	pricing := requireOnlyModelPricingUpdate(t, svc.updatedInput)
	require.Len(t, pricing, 3)
	require.Equal(t, []string{"o3"}, pricing[2].Models)
	require.NotNil(t, pricing[2].OutputPrice)
	require.Equal(t, 8.0, *pricing[2].OutputPrice)
}

func TestDeleteModelPricingEntryRemovesOnlySelectedEntry(t *testing.T) {
	group := modelPricingGroupFixture()
	svc := &modelPricingAdminServiceStub{stubAdminService: newStubAdminService(), group: group}
	r := newModelPricingGroupRouter(svc, nil)

	res := performModelPricingRequest(r, http.MethodDelete, "/groups/7/model-pricing/0", "")

	require.Equal(t, http.StatusOK, res.Code)
	require.Equal(t, int64(7), svc.updatedID)
	pricing := requireOnlyModelPricingUpdate(t, svc.updatedInput)
	require.Len(t, pricing, 1)
	require.Equal(t, []string{"gpt-4.1"}, pricing[0].Models)
	require.Equal(t, []string{"gpt-5"}, group.ModelPricing[0].Models)
}

func TestModelPricingEntryRejectsOutOfRangeIndexes(t *testing.T) {
	for _, tt := range []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "replace beyond length", method: http.MethodPut, path: "/groups/7/model-pricing/3", body: `{"pricing":{"models":["o3"]}}`},
		{name: "delete at length", method: http.MethodDelete, path: "/groups/7/model-pricing/2"},
		{name: "negative index", method: http.MethodPut, path: "/groups/7/model-pricing/-1", body: `{"pricing":{"models":["o3"]}}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			svc := &modelPricingAdminServiceStub{stubAdminService: newStubAdminService(), group: modelPricingGroupFixture()}
			res := performModelPricingRequest(newModelPricingGroupRouter(svc, nil), tt.method, tt.path, tt.body)

			require.Equal(t, http.StatusBadRequest, res.Code)
			require.Nil(t, svc.updatedInput)
		})
	}
}

func TestSaveModelPricingEntryRejectsMissingOrEmptyModels(t *testing.T) {
	for _, body := range []string{
		`{"pricing":{"models":[]}}`,
		`{"pricing":{"models":["","   "]}}`,
		`{"models":["gpt-5"]}`,
	} {
		t.Run(body, func(t *testing.T) {
			svc := &modelPricingAdminServiceStub{stubAdminService: newStubAdminService(), group: modelPricingGroupFixture()}
			res := performModelPricingRequest(newModelPricingGroupRouter(svc, nil), http.MethodPut, "/groups/7/model-pricing/0", body)

			require.Equal(t, http.StatusBadRequest, res.Code)
			require.Nil(t, svc.updatedInput)
		})
	}
}

func TestModelPricingEntryRejectsInvalidGroupID(t *testing.T) {
	for _, id := range []string{"bad", "0", "-1"} {
		svc := &modelPricingAdminServiceStub{stubAdminService: newStubAdminService(), group: modelPricingGroupFixture()}
		res := performModelPricingRequest(newModelPricingGroupRouter(svc, nil), http.MethodDelete, "/groups/"+id+"/model-pricing/0", "")

		require.Equal(t, http.StatusBadRequest, res.Code)
		require.Zero(t, svc.getGroupCalls)
		require.Nil(t, svc.updatedInput)
	}
}

func TestModelPricingEntryRejectsSimpleMode(t *testing.T) {
	svc := &modelPricingAdminServiceStub{stubAdminService: newStubAdminService(), group: modelPricingGroupFixture()}
	cfg := &config.Config{RunMode: config.RunModeSimple}
	res := performModelPricingRequest(newModelPricingGroupRouter(svc, cfg), http.MethodDelete, "/groups/7/model-pricing/0", "")

	require.Equal(t, http.StatusForbidden, res.Code)
	require.Zero(t, svc.getGroupCalls)
	require.Nil(t, svc.updatedInput)
}
