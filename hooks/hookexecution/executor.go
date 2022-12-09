package hookexecution

import (
	"context"
	"net/http"
	"sync"

	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/hooks"
	"github.com/prebid/prebid-server/hooks/hookstage"
	"github.com/prebid/prebid-server/metrics"
)

const (
	EndpointAuction = "/openrtb2/auction"
	EndpointAmp     = "/openrtb2/amp"
)

// An entity specifies the type of object that was processed during the execution of the stage.
type entity string

const (
	entityHttpRequest              entity = "http-request"
	entityAuctionRequest           entity = "auction-request"
	entityAuctionResponse          entity = "auction-response"
	entityAllProcessedBidResponses entity = "all-processed-bid-responses"
)

type StageExecutor interface {
	ExecuteEntrypointStage(req *http.Request, body []byte) ([]byte, *RejectError)
	ExecuteRawAuctionStage(body []byte) ([]byte, *RejectError)
	ExecuteBidderRequestStage(req *openrtb2.BidRequest, bidder string) *RejectError
	ExecuteRawBidderResponseStage(response *adapters.BidderResponse, bidder string) *RejectError
}

type HookStageExecutor interface {
	StageExecutor
	SetAccount(account *config.Account)
	GetOutcomes() []StageOutcome
}

type hookExecutor struct {
	account        *config.Account
	accountId      string
	endpoint       string
	planBuilder    hooks.ExecutionPlanBuilder
	stageOutcomes  []StageOutcome
	moduleContexts *moduleContexts
	metricEngine   metrics.MetricsEngine
	// Mutex needed for BidderRequest and RawBidderResponse Stages as they are run in several goroutines
	sync.Mutex
}

func NewHookExecutor(builder hooks.ExecutionPlanBuilder, endpoint string, me metrics.MetricsEngine) *hookExecutor {
	return &hookExecutor{
		endpoint:       endpoint,
		planBuilder:    builder,
		stageOutcomes:  []StageOutcome{},
		moduleContexts: &moduleContexts{ctxs: make(map[string]hookstage.ModuleContext)},
		metricEngine:   me,
	}
}

func (e *hookExecutor) SetAccount(account *config.Account) {
	if account == nil {
		return
	}

	e.account = account
	e.accountId = account.ID
}

func (e *hookExecutor) GetOutcomes() []StageOutcome {
	return e.stageOutcomes
}

func (e *hookExecutor) ExecuteEntrypointStage(req *http.Request, body []byte) ([]byte, *RejectError) {
	plan := e.planBuilder.PlanForEntrypointStage(e.endpoint)
	if len(plan) == 0 {
		return body, nil
	}

	handler := func(
		ctx context.Context,
		moduleCtx hookstage.ModuleInvocationContext,
		hook hookstage.Entrypoint,
		payload hookstage.EntrypointPayload,
	) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
		return hook.HandleEntrypointHook(ctx, moduleCtx, payload)
	}

	stageName := hooks.StageEntrypoint.String()
	executionCtx := e.newContext(stageName)
	payload := hookstage.EntrypointPayload{Request: req, Body: body}

	outcome, payload, contexts, rejectErr := executeStage(executionCtx, plan, payload, handler, e.metricEngine)
	outcome.Entity = entityHttpRequest
	outcome.Stage = stageName

	e.saveModuleContexts(contexts)
	e.pushStageOutcome(outcome)

	return payload.Body, rejectErr
}

func (e *hookExecutor) ExecuteRawAuctionStage(requestBody []byte) ([]byte, *RejectError) {
	plan := e.planBuilder.PlanForRawAuctionStage(e.endpoint, e.account)
	if len(plan) == 0 {
		return requestBody, nil
	}

	handler := func(
		ctx context.Context,
		moduleCtx hookstage.ModuleInvocationContext,
		hook hookstage.RawAuctionRequest,
		payload hookstage.RawAuctionRequestPayload,
	) (hookstage.HookResult[hookstage.RawAuctionRequestPayload], error) {
		return hook.HandleRawAuctionHook(ctx, moduleCtx, payload)
	}

	stageName := hooks.StageRawAuction.String()
	executionCtx := e.newContext(stageName)
	payload := hookstage.RawAuctionRequestPayload(requestBody)

	outcome, payload, contexts, reject := executeStage(executionCtx, plan, payload, handler, e.metricEngine)
	outcome.Entity = entityAuctionRequest
	outcome.Stage = stageName

	e.saveModuleContexts(contexts)
	e.pushStageOutcome(outcome)

	return payload, reject
}

