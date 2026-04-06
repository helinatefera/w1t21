package service

// ExperimentComponent defines a registered UI component that supports A/B
// testing. Every experiment name must match an entry here, and variant names
// must be drawn from the component's allowed set.
type ExperimentComponent struct {
	// Description is a human-readable summary shown in the admin UI.
	Description string
	// AllowedVariants maps variant name → short description.
	// The map must contain at least two entries (control + test).
	AllowedVariants map[string]string
}

// ExperimentRegistry maps experiment names to the UI components they control.
// Adding a new A/B-testable component requires registering it here AND
// implementing the corresponding frontend branch.
var ExperimentRegistry = map[string]ExperimentComponent{
	"catalog_layout": {
		Description: "Controls the collectible catalog grid layout (CatalogPage.tsx)",
		AllowedVariants: map[string]string{
			"grid": "Multi-column responsive grid (default)",
			"list": "Single-column list layout",
		},
	},
	"checkout_flow": {
		Description: "Controls the order checkout experience (OrderDetailPage.tsx)",
		AllowedVariants: map[string]string{
			"standard": "Current multi-step checkout (default)",
			"express":  "Single-page express checkout",
		},
	},
	"search_ranking": {
		Description: "Controls the collectible search/sort algorithm (CatalogPage.tsx)",
		AllowedVariants: map[string]string{
			"relevance": "Relevance-based ranking (default)",
			"popular":   "Popularity-weighted ranking",
		},
	},
	"datetime-local-test": {
		Description: "Test experiment for validating datetime-local input format",
		AllowedVariants: map[string]string{
			"A": "Variant A",
			"B": "Variant B",
		},
	},
	"rfc3339-test": {
		Description: "Test experiment for validating RFC3339 input format",
		AllowedVariants: map[string]string{
			"A": "Variant A",
			"B": "Variant B",
		},
	},
}

// ValidateExperiment checks that the experiment name and both variant names
// are present in the registry. Returns an empty string on success, or a
// human-readable error message on failure.
func ValidateExperiment(name, controlVariant, testVariant string) string {
	comp, ok := ExperimentRegistry[name]
	if !ok {
		allowed := make([]string, 0, len(ExperimentRegistry))
		for k := range ExperimentRegistry {
			allowed = append(allowed, k)
		}
		return "unknown experiment name '" + name + "'; registered experiments: " + joinStrings(allowed)
	}

	if _, ok := comp.AllowedVariants[controlVariant]; !ok {
		return "control variant '" + controlVariant + "' is not registered for experiment '" + name + "'; allowed: " + joinStrings(variantKeys(comp))
	}
	if _, ok := comp.AllowedVariants[testVariant]; !ok {
		return "test variant '" + testVariant + "' is not registered for experiment '" + name + "'; allowed: " + joinStrings(variantKeys(comp))
	}
	if controlVariant == testVariant {
		return "control and test variants must be different"
	}
	return ""
}

func variantKeys(c ExperimentComponent) []string {
	keys := make([]string, 0, len(c.AllowedVariants))
	for k := range c.AllowedVariants {
		keys = append(keys, k)
	}
	return keys
}

func joinStrings(ss []string) string {
	if len(ss) == 0 {
		return "(none)"
	}
	out := ss[0]
	for _, s := range ss[1:] {
		out += ", " + s
	}
	return out
}

// RegistryEntryDTO is the JSON-safe representation of a registered experiment.
type RegistryEntryDTO struct {
	Description string   `json:"description"`
	Variants    []string `json:"variants"`
}

// GetRegistryDTO returns the experiment registry as a JSON-serialisable map
// so the frontend can fetch it instead of hardcoding experiment names.
func GetRegistryDTO() map[string]RegistryEntryDTO {
	out := make(map[string]RegistryEntryDTO, len(ExperimentRegistry))
	for name, comp := range ExperimentRegistry {
		out[name] = RegistryEntryDTO{
			Description: comp.Description,
			Variants:    variantKeys(comp),
		}
	}
	return out
}
