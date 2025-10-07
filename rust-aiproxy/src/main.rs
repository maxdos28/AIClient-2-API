use aiproxy::providers::{ClaudeProvider, GeminiProvider, OpenAIProvider};
use aiproxy::server::{create_router, AppState};
use clap::Parser;
use std::sync::Arc;
use tracing::info;
use tracing_subscriber;

#[derive(Parser, Debug)]
#[command(name = "aiproxy")]
#[command(about = "AI API proxy supporting multiple providers", long_about = None)]
struct Args {
    /// Server host
    #[arg(short = 'H', long, default_value = "0.0.0.0")]
    host: String,

    /// Server port
    #[arg(short, long, default_value = "3000")]
    port: u16,

    /// OpenAI API key
    #[arg(long, env)]
    openai_api_key: Option<String>,

    /// Claude API key
    #[arg(long, env)]
    claude_api_key: Option<String>,

    /// Gemini API key
    #[arg(long, env)]
    gemini_api_key: Option<String>,

    /// Log level
    #[arg(long, default_value = "info")]
    log_level: String,
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    let args = Args::parse();

    // Initialize tracing
    tracing_subscriber::fmt()
        .with_env_filter(&args.log_level)
        .init();

    info!("🚀 Starting AI Proxy Server v{}", env!("CARGO_PKG_VERSION"));

    // Create app state
    let mut state = AppState::new();

    // Register providers
    if let Some(api_key) = args.openai_api_key {
        info!("✓ Registered OpenAI provider");
        state.add_provider(
            "openai".to_string(),
            Arc::new(OpenAIProvider::new(api_key, None)),
        );
    }

    if let Some(api_key) = args.claude_api_key {
        info!("✓ Registered Claude provider");
        state.add_provider(
            "claude".to_string(),
            Arc::new(ClaudeProvider::new(api_key, None)),
        );
    }

    if let Some(api_key) = args.gemini_api_key {
        info!("✓ Registered Gemini provider");
        state.add_provider(
            "gemini".to_string(),
            Arc::new(GeminiProvider::new(api_key, None)),
        );
    }

    if state.providers.is_empty() {
        eprintln!("⚠️  Warning: No providers configured. Please provide at least one API key.");
        eprintln!("   Use --openai-api-key, --claude-api-key, or --gemini-api-key");
    }

    let state = Arc::new(state);

    // Create router
    let app = create_router(state);

    // Start server
    let addr = format!("{}:{}", args.host, args.port);
    let listener = tokio::net::TcpListener::bind(&addr).await?;
    
    info!("✓ Server listening on http://{}", addr);
    info!("✓ Health check available at http://{}/health", addr);
    info!("✓ API endpoint: http://{}/v1/chat/completions", addr);

    axum::serve(listener, app).await?;

    Ok(())
}
