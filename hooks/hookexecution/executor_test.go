package hookexecution

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/buger/jsonparser"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/hooks"
	"github.com/prebid/prebid-server/hooks/hookanalytics"
	"github.com/prebid/prebid-server/hooks/hookstage"
	metric_config "github.com/prebid/prebid-server/metrics/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteStages_DoesNotChangeRequestForEmptyPlan(t *testing.T) {
	body := []byte(`{"name": "John", "last_name": "Doe"}`)
	reader := bytes.NewReader(body)
	req, err := http.NewRequest(http.MethodPost, "https://prebid.com/openrtb2/auction", reader)
	if err != nil {
		t.Fatalf("Unexpected error creating http request: %s", err)
	}
	exec := HookExecutor{
		InvocationCtx: &hookstage.InvocationContext{},
		Endpoint:      EndpointAuction,
		PlanBuilder:   hooks.EmptyPlanBuilder{},
		MetricEngine:  &metric_config.NilMetricsEngine{},
	}

	newBody, reject := exec.ExecuteEntrypointStage(req, body)
	require.Nil(t, reject, "Unexpected stage reject")

	stOut := exec.GetOutcomes()
	assert.Empty(t, stOut)
	if bytes.Compare(body, newBody) != 0 {
		t.Error("request body should not change")
	}

	newBody, reject = exec.ExecuteRawAuctionStage(body)
	require.Nil(t, reject, "Unexpected stage reject")

	stOut = exec.GetOutcomes()
	assert.Empty(t, stOut)
	if bytes.Compare(body, newBody) != 0 {
		t.Error("request body should not change")
	}

	reject = exec.ExecuteRawBidderResponseStage(&adapters.BidderResponse{}, "")
	require.Nil(t, reject, "Unexpected stage reject")

	stOut = exec.GetOutcomes()
	assert.Empty(t, stOut)
}

func TestExecuteEntrypointStage_CanApplyHookMutations(t *testing.T) {
	expectedOutcome := StageOutcome{
		Entity: hookstage.EntityHttpRequest,
		Stage:  hooks.StageEntrypoint,
		Groups: []GroupOutcome{
			{
				InvocationResults: []HookOutcome{
					{
						AnalyticsTags: hookanalytics.Analytics{},
						HookID:        HookID{"foobar", "foo"},
						Status:        StatusSuccess,
						Action:        ActionUpdate,
						Message:       "",
						DebugMessages: []string{fmt.Sprintf("Hook mutation successfully applied, affected key: header.foo, mutation type: %s", hookstage.MutationUpdate)},
						Errors:        nil,
						Warnings:      nil,
					},
					{
						AnalyticsTags: hookanalytics.Analytics{},
						HookID:        HookID{"foobar", "bar"},
						Status:        StatusSuccess,
						Action:        ActionUpdate,
						Message:       "",
						DebugMessages: []string{fmt.Sprintf("Hook mutation successfully applied, affected key: param.foo, mutation type: %s", hookstage.MutationUpdate)},
						Errors:        nil,
						Warnings:      nil,
					},
				},
			},
			{
				InvocationResults: []HookOutcome{
					{
						AnalyticsTags: hookanalytics.Analytics{},
						HookID:        HookID{"foobar", "baz"},
						Status:        StatusSuccess,
						Action:        ActionUpdate,
						Message:       "",
						DebugMessages: []string{
							fmt.Sprintf("Hook mutation successfully applied, affected key: body.foo, mutation type: %s", hookstage.MutationUpdate),
							fmt.Sprintf("Hook mutation successfully applied, affected key: body.name, mutation type: %s", hookstage.MutationDelete),
						},
						Errors:   nil,
						Warnings: nil,
					},
				},
			},
		},
	}

	body := []byte(`{"name": "John", "last_name": "Doe"}`)
	reader := bytes.NewReader(body)
	req, err := http.NewRequest(http.MethodPost, "https://prebid.com/openrtb2/auction", reader)
	if err != nil {
		t.Fatalf("Unexpected error creating http request: %s", err)
	}
	exec := HookExecutor{
		InvocationCtx: &hookstage.InvocationContext{},
		Endpoint:      EndpointAuction,
		PlanBuilder:   TestApplyHookMutationsBuilder{},
		MetricEngine:  &metric_config.NilMetricsEngine{},
	}

	newBody, reject := exec.ExecuteEntrypointStage(req, body)
	require.Nil(t, reject, "Unexpected stage reject")

	stOut := exec.GetOutcomes()[0]
	assertEqualStageOutcomes(t, expectedOutcome, stOut)

	if bytes.Compare(body, newBody) == 0 {
		t.Error("request body not changed after applying hook result")
	}

	if _, dt, _, _ := jsonparser.Get(newBody, "name"); dt != jsonparser.NotExist {
		t.Error("'name' property expected to be deleted from request body.")
	}

	if req.Header.Get("foo") == "" {
		t.Error("header not changed inside hook.Call method")
	}

	if req.URL.Query().Get("foo") == "" {
		t.Error("query params not changed inside hook.Call method")
	}
}

