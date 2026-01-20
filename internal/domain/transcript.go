package domain

import (
	"fmt"
	"strings"
	"time"
)

// Segment represents a timed segment of transcribed text
type Segment struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
}

// Transcript represents the full transcription result
type Transcript struct {
	Text          string    `json:"text"`
	Segments      []Segment `json:"segments"`
	Model         string    `json:"model"`
	Language      string    `json:"language"`
	TranscribedAt time.Time `json:"transcribed_at"`
}

// ToText returns plain text concatenation of all segments
func (t *Transcript) ToText() string {
	if t.Text != "" {
		return t.Text
	}

	var parts []string
	for _, seg := range t.Segments {
		parts = append(parts, strings.TrimSpace(seg.Text))
	}
	return strings.Join(parts, " ")
}

// ToSRT returns the transcript in SRT subtitle format
func (t *Transcript) ToSRT() string {
	var sb strings.Builder

	for i, seg := range t.Segments {
		// Sequence number
		sb.WriteString(fmt.Sprintf("%d\n", i+1))
		// Timestamps
		sb.WriteString(fmt.Sprintf("%s --> %s\n", formatSRTTime(seg.Start), formatSRTTime(seg.End)))
		// Text
		sb.WriteString(strings.TrimSpace(seg.Text))
		sb.WriteString("\n\n")
	}

	return strings.TrimSuffix(sb.String(), "\n")
}

// formatSRTTime converts seconds to SRT timestamp format (HH:MM:SS,mmm)
func formatSRTTime(seconds float64) string {
	hours := int(seconds) / 3600
	minutes := (int(seconds) % 3600) / 60
	secs := int(seconds) % 60
	millis := int((seconds - float64(int(seconds))) * 1000)

	return fmt.Sprintf("%02d:%02d:%02d,%03d", hours, minutes, secs, millis)
}
