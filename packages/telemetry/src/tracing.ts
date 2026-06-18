import { trace, context, SpanStatusCode } from '@opentelemetry/api';
import { NodeSDK } from '@opentelemetry/sdk-node';
import { getNodeAutoInstrumentations } from '@opentelemetry/auto-instrumentations-node';
import { JaegerExporter } from '@opentelemetry/exporter-jaeger-thrift';
import { Resource } from '@opentelemetry/resources';
import { SemanticResourceAttributes } from '@opentelemetry/semantic-conventions';

export interface TracingConfig {
  serviceName: string;
  jaegerHost?: string;
  jaegerPort?: number;
  jaegerMaxPacketSize?: number;
  samplingRate?: number;
  environment?: string;
}

export class TracingCollector {
  private sdk: NodeSDK;
  private tracer: any;

  constructor(config: TracingConfig) {
    const resource = Resource.default().merge(
      new Resource({
        [SemanticResourceAttributes.SERVICE_NAME]: config.serviceName,
        environment: config.environment || 'development',
      })
    );

    const jaegerExporter = new JaegerExporter({
      host: config.jaegerHost || 'localhost',
      port: config.jaegerPort || 6832,
      maxPacketSize: config.jaegerMaxPacketSize || 65000,
    });

    this.sdk = new NodeSDK({
      resource,
      traceExporter: jaegerExporter,
      instrumentations: [getNodeAutoInstrumentations()],
      serviceName: config.serviceName,
    });

    this.sdk.start();
    this.tracer = trace.getTracer('titan-os', '1.0.0');

    console.log(`✓ Tracing initialized (Jaeger: ${config.jaegerHost}:${config.jaegerPort})`);
  }

  startSpan(name: string, attributes?: Record<string, string | number>) {
    const span = this.tracer.startSpan(name, {
      attributes: attributes || {},
    });
    return span;
  }

  async runWithSpan<T>(name: string, fn: (span: any) => Promise<T>, attributes?: Record<string, string | number>): Promise<T> {
    const span = this.startSpan(name, attributes);

    try {
      const result = await context.with(trace.setSpan(context.active(), span), async () => {
        return fn(span);
      });

      span.setStatus({ code: SpanStatusCode.OK });
      return result;
    } catch (error) {
      span.setStatus({
        code: SpanStatusCode.ERROR,
        message: error instanceof Error ? error.message : String(error),
      });
      span.recordException(error instanceof Error ? error : new Error(String(error)));
      throw error;
    } finally {
      span.end();
    }
  }

  // Convenience methods for common operations

  async traceTaskExecution(taskId: string, fn: () => Promise<any>) {
    return this.runWithSpan(`task.execute`, fn, {
      'task.id': taskId,
    });
  }

  async traceWorkflowExecution(workflowId: string, executionId: string, fn: () => Promise<any>) {
    return this.runWithSpan(
      `workflow.execute`,
      fn,
      {
        'workflow.id': workflowId,
        'execution.id': executionId,
      }
    );
  }

  async traceAPICall(method: string, path: string, fn: () => Promise<any>) {
    return this.runWithSpan(`http.${method}`, fn, {
      'http.method': method,
      'http.url': path,
    });
  }

  async traceDatabaseQuery(query: string, fn: () => Promise<any>) {
    return this.runWithSpan(`db.query`, fn, {
      'db.statement': query.substring(0, 100),
    });
  }

  async traceEventPublish(eventType: string, fn: () => Promise<any>) {
    return this.runWithSpan(`event.publish`, fn, {
      'event.type': eventType,
    });
  }

  async shutdown() {
    await this.sdk.shutdown();
  }
}

export default TracingCollector;
