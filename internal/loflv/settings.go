package loflv

import (
	"fmt"
	"strings"

	"github.com/dariuskasiulevicius/reitingtuvas/internal/texts"
)

// LVSettings holds lof.lv source configuration.
type LVSettings struct {
	BaseURL  string
	SeriesID string
}

// ParseLVSettings extracts LV settings from the raw settings map.
// Returns nil (no error) when both keys are empty (LV disabled).
// Returns error when exactly one key is set.
func ParseLVSettings(raw map[string]string) (*LVSettings, error) {
	base := strings.TrimSpace(raw[texts.SettingLVBaseURL])
	series := strings.TrimSpace(raw[texts.SettingLVSeriesID])

	if base == "" && series == "" {
		return nil, nil
	}
	if base == "" || series == "" {
		return nil, fmt.Errorf("abu LV nustatymai turi būti užpildyti: %q ir %q",
			texts.SettingLVBaseURL, texts.SettingLVSeriesID)
	}

	return &LVSettings{BaseURL: base, SeriesID: series}, nil
}

// Enabled returns true when LV collection is configured.
func (s *LVSettings) Enabled() bool {
	return s != nil && s.BaseURL != "" && s.SeriesID != ""
}
