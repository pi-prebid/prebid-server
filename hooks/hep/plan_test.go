package hep

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/hooks/invocation"
	"github.com/prebid/prebid-server/hooks/stages"
	"github.com/stretchr/testify/assert"
)

func TestPlanForEntrypointStage(t *testing.T) {
	testCases := map[string]struct {
		givenEndpoint               string
		givenHostPlanData           []byte
		givenDefaultAccountPlanData []byte
		givenHooks                  map[string]map[string]interface{}
		expectedPlan                Plan[stages.EntrypointHook]
	}{
		"Host and default-account execution plans successfully merged": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{"endpoints":{"/openrtb2/auction":{"stages":{"entrypoint":{"groups":[{"timeout":5,"hook-sequence":[{"module-code":"foobar","hook-impl-code":"foo"}]}]}}}}}`),
			givenDefaultAccountPlanData: []byte(`{"endpoints": {"/openrtb2/auction": {"stages": {"entrypoint": {"groups": [{"timeout": 10, "hook-sequence": [{"module-code": "foobar", "hook-impl-code": "bar"}, {"module-code": "ortb2blocking", "hook-impl-code": "block_request"}]}, {"timeout": 5, "hook-sequence": [{"module-code": "foobar", "hook-impl-code": "foo"}]}]}}}, "/openrtb2/amp": {"stages": {"entrypoint": {"groups": [{"timeout": 5, "hook-sequence": [{"module-code": "foobar", "hook-impl-code": "foo"}]}]}}}}}`),
			givenHooks: map[string]map[string]interface{}{
				"foobar": {
					"foo": fakeEntrypointHook{},
					"bar": fakeEntrypointHook{},
				},
				"ortb2blocking": {
					"block_request": fakeEntrypointHook{},
				},
			},
			expectedPlan: Plan[stages.EntrypointHook]{
				// first group from host-level plan
				Group[stages.EntrypointHook]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[stages.EntrypointHook]{
						{Module: "foobar", Code: "foo", Hook: fakeEntrypointHook{}},
					},
				},
				// then groups from the account-level plan
				Group[stages.EntrypointHook]{
					Timeout: 10 * time.Millisecond,
					Hooks: []HookWrapper[stages.EntrypointHook]{
						{Module: "foobar", Code: "bar", Hook: fakeEntrypointHook{}},
						{Module: "ortb2blocking", Code: "block_request", Hook: fakeEntrypointHook{}},
					},
				},
				Group[stages.EntrypointHook]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[stages.EntrypointHook]{
						{Module: "foobar", Code: "foo", Hook: fakeEntrypointHook{}},
					},
				},
			},
		},
		"Works with empty default-account-execution-plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{"endpoints":{"/openrtb2/auction":{"stages":{"entrypoint":{"groups":[{"timeout":5,"hook-sequence":[{"module-code":"foobar","hook-impl-code":"foo"}]}]}}}}}`),
			givenDefaultAccountPlanData: []byte(`{}`),
			givenHooks:                  map[string]map[string]interface{}{"foobar": {"foo": fakeEntrypointHook{}}},
			expectedPlan: Plan[stages.EntrypointHook]{
				Group[stages.EntrypointHook]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[stages.EntrypointHook]{
						{Module: "foobar", Code: "foo", Hook: fakeEntrypointHook{}},
					},
				},
			},
		},
		"Works with empty host-execution-plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{}`),
			givenDefaultAccountPlanData: []byte(`{"endpoints":{"/openrtb2/auction":{"stages":{"entrypoint":{"groups":[{"timeout":5,"hook-sequence":[{"module-code":"foobar","hook-impl-code":"foo"}]}]}}}}}`),
			givenHooks:                  map[string]map[string]interface{}{"foobar": {"foo": fakeEntrypointHook{}}},
			expectedPlan: Plan[stages.EntrypointHook]{
				Group[stages.EntrypointHook]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[stages.EntrypointHook]{
						{Module: "foobar", Code: "foo", Hook: fakeEntrypointHook{}},
					},
				},
			},
		},
		"Empty plan if hooks config not defined": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{}`),
			givenDefaultAccountPlanData: []byte(`{}`),
			givenHooks:                  map[string]map[string]interface{}{"foobar": {"foo": fakeEntrypointHook{}}},
			expectedPlan:                Plan[stages.EntrypointHook]{},
		},
		"Empty plan if hook repository empty": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{"endpoints":{"/openrtb2/auction":{"stages":{"entrypoint":{"groups":[{"timeout":5,"hook-sequence":[{"module-code":"foobar","hook-impl-code":"foo"}]}]}}}}}`),
			givenDefaultAccountPlanData: []byte(`{}`),
			givenHooks:                  nil,
			expectedPlan:                Plan[stages.EntrypointHook]{},
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			planBuilder, err := getPlanBuilder(test.givenHooks, test.givenHostPlanData, test.givenDefaultAccountPlanData)
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectedPlan, planBuilder.PlanForEntrypointStage(test.givenEndpoint))
		})
	}
}

