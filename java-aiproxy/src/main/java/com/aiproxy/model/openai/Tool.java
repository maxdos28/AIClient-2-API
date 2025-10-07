package com.aiproxy.model.openai;

import lombok.Data;
import java.util.Map;

@Data
public class Tool {
    private String type;
    private ToolFunction function;
    
    @Data
    public static class ToolFunction {
        private String name;
        private String description;
        private Map<String, Object> parameters;
    }
}
