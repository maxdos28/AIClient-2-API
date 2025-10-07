package com.aiproxy.model.openai;

import lombok.Data;

@Data
public class ToolCall {
    private String id;
    private String type;
    private ToolCallFunction function;
    
    @Data
    public static class ToolCallFunction {
        private String name;
        private String arguments;
    }
}
