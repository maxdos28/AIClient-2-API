package com.aiproxy.model.claude;

import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.Data;
import java.util.Map;

@Data
public class ClaudeTool {
    private String name;
    private String description;
    
    @JsonProperty("input_schema")
    private Map<String, Object> inputSchema;
}
