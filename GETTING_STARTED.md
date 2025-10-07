# 🚀 快速开始 - 五分钟上手指南

选择你喜欢的语言，五分钟内启动 AI Proxy！

---

## 1️⃣ Node.js 版本 (最快)

```bash
# 进入目录
cd src

# 安装依赖
npm install

# 设置 API Key
export OPENAI_API_KEY=sk-your-api-key-here

# 启动服务
npm start

# 测试
curl http://localhost:3000/health
```

**时间: ~1 分钟** ⚡

---

## 2️⃣ Go 版本 (推荐生产)

```bash
# 进入目录
cd go-aiproxy

# 直接运行
go run cmd/server/main.go --openai-api-key sk-your-key

# 或编译后运行
go build -o aiproxy cmd/server/main.go
./aiproxy --openai-api-key sk-your-key

# 运行测试
go test -v ./...
```

**时间: ~2 分钟** ⚡

---

## 3️⃣ Rust 版本 (最高性能)

```bash
# 进入目录
cd rust-aiproxy

# 运行 (开发模式)
cargo run -- --openai-api-key sk-your-key

# 或编译发布版本
cargo build --release
./target/release/aiproxy --openai-api-key sk-your-key

# 运行测试
cargo test
```

**时间: ~5 分钟** (首次编译)

---

## 4️⃣ Java 版本 (企业级)

```bash
# 进入目录
cd java-aiproxy

# 设置环境变量
export OPENAI_API_KEY=sk-your-key

# 使用 Maven 运行
mvn spring-boot:run

# 或打包后运行
mvn package
java -jar target/aiproxy-1.0.0.jar
```

**时间: ~3 分钟**

---

## 5️⃣ Scala 版本 (函数式)

```bash
# 进入目录
cd scala-aiproxy

# 设置环境变量
export OPENAI_API_KEY=sk-your-key

# 使用 sbt 运行
sbt run

# 或打包后运行
sbt assembly
java -jar target/scala-3.3.1/scala-aiproxy-assembly-1.0.0.jar
```

**时间: ~5 分钟** (首次编译)

---

## 📝 测试 API

所有版本默认在 `http://localhost:3000` 运行

### 健康检查

```bash
curl http://localhost:3000/health
```

### 聊天对话

```bash
curl -X POST http://localhost:3000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [
      {"role": "user", "content": "Hello, how are you?"}
    ]
  }'
```

### 获取模型列表

```bash
curl http://localhost:3000/v1/models
```

---

## 🐳 使用 Docker

所有版本都支持 Docker：

```bash
# 以 Go 版本为例
cd go-aiproxy
docker build -t aiproxy:go .
docker run -p 3000:3000 -e OPENAI_API_KEY=sk-xxx aiproxy:go

# 其他版本类似
```

---

## 🔧 配置 API Keys

### 方式 1: 环境变量 (推荐)

```bash
export OPENAI_API_KEY=sk-your-openai-key
export CLAUDE_API_KEY=claude-your-claude-key
export GEMINI_API_KEY=gemini-your-gemini-key
```

### 方式 2: 命令行参数 (Go/Rust)

```bash
# Go
./aiproxy --openai-api-key sk-xxx --port 3000

# Rust
./aiproxy --openai-api-key sk-xxx --port 3000
```

### 方式 3: 配置文件 (Java/Scala)

编辑 `application.yml` 或 `application.conf`

---

## 🎯 快速决策

**没时间？看这里！**

| 你的情况 | 推荐版本 | 原因 |
|---------|---------|------|
| 我是前端开发 | Node.js | 熟悉的技术栈 |
| 我要快速测试 | Node.js | 最快启动 |
| 我要生产部署 | Go | 简单可靠 |
| 我追求性能 | Rust | 最高效率 |
| 我是 Java 开发 | Java | 熟悉的框架 |
| 我喜欢函数式 | Scala | 优雅简洁 |

---

## ⚡ 最快启动方式

```bash
# 1. 克隆项目 (如果还没有)
git clone <repository>
cd aiproxy

# 2. 选择版本并启动
cd src && npm install && npm start

# 就这么简单！
```

---

## 🔍 遇到问题？

### Node.js
- 确保安装了 Node.js 20+
- 检查 npm install 是否成功
- 查看 package.json 依赖

### Go
- 确保安装了 Go 1.21+
- 运行 go mod tidy
- 检查 go.sum

### Rust
- 确保安装了 Rust 1.75+
- 运行 cargo clean 后重试
- 检查 Cargo.lock

### Java
- 确保安装了 JDK 21+
- 运行 mvn clean install
- 检查 Maven 配置

### Scala
- 确保安装了 JDK 21+
- 确保安装了 sbt
- 运行 sbt clean compile

---

## 📚 下一步

1. ✅ 启动服务
2. 📖 阅读 API 文档
3. 🧪 运行测试
4. 🚀 部署到生产

---

**💡 提示**: 所有版本的 API 接口完全相同，可以随时切换！

**🎉 开始你的 AI 代理之旅吧！**
