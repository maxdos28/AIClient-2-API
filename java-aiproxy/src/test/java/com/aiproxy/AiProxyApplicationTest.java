package com.aiproxy;

import org.junit.jupiter.api.Test;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.test.context.TestPropertySource;

@SpringBootTest
@TestPropertySource(properties = {
    "openai.api-key=test-key",
    "server.port=0"
})
class AiProxyApplicationTest {
    
    @Test
    void contextLoads() {
        // Test that Spring context loads successfully
    }
}
