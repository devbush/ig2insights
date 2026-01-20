package tui

// OutputOptions represents what the user wants to download
type OutputOptions struct {
	Transcript bool
	Audio      bool
	Video      bool
	Thumbnail  bool
}

// RunOutputSelector displays output options and returns selections
func RunOutputSelector(reelCount int) (*OutputOptions, error) {
	title := "What to download for selected reels?"
	if reelCount == 1 {
		title = "What to download?"
	}

	options := []CheckboxOption{
		{Label: "Transcript (text)", Value: "transcript", Checked: true},
		{Label: "Audio (WAV)", Value: "audio", Checked: false},
		{Label: "Video (MP4)", Value: "video", Checked: false},
		{Label: "Thumbnail (JPG)", Value: "thumbnail", Checked: false},
	}

	selected, err := RunCheckbox(title, options)
	if err != nil {
		return nil, err
	}
	if selected == nil {
		return nil, nil // Cancelled
	}

	opts := &OutputOptions{}
	for _, v := range selected {
		switch v {
		case "transcript":
			opts.Transcript = true
		case "audio":
			opts.Audio = true
		case "video":
			opts.Video = true
		case "thumbnail":
			opts.Thumbnail = true
		}
	}

	return opts, nil
}
