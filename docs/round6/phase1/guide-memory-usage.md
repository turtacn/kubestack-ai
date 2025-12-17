# Memory System Usage Guide

## Quick Start

### Basic Usage

```go
package main

import (
    "log"
    "github.com/kubestack-ai/kubestack-ai/internal/memory"
)

func main() {
    // Create memory manager with default config
    cfg := memory.DefaultMemoryConfig()
    manager, err := memory.NewMemoryManager(cfg)
    if err != nil {
        log.Fatal(err)
    }
    defer manager.Close()

    sessionID := "user-session-123"

    // Record a user message
    userEntry := memory.MemoryEntry{
        Role:    "user",
        Content: "What pods are running in the default namespace?",
    }
    manager.RecordMessage(sessionID, userEntry)

    // Record assistant response
    assistantEntry := memory.MemoryEntry{
        Role:    "assistant",
        Content: "There are 3 pods running: nginx-1, redis-1, mysql-1",
    }
    manager.RecordMessage(sessionID, assistantEntry)

    // Get conversation context
    context, err := manager.GetContext(sessionID, 0)
    if err != nil {
        log.Fatal(err)
    }

    // Display conversation
    for _, entry := range context {
        log.Printf("[%s] %s\n", entry.Role, entry.Content)
    }
}
```

### With Agent Integration

```go
package main

import (
    "context"
    "log"
    
    "github.com/kubestack-ai/kubestack-ai/internal/ai/agent"
    "github.com/kubestack-ai/kubestack-ai/internal/memory"
    "github.com/kubestack-ai/kubestack-ai/internal/nlp"
)

func main() {
    // Create NLP processor
    nlpProcessor := nlp.NewNLPProcessor(nil)
    
    // Create memory manager
    memoryManager, err := memory.NewMemoryManager(memory.DefaultMemoryConfig())
    if err != nil {
        log.Fatal(err)
    }
    defer memoryManager.Close()
    
    // Create agent with memory
    ag := agent.NewAgent(nlpProcessor, memoryManager)
    defer ag.Close()
    
    // Process user input (memory is handled automatically)
    response, err := ag.ProcessUserInput(context.Background(), &agent.UserInput{
        Text:      "Show me the status of nginx pods",
        SessionID: "session-123",
        UserID:    "user-456",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Response: %s\n", response.Text)
    
    // Get conversation history
    history, err := ag.GetConversationHistory("session-123", 1000)
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Conversation history has %d entries\n", len(history))
}
```

## Configuration

### Default Configuration

```go
cfg := memory.DefaultMemoryConfig()
// WorkingWindowSize: 20
// ShortTermTTL: 7 days
// StorePath: "./data/memory"
```

### Custom Configuration

```go
cfg := memory.MemoryConfig{
    WorkingWindowSize: 50,                      // Store more messages in memory
    ShortTermTTL:      30 * 24 * time.Hour,     // Keep for 30 days
    StorePath:         "/var/lib/kubestack/memory",
}

manager, err := memory.NewMemoryManager(cfg)
```

### Environment-Specific Configurations

**Development Environment**
```go
cfg := memory.MemoryConfig{
    WorkingWindowSize: 10,
    ShortTermTTL:      24 * time.Hour,
    StorePath:         "./tmp/memory",
}
```

**Production Environment**
```go
cfg := memory.MemoryConfig{
    WorkingWindowSize: 50,
    ShortTermTTL:      30 * 24 * time.Hour,
    StorePath:         "/var/lib/kubestack/memory",
}
```

**Memory-Constrained Environment**
```go
cfg := memory.MemoryConfig{
    WorkingWindowSize: 5,
    ShortTermTTL:      24 * time.Hour,
    StorePath:         "./data/memory",
}
```

## Common Operations

### Session Management

#### Save Session
```go
// Manually save working memory to persistent storage
err := manager.SaveSession(sessionID)
```

#### Load Session
```go
// Load a previous session
err := manager.LoadSession(sessionID)
```

#### Clear Working Memory
```go
// Start fresh (doesn't affect persistent storage)
manager.ClearWorking()
```

### Context Retrieval

#### Get All Context
```go
// Get all messages in context
context, err := manager.GetContext(sessionID, 0)
```

#### Get Context with Token Limit
```go
// Get messages within ~500 token budget
context, err := manager.GetContext(sessionID, 500)
```

### Recording Messages

#### Record with Metadata
```go
entry := memory.MemoryEntry{
    Role:    "user",
    Content: "Diagnose MySQL performance issues",
    Metadata: map[string]interface{}{
        "intent": "diagnose",
        "target": "mysql",
        "priority": "high",
    },
}
manager.RecordMessage(sessionID, entry)
```

#### Record System Messages
```go
systemEntry := memory.MemoryEntry{
    Role:    "system",
    Content: "Alert: High CPU usage detected on mysql-primary",
}
manager.RecordMessage(sessionID, systemEntry)
```

## Best Practices

### 1. Always Close Resources

