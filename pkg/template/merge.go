// merge.go — Merge data.json overrides onto preset defaults.
package template

import "sort"

// MergeData combines preset component defaults with user-provided data overrides.
// Components with visible=false are excluded from the result.
// Position (X/Y/Width/Height) is always from the preset — data cannot override it.
func MergeData(preset *Preset, data *DataSpec) []ResolvedComponent {
	w := preset.Canvas.Width
	h := preset.Canvas.Height

	var result []ResolvedComponent

	for _, comp := range preset.Components {
		merged := comp.Defaults

		// Apply data overrides if present.
		if data != nil {
			if override, ok := data.Components[comp.ID]; ok {
				mergeComponentData(&merged, override)
			}
		}

		// Check visibility.
		if merged.Visible != nil && !*merged.Visible {
			continue
		}

		// Merge style: preset style + data style override.
		finalStyle := comp.Style
		if merged.Style != nil {
			mergeComponentStyle(&finalStyle, *merged.Style)
		}

		result = append(result, ResolvedComponent{
			ID:      comp.ID,
			X:       int(comp.X * float64(w)),
			Y:       int(comp.Y * float64(h)),
			Width:   int(comp.Width * float64(w)),
			Height:  int(comp.Height * float64(h)),
			ZIndex:  comp.ZIndex,
			Padding: max(comp.Padding, 0),
			Style:   finalStyle,
			Data:    merged,
		})
	}

	// Sort by z-index (lower renders first, higher renders on top).
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].ZIndex < result[j].ZIndex
	})

	return result
}

// mergeComponentData overlays user overrides onto defaults.
func mergeComponentData(base *ComponentData, over ComponentData) {
	if over.Visible != nil {
		base.Visible = over.Visible
	}
	if over.Title != "" {
		base.Title = over.Title
	}
	if over.Items != nil {
		base.Items = over.Items // replace, not append
	}
	if over.Style != nil {
		base.Style = over.Style
	}
}

// mergeComponentStyle applies non-zero style overrides.
func mergeComponentStyle(base *ComponentStyle, over ComponentStyle) {
	if over.BackgroundColor != "" {
		base.BackgroundColor = over.BackgroundColor
	}
	if over.BackgroundImage != "" {
		base.BackgroundImage = over.BackgroundImage
	}
	if over.BackgroundFit != "" {
		base.BackgroundFit = over.BackgroundFit
	}
	if over.BorderColor != "" {
		base.BorderColor = over.BorderColor
	}
	if over.BorderWidth > 0 {
		base.BorderWidth = over.BorderWidth
	}
	if over.CornerRadius > 0 {
		base.CornerRadius = over.CornerRadius
	}
	if over.FontPath != "" {
		base.FontPath = over.FontPath
	}
	if over.FontSize > 0 {
		base.FontSize = over.FontSize
	}
	if over.Color != "" {
		base.Color = over.Color
	}
	if over.LineHeight > 0 {
		base.LineHeight = over.LineHeight
	}
	if over.TextAlign != "" {
		base.TextAlign = over.TextAlign
	}
}
