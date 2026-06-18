import pino, { Logger as PinoLogger } from 'pino';
import { trace } from '@opentelemetry/api';

export interface LoggingConfig {
  serviceName: string;
  level?: string;
  environment?: string;
  pretty?: boolean;
}

export interface LogContext {
  traceId?: string;
  spanId?: string;
  userId?: string;
  requestId?: string;
  [key: string]: any;
}

export class Logger {
  private logger: PinoLogger;
  private serviceName: string;
  private context: LogContext;

  constructor(config: LoggingConfig) {
    this.serviceName = config.serviceName;
    this.context = {};

    const transport = config.pretty
      ? {
          target: 'pino-pretty',
          options: {
            colorize: true,
            levelFirst: true,
            singleLine: false,
          },
        }
      : undefined;

    this.logger = pino(
      {
        name: config.serviceName,
        level: config.level || 'info',
        base: {
          service: config.serviceName,
          environment: config.environment || 'development',
        },
        timestamp: pino.stdTimeFunctions.isoTime,
      },
      transport ? pino.transport(transport) : undefined
    );

    console.log(`✓ Logger initialized for service: ${config.serviceName}`);
  }

  setContext(context: LogContext) {
    this.context = { ...this.context, ...context };
  }

  clearContext() {
    this.context = {};
  }

  getContext(): LogContext {
    const span = trace.getActiveSpan();
    const spanContext = span?.spanContext();

    return {
      ...this.context,
      traceId: spanContext?.traceId,
      spanId: spanContext?.spanId,
    };
  }

  private getMergedContext(additionalContext?: Record<string, any>) {
    return {
      ...this.getContext(),
      ...additionalContext,
    };
  }

  debug(message: string, context?: Record<string, any>) {
    this.logger.debug(this.getMergedContext(context), message);
  }

  info(message: string, context?: Record<string, any>) {
    this.logger.info(this.getMergedContext(context), message);
  }

  warn(message: string, context?: Record<string, any>) {
    this.logger.warn(this.getMergedContext(context), message);
  }

  error(message: string, error?: Error | string, context?: Record<string, any>) {
    const mergedContext = this.getMergedContext(context);

    if (error instanceof Error) {
      mergedContext.error = {
        message: error.message,
        stack: error.stack,
      };
    } else if (typeof error === 'string') {
      mergedContext.error = error;
    }

    this.logger.error(mergedContext, message);
  }

  fatal(message: string, error?: Error | string, context?: Record<string, any>) {
    const mergedContext = this.getMergedContext(context);

    if (error instanceof Error) {
      mergedContext.error = {
        message: error.message,
        stack: error.stack,
      };
    } else if (typeof error === 'string') {
      mergedContext.error = error;
    }

    this.logger.fatal(mergedContext, message);
  }

  // Structured logging methods

  logTaskSubmitted(taskId: string, projectId: string, metadata?: Record<string, any>) {
    this.info('Task submitted', {
      event: 'task.submitted',
      taskId,
      projectId,
      ...metadata,
    });
  }

  logTaskCompleted(taskId: string, duration: number, metadata?: Record<string, any>) {
    this.info('Task completed', {
      event: 'task.completed',
      taskId,
      duration,
      ...metadata,
    });
  }

  logTaskFailed(taskId: string, error: Error | string, metadata?: Record<string, any>) {
    this.error('Task failed', error, {
      event: 'task.failed',
      taskId,
      ...metadata,
    });
  }

  logWorkflowStarted(workflowId: string, executionId: string, metadata?: Record<string, any>) {
    this.info('Workflow started', {
      event: 'workflow.started',
      workflowId,
      executionId,
      ...metadata,
    });
  }

  logWorkflowCompleted(workflowId: string, executionId: string, duration: number, metadata?: Record<string, any>) {
    this.info('Workflow completed', {
      event: 'workflow.completed',
      workflowId,
      executionId,
      duration,
      ...metadata,
    });
  }

  logWorkflowFailed(workflowId: string, executionId: string, error: Error | string, metadata?: Record<string, any>) {
    this.error('Workflow failed', error, {
      event: 'workflow.failed',
      workflowId,
      executionId,
      ...metadata,
    });
  }

  logAPIRequest(method: string, path: string, statusCode: number, duration: number, metadata?: Record<string, any>) {
    this.info('API request', {
      event: 'http.request',
      method,
      path,
      statusCode,
      duration,
      ...metadata,
    });
  }

  logDatabaseQuery(query: string, duration: number, rowsAffected?: number, error?: Error | string) {
    if (error) {
      this.error('Database query failed', error, {
        event: 'db.query',
        query: query.substring(0, 100),
        duration,
      });
    } else {
      this.debug('Database query', {
        event: 'db.query',
        query: query.substring(0, 100),
        duration,
        rowsAffected,
      });
    }
  }

  logSecurityEvent(eventType: string, details: Record<string, any>) {
    this.warn('Security event', {
      event: `security.${eventType}`,
      ...details,
    });
  }

  logAuditEvent(action: string, resource: string, userId: string, details?: Record<string, any>) {
    this.info('Audit event', {
      event: 'audit',
      action,
      resource,
      userId,
      ...details,
    });
  }
}

export default Logger;
