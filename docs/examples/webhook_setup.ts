import axios from 'axios';

// Webhook management examples

const gatewayUrl = 'http://localhost:8000';
const userId = 'user-1';
const projectId = 'proj-1';

// Initialize client with auth
const client = axios.create({
  baseURL: gatewayUrl,
  headers: {
    Authorization: 'your-jwt-token-here',
    'Content-Type': 'application/json',
  },
});

// Example 1: Create a webhook subscription
async function createWebhookSubscription() {
  try {
    const response = await client.post('/api/v1/webhooks', {
      userId,
      projectId,
      webhookUrl: 'https://your-app.com/webhook',
      eventTypeFilter: 'titan.task.*',
      sourceFilter: 'scheduler',
      retryPolicy: {
        maxRetries: 5,
        initialDelayMs: 1000,
        maxDelayMs: 30000,
        backoffMultiplier: 2.0,
      },
    });

    const subscription = response.data;
    console.log('✓ Webhook subscription created:', {
      subscriptionId: subscription.subscriptionId,
      webhookUrl: subscription.webhookUrl,
      eventTypeFilter: subscription.eventTypeFilter,
    });

    return subscription.subscriptionId;
  } catch (error) {
    console.error('Failed to create subscription:', error);
    throw error;
  }
}

// Example 2: List webhook subscriptions
async function listWebhookSubscriptions() {
  try {
    const response = await client.get('/api/v1/webhooks', {
      params: {
        userId,
        projectId,
      },
    });

    const subscriptions = response.data.subscriptions;
    console.log(`✓ Found ${subscriptions.length} webhook subscriptions:`);

    subscriptions.forEach((sub: any) => {
      console.log(`  - ID: ${sub.subscriptionId}`);
      console.log(`    URL: ${sub.webhookUrl}`);
      console.log(`    Event Filter: ${sub.eventTypeFilter}`);
      console.log(`    Active: ${sub.active}`);
    });

    return subscriptions;
  } catch (error) {
    console.error('Failed to list subscriptions:', error);
    throw error;
  }
}

// Example 3: Update webhook subscription
async function updateWebhookSubscription(subscriptionId: string) {
  try {
    const response = await client.put(`/api/v1/webhooks/${subscriptionId}`, {
      webhookUrl: 'https://new-url.com/webhook',
      eventTypeFilter: 'titan.workflow.*',
      active: true,
      retryPolicy: {
        maxRetries: 3,
        initialDelayMs: 500,
        maxDelayMs: 20000,
        backoffMultiplier: 2.0,
      },
    });

    const updated = response.data;
    console.log('✓ Webhook subscription updated:', {
      subscriptionId: updated.subscriptionId,
      newUrl: updated.webhookUrl,
      newFilter: updated.eventTypeFilter,
    });

    return updated;
  } catch (error) {
    console.error('Failed to update subscription:', error);
    throw error;
  }
}

// Example 4: Test webhook delivery
async function testWebhookDelivery(subscriptionId: string) {
  try {
    const response = await client.post(`/api/v1/webhooks/${subscriptionId}/test`, {});

    const result = response.data;
    console.log('✓ Webhook test completed:', {
      success: result.success,
      httpStatus: result.httpStatus,
      latencyMs: result.latencyMs,
      responseBody: result.responseBody,
    });

    return result;
  } catch (error) {
    console.error('Failed to test webhook:', error);
    throw error;
  }
}

// Example 5: Get delivery history
async function getDeliveryHistory(subscriptionId: string) {
  try {
    const response = await client.get(`/api/v1/webhooks/${subscriptionId}/deliveries`, {
      params: {
        status: 'delivered',
        limit: 50,
      },
    });

    const deliveries = response.data.deliveries;
    console.log(`✓ Retrieved ${deliveries.length} webhook deliveries:`);

    deliveries.slice(0, 5).forEach((delivery: any) => {
      console.log(`  - Event: ${delivery.eventId}`);
      console.log(`    Status: ${delivery.status}`);
      console.log(`    HTTP Status: ${delivery.httpStatus}`);
      console.log(`    Latency: ${delivery.latencyMs}ms`);
      console.log(`    Delivered: ${delivery.deliveredAt}`);
    });

    return deliveries;
  } catch (error) {
    console.error('Failed to get delivery history:', error);
    throw error;
  }
}

// Example 6: Retry failed delivery
async function retryFailedDelivery(subscriptionId: string, eventId: string) {
  try {
    const response = await client.post(`/api/v1/webhooks/${subscriptionId}/retry`, {
      eventId,
    });

    const result = response.data;
    console.log('✓ Failed delivery retry initiated:', {
      deliveryId: result.deliveryId,
      success: result.success,
    });

    return result;
  } catch (error) {
    console.error('Failed to retry delivery:', error);
    throw error;
  }
}

// Example 7: Delete webhook subscription
async function deleteWebhookSubscription(subscriptionId: string) {
  try {
    const response = await client.delete(`/api/v1/webhooks/${subscriptionId}`);

    const result = response.data;
    console.log('✓ Webhook subscription deleted:', {
      success: result.success,
      subscriptionId,
    });

    return result;
  } catch (error) {
    console.error('Failed to delete subscription:', error);
    throw error;
  }
}

// Example 8: Webhook signature verification (for your webhook endpoint)
import * as crypto from 'crypto';

function verifyWebhookSignature(payload: string, signature: string, secret: string): boolean {
  const [algorithm, hash] = signature.split('=');

  if (algorithm !== 'sha256') {
    console.warn('Unknown signature algorithm:', algorithm);
    return false;
  }

  const hmac = crypto.createHmac('sha256', secret);
  hmac.update(payload);
  const expected = hmac.digest('hex');

  return crypto.timingSafeEqual(Buffer.from(hash), Buffer.from(expected));
}

// Example webhook endpoint handler
function webhookHandler(req: any, res: any) {
  const signature = req.headers['x-webhook-signature'];
  const payload = JSON.stringify(req.body);
  const webhookSecret = 'your-webhook-secret';

  if (!verifyWebhookSignature(payload, signature, webhookSecret)) {
    console.error('Invalid webhook signature');
    return res.status(401).json({ error: 'Invalid signature' });
  }

  const event = req.body;
  console.log('✓ Valid webhook received:', {
    type: event.type,
    id: event.id,
    source: event.source,
  });

  // Process the event
  // ...

  res.json({ success: true });
}

// Main execution
async function main() {
  console.log('=== Titan OS Webhook Management Examples ===\n');

  try {
    console.log('1. Creating webhook subscription...');
    const subscriptionId = await createWebhookSubscription();

    console.log('\n2. Listing webhook subscriptions...');
    await listWebhookSubscriptions();

    console.log('\n3. Updating webhook subscription...');
    await updateWebhookSubscription(subscriptionId);

    console.log('\n4. Testing webhook delivery...');
    await testWebhookDelivery(subscriptionId);

    console.log('\n5. Getting delivery history...');
    await getDeliveryHistory(subscriptionId);

    console.log('\n6. Retrying failed delivery...');
    // await retryFailedDelivery(subscriptionId, 'event-123');

    console.log('\n7. Webhook signature verification example shown');

    console.log('\n8. Deleting webhook subscription...');
    // await deleteWebhookSubscription(subscriptionId);

    console.log('\n=== All examples completed ===');
  } catch (error) {
    console.error('Example execution failed:', error);
  }
}

main();
