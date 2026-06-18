export interface CloudEvent {
  specversion: string;
  type: string;
  source: string;
  id: string;
  time: string;
  datacontenttype: string;
  subject?: string;
  dataschema?: string;
  data?: Record<string, any>;
  attributes?: Record<string, string>;
}

export interface EventPublisherConfig {
  eventBusUrl: string;
  timeout?: number;
}

export interface EventSubscriberConfig {
  eventBusUrl: string;
  eventTypeFilter?: string;
  sourceFilter?: string;
  consumerGroupId?: string;
}

export interface WebhookSubscription {
  subscriptionId: string;
  userId: string;
  projectId: string;
  webhookUrl: string;
  eventTypeFilter: string;
  sourceFilter?: string;
  subjectFilter?: string;
  active: boolean;
  signatureAlgorithm: string;
  retryPolicy: RetryPolicy;
  createdAt: string;
  updatedAt: string;
}

export interface RetryPolicy {
  maxRetries: number;
  initialDelayMs: number;
  maxDelayMs: number;
  backoffMultiplier: number;
}

export interface WebhookDelivery {
  deliveryId: string;
  subscriptionId: string;
  eventId: string;
  attemptNumber: number;
  status: 'pending' | 'delivered' | 'failed' | 'dlq';
  httpStatus?: number;
  deliveredAt: string;
  latencyMs?: number;
  errorMessage?: string;
}

// Task Events
export interface TaskSubmittedEvent {
  taskId: string;
  projectId: string;
  name: string;
  labels?: Record<string, string>;
}

export interface TaskScheduledEvent {
  taskId: string;
  nodeId: string;
  scheduledAt: string;
}

export interface TaskCompletedEvent {
  taskId: string;
  nodeId: string;
  completedAt: string;
  exitCode: number;
  output?: string;
}

export interface TaskFailedEvent {
  taskId: string;
  nodeId: string;
  failedAt: string;
  errorMessage: string;
}

// Workflow Events
export interface WorkflowExecutionStartedEvent {
  workflowId: string;
  executionId: string;
  startedAt: string;
  input?: Record<string, any>;
}

export interface WorkflowExecutionCompletedEvent {
  workflowId: string;
  executionId: string;
  completedAt: string;
  output?: Record<string, any>;
}

export interface ActivityExecutedEvent {
  workflowId: string;
  executionId: string;
  activityId: string;
  executedAt: string;
  result?: Record<string, any>;
}

// Agent Events
export interface AgentCreatedEvent {
  agentId: string;
  userId: string;
  createdAt: string;
  config?: Record<string, any>;
}

export interface AgentWokenEvent {
  agentId: string;
  wokenAt: string;
}

export interface AgentSleptEvent {
  agentId: string;
  sleptAt: string;
}

export interface AgentToolExecutedEvent {
  agentId: string;
  toolName: string;
  executedAt: string;
  input?: Record<string, any>;
  output?: Record<string, any>;
}

// Node Events
export interface NodeHeartbeatEvent {
  nodeId: string;
  timestamp: string;
  cpuUsage: number;
  memoryUsage: number;
  runningTasks: number;
}

export interface NodeHealthEvent {
  nodeId: string;
  status: 'healthy' | 'degraded' | 'down';
  timestamp: string;
  message?: string;
}
