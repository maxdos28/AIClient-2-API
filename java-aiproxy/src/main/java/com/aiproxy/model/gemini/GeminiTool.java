package com.aiproxy.model.gemini;

import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.Data;
import java.util.List;
import java.util.Map;

@Data
public class GeminiTool {
    @JsonProperty("functionDeclarations")
    private List<FunctionDeclaration> functionDeclarations;
    
    @Data
    public static class FunctionDeclaration {
        private String name;
        private String description;
        private Map<String, Object> parameters;
    }
}
