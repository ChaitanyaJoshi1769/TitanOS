import axios, { AxiosInstance } from "axios";

/**
 * SchedulerClient - TypeScript SDK for submitting tasks to Titan OS Scheduler
 */

export interface SubmitTaskRequest {
  taskId: string;
  projectId: string;
  name: string;
  inputData?: Buffer | string;
  timeoutSeconds?: number;
  priority?: number;
  maxRetries?: number;
  labels?: Record<string, string>;
}

export interface SubmitTaskResponse {
  taskId: string;
  success: boolean;
  message: string;
}

export interface GetTaskResponse {
  id: string;
  projectId: string;
  nodeId?: string;
  name: string;
  status: string;
  inputData?: Buffer;
  outputData?: Buffer;
  errorMessage?: string;
  retryCount: number;
  maxRetries: number;
  timeoutSeconds: number;
  priority: number;
  createdAt: Date;
  startedAt?: Date;
  completedAt?: Date;
  updatedAt: Date;
}

export interface ListTasksRequest {
  projectId: string;
  status?: string;
  limit?: number;
  offset?: number;
}

export interface ListTasksResponse {
  tasks: GetTaskResponse[];
  total: number;
}

export interface NodeInfo {
  id: string;
  name: string;
  hostname: string;
  status: string;
  cpuCores: number;
  memoryGb: number;
  gpuCount: number;
  diskGb: number;
  lastHeartbeat: Date;
  createdAt: Date;
  updatedAt: Date;
}

export interface ListNodesResponse {
  nodes: NodeInfo[];
  total: number;
}

export class SchedulerClient {
  private apiClient: AxiosInstance;
  private baseURL: string;

  constructor(schedulerURL: string = "http://localhost:8000") {
    this.baseURL = schedulerURL;
    this.apiClient = axios.create({
      baseURL: this.baseURL,
      timeout: 10000,
      headers: {
        "Content-Type": "application/json",
      },
    });
  }

  /**
   * Submit a task to the scheduler
   */
  async submitTask(request: SubmitTaskRequest): Promise<SubmitTaskResponse> {
    try {
      let inputData: string | undefined;
      if (request.inputData) {
        inputData =
          typeof request.inputData === "string"
            ? request.inputData
            : request.inputData.toString("base64");
      }

      const response = await this.apiClient.post<SubmitTaskResponse>(
        "/api/v1/tasks",
        {
          taskId: request.taskId,
          projectId: request.projectId,
          name: request.name,
          inputData,
          timeoutSeconds: request.timeoutSeconds || 300,
          priority: request.priority || 0,
          maxRetries: request.maxRetries || 3,
          labels: request.labels || {},
        }
      );

      return response.data;
    } catch (error) {
      throw new Error(`Failed to submit task: ${this._getErrorMessage(error)}`);
    }
  }

  /**
   * Get a task by ID
   */
  async getTask(taskId: string): Promise<GetTaskResponse> {
    try {
      const response = await this.apiClient.get<GetTaskResponse>(
        `/api/v1/tasks/${taskId}`
      );
      return this._parseTaskResponse(response.data);
    } catch (error) {
      throw new Error(`Failed to get task: ${this._getErrorMessage(error)}`);
    }
  }

  /**
   * List tasks with filtering
   */
  async listTasks(request: ListTasksRequest): Promise<ListTasksResponse> {
    try {
      const params = new URLSearchParams({
        projectId: request.projectId,
        limit: (request.limit || 100).toString(),
        offset: (request.offset || 0).toString(),
      });

      if (request.status) {
        params.append("status", request.status);
      }

      const response = await this.apiClient.get<ListTasksResponse>(
        "/api/v1/tasks",
        { params }
      );

      return {
        tasks: response.data.tasks.map((task) => this._parseTaskResponse(task)),
        total: response.data.total,
      };
    } catch (error) {
      throw new Error(`Failed to list tasks: ${this._getErrorMessage(error)}`);
    }
  }

  /**
   * Get a node by ID
   */
  async getNode(nodeId: string): Promise<NodeInfo> {
    try {
      const response = await this.apiClient.get<NodeInfo>(
        `/api/v1/nodes/${nodeId}`
      );
      return this._parseNodeResponse(response.data);
    } catch (error) {
      throw new Error(`Failed to get node: ${this._getErrorMessage(error)}`);
    }
  }

  /**
   * List all nodes
   */
  async listNodes(
    limit: number = 100,
    offset: number = 0
  ): Promise<ListNodesResponse> {
    try {
      const response = await this.apiClient.get<ListNodesResponse>(
        "/api/v1/nodes",
        {
          params: { limit, offset },
        }
      );

      return {
        nodes: response.data.nodes.map((node) => this._parseNodeResponse(node)),
        total: response.data.total,
      };
    } catch (error) {
      throw new Error(`Failed to list nodes: ${this._getErrorMessage(error)}`);
    }
  }

  /**
   * Wait for a task to complete
   */
  async waitForTask(
    taskId: string,
    maxWaitSeconds: number = 300,
    pollIntervalMs: number = 1000
  ): Promise<GetTaskResponse> {
    const startTime = Date.now();

    while (Date.now() - startTime < maxWaitSeconds * 1000) {
      const task = await this.getTask(taskId);

      if (task.status === "completed" || task.status === "failed") {
        return task;
      }

      await this._sleep(pollIntervalMs);
    }

    throw new Error(`Task ${taskId} did not complete within ${maxWaitSeconds}s`);
  }

  /**
   * Submit multiple tasks
   */
  async submitTasks(
    requests: SubmitTaskRequest[]
  ): Promise<SubmitTaskResponse[]> {
    return Promise.all(requests.map((req) => this.submitTask(req)));
  }

  /**
   * Get task status
   */
  async getTaskStatus(taskId: string): Promise<string> {
    const task = await this.getTask(taskId);
    return task.status;
  }

  // Helper methods

  private _parseTaskResponse(data: any): GetTaskResponse {
    return {
      ...data,
      createdAt: new Date(data.createdAt),
      startedAt: data.startedAt ? new Date(data.startedAt) : undefined,
      completedAt: data.completedAt ? new Date(data.completedAt) : undefined,
      updatedAt: new Date(data.updatedAt),
    };
  }

  private _parseNodeResponse(data: any): NodeInfo {
    return {
      ...data,
      lastHeartbeat: new Date(data.lastHeartbeat),
      createdAt: new Date(data.createdAt),
      updatedAt: new Date(data.updatedAt),
    };
  }

  private _getErrorMessage(error: any): string {
    if (axios.isAxiosError(error)) {
      return (
        error.response?.data?.message ||
        error.message ||
        "Unknown error"
      );
    }
    return String(error);
  }

  private _sleep(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }
}

/**
 * Convenience function to create a scheduler client
 */
export function createSchedulerClient(
  schedulerURL?: string
): SchedulerClient {
  return new SchedulerClient(schedulerURL);
}

/**
 * Generate a unique task ID
 */
export function generateTaskId(prefix: string = "task"): string {
  return `${prefix}-${Date.now()}-${Math.random()
    .toString(36)
    .substr(2, 9)}`;
}