func TestExecuteRawAuctionStage_CanApplyHookMutations(t *testing.T) {
	expectedOutcome := StageOutcome{
		Entity: hookstage.EntityAuctionRequest,
		Stage:  hooks.StageRawAuction,
		Groups: []GroupOutcome{
			{
				InvocationResults: []HookOutcome{
					{
						AnalyticsTags: hookanalytics.Analytics{},
						HookID:        HookID{"foobar", "foo"},
						Status:        StatusSuccess,
						Action:        ActionUpdate,
						Message:       "",
						DebugMessages: []string{
							fmt.Sprintf("Hook mutation successfully applied, affected key: body.foo, mutation type: %s", hookstage.MutationUpdate),
							fmt.Sprintf("Hook mutation successfully applied, affected key: body.name, mutation type: %s", hookstage.MutationDelete),
						},
						Errors:   nil,
						Warnings: nil,
					},
				},
			},
		},
	}

	body := []byte(`{"name": "John", "last_name": "Doe"}`)
	exec := HookExecutor{
		InvocationCtx: &hookstage.InvocationContext{},
		Endpoint:      EndpointAuction,
		PlanBuilder:   TestApplyHookMutationsBuilder{},
		MetricEngine:  &metric_config.NilMetricsEngine{},
	}

	newBody, reject := exec.ExecuteRawAuctionStage(body)
	require.Nil(t, reject, "Unexpected stage reject")

	stOut := exec.GetOutcomes()[0]
	assertEqualStageOutcomes(t, expectedOutcome, stOut)

	if bytes.Compare(body, newBody) == 0 {
		t.Error("request body not changed after applying hook result")
	}

	if _, dt, _, _ := jsonparser.Get(newBody, "name"); dt != jsonparser.NotExist {
		t.Error("'name' property expected to be deleted from request body.")
	}
}

func TestExecuteRawBidderResponseStage_CanApplyHookMutations(t *testing.T) {
	expectedOutcome := StageOutcome{
		Entity: "the-bidder",
		Stage:  hooks.StageRawBidderResponse,
		Groups: []GroupOutcome{
			{
				InvocationResults: []HookOutcome{
					{
						AnalyticsTags: hookanalytics.Analytics{},
						HookID:        HookID{"foobar", "foo"},
						Status:        StatusSuccess,
						Action:        ActionUpdate,
						Message:       "",
						DebugMessages: []string{
							fmt.Sprintf("Hook mutation successfully applied, affected key: bidderResponse.bid.deal-priority, mutation type: %s", hookstage.MutationUpdate),
						},
						Errors:   nil,
						Warnings: nil,
					},
				},
			},
		},
	}

	exec := HookExecutor{
		InvocationCtx: &hookstage.InvocationContext{},
		Endpoint:      EndpointAuction,
		PlanBuilder:   TestApplyHookMutationsBuilder{},
		MetricEngine:  &metric_config.NilMetricsEngine{},
	}
	resp := adapters.BidderResponse{Bids: []*adapters.TypedBid{{DealPriority: 1}}}

	reject := exec.ExecuteRawBidderResponseStage(&resp, "the-bidder")
	require.Nil(t, reject, "Unexpected stage reject")

	stOut := exec.GetOutcomes()[0]
	assertEqualStageOutcomes(t, expectedOutcome, stOut)

	if resp.Bids[0].DealPriority == 1 {
		t.Error("bidder response not changed inside hook.Call method")
	}
}

