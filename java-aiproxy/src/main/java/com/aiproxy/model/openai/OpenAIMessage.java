package com.aiproxy.model.openai;

import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.Data;
import java.util.List;

@Data
public class OpenAIMessage {
    private String role;
    private Object content; // Can be String or List<ContentPart>
    private String name;
    
    @JsonProperty("tool_calls")
    private List<ToolCall> toolCalls;
    
    @JsonProperty("tool_call_id")
    private String toolCallId;
}
