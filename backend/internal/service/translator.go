package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/vertexai/genai"
	"github.com/kickwatch/backend/internal/model"
	"google.golang.org/api/option"
)

// TranslatorService handles translation of campaign content using Vertex AI.
type TranslatorService struct {
	client    *genai.Client
	projectID string
	location  string
}

// NewTranslatorService creates a new translator service using Vertex AI.
// Credentials are loaded from GOOGLE_SERVICE_ACCOUNT_JSON env var.
// Uses GCP startup credits from Vertex AI.
func NewTranslatorService(ctx context.Context, projectID, location string) (*TranslatorService, error) {
	var opts []option.ClientOption

	// Load credentials from JSON string (from AWS Secrets Manager)
	if credsJSON := os.Getenv("GOOGLE_SERVICE_ACCOUNT_JSON"); credsJSON != "" {
		opts = append(opts, option.WithCredentialsJSON([]byte(credsJSON)))
		log.Println("Translator: using Vertex AI credentials from GOOGLE_SERVICE_ACCOUNT_JSON")
	} else if credsFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); credsFile != "" {
		opts = append(opts, option.WithCredentialsFile(credsFile))
		log.Printf("Translator: using Vertex AI credentials from file %s", credsFile)
	} else {
		return nil, fmt.Errorf("no Google credentials found (set GOOGLE_SERVICE_ACCOUNT_JSON or GOOGLE_APPLICATION_CREDENTIALS)")
	}

	client, err := genai.NewClient(ctx, projectID, location, opts...)
	if err != nil {
		return nil, fmt.Errorf("create vertex ai client: %w", err)
	}

	log.Printf("Translator: initialized Vertex AI (project=%s, location=%s)", projectID, location)
	return &TranslatorService{
		client:    client,
		projectID: projectID,
		location:  location,
	}, nil
}

// TranslateCampaigns translates campaign names, blurbs, and creator names to Chinese.
// Uses batch translation to minimize API calls and costs.
// Uses Vertex AI which consumes GCP startup credits.
func (t *TranslatorService) TranslateCampaigns(campaigns []model.Campaign) error {
	if len(campaigns) == 0 {
		return nil
	}

	// Use Gemini 1.5 Flash (stable model with higher quotas than experimental)
	model := t.client.GenerativeModel("gemini-1.5-flash-002")
	model.SetTemperature(0.3) // Lower temperature for more consistent translations

	// Batch translate in groups of 5 to stay within rate limits
	const batchSize = 5
	for i := 0; i < len(campaigns); i += batchSize {
		end := i + batchSize
		if end > len(campaigns) {
			end = len(campaigns)
		}
		batch := campaigns[i:end]

		if err := t.translateBatch(model, batch); err != nil {
			log.Printf("Translator: batch %d-%d error: %v", i, end-1, err)
			// Continue with next batch instead of failing entirely
			continue
		}

		// Rate limiting: 2-second delay between batches to avoid quota exhaustion
		if end < len(campaigns) {
			time.Sleep(2 * time.Second)
		}
	}

	return nil
}

// translateBatch translates a batch of campaigns using a single Vertex AI API call.
func (t *TranslatorService) translateBatch(model *genai.GenerativeModel, campaigns []model.Campaign) error {
	type translationInput struct {
		Index       int    `json:"index"`
		Name        string `json:"name"`
		Blurb       string `json:"blurb,omitempty"`
		CreatorName string `json:"creator_name,omitempty"`
	}

	type translationOutput struct {
		Index     int    `json:"index"`
		NameZh    string `json:"name_zh"`
		BlurbZh   string `json:"blurb_zh,omitempty"`
		CreatorZh string `json:"creator_zh,omitempty"`
	}

	// Build input JSON
	inputs := make([]translationInput, len(campaigns))
	for i, c := range campaigns {
		inputs[i] = translationInput{
			Index:       i,
			Name:        c.Name,
			Blurb:       c.Blurb,
			CreatorName: c.CreatorName,
		}
	}

	inputJSON, _ := json.MarshalIndent(inputs, "", "  ")

	prompt := fmt.Sprintf(`你是一个专业的英译中翻译助手，专门翻译 Kickstarter 众筹项目信息。

请将以下 JSON 数组中的每个项目翻译成中文：
- name: 项目名称（保持简洁有力，符合中文众筹项目命名习惯）
- blurb: 项目简介（翻译要自然流畅，保留原意）
- creator_name: 创作者名称（人名可音译，公司名保留原文或使用常见中文译名）

**重要规则：**
1. 专业术语保持一致性（如 "3D printing" → "3D打印"）
2. 品牌名称保留英文或使用官方中文名
3. 如果 blurb 或 creator_name 为空，输出也为空
4. 输出必须是有效的 JSON 数组格式
5. 保留原文的语气和风格

输入 JSON：
%s

输出格式示例：
[
  {
    "index": 0,
    "name_zh": "迷你电动往复式细节打磨机",
    "blurb_zh": "专业级便携打磨工具，适用于木工、金属加工和精细雕刻",
    "creator_zh": "HOZO Design 公司"
  }
]

请直接输出 JSON 数组，不要有其他文字：`, string(inputJSON))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return fmt.Errorf("vertex ai generate content: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return fmt.Errorf("empty response from vertex ai")
	}

	// Extract JSON from response
	responseText := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])

	// Clean markdown code blocks if present
	responseText = strings.TrimSpace(responseText)
	responseText = strings.TrimPrefix(responseText, "```json")
	responseText = strings.TrimPrefix(responseText, "```")
	responseText = strings.TrimSuffix(responseText, "```")
	responseText = strings.TrimSpace(responseText)

	// Parse response
	var outputs []translationOutput
	if err := json.Unmarshal([]byte(responseText), &outputs); err != nil {
		log.Printf("Translator: failed to parse response, raw text:\n%s", responseText)
		return fmt.Errorf("parse translation response: %w", err)
	}

	// Apply translations back to campaigns
	for _, out := range outputs {
		if out.Index >= 0 && out.Index < len(campaigns) {
			campaigns[out.Index].NameZh = out.NameZh
			campaigns[out.Index].BlurbZh = out.BlurbZh
			campaigns[out.Index].CreatorNameZh = out.CreatorZh
		}
	}

	log.Printf("Translator: translated %d campaigns via Vertex AI", len(outputs))
	return nil
}

// Close releases resources held by the translator.
func (t *TranslatorService) Close() error {
	if t.client != nil {
		return t.client.Close()
	}
	return nil
}
