#!/usr/bin/env node

import { program } from 'commander'
import { TitanOSClient } from '@titanos/sdk-typescript'

const version = '1.0.0'
const apiUrl = process.env.TITAN_API_URL || 'http://localhost:8000'
const token = process.env.TITAN_TOKEN

const client = new TitanOSClient({ apiUrl, token })

program
  .name('titan')
  .description('Titan OS CLI - AI Agent and Workflow Orchestration')
  .version(version)

// Task commands
program
  .command('task:submit <input>')
  .description('Submit a new task')
  .action(async (input) => {
    try {
      const data = JSON.parse(input)
      const task = await client.submitTask(data)
      console.log('✓ Task submitted:', task.id)
    } catch (error) {
      console.error('Error:', error)
      process.exit(1)
    }
  })

program
  .command('task:status <taskId>')
  .description('Get task status')
  .action(async (taskId) => {
    try {
      const task = await client.getTask(taskId)
      console.log(`Status: ${task.status}`)
      if (task.output) console.log('Output:', task.output)
      if (task.error) console.log('Error:', task.error)
    } catch (error) {
      console.error('Error:', error)
      process.exit(1)
    }
  })

program
  .command('task:list')
  .description('List tasks')
  .option('-l, --limit <number>', 'Limit number of tasks', '100')
  .action(async (options) => {
    try {
      const tasks = await client.listTasks(parseInt(options.limit))
      console.table(tasks.map(t => ({
        id: t.id,
        status: t.status,
        created: new Date(t.createdAt).toISOString(),
      })))
    } catch (error) {
      console.error('Error:', error)
      process.exit(1)
    }
  })

// Agent commands
program
  .command('agent:create <name>')
  .description('Create a new agent')
  .action(async (name) => {
    try {
      const agent = await client.createAgent(name)
      console.log('✓ Agent created:', agent.id)
    } catch (error) {
      console.error('Error:', error)
      process.exit(1)
    }
  })

program
  .command('agent:list')
  .description('List agents')
  .action(async () => {
    try {
      const agents = await client.listAgents()
      console.table(agents.map(a => ({
        id: a.id,
        name: a.name,
        status: a.status,
        created: new Date(a.createdAt).toISOString(),
      })))
    } catch (error) {
      console.error('Error:', error)
      process.exit(1)
    }
  })

// Workflow commands
program
  .command('workflow:create <name> <definition>')
  .description('Create a new workflow')
  .action(async (name, definition) => {
    try {
      const def = JSON.parse(definition)
      const workflow = await client.createWorkflow(name, def)
      console.log('✓ Workflow created:', workflow.id)
    } catch (error) {
      console.error('Error:', error)
      process.exit(1)
    }
  })

program
  .command('workflow:execute <workflowId> <input>')
  .description('Execute a workflow')
  .action(async (workflowId, input) => {
    try {
      const data = JSON.parse(input)
      const result = await client.executeWorkflow(workflowId, data)
      console.log('✓ Workflow execution started:', result.executionId)
    } catch (error) {
      console.error('Error:', error)
      process.exit(1)
    }
  })

// Health check
program
  .command('health')
  .description('Check service health')
  .action(async () => {
    try {
      const health = await client.healthCheck()
      console.log('✓ Service status:', health.status)
    } catch (error) {
      console.error('✗ Service unavailable:', error)
      process.exit(1)
    }
  })

program.parse(process.argv)

if (!process.argv.slice(2).length) {
  program.outputHelp()
}