func (e *hookExecutor) ExecuteBidderRequestStage(req *openrtb2.BidRequest, bidder string) *RejectError {
	plan := e.planBuilder.PlanForBidderRequestStage(e.endpoint, e.account)
	if len(plan) == 0 {
		return nil
	}

	handler := func(
		ctx context.Context,
		moduleCtx hookstage.ModuleInvocationContext,
		hook hookstage.BidderRequest,
		payload hookstage.BidderRequestPayload,
	) (hookstage.HookResult[hookstage.BidderRequestPayload], error) {
		return hook.HandleBidderRequestHook(ctx, moduleCtx, payload)
	}

	stageName := hooks.StageBidderRequest.String()
	executionCtx := e.newContext(stageName)
	payload := hookstage.BidderRequestPayload{BidRequest: req, Bidder: bidder}
	outcome, payload, contexts, reject := executeStage(executionCtx, plan, payload, handler, e.metricEngine)
	outcome.Entity = entity(bidder)
	outcome.Stage = stageName

	e.saveModuleContexts(contexts)
	e.pushStageOutcome(outcome)

	return reject
}

func (e *hookExecutor) ExecuteRawBidderResponseStage(response *adapters.BidderResponse, bidder string) *RejectError {
	plan := e.planBuilder.PlanForRawBidderResponseStage(e.endpoint, e.account)
	if len(plan) == 0 {
		return nil
	}

	handler := func(
		ctx context.Context,
		moduleCtx hookstage.ModuleInvocationContext,
		hook hookstage.RawBidderResponse,
		payload hookstage.RawBidderResponsePayload,
	) (hookstage.HookResult[hookstage.RawBidderResponsePayload], error) {
		return hook.HandleRawBidderResponseHook(ctx, moduleCtx, payload)
	}

	stageName := hooks.StageRawBidderResponse.String()
	executionCtx := e.newContext(stageName)
	payload := hookstage.RawBidderResponsePayload{Bids: response.Bids, Bidder: bidder}

	outcome, _, contexts, reject := executeStage(executionCtx, plan, payload, handler, e.metricEngine)
	outcome.Entity = entity(bidder)
	outcome.Stage = stageName

	e.saveModuleContexts(contexts)
	e.pushStageOutcome(outcome)

	return reject
}

func (e *hookExecutor) newContext(stage string) executionContext {
	return executionContext{
		account:        e.account,
		accountId:      e.accountId,
		endpoint:       e.endpoint,
		moduleContexts: e.moduleContexts,
		stage:          stage,
	}
}

func (e *hookExecutor) saveModuleContexts(ctxs stageModuleContext) {
	for _, moduleCtxs := range ctxs.groupCtx {
		for moduleName, moduleCtx := range moduleCtxs {
			e.moduleContexts.put(moduleName, moduleCtx)
		}
	}
}

func (e *hookExecutor) pushStageOutcome(outcome StageOutcome) {
	e.Lock()
	defer e.Unlock()
	e.stageOutcomes = append(e.stageOutcomes, outcome)
}

type EmptyHookExecutor struct{}

func (executor *EmptyHookExecutor) SetAccount(_ *config.Account) {}

func (executor *EmptyHookExecutor) GetOutcomes() []StageOutcome {
	return []StageOutcome{}
}

func (executor *EmptyHookExecutor) ExecuteEntrypointStage(_ *http.Request, body []byte) ([]byte, *RejectError) {
	return body, nil
}

func (executor *EmptyHookExecutor) ExecuteRawAuctionStage(body []byte) ([]byte, *RejectError) {
	return body, nil
}

func (executor *EmptyHookExecutor) ExecuteBidderRequestStage(_ *openrtb2.BidRequest, bidder string) *RejectError {
	return nil
}

func (executor *EmptyHookExecutor) ExecuteRawBidderResponseStage(_ *adapters.BidderResponse, _ string) *RejectError {
	return nil
}