func TestPlanForRawAuctionStage(t *testing.T) {
	hooks := map[string]map[string]interface{}{
		"foobar": {
			"foo": fakeRawAuctionHook{},
			"bar": fakeRawAuctionHook{},
		},
		"ortb2blocking": {
			"block_request": fakeRawAuctionHook{},
		},
		"prebid": {
			"baz": fakeRawAuctionHook{},
		},
	}

	testCases := map[string]struct {
		givenEndpoint               string
		givenHostPlanData           []byte
		givenDefaultAccountPlanData []byte
		giveAccountPlanData         []byte
		givenHooks                  map[string]map[string]interface{}
		expectedPlan                Plan[stages.RawAuctionHook]
	}{
		"Account-specific execution plan rewrites default-account execution plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{"endpoints":{"/openrtb2/auction":{"stages":{"rawauction":{"groups":[{"timeout":5,"hook-sequence":[{"module-code":"foobar","hook-impl-code":"foo"}]}]}}}}}`),
			givenDefaultAccountPlanData: []byte(`{"endpoints": {"/openrtb2/auction": {"stages": {"rawauction": {"groups": [{"timeout": 10, "hook-sequence": [{"module-code": "foobar", "hook-impl-code": "bar"}, {"module-code": "ortb2blocking", "hook-impl-code": "block_request"}]}, {"timeout": 5, "hook-sequence": [{"module-code": "foobar", "hook-impl-code": "foo"}]}]}}}, "/openrtb2/amp": {"stages": {"entrypoint": {"groups": [{"timeout": 5, "hook-sequence": [{"module-code": "foobar", "hook-impl-code": "foo"}]}]}}}}}`),
			giveAccountPlanData:         []byte(`{"endpoints": {"/openrtb2/auction": {"stages": {"rawauction": {"groups": [{"timeout": 15, "hook-sequence": [{"module-code": "prebid", "hook-impl-code": "baz"}]}]}}}}}`),
			givenHooks:                  hooks,
			expectedPlan: Plan[stages.RawAuctionHook]{
				// first group from host-level plan
				Group[stages.RawAuctionHook]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[stages.RawAuctionHook]{
						{Module: "foobar", Code: "foo", Hook: fakeRawAuctionHook{}},
					},
				},
				// then come groups from account-level plan (default-account-level plan ignored)
				Group[stages.RawAuctionHook]{
					Timeout: 15 * time.Millisecond,
					Hooks: []HookWrapper[stages.RawAuctionHook]{
						{Module: "prebid", Code: "baz", Hook: fakeRawAuctionHook{}},
					},
				},
			},
		},
		"Works with only account-specific plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{}`),
			givenDefaultAccountPlanData: []byte(`{}`),
			giveAccountPlanData:         []byte(`{"endpoints": {"/openrtb2/auction": {"stages": {"rawauction": {"groups": [{"timeout": 15, "hook-sequence": [{"module-code": "prebid", "hook-impl-code": "baz"}]}]}}}}}`),
			givenHooks:                  hooks,
			expectedPlan: Plan[stages.RawAuctionHook]{
				Group[stages.RawAuctionHook]{
					Timeout: 15 * time.Millisecond,
					Hooks: []HookWrapper[stages.RawAuctionHook]{
						{Module: "prebid", Code: "baz", Hook: fakeRawAuctionHook{}},
					},
				},
			},
		},
		"Works with empty account-specific execution plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{"endpoints":{"/openrtb2/auction":{"stages":{"rawauction":{"groups":[{"timeout":5,"hook-sequence":[{"module-code":"foobar","hook-impl-code":"foo"}]}]}}}}}`),
			givenDefaultAccountPlanData: []byte(`{"endpoints": {"/openrtb2/auction": {"stages": {"rawauction": {"groups": [{"timeout": 10, "hook-sequence": [{"module-code": "foobar", "hook-impl-code": "bar"}, {"module-code": "ortb2blocking", "hook-impl-code": "block_request"}]}, {"timeout": 5, "hook-sequence": [{"module-code": "foobar", "hook-impl-code": "foo"}]}]}}}, "/openrtb2/amp": {"stages": {"entrypoint": {"groups": [{"timeout": 5, "hook-sequence": [{"module-code": "foobar", "hook-impl-code": "foo"}]}]}}}}}`),
			giveAccountPlanData:         []byte(`{}`),
			givenHooks:                  hooks,
			expectedPlan: Plan[stages.RawAuctionHook]{
				Group[stages.RawAuctionHook]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[stages.RawAuctionHook]{
						{Module: "foobar", Code: "foo", Hook: fakeRawAuctionHook{}},
					},
				},
				Group[stages.RawAuctionHook]{
					Timeout: 10 * time.Millisecond,
					Hooks: []HookWrapper[stages.RawAuctionHook]{
						{Module: "foobar", Code: "bar", Hook: fakeRawAuctionHook{}},
						{Module: "ortb2blocking", Code: "block_request", Hook: fakeRawAuctionHook{}},
					},
				},
				Group[stages.RawAuctionHook]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[stages.RawAuctionHook]{
						{Module: "foobar", Code: "foo", Hook: fakeRawAuctionHook{}},
					},
				},
			},
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			account := new(config.Account)
			err := json.Unmarshal(test.giveAccountPlanData, &account.Hooks)
			if err != nil {
				t.Fatal(err)
			}

			planBuilder, err := getPlanBuilder(test.givenHooks, test.givenHostPlanData, test.givenDefaultAccountPlanData)
			if err != nil {
				t.Fatal(err)
			}

			plan := planBuilder.PlanForRawAuctionStage(test.givenEndpoint, account)
			assert.Equal(t, test.expectedPlan, plan)
		})
	}
}

