package responses

import (
	"testing"

	"github.com/tidwall/gjson"
)

func TestConvertOpenAIResponsesRequestToOpenAIChatCompletions_PreservesObjectToolChoice(t *testing.T) {
	inputJSON := []byte(`{
		"model": "gpt-5.4-mini",
		"input": "generate an image",
		"tool_choice": {
			"type": "allowed_tools",
			"tools": [
				{"type": "image_generation"}
			]
		}
	}`)

	output := ConvertOpenAIResponsesRequestToOpenAIChatCompletions("gpt-5.4-mini", inputJSON, false)

	if got := gjson.GetBytes(output, "tool_choice.type").String(); got != "allowed_tools" {
		t.Fatalf("tool_choice.type = %q, want %q: %s", got, "allowed_tools", string(output))
	}
	if got := gjson.GetBytes(output, "tool_choice.tools.0.type").String(); got != "image_generation" {
		t.Fatalf("tool_choice.tools.0.type = %q, want %q: %s", got, "image_generation", string(output))
	}
}
