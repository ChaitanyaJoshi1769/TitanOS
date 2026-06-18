/**
 * Titan OS Task Submission Example
 *
 * This example demonstrates how to:
 * 1. Create a scheduler client
 * 2. Submit tasks
 * 3. Monitor task status
 * 4. Wait for task completion
 */

import {
  createSchedulerClient,
  generateTaskId,
  SubmitTaskRequest,
} from "@titan-os/sdk";

async function main() {
  console.log("🚀 Titan OS Task Submission Example\n");

  // Create scheduler client (connects to http://localhost:8000 by default)
  const scheduler = createSchedulerClient();

  // Example 1: Submit a single task
  console.log("📝 Submitting a single task...");
  const taskRequest: SubmitTaskRequest = {
    taskId: generateTaskId("example"),
    projectId: "example-project",
    name: "Processing Job",
    inputData: JSON.stringify({ url: "https://example.com", timeout: 30 }),
    timeoutSeconds: 300,
    priority: 1,
    maxRetries: 3,
    labels: {
      type: "http-request",
      category: "example",
    },
  };

  try {
    const submitResponse = await scheduler.submitTask(taskRequest);
    console.log(`✓ Task submitted: ${submitResponse.taskId}`);
    console.log(`  Status: ${submitResponse.message}\n`);

    // Example 2: Get task status
    console.log("🔍 Checking task status...");
    const taskStatus = await scheduler.getTaskStatus(submitResponse.taskId);
    console.log(`✓ Task status: ${taskStatus}\n`);

    // Example 3: Wait for task completion
    console.log("⏳ Waiting for task to complete (max 60 seconds)...");
    const completedTask = await scheduler.waitForTask(
      submitResponse.taskId,
      60,
      2000 // Poll every 2 seconds
    );
    console.log(`✓ Task completed with status: ${completedTask.status}`);
    if (completedTask.outputData) {
      console.log(`  Output: ${Buffer.from(completedTask.outputData).toString()}\n`);
    }
  } catch (error) {
    console.error(`✗ Error: ${(error as Error).message}`);
  }

  // Example 4: Submit multiple tasks in parallel
  console.log("\n📤 Submitting multiple tasks in parallel...");
  const taskRequests: SubmitTaskRequest[] = Array.from({ length: 5 }, (_, i) => ({
    taskId: generateTaskId(`batch-${i}`),
    projectId: "example-project",
    name: `Batch Task ${i + 1}`,
    inputData: JSON.stringify({ index: i, value: Math.random() }),
    timeoutSeconds: 300,
    priority: 0,
    maxRetries: 2,
    labels: {
      batch: "example",
      index: String(i),
    },
  }));

  try {
    const responses = await scheduler.submitTasks(taskRequests);
    console.log(`✓ Submitted ${responses.length} tasks`);
    responses.forEach((resp, idx) => {
      console.log(`  Task ${idx + 1}: ${resp.taskId} - ${resp.message}`);
    });
  } catch (error) {
    console.error(`✗ Error submitting tasks: ${(error as Error).message}`);
  }

  // Example 5: List tasks
  console.log("\n📋 Listing tasks for project...");
  try {
    const taskList = await scheduler.listTasks({
      projectId: "example-project",
      status: "pending",
      limit: 10,
      offset: 0,
    });
    console.log(`✓ Found ${taskList.tasks.length} pending tasks (${taskList.total} total)`);
    taskList.tasks.slice(0, 3).forEach((task) => {
      console.log(
        `  - ${task.id}: ${task.name} (status: ${task.status}, priority: ${task.priority})`
      );
    });
  } catch (error) {
    console.error(`✗ Error listing tasks: ${(error as Error).message}`);
  }

  // Example 6: List nodes
  console.log("\n🖥️  Listing available nodes...");
  try {
    const nodeList = await scheduler.listNodes(10, 0);
    console.log(`✓ Found ${nodeList.nodes.length} nodes (${nodeList.total} total)`);
    nodeList.nodes.slice(0, 3).forEach((node) => {
      console.log(
        `  - ${node.name}: ${node.cpuCores} CPU, ${node.memoryGb}GB RAM (status: ${node.status})`
      );
    });
  } catch (error) {
    console.error(`✗ Error listing nodes: ${(error as Error).message}`);
  }

  console.log("\n✨ Example completed!");
}

main().catch((error) => {
  console.error("Fatal error:", error);
  process.exit(1);
});
