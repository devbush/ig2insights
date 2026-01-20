package domain

import (
	"strings"
	"testing"
)

func TestTranscript_ToText(t *testing.T) {
	tr := &Transcript{
		Segments: []Segment{
			{Start: 0.0, End: 3.5, Text: "Hello world."},
			{Start: 3.5, End: 7.0, Text: "How are you?"},
		},
	}

	result := tr.ToText()
	expected := "Hello world. How are you?"

	if result != expected {
		t.Errorf("ToText() = %q, want %q", result, expected)
	}
}

func TestTranscript_ToSRT(t *testing.T) {
	tr := &Transcript{
		Segments: []Segment{
			{Start: 0.0, End: 3.5, Text: "Hello world."},
			{Start: 3.5, End: 7.2, Text: "How are you?"},
		},
	}

	result := tr.ToSRT()

	if !strings.Contains(result, "00:00:00,000 --> 00:00:03,500") {
		t.Errorf("ToSRT() missing first timestamp, got:\n%s", result)
	}
	if !strings.Contains(result, "Hello world.") {
		t.Errorf("ToSRT() missing first text")
	}
	if !strings.Contains(result, "00:00:03,500 --> 00:00:07,200") {
		t.Errorf("ToSRT() missing second timestamp, got:\n%s", result)
	}
}
