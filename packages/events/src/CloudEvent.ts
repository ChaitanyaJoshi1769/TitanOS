import { v4 as uuidv4 } from 'uuid';
import { CloudEvent } from './types';

export class CloudEventBuilder {
  private event: CloudEvent;

  constructor(type: string, source: string) {
    this.event = {
      specversion: '1.0',
      type,
      source,
      id: uuidv4(),
      time: new Date().toISOString(),
      datacontenttype: 'application/json',
      attributes: {},
    };
  }

  withId(id: string): CloudEventBuilder {
    this.event.id = id;
    return this;
  }

  withSubject(subject: string): CloudEventBuilder {
    this.event.subject = subject;
    return this;
  }

  withDataSchema(schema: string): CloudEventBuilder {
    this.event.dataschema = schema;
    return this;
  }

  withData(data: Record<string, any>): CloudEventBuilder {
    this.event.data = data;
    return this;
  }

  withAttribute(key: string, value: string): CloudEventBuilder {
    if (!this.event.attributes) {
      this.event.attributes = {};
    }
    this.event.attributes[key] = value;
    return this;
  }

  build(): CloudEvent {
    this.validate();
    return this.event;
  }

  private validate(): void {
    if (this.event.specversion !== '1.0') {
      throw new Error(`Invalid specversion: ${this.event.specversion}`);
    }
    if (!this.event.type) {
      throw new Error('type is required');
    }
    if (!this.event.source) {
      throw new Error('source is required');
    }
    if (!this.event.id) {
      throw new Error('id is required');
    }
  }
}

export class CloudEventValidator {
  static validate(event: CloudEvent): { valid: boolean; errors: string[] } {
    const errors: string[] = [];

    if (event.specversion !== '1.0') {
      errors.push(`Invalid specversion: ${event.specversion}`);
    }
    if (!event.type) {
      errors.push('type is required');
    }
    if (!event.source) {
      errors.push('source is required');
    }
    if (!event.id) {
      errors.push('id is required');
    }
    if (!event.time) {
      errors.push('time is required');
    }
    if (!event.datacontenttype) {
      errors.push('datacontenttype is required');
    }

    return {
      valid: errors.length === 0,
      errors,
    };
  }
}
