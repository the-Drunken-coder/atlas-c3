package mediatype

import "testing"

func TestNormalizeContentType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		input     string
		want      string
		wantValid bool
	}{
		{name: "empty", input: "", want: "application/octet-stream", wantValid: true},
		{name: "valid", input: " text/plain ; charset=utf-8 ", want: "text/plain; charset=utf-8", wantValid: true},
		{name: "invalid", input: "text/plain\r\nX-Test: injected", want: "application/octet-stream", wantValid: false},
		{name: "control characters", input: "application/\x00json", want: "application/octet-stream", wantValid: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, ok := NormalizeContentType(test.input)
			if got != test.want || ok != test.wantValid {
				t.Fatalf("NormalizeContentType(%q) = (%q, %v), want (%q, %v)", test.input, got, ok, test.want, test.wantValid)
			}
		})
	}
}
