package hookexecution

import (
	"context"
	"errors"
	"time"

	"github.com/prebid/prebid-server/hooks/hookstage"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type mockUpdateHeaderEntrypointHook struct{}

func (e mockUpdateHeaderEntrypointHook) HandleEntrypointHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.EntrypointPayload) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	c := &hookstage.ChangeSet[hookstage.EntrypointPayload]{}
	c.AddMutation(func(payload hookstage.EntrypointPayload) (hookstage.EntrypointPayload, error) {
		payload.Request.Header.Add("foo", "bar")
		return payload, nil
	}, hookstage.MutationUpdate, "header", "foo")

	return hookstage.HookResult[hookstage.EntrypointPayload]{ChangeSet: c}, nil
}

type mockUpdateQueryEntrypointHook struct{}

func (e mockUpdateQueryEntrypointHook) HandleEntrypointHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.EntrypointPayload) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	c := &hookstage.ChangeSet[hookstage.EntrypointPayload]{}
	c.AddMutation(func(payload hookstage.EntrypointPayload) (hookstage.EntrypointPayload, error) {
		params := payload.Request.URL.Query()
		params.Add("foo", "baz")
		payload.Request.URL.RawQuery = params.Encode()
		return payload, nil
	}, hookstage.MutationUpdate, "param", "foo")

	return hookstage.HookResult[hookstage.EntrypointPayload]{ChangeSet: c}, nil
}

type mockUpdateBodyHook struct{}

func (e mockUpdateBodyHook) HandleEntrypointHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.EntrypointPayload) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	c := &hookstage.ChangeSet[hookstage.EntrypointPayload]{}
	c.AddMutation(
		func(payload hookstage.EntrypointPayload) (hookstage.EntrypointPayload, error) {
			payload.Body = []byte(`{"name": "John", "last_name": "Doe", "foo": "bar"}`)
			return payload, nil
		}, hookstage.MutationUpdate, "body", "foo",
	).AddMutation(
		func(payload hookstage.EntrypointPayload) (hookstage.EntrypointPayload, error) {
			payload.Body = []byte(`{"last_name": "Doe", "foo": "bar"}`)
			return payload, nil
		}, hookstage.MutationDelete, "body", "name",
	)

	return hookstage.HookResult[hookstage.EntrypointPayload]{ChangeSet: c}, nil
}

func (e mockUpdateBodyHook) HandleRawAuctionHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.RawAuctionRequestPayload) (hookstage.HookResult[hookstage.RawAuctionRequestPayload], error) {
	c := &hookstage.ChangeSet[hookstage.RawAuctionRequestPayload]{}
	c.AddMutation(
		func(payload hookstage.RawAuctionRequestPayload) (hookstage.RawAuctionRequestPayload, error) {
			payload = []byte(`{"name": "John", "last_name": "Doe", "foo": "bar"}`)
			return payload, nil
		}, hookstage.MutationUpdate, "body", "foo",
	).AddMutation(
		func(payload hookstage.RawAuctionRequestPayload) (hookstage.RawAuctionRequestPayload, error) {
			payload = []byte(`{"last_name": "Doe", "foo": "bar"}`)
			return payload, nil
		}, hookstage.MutationDelete, "body", "name",
	)

	return hookstage.HookResult[hookstage.RawAuctionRequestPayload]{ChangeSet: c}, nil
}

type mockRejectHook struct{}

func (e mockRejectHook) HandleEntrypointHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.EntrypointPayload) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	return hookstage.HookResult[hookstage.EntrypointPayload]{Reject: true}, nil
}

func (e mockRejectHook) HandleRawAuctionHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.RawAuctionRequestPayload) (hookstage.HookResult[hookstage.RawAuctionRequestPayload], error) {
	return hookstage.HookResult[hookstage.RawAuctionRequestPayload]{Reject: true}, nil
}

func (e mockRejectHook) HandleAllProcessedBidResponsesHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.AllProcessedBidResponsesPayload) (hookstage.HookResult[hookstage.AllProcessedBidResponsesPayload], error) {
	return hookstage.HookResult[hookstage.AllProcessedBidResponsesPayload]{Reject: true}, nil
}

type mockTimeoutHook struct{}

