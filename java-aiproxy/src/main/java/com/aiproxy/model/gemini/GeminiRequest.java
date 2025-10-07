package com.aiproxy.model.gemini;

import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.Data;
import java.util.List;

@Data
public class GeminiRequest {
    private List<GeminiContent> contents;
    
    @JsonProperty("systemInstruction")
    private SystemInstruction systemInstruction;
    
    @JsonProperty("generationConfig")
    private GenerationConfig generationConfig;
    
    private List<GeminiTool> tools;
    
    @Data
    public static class SystemInstruction {
        private List<GeminiPart> parts;
    }
    
    @Data
    public static class GenerationConfig {
        private Float temperature;
        
        @JsonProperty("topP")
        private Float topP;
        
        @JsonProperty("maxOutputTokens")
        private Integer maxOutputTokens;
    }
}
