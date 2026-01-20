package domain

import (
	"testing"
)

func TestParseReelInput_URL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantID   string
		wantErr  bool
	}{
		{
			name:    "full URL with /p/",
			input:   "https://www.instagram.com/p/DToLsd-EvGJ/",
			wantID:  "DToLsd-EvGJ",
			wantErr: false,
		},
		{
			name:    "full URL with /reel/",
			input:   "https://www.instagram.com/reel/DToLsd-EvGJ/",
			wantID:  "DToLsd-EvGJ",
			wantErr: false,
		},
		{
			name:    "URL without trailing slash",
			input:   "https://www.instagram.com/p/DToLsd-EvGJ",
			wantID:  "DToLsd-EvGJ",
			wantErr: false,
		},
		{
			name:    "just reel ID",
			input:   "DToLsd-EvGJ",
			wantID:  "DToLsd-EvGJ",
			wantErr: false,
		},
		{
			name:    "invalid input",
			input:   "",
			wantID:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reel, err := ParseReelInput(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseReelInput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && reel.ID != tt.wantID {
				t.Errorf("ParseReelInput() ID = %v, want %v", reel.ID, tt.wantID)
			}
		})
	}
}
