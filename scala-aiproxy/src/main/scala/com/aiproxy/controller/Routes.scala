package com.aiproxy.controller

import akka.actor.typed.ActorSystem
import akka.http.scaladsl.server.Directives.*
import akka.http.scaladsl.server.Route
import akka.http.scaladsl.model.{StatusCodes, ContentTypes, HttpEntity}
import akka.http.scaladsl.marshallers.sprayjson.SprayJsonSupport.*
import spray.json.*
import com.aiproxy.model.{OpenAI, Claude, Protocol}
import com.aiproxy.provider.AIProvider
import com.aiproxy.converter.ProtocolConverter
import com.typesafe.scalalogging.LazyLogging
import scala.concurrent.{ExecutionContext, Future}
import scala.util.{Success, Failure}

class Routes(providers: List[AIProvider])(using system: ActorSystem[?], ec: ExecutionContext) 
  extends LazyLogging:

  import OpenAI.given
  import Claude.given

  val routes: Route =
    pathPrefix("v1") {
      healthCheck ~ chatCompletions ~ listModels
    } ~ health

  private def health: Route =
    path("health") {
      get {
        complete(StatusCodes.OK, JsObject(
          "status" -> JsString("ok"),
          "version" -> JsString("1.0.0")
        ))
      }
    }

  private def healthCheck: Route =
    path("health") {
      get {
        complete(StatusCodes.OK, JsObject(
          "status" -> JsString("ok"),
          "version" -> JsString("1.0.0")
        ))
      }
    }

  private def chatCompletions: Route =
    path("chat" / "completions") {
      post {
        entity(as[OpenAI.Request]) { request =>
          logger.info(s"Received chat completion request for model: ${request.model}")
          
          val provider = providers.headOption.getOrElse {
            throw new Exception("No providers configured")
          }

          val providerRequest = provider.protocol match
            case Protocol.OpenAI => request
            case Protocol.Claude => ProtocolConverter.openAIToClaude(request)
            case Protocol.Gemini =>
              val claudeReq = ProtocolConverter.openAIToClaude(request)
              ProtocolConverter.claudeToGemini(claudeReq)

          onComplete(provider.chatCompletion(providerRequest)) {
            case Success(response) =>
              val convertedResponse = convertResponse(response, provider, request.model)
              complete(StatusCodes.OK, convertedResponse)
            
            case Failure(ex) =>
              logger.error(s"Error processing request: ${ex.getMessage}", ex)
              complete(StatusCodes.InternalServerError, JsObject(
                "error" -> JsObject(
                  "message" -> JsString(ex.getMessage),
                  "type" -> JsString("api_error")
                )
              ))
          }
        }
      }
    }

  private def listModels: Route =
    path("models") {
      get {
        complete(StatusCodes.OK, JsObject(
          "object" -> JsString("list"),
          "data" -> JsArray(
            JsObject(
              "id" -> JsString("gpt-3.5-turbo"),
              "object" -> JsString("model"),
              "created" -> JsNumber(System.currentTimeMillis() / 1000),
              "owned_by" -> JsString("openai")
            ),
            JsObject(
              "id" -> JsString("claude-3-opus-20240229"),
              "object" -> JsString("model"),
              "created" -> JsNumber(System.currentTimeMillis() / 1000),
              "owned_by" -> JsString("anthropic")
            )
          )
        ))
      }
    }

  private def convertResponse(response: Any, provider: AIProvider, model: String): JsValue =
    provider.protocol match
      case Protocol.OpenAI =>
        response.asInstanceOf[OpenAI.Response].toJson
      
      case Protocol.Claude =>
        val claudeResp = response.asInstanceOf[Claude.Response]
        ProtocolConverter.claudeToOpenAI(claudeResp, model).toJson
      
      case Protocol.Gemini =>
        // Handle Gemini response conversion
        response.asInstanceOf[JsValue]
