package com.aiproxy.provider;

import com.aiproxy.model.Protocol;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.http.MediaType;
import org.springframework.stereotype.Component;
import org.springframework.web.reactive.function.client.WebClient;
import reactor.core.publisher.Mono;

@Slf4j
@Component
public class OpenAIProvider implements AIProvider {
    
    private final WebClient webClient;
    
    public OpenAIProvider(
            @Value("${openai.api-key:}") String apiKey,
            @Value("${openai.base-url:https://api.openai.com/v1}") String baseUrl,
            WebClient.Builder webClientBuilder) {
        this.webClient = webClientBuilder
                .baseUrl(baseUrl)
                .defaultHeader("Authorization", "Bearer " + apiKey)
                .defaultHeader("Content-Type", "application/json")
                .build();
    }
    
    @Override
    public Mono<Object> chatCompletion(Object request) {
        return webClient.post()
                .uri("/chat/completions")
                .contentType(MediaType.APPLICATION_JSON)
                .bodyValue(request)
                .retrieve()
                .bodyToMono(Object.class)
                .doOnError(e -> log.error("OpenAI API error", e));
    }
    
    @Override
    public Protocol getProtocol() {
        return Protocol.OPENAI;
    }
    
    @Override
    public String getName() {
        return "openai";
    }
}