func TestExecuteEntrypointStage_CanRejectHook(t *testing.T) {
	expectedOutcome := StageOutcome{
		ExecutionTime: ExecutionTime{},
		Entity:        hookstage.EntityHttpRequest,
		Stage:         hooks.StageEntrypoint,
		Groups: []GroupOutcome{
			{
				ExecutionTime: ExecutionTime{},
				InvocationResults: []HookOutcome{
					{
						ExecutionTime: ExecutionTime{},
						AnalyticsTags: hookanalytics.Analytics{},
						HookID:        HookID{"foobar", "foo"},
						Status:        StatusSuccess,
						Action:        ActionUpdate,
						Message:       "",
						DebugMessages: []string{
							fmt.Sprintf("Hook mutation successfully applied, affected key: header.foo, mutation type: %s", hookstage.MutationUpdate),
						},
						Errors:   nil,
						Warnings: nil,
					},
				},
			},
			{
				ExecutionTime: ExecutionTime{},
				InvocationResults: []HookOutcome{
					{
						ExecutionTime: ExecutionTime{},
						AnalyticsTags: hookanalytics.Analytics{},
						HookID:        HookID{"foobar", "bar"},
						Status:        StatusSuccess,
						Action:        ActionReject,
						Message:       "",
						DebugMessages: nil,
						Errors: []string{
							`Module rejected stage, reason: ""`,
						},
						Warnings: nil,
					},
				},
			},
		},
	}

	body := []byte(`{"name": "John", "last_name": "Doe"}`)
	reader := bytes.NewReader(body)
	req, err := http.NewRequest(http.MethodPost, "https://prebid.com/openrtb2/auction", reader)
	require.NoError(t, err, "Unexpected error creating http request: %s", err)
	exec := HookExecutor{
		InvocationCtx: &hookstage.InvocationContext{},
		Endpoint:      EndpointAuction,
		PlanBuilder:   TestRejectPlanBuilder{},
		MetricEngine:  &metric_config.NilMetricsEngine{},
	}

	newBody, reject := exec.ExecuteEntrypointStage(req, body)
	require.NotNil(t, reject, "Unexpected successful execution of entrypoint hook")
	require.Equal(t, reject, &RejectError{}, "Unexpected reject returned from entrypoint hook")

	stOut := exec.GetOutcomes()[0]
	assertEqualStageOutcomes(t, expectedOutcome, stOut)
	assert.Equal(t, body, newBody, "request body shouldn't change if request rejected")
}

func TestExecuteRawAuctionStage_CanRejectHook(t *testing.T) {
	expectedOutcome := StageOutcome{
		ExecutionTime: ExecutionTime{},
		Entity:        hookstage.EntityAuctionRequest,
		Stage:         hooks.StageRawAuction,
		Groups: []GroupOutcome{
			{
				ExecutionTime: ExecutionTime{},
				InvocationResults: []HookOutcome{
					{
						ExecutionTime: ExecutionTime{},
						AnalyticsTags: hookanalytics.Analytics{},
						HookID:        HookID{"foobar", "foo"},
						Status:        StatusSuccess,
						Action:        ActionUpdate,
						Message:       "",
						DebugMessages: []string{
							fmt.Sprintf("Hook mutation successfully applied, affected key: body.foo, mutation type: %s", hookstage.MutationUpdate),
							fmt.Sprintf("Hook mutation successfully applied, affected key: body.name, mutation type: %s", hookstage.MutationDelete),
						},
						Errors:   nil,
						Warnings: nil,
					},
				},
			},
			{
				ExecutionTime: ExecutionTime{},
				InvocationResults: []HookOutcome{
					{
						ExecutionTime: ExecutionTime{},
						AnalyticsTags: hookanalytics.Analytics{},
						HookID:        HookID{"foobar", "bar"},
						Status:        StatusSuccess,
						Action:        ActionReject,
						Message:       "",
						DebugMessages: nil,
						Errors: []string{
							`Module rejected stage, reason: ""`,
						},
						Warnings: nil,
					},
				},
			},
		},
	}

	body := []byte(`{"name": "John", "last_name": "Doe"}`)
	exec := HookExecutor{
		InvocationCtx: &hookstage.InvocationContext{},
		Endpoint:      EndpointAuction,
		PlanBuilder:   TestRejectPlanBuilder{},
		MetricEngine:  &metric_config.NilMetricsEngine{},
	}

	_, reject := exec.ExecuteRawAuctionStage(body)
	require.NotNil(t, reject, "Unexpected successful execution of raw auction hook")
	require.Equal(t, reject, &RejectError{}, "Unexpected reject returned from raw auction hook")

	stOut := exec.GetOutcomes()[0]
	assertEqualStageOutcomes(t, expectedOutcome, stOut)
}

