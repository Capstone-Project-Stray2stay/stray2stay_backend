package service

import "testing"

func TestParseDiaryDay(t *testing.T) { // UT-BE-DIARY-01
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr bool
	}{
		{name: "valid leap day", value: "2024-02-29", want: "2024-02-29"},
		{name: "ordinary date", value: "2026-11-04", want: "2026-11-04"},
		{name: "non-leap February 29", value: "2023-02-29", wantErr: true},
		{name: "overflowing calendar day", value: "2024-02-30", wantErr: true},
		{name: "wrong date format", value: "29-02-2024", wantErr: true},
		{name: "timestamp is not a date key", value: "2024-02-29T00:00:00", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseDiaryDay(test.value)
			if test.wantErr {
				if err == nil {
					t.Fatalf("parseDiaryDay(%q) succeeded; want an error", test.value)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseDiaryDay(%q) error = %v", test.value, err)
			}
			if got.Format(diaryDateLayout) != test.want {
				t.Errorf("parseDiaryDay(%q) = %q, want %q", test.value, got.Format(diaryDateLayout), test.want)
			}
			if got.Hour() != 0 || got.Minute() != 0 || got.Second() != 0 {
				t.Errorf("parseDiaryDay(%q) time = %s, want local midnight", test.value, got.Format("15:04:05"))
			}
		})
	}
}
