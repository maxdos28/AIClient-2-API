package com.aiproxy.model.claude;

import lombok.Data;
import java.util.List;

@Data
public class ClaudeMessage {
    private String role;
    private List<ClaudeContent> content;
}
