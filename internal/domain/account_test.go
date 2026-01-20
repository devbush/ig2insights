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
		{
			name:         "username with underscore",
			input:        "user_name",
			wantUsername: "user_name",
			wantErr:      false,
		},
		{
			name:         "username with dot",
			input:        "user.name",
			wantUsername: "user.name",
			wantErr:      false,
		},
		{
			name:         "username starting with number",
			input:        "123user",
			wantUsername: "123user",
			wantErr:      false,
		},
		{
			name:         "URL without www",
			input:        "https://instagram.com/npcfaizan/",
			wantUsername: "npcfaizan",
			wantErr:      false,
		},
		{
			name:         "reject post URL",
			input:        "https://www.instagram.com/p/ABC123/",
			wantUsername: "",
			wantErr:      true,
		},
		{
			name:         "reject reel URL",
			input:        "https://www.instagram.com/reel/ABC123/",
			wantUsername: "",
			wantErr:      true,
		},
		{
			name:         "invalid characters rejected",
			input:        "user@name",
			wantUsername: "",
			wantErr:      true,
		},
		{
			name:         "whitespace trimmed",
			input:        "  npcfaizan  ",
			wantUsername: "npcfaizan",
			wantErr:      false,
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

func TestAccount_AccountURL(t *testing.T) {
	tests := []struct {
		name     string
		username string
		want     string
	}{
		{"simple username", "npcfaizan", "https://www.instagram.com/npcfaizan/"},
		{"username with underscore", "user_name", "https://www.instagram.com/user_name/"},
		{"username with dot", "user.name", "https://www.instagram.com/user.name/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			acc := &Account{Username: tt.username}
			if got := acc.AccountURL(); got != tt.want {
				t.Errorf("AccountURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
