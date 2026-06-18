import axios, { AxiosInstance } from 'axios';
import { CloudEvent, EventPublisherConfig } from './types';
import { CloudEventValidator } from './CloudEvent';

export class EventPublisher {
  private client: AxiosInstance;
  private eventBusUrl: string;

  constructor(config: EventPublisherConfig) {
    this.eventBusUrl = config.eventBusUrl;
    this.client = axios.create({
      baseURL: config.eventBusUrl,
      timeout: config.timeout || 30000,
      headers: {
        'Content-Type': 'application/json',
      },
    });
  }

  async publishEvent(event: CloudEvent): Promise<{ eventId: string; partitionKey: string; offset: number }> {
    const validation = CloudEventValidator.validate(event);
    if (!validation.valid) {
      throw new Error(`Invalid CloudEvent: ${validation.errors.join(', ')}`);
    }

    try {
      const response = await this.client.post('/api/v1/events', event);
      return {
        eventId: response.data.eventId,
        partitionKey: response.data.partitionKey,
        offset: response.data.offset,
      };
    } catch (error) {
      throw new Error(`Failed to publish event: ${error instanceof Error ? error.message : String(error)}`);
    }
  }

  async publishEventBatch(events: CloudEvent[]): Promise<Array<{ eventId: string; offset: number }>> {
    for (const event of events) {
      const validation = CloudEventValidator.validate(event);
      if (!validation.valid) {
        throw new Error(`Invalid CloudEvent: ${validation.errors.join(', ')}`);
      }
    }

    try {
      const response = await this.client.post('/api/v1/events/batch', { events });
      return response.data.results;
    } catch (error) {
      throw new Error(`Failed to publish event batch: ${error instanceof Error ? error.message : String(error)}`);
    }
  }

  async getEventSchema(eventType: string): Promise<any> {
    try {
      const response = await this.client.get(`/api/v1/events/schema/${eventType}`);
      return response.data;
    } catch (error) {
      throw new Error(`Failed to get event schema: ${error instanceof Error ? error.message : String(error)}`);
    }
  }

  async listEventTypes(sourceFilter?: string): Promise<string[]> {
    try {
      const params = sourceFilter ? { source: sourceFilter } : {};
      const response = await this.client.get('/api/v1/events/types', { params });
      return response.data.eventTypes;
    } catch (error) {
      throw new Error(`Failed to list event types: ${error instanceof Error ? error.message : String(error)}`);
    }
  }

  async getEventMetrics(): Promise<{
    eventsPublished: number;
    eventsConsumed: number;
    webhookDeliveries: number;
    failedDeliveries: number;
  }> {
    try {
      const response = await this.client.get('/metrics');
      return response.data;
    } catch (error) {
      throw new Error(`Failed to get metrics: ${error instanceof Error ? error.message : String(error)}`);
    }
  }
}
