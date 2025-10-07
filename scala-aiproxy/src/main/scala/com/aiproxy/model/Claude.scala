package com.aiproxy.model

import spray.json.*

object Claude:
  sealed trait Content
  
  case class TextContent(
    `type`: String = "text",
    text: String
  ) extends Content

  case class ImageContent(
    `type`: String = "image",
    source: ImageSource
  ) extends Content

  case class ImageSource(
    `type`: String,
    mediaType: String,
    data: String
  )

  case class ToolUseContent(
    `type`: String = "tool_use",
    id: String,
    name: String,
    input: Map[String, Any]
  ) extends Content

  case class Message(
    role: String,
    content: List[Content]
  )

  case class Tool(
    name: String,
    description: Option[String] = None,
    inputSchema: Option[Map[String, Any]] = None
  )

  case class Request(
    model: String,
    messages: List[Message],
    maxTokens: Int = 8192,
    system: Option[String] = None,
    temperature: Option[Double] = None,
    topP: Option[Double] = None,
    stream: Option[Boolean] = None,
    tools: Option[List[Tool]] = None
  )

  case class Response(
    id: String,
    `type`: String,
    role: String,
    content: List[Content],
    model: String,
    stopReason: Option[String] = None,
    usage: Option[Usage] = None
  )

  case class Usage(
    inputTokens: Int,
    outputTokens: Int
  )

  // JSON Formats
  given JsonFormat[ImageSource] = jsonFormat3(ImageSource.apply)
  
  given JsonFormat[Content] = new JsonFormat[Content]:
    def write(c: Content): JsValue = c match
      case tc: TextContent => 
        JsObject("type" -> JsString("text"), "text" -> JsString(tc.text))
      case ic: ImageContent =>
        JsObject("type" -> JsString("image"), "source" -> ic.source.toJson)
      case tuc: ToolUseContent =>
        JsObject(
          "type" -> JsString("tool_use"),
          "id" -> JsString(tuc.id),
          "name" -> JsString(tuc.name),
          "input" -> tuc.input.toJson
        )

    def read(value: JsValue): Content = value.asJsObject.getFields("type") match
      case Seq(JsString("text")) =>
        val text = value.asJsObject.fields("text").convertTo[String]
        TextContent(text = text)
      case Seq(JsString("image")) =>
        val source = value.asJsObject.fields("source").convertTo[ImageSource]
        ImageContent(source = source)
      case Seq(JsString("tool_use")) =>
        val fields = value.asJsObject.fields
        ToolUseContent(
          id = fields("id").convertTo[String],
          name = fields("name").convertTo[String],
          input = fields("input").convertTo[Map[String, Any]]
        )
      case _ => throw DeserializationException("Unknown content type")

  given JsonFormat[Message] = jsonFormat2(Message.apply)
  given JsonFormat[Tool] = jsonFormat3(Tool.apply)
  given JsonFormat[Request] = jsonFormat8(Request.apply)
  given JsonFormat[Usage] = jsonFormat2(Usage.apply)
  given JsonFormat[Response] = jsonFormat7(Response.apply)