func TestPlanForProcessedAuctionStage(t *testing.T) {
	hooks := map[string]map[string]interface{}{
		"foobar": {
			"foo": fakeProcessedAuctionHook{},
			"bar": fakeProcessedAuctionHook{},
		},
		"ortb2blocking": {
			"block_request": fakeProcessedAuctionHook{},
		},
		"prebid": {
			"baz": fakeProcessedAuctionHook{},
		},
	}

	testCases := map[string]struct {
		givenEndpoint               string
		givenHostPlanData           []byte
		givenDefaultAccountPlanData []byte
		giveAccountPlanData         []byte
		givenHooks                  map[string]map[string]interface{}
		expectedPlan                Plan[stages.ProcessedAuctionHook]
	}{
		"Account-specific execution plan rewrites default-account execution plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{"endpoints":{"/openrtb2/auction":{"stages":{"procauction":{"groups":[{"timeout":5,"hook-sequence":[{"module-code":"foobar","hook-impl-code":"foo"}]}]}}}}}`),
			givenDefaultAccountPlanData: []byte(`{"endpoints": {"/openrtb2/auction": {"stages": {"procauction": {"groups": [{"timeout": 10, "hook-sequence": [{"module-code": "foobar", "hook-impl-code": "bar"}, {"module-code": "ortb2blocking", "hook-impl-code": "block_request"}]}, {"timeout": 5, "hook-sequence": [{"module-code": "foobar", "hook-impl-code": "foo"}]}]}}}, "/openrtb2/amp": {"stages": {"entrypoint": {"groups": [{"timeout": 5, "hook-sequence": [{"module-code": "foobar", "hook-impl-code": "foo"}]}]}}}}}`),
			giveAccountPlanData:         []byte(`{"endpoints": {"/openrtb2/auction": {"stages": {"procauction": {"groups": [{"timeout": 15, "hook-sequence": [{"module-code": "prebid", "hook-impl-code": "baz"}]}]}}}}}`),
			givenHooks:                  hooks,
			expectedPlan: Plan[stages.ProcessedAuctionHook]{
				// first group from host-level plan
				Group[stages.ProcessedAuctionHook]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[stages.ProcessedAuctionHook]{
						{Module: "foobar", Code: "foo", Hook: fakeProcessedAuctionHook{}},
					},
				},
				// then come groups from account-level plan (default-account-level plan ignored)
				Group[stages.ProcessedAuctionHook]{
					Timeout: 15 * time.Millisecond,
					Hooks: []HookWrapper[stages.ProcessedAuctionHook]{
						{Module: "prebid", Code: "baz", Hook: fakeProcessedAuctionHook{}},
					},
				},
			},
		},
		"Works with only account-specific plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{}`),
			givenDefaultAccountPlanData: []byte(`{}`),
			giveAccountPlanData:         []byte(`{"endpoints": {"/openrtb2/auction": {"stages": {"procauction": {"groups": [{"timeout": 15, "hook-sequence": [{"module-code": "prebid", "hook-impl-code": "baz"}]}]}}}}}`),
			givenHooks:                  hooks,
			expectedPlan: Plan[stages.ProcessedAuctionHook]{
				Group[stages.ProcessedAuctionHook]{
					Timeout: 15 * time.Millisecond,
					Hooks: []HookWrapper[stages.ProcessedAuctionHook]{
						{Module: "prebid", Code: "baz", Hook: fakeProcessedAuctionHook{}},
					},
				},
			},
		},
		"Works with empty account-specific execution plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{"endpoints":{"/openrtb2/auction":{"stages":{"procauction":{"groups":[{"timeout":5,"hook-sequence":[{"module-code":"foobar","hook-impl-code":"foo"}]}]}}}}}`),
			givenDefaultAccountPlanData: []byte(`{"endpoints": {"/openrtb2/auction": {"stages": {"procauction": {"groups": [{"timeout": 10, "hook-sequence": [{"module-code": "foobar", "hook-impl-code": "bar"}, {"module-code": "ortb2blocking", "hook-impl-code": "block_request"}]}, {"timeout": 5, "hook-sequence": [{"module-code": "foobar", "hook-impl-code": "foo"}]}]}}}, "/openrtb2/amp": {"stages": {"entrypoint": {"groups": [{"timeout": 5, "hook-sequence": [{"module-code": "foobar", "hook-impl-code": "foo"}]}]}}}}}`),
			giveAccountPlanData:         []byte(`{}`),
			givenHooks:                  hooks,
			expectedPlan: Plan[stages.ProcessedAuctionHook]{
				Group[stages.ProcessedAuctionHook]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[stages.ProcessedAuctionHook]{
						{Module: "foobar", Code: "foo", Hook: fakeProcessedAuctionHook{}},
					},
				},
				Group[stages.ProcessedAuctionHook]{
					Timeout: 10 * time.Millisecond,
					Hooks: []HookWrapper[stages.ProcessedAuctionHook]{
						{Module: "foobar", Code: "bar", Hook: fakeProcessedAuctionHook{}},
						{Module: "ortb2blocking", Code: "block_request", Hook: fakeProcessedAuctionHook{}},
					},
				},
				Group[stages.ProcessedAuctionHook]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[stages.ProcessedAuctionHook]{
						{Module: "foobar", Code: "foo", Hook: fakeProcessedAuctionHook{}},
					},
				},
			},
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			account := new(config.Account)
			err := json.Unmarshal(test.giveAccountPlanData, &account.Hooks)
			if err != nil {
				t.Fatal(err)
			}

			planBuilder, err := getPlanBuilder(test.givenHooks, test.givenHostPlanData, test.givenDefaultAccountPlanData)
			if err != nil {
				t.Fatal(err)
			}

			plan := planBuilder.PlanForProcessedAuctionStage(test.givenEndpoint, account)
			assert.Equal(t, test.expectedPlan, plan)
		})
	}
}

