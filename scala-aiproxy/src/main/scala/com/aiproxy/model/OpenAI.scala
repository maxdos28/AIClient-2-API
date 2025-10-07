package com.aiproxy.model

import spray.json.*

object OpenAI:
  case class Message(
    role: String,
    content: String,
    name: Option[String] = None,
    toolCalls: Option[List[ToolCall]] = None,
    toolCallId: Option[String] = None
  )

  case class Tool(
    `type`: String,
    function: ToolFunction
  )

  case class ToolFunction(
    name: String,
    description: Option[String] = None,
    parameters: Option[Map[String, Any]] = None
  )

  case class ToolCall(
    id: String,
    `type`: String,
    function: ToolCallFunction
  )

  case class ToolCallFunction(
    name: String,
    arguments: String
  )

  case class Request(
    model: String,
    messages: List[Message],
    maxTokens: Option[Int] = None,
    temperature: Option[Double] = None,
    topP: Option[Double] = None,
    stream: Option[Boolean] = None,
    tools: Option[List[Tool]] = None
  )

  case class Response(
    id: String,
    `object`: String,
    created: Long,
    model: String,
    choices: List[Choice],
    usage: Option[Usage] = None
  )

  case class Choice(
    index: Int,
    message: Option[Message] = None,
    delta: Option[Message] = None,
    finishReason: Option[String] = None
  )

  case class Usage(
    promptTokens: Int,
    completionTokens: Int,
    totalTokens: Int
  )

  // JSON Format definitions
  given JsonFormat[ToolCallFunction] = jsonFormat2(ToolCallFunction.apply)
  given JsonFormat[ToolCall] = jsonFormat3(ToolCall.apply)
  given JsonFormat[Message] = jsonFormat5(Message.apply)
  given JsonFormat[ToolFunction] = jsonFormat3(ToolFunction.apply)
  given JsonFormat[Tool] = jsonFormat2(Tool.apply)
  given JsonFormat[Request] = jsonFormat7(Request.apply)
  given JsonFormat[Usage] = jsonFormat3(Usage.apply)
  given JsonFormat[Choice] = jsonFormat4(Choice.apply)
  given JsonFormat[Response] = jsonFormat6(Response.apply)
