package com.aiproxy.provider

import com.aiproxy.model.{Protocol, OpenAI}
import sttp.client3.*
import sttp.client3.akkahttp.AkkaHttpBackend
import spray.json.*
import scala.concurrent.{Future, ExecutionContext}
import com.typesafe.scalalogging.LazyLogging

class OpenAIProvider(
  apiKey: String,
  baseUrl: String = "https://api.openai.com/v1"
)(using ec: ExecutionContext) extends AIProvider with LazyLogging:

  private val backend = AkkaHttpBackend()

  def chatCompletion(request: Any): Future[Any] =
    import OpenAI.given
    
    request match
      case req: OpenAI.Request =>
        val requestBody = req.toJson.compactPrint
        
        val response = basicRequest
          .post(uri"$baseUrl/chat/completions")
          .header("Authorization", s"Bearer $apiKey")
          .header("Content-Type", "application/json")
          .body(requestBody)
          .send(backend)

        response.map { resp =>
          logger.info(s"OpenAI response status: ${resp.code}")
          resp.body match
            case Right(body) => body.parseJson.convertTo[OpenAI.Response]
            case Left(error) => throw new Exception(s"OpenAI API error: $error")
        }
      
      case _ => Future.failed(new Exception("Invalid request type"))

  def protocol: Protocol = Protocol.OpenAI
  def name: String = "openai"