func TestPlanForBidRequestStage(t *testing.T) {
	hooks := map[string]map[string]interface{}{
		"foobar": {
			"foo": fakeBidRequestHook{},
			"bar": fakeBidRequestHook{},
		},
		"ortb2blocking": {
			"block_request": fakeBidRequestHook{},
		},
		"prebid": {
			"baz": fakeBidRequestHook{},
		},
	}

	testCases := map[string]struct {
		givenEndpoint               string
		givenHostPlanData           []byte
		givenDefaultAccountPlanData []byte
		giveAccountPlanData         []byte
		givenHooks                  map[string]map[string]interface{}
		expectedPlan                Plan[stages.BidRequestHook]
	}{
		"Account-specific execution plan rewrites default-account execution plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{"endpoints":{"/openrtb2/auction":{"stages":{"bidrequest":{"groups":[{"timeout":5,"hook-sequence":[{"module-code":"foobar","hook-impl-code":"foo"}]}]}}}}}`),
			givenDefaultAccountPlanData: []byte(`{"endpoints": {"/openrtb2/auction": {"stages": {"bidrequest": {"groups": [{"timeout": 10, "hook-sequence": [{"module-code": "foobar", "hook-impl-code": "bar"}, {"module-code": "ortb2blocking", "hook-impl-code": "block_request"}]}, {"timeout": 5, "hook-sequence": [{"module-code": "foobar", "hook-impl-code": "foo"}]}]}}}, "/openrtb2/amp": {"stages": {"entrypoint": {"groups": [{"timeout": 5, "hook-sequence": [{"module-code": "foobar", "hook-impl-code": "foo"}]}]}}}}}`),
			giveAccountPlanData:         []byte(`{"endpoints": {"/openrtb2/auction": {"stages": {"bidrequest": {"groups": [{"timeout": 15, "hook-sequence": [{"module-code": "prebid", "hook-impl-code": "baz"}]}]}}}}}`),
			givenHooks:                  hooks,
			expectedPlan: Plan[stages.BidRequestHook]{
				// first group from host-level plan
				Group[stages.BidRequestHook]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[stages.BidRequestHook]{
						{Module: "foobar", Code: "foo", Hook: fakeBidRequestHook{}},
					},
				},
				// then come groups from account-level plan (default-account-level plan ignored)
				Group[stages.BidRequestHook]{
					Timeout: 15 * time.Millisecond,
					Hooks: []HookWrapper[stages.BidRequestHook]{
						{Module: "prebid", Code: "baz", Hook: fakeBidRequestHook{}},
					},
				},
			},
		},
		"Works with only account-specific plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{}`),
			givenDefaultAccountPlanData: []byte(`{}`),
			giveAccountPlanData:         []byte(`{"endpoints": {"/openrtb2/auction": {"stages": {"bidrequest": {"groups": [{"timeout": 15, "hook-sequence": [{"module-code": "prebid", "hook-impl-code": "baz"}]}]}}}}}`),
			givenHooks:                  hooks,
			expectedPlan: Plan[stages.BidRequestHook]{
				Group[stages.BidRequestHook]{
					Timeout: 15 * time.Millisecond,
					Hooks: []HookWrapper[stages.BidRequestHook]{
						{Module: "prebid", Code: "baz", Hook: fakeBidRequestHook{}},
					},
				},
			},
		},
		"Works with empty account-specific execution plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{"endpoints":{"/openrtb2/auction":{"stages":{"bidrequest":{"groups":[{"timeout":5,"hook-sequence":[{"module-code":"foobar","hook-impl-code":"foo"}]}]}}}}}`),
			givenDefaultAccountPlanData: []byte(`{"endpoints": {"/openrtb2/auction": {"stages": {"bidrequest": {"groups": [{"timeout": 10, "hook-sequence": [{"module-code": "foobar", "hook-impl-code": "bar"}, {"module-code": "ortb2blocking", "hook-impl-code": "block_request"}]}, {"timeout": 5, "hook-sequence": [{"module-code": "foobar", "hook-impl-code": "foo"}]}]}}}, "/openrtb2/amp": {"stages": {"entrypoint": {"groups": [{"timeout": 5, "hook-sequence": [{"module-code": "foobar", "hook-impl-code": "foo"}]}]}}}}}`),
			giveAccountPlanData:         []byte(`{}`),
			givenHooks:                  hooks,
			expectedPlan: Plan[stages.BidRequestHook]{
				Group[stages.BidRequestHook]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[stages.BidRequestHook]{
						{Module: "foobar", Code: "foo", Hook: fakeBidRequestHook{}},
					},
				},
				Group[stages.BidRequestHook]{
					Timeout: 10 * time.Millisecond,
					Hooks: []HookWrapper[stages.BidRequestHook]{
						{Module: "foobar", Code: "bar", Hook: fakeBidRequestHook{}},
						{Module: "ortb2blocking", Code: "block_request", Hook: fakeBidRequestHook{}},
					},
				},
				Group[stages.BidRequestHook]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[stages.BidRequestHook]{
						{Module: "foobar", Code: "foo", Hook: fakeBidRequestHook{}},
					},
				},
			},
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			account := new(config.Account)
			err := json.Unmarshal(test.giveAccountPlanData, &account.Hooks)
			if err != nil {
				t.Fatal(err)
			}

			planBuilder, err := getPlanBuilder(test.givenHooks, test.givenHostPlanData, test.givenDefaultAccountPlanData)
			if err != nil {
				t.Fatal(err)
			}

			plan := planBuilder.PlanForBidRequestStage(test.givenEndpoint, account)
			assert.Equal(t, test.expectedPlan, plan)
		})
	}
}