func TestExecuteRawBidderResponseStage_CanRejectHook(t *testing.T) {
	expectedOutcome := StageOutcome{
		ExecutionTime: ExecutionTime{},
		Entity:        "the-bidder",
		Stage:         hooks.StageRawBidderResponse,
		Groups: []GroupOutcome{
			{
				ExecutionTime: ExecutionTime{},
				InvocationResults: []HookOutcome{
					{
						ExecutionTime: ExecutionTime{},
						AnalyticsTags: hookanalytics.Analytics{},
						HookID:        HookID{"foobar", "foo"},
						Status:        StatusSuccess,
						Action:        ActionReject,
						Message:       "",
						DebugMessages: nil,
						Errors: []string{
							`Module rejected stage, reason: ""`,
						},
						Warnings: nil,
					},
				},
			},
		},
	}

	exec := HookExecutor{
		InvocationCtx: &hookstage.InvocationContext{},
		Endpoint:      EndpointAuction,
		PlanBuilder:   TestRejectPlanBuilder{},
		MetricEngine:  &metric_config.NilMetricsEngine{},
	}

	reject := exec.ExecuteRawBidderResponseStage(&adapters.BidderResponse{}, "the-bidder")
	require.NotNil(t, reject, "Unexpected successful execution of raw bidder response hook")
	require.Equal(t, reject, &RejectError{}, "Unexpected error returned from raw bidder response hook")

	stOut := exec.GetOutcomes()[0]
	assertEqualStageOutcomes(t, expectedOutcome, stOut)
}

func TestExecuteEntrypointStage_CanTimeoutOneOfHooks(t *testing.T) {
	expectedOutcome := StageOutcome{
		ExecutionTime: ExecutionTime{},
		Entity:        hookstage.EntityHttpRequest,
		Stage:         hooks.StageEntrypoint,
		Groups: []GroupOutcome{
			{
				ExecutionTime: ExecutionTime{},
				InvocationResults: []HookOutcome{
					{
						ExecutionTime: ExecutionTime{},
						AnalyticsTags: hookanalytics.Analytics{},
						HookID:        HookID{"foobar", "foo"},
						Status:        StatusSuccess,
						Action:        ActionUpdate,
						Message:       "",
						DebugMessages: []string{
							fmt.Sprintf("Hook mutation successfully applied, affected key: header.foo, mutation type: %s", hookstage.MutationUpdate),
						},
						Errors:   nil,
						Warnings: nil,
					},
					{
						ExecutionTime: ExecutionTime{},
						AnalyticsTags: hookanalytics.Analytics{},
						HookID:        HookID{"foobar", "bar"},
						Status:        StatusTimeout,
						Action:        "",
						Message:       "",
						DebugMessages: nil,
						Errors:        []string{"Hook execution timeout"},
						Warnings:      nil,
					},
				},
			},
			{
				ExecutionTime: ExecutionTime{},
				InvocationResults: []HookOutcome{
					{
						ExecutionTime: ExecutionTime{},
						AnalyticsTags: hookanalytics.Analytics{},
						HookID:        HookID{"foobar", "baz"},
						Status:        StatusSuccess,
						Action:        ActionUpdate,
						Message:       "",
						DebugMessages: []string{
							fmt.Sprintf("Hook mutation successfully applied, affected key: body.foo, mutation type: %s", hookstage.MutationUpdate),
							fmt.Sprintf("Hook mutation successfully applied, affected key: body.name, mutation type: %s", hookstage.MutationDelete),
						},
						Errors:   nil,
						Warnings: nil,
					},
				},
			},
		},
	}

	body := []byte(`{"name": "John", "last_name": "Doe"}`)
	reader := bytes.NewReader(body)
	req, err := http.NewRequest(http.MethodPost, "https://prebid.com/openrtb2/auction", reader)
	if err != nil {
		t.Fatalf("Unexpected error creating http request: %s", err)
	}
	exec := HookExecutor{
		InvocationCtx: &hookstage.InvocationContext{},
		Endpoint:      EndpointAuction,
		PlanBuilder:   TestWithTimeoutPlanBuilder{},
		MetricEngine:  &metric_config.NilMetricsEngine{},
	}

	newBody, reject := exec.ExecuteEntrypointStage(req, body)
	require.Nil(t, reject, "Unexpected stage reject")

	stOut := exec.GetOutcomes()[0]
	assertEqualStageOutcomes(t, expectedOutcome, stOut)

	if bytes.Compare(body, newBody) == 0 {
		t.Error("request body not changed after applying hook result")
	}

	if req.Header.Get("foo") == "" {
		t.Error("header not changed inside hook.Call method")
	}

	if req.URL.Query().Get("bar") != "" {
		t.Errorf("query params should not change inside hook.Call method because of timeout")
	}
}

