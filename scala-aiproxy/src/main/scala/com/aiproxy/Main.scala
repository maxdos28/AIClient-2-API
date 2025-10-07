package com.aiproxy

import akka.actor.typed.ActorSystem
import akka.actor.typed.scaladsl.Behaviors
import akka.http.scaladsl.Http
import akka.http.scaladsl.server.Route
import com.aiproxy.controller.Routes
import com.aiproxy.provider.{AIProvider, OpenAIProvider}
import com.typesafe.config.ConfigFactory
import com.typesafe.scalalogging.LazyLogging
import scala.concurrent.{ExecutionContext, Future}
import scala.util.{Success, Failure}

object Main extends App with LazyLogging:
  given system: ActorSystem[Nothing] = ActorSystem(Behaviors.empty, "aiproxy-system")
  given ec: ExecutionContext = system.executionContext

  // Load configuration
  val config = ConfigFactory.load()
  val host = config.getString("server.host")
  val port = config.getInt("server.port")

  // Initialize providers
  val providers = initializeProviders()

  // Create routes
  val routes = Routes(providers)

  // Start HTTP server
  val bindingFuture: Future[Http.ServerBinding] = Http()
    .newServerAt(host, port)
    .bind(routes.routes)

  bindingFuture.onComplete {
    case Success(binding) =>
      val address = binding.localAddress
      logger.info(s"🚀 Scala AI Proxy server online at http://${address.getHostString}:${address.getPort}/")
      logger.info(s"📊 ${providers.length} provider(s) configured")
      logger.info(s"✨ Using Scala 3 with functional programming style")
    
    case Failure(ex) =>
      logger.error(s"Failed to bind HTTP server: ${ex.getMessage}", ex)
      system.terminate()
  }

  private def initializeProviders(): List[AIProvider] =
    val openaiKey = sys.env.get("OPENAI_API_KEY")
      .orElse(Option(config.getString("openai.api-key")).filter(_.nonEmpty))

    val openaiBaseUrl = sys.env.get("OPENAI_BASE_URL")
      .orElse(Option(config.getString("openai.base-url")))
      .getOrElse("https://api.openai.com/v1")

    val providerList = List.newBuilder[AIProvider]

    openaiKey.foreach { key =>
      logger.info(s"Initializing OpenAI provider with base URL: $openaiBaseUrl")
      providerList += OpenAIProvider(key, openaiBaseUrl)
    }

    val result = providerList.result()
    
    if result.isEmpty then
      logger.warn("⚠️  No providers configured. Please set API keys via environment variables.")
    
    result
