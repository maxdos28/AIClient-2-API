# AI Proxy - Scala Edition 🔴

A functional, type-safe AI API proxy built with Scala 3, Akka HTTP, and modern functional programming principles.

## Features

- ✅ **Scala 3**: Latest language features (enums, given/using, extension methods)
- ✅ **Functional Programming**: Immutable data, pure functions, composability
- ✅ **Type Safety**: Strong static typing with ADTs
- ✅ **Akka HTTP**: Reactive, non-blocking HTTP server
- ✅ **Multi-Provider**: OpenAI, Claude, Gemini support
- ✅ **Protocol Conversion**: Automatic format conversion
- ✅ **Concurrency**: Actor-based concurrency model
- ✅ **JVM Interop**: Full Java interoperability

## Requirements

- JDK 21+
- Scala 3.3.1
- sbt 1.9.7+

## Quick Start

### Build

```bash
cd scala-aiproxy

# Compile
sbt compile

# Run
sbt run

# Create fat JAR
sbt assembly
```

### Run

```bash
# With sbt
export OPENAI_API_KEY=sk-xxx
sbt run

# With assembled JAR
export OPENAI_API_KEY=sk-xxx
java -jar target/scala-3.3.1/scala-aiproxy-assembly-1.0.0.jar
```

### Development

```bash
# Interactive mode
sbt

# Compile continuously
~compile

# Run tests
test

# Run specific test
testOnly com.aiproxy.converter.ProtocolConverterSpec
```

## API Usage

### Health Check

```bash
curl http://localhost:3000/health
```

Response:
```json
{
  "status": "ok",
  "version": "1.0.0"
}
```

### Chat Completion

```bash
curl -X POST http://localhost:3000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [
      {"role": "user", "content": "Hello!"}
    ]
  }'
```

### List Models

```bash
curl http://localhost:3000/v1/models
```

## Configuration

Configuration is done via `application.conf`:

```hocon
server {
  host = "0.0.0.0"
  port = 3000
}

openai {
  api-key = ${?OPENAI_API_KEY}
  base-url = "https://api.openai.com/v1"
}
```

Environment variables override config values.

## Architecture

```
┌──────────────────┐
│   Scala 3 FP     │
│  + Akka HTTP     │
└────────┬─────────┘
         │
    ┌────┴────┐
    │ Routes  │
    └────┬────┘
         │
    ┌────┴──────────────┐
    │  Controllers      │
    │  - chatCompletions│
    │  - listModels     │
    └────┬──────────────┘
         │
    ┌────┴──────────────┐
    │  Converter        │
    │  (Pure Functions) │
    └────┬──────────────┘
         │
    ┌────┴──────────────┐
    │  Providers        │
    │  (Trait-based)    │
    └───────────────────┘
```

## Project Structure

```
scala-aiproxy/
├── build.sbt
├── project/
│   ├── build.properties
│   └── plugins.sbt
├── src/
│   ├── main/
│   │   ├── scala/com/aiproxy/
│   │   │   ├── Main.scala
│   │   │   ├── model/
│   │   │   │   ├── Common.scala
│   │   │   │   ├── OpenAI.scala
│   │   │   │   ├── Claude.scala
│   │   │   │   └── Gemini.scala
│   │   │   ├── converter/
│   │   │   │   └── ProtocolConverter.scala
│   │   │   ├── provider/
│   │   │   │   ├── Provider.scala
│   │   │   │   └── OpenAIProvider.scala
│   │   │   └── controller/
│   │   │       └── Routes.scala
│   │   └── resources/
│   │       ├── application.conf
│   │       └── logback.xml
│   └── test/
│       └── scala/com/aiproxy/
│           └── converter/
│               └── ProtocolConverterSpec.scala
└── README.md
```

## Scala 3 Features

### Enums

```scala
enum Provider:
  case OpenAI, Claude, Gemini

enum Protocol:
  case OpenAI, Claude, Gemini
```

### Given/Using (Context Parameters)

```scala
class OpenAIProvider(apiKey: String)(using ec: ExecutionContext)

def createRoute(using system: ActorSystem[?])
```

### Extension Methods

```scala
extension (req: OpenAI.Request)
  def toClaude: Claude.Request = ???
```

### Pattern Matching

```scala
provider.protocol match
  case Protocol.OpenAI => request
  case Protocol.Claude => converter.openAIToClaude(request)
  case Protocol.Gemini => converter.claudeToGemini(request)
```

## Functional Programming Principles

### Immutability

All data structures are immutable case classes:

```scala
case class Request(
  model: String,
  messages: List[Message],
  maxTokens: Option[Int]
)
```

