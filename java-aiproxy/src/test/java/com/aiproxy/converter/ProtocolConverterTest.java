package com.aiproxy.converter;

import com.aiproxy.model.claude.ClaudeRequest;
import com.aiproxy.model.gemini.GeminiRequest;
import com.aiproxy.model.openai.OpenAIMessage;
import com.aiproxy.model.openai.OpenAIRequest;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import java.util.List;

import static org.junit.jupiter.api.Assertions.*;

class ProtocolConverterTest {
    
    private ProtocolConverter converter;
    
    @BeforeEach
    void setUp() {
        converter = new ProtocolConverter();
    }
    
    @Test
    void testOpenAIToClaude() {
        OpenAIRequest request = new OpenAIRequest();
        request.setModel("gpt-3.5-turbo");
        
        OpenAIMessage systemMsg = new OpenAIMessage();
        systemMsg.setRole("system");
        systemMsg.setContent("You are helpful");
        
        OpenAIMessage userMsg = new OpenAIMessage();
        userMsg.setRole("user");
        userMsg.setContent("Hello");
        
        request.setMessages(List.of(systemMsg, userMsg));
        request.setMaxTokens(100);
        request.setTemperature(0.7f);
        
        ClaudeRequest result = converter.openAIToClaude(request);
        
        assertNotNull(result);
        assertEquals("gpt-3.5-turbo", result.getModel());
        assertEquals("You are helpful", result.getSystem());
        assertEquals(1, result.getMessages().size());
        assertEquals(100, result.getMaxTokens());
    }
    
    @Test
    void testClaudeToGemini() {
        ClaudeRequest request = new ClaudeRequest();
        request.setModel("claude-3-opus");
        request.setSystem("You are helpful");
        request.setMaxTokens(100);
        
        GeminiRequest result = converter.claudeToGemini(request);
        
        assertNotNull(result);
        assertNotNull(result.getSystemInstruction());
        assertNotNull(result.getGenerationConfig());
        assertEquals(100, result.getGenerationConfig().getMaxOutputTokens());
    }
}
