package data

import (
	"testing"
	"time"
)

func TestResolveAdminUserSubscribeExpiry(t *testing.T) {
	now := time.Date(2026, time.July, 21, 12, 0, 0, 0, time.UTC)
	future := time.Date(2026, time.August, 21, 23, 59, 59, 0, time.UTC)
	past := time.Date(2026, time.June, 21, 23, 59, 59, 0, time.UTC)

	tests := []struct {
		name           string
		expiredAt      int64
		wantExpiry     *time.Time
		wantStatus     int8
		wantFinishedAt bool
	}{
		{name: "permanent", expiredAt: 0, wantStatus: 1},
		{name: "future unix seconds", expiredAt: future.Unix(), wantExpiry: &future, wantStatus: 1},
		{name: "future unix milliseconds", expiredAt: future.UnixMilli(), wantExpiry: &future, wantStatus: 1},
		{name: "past unix seconds", expiredAt: past.Unix(), wantExpiry: &past, wantStatus: 3, wantFinishedAt: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expiry, status, finishedAt := resolveAdminUserSubscribeExpiry(tt.expiredAt, now)
			if status != tt.wantStatus {
				t.Fatalf("status = %d, want %d", status, tt.wantStatus)
			}
			if tt.wantExpiry == nil {
				if expiry != nil {
					t.Fatalf("expiry = %v, want nil", expiry)
				}
			} else if expiry == nil || !expiry.Equal(*tt.wantExpiry) {
				t.Fatalf("expiry = %v, want %v", expiry, *tt.wantExpiry)
			}
			if tt.wantFinishedAt {
				if finishedAt == nil || !finishedAt.Equal(now) {
					t.Fatalf("finishedAt = %v, want %v", finishedAt, now)
				}
			} else if finishedAt != nil {
				t.Fatalf("finishedAt = %v, want nil", finishedAt)
			}
		})
	}
}
