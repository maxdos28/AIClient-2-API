package com.aiproxy.controller;

import com.aiproxy.converter.ProtocolConverter;
import com.aiproxy.model.openai.OpenAIRequest;
import com.aiproxy.model.openai.OpenAIResponse;
import com.aiproxy.provider.AIProvider;
import com.fasterxml.jackson.databind.ObjectMapper;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.web.bind.annotation.*;
import reactor.core.publisher.Mono;

import java.util.List;
import java.util.Map;

@Slf4j
@RestController
@RequestMapping("/v1")
@RequiredArgsConstructor
public class ChatController {
    
    private final List<AIProvider> providers;
    private final ProtocolConverter converter;
    private final ObjectMapper objectMapper;
    
    @PostMapping("/chat/completions")
    public Mono<Object> chatCompletions(@RequestBody OpenAIRequest request) {
        log.info("Received chat completion request for model: {}", request.getModel());
        
        // Get first available provider
        AIProvider provider = providers.stream()
                .findFirst()
                .orElseThrow(() -> new RuntimeException("No providers configured"));
        
        // Convert request based on provider protocol
        Object providerRequest = switch (provider.getProtocol()) {
            case OPENAI -> request;
            case CLAUDE -> converter.openAIToClaude(request);
            case GEMINI -> {
                var claudeReq = converter.openAIToClaude(request);
                yield converter.claudeToGemini(claudeReq);
            }
        };
        
        // Call provider and convert response
        return provider.chatCompletion(providerRequest)
                .map(response -> convertResponse(response, provider, request.getModel()));
    }
    
    @GetMapping("/models")
    public Mono<Map<String, Object>> listModels() {
        return Mono.just(Map.of(
                "object", "list",
                "data", List.of(
                        Map.of(
                                "id", "gpt-3.5-turbo",
                                "object", "model",
                                "created", System.currentTimeMillis() / 1000,
                                "owned_by", "openai"
                        ),
                        Map.of(
                                "id", "claude-3-opus-20240229",
                                "object", "model",
                                "created", System.currentTimeMillis() / 1000,
                                "owned_by", "anthropic"
                        )
                )
        ));
    }
    
    private Object convertResponse(Object response, AIProvider provider, String model) {
        try {
            return switch (provider.getProtocol()) {
                case OPENAI -> response;
                case CLAUDE -> {
                    var claudeResp = objectMapper.convertValue(response, 
                            com.aiproxy.model.claude.ClaudeResponse.class);
                    yield convertClaudeToOpenAI(claudeResp, model);
                }
                case GEMINI -> {
                    var geminiResp = objectMapper.convertValue(response,
                            com.aiproxy.model.gemini.GeminiResponse.class);
                    var claudeResp = converter.geminiResponseToClaude(geminiResp, model);
                    yield convertClaudeToOpenAI(claudeResp, model);
                }
            };
        } catch (Exception e) {
            log.error("Error converting response", e);
            return response;
        }
    }
    
    private OpenAIResponse convertClaudeToOpenAI(
            com.aiproxy.model.claude.ClaudeResponse claudeResp, String model) {
        OpenAIResponse openAI = new OpenAIResponse();
        openAI.setId(claudeResp.getId());
        openAI.setObject("chat.completion");
        openAI.setCreated(System.currentTimeMillis() / 1000);
        openAI.setModel(model);
        
        OpenAIResponse.OpenAIChoice choice = new OpenAIResponse.OpenAIChoice();
        choice.setIndex(0);
        choice.setFinishReason("stop");
        
        com.aiproxy.model.openai.OpenAIMessage message = 
                new com.aiproxy.model.openai.OpenAIMessage();
        message.setRole("assistant");
        
        String text = claudeResp.getContent().stream()
                .filter(c -> c instanceof com.aiproxy.model.claude.ClaudeContent.TextContent)
                .map(c -> ((com.aiproxy.model.claude.ClaudeContent.TextContent) c).getText())
                .reduce("", (a, b) -> a + b);
        message.setContent(text);
        choice.setMessage(message);
        
        openAI.setChoices(List.of(choice));
        
        if (claudeResp.getUsage() != null) {
            OpenAIResponse.Usage usage = new OpenAIResponse.Usage();
            usage.setPromptTokens(claudeResp.getUsage().getInputTokens());
            usage.setCompletionTokens(claudeResp.getUsage().getOutputTokens());
            usage.setTotalTokens(claudeResp.getUsage().getInputTokens() + 
                    claudeResp.getUsage().getOutputTokens());
            openAI.setUsage(usage);
        }
        
        return openAI;
    }
}
