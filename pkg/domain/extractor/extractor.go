package extractor

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"strings"
	"text/template"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/beehive/pkg/domain/model"
)

var (
	errExtractionFailed = goerr.New("failed to extract IoCs")
)

//go:embed prompts/extract_ioc.md
var extractionPromptTemplate string

var extractionTmpl = template.Must(template.New("extraction").Parse(extractionPromptTemplate))

// ExtractedIoC represents an IoC extracted from text
type ExtractedIoC struct {
	Type        string `json:"type"`
	Value       string `json:"value"`
	Description string `json:"description"`
}

// getIoCSchema returns the JSON schema for IoC extraction
func getIoCSchema() *gollem.Parameter {
	return &gollem.Parameter{
		Type:        gollem.TypeObject,
		Description: "Extracted Indicators of Compromise from the article",
		Properties: map[string]*gollem.Parameter{
			"iocs": {
				Type:        gollem.TypeArray,
				Description: "List of extracted IoCs",
				Items: &gollem.Parameter{
					Type:        gollem.TypeObject,
					Description: "Individual IoC",
					Properties: map[string]*gollem.Parameter{
						"type": {
							Type:        gollem.TypeString,
							Description: "The type of Indicator of Compromise",
							Enum:        []string{"ipv4", "ipv6", "domain", "url", "email", "md5", "sha1", "sha256"},
						},
						"value": {
							Type:        gollem.TypeString,
							Description: "The exact IoC value extracted from the article",
						},
						"description": {
							Type:        gollem.TypeString,
							Description: "Brief context or description of the IoC from the article",
						},
					},
					Required: []string{"type", "value", "description"},
				},
			},
		},
		Required: []string{"iocs"},
	}
}

// extractionResponse represents the structured response from LLM
type extractionResponse struct {
	IoCs []*ExtractedIoC `json:"iocs"`
}

// Extractor provides IoC extraction and processing functionality
type Extractor struct {
	llmClient gollem.LLMClient
}

// New creates a new IoC extractor
func New(llmClient gollem.LLMClient) *Extractor {
	return &Extractor{
		llmClient: llmClient,
	}
}

// ExtractFromArticle extracts IoCs from a blog article using LLM
func (e *Extractor) ExtractFromArticle(ctx context.Context, title, content string) ([]*ExtractedIoC, error) {
	if e.llmClient == nil {
		return nil, goerr.New("LLM client not configured")
	}

	// Render prompt template
	var promptBuf bytes.Buffer
	if err := extractionTmpl.Execute(&promptBuf, map[string]string{
		"Title":   title,
		"Content": content,
	}); err != nil {
		return nil, goerr.Wrap(err, "failed to render prompt template")
	}
	prompt := promptBuf.String()

	// Create a session with response schema and JSON content type
	session, err := e.llmClient.NewSession(ctx,
		gollem.WithSessionContentType(gollem.ContentTypeJSON),
		gollem.WithSessionResponseSchema(getIoCSchema()),
	)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create LLM session")
	}

	// Generate content using LLM
	resp, err := session.GenerateContent(ctx, gollem.Text(prompt))
	if err != nil {
		return nil, goerr.Wrap(errExtractionFailed, "LLM generation failed",
			goerr.V("error", err.Error()))
	}

	// Extract text from response
	var responseText string
	if len(resp.Texts) > 0 {
		responseText = resp.Texts[0]
	}

	if responseText == "" {
		return nil, goerr.Wrap(errExtractionFailed, "empty LLM response")
	}

	// Parse JSON response with schema
	var response extractionResponse
	if err := json.Unmarshal([]byte(responseText), &response); err != nil {
		return nil, goerr.Wrap(errExtractionFailed, "failed to parse LLM response",
			goerr.V("response", responseText),
			goerr.V("error", err.Error()))
	}

	return response.IoCs, nil
}

// GenerateEmbedding generates a vector embedding for the given text
func (e *Extractor) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if e.llmClient == nil {
		return nil, goerr.New("LLM client not configured")
	}

	// Use gollem's embedding generation if available
	// For now, return a placeholder implementation
	// This should be implemented when gollem's embedding API is available

	// Create a dummy embedding with correct dimensions
	embedding := make([]float32, model.EmbeddingDimension)

	// TODO: Replace with actual LLM embedding generation
	// resp, err := e.llmClient.GenerateEmbedding(ctx, text)
	// if err != nil {
	//     return nil, goerr.Wrap(errEmbeddingFailed, "embedding generation failed", err)
	// }
	// return resp.Embedding, nil

	return embedding, nil
}

// ConvertToIoC converts an ExtractedIoC to a domain IoC model.
// The contextParams should contain feed-specific context for deduplication:
//   - For RSS feeds: {"article_guid": "...", "article_url": "..."}
//   - For threat feeds: {"entry_id": "..."}
func ConvertToIoC(sourceID, sourceType, sourceURL string, extracted *ExtractedIoC, contextParams map[string]string) (*model.IoC, error) {
	// Map string type to IoCType
	iocType := mapStringToIoCType(extracted.Type)
	if iocType == "" {
		return nil, goerr.Wrap(model.ErrInvalidIoCType, "unknown IoC type from LLM",
			goerr.V("type", extracted.Type),
			goerr.V("value", extracted.Value))
	}

	// Normalize the value
	normalizedValue := model.NormalizeValue(iocType, extracted.Value)

	// Generate context key based on source type and parameters
	contextKey := model.GenerateContextKey(sourceType, contextParams)

	// Generate ID with context awareness
	iocID := model.GenerateID(sourceID, iocType, normalizedValue, contextKey)

	ioc := &model.IoC{
		ID:          iocID,
		SourceID:    sourceID,
		SourceType:  sourceType,
		Type:        iocType,
		Value:       normalizedValue,
		Description: extracted.Description,
		SourceURL:   sourceURL,
		Status:      model.IoCStatusActive,
		Embedding:   make([]float32, model.EmbeddingDimension), // Will be filled later
	}

	return ioc, nil
}

// mapStringToIoCType maps a string type to IoCType
// Returns an empty IoCType if the type string is unknown
func mapStringToIoCType(typeStr string) model.IoCType {
	typeStr = strings.ToLower(strings.TrimSpace(typeStr))

	switch typeStr {
	case "ipv4":
		return model.IoCTypeIPv4
	case "ipv6":
		return model.IoCTypeIPv6
	case "domain":
		return model.IoCTypeDomain
	case "url":
		return model.IoCTypeURL
	case "email":
		return model.IoCTypeEmail
	case "md5":
		return model.IoCTypeMD5
	case "sha1":
		return model.IoCTypeSHA1
	case "sha256":
		return model.IoCTypeSHA256
	default:
		// Unknown type - return empty string
		// This should not happen if LLM follows the schema, but handle gracefully
		return ""
	}
}

// ExtractIoCsFromFeedEntry converts a feed entry to IoC models
func ExtractIoCsFromFeedEntry(sourceID string, entry interface{}) ([]*model.IoC, error) {
	// This is a placeholder - actual implementation depends on feed entry structure
	// For now, return empty slice
	return nil, nil
}

// ValidateAndNormalize validates and normalizes an IoC.
// Note: This function cannot regenerate the ID as it requires contextParams.
// ID generation should be done during IoC creation, not during validation.
func ValidateAndNormalize(ioc *model.IoC) error {
	// Normalize value based on type
	ioc.Value = model.NormalizeValue(ioc.Type, ioc.Value)

	// Validate
	return model.ValidateIoC(ioc)
}
