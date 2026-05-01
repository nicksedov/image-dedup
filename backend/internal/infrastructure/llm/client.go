package llm

// Client interface for VL LLM communication
type Client interface {
	// Recognize performs OCR recognition on an image with given system prompt
	// Returns markdown content and error
	Recognize(imagePath string, systemPrompt string) (string, error)

	// ListModels returns a list of available models from the LLM server
	ListModels() ([]ModelInfo, error)
}

// ModelInfo represents information about an available LLM model
type ModelInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Size int64  `json:"size,omitempty"`
}

// Provider type enumeration
const (
	ProviderOllama = "ollama"
	ProviderOpenAI = "openai"
)
