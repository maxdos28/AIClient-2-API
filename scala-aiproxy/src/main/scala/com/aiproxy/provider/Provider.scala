package com.aiproxy.provider

import com.aiproxy.model.Protocol
import scala.concurrent.Future

trait AIProvider:
  def chatCompletion(request: Any): Future[Any]
  def protocol: Protocol
  def name: String