func (e mockTimeoutHook) HandleEntrypointHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.EntrypointPayload) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	time.Sleep(3 * time.Millisecond)
	c := &hookstage.ChangeSet[hookstage.EntrypointPayload]{}
	c.AddMutation(func(payload hookstage.EntrypointPayload) (hookstage.EntrypointPayload, error) {
		params := payload.Request.URL.Query()
		params.Add("bar", "foo")
		payload.Request.URL.RawQuery = params.Encode()
		return payload, nil
	}, hookstage.MutationUpdate, "param", "bar")

	return hookstage.HookResult[hookstage.EntrypointPayload]{ChangeSet: c}, nil
}

func (e mockTimeoutHook) HandleRawAuctionHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.RawAuctionRequestPayload) (hookstage.HookResult[hookstage.RawAuctionRequestPayload], error) {
	time.Sleep(2 * time.Millisecond)
	c := &hookstage.ChangeSet[hookstage.RawAuctionRequestPayload]{}
	c.AddMutation(func(payload hookstage.RawAuctionRequestPayload) (hookstage.RawAuctionRequestPayload, error) {
		payload = []byte(`{"last_name": "Doe", "foo": "bar", "address": "A st."}`)
		return payload, nil
	}, hookstage.MutationUpdate, "param", "address")

	return hookstage.HookResult[hookstage.RawAuctionRequestPayload]{ChangeSet: c}, nil
}

func (e mockTimeoutHook) HandleAllProcessedBidResponsesHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.AllProcessedBidResponsesPayload) (hookstage.HookResult[hookstage.AllProcessedBidResponsesPayload], error) {
	time.Sleep(2 * time.Millisecond)
	c := &hookstage.ChangeSet[hookstage.AllProcessedBidResponsesPayload]{}
	c.AddMutation(func(payload hookstage.AllProcessedBidResponsesPayload) (hookstage.AllProcessedBidResponsesPayload, error) {
		payload.Responses["some-bidder"].Bids[0].BidMeta = &openrtb_ext.ExtBidPrebidMeta{AdapterCode: "new-code"}
		return payload, nil
	}, hookstage.MutationUpdate, "processedBidderResponse", "bidMeta.AdapterCode")

	return hookstage.HookResult[hookstage.AllProcessedBidResponsesPayload]{ChangeSet: c}, nil
}

type mockModuleContextHook struct {
	key, val string
}

func (e mockModuleContextHook) HandleEntrypointHook(_ context.Context, miCtx hookstage.ModuleInvocationContext, _ hookstage.EntrypointPayload) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	miCtx.ModuleContext = map[string]interface{}{e.key: e.val}
	return hookstage.HookResult[hookstage.EntrypointPayload]{ModuleContext: miCtx.ModuleContext}, nil
}

func (e mockModuleContextHook) HandleRawAuctionHook(_ context.Context, miCtx hookstage.ModuleInvocationContext, _ hookstage.RawAuctionRequestPayload) (hookstage.HookResult[hookstage.RawAuctionRequestPayload], error) {
	miCtx.ModuleContext = map[string]interface{}{e.key: e.val}
	return hookstage.HookResult[hookstage.RawAuctionRequestPayload]{ModuleContext: miCtx.ModuleContext}, nil
}

func (e mockModuleContextHook) HandleAllProcessedBidResponsesHook(_ context.Context, miCtx hookstage.ModuleInvocationContext, _ hookstage.AllProcessedBidResponsesPayload) (hookstage.HookResult[hookstage.AllProcessedBidResponsesPayload], error) {
	miCtx.ModuleContext = map[string]interface{}{e.key: e.val}
	return hookstage.HookResult[hookstage.AllProcessedBidResponsesPayload]{ModuleContext: miCtx.ModuleContext}, nil
}

type mockFailureHook struct{}

func (h mockFailureHook) HandleEntrypointHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.EntrypointPayload) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	return hookstage.HookResult[hookstage.EntrypointPayload]{}, FailureError{Message: "attribute not found"}
}

func (h mockFailureHook) HandleRawAuctionHook(_ context.Context, miCtx hookstage.ModuleInvocationContext, _ hookstage.RawAuctionRequestPayload) (hookstage.HookResult[hookstage.RawAuctionRequestPayload], error) {
	return hookstage.HookResult[hookstage.RawAuctionRequestPayload]{}, FailureError{Message: "attribute not found"}
}

