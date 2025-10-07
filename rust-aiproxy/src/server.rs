use crate::cache::Cache;
use crate::converter::Converter;
use crate::error::{Error, Result};
use crate::models::*;
use crate::providers::Provider;
use axum::{
    extract::{Json, State},
    http::StatusCode,
    response::{IntoResponse, Response},
    routing::{get, post},
    Router,
};
use std::collections::HashMap;
use std::sync::Arc;
use tower_http::cors::CorsLayer;
use tracing::info;

pub struct AppState {
    pub converter: Converter,
    pub providers: HashMap<String, Arc<dyn Provider>>,
    pub cache: Cache,
}

impl AppState {
    pub fn new() -> Self {
        Self {
            converter: Converter::new(),
            providers: HashMap::new(),
            cache: Cache::default(),
        }
    }

    pub fn add_provider(&mut self, name: String, provider: Arc<dyn Provider>) {
        self.providers.insert(name, provider);
    }
}

pub fn create_router(state: Arc<AppState>) -> Router {
    Router::new()
        .route("/health", get(health_check))
        .route("/v1/chat/completions", post(chat_completions))
        .route("/v1/models", get(list_models))
        .layer(CorsLayer::permissive())
        .with_state(state)
}

async fn health_check() -> impl IntoResponse {
    Json(serde_json::json!({
        "status": "ok",
        "version": env!("CARGO_PKG_VERSION"),
    }))
}

async fn chat_completions(
    State(state): State<Arc<AppState>>,
    Json(request): Json<OpenAIRequest>,
) -> Result<Json<serde_json::Value>> {
    info!("Received chat completion request for model: {}", request.model);

    // Determine provider (simplified - just use first provider)
    let provider = state
        .providers
        .values()
        .next()
        .ok_or_else(|| Error::Other("No providers configured".to_string()))?;

    // Convert request based on provider protocol
    let provider_request = match provider.get_protocol() {
        Protocol::OpenAI => serde_json::to_value(&request)?,
        Protocol::Claude => {
            let claude_req = state.converter.openai_to_claude(&request)?;
            serde_json::to_value(&claude_req)?
        }
        Protocol::Gemini => {
            let claude_req = state.converter.openai_to_claude(&request)?;
            let gemini_req = state.converter.claude_to_gemini(&claude_req)?;
            serde_json::to_value(&gemini_req)?
        }
    };

    // Call provider
    let response = provider.chat_completion(provider_request).await?;

    // Convert response back to OpenAI format if needed
    let openai_response = match provider.get_protocol() {
        Protocol::OpenAI => response,
        Protocol::Claude => {
            let claude_resp: ClaudeResponse = serde_json::from_value(response)?;
            let openai_resp = OpenAIResponse {
                id: claude_resp.id,
                object: "chat.completion".to_string(),
                created: chrono::Utc::now().timestamp(),
                model: request.model.clone(),
                choices: vec![OpenAIChoice {
                    index: 0,
                    message: Some(OpenAIMessage {
                        role: "assistant".to_string(),
                        content: serde_json::json!(
                            claude_resp.content.iter()
                                .filter_map(|c| match c {
                                    ClaudeContent::Text { text } => Some(text.clone()),
                                    _ => None,
                                })
                                .collect::<Vec<_>>()
                                .join(" ")
                        ),
                        name: None,
                        tool_calls: None,
                    }),
                    delta: None,
                    finish_reason: Some("stop".to_string()),
                }],
                usage: claude_resp.usage.map(|u| Usage {
                    prompt_tokens: u.input_tokens,
                    completion_tokens: u.output_tokens,
                    total_tokens: u.input_tokens + u.output_tokens,
                }),
            };
            serde_json::to_value(&openai_resp)?
        }
        Protocol::Gemini => {
            let gemini_resp: GeminiResponse = serde_json::from_value(response)?;
            let claude_resp = state.converter.gemini_response_to_claude(&gemini_resp, &request.model)?;
            let openai_resp = OpenAIResponse {
                id: claude_resp.id,
                object: "chat.completion".to_string(),
                created: chrono::Utc::now().timestamp(),
                model: request.model.clone(),
                choices: vec![OpenAIChoice {
                    index: 0,
                    message: Some(OpenAIMessage {
                        role: "assistant".to_string(),
                        content: serde_json::json!(
                            claude_resp.content.iter()
                                .filter_map(|c| match c {
                                    ClaudeContent::Text { text } => Some(text.clone()),
                                    _ => None,
                                })
                                .collect::<Vec<_>>()
                                .join(" ")
                        ),
                        name: None,
                        tool_calls: None,
                    }),
                    delta: None,
                    finish_reason: Some("stop".to_string()),
                }],
                usage: claude_resp.usage.map(|u| Usage {
                    prompt_tokens: u.input_tokens,
                    completion_tokens: u.output_tokens,
                    total_tokens: u.input_tokens + u.output_tokens,
                }),
            };
            serde_json::to_value(&openai_resp)?
        }
    };

    Ok(Json(openai_response))
}

async fn list_models(State(_state): State<Arc<AppState>>) -> impl IntoResponse {
    let models = ModelList {
        object: "list".to_string(),
        data: vec![
            ModelInfo {
                id: "gpt-3.5-turbo".to_string(),
                object: "model".to_string(),
                created: chrono::Utc::now().timestamp(),
                owned_by: "openai".to_string(),
            },
            ModelInfo {
                id: "claude-3-opus-20240229".to_string(),
                object: "model".to_string(),
                created: chrono::Utc::now().timestamp(),
                owned_by: "anthropic".to_string(),
            },
            ModelInfo {
                id: "gemini-pro".to_string(),
                object: "model".to_string(),
                created: chrono::Utc::now().timestamp(),
                owned_by: "google".to_string(),
            },
        ],
    };

    Json(models)
}

// Error handling
impl IntoResponse for Error {
    fn into_response(self) -> Response {
        let (status, message) = match self {
            Error::InvalidRequest(msg) => (StatusCode::BAD_REQUEST, msg),
            Error::ProviderError(msg) => (StatusCode::BAD_GATEWAY, msg),
            Error::ConversionError(msg) => (StatusCode::INTERNAL_SERVER_ERROR, msg),
            _ => (StatusCode::INTERNAL_SERVER_ERROR, self.to_string()),
        };

        let body = Json(serde_json::json!({
            "error": {
                "message": message,
                "type": "api_error",
            }
        }));

        (status, body).into_response()
    }
}
