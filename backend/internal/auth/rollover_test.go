package auth

import (
	"testing"
	"time"
)

func TestNextRollover(t *testing.T) {
	istanbul, err := time.LoadLocation("Europe/Istanbul")
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		from time.Time
		want time.Time
	}{
		{
			name: "afternoon → next-day 03:00",
			from: time.Date(2026, 6, 10, 18, 0, 0, 0, istanbul),
			want: time.Date(2026, 6, 11, 3, 0, 0, 0, istanbul),
		},
		{
			name: "late night → next-day 03:00",
			from: time.Date(2026, 6, 10, 23, 30, 0, 0, istanbul),
			want: time.Date(2026, 6, 11, 3, 0, 0, 0, istanbul),
		},
		{
			name: "post-midnight → same-day 03:00",
			from: time.Date(2026, 6, 11, 1, 0, 0, 0, istanbul),
			want: time.Date(2026, 6, 11, 3, 0, 0, 0, istanbul),
		},
		{
			name: "exactly 03:00 → next-day 03:00",
			from: time.Date(2026, 6, 10, 3, 0, 0, 0, istanbul),
			want: time.Date(2026, 6, 11, 3, 0, 0, 0, istanbul),
		},
		{
			name: "morning → next-day 03:00",
			from: time.Date(2026, 6, 10, 9, 0, 0, 0, istanbul),
			want: time.Date(2026, 6, 11, 3, 0, 0, 0, istanbul),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := NextRollover(tc.from, istanbul)
			if !got.Equal(tc.want) {
				t.Errorf("NextRollover(%v) = %v, want %v", tc.from, got, tc.want)
			}
		})
	}
}
