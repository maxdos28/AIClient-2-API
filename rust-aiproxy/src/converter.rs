use crate::error::{Error, Result};
use crate::models::*;
use uuid::Uuid;

pub struct Converter;

impl Converter {
    pub fn new() -> Self {
        Self
    }

    /// Convert OpenAI request to Claude request
    pub fn openai_to_claude(&self, req: &OpenAIRequest) -> Result<ClaudeRequest> {
        let mut system = None;
        let mut messages = Vec::new();

        for msg in &req.messages {
            if msg.role == "system" {
                system = Some(self.extract_text_content(&msg.content));
                continue;
            }

            let content = self.convert_openai_content_to_claude(&msg.content)?;
            messages.push(ClaudeMessage {
                role: msg.role.clone(),
                content,
            });
        }

        let max_tokens = req.max_tokens.unwrap_or(8192);

        Ok(ClaudeRequest {
            model: req.model.clone(),
            messages,
            max_tokens,
            system,
            temperature: req.temperature,
            top_p: req.top_p,
            stream: req.stream,
            tools: req.tools.as_ref().map(|tools| {
                tools
                    .iter()
                    .map(|t| ClaudeTool {
                        name: t.function.name.clone(),
                        description: t.function.description.clone().unwrap_or_default(),
                        input_schema: t.function.parameters.clone().unwrap_or(serde_json::json!({})),
                    })
                    .collect()
            }),
        })
    }

    /// Convert Claude request to OpenAI request
    pub fn claude_to_openai(&self, req: &ClaudeRequest) -> Result<OpenAIRequest> {
        let mut messages = Vec::new();

        // Add system message if present
        if let Some(system) = &req.system {
            messages.push(OpenAIMessage {
                role: "system".to_string(),
                content: serde_json::json!(system),
                name: None,
                tool_calls: None,
            });
        }

        // Convert Claude messages
        for msg in &req.messages {
            let content = self.convert_claude_content_to_openai(&msg.content)?;
            messages.push(OpenAIMessage {
                role: msg.role.clone(),
                content,
                name: None,
                tool_calls: None,
            });
        }

        Ok(OpenAIRequest {
            model: req.model.clone(),
            messages,
            max_tokens: Some(req.max_tokens),
            temperature: req.temperature,
            top_p: req.top_p,
            stream: req.stream,
            tools: req.tools.as_ref().map(|tools| {
                tools
                    .iter()
                    .map(|t| Tool {
                        tool_type: "function".to_string(),
                        function: ToolFunction {
                            name: t.name.clone(),
                            description: Some(t.description.clone()),
                            parameters: Some(t.input_schema.clone()),
                        },
                    })
                    .collect()
            }),
        })
    }

    /// Convert Claude request to Gemini request
    pub fn claude_to_gemini(&self, req: &ClaudeRequest) -> Result<GeminiRequest> {
        let system_instruction = req.system.as_ref().map(|s| GeminiSystemInstruction {
            parts: vec![GeminiPart {
                text: Some(s.clone()),
                inline_data: None,
                function_call: None,
                function_response: None,
            }],
        });

        let contents = req
            .messages
            .iter()
            .map(|msg| {
                let role = if msg.role == "assistant" {
                    "model"
                } else {
                    &msg.role
                };

                let parts = msg
                    .content
                    .iter()
                    .filter_map(|c| match c {
                        ClaudeContent::Text { text } => Some(GeminiPart {
                            text: Some(text.clone()),
                            inline_data: None,
                            function_call: None,
                            function_response: None,
                        }),
                        ClaudeContent::Image { source } => Some(GeminiPart {
                            text: None,
                            inline_data: Some(GeminiInlineData {
                                mime_type: source.media_type.clone(),
                                data: source.data.clone(),
                            }),
                            function_call: None,
                            function_response: None,
                        }),
                        ClaudeContent::ToolUse { name, input, .. } => {
                            let args = input
                                .as_object()
                                .map(|obj| {
                                    obj.iter()
                                        .map(|(k, v)| (k.clone(), v.clone()))
                                        .collect()
                                })
                                .unwrap_or_default();

                            Some(GeminiPart {
                                text: None,
                                inline_data: None,
                                function_call: Some(GeminiFunctionCall {
                                    name: name.clone(),
                                    args,
                                }),
                                function_response: None,
                            })
                        }
                        _ => None,
                    })
                    .collect();

                GeminiContent {
                    role: role.to_string(),
                    parts,
                }
            })
            .collect();

        Ok(GeminiRequest {
            contents,
            system_instruction,
            generation_config: Some(GeminiGenerationConfig {
                temperature: req.temperature,
                top_p: req.top_p,
                max_output_tokens: Some(req.max_tokens),
            }),
            tools: req.tools.as_ref().map(|tools| {
                vec![GeminiTool {
                    function_declarations: tools
                        .iter()
                        .map(|t| GeminiFunctionDeclaration {
                            name: t.name.clone(),
                            description: t.description.clone(),
                            parameters: t.input_schema.clone(),
                        })
                        .collect(),
                }]
            }),
        })
    }