```go
manager, err := memory.NewMemoryManager(cfg)
if err != nil {
    return err
}
defer manager.Close()  // Ensures proper cleanup
```

### 2. Use Session IDs Consistently

```go
// Use user-specific or request-specific session IDs
sessionID := fmt.Sprintf("user-%s-session-%d", userID, time.Now().Unix())
```

### 3. Handle Errors Gracefully

```go
context, err := manager.GetContext(sessionID, 0)
if err != nil {
    // Session might not exist yet - that's OK
    context = []memory.MemoryEntry{}
}
```

### 4. Save Sessions Before Shutdown

```go
// In shutdown handler
func shutdown() {
    for _, sessionID := range activeSessions {
        manager.SaveSession(sessionID)
    }
    manager.Close()
}
```

### 5. Monitor Memory Usage

```go
// Check working memory size
size := manager.working.Size()
if size > cfg.WorkingWindowSize*0.9 {
    log.Printf("Working memory nearly full: %d/%d", size, cfg.WorkingWindowSize)
}
```

## Troubleshooting

### Issue: Cannot acquire directory lock

**Problem:** Another process is using the same storage directory.

**Solution:**
```go
// Use unique storage paths for different instances
cfg := memory.MemoryConfig{
    StorePath: fmt.Sprintf("./data/memory-%d", os.Getpid()),
    // ... other config
}
```

### Issue: Memory not persisting

**Problem:** Not calling `Close()` or `SaveSession()`.

**Solution:**
```go
// Always defer Close()
defer manager.Close()

// Or explicitly save before exit
manager.SaveSession(sessionID)
```

### Issue: High memory usage

**Problem:** Working window size too large.

**Solution:**
```go
cfg := memory.MemoryConfig{
    WorkingWindowSize: 10,  // Reduce window size
    // ... other config
}
```

### Issue: Session data growing too large

**Problem:** Long-running sessions with many messages.

**Solution:**
```go
// Periodically clear old sessions
if sessionAge > 7*24*time.Hour {
    manager.ClearWorking()
    // Start new session
}
```

## Integration Examples

### With Web Server

```go
func handleChatRequest(w http.ResponseWriter, r *http.Request) {
    sessionID := r.Header.Get("X-Session-ID")
    if sessionID == "" {
        sessionID = uuid.New().String()
    }
    
    var req ChatRequest
    json.NewDecoder(r.Body).Decode(&req)
    
    // Load existing session
    memoryManager.LoadSession(sessionID)
    
    // Process with agent
    response, err := agent.ProcessUserInput(r.Context(), &agent.UserInput{
        Text:      req.Message,
        SessionID: sessionID,
        UserID:    req.UserID,
    })
    
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    json.NewEncoder(w).Encode(response)
}
```

### With CLI

```go
func main() {
    sessionID := "cli-session-" + time.Now().Format("20060102")
    
    memoryManager, _ := memory.NewMemoryManager(memory.DefaultMemoryConfig())
    defer memoryManager.Close()
    
    // Load previous CLI session if exists
    memoryManager.LoadSession(sessionID)
    
    for {
        fmt.Print("> ")
        reader := bufio.NewReader(os.Stdin)
        input, _ := reader.ReadString('\n')
        
        if input == "exit\n" {
            break
        }
        
        // Record and process
        entry := memory.MemoryEntry{
            Role:    "user",
            Content: input,
        }
        memoryManager.RecordMessage(sessionID, entry)
        
        // ... process and respond ...
    }
    
    // Save before exit
    memoryManager.SaveSession(sessionID)
}
```

### With Background Jobs

```go
func backgroundDiagnosis(sessionID string) {
    // Load session context
    context, _ := memoryManager.GetContext(sessionID, 0)
    
    // Perform diagnosis based on context
    result := performDiagnosis(context)
    
    // Record result
    entry := memory.MemoryEntry{
        Role:    "system",
        Content: fmt.Sprintf("Background diagnosis: %s", result),
        Metadata: map[string]interface{}{
            "type": "background_diagnosis",
            "timestamp": time.Now(),
        },
    }
    memoryManager.RecordMessage(sessionID, entry)
}
```

## API Reference

See [design-memory-system.md](design-memory-system.md) for complete API documentation.

## Performance Tips

1. **Use appropriate window sizes** - Don't over-allocate
2. **Set reasonable TTLs** - Balance storage vs. utility
3. **Use SSD storage** - For high-throughput scenarios
4. **Monitor disk usage** - Set up cleanup policies
5. **Batch operations** - When recording multiple messages

## Migration from Non-Memory Systems

If upgrading from an Agent without memory support:

```go
// Before (no memory)
agent := agent.NewAgent(nlpProcessor)

// After (with memory)
memoryManager, _ := memory.NewMemoryManager(memory.DefaultMemoryConfig())
agent := agent.NewAgent(nlpProcessor, memoryManager)
defer agent.Close()
```

All existing functionality remains compatible. The Agent will handle memory automatically.
