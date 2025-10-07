package com.aiproxy.converter;

import com.aiproxy.model.claude.*;
import com.aiproxy.model.gemini.*;
import com.aiproxy.model.openai.*;
import org.springframework.stereotype.Component;

import java.util.*;
import java.util.stream.Collectors;

@Component
public class ProtocolConverter {
    
    private static final int DEFAULT_MAX_TOKENS = 8192;
    
    /**
     * Convert OpenAI request to Claude request
     */
    public ClaudeRequest openAIToClaude(OpenAIRequest request) {
        ClaudeRequest claude = new ClaudeRequest();
        claude.setModel(request.getModel());
        claude.setMaxTokens(request.getMaxTokens() != null ? request.getMaxTokens() : DEFAULT_MAX_TOKENS);
        claude.setTemperature(request.getTemperature());
        claude.setTopP(request.getTopP());
        claude.setStream(request.getStream());
        
        List<ClaudeMessage> messages = new ArrayList<>();
        String system = null;
        
        for (OpenAIMessage msg : request.getMessages()) {
            if ("system".equals(msg.getRole())) {
                system = extractTextContent(msg.getContent());
                continue;
            }
            
            ClaudeMessage claudeMsg = new ClaudeMessage();
            claudeMsg.setRole(msg.getRole());
            claudeMsg.setContent(List.of(new ClaudeContent.TextContent() {{
                setText(extractTextContent(msg.getContent()));
            }}));
            messages.add(claudeMsg);
        }
        
        claude.setSystem(system);
        claude.setMessages(messages);
        
        return claude;
    }
    
    /**
     * Convert Claude request to Gemini request
     */
    public GeminiRequest claudeToGemini(ClaudeRequest request) {
        GeminiRequest gemini = new GeminiRequest();
        
        // System instruction
        if (request.getSystem() != null) {
            GeminiRequest.SystemInstruction sysInst = new GeminiRequest.SystemInstruction();
            GeminiPart part = new GeminiPart();
            part.setText(request.getSystem());
            sysInst.setParts(List.of(part));
            gemini.setSystemInstruction(sysInst);
        }
        
        // Messages
        List<GeminiContent> contents = request.getMessages().stream()
            .map(msg -> {
                GeminiContent content = new GeminiContent();
                content.setRole("assistant".equals(msg.getRole()) ? "model" : msg.getRole());
                
                List<GeminiPart> parts = msg.getContent().stream()
                    .map(c -> {
                        GeminiPart part = new GeminiPart();
                        if (c instanceof ClaudeContent.TextContent textContent) {
                            part.setText(textContent.getText());
                        }
                        return part;
                    })
                    .collect(Collectors.toList());
                
                content.setParts(parts);
                return content;
            })
            .collect(Collectors.toList());
        
        gemini.setContents(contents);
        
        // Generation config
        GeminiRequest.GenerationConfig config = new GeminiRequest.GenerationConfig();
        config.setTemperature(request.getTemperature());
        config.setTopP(request.getTopP());
        config.setMaxOutputTokens(request.getMaxTokens());
        gemini.setGenerationConfig(config);
        
        return gemini;
    }
    
    /**
     * Convert OpenAI response to Claude response
     */
    public ClaudeResponse openAIResponseToClaude(OpenAIResponse response, String model) {
        ClaudeResponse claude = new ClaudeResponse();
        claude.setId(response.getId());
        claude.setType("message");
        claude.setRole("assistant");
        claude.setModel(model);
        
        if (!response.getChoices().isEmpty()) {
            OpenAIResponse.OpenAIChoice choice = response.getChoices().get(0);
            if (choice.getMessage() != null) {
                String text = extractTextContent(choice.getMessage().getContent());
                ClaudeContent.TextContent content = new ClaudeContent.TextContent();
                content.setText(text);
                claude.setContent(List.of(content));
            }
            
            String stopReason = switch (choice.getFinishReason() != null ? choice.getFinishReason() : "") {
                case "stop" -> "end_turn";
                case "length" -> "max_tokens";
                case "tool_calls" -> "tool_use";
                default -> "end_turn";
            };
            claude.setStopReason(stopReason);
        }
        
        if (response.getUsage() != null) {
            ClaudeResponse.Usage usage = new ClaudeResponse.Usage();
            usage.setInputTokens(response.getUsage().getPromptTokens());
            usage.setOutputTokens(response.getUsage().getCompletionTokens());
            claude.setUsage(usage);
        }
        
        return claude;
    }
    
    /**
     * Convert Gemini response to Claude response
     */
    public ClaudeResponse geminiResponseToClaude(GeminiResponse response, String model) {
        ClaudeResponse claude = new ClaudeResponse();
        claude.setId(UUID.randomUUID().toString());
        claude.setType("message");
        claude.setRole("assistant");
        claude.setModel(model);
        
        if (!response.getCandidates().isEmpty()) {
            GeminiResponse.Candidate candidate = response.getCandidates().get(0);
            
            List<ClaudeContent> contents = candidate.getContent().getParts().stream()
                .filter(p -> p.getText() != null)
                .map(p -> {
                    ClaudeContent.TextContent content = new ClaudeContent.TextContent();
                    content.setText(p.getText());
                    return (ClaudeContent) content;
                })
                .collect(Collectors.toList());
            
            claude.setContent(contents);
            
            String stopReason = switch (candidate.getFinishReason() != null ? candidate.getFinishReason() : "") {
                case "STOP" -> "end_turn";
                case "MAX_TOKENS" -> "max_tokens";
                default -> "end_turn";
            };
            claude.setStopReason(stopReason);
        }
        
        if (response.getUsageMetadata() != null) {
            ClaudeResponse.Usage usage = new ClaudeResponse.Usage();
            usage.setInputTokens(response.getUsageMetadata().getPromptTokenCount());
            usage.setOutputTokens(response.getUsageMetadata().getCandidatesTokenCount());
            claude.setUsage(usage);
        }
        
        return claude;
    }
    
    private String extractTextContent(Object content) {
        if (content instanceof String) {
            return (String) content;
        }
        if (content instanceof List) {
            // Handle array of content parts
            return "";
        }
        return "";
    }
}