func (e mockFailureHook) HandleAllProcessedBidResponsesHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.AllProcessedBidResponsesPayload) (hookstage.HookResult[hookstage.AllProcessedBidResponsesPayload], error) {
	return hookstage.HookResult[hookstage.AllProcessedBidResponsesPayload]{}, FailureError{Message: "attribute not found"}
}

type mockErrorHook struct{}

func (h mockErrorHook) HandleEntrypointHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.EntrypointPayload) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	return hookstage.HookResult[hookstage.EntrypointPayload]{}, errors.New("unexpected error")
}

func (h mockErrorHook) HandleRawAuctionHook(_ context.Context, miCtx hookstage.ModuleInvocationContext, _ hookstage.RawAuctionRequestPayload) (hookstage.HookResult[hookstage.RawAuctionRequestPayload], error) {
	return hookstage.HookResult[hookstage.RawAuctionRequestPayload]{}, errors.New("unexpected error")
}

func (e mockErrorHook) HandleAllProcessedBidResponsesHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.AllProcessedBidResponsesPayload) (hookstage.HookResult[hookstage.AllProcessedBidResponsesPayload], error) {
	return hookstage.HookResult[hookstage.AllProcessedBidResponsesPayload]{}, errors.New("unexpected error")
}

type mockFailedMutationHook struct{}

func (h mockFailedMutationHook) HandleEntrypointHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.EntrypointPayload) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	changeSet := &hookstage.ChangeSet[hookstage.EntrypointPayload]{}
	changeSet.AddMutation(func(payload hookstage.EntrypointPayload) (hookstage.EntrypointPayload, error) {
		return payload, errors.New("key not found")
	}, hookstage.MutationUpdate, "header", "foo")

	return hookstage.HookResult[hookstage.EntrypointPayload]{ChangeSet: changeSet}, nil
}

func (h mockFailedMutationHook) HandleRawAuctionHook(_ context.Context, miCtx hookstage.ModuleInvocationContext, _ hookstage.RawAuctionRequestPayload) (hookstage.HookResult[hookstage.RawAuctionRequestPayload], error) {
	changeSet := &hookstage.ChangeSet[hookstage.RawAuctionRequestPayload]{}
	changeSet.AddMutation(func(payload hookstage.RawAuctionRequestPayload) (hookstage.RawAuctionRequestPayload, error) {
		return payload, errors.New("key not found")
	}, hookstage.MutationUpdate, "header", "foo")

	return hookstage.HookResult[hookstage.RawAuctionRequestPayload]{ChangeSet: changeSet}, nil
}

func (e mockFailedMutationHook) HandleAllProcessedBidResponsesHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.AllProcessedBidResponsesPayload) (hookstage.HookResult[hookstage.AllProcessedBidResponsesPayload], error) {
	changeSet := &hookstage.ChangeSet[hookstage.AllProcessedBidResponsesPayload]{}
	changeSet.AddMutation(func(payload hookstage.AllProcessedBidResponsesPayload) (hookstage.AllProcessedBidResponsesPayload, error) {
		return payload, errors.New("key not found")
	}, hookstage.MutationUpdate, "some-bidder", "bids[0]", "deal_priority")

	return hookstage.HookResult[hookstage.AllProcessedBidResponsesPayload]{ChangeSet: changeSet}, nil
}

type mockUpdateBiddersResponsesHook struct{}

func (e mockUpdateBiddersResponsesHook) HandleAllProcessedBidResponsesHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.AllProcessedBidResponsesPayload) (hookstage.HookResult[hookstage.AllProcessedBidResponsesPayload], error) {
	c := &hookstage.ChangeSet[hookstage.AllProcessedBidResponsesPayload]{}
	c.AddMutation(
		func(payload hookstage.AllProcessedBidResponsesPayload) (hookstage.AllProcessedBidResponsesPayload, error) {
			payload.Responses["some-bidder"].Bids[0].DealPriority = 10
			return payload, nil
		}, hookstage.MutationUpdate, "processedBidderResponse", "bid.deal-priority",
	)

	return hookstage.HookResult[hookstage.AllProcessedBidResponsesPayload]{ChangeSet: c}, nil
}