    /// Convert OpenAI response to Claude response
    pub fn openai_response_to_claude(&self, resp: &OpenAIResponse, model: &str) -> Result<ClaudeResponse> {
        if resp.choices.is_empty() {
            return Err(Error::ConversionError("No choices in response".to_string()));
        }

        let choice = &resp.choices[0];
        let message = choice.message.as_ref()
            .ok_or_else(|| Error::ConversionError("No message in choice".to_string()))?;

        let content = vec![ClaudeContent::Text {
            text: self.extract_text_content(&message.content),
        }];

        let stop_reason = match choice.finish_reason.as_deref() {
            Some("stop") => "end_turn",
            Some("length") => "max_tokens",
            Some("tool_calls") => "tool_use",
            _ => "end_turn",
        };

        Ok(ClaudeResponse {
            id: resp.id.clone(),
            response_type: "message".to_string(),
            role: "assistant".to_string(),
            content,
            model: model.to_string(),
            stop_reason: stop_reason.to_string(),
            usage: resp.usage.as_ref().map(|u| ClaudeUsage {
                input_tokens: u.prompt_tokens,
                output_tokens: u.completion_tokens,
            }),
        })
    }

    /// Convert Gemini response to Claude response  
    pub fn gemini_response_to_claude(&self, resp: &GeminiResponse, model: &str) -> Result<ClaudeResponse> {
        if resp.candidates.is_empty() {
            return Err(Error::ConversionError("No candidates in response".to_string()));
        }

        let candidate = &resp.candidates[0];
        let content: Vec<ClaudeContent> = candidate
            .content
            .parts
            .iter()
            .filter_map(|part| {
                part.text.as_ref().map(|text| ClaudeContent::Text {
                    text: text.clone(),
                })
            })
            .collect();

        let stop_reason = match candidate.finish_reason.as_str() {
            "STOP" => "end_turn",
            "MAX_TOKENS" => "max_tokens",
            _ => "end_turn",
        };

        Ok(ClaudeResponse {
            id: Uuid::new_v4().to_string(),
            response_type: "message".to_string(),
            role: "assistant".to_string(),
            content,
            model: model.to_string(),
            stop_reason: stop_reason.to_string(),
            usage: resp.usage_metadata.as_ref().map(|u| ClaudeUsage {
                input_tokens: u.prompt_token_count,
                output_tokens: u.candidates_token_count,
            }),
        })
    }

    // Helper methods
    fn extract_text_content(&self, content: &serde_json::Value) -> String {
        match content {
            serde_json::Value::String(s) => s.clone(),
            serde_json::Value::Array(arr) => arr
                .iter()
                .filter_map(|v| {
                    if let Some(obj) = v.as_object() {
                        if obj.get("type")?.as_str()? == "text" {
                            return obj.get("text")?.as_str().map(|s| s.to_string());
                        }
                    }
                    None
                })
                .collect::<Vec<_>>()
                .join(" "),
            _ => String::new(),
        }
    }

    fn convert_openai_content_to_claude(&self, content: &serde_json::Value) -> Result<Vec<ClaudeContent>> {
        let text = self.extract_text_content(content);
        Ok(vec![ClaudeContent::Text { text }])
    }

    fn convert_claude_content_to_openai(&self, content: &[ClaudeContent]) -> Result<serde_json::Value> {
        let texts: Vec<String> = content
            .iter()
            .filter_map(|c| match c {
                ClaudeContent::Text { text } => Some(text.clone()),
                _ => None,
            })
            .collect();

        Ok(serde_json::json!(texts.join(" ")))
    }
}

impl Default for Converter {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_openai_to_claude_conversion() {
        let converter = Converter::new();
        
        let openai_req = OpenAIRequest {
            model: "gpt-3.5-turbo".to_string(),
            messages: vec![
                OpenAIMessage {
                    role: "system".to_string(),
                    content: serde_json::json!("You are helpful"),
                    name: None,
                    tool_calls: None,
                },
                OpenAIMessage {
                    role: "user".to_string(),
                    content: serde_json::json!("Hello"),
                    name: None,
                    tool_calls: None,
                },
            ],
            max_tokens: Some(100),
            temperature: Some(0.7),
            top_p: None,
            stream: Some(false),
            tools: None,
        };

        let result = converter.openai_to_claude(&openai_req);
        assert!(result.is_ok());
        
        let claude_req = result.unwrap();
        assert_eq!(claude_req.system, Some("You are helpful".to_string()));
        assert_eq!(claude_req.messages.len(), 1);
        assert_eq!(claude_req.max_tokens, 100);
    }

    #[test]
    fn test_claude_to_gemini_conversion() {
        let converter = Converter::new();
        
        let claude_req = ClaudeRequest {
            model: "claude-3-opus".to_string(),
            messages: vec![ClaudeMessage {
                role: "user".to_string(),
                content: vec![ClaudeContent::Text {
                    text: "Hello".to_string(),
                }],
            }],
            max_tokens: 100,
            system: Some("You are helpful".to_string()),
            temperature: Some(0.7),
            top_p: None,
            stream: Some(false),
            tools: None,
        };

        let result = converter.claude_to_gemini(&claude_req);
        assert!(result.is_ok());
        
        let gemini_req = result.unwrap();
        assert!(gemini_req.system_instruction.is_some());
        assert_eq!(gemini_req.contents.len(), 1);
    }
}
