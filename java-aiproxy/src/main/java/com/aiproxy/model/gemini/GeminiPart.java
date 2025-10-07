package com.aiproxy.model.gemini;

import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.Data;
import java.util.Map;

@Data
public class GeminiPart {
    private String text;
    
    @JsonProperty("inlineData")
    private InlineData inlineData;
    
    @JsonProperty("functionCall")
    private FunctionCall functionCall;
    
    @JsonProperty("functionResponse")
    private FunctionResponse functionResponse;
    
    @Data
    public static class InlineData {
        @JsonProperty("mimeType")
        private String mimeType;
        
        private String data;
    }
    
    @Data
    public static class FunctionCall {
        private String name;
        private Map<String, Object> args;
    }
    
    @Data
    public static class FunctionResponse {
        private String name;
        private Map<String, Object> response;
    }
}