func TestExecuteRawAuctionStage_CanTimeoutOneOfHooks(t *testing.T) {
	expectedOutcome := StageOutcome{
		ExecutionTime: ExecutionTime{},
		Entity:        hookstage.EntityAuctionRequest,
		Stage:         hooks.StageRawAuction,
		Groups: []GroupOutcome{
			{
				ExecutionTime: ExecutionTime{},
				InvocationResults: []HookOutcome{
					{
						ExecutionTime: ExecutionTime{},
						AnalyticsTags: hookanalytics.Analytics{},
						HookID:        HookID{"foobar", "foo"},
						Status:        StatusSuccess,
						Action:        ActionUpdate,
						Message:       "",
						DebugMessages: []string{
							fmt.Sprintf("Hook mutation successfully applied, affected key: body.foo, mutation type: %s", hookstage.MutationUpdate),
							fmt.Sprintf("Hook mutation successfully applied, affected key: body.name, mutation type: %s", hookstage.MutationDelete),
						},
						Errors:   nil,
						Warnings: nil,
					},
				},
			},
			{
				ExecutionTime: ExecutionTime{},
				InvocationResults: []HookOutcome{
					{
						ExecutionTime: ExecutionTime{},
						AnalyticsTags: hookanalytics.Analytics{},
						HookID:        HookID{"foobar", "bar"},
						Status:        StatusTimeout,
						Action:        "",
						Message:       "",
						DebugMessages: nil,
						Errors:        []string{"Hook execution timeout"},
						Warnings:      nil,
					},
				},
			},
		},
	}

	body := []byte(`{"name": "John", "last_name": "Doe"}`)
	exec := HookExecutor{
		InvocationCtx: &hookstage.InvocationContext{},
		Endpoint:      EndpointAuction,
		PlanBuilder:   TestWithTimeoutPlanBuilder{},
		MetricEngine:  &metric_config.NilMetricsEngine{},
	}

	newBody, reject := exec.ExecuteRawAuctionStage(body)
	require.Nil(t, reject, "Unexpected stage reject")

	stOut := exec.GetOutcomes()[0]
	assertEqualStageOutcomes(t, expectedOutcome, stOut)

	if bytes.Compare(body, newBody) == 0 {
		t.Error("request body not changed after applying hook result")
	}

	if _, dt, _, _ := jsonparser.Get(newBody, "name"); dt != jsonparser.NotExist {
		t.Error("'name' property expected to be deleted from request body.")
	}

	if _, dt, _, _ := jsonparser.Get(newBody, "address"); dt != jsonparser.NotExist {
		t.Error("'address' property should not be added because of timeout.")
	}
}

