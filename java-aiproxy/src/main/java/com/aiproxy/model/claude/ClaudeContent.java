package com.aiproxy.model.claude;

import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.annotation.JsonSubTypes;
import com.fasterxml.jackson.annotation.JsonTypeInfo;
import lombok.Data;
import java.util.Map;

@JsonTypeInfo(use = JsonTypeInfo.Id.NAME, property = "type")
@JsonSubTypes({
    @JsonSubTypes.Type(value = ClaudeContent.TextContent.class, name = "text"),
    @JsonSubTypes.Type(value = ClaudeContent.ImageContent.class, name = "image"),
    @JsonSubTypes.Type(value = ClaudeContent.ToolUseContent.class, name = "tool_use"),
    @JsonSubTypes.Type(value = ClaudeContent.ToolResultContent.class, name = "tool_result")
})
public abstract class ClaudeContent {
    public abstract String getType();
    
    @Data
    public static class TextContent extends ClaudeContent {
        private String text;
        
        @Override
        public String getType() {
            return "text";
        }
    }
    
    @Data
    public static class ImageContent extends ClaudeContent {
        private ImageSource source;
        
        @Override
        public String getType() {
            return "image";
        }
        
        @Data
        public static class ImageSource {
            private String type;
            
            @JsonProperty("media_type")
            private String mediaType;
            
            private String data;
        }
    }
    
    @Data
    public static class ToolUseContent extends ClaudeContent {
        private String id;
        private String name;
        private Map<String, Object> input;
        
        @Override
        public String getType() {
            return "tool_use";
        }
    }
    
    @Data
    public static class ToolResultContent extends ClaudeContent {
        @JsonProperty("tool_use_id")
        private String toolUseId;
        
        private String content;
        
        @Override
        public String getType() {
            return "tool_result";
        }
    }
}
