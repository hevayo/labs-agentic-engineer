// Copyright (c) 2026, WSO2 LLC. (https://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package spec

// A REGISTERED EXTERNAL BRINGS ITS SCHEMA WITH IT.
//
// The project's copy of a Registered External carries no `config`: the keys
// live on the org record, which IS the registry (ADR-0009). Two things
// downstream need them anyway — the wiring derivation turns them into the
// env-var names the coding agent codes against, and provisioning authors the
// resource type from them. Without them a build fails with "at least one config
// key required" and the component's design.json carries no wiring at all.

import (
	"context"
	"testing"
)

const registeredDesignJSON = `{
  "name": "expense-api",
  "type": "service",
  "dependencies": [
    {
      "kind": "external",
      "name": "currency-service",
      "description": "Converts a foreign-currency amount to the household currency."
    }
  ]
}`

// The definition the platform stamps for a Registered External: provider and
// contract, and deliberately no config.
const registeredDefinitionJSON = `{
  "name": "currency-service",
  "source": "org",
  "description": "Converts an amount into another currency.",
  "provider": "Open Exchange Rates",
  "style": "rest-api",
  "contract": "openapi.yaml"
}`

func registeredDesignFiles() map[string]string {
	return map[string]string{
		DesignRootFile:                                  "cell {}\n",
		"components/expense-api/design.json":            registeredDesignJSON,
		"dependencies/currency-service/dependency.json": registeredDefinitionJSON,
		"dependencies/currency-service/openapi.yaml":    "openapi: 3.0.3\n",
	}
}

func orgKeys() []ConfigKey {
	return []ConfigKey{{
		Key:         "OPEN_EXCHANGE_RATES_APP_ID",
		Secret:      true,
		Description: "Your Open Exchange Rates App ID.",
	}}
}

func TestRegisteredExternal_ArrivesCarryingTheOrgSchema(t *testing.T) {
	store := NewArtifactStore(nil)
	store.SetExternalResourceResolver(fakeExternalResourceResolver{
		names: map[string]bool{"currency-service": true},
		keys:  map[string][]ConfigKey{"currency-service": orgKeys()},
	})

	design, err := store.AssembleDesignFrom(context.Background(), "acme", registeredDesignFiles())
	if err != nil {
		t.Fatalf("AssembleDesignFrom: %v", err)
	}
	dep := design.Components[0].Dependencies[0]
	if len(dep.Config) != 1 || dep.Config[0].Key != "OPEN_EXCHANGE_RATES_APP_ID" {
		t.Fatalf("dependency config = %+v, want the org record's key — the project's copy carries none",
			dep.Config)
	}
	if !dep.Config[0].Secret {
		t.Errorf("key lost its secret flag: %+v", dep.Config[0])
	}
}

// The keys are what the wiring is made of, so this is the coding agent's half:
// without them the component's design.json carries no wiring and the agent has
// no env-var names to write into workload.yaml.
func TestRegisteredExternal_WiringIsDerivedFromTheOrgSchema(t *testing.T) {
	store := NewArtifactStore(nil)
	store.SetExternalResourceResolver(fakeExternalResourceResolver{
		names: map[string]bool{"currency-service": true},
		keys:  map[string][]ConfigKey{"currency-service": orgKeys()},
	})
	design, err := store.AssembleDesignFrom(context.Background(), "acme", registeredDesignFiles())
	if err != nil {
		t.Fatalf("AssembleDesignFrom: %v", err)
	}

	deriveDependencyWiring(design.Components, map[string]CRTType{}, "expenses")

	wiring := design.Components[0].Dependencies[0].Wiring
	if wiring == nil {
		t.Fatal("no wiring derived for a registered external — the coding agent has nothing to bind")
	}
	if wiring.Ref == "" {
		t.Errorf("wiring has no ref: %+v", wiring)
	}
	if got := wiring.EnvBindings["OPEN_EXCHANGE_RATES_APP_ID"]; got != "OPEN_EXCHANGE_RATES_APP_ID" {
		t.Errorf("env bindings = %+v, want the org key bound to itself", wiring.EnvBindings)
	}
}

// A dependency the org does not hold keeps the design's own answer: a Project
// External's keys are its definition's, and an empty one stays empty.
func TestRegisteredExternal_UnregisteredNameIsLeftAlone(t *testing.T) {
	store := NewArtifactStore(nil)
	store.SetExternalResourceResolver(fakeExternalResourceResolver{
		names: map[string]bool{},
		keys:  map[string][]ConfigKey{"currency-service": orgKeys()},
	})

	design, err := store.AssembleDesignFrom(context.Background(), "acme", registeredDesignFiles())
	if err != nil {
		t.Fatalf("AssembleDesignFrom: %v", err)
	}
	if got := design.Components[0].Dependencies[0].Config; len(got) != 0 {
		t.Fatalf("config = %+v, want none — the catalog does not hold this name", got)
	}
}