func TestExecuteRawBidderResponseStage_CanTimeoutOneOfHooks(t *testing.T) {
	expectedOutcome := StageOutcome{
		ExecutionTime: ExecutionTime{},
		Entity:        "the-bidder",
		Stage:         hooks.StageRawBidderResponse,
		Groups: []GroupOutcome{
			{
				ExecutionTime: ExecutionTime{},
				InvocationResults: []HookOutcome{
					{
						ExecutionTime: ExecutionTime{},
						AnalyticsTags: hookanalytics.Analytics{},
						HookID:        HookID{"foobar", "foo"},
						Status:        StatusTimeout,
						Action:        "",
						Message:       "",
						DebugMessages: nil,
						Errors:        []string{"Hook execution timeout"},
						Warnings:      nil,
					},
				},
			},
			{
				ExecutionTime: ExecutionTime{},
				InvocationResults: []HookOutcome{
					{
						ExecutionTime: ExecutionTime{},
						AnalyticsTags: hookanalytics.Analytics{},
						HookID:        HookID{"foobar", "bar"},
						Status:        StatusSuccess,
						Action:        ActionUpdate,
						Message:       "",
						DebugMessages: []string{
							fmt.Sprintf("Hook mutation successfully applied, affected key: bidderResponse.bid.deal-priority, mutation type: %s", hookstage.MutationUpdate),
						},
						Errors:   nil,
						Warnings: nil,
					},
				},
			},
		},
	}

	exec := HookExecutor{
		InvocationCtx: &hookstage.InvocationContext{},
		Endpoint:      EndpointAuction,
		PlanBuilder:   TestWithTimeoutPlanBuilder{},
		MetricEngine:  &metric_config.NilMetricsEngine{},
	}
	resp := adapters.BidderResponse{Bids: []*adapters.TypedBid{{DealPriority: 1}}}

	reject := exec.ExecuteRawBidderResponseStage(&resp, "the-bidder")
	require.Nil(t, reject, "Unexpected stage reject")

	stOut := exec.GetOutcomes()[0]
	assertEqualStageOutcomes(t, expectedOutcome, stOut)

	if resp.Bids[0].BidMeta != nil {
		t.Error("bidder response should not change because of timeout")
	}

	if resp.Bids[0].DealPriority == 1 {
		t.Error("bidder response not changed inside hook.Call method")
	}
}

func TestExecuteEntrypointStage_ModuleContextsAreCreated(t *testing.T) {
	body := []byte(`{"name": "John", "last_name": "Doe"}`)
	reader := bytes.NewReader(body)
	req, err := http.NewRequest(http.MethodPost, "https://prebid.com/openrtb2/auction", reader)
	if err != nil {
		t.Fatalf("Unexpected error creating http request: %s", err)
	}

	exec := HookExecutor{
		InvocationCtx: &hookstage.InvocationContext{},
		Endpoint:      EndpointAuction,
		PlanBuilder:   TestWithModuleContextsPlanBuilder{},
		MetricEngine:  &metric_config.NilMetricsEngine{},
	}
	_, reject := exec.ExecuteEntrypointStage(req, body)
	require.Nil(t, reject, "Unexpected stage reject")

	checkModuleContexts(t, exec)
}

func TestExecuteRawAuctionStage_ModuleContextsAreCreated(t *testing.T) {
	body := []byte(`{"name": "John", "last_name": "Doe"}`)

	exec := HookExecutor{
		InvocationCtx: &hookstage.InvocationContext{},
		Endpoint:      EndpointAuction,
		PlanBuilder:   TestWithModuleContextsPlanBuilder{},
		MetricEngine:  &metric_config.NilMetricsEngine{},
	}
	_, reject := exec.ExecuteRawAuctionStage(body)
	require.Nil(t, reject, "Unexpected stage reject")

	checkModuleContexts(t, exec)
}

