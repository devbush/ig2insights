package domain

import "testing"

func TestParseAccountInput(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantUsername string
		wantErr      bool
	}{
		{
			name:         "full URL",
			input:        "https://www.instagram.com/npcfaizan/",
			wantUsername: "npcfaizan",
			wantErr:      false,
		},
		{
			name:         "URL without trailing slash",
			input:        "https://www.instagram.com/npcfaizan",
			wantUsername: "npcfaizan",
			wantErr:      false,
		},
		{
			name:         "with @ prefix",
			input:        "@npcfaizan",
			wantUsername: "npcfaizan",
			wantErr:      false,
		},
		{
			name:         "plain username",
			input:        "npcfaizan",
			wantUsername: "npcfaizan",
			wantErr:      false,
		},
		{
			name:         "empty input",
			input:        "",
			wantUsername: "",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			acc, err := ParseAccountInput(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseAccountInput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && acc.Username != tt.wantUsername {
				t.Errorf("ParseAccountInput() Username = %v, want %v", acc.Username, tt.wantUsername)
			}
		})
	}
}
