package com.aiproxy.model.claude;

import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.Data;
import java.util.List;

@Data
public class ClaudeRequest {
    private String model;
    private List<ClaudeMessage> messages;
    
    @JsonProperty("max_tokens")
    private Integer maxTokens;
    
    private String system;
    private Float temperature;
    
    @JsonProperty("top_p")
    private Float topP;
    
    private Boolean stream;
    private List<ClaudeTool> tools;
}
