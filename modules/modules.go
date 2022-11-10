package modules

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/hooks"
)

func NewBuilder() Builder {
	return &builder{builders()}
}

type Builder interface {
	Build(cfg config.Modules, client *http.Client) (hooks.HookRepository, error)
}

type (
	// ModuleBuilders mapping between module name and its builder: map[vendor]map[module]ModuleBuilderFn
	ModuleBuilders map[string]map[string]ModuleBuilderFn
	// ModuleBuilderFn returns an interface{} type that implements certain hook interfaces
	ModuleBuilderFn func(cfg json.RawMessage, client *http.Client) (interface{}, error)
)

type builder struct {
	builders ModuleBuilders
}

func (m *builder) Build(cfg config.Modules, client *http.Client) (hooks.HookRepository, error) {
	modules := make(map[string]interface{})
	for vendor, moduleBuilders := range m.builders {
		for moduleName, builder := range moduleBuilders {
			var err error
			var conf json.RawMessage

			id := fmt.Sprintf("%s.%s", vendor, moduleName)
			if data, ok := cfg[vendor][moduleName]; ok {
				if conf, err = json.Marshal(data); err != nil {
					return nil, fmt.Errorf(`failed to marshal "%s" module config: %s`, id, err)
				}
			}

			module, err := builder(conf, client)
			if err != nil {
				return nil, fmt.Errorf(`failed to init "%s" module: %s`, id, err)
			}

			modules[id] = module
		}
	}

	return hooks.NewHookRepository(modules)
}