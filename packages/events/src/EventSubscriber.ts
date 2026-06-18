import axios, { AxiosInstance } from 'axios';
import { CloudEvent, EventSubscriberConfig } from './types';

export class EventSubscriber {
  private client: AxiosInstance;
  private eventBusUrl: string;
  private eventTypeFilter: string;
  private sourceFilter?: string;
  private isSubscribed: boolean = false;

  constructor(config: EventSubscriberConfig) {
    this.eventBusUrl = config.eventBusUrl;
    this.eventTypeFilter = config.eventTypeFilter || '*';
    this.sourceFilter = config.sourceFilter;

    this.client = axios.create({
      baseURL: config.eventBusUrl,
      timeout: 60000,
      headers: {
        'Content-Type': 'application/json',
      },
    });
  }

  async subscribe(
    onEvent: (event: CloudEvent) => void | Promise<void>,
    onError?: (error: Error) => void
  ): Promise<void> {
    this.isSubscribed = true;

    try {
      const response = await this.client.get('/api/v1/events/subscribe', {
        params: {
          eventType: this.eventTypeFilter,
          source: this.sourceFilter,
        },
        responseType: 'stream',
      });

      response.data.on('data', async (chunk: Buffer) => {
        try {
          const event = JSON.parse(chunk.toString());
          await onEvent(event);
        } catch (error) {
          if (onError) {
            onError(error instanceof Error ? error : new Error(String(error)));
          }
        }
      });

      response.data.on('end', () => {
        this.isSubscribed = false;
      });

      response.data.on('error', (error: Error) => {
        this.isSubscribed = false;
        if (onError) {
          onError(error);
        }
      });
    } catch (error) {
      this.isSubscribed = false;
      throw new Error(`Failed to subscribe to events: ${error instanceof Error ? error.message : String(error)}`);
    }
  }

  async unsubscribe(): Promise<void> {
    this.isSubscribed = false;
  }

  isConnected(): boolean {
    return this.isSubscribed;
  }

  async getEventTypes(): Promise<string[]> {
    try {
      const response = await this.client.get('/api/v1/events/types', {
        params: {
          source: this.sourceFilter,
        },
      });
      return response.data.eventTypes;
    } catch (error) {
      throw new Error(`Failed to get event types: ${error instanceof Error ? error.message : String(error)}`);
    }
  }

  async getEventHistory(
    limit: number = 100,
    offset: number = 0
  ): Promise<{ events: CloudEvent[]; total: number }> {
    try {
      const response = await this.client.get('/api/v1/events/history', {
        params: {
          eventType: this.eventTypeFilter,
          source: this.sourceFilter,
          limit,
          offset,
        },
      });
      return response.data;
    } catch (error) {
      throw new Error(`Failed to get event history: ${error instanceof Error ? error.message : String(error)}`);
    }
  }
}
