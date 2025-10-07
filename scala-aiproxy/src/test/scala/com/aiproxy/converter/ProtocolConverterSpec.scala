package com.aiproxy.converter

import com.aiproxy.model.{OpenAI, Claude, Gemini}
import org.scalatest.flatspec.AnyFlatSpec
import org.scalatest.matchers.should.Matchers

class ProtocolConverterSpec extends AnyFlatSpec with Matchers:

  "ProtocolConverter" should "convert OpenAI request to Claude" in {
    val openAIReq = OpenAI.Request(
      model = "gpt-3.5-turbo",
      messages = List(
        OpenAI.Message(role = "system", content = "You are helpful"),
        OpenAI.Message(role = "user", content = "Hello")
      ),
      maxTokens = Some(100),
      temperature = Some(0.7)
    )

    val claudeReq = ProtocolConverter.openAIToClaude(openAIReq)

    claudeReq.model shouldBe "gpt-3.5-turbo"
    claudeReq.system shouldBe Some("You are helpful")
    claudeReq.messages should have length 1
    claudeReq.messages.head.role shouldBe "user"
    claudeReq.maxTokens shouldBe 100
    claudeReq.temperature shouldBe Some(0.7)
  }

  it should "convert Claude request to Gemini" in {
    val claudeReq = Claude.Request(
      model = "claude-3-opus",
      messages = List(
        Claude.Message(
          role = "user",
          content = List(Claude.TextContent(text = "Hello"))
        )
      ),
      maxTokens = 100,
      system = Some("You are helpful")
    )

    val geminiReq = ProtocolConverter.claudeToGemini(claudeReq)

    geminiReq.systemInstruction shouldBe defined
    geminiReq.contents should have length 1
    geminiReq.contents.head.role shouldBe "user"
    geminiReq.generationConfig shouldBe defined
    geminiReq.generationConfig.get.maxOutputTokens shouldBe Some(100)
  }

  it should "convert Claude response to OpenAI" in {
    val claudeResp = Claude.Response(
      id = "test-123",
      `type` = "message",
      role = "assistant",
      content = List(Claude.TextContent(text = "Hello there!")),
      model = "claude-3-opus",
      stopReason = Some("end_turn"),
      usage = Some(Claude.Usage(inputTokens = 10, outputTokens = 5))
    )

    val openAIResp = ProtocolConverter.claudeToOpenAI(claudeResp, "claude-3-opus")

    openAIResp.id shouldBe "test-123"
    openAIResp.model shouldBe "claude-3-opus"
    openAIResp.choices should have length 1
    openAIResp.choices.head.message.get.content shouldBe "Hello there!"
    openAIResp.usage shouldBe defined
    openAIResp.usage.get.totalTokens shouldBe 15
  }
