use crate::error::{Error, Result};
use crate::models::*;
use async_trait::async_trait;
use reqwest::Client;

#[async_trait]
pub trait Provider: Send + Sync {
    async fn chat_completion(&self, request: serde_json::Value) -> Result<serde_json::Value>;
    fn get_protocol(&self) -> Protocol;
    fn get_name(&self) -> &str;
}

pub struct OpenAIProvider {
    client: Client,
    api_key: String,
    base_url: String,
}

impl OpenAIProvider {
    pub fn new(api_key: String, base_url: Option<String>) -> Self {
        Self {
            client: Client::new(),
            api_key,
            base_url: base_url.unwrap_or_else(|| "https://api.openai.com/v1".to_string()),
        }
    }
}

#[async_trait]
impl Provider for OpenAIProvider {
    async fn chat_completion(&self, request: serde_json::Value) -> Result<serde_json::Value> {
        let url = format!("{}/chat/completions", self.base_url);
        
        let response = self
            .client
            .post(&url)
            .header("Authorization", format!("Bearer {}", self.api_key))
            .header("Content-Type", "application/json")
            .json(&request)
            .send()
            .await?;

        if !response.status().is_success() {
            let error_text = response.text().await?;
            return Err(Error::ProviderError(format!("OpenAI API error: {}", error_text)));
        }

        let result = response.json().await?;
        Ok(result)
    }

    fn get_protocol(&self) -> Protocol {
        Protocol::OpenAI
    }

    fn get_name(&self) -> &str {
        "openai"
    }
}

pub struct ClaudeProvider {
    client: Client,
    api_key: String,
    base_url: String,
}

impl ClaudeProvider {
    pub fn new(api_key: String, base_url: Option<String>) -> Self {
        Self {
            client: Client::new(),
            api_key,
            base_url: base_url.unwrap_or_else(|| "https://api.anthropic.com".to_string()),
        }
    }
}

#[async_trait]
impl Provider for ClaudeProvider {
    async fn chat_completion(&self, request: serde_json::Value) -> Result<serde_json::Value> {
        let url = format!("{}/v1/messages", self.base_url);
        
        let response = self
            .client
            .post(&url)
            .header("x-api-key", &self.api_key)
            .header("anthropic-version", "2023-06-01")
            .header("Content-Type", "application/json")
            .json(&request)
            .send()
            .await?;

        if !response.status().is_success() {
            let error_text = response.text().await?;
            return Err(Error::ProviderError(format!("Claude API error: {}", error_text)));
        }

        let result = response.json().await?;
        Ok(result)
    }

    fn get_protocol(&self) -> Protocol {
        Protocol::Claude
    }

    fn get_name(&self) -> &str {
        "claude"
    }
}

pub struct GeminiProvider {
    client: Client,
    api_key: String,
    base_url: String,
}

impl GeminiProvider {
    pub fn new(api_key: String, base_url: Option<String>) -> Self {
        Self {
            client: Client::new(),
            api_key,
            base_url: base_url.unwrap_or_else(|| "https://generativelanguage.googleapis.com".to_string()),
        }
    }
}

#[async_trait]
impl Provider for GeminiProvider {
    async fn chat_completion(&self, request: serde_json::Value) -> Result<serde_json::Value> {
        let url = format!(
            "{}/v1beta/models/gemini-pro:generateContent?key={}",
            self.base_url, self.api_key
        );
        
        let response = self
            .client
            .post(&url)
            .header("Content-Type", "application/json")
            .json(&request)
            .send()
            .await?;

        if !response.status().is_success() {
            let error_text = response.text().await?;
            return Err(Error::ProviderError(format!("Gemini API error: {}", error_text)));
        }

        let result = response.json().await?;
        Ok(result)
    }

    fn get_protocol(&self) -> Protocol {
        Protocol::Gemini
    }

    fn get_name(&self) -> &str {
        "gemini"
    }
}