func TestExecuteRawBidderResponseStage_ModuleContextsAreCreated(t *testing.T) {
	exec := HookExecutor{
		InvocationCtx: &hookstage.InvocationContext{},
		Endpoint:      EndpointAuction,
		PlanBuilder:   TestWithModuleContextsPlanBuilder{},
		MetricEngine:  &metric_config.NilMetricsEngine{},
	}
	reject := exec.ExecuteRawBidderResponseStage(&adapters.BidderResponse{}, "the-bidder")
	require.Nil(t, reject, "Unexpected stage reject")

	checkModuleContexts(t, exec)
}

func checkModuleContexts(t *testing.T, exec HookExecutor) {
	stOut := exec.GetOutcomes()[0]
	if len(stOut.Groups) != 2 {
		t.Error("some hook groups have not been processed")
	}

	ctx1 := exec.InvocationCtx.ModuleContextFor("module-1")
	if ctx1.Ctx["some-ctx-1"] != "some-ctx-1" {
		t.Error("context for module-1 not created")
	}

	ctx2 := exec.InvocationCtx.ModuleContextFor("module-2")
	if ctx2.Ctx["some-ctx-2"] != "some-ctx-2" {
		t.Error("context for module-2 not created")
	}
}

type TestApplyHookMutationsBuilder struct {
	hooks.EmptyPlanBuilder
}

func (e TestApplyHookMutationsBuilder) PlanForEntrypointStage(_ string) hooks.Plan[hookstage.Entrypoint] {
	return hooks.Plan[hookstage.Entrypoint]{
		hooks.Group[hookstage.Entrypoint]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.Entrypoint]{
				{Module: "foobar", Code: "foo", Hook: mockUpdateHeaderEntrypointHook{}},
				{Module: "foobar", Code: "bar", Hook: mockUpdateQueryEntrypointHook{}},
			},
		},
		hooks.Group[hookstage.Entrypoint]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.Entrypoint]{
				{Module: "foobar", Code: "baz", Hook: mockUpdateBodyHook{}},
			},
		},
	}
}

func (e TestApplyHookMutationsBuilder) PlanForRawAuctionStage(_ string, _ *config.Account) hooks.Plan[hookstage.RawAuction] {
	return hooks.Plan[hookstage.RawAuction]{
		hooks.Group[hookstage.RawAuction]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.RawAuction]{
				{Module: "foobar", Code: "foo", Hook: mockUpdateBodyHook{}},
			},
		},
	}
}

func (e TestApplyHookMutationsBuilder) PlanForRawBidderResponseStage(_ string, _ *config.Account) hooks.Plan[hookstage.RawBidderResponse] {
	return hooks.Plan[hookstage.RawBidderResponse]{
		hooks.Group[hookstage.RawBidderResponse]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.RawBidderResponse]{
				{Module: "foobar", Code: "foo", Hook: mockUpdateBidderResponseHook{}},
			},
		},
	}
}

type TestRejectPlanBuilder struct {
	hooks.EmptyPlanBuilder
}

func (e TestRejectPlanBuilder) PlanForEntrypointStage(_ string) hooks.Plan[hookstage.Entrypoint] {
	return hooks.Plan[hookstage.Entrypoint]{
		hooks.Group[hookstage.Entrypoint]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.Entrypoint]{
				{Module: "foobar", Code: "foo", Hook: mockUpdateHeaderEntrypointHook{}},
			},
		},
		hooks.Group[hookstage.Entrypoint]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.Entrypoint]{
				{Module: "foobar", Code: "bar", Hook: mockRejectHook{}},
			},
		},
	}
}

func (e TestRejectPlanBuilder) PlanForRawAuctionStage(_ string, _ *config.Account) hooks.Plan[hookstage.RawAuction] {
	return hooks.Plan[hookstage.RawAuction]{
		hooks.Group[hookstage.RawAuction]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.RawAuction]{
				{Module: "foobar", Code: "foo", Hook: mockUpdateBodyHook{}},
			},
		},
		hooks.Group[hookstage.RawAuction]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.RawAuction]{
				{Module: "foobar", Code: "bar", Hook: mockRejectHook{}},
			},
		},
	}
}

func (e TestRejectPlanBuilder) PlanForRawBidderResponseStage(_ string, _ *config.Account) hooks.Plan[hookstage.RawBidderResponse] {
	return hooks.Plan[hookstage.RawBidderResponse]{
		hooks.Group[hookstage.RawBidderResponse]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.RawBidderResponse]{
				{Module: "foobar", Code: "foo", Hook: mockRejectHook{}},
			},
		},
	}
}

