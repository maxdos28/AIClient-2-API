package com.aiproxy.model.openai;

import lombok.Data;
import java.util.List;

@Data
public class OpenAIRequest {
    private String model;
    private List<OpenAIMessage> messages;
    private Integer maxTokens;
    private Float temperature;
    private Float topP;
    private Boolean stream;
    private List<Tool> tools;
}
