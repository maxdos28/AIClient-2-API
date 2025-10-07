package com.aiproxy.model.gemini;

import lombok.Data;
import java.util.List;

@Data
public class GeminiContent {
    private String role;
    private List<GeminiPart> parts;
}
