package com.aiproxy.converter

import com.aiproxy.model.{OpenAI, Claude, Gemini}
import scala.util.{Try, Success, Failure}

object ProtocolConverter:
  private val defaultMaxTokens = 8192

  /** Convert OpenAI request to Claude request */
  def openAIToClaude(req: OpenAI.Request): Claude.Request =
    val (systemMsg, otherMsgs) = req.messages.partition(_.role == "system")
    val system = systemMsg.headOption.map(_.content)
    
    val claudeMsgs = otherMsgs.map { msg =>
      Claude.Message(
        role = msg.role,
        content = List(Claude.TextContent(text = msg.content))
      )
    }

    Claude.Request(
      model = req.model,
      messages = claudeMsgs,
      maxTokens = req.maxTokens.getOrElse(defaultMaxTokens),
      system = system,
      temperature = req.temperature,
      topP = req.topP,
      stream = req.stream
    )

  /** Convert Claude request to Gemini request */
  def claudeToGemini(req: Claude.Request): Gemini.Request =
    val sysInstruction = req.system.map { sys =>
      Gemini.SystemInstruction(
        parts = List(Gemini.Part(text = Some(sys)))
      )
    }

    val contents = req.messages.map { msg =>
      val role = if msg.role == "assistant" then "model" else msg.role
      val parts = msg.content.collect {
        case Claude.TextContent(_, text) => Gemini.Part(text = Some(text))
      }
      Gemini.Content(role = role, parts = parts)
    }

    val config = Gemini.GenerationConfig(
      temperature = req.temperature,
      topP = req.topP,
      maxOutputTokens = Some(req.maxTokens)
    )

    Gemini.Request(
      contents = contents,
      systemInstruction = sysInstruction,
      generationConfig = Some(config)
    )

  /** Convert OpenAI response to Claude response */
  def openAIResponseToClaude(resp: OpenAI.Response, model: String): Claude.Response =
    val content = resp.choices.headOption
      .flatMap(_.message)
      .map(msg => List(Claude.TextContent(text = msg.content)))
      .getOrElse(List.empty)

    val stopReason = resp.choices.headOption
      .flatMap(_.finishReason)
      .map {
        case "stop" => "end_turn"
        case "length" => "max_tokens"
        case "tool_calls" => "tool_use"
        case _ => "end_turn"
      }

    val usage = resp.usage.map { u =>
      Claude.Usage(
        inputTokens = u.promptTokens,
        outputTokens = u.completionTokens
      )
    }

    Claude.Response(
      id = resp.id,
      `type` = "message",
      role = "assistant",
      content = content,
      model = model,
      stopReason = stopReason,
      usage = usage
    )

  /** Convert Gemini response to Claude response */
  def geminiResponseToClaude(resp: Gemini.Response, model: String): Claude.Response =
    val content = resp.candidates.headOption
      .map { candidate =>
        candidate.content.parts.flatMap { part =>
          part.text.map(t => Claude.TextContent(text = t))
        }
      }
      .getOrElse(List.empty)

    val stopReason = resp.candidates.headOption
      .flatMap(_.finishReason)
      .map {
        case "STOP" => "end_turn"
        case "MAX_TOKENS" => "max_tokens"
        case _ => "end_turn"
      }

    val usage = resp.usageMetadata.map { u =>
      Claude.Usage(
        inputTokens = u.promptTokenCount,
        outputTokens = u.candidatesTokenCount
      )
    }

    Claude.Response(
      id = java.util.UUID.randomUUID().toString,
      `type` = "message",
      role = "assistant",
      content = content,
      model = model,
      stopReason = stopReason,
      usage = usage
    )

  /** Convert Claude response to OpenAI response */
  def claudeToOpenAI(resp: Claude.Response, model: String): OpenAI.Response =
    val text = resp.content.collect {
      case Claude.TextContent(_, t) => t
    }.mkString("")

    val message = OpenAI.Message(
      role = "assistant",
      content = text
    )

    val choice = OpenAI.Choice(
      index = 0,
      message = Some(message),
      finishReason = resp.stopReason
    )

    val usage = resp.usage.map { u =>
      OpenAI.Usage(
        promptTokens = u.inputTokens,
        completionTokens = u.outputTokens,
        totalTokens = u.inputTokens + u.outputTokens
      )
    }

    OpenAI.Response(
      id = resp.id,
      `object` = "chat.completion",
      created = System.currentTimeMillis() / 1000,
      model = model,
      choices = List(choice),
      usage = usage
    )
