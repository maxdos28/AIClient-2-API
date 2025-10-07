#!/bin/bash
# Quick start script

set -e

echo "☕ Java AI Proxy - Quick Start"
echo ""

# Check Java version
if ! command -v java &> /dev/null; then
    echo "❌ Java not found. Please install Java 21+"
    exit 1
fi

echo "✅ Java version:"
java -version 2>&1 | head -1

echo ""
echo "📋 Build options:"
echo ""
echo "1. Using Maven:"
echo "   mvn clean package"
echo "   java -jar target/aiproxy-1.0.0.jar"
echo ""
echo "2. Using Docker:"
echo "   docker build -t aiproxy:java ."
echo "   docker run -p 3000:3000 -e OPENAI_API_KEY=sk-xxx aiproxy:java"
echo ""
echo "3. Using IDE:"
echo "   - Open in IntelliJ IDEA / Eclipse / VS Code"
echo "   - Run AiProxyApplication.java"
echo ""
echo "🔧 Environment variables:"
echo "   OPENAI_API_KEY=sk-xxx"
echo "   CLAUDE_API_KEY=claude-xxx"
echo "   GEMINI_API_KEY=gemini-xxx"
echo ""
echo "📚 Documentation: See README.md"
