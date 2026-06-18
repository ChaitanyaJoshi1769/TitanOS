import { EventPublisher, CloudEventBuilder, TaskSubmittedEvent, TaskCompletedEvent } from '@titanos/events';

// Initialize publisher
const publisher = new EventPublisher({
  eventBusUrl: 'http://localhost:8002',
  timeout: 30000,
});

// Example 1: Publish a task submitted event
async function publishTaskSubmitted() {
  const event = new CloudEventBuilder('titan.task.submitted', 'scheduler')
    .withId('task-123')
    .withSubject('tasks/task-123')
    .withData({
      taskId: 'task-123',
      projectId: 'proj-1',
      name: 'Process User Data',
      labels: {
        priority: 'high',
        service: 'data-pipeline',
      },
    } as TaskSubmittedEvent)
    .build();

  try {
    const result = await publisher.publishEvent(event);
    console.log('✓ Task submitted event published:', result);
  } catch (error) {
    console.error('Failed to publish event:', error);
  }
}

// Example 2: Publish a task completed event
async function publishTaskCompleted() {
  const event = new CloudEventBuilder('titan.task.completed', 'node-agent-5')
    .withId('task-123-completion')
    .withSubject('tasks/task-123')
    .withData({
      taskId: 'task-123',
      nodeId: 'node-5',
      completedAt: new Date().toISOString(),
      exitCode: 0,
      output: 'Task completed successfully. Processed 1000 records.',
    } as TaskCompletedEvent)
    .build();

  try {
    const result = await publisher.publishEvent(event);
    console.log('✓ Task completed event published:', result);
  } catch (error) {
    console.error('Failed to publish event:', error);
  }
}

// Example 3: Publish batch events
async function publishEventBatch() {
  const events = [];

  for (let i = 0; i < 5; i++) {
    const event = new CloudEventBuilder('titan.task.submitted', 'scheduler')
      .withId(`task-batch-${i}`)
      .withSubject(`tasks/task-batch-${i}`)
      .withData({
        taskId: `task-batch-${i}`,
        projectId: 'proj-1',
        name: `Batch Task ${i}`,
      } as TaskSubmittedEvent)
      .build();

    events.push(event);
  }

  try {
    const results = await publisher.publishEventBatch(events);
    console.log(`✓ Published ${results.length} events in batch`);
    results.forEach((result, idx) => {
      console.log(`  Event ${idx}: ID=${result.eventId}, Offset=${result.offset}`);
    });
  } catch (error) {
    console.error('Failed to publish batch:', error);
  }
}

// Example 4: Get event schema
async function getEventSchema() {
  try {
    const schema = await publisher.getEventSchema('titan.task.submitted');
    console.log('✓ Event schema:', JSON.stringify(schema, null, 2));
  } catch (error) {
    console.error('Failed to get schema:', error);
  }
}

// Example 5: List event types
async function listEventTypes() {
  try {
    const eventTypes = await publisher.listEventTypes('scheduler');
    console.log('✓ Event types from scheduler:', eventTypes);
  } catch (error) {
    console.error('Failed to list event types:', error);
  }
}

// Example 6: Get metrics
async function getMetrics() {
  try {
    const metrics = await publisher.getEventMetrics();
    console.log('✓ Event bus metrics:', {
      eventsPublished: metrics.eventsPublished,
      eventsConsumed: metrics.eventsConsumed,
      webhookDeliveries: metrics.webhookDeliveries,
      failedDeliveries: metrics.failedDeliveries,
    });
  } catch (error) {
    console.error('Failed to get metrics:', error);
  }
}

// Main execution
async function main() {
  console.log('=== Titan OS Event Publishing Examples ===\n');

  console.log('1. Publishing single task submitted event...');
  await publishTaskSubmitted();

  console.log('\n2. Publishing task completed event...');
  await publishTaskCompleted();

  console.log('\n3. Publishing batch of events...');
  await publishEventBatch();

  console.log('\n4. Retrieving event schema...');
  await getEventSchema();

  console.log('\n5. Listing event types...');
  await listEventTypes();

  console.log('\n6. Getting event bus metrics...');
  await getMetrics();

  console.log('\n=== All examples completed ===');
}

main().catch(console.error);
