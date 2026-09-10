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

// READING BACK A VERSION THE USER NAMED.
//
// Cutting `m1` always worked; every read-at-tag then refused it, because the
// readers asked whether the NAME parsed as `v<N>` rather than whether the tag
// was one of the project's versions. The build died at the roles gate with
// `read design at m1: "m1" is not a v<N> spec tag`, so a named version was
// unbuildable — a whole feature that could be used but not used twice.
//
// These run over the real-git harness: a real annotated tag, read back through
// the real store.

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func seedForRead() map[string]string {
	return map[string]string{
		"specs/requirements/prd.md": "# PRD\n\nfirst\n",
		"specs/design/design.cell":  "cell {}\n",
		"specs/design/components/orders-api/design.json": `{
  "name": "orders-api",
  "type": "service",
  "dependencies": []
}`,
	}
}

func TestReadAtTag_AcceptsAVersionTheUserNamed(t *testing.T) {
	ctx := context.Background()
	r := newRig(t, seedForRead())
	tagAt(t, r, "m1", specTagSubject+"m1", "2026-01-01T10:00:00+00:00")

	design, err := r.svc.GetDesignAtSpecTag(ctx, r.org, r.proj, "m1")
	if err != nil {
		t.Fatalf("GetDesignAtSpecTag(m1): %v — a named version must read back", err)
	}
	if len(design) == 0 {
		t.Fatalf("design at m1 is empty, want the seeded bundle")
	}

	reqs, err := r.svc.GetRequirementsAtTag(ctx, r.org, r.proj, "m1")
	if err != nil {
		t.Fatalf("GetRequirementsAtTag(m1): %v", err)
	}
	if len(reqs) == 0 {
		t.Fatalf("requirements at m1 are empty, want prd.md")
	}
}

// The dev workflow re-checks the tag it is about to plan from. It has to accept
// the same names the cut does, or the run fails one step later than the gate.
func TestValidateSpecAtTag_AcceptsAVersionTheUserNamed(t *testing.T) {
	ctx := context.Background()
	r := newRig(t, seedForRead())
	tagAt(t, r, "payments-v2", specTagSubject+"payments-v2", "2026-01-01T10:00:00+00:00")

	if err := r.svc.ValidateSpecAtTag(ctx, r.org, r.proj, "payments-v2"); err != nil {
		// The gate itself may refuse the seeded spec on its own merits; what
		// must never happen is a refusal about the NAME.
		if errors.Is(err, ErrInvalidVersionTag) {
			t.Fatalf("ValidateSpecAtTag(payments-v2) refused the name: %v", err)
		}
	}
}

// The guard still earns its place: a tag this project does not have is refused
// by name, rather than surfacing later as an empty tree.
func TestReadAtTag_RefusesATagThatIsNotAVersion(t *testing.T) {
	ctx := context.Background()
	r := newRig(t, seedForRead())
	tagAt(t, r, "m1", specTagSubject+"m1", "2026-01-01T10:00:00+00:00")
	// A release tag somebody pushed — a real ref, and not a version.
	tagAt(t, r, "release-2026-01", "ship it", "2026-01-02T10:00:00+00:00")

	for _, tag := range []string{"release-2026-01", "nope", ""} {
		_, err := r.svc.GetDesignAtSpecTag(ctx, r.org, r.proj, tag)
		if err == nil {
			t.Errorf("GetDesignAtSpecTag(%q) = nil error, want a refusal", tag)
			continue
		}
		if !errors.Is(err, ErrInvalidVersionTag) {
			t.Errorf("GetDesignAtSpecTag(%q) = %v, want ErrInvalidVersionTag", tag, err)
		}
		if tag != "" && !strings.Contains(err.Error(), tag) {
			t.Errorf("refusal for %q does not name it: %v", tag, err)
		}
	}
}
