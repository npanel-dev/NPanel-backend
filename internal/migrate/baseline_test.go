package migrate

import "testing"

func TestPortableBaselineIsCompleteAndUnambiguous(t *testing.T) {
	baseline, err := loadPortableBaseline()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(baseline.AuthMethods), 8; got != want {
		t.Fatalf("auth method count = %d, want %d", got, want)
	}
	if got, want := len(baseline.Payments), 1; got != want {
		t.Fatalf("payment count = %d, want %d", got, want)
	}
	if got, want := len(baseline.Systems), 61; got != want {
		t.Fatalf("system setting count = %d, want %d", got, want)
	}
	if got, want := len(baseline.Applications), 6; got != want {
		t.Fatalf("application count = %d, want %d", got, want)
	}

	authMethods := make(map[string]struct{}, len(baseline.AuthMethods))
	for _, item := range baseline.AuthMethods {
		if item.Method == "" {
			t.Fatal("portable baseline contains an empty auth method")
		}
		if _, exists := authMethods[item.Method]; exists {
			t.Fatalf("duplicate auth method %q", item.Method)
		}
		authMethods[item.Method] = struct{}{}
	}

	systemKeys := make(map[string]struct{}, len(baseline.Systems))
	for _, item := range baseline.Systems {
		if item.Key == "" {
			t.Fatal("portable baseline contains an empty system key")
		}
		if _, exists := systemKeys[item.Key]; exists {
			t.Fatalf("duplicate system key %q", item.Key)
		}
		systemKeys[item.Key] = struct{}{}
	}
	if _, exists := systemKeys["DeviceLimit"]; !exists {
		t.Fatal("portable baseline is missing DeviceLimit")
	}

	applicationNames := make(map[string]struct{}, len(baseline.Applications))
	applicationIDs := make(map[int64]struct{}, len(baseline.Applications))
	for _, item := range baseline.Applications {
		if _, exists := applicationNames[item.Name]; exists {
			t.Fatalf("duplicate application name %q", item.Name)
		}
		if _, exists := applicationIDs[item.ID]; exists {
			t.Fatalf("duplicate application id %d", item.ID)
		}
		applicationNames[item.Name] = struct{}{}
		applicationIDs[item.ID] = struct{}{}
	}
}
