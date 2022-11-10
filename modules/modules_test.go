package modules

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/prebid/prebid-server/hooks"
	"github.com/prebid/prebid-server/hooks/hookstage"
	"github.com/stretchr/testify/assert"
)

func TestModuleBuilderBuild(t *testing.T) {
	vendor := "acme"
	moduleName := "foobar"

	testCases := map[string]struct {
		isHookFound         bool
		expectedHook        interface{}
		givenModule         interface{}
		expectedErr         error
		givenHookBuilderErr error
		givenGetHookFn      func(repo hooks.HookRepository, module string) (interface{}, bool)
	}{
		"Can build with entrypoint hook": {
			givenModule:  module{},
			expectedHook: module{},
			isHookFound:  true,
			givenGetHookFn: func(repo hooks.HookRepository, module string) (interface{}, bool) {
				return repo.GetEntrypointHook(module)
			},
		},
		"Can build with auction response hook": {
			givenModule:  module{},
			expectedHook: module{},
			isHookFound:  true,
			givenGetHookFn: func(repo hooks.HookRepository, module string) (interface{}, bool) {
				return repo.GetAuctionResponseHook(module)
			},
		},
		"Fails to find not registered hook": {
			givenModule:  module{},
			expectedHook: nil,
			isHookFound:  false,
			givenGetHookFn: func(repo hooks.HookRepository, module string) (interface{}, bool) {
				return repo.GetAllProcessedBidResponsesHook(module) // ask for hook not implemented in module
			},
		},
		"Builder fails if module does not implement any hook interface": {
			expectedHook: struct{}{},
			expectedErr:  fmt.Errorf(`hook "%s.%s" does not implement any supported hook interface`, vendor, moduleName),
		},
		"Fails if module builder function returns error": {
			givenModule:         module{},
			givenHookBuilderErr: errors.New("failed to build module"),
			expectedErr:         fmt.Errorf(`failed to init "%s.%s" module: %s`, vendor, moduleName, "failed to build module"),
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			builder := &builder{
				builders: ModuleBuilders{
					vendor: {
						moduleName: func(cfg json.RawMessage, client *http.Client) (interface{}, error) {
							return test.givenModule, test.givenHookBuilderErr
						},
					},
				},
			}

			repo, err := builder.Build(nil, http.DefaultClient)
			assert.Equal(t, test.expectedErr, err)
			if test.expectedErr == nil {
				hook, found := test.givenGetHookFn(repo, fmt.Sprintf("%s.%s", vendor, moduleName))
				assert.Equal(t, test.isHookFound, found)
				assert.IsType(t, test.expectedHook, hook)
			}
		})
	}
}

type module struct{}

func (h module) HandleEntrypointHook(_ context.Context, _ *hookstage.ModuleContext, _ hookstage.EntrypointPayload) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	return hookstage.HookResult[hookstage.EntrypointPayload]{}, nil
}

func (h module) HandleAuctionResponseHook(_ context.Context, _ *hookstage.ModuleContext, _ hookstage.AuctionResponsePayload) (hookstage.HookResult[hookstage.AuctionResponsePayload], error) {
	return hookstage.HookResult[hookstage.AuctionResponsePayload]{}, nil
}