func TestPlanForRawBidResponseStage(t *testing.T) {
	hooks := map[string]map[string]interface{}{
		"foobar": {
			"foo": fakeRawBidResponseHook{},
			"bar": fakeRawBidResponseHook{},
		},
		"ortb2blocking": {
			"block_request": fakeRawBidResponseHook{},
		},
		"prebid": {
			"baz": fakeRawBidResponseHook{},
		},
	}

	testCases := map[string]struct {
		givenEndpoint               string
		givenHostPlanData           []byte
		givenDefaultAccountPlanData []byte
		giveAccountPlanData         []byte
		givenHooks                  map[string]map[string]interface{}
		expectedPlan                Plan[stages.RawBidResponseHook]
	}{
		"Account-specific execution plan rewrites default-account execution plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{"endpoints":{"/openrtb2/auction":{"stages":{"rawbidresponse":{"groups":[{"timeout":5,"hook-sequence":[{"module-code":"foobar","hook-impl-code":"foo"}]}]}}}}}`),
			givenDefaultAccountPlanData: []byte(`{"endpoints": {"/openrtb2/auction": {"stages": {"rawbidresponse": {"groups": [{"timeout": 10, "hook-sequence": [{"module-code": "foobar", "hook-impl-code": "bar"}, {"module-code": "ortb2blocking", "hook-impl-code": "block_request"}]}, {"timeout": 5, "hook-sequence": [{"module-code": "foobar", "hook-impl-code": "foo"}]}]}}}, "/openrtb2/amp": {"stages": {"entrypoint": {"groups": [{"timeout": 5, "hook-sequence": [{"module-code": "foobar", "hook-impl-code": "foo"}]}]}}}}}`),
			giveAccountPlanData:         []byte(`{"endpoints": {"/openrtb2/auction": {"stages": {"rawbidresponse": {"groups": [{"timeout": 15, "hook-sequence": [{"module-code": "prebid", "hook-impl-code": "baz"}]}]}}}}}`),
			givenHooks:                  hooks,
			expectedPlan: Plan[stages.RawBidResponseHook]{
				// first group from host-level plan
				Group[stages.RawBidResponseHook]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[stages.RawBidResponseHook]{
						{Module: "foobar", Code: "foo", Hook: fakeRawBidResponseHook{}},
					},
				},
				// then come groups from account-level plan (default-account-level plan ignored)
				Group[stages.RawBidResponseHook]{
					Timeout: 15 * time.Millisecond,
					Hooks: []HookWrapper[stages.RawBidResponseHook]{
						{Module: "prebid", Code: "baz", Hook: fakeRawBidResponseHook{}},
					},
				},
			},
		},
		"Works with only account-specific plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{}`),
			givenDefaultAccountPlanData: []byte(`{}`),
			giveAccountPlanData:         []byte(`{"endpoints": {"/openrtb2/auction": {"stages": {"rawbidresponse": {"groups": [{"timeout": 15, "hook-sequence": [{"module-code": "prebid", "hook-impl-code": "baz"}]}]}}}}}`),
			givenHooks:                  hooks,
			expectedPlan: Plan[stages.RawBidResponseHook]{
				Group[stages.RawBidResponseHook]{
					Timeout: 15 * time.Millisecond,
					Hooks: []HookWrapper[stages.RawBidResponseHook]{
						{Module: "prebid", Code: "baz", Hook: fakeRawBidResponseHook{}},
					},
				},
			},
		},
		"Works with empty account-specific execution plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{"endpoints":{"/openrtb2/auction":{"stages":{"rawbidresponse":{"groups":[{"timeout":5,"hook-sequence":[{"module-code":"foobar","hook-impl-code":"foo"}]}]}}}}}`),
			givenDefaultAccountPlanData: []byte(`{"endpoints": {"/openrtb2/auction": {"stages": {"rawbidresponse": {"groups": [{"timeout": 10, "hook-sequence": [{"module-code": "foobar", "hook-impl-code": "bar"}, {"module-code": "ortb2blocking", "hook-impl-code": "block_request"}]}, {"timeout": 5, "hook-sequence": [{"module-code": "foobar", "hook-impl-code": "foo"}]}]}}}, "/openrtb2/amp": {"stages": {"entrypoint": {"groups": [{"timeout": 5, "hook-sequence": [{"module-code": "foobar", "hook-impl-code": "foo"}]}]}}}}}`),
			giveAccountPlanData:         []byte(`{}`),
			givenHooks:                  hooks,
			expectedPlan: Plan[stages.RawBidResponseHook]{
				Group[stages.RawBidResponseHook]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[stages.RawBidResponseHook]{
						{Module: "foobar", Code: "foo", Hook: fakeRawBidResponseHook{}},
					},
				},
				Group[stages.RawBidResponseHook]{
					Timeout: 10 * time.Millisecond,
					Hooks: []HookWrapper[stages.RawBidResponseHook]{
						{Module: "foobar", Code: "bar", Hook: fakeRawBidResponseHook{}},
						{Module: "ortb2blocking", Code: "block_request", Hook: fakeRawBidResponseHook{}},
					},
				},
				Group[stages.RawBidResponseHook]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[stages.RawBidResponseHook]{
						{Module: "foobar", Code: "foo", Hook: fakeRawBidResponseHook{}},
					},
				},
			},
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			account := new(config.Account)
			err := json.Unmarshal(test.giveAccountPlanData, &account.Hooks)
			if err != nil {
				t.Fatal(err)
			}

			planBuilder, err := getPlanBuilder(test.givenHooks, test.givenHostPlanData, test.givenDefaultAccountPlanData)
			if err != nil {
				t.Fatal(err)
			}

			plan := planBuilder.PlanForRawBidResponseStage(test.givenEndpoint, account)
			assert.Equal(t, test.expectedPlan, plan)
		})
	}
}

