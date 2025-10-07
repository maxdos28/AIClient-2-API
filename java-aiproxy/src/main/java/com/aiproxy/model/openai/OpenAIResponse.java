package com.aiproxy.model.openai;

import lombok.Data;
import java.util.List;

@Data
public class OpenAIResponse {
    private String id;
    private String object;
    private Long created;
    private String model;
    private List<OpenAIChoice> choices;
    private Usage usage;
    
    @Data
    public static class OpenAIChoice {
        private Integer index;
        private OpenAIMessage message;
        private OpenAIMessage delta;
        private String finishReason;
    }
    
    @Data
    public static class Usage {
        private Integer promptTokens;
        private Integer completionTokens;
        private Integer totalTokens;
    }
}
