#!/bin/bash
# Quick start script for Scala AI Proxy

set -e

echo "🔴 Scala AI Proxy - Quick Start"
echo ""

# Check Java
if ! command -v java &> /dev/null; then
    echo "❌ Java not found. Please install JDK 21+"
    exit 1
fi

echo "✅ Java version:"
java -version 2>&1 | head -1

# Check sbt
if ! command -v sbt &> /dev/null; then
    echo ""
    echo "⚠️  sbt not found. Installing sbt is recommended."
    echo ""
    echo "Install sbt:"
    echo "  macOS:   brew install sbt"
    echo "  Linux:   https://www.scala-sbt.org/download.html"
    echo "  Windows: https://www.scala-sbt.org/download.html"
    echo ""
    echo "Or download from: https://github.com/sbt/sbt/releases"
    exit 1
fi

echo ""
echo "✅ sbt version:"
sbt --version | head -1

echo ""
echo "📋 Build & Run options:"
echo ""
echo "1. Development mode:"
echo "   export OPENAI_API_KEY=sk-xxx"
echo "   sbt run"
echo ""
echo "2. Build fat JAR:"
echo "   sbt assembly"
echo "   java -jar target/scala-3.3.1/scala-aiproxy-assembly-1.0.0.jar"
echo ""
echo "3. Interactive sbt shell:"
echo "   sbt"
echo "   > compile"
echo "   > test"
echo "   > run"
echo ""
echo "4. Docker:"
echo "   docker build -t aiproxy:scala ."
echo "   docker run -p 3000:3000 -e OPENAI_API_KEY=sk-xxx aiproxy:scala"
echo ""
echo "🔧 Environment variables:"
echo "   OPENAI_API_KEY=sk-xxx"
echo "   SERVER_PORT=3000"
echo ""
echo "📚 Documentation: See README.md"
