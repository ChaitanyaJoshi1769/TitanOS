import { metrics } from '@opentelemetry/api';
import { MeterProvider, PeriodicExportingMetricReader } from '@opentelemetry/sdk-metrics';
import { PrometheusExporter } from '@opentelemetry/exporter-prometheus';
import { Resource } from '@opentelemetry/resources';
import { SemanticResourceAttributes } from '@opentelemetry/semantic-conventions';

export interface MetricsConfig {
  serviceName: string;
  version?: string;
  prometheusPort?: number;
  environment?: string;
}

export class MetricsCollector {
  private meterProvider: MeterProvider;
  private exporter: PrometheusExporter;

  // Counters
  private requestsTotal: any;
  private requestsSuccess: any;
  private requestsFailed: any;
  private tasksSubmitted: any;
  private tasksCompleted: any;
  private tasksFailed: any;
  private workflowsStarted: any;
  private workflowsCompleted: any;
  private workflowsFailed: any;

  // Histograms
  private requestDuration: any;
  private taskExecutionTime: any;
  private workflowExecutionTime: any;

  // Gauges
  private activeConnections: any;
  private queueSize: any;
  private nodeHealth: any;

  constructor(config: MetricsConfig) {
    const resource = Resource.default().merge(
      new Resource({
        [SemanticResourceAttributes.SERVICE_NAME]: config.serviceName,
        [SemanticResourceAttributes.SERVICE_VERSION]: config.version || '1.0.0',
        environment: config.environment || 'development',
      })
    );

    this.exporter = new PrometheusExporter(
      {
        port: config.prometheusPort || 8888,
        defaultHistogramBuckets: [0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10],
      },
      () => {
        console.log(`✓ Prometheus metrics exporter started on port ${config.prometheusPort || 8888}`);
      }
    );

    this.meterProvider = new MeterProvider({
      resource,
      readers: [this.exporter],
    });

    metrics.setGlobalMeterProvider(this.meterProvider);
    this.initializeMetrics();
  }

  private initializeMetrics() {
    const meter = metrics.getMeter('titan-os', '1.0.0');

    // Counters
    this.requestsTotal = meter.createCounter('titan_requests_total', {
      description: 'Total number of API requests',
      unit: '1',
    });

    this.requestsSuccess = meter.createCounter('titan_requests_success', {
      description: 'Total number of successful API requests',
      unit: '1',
    });

    this.requestsFailed = meter.createCounter('titan_requests_failed', {
      description: 'Total number of failed API requests',
      unit: '1',
    });

    this.tasksSubmitted = meter.createCounter('titan_tasks_submitted_total', {
      description: 'Total number of tasks submitted',
      unit: '1',
    });

    this.tasksCompleted = meter.createCounter('titan_tasks_completed_total', {
      description: 'Total number of tasks completed',
      unit: '1',
    });

    this.tasksFailed = meter.createCounter('titan_tasks_failed_total', {
      description: 'Total number of failed tasks',
      unit: '1',
    });

    this.workflowsStarted = meter.createCounter('titan_workflows_started_total', {
      description: 'Total number of workflows started',
      unit: '1',
    });

    this.workflowsCompleted = meter.createCounter('titan_workflows_completed_total', {
      description: 'Total number of workflows completed',
      unit: '1',
    });

    this.workflowsFailed = meter.createCounter('titan_workflows_failed_total', {
      description: 'Total number of failed workflows',
      unit: '1',
    });

    // Histograms
    this.requestDuration = meter.createHistogram('titan_request_duration_seconds', {
      description: 'Request duration in seconds',
      unit: 's',
    });

    this.taskExecutionTime = meter.createHistogram('titan_task_execution_time_seconds', {
      description: 'Task execution time in seconds',
      unit: 's',
    });

    this.workflowExecutionTime = meter.createHistogram('titan_workflow_execution_time_seconds', {
      description: 'Workflow execution time in seconds',
      unit: 's',
    });

    // Gauges
    this.activeConnections = meter.createObservableGauge('titan_active_connections', {
      description: 'Number of active connections',
      unit: '1',
    });

    this.queueSize = meter.createObservableGauge('titan_queue_size', {
      description: 'Current task queue size',
      unit: '1',
    });

    this.nodeHealth = meter.createObservableGauge('titan_node_health', {
      description: 'Node health status (1=healthy, 0=unhealthy)',
      unit: '1',
    });
  }

  // Counter methods
  recordRequestTotal(labels?: Record<string, string>) {
    this.requestsTotal.add(1, labels);
  }

  recordRequestSuccess(labels?: Record<string, string>) {
    this.requestsSuccess.add(1, labels);
  }

  recordRequestFailed(labels?: Record<string, string>) {
    this.requestsFailed.add(1, labels);
  }

  recordTaskSubmitted(labels?: Record<string, string>) {
    this.tasksSubmitted.add(1, labels);
  }

  recordTaskCompleted(labels?: Record<string, string>) {
    this.tasksCompleted.add(1, labels);
  }

  recordTaskFailed(labels?: Record<string, string>) {
    this.tasksFailed.add(1, labels);
  }

  recordWorkflowStarted(labels?: Record<string, string>) {
    this.workflowsStarted.add(1, labels);
  }

  recordWorkflowCompleted(labels?: Record<string, string>) {
    this.workflowsCompleted.add(1, labels);
  }

  recordWorkflowFailed(labels?: Record<string, string>) {
    this.workflowsFailed.add(1, labels);
  }

  // Histogram methods
  recordRequestDuration(duration: number, labels?: Record<string, string>) {
    this.requestDuration.record(duration, labels);
  }

  recordTaskExecutionTime(duration: number, labels?: Record<string, string>) {
    this.taskExecutionTime.record(duration, labels);
  }

  recordWorkflowExecutionTime(duration: number, labels?: Record<string, string>) {
    this.workflowExecutionTime.record(duration, labels);
  }

  // Utility methods
  async shutdown() {
    await this.meterProvider.shutdown();
  }
}

export default MetricsCollector;