### Pure Functions

All conversions are pure functions:

```scala
def openAIToClaude(req: OpenAI.Request): Claude.Request =
  // No side effects, deterministic
```

### Composability

```scala
val result = for
  claudeReq <- openAIToClaude(openAIReq)
  geminiReq <- claudeToGemini(claudeReq)
yield geminiReq
```

### ADTs (Algebraic Data Types)

```scala
sealed trait Content
case class TextContent(text: String) extends Content
case class ImageContent(source: ImageSource) extends Content
```

## Testing

```bash
# Run all tests
sbt test

# Run with coverage
sbt coverage test coverageReport

# Run specific test
sbt "testOnly com.aiproxy.converter.ProtocolConverterSpec"
```

## Performance

- **Startup Time**: ~2-3 seconds (JVM warmup)
- **Memory Usage**: ~200-300 MB
- **Throughput**: High (Akka Streams)
- **Concurrency**: Actor-based, highly scalable

## Technologies

- **Scala 3.3.1**: Modern functional programming
- **Akka HTTP 10.5.3**: Reactive HTTP server
- **Akka Streams**: Backpressure-aware streaming
- **Spray JSON**: JSON serialization
- **STTP Client 3**: HTTP client
- **ScalaTest 3.2.17**: Testing framework
- **Logback**: Logging

## Advantages

### Type Safety
- Compile-time guarantees
- ADTs for domain modeling
- Pattern matching exhaustiveness

### Functional Style
- Immutable data structures
- Pure functions
- Composable operations
- No null references (Option type)

### JVM Benefits
- Mature ecosystem
- Excellent tooling
- Java interoperability
- Production-proven runtime

### Akka Ecosystem
- Actor model for concurrency
- Backpressure handling
- Clustering support
- Battle-tested in production

## Development Tips

### REPL

```bash
# Start Scala REPL with project classes
sbt console

scala> import com.aiproxy.model.*
scala> val req = OpenAI.Request(...)
```

### Hot Reload

```bash
# In sbt shell
~reStart
```

### Code Formatting

```bash
# Format code (if using scalafmt)
sbt scalafmt
```

## Comparison with Java

| Feature | Java | Scala |
|---------|------|-------|
| **Verbosity** | High | Low |
| **Type Inference** | Limited | Excellent |
| **Pattern Matching** | Basic (21+) | Advanced |
| **Immutability** | Manual | Default |
| **Null Safety** | @Nullable | Option[T] |
| **Collections** | Mutable by default | Immutable by default |
| **Functional** | Streams | First-class |

### Code Comparison

**Java:**
```java
public class OpenAIRequest {
    private String model;
    private List<Message> messages;
    
    // 20+ lines of boilerplate (getters, setters, equals, hashCode)
}
```

**Scala:**
```scala
case class Request(model: String, messages: List[Message])
// That's it! Getters, equals, hashCode, toString auto-generated
```

## Docker

```bash
# Build
docker build -t aiproxy:scala .

# Run
docker run -p 3000:3000 \
  -e OPENAI_API_KEY=sk-xxx \
  aiproxy:scala
```

## Deployment

### Fat JAR

```bash
sbt assembly
java -jar target/scala-3.3.1/scala-aiproxy-assembly-1.0.0.jar
```

### Native Image (GraalVM)

```bash
# Requires GraalVM
sbt nativeImage
./target/native-image/scala-aiproxy
```

## Why Scala?

### When to Use Scala

✅ Need strong type safety  
✅ Prefer functional programming  
✅ Want JVM ecosystem  
✅ Complex domain logic  
✅ Team experienced in FP  

### When NOT to Use Scala

❌ Team unfamiliar with FP  
❌ Simple CRUD applications  
❌ Rapid prototyping  
❌ Startup time critical  

## Common Patterns

### Error Handling

```scala
import scala.util.{Try, Success, Failure}

Try {
  provider.chatCompletion(request)
} match
  case Success(response) => complete(response)
  case Failure(ex) => complete(StatusCodes.InternalServerError)
```

### Option Chaining

```scala
val token = config.getString("api-key")
  .filter(_.nonEmpty)
  .orElse(sys.env.get("API_KEY"))
  .getOrElse("default")
```

### For Comprehensions

```scala
val result = for
  claudeReq <- Try(openAIToClaude(req))
  response <- provider.chatCompletion(claudeReq)
yield response
```

## License

MIT

## Author

AI Proxy Team

---

**💡 Pro Tip**: Scala combines the best of functional and object-oriented programming with excellent Java interoperability!
