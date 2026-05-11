package source

import "testing"

func TestNormalizeTelegramChannelInput(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		channel   string
		publicURL string
	}{
		{
			name:      "at username",
			input:     "@llm_under_hood",
			channel:   "llm_under_hood",
			publicURL: "https://t.me/s/llm_under_hood",
		},
		{
			name:      "bare username",
			input:     "neuraldeep",
			channel:   "neuraldeep",
			publicURL: "https://t.me/s/neuraldeep",
		},
		{
			name:      "telegram channel url",
			input:     "https://t.me/cryptovalerii",
			channel:   "cryptovalerii",
			publicURL: "https://t.me/s/cryptovalerii",
		},
		{
			name:      "telegram public web url",
			input:     "https://t.me/s/llm_under_hood",
			channel:   "llm_under_hood",
			publicURL: "https://t.me/s/llm_under_hood",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeTelegramChannelInput(tt.input)
			if err != nil {
				t.Fatalf("normalize: %v", err)
			}
			if got.Channel != tt.channel || got.PublicURL != tt.publicURL {
				t.Fatalf("NormalizeTelegramChannelInput(%q) = %#v", tt.input, got)
			}
		})
	}
}

func TestNormalizeTelegramChannelInputRejectsInvalidInput(t *testing.T) {
	for _, input := range []string{"", "@", "https://example.com/channel", "https://t.me/"} {
		t.Run(input, func(t *testing.T) {
			if got, err := NormalizeTelegramChannelInput(input); err == nil {
				t.Fatalf("expected error, got %#v", got)
			}
		})
	}
}
