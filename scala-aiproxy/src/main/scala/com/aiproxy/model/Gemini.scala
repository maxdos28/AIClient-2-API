package com.aiproxy.model

import spray.json.*

object Gemini:
  case class Part(
    text: Option[String] = None,
    inlineData: Option[InlineData] = None,
    functionCall: Option[FunctionCall] = None,
    functionResponse: Option[FunctionResponse] = None
  )

  case class InlineData(
    mimeType: String,
    data: String
  )

  case class FunctionCall(
    name: String,
    args: Map[String, Any]
  )

  case class FunctionResponse(
    name: String,
    response: Map[String, Any]
  )

  case class Content(
    role: String,
    parts: List[Part]
  )

  case class SystemInstruction(
    parts: List[Part]
  )

  case class GenerationConfig(
    temperature: Option[Double] = None,
    topP: Option[Double] = None,
    maxOutputTokens: Option[Int] = None
  )

  case class FunctionDeclaration(
    name: String,
    description: Option[String] = None,
    parameters: Option[Map[String, Any]] = None
  )

  case class Tool(
    functionDeclarations: List[FunctionDeclaration]
  )

  case class Request(
    contents: List[Content],
    systemInstruction: Option[SystemInstruction] = None,
    generationConfig: Option[GenerationConfig] = None,
    tools: Option[List[Tool]] = None
  )

  case class Response(
    candidates: List[Candidate],
    usageMetadata: Option[UsageMetadata] = None
  )

  case class Candidate(
    content: Content,
    finishReason: Option[String] = None
  )

  case class UsageMetadata(
    promptTokenCount: Int,
    candidatesTokenCount: Int,
    totalTokenCount: Int
  )

  // JSON Formats
  given JsonFormat[InlineData] = jsonFormat2(InlineData.apply)
  given JsonFormat[FunctionCall] = jsonFormat2(FunctionCall.apply)
  given JsonFormat[FunctionResponse] = jsonFormat2(FunctionResponse.apply)
  given JsonFormat[Part] = jsonFormat4(Part.apply)
  given JsonFormat[Content] = jsonFormat2(Content.apply)
  given JsonFormat[SystemInstruction] = jsonFormat1(SystemInstruction.apply)
  given JsonFormat[GenerationConfig] = jsonFormat3(GenerationConfig.apply)
  given JsonFormat[FunctionDeclaration] = jsonFormat3(FunctionDeclaration.apply)
  given JsonFormat[Tool] = jsonFormat1(Tool.apply)
  given JsonFormat[Request] = jsonFormat4(Request.apply)
  given JsonFormat[UsageMetadata] = jsonFormat3(UsageMetadata.apply)
  given JsonFormat[Candidate] = jsonFormat2(Candidate.apply)
  given JsonFormat[Response] = jsonFormat2(Response.apply)
