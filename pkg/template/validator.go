// validator.go — Validate data.json against a preset's schema.
package template

import "fmt"

// ValidateData checks that data.json references only known component IDs.
// Returns warnings (never fatal errors) for graceful degradation.
func ValidateData(data *DataSpec, preset *Preset) []string {
	if data == nil {
		return nil
	}

	// Build ID set from preset components.
	known := make(map[string]struct{}, len(preset.Components))
	for _, c := range preset.Components {
		known[c.ID] = struct{}{}
	}

	var warnings []string
	for id := range data.Components {
		if _, ok := known[id]; !ok {
			warnings = append(warnings, fmt.Sprintf("data references unknown component %q — ignored", id))
		}
	}

	return warnings
}

// FormatSchema returns a human-readable description of the preset's schema.
func FormatSchema(preset *Preset) string {
	if preset.Schema.Description == "" && len(preset.Schema.Components) == 0 {
		return "This preset has no schema documentation.\n"
	}

	var s string
	s += fmt.Sprintf("Preset: %s (v%s) by %s\n", preset.Meta.Name, preset.Meta.Version, preset.Meta.Author)
	if preset.Meta.Description != "" {
		s += preset.Meta.Description + "\n"
	}
	s += "\n"

	if preset.Schema.Description != "" {
		s += preset.Schema.Description + "\n\n"
	}

	s += "Components:\n"
	for id, sc := range preset.Schema.Components {
		s += fmt.Sprintf("\n  [%s] %s\n", id, sc.Description)
		for field, desc := range sc.Fields {
			s += fmt.Sprintf("    %-12s %s\n", field+":", desc)
		}
	}

	return s
}