func TestPlanForProcessedBidResponseStage(t *testing.T) {
	hooks := map[string]map[string]interface{}{
		"foobar": {
			"foo": fakeProcessedBidResponseHook{},
			"bar": fakeProcessedBidResponseHook{},
		},
		"ortb2blocking": {
			"block_request": fakeProcessedBidResponseHook{},
		},
		"prebid": {
			"baz": fakeProcessedBidResponseHook{},
		},
	}

	testCases := map[string]struct {
		givenEndpoint               string
		givenHostPlanData           []byte
		givenDefaultAccountPlanData []byte
		giveAccountPlanData         []byte
		givenHooks                  map[string]map[string]interface{}
		expectedPlan                Plan[stages.ProcessedBidResponseHook]
	}{
		"Account-specific execution plan rewrites default-account execution plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{"endpoints":{"/openrtb2/auction":{"stages":{"procbidresponse":{"groups":[{"timeout":5,"hook-sequence":[{"module-code":"foobar","hook-impl-code":"foo"}]}]}}}}}`),
			givenDefaultAccountPlanData: []byte(`{"endpoints": {"/openrtb2/auction": {"stages": {"procbidresponse": {"groups": [{"timeout": 10, "hook-sequence": [{"module-code": "foobar", "hook-impl-code": "bar"}, {"module-code": "ortb2blocking", "hook-impl-code": "block_request"}]}, {"timeout": 5, "hook-sequence": [{"module-code": "foobar", "hook-impl-code": "foo"}]}]}}}, "/openrtb2/amp": {"stages": {"entrypoint": {"groups": [{"timeout": 5, "hook-sequence": [{"module-code": "foobar", "hook-impl-code": "foo"}]}]}}}}}`),
			giveAccountPlanData:         []byte(`{"endpoints": {"/openrtb2/auction": {"stages": {"procbidresponse": {"groups": [{"timeout": 15, "hook-sequence": [{"module-code": "prebid", "hook-impl-code": "baz"}]}]}}}}}`),
			givenHooks:                  hooks,
			expectedPlan: Plan[stages.ProcessedBidResponseHook]{
				// first group from host-level plan
				Group[stages.ProcessedBidResponseHook]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[stages.ProcessedBidResponseHook]{
						{Module: "foobar", Code: "foo", Hook: fakeProcessedBidResponseHook{}},
					},
				},
				// then come groups from account-level plan (default-account-level plan ignored)
				Group[stages.ProcessedBidResponseHook]{
					Timeout: 15 * time.Millisecond,
					Hooks: []HookWrapper[stages.ProcessedBidResponseHook]{
						{Module: "prebid", Code: "baz", Hook: fakeProcessedBidResponseHook{}},
					},
				},
			},
		},
		"Works with only account-specific plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{}`),
			givenDefaultAccountPlanData: []byte(`{}`),
			giveAccountPlanData:         []byte(`{"endpoints": {"/openrtb2/auction": {"stages": {"procbidresponse": {"groups": [{"timeout": 15, "hook-sequence": [{"module-code": "prebid", "hook-impl-code": "baz"}]}]}}}}}`),
			givenHooks:                  hooks,
			expectedPlan: Plan[stages.ProcessedBidResponseHook]{
				Group[stages.ProcessedBidResponseHook]{
					Timeout: 15 * time.Millisecond,
					Hooks: []HookWrapper[stages.ProcessedBidResponseHook]{
						{Module: "prebid", Code: "baz", Hook: fakeProcessedBidResponseHook{}},
					},
				},
			},
		},
		"Works with empty account-specific execution plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{"endpoints":{"/openrtb2/auction":{"stages":{"procbidresponse":{"groups":[{"timeout":5,"hook-sequence":[{"module-code":"foobar","hook-impl-code":"foo"}]}]}}}}}`),
			givenDefaultAccountPlanData: []byte(`{"endpoints": {"/openrtb2/auction": {"stages": {"procbidresponse": {"groups": [{"timeout": 10, "hook-sequence": [{"module-code": "foobar", "hook-impl-code": "bar"}, {"module-code": "ortb2blocking", "hook-impl-code": "block_request"}]}, {"timeout": 5, "hook-sequence": [{"module-code": "foobar", "hook-impl-code": "foo"}]}]}}}, "/openrtb2/amp": {"stages": {"entrypoint": {"groups": [{"timeout": 5, "hook-sequence": [{"module-code": "foobar", "hook-impl-code": "foo"}]}]}}}}}`),
			giveAccountPlanData:         []byte(`{}`),
			givenHooks:                  hooks,
			expectedPlan: Plan[stages.ProcessedBidResponseHook]{
				Group[stages.ProcessedBidResponseHook]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[stages.ProcessedBidResponseHook]{
						{Module: "foobar", Code: "foo", Hook: fakeProcessedBidResponseHook{}},
					},
				},
				Group[stages.ProcessedBidResponseHook]{
					Timeout: 10 * time.Millisecond,
					Hooks: []HookWrapper[stages.ProcessedBidResponseHook]{
						{Module: "foobar", Code: "bar", Hook: fakeProcessedBidResponseHook{}},
						{Module: "ortb2blocking", Code: "block_request", Hook: fakeProcessedBidResponseHook{}},
					},
				},
				Group[stages.ProcessedBidResponseHook]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[stages.ProcessedBidResponseHook]{
						{Module: "foobar", Code: "foo", Hook: fakeProcessedBidResponseHook{}},
					},
				},
			},
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			account := new(config.Account)
			err := json.Unmarshal(test.giveAccountPlanData, &account.Hooks)
			if err != nil {
				t.Fatal(err)
			}

			planBuilder, err := getPlanBuilder(test.givenHooks, test.givenHostPlanData, test.givenDefaultAccountPlanData)
			if err != nil {
				t.Fatal(err)
			}

			plan := planBuilder.PlanForProcessedBidResponseStage(test.givenEndpoint, account)
			assert.Equal(t, test.expectedPlan, plan)
		})
	}
}

func TestPlanForAllProcBidResponsesStage(t *testing.T) {
	hooks := map[string]map[string]interface{}{
		"foobar": {
			"foo": fakeAllProcBidResponsesHook{},
			"bar": fakeAllProcBidResponsesHook{},
		},
		"ortb2blocking": {
			"block_request": fakeAllProcBidResponsesHook{},
		},
		"prebid": {
			"baz": fakeAllProcBidResponsesHook{},
		},
	}

	testCases := map[string]struct {
		givenEndpoint               string
		givenHostPlanData           []byte
		givenDefaultAccountPlanData []byte
		giveAccountPlanData         []byte
		givenHooks                  map[string]map[string]interface{}
		expectedPlan                Plan[stages.AllProcBidResponsesHook]
	}{
		"Account-specific execution plan rewrites default-account execution plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{"endpoints":{"/openrtb2/auction":{"stages":{"allprocbidresponses":{"groups":[{"timeout":5,"hook-sequence":[{"module-code":"foobar","hook-impl-code":"foo"}]}]}}}}}`),
			givenDefaultAccountPlanData: []byte(`{"endpoints": {"/openrtb2/auction": {"stages": {"allprocbidresponses": {"groups": [{"timeout": 10, "hook-sequence": [{"module-code": "foobar", "hook-impl-code": "bar"}, {"module-code": "ortb2blocking", "hook-impl-code": "block_request"}]}, {"timeout": 5, "hook-sequence": [{"module-code": "foobar", "hook-impl-code": "foo"}]}]}}}, "/openrtb2/amp": {"stages": {"entrypoint": {"groups": [{"timeout": 5, "hook-sequence": [{"module-code": "foobar", "hook-impl-code": "foo"}]}]}}}}}`),
			giveAccountPlanData:         []byte(`{"endpoints": {"/openrtb2/auction": {"stages": {"allprocbidresponses": {"groups": [{"timeout": 15, "hook-sequence": [{"module-code": "prebid", "hook-impl-code": "baz"}]}]}}}}}`),
			givenHooks:                  hooks,
			expectedPlan: Plan[stages.AllProcBidResponsesHook]{
				// first group from host-level plan
				Group[stages.AllProcBidResponsesHook]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[stages.AllProcBidResponsesHook]{
						{Module: "foobar", Code: "foo", Hook: fakeAllProcBidResponsesHook{}},
					},
				},
				// then come groups from account-level plan (default-account-level plan ignored)
				Group[stages.AllProcBidResponsesHook]{
					Timeout: 15 * time.Millisecond,
					Hooks: []HookWrapper[stages.AllProcBidResponsesHook]{
						{Module: "prebid", Code: "baz", Hook: fakeAllProcBidResponsesHook{}},
					},
				},
			},
		},
		"Works with only account-specific plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{}`),
			givenDefaultAccountPlanData: []byte(`{}`),
			giveAccountPlanData:         []byte(`{"endpoints": {"/openrtb2/auction": {"stages": {"allprocbidresponses": {"groups": [{"timeout": 15, "hook-sequence": [{"module-code": "prebid", "hook-impl-code": "baz"}]}]}}}}}`),
			givenHooks:                  hooks,
			expectedPlan: Plan[stages.AllProcBidResponsesHook]{
				Group[stages.AllProcBidResponsesHook]{
					Timeout: 15 * time.Millisecond,
					Hooks: []HookWrapper[stages.AllProcBidResponsesHook]{
						{Module: "prebid", Code: "baz", Hook: fakeAllProcBidResponsesHook{}},
					},
				},
			},
		},
		"Works with empty account-specific execution plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{"endpoints":{"/openrtb2/auction":{"stages":{"allprocbidresponses":{"groups":[{"timeout":5,"hook-sequence":[{"module-code":"foobar","hook-impl-code":"foo"}]}]}}}}}`),
			givenDefaultAccountPlanData: []byte(`{"endpoints": {"/openrtb2/auction": {"stages": {"allprocbidresponses": {"groups": [{"timeout": 10, "hook-sequence": [{"module-code": "foobar", "hook-impl-code": "bar"}, {"module-code": "ortb2blocking", "hook-impl-code": "block_request"}]}, {"timeout": 5, "hook-sequence": [{"module-code": "foobar", "hook-impl-code": "foo"}]}]}}}, "/openrtb2/amp": {"stages": {"entrypoint": {"groups": [{"timeout": 5, "hook-sequence": [{"module-code": "foobar", "hook-impl-code": "foo"}]}]}}}}}`),
			giveAccountPlanData:         []byte(`{}`),
			givenHooks:                  hooks,
			expectedPlan: Plan[stages.AllProcBidResponsesHook]{
				Group[stages.AllProcBidResponsesHook]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[stages.AllProcBidResponsesHook]{
						{Module: "foobar", Code: "foo", Hook: fakeAllProcBidResponsesHook{}},
					},
				},
				Group[stages.AllProcBidResponsesHook]{
					Timeout: 10 * time.Millisecond,
					Hooks: []HookWrapper[stages.AllProcBidResponsesHook]{
						{Module: "foobar", Code: "bar", Hook: fakeAllProcBidResponsesHook{}},
						{Module: "ortb2blocking", Code: "block_request", Hook: fakeAllProcBidResponsesHook{}},
					},
				},
				Group[stages.AllProcBidResponsesHook]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[stages.AllProcBidResponsesHook]{
						{Module: "foobar", Code: "foo", Hook: fakeAllProcBidResponsesHook{}},
					},
				},
			},
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			account := new(config.Account)
			err := json.Unmarshal(test.giveAccountPlanData, &account.Hooks)
			if err != nil {
				t.Fatal(err)
			}

			planBuilder, err := getPlanBuilder(test.givenHooks, test.givenHostPlanData, test.givenDefaultAccountPlanData)
			if err != nil {
				t.Fatal(err)
			}

			plan := planBuilder.PlanForAllProcBidResponsesStage(test.givenEndpoint, account)
			assert.Equal(t, test.expectedPlan, plan)
		})
	}
}

