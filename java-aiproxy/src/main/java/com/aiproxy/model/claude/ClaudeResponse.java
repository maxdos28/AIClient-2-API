package com.aiproxy.model.claude;

import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.Data;
import java.util.List;

@Data
public class ClaudeResponse {
    private String id;
    private String type;
    private String role;
    private List<ClaudeContent> content;
    private String model;
    
    @JsonProperty("stop_reason")
    private String stopReason;
    
    private Usage usage;
    
    @Data
    public static class Usage {
        @JsonProperty("input_tokens")
        private Integer inputTokens;
        
        @JsonProperty("output_tokens")
        private Integer outputTokens;
    }
}