type TestWithTimeoutPlanBuilder struct {
	hooks.EmptyPlanBuilder
}

func (e TestWithTimeoutPlanBuilder) PlanForEntrypointStage(_ string) hooks.Plan[hookstage.Entrypoint] {
	return hooks.Plan[hookstage.Entrypoint]{
		hooks.Group[hookstage.Entrypoint]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.Entrypoint]{
				{Module: "foobar", Code: "foo", Hook: mockUpdateHeaderEntrypointHook{}},
				{Module: "foobar", Code: "bar", Hook: mockTimeoutHook{}},
			},
		},
		hooks.Group[hookstage.Entrypoint]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.Entrypoint]{
				{Module: "foobar", Code: "baz", Hook: mockUpdateBodyHook{}},
			},
		},
	}
}

func (e TestWithTimeoutPlanBuilder) PlanForRawAuctionStage(_ string, _ *config.Account) hooks.Plan[hookstage.RawAuction] {
	return hooks.Plan[hookstage.RawAuction]{
		hooks.Group[hookstage.RawAuction]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.RawAuction]{
				{Module: "foobar", Code: "foo", Hook: mockUpdateBodyHook{}},
			},
		},
		hooks.Group[hookstage.RawAuction]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.RawAuction]{
				{Module: "foobar", Code: "bar", Hook: mockTimeoutHook{}},
			},
		},
	}
}

func (e TestWithTimeoutPlanBuilder) PlanForRawBidderResponseStage(_ string, _ *config.Account) hooks.Plan[hookstage.RawBidderResponse] {
	return hooks.Plan[hookstage.RawBidderResponse]{
		hooks.Group[hookstage.RawBidderResponse]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.RawBidderResponse]{
				{Module: "foobar", Code: "foo", Hook: mockTimeoutHook{}},
			},
		},
		hooks.Group[hookstage.RawBidderResponse]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.RawBidderResponse]{
				{Module: "foobar", Code: "bar", Hook: mockUpdateBidderResponseHook{}},
			},
		},
	}
}

type TestWithModuleContextsPlanBuilder struct {
	hooks.EmptyPlanBuilder
}

func (e TestWithModuleContextsPlanBuilder) PlanForEntrypointStage(_ string) hooks.Plan[hookstage.Entrypoint] {
	return hooks.Plan[hookstage.Entrypoint]{
		hooks.Group[hookstage.Entrypoint]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.Entrypoint]{
				{Module: "module-1", Code: "foo", Hook: mockModuleContextHook1{}},
			},
		},
		hooks.Group[hookstage.Entrypoint]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.Entrypoint]{
				{Module: "module-2", Code: "bar", Hook: mockModuleContextHook2{}},
			},
		},
	}
}

func (e TestWithModuleContextsPlanBuilder) PlanForRawAuctionStage(_ string, _ *config.Account) hooks.Plan[hookstage.RawAuction] {
	return hooks.Plan[hookstage.RawAuction]{
		hooks.Group[hookstage.RawAuction]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.RawAuction]{
				{Module: "module-1", Code: "foo", Hook: mockModuleContextHook1{}},
			},
		},
		hooks.Group[hookstage.RawAuction]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.RawAuction]{
				{Module: "module-2", Code: "bar", Hook: mockModuleContextHook2{}},
			},
		},
	}
}

func (e TestWithModuleContextsPlanBuilder) PlanForRawBidderResponseStage(_ string, _ *config.Account) hooks.Plan[hookstage.RawBidderResponse] {
	return hooks.Plan[hookstage.RawBidderResponse]{
		hooks.Group[hookstage.RawBidderResponse]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.RawBidderResponse]{
				{Module: "module-1", Code: "foo", Hook: mockModuleContextHook1{}},
			},
		},
		hooks.Group[hookstage.RawBidderResponse]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.RawBidderResponse]{
				{Module: "module-2", Code: "bar", Hook: mockModuleContextHook2{}},
			},
		},
	}
}