func TestPlanForAuctionResponseStage(t *testing.T) {
	hooks := map[string]map[string]interface{}{
		"foobar": {
			"foo": fakeAuctionResponseHook{},
			"bar": fakeAuctionResponseHook{},
		},
		"ortb2blocking": {
			"block_request": fakeAuctionResponseHook{},
		},
		"prebid": {
			"baz": fakeAuctionResponseHook{},
		},
	}

	testCases := map[string]struct {
		givenEndpoint               string
		givenHostPlanData           []byte
		givenDefaultAccountPlanData []byte
		giveAccountPlanData         []byte
		givenHooks                  map[string]map[string]interface{}
		expectedPlan                Plan[stages.AuctionResponseHook]
	}{
		"Account-specific execution plan rewrites default-account execution plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{"endpoints":{"/openrtb2/auction":{"stages":{"auctionresponse":{"groups":[{"timeout":5,"hook-sequence":[{"module-code":"foobar","hook-impl-code":"foo"}]}]}}}}}`),
			givenDefaultAccountPlanData: []byte(`{"endpoints": {"/openrtb2/auction": {"stages": {"auctionresponse": {"groups": [{"timeout": 10, "hook-sequence": [{"module-code": "foobar", "hook-impl-code": "bar"}, {"module-code": "ortb2blocking", "hook-impl-code": "block_request"}]}, {"timeout": 5, "hook-sequence": [{"module-code": "foobar", "hook-impl-code": "foo"}]}]}}}, "/openrtb2/amp": {"stages": {"entrypoint": {"groups": [{"timeout": 5, "hook-sequence": [{"module-code": "foobar", "hook-impl-code": "foo"}]}]}}}}}`),
			giveAccountPlanData:         []byte(`{"endpoints": {"/openrtb2/auction": {"stages": {"auctionresponse": {"groups": [{"timeout": 15, "hook-sequence": [{"module-code": "prebid", "hook-impl-code": "baz"}]}]}}}}}`),
			givenHooks:                  hooks,
			expectedPlan: Plan[stages.AuctionResponseHook]{
				// first group from host-level plan
				Group[stages.AuctionResponseHook]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[stages.AuctionResponseHook]{
						{Module: "foobar", Code: "foo", Hook: fakeAuctionResponseHook{}},
					},
				},
				// then come groups from account-level plan (default-account-level plan ignored)
				Group[stages.AuctionResponseHook]{
					Timeout: 15 * time.Millisecond,
					Hooks: []HookWrapper[stages.AuctionResponseHook]{
						{Module: "prebid", Code: "baz", Hook: fakeAuctionResponseHook{}},
					},
				},
			},
		},
		"Works with only account-specific plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{}`),
			givenDefaultAccountPlanData: []byte(`{}`),
			giveAccountPlanData:         []byte(`{"endpoints": {"/openrtb2/auction": {"stages": {"auctionresponse": {"groups": [{"timeout": 15, "hook-sequence": [{"module-code": "prebid", "hook-impl-code": "baz"}]}]}}}}}`),
			givenHooks:                  hooks,
			expectedPlan: Plan[stages.AuctionResponseHook]{
				Group[stages.AuctionResponseHook]{
					Timeout: 15 * time.Millisecond,
					Hooks: []HookWrapper[stages.AuctionResponseHook]{
						{Module: "prebid", Code: "baz", Hook: fakeAuctionResponseHook{}},
					},
				},
			},
		},
		"Works with empty account-specific execution plan": {
			givenEndpoint:               "/openrtb2/auction",
			givenHostPlanData:           []byte(`{"endpoints":{"/openrtb2/auction":{"stages":{"auctionresponse":{"groups":[{"timeout":5,"hook-sequence":[{"module-code":"foobar","hook-impl-code":"foo"}]}]}}}}}`),
			givenDefaultAccountPlanData: []byte(`{"endpoints": {"/openrtb2/auction": {"stages": {"auctionresponse": {"groups": [{"timeout": 10, "hook-sequence": [{"module-code": "foobar", "hook-impl-code": "bar"}, {"module-code": "ortb2blocking", "hook-impl-code": "block_request"}]}, {"timeout": 5, "hook-sequence": [{"module-code": "foobar", "hook-impl-code": "foo"}]}]}}}, "/openrtb2/amp": {"stages": {"entrypoint": {"groups": [{"timeout": 5, "hook-sequence": [{"module-code": "foobar", "hook-impl-code": "foo"}]}]}}}}}`),
			giveAccountPlanData:         []byte(`{}`),
			givenHooks:                  hooks,
			expectedPlan: Plan[stages.AuctionResponseHook]{
				Group[stages.AuctionResponseHook]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[stages.AuctionResponseHook]{
						{Module: "foobar", Code: "foo", Hook: fakeAuctionResponseHook{}},
					},
				},
				Group[stages.AuctionResponseHook]{
					Timeout: 10 * time.Millisecond,
					Hooks: []HookWrapper[stages.AuctionResponseHook]{
						{Module: "foobar", Code: "bar", Hook: fakeAuctionResponseHook{}},
						{Module: "ortb2blocking", Code: "block_request", Hook: fakeAuctionResponseHook{}},
					},
				},
				Group[stages.AuctionResponseHook]{
					Timeout: 5 * time.Millisecond,
					Hooks: []HookWrapper[stages.AuctionResponseHook]{
						{Module: "foobar", Code: "foo", Hook: fakeAuctionResponseHook{}},
					},
				},
			},
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			account := new(config.Account)
			err := json.Unmarshal(test.giveAccountPlanData, &account.Hooks)
			if err != nil {
				t.Fatal(err)
			}

			planBuilder, err := getPlanBuilder(test.givenHooks, test.givenHostPlanData, test.givenDefaultAccountPlanData)
			if err != nil {
				t.Fatal(err)
			}

			plan := planBuilder.PlanForAuctionResponseStage(test.givenEndpoint, account)
			assert.Equal(t, test.expectedPlan, plan)
		})
	}
}

