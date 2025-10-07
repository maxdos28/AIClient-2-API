package com.aiproxy.provider;

import com.aiproxy.model.Protocol;
import reactor.core.publisher.Mono;

public interface AIProvider {
    Mono<Object> chatCompletion(Object request);
    Protocol getProtocol();
    String getName();
}
