package flow_test

import (
	"strings"
	"testing"

	"github.com/Hacieva/clinic-scheduler/bot/internal/flow"
)

func TestStateConstants_UniqueAndNonEmpty(t *testing.T) {
	states := []string{
		flow.StateStart,
		flow.StateChooseDirection,
		flow.StateChooseDoctor,
		flow.StateChooseService,
		flow.StateChooseDate,
		flow.StateChooseTime,
		flow.StateEnterName,
		flow.StateEnterPhone,
		flow.StateConfirm,
	}
	seen := make(map[string]bool, len(states))
	for _, s := range states {
		if s == "" {
			t.Error("state constant must not be empty")
		}
		if seen[s] {
			t.Errorf("duplicate state constant value: %q", s)
		}
		seen[s] = true
	}
}

func TestCallbackPrefixes_NoColon(t *testing.T) {
	// Prefixes are combined with values using ":", so they must not contain ":" themselves.
	prefixes := []string{
		flow.CallbackPrefixDirection,
		flow.CallbackPrefixDoctor,
		flow.CallbackPrefixService,
		flow.CallbackPrefixDate,
		flow.CallbackPrefixTime,
	}
	for _, p := range prefixes {
		if p == "" {
			t.Errorf("callback prefix must not be empty")
		}
		if strings.Contains(p, ":") {
			t.Errorf("callback prefix %q must not contain ':', it is used as a separator", p)
		}
	}
}

func TestCallbackActions_NonEmpty(t *testing.T) {
	for _, a := range []string{flow.CallbackConfirm, flow.CallbackCancel} {
		if a == "" {
			t.Error("action callback constant must not be empty")
		}
	}
}

func TestCallbackCancel_IsCancel(t *testing.T) {
	// Ensure the cancel action is exactly "cancel" — handler routing depends on this value.
	if flow.CallbackCancel != "cancel" {
		t.Errorf("CallbackCancel: want %q, got %q", "cancel", flow.CallbackCancel)
	}
}
