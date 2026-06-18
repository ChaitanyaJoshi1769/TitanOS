import axios, { AxiosInstance } from 'axios'

export interface TitanOSConfig {
  apiUrl: string
  token?: string
  timeout?: number
}

export interface Task {
  id: string
  status: 'pending' | 'running' | 'completed' | 'failed'
  input: Record<string, any>
  output?: Record<string, any>
  error?: string
  createdAt: string
  completedAt?: string
}

export interface Workflow {
  id: string
  name: string
  description?: string
  definition: Record<string, any>
  createdAt: string
}

export interface Agent {
  id: string
  name: string
  status: 'online' | 'offline' | 'idle'
  createdAt: string
  lastSeen?: string
}

export class TitanOSClient {
  private client: AxiosInstance
  private config: TitanOSConfig

  constructor(config: TitanOSConfig) {
    this.config = config
    this.client = axios.create({
      baseURL: config.apiUrl,
      timeout: config.timeout || 30000,
    })

    if (config.token) {
      this.client.defaults.headers.common['Authorization'] = config.token
    }
  }

  // Task methods
  async submitTask(input: Record<string, any>): Promise<Task> {
    const response = await this.client.post('/api/v1/tasks', input)
    return response.data
  }

  async getTask(taskId: string): Promise<Task> {
    const response = await this.client.get(`/api/v1/tasks/${taskId}`)
    return response.data
  }

  async listTasks(limit: number = 100): Promise<Task[]> {
    const response = await this.client.get('/api/v1/tasks', { params: { limit } })
    return response.data.tasks
  }

  async waitForTask(taskId: string, timeout: number = 300000): Promise<Task> {
    const startTime = Date.now()
    while (Date.now() - startTime < timeout) {
      const task = await this.getTask(taskId)
      if (task.status === 'completed' || task.status === 'failed') {
        return task
      }
      await new Promise(r => setTimeout(r, 1000))
    }
    throw new Error(`Task ${taskId} timed out`)
  }

  // Workflow methods
  async createWorkflow(name: string, definition: Record<string, any>): Promise<Workflow> {
    const response = await this.client.post('/api/v1/workflows', { name, definition })
    return response.data
  }

  async getWorkflow(workflowId: string): Promise<Workflow> {
    const response = await this.client.get(`/api/v1/workflows/${workflowId}`)
    return response.data
  }

  async executeWorkflow(workflowId: string, input: Record<string, any>): Promise<{ executionId: string }> {
    const response = await this.client.post(`/api/v1/workflows/${workflowId}/execute`, input)
    return response.data
  }

  // Agent methods
  async createAgent(name: string, config?: Record<string, any>): Promise<Agent> {
    const response = await this.client.post('/api/v1/agents', { name, ...config })
    return response.data
  }

  async getAgent(agentId: string): Promise<Agent> {
    const response = await this.client.get(`/api/v1/agents/${agentId}`)
    return response.data
  }

  async listAgents(): Promise<Agent[]> {
    const response = await this.client.get('/api/v1/agents')
    return response.data.agents
  }

  async executeAgentTool(agentId: string, toolName: string, input: Record<string, any>): Promise<any> {
    const response = await this.client.post(
      `/api/v1/agents/${agentId}/tools/${toolName}`,
      input
    )
    return response.data
  }

  // Health check
  async healthCheck(): Promise<{ status: string }> {
    const response = await this.client.get('/health')
    return response.data
  }

  // Metrics
  async getMetrics(): Promise<Record<string, any>> {
    const response = await this.client.get('/metrics')
    return response.data
  }
}

export default TitanOSClient
