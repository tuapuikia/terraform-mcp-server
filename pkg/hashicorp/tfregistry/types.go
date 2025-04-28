package tfregistry

import "time"

// TerraformModule represents the structure of a Terraform module list response.
// Note: The API seems to return different structures, this one matches the
// format where the top-level key is "modules".
type TerraformModules struct {
	Data []struct {
		ID          string    `json:"id"`
		Owner       string    `json:"owner"`
		Namespace   string    `json:"namespace"`
		Name        string    `json:"name"`
		Version     string    `json:"version"`
		Provider    string    `json:"provider"`
		Description string    `json:"description"`
		Source      string    `json:"source"`
		Tag         string    `json:"tag"`
		PublishedAt time.Time `json:"published_at"`
		Downloads   int64     `json:"downloads"`
		Verified    bool      `json:"verified"`
	} `json:"modules"`
}

// ModuleInput represents a Terraform module input variable.
type ModuleInput struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Default     any    `json:"default"` // Can be string, bool, number, etc.
	Required    bool   `json:"required"`
}

// ModuleOutput represents a Terraform module output value.
type ModuleOutput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ModuleDependency represents a Terraform module dependency.
type ModuleDependency struct {
	Name    string `json:"name"`
	Source  string `json:"source"`
	Version string `json:"version"`
}

// ModuleProviderDependency represents a Terraform provider dependency.
type ModuleProviderDependency struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Source    string `json:"source"`
	Version   string `json:"version"`
}

// ModuleResource represents a resource within a Terraform module.
type ModuleResource struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// ModulePart represents the structure of the root, submodules, or examples
// within a Terraform module version details response.
type ModulePart struct {
	Path                 string                     `json:"path"`
	Name                 string                     `json:"name"`
	Readme               string                     `json:"readme"`
	Empty                bool                       `json:"empty"`
	Inputs               []ModuleInput              `json:"inputs"`
	Outputs              []ModuleOutput             `json:"outputs"`
	Dependencies         []ModuleDependency         `json:"dependencies"`
	ProviderDependencies []ModuleProviderDependency `json:"provider_dependencies"`
	Resources            []ModuleResource           `json:"resources"`
}

// TerraformModuleVersionDetails represents the detailed structure of a specific
// Terraform module version response.
type TerraformModuleVersionDetails struct {
	ID              string       `json:"id"`
	Owner           string       `json:"owner"`
	Namespace       string       `json:"namespace"`
	Name            string       `json:"name"`
	Version         string       `json:"version"`
	Provider        string       `json:"provider"`
	ProviderLogoURL string       `json:"provider_logo_url"`
	Description     string       `json:"description"`
	Source          string       `json:"source"`
	Tag             string       `json:"tag"`
	PublishedAt     time.Time    `json:"published_at"`
	Downloads       int64        `json:"downloads"`
	Verified        bool         `json:"verified"`
	Root            ModulePart   `json:"root"`
	Submodules      []ModulePart `json:"submodules"`
	Examples        []ModulePart `json:"examples"`
	Providers       []string     `json:"providers"`
	Versions        []string     `json:"versions"`
	Deprecation     any          `json:"deprecation"` // Assuming it can be null or an object
}