func getPlanBuilder(
	moduleHooks map[string]map[string]interface{},
	hostPlanData, accountPlanData []byte,
) (HookExecutionPlanBuilder, error) {
	var err error
	var hooks config.Hooks
	var hostPlan config.HookExecutionPlan
	var defaultAccountPlan config.HookExecutionPlan

	err = json.Unmarshal(hostPlanData, &hostPlan)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(accountPlanData, &defaultAccountPlan)
	if err != nil {
		return nil, err
	}

	hooks.HostExecutionPlan = hostPlan
	hooks.DefaultAccountExecutionPlan = defaultAccountPlan

	repo, err := NewHookRepository(moduleHooks)
	if err != nil {
		return nil, err
	}

	return NewHookExecutionPlanBuilder(hooks, repo), nil
}

type fakeEntrypointHook struct{}

func (h fakeEntrypointHook) Call(ctx context.Context, context *invocation.ModuleContext, payload stages.EntrypointPayload, _ bool) (invocation.HookResult[stages.EntrypointPayload], error) {
	return invocation.HookResult[stages.EntrypointPayload]{}, nil
}

type fakeRawAuctionHook struct{}

func (f fakeRawAuctionHook) Call(ctx context.Context, i invocation.InvocationContext, request stages.BidRequest) (invocation.HookResult[stages.BidRequest], error) {
	return invocation.HookResult[stages.BidRequest]{}, nil
}

type fakeProcessedAuctionHook struct{}

func (f fakeProcessedAuctionHook) Call(ctx context.Context, i invocation.InvocationContext, payload stages.ProcessedAuctionPayload) (invocation.HookResult[stages.ProcessedAuctionPayload], error) {
	return invocation.HookResult[stages.ProcessedAuctionPayload]{}, nil
}

type fakeBidRequestHook struct{}

func (f fakeBidRequestHook) Call(ctx context.Context, i invocation.InvocationContext, payload stages.BidRequestPayload) (invocation.HookResult[stages.BidRequestPayload], error) {
	return invocation.HookResult[stages.BidRequestPayload]{}, nil
}

type fakeRawBidResponseHook struct{}

func (f fakeRawBidResponseHook) Call(ctx context.Context, i invocation.InvocationContext, payload stages.RawBidResponsePayload) (invocation.HookResult[stages.RawBidResponsePayload], error) {
	return invocation.HookResult[stages.RawBidResponsePayload]{}, nil
}

type fakeProcessedBidResponseHook struct{}

func (f fakeProcessedBidResponseHook) Call(ctx context.Context, i invocation.InvocationContext, payload stages.ProcessedBidResponsePayload) (invocation.HookResult[stages.ProcessedBidResponsePayload], error) {
	return invocation.HookResult[stages.ProcessedBidResponsePayload]{}, nil
}

type fakeAllProcBidResponsesHook struct{}

func (f fakeAllProcBidResponsesHook) Call(ctx context.Context, i invocation.InvocationContext, payload stages.AllProcBidResponsesPayload) (invocation.HookResult[stages.AllProcBidResponsesPayload], error) {
	return invocation.HookResult[stages.AllProcBidResponsesPayload]{}, nil
}

type fakeAuctionResponseHook struct{}

func (f fakeAuctionResponseHook) Call(ctx context.Context, i invocation.InvocationContext, response *openrtb2.BidResponse) (invocation.HookResult[*openrtb2.BidResponse], error) {
	return invocation.HookResult[*openrtb2.BidResponse]{}, nil
}
