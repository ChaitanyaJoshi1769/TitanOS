# Titan OS SDK Guide

## Overview

Titan OS provides SDKs for TypeScript/JavaScript, Python, and Go to interact with the platform.

## TypeScript/JavaScript SDK

### Installation

```bash
npm install @titanos/sdk-typescript
```

### Basic Usage

```typescript
import { TitanOSClient } from '@titanos/sdk-typescript'

const client = new TitanOSClient({
  apiUrl: 'http://localhost:8000',
  token: 'your-api-token',
})

// Submit a task
const task = await client.submitTask({ 
  data: 'input data' 
})

// Wait for completion
const completed = await client.waitForTask(task.id)
console.log('Result:', completed.output)
```

### Task Management

```typescript
// Submit task
const task = await client.submitTask({ /* input */ })

// Get task status
const task = await client.getTask(taskId)

// List all tasks
const tasks = await client.listTasks(100)

// Wait for task completion
const result = await client.waitForTask(taskId, 300000) // 5 minute timeout
```

### Workflow Management

```typescript
// Create workflow
const workflow = await client.createWorkflow('my-workflow', {
  steps: [
    { name: 'step1', action: 'process' },
    { name: 'step2', action: 'store' }
  ]
})

// Execute workflow
const execution = await client.executeWorkflow(workflowId, {
  input: 'data'
})
```

### Agent Management

```typescript
// Create agent
const agent = await client.createAgent('my-agent')

// Get agent
const agent = await client.getAgent(agentId)

// List agents
const agents = await client.listAgents()

// Execute agent tool
const result = await client.executeAgentTool(agentId, 'tool-name', {
  param1: 'value'
})
```

## CLI Tool

### Installation

```bash
npm install -g @titanos/cli
```

### Commands

```bash
# Task commands
titan task:submit '{"data":"value"}'
titan task:status <taskId>
titan task:list --limit 50

# Agent commands
titan agent:create my-agent
titan agent:list

# Workflow commands
titan workflow:create my-flow definition.json
titan workflow:execute <workflowId> '{"input":"data"}'

# Health check
titan health
```

## Python SDK

### Installation

```bash
pip install titan-os
```

### Usage

```python
from titan import TitanOS

client = TitanOS(
    api_url='http://localhost:8000',
    token='your-api-token'
)

# Submit task
task = client.submit_task(data={'key': 'value'})

# Wait for completion
result = client.wait_for_task(task['id'])
print(result['output'])
```

## Go SDK

### Installation

```bash
go get github.com/ChaitanyaJoshi1769/TitanOS/packages/sdk-go
```

### Usage

```go
package main

import (
    "context"
    "log"
    "github.com/ChaitanyaJoshi1769/TitanOS/packages/sdk-go"
)

func main() {
    client, _ := sdk.NewClient(
        "http://localhost:8000",
        "your-api-token",
    )
    
    ctx := context.Background()
    
    // Submit task
    task, _ := client.SubmitTask(ctx, map[string]interface{}{
        "data": "value",
    })
    
    // Wait for completion
    result, _ := client.WaitForTask(ctx, task.ID)
    log.Printf("Result: %v", result)
}
```

## Error Handling

```typescript
try {
  const task = await client.submitTask({ data: 'test' })
} catch (error) {
  if (error.response?.status === 401) {
    console.error('Authentication failed')
  } else if (error.response?.status === 429) {
    console.error('Rate limited')
  } else {
    console.error('Error:', error.message)
  }
}
```

## Authentication

Set token via environment variable:

```bash
export TITAN_TOKEN="your-api-token"
```

Or pass directly:

```typescript
const client = new TitanOSClient({
  apiUrl: 'http://localhost:8000',
  token: 'your-token'
})
```

## Rate Limiting

The SDKs include automatic retry logic with exponential backoff:

```typescript
// Automatic retry: 1s, 2s, 4s, 8s delays
const task = await client.submitTask(data)
```

## Batch Operations

```typescript
const tasks = await Promise.all([
  client.submitTask(data1),
  client.submitTask(data2),
  client.submitTask(data3),
])
```

## Examples

See `docs/examples/` directory for complete examples.
