'use client'

import { useQuery } from '@tanstack/react-query'
import { LineChart, Line, BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts'
import axios from 'axios'

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8000'

interface SystemMetrics {
  tasksSubmitted: number
  tasksCompleted: number
  tasksFailed: number
  workflowsRunning: number
  workflowsCompleted: number
  activeAgents: number
  activeNodes: number
  avgLatency: number
}

export default function OverviewPage() {
  const { data: metrics, isLoading } = useQuery({
    queryKey: ['metrics'],
    queryFn: async () => {
      const res = await axios.get(`${API_BASE}/api/v1/metrics`)
      return res.data as SystemMetrics
    },
    refetchInterval: 5000,
  })

  const chartData = [
    { time: '00:00', tasks: 120, workflows: 45 },
    { time: '04:00', tasks: 300, workflows: 120 },
    { time: '08:00', tasks: 450, workflows: 200 },
    { time: '12:00', tasks: 600, workflows: 280 },
    { time: '16:00', tasks: 500, workflows: 250 },
    { time: '20:00', tasks: 400, workflows: 180 },
  ]

  if (isLoading) return <div className="text-center py-12">Loading...</div>

  return (
    <div className="space-y-8">
      {/* Key Metrics */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <MetricCard
          title="Tasks Submitted"
          value={metrics?.tasksSubmitted || 0}
          color="bg-blue-500"
        />
        <MetricCard
          title="Tasks Completed"
          value={metrics?.tasksCompleted || 0}
          color="bg-green-500"
        />
        <MetricCard
          title="Tasks Failed"
          value={metrics?.tasksFailed || 0}
          color="bg-red-500"
        />
        <MetricCard
          title="Active Agents"
          value={metrics?.activeAgents || 0}
          color="bg-purple-500"
        />
      </div>

      {/* Charts */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Tasks & Workflows Over Time */}
        <div className="bg-white p-6 rounded-lg shadow">
          <h2 className="text-lg font-semibold mb-4">Throughput</h2>
          <ResponsiveContainer width="100%" height={300}>
            <LineChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="time" />
              <YAxis />
              <Tooltip />
              <Legend />
              <Line type="monotone" dataKey="tasks" stroke="#3b82f6" />
              <Line type="monotone" dataKey="workflows" stroke="#10b981" />
            </LineChart>
          </ResponsiveContainer>
        </div>

        {/* System Status */}
        <div className="bg-white p-6 rounded-lg shadow">
          <h2 className="text-lg font-semibold mb-4">System Status</h2>
          <div className="space-y-4">
            <StatusRow label="Active Nodes" value={metrics?.activeNodes || 0} status="healthy" />
            <StatusRow label="Workflows Running" value={metrics?.workflowsRunning || 0} status="running" />
            <StatusRow label="Avg Latency" value={`${metrics?.avgLatency || 0}ms`} status={metrics?.avgLatency! < 100 ? 'healthy' : 'warning'} />
            <StatusRow label="API Health" value="Online" status="healthy" />
          </div>
        </div>
      </div>

      {/* Resource Utilization */}
      <div className="bg-white p-6 rounded-lg shadow">
        <h2 className="text-lg font-semibold mb-4">Resource Utilization</h2>
        <ResponsiveContainer width="100%" height={300}>
          <BarChart data={[
            { resource: 'CPU', usage: 65 },
            { resource: 'Memory', usage: 72 },
            { resource: 'Disk', usage: 48 },
            { resource: 'Network', usage: 35 },
          ]}>
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis dataKey="resource" />
            <YAxis domain={[0, 100]} />
            <Tooltip />
            <Bar dataKey="usage" fill="#3b82f6" />
          </BarChart>
        </ResponsiveContainer>
      </div>
    </div>
  )
}

function MetricCard({ title, value, color }: { title: string; value: number; color: string }) {
  return (
    <div className="bg-white p-6 rounded-lg shadow">
      <div className={`${color} text-white text-2xl font-bold rounded mb-2 p-3 inline-block`}>
        {value}
      </div>
      <p className="text-gray-600 text-sm">{title}</p>
    </div>
  )
}

function StatusRow({ label, value, status }: { label: string; value: string | number; status: string }) {
  const statusColor = status === 'healthy' ? 'text-green-600' : status === 'running' ? 'text-blue-600' : 'text-yellow-600'
  return (
    <div className="flex justify-between items-center p-3 bg-gray-50 rounded">
      <span className="text-gray-700">{label}</span>
      <span className={`font-semibold ${statusColor}`}>{value}</span>
    </div>
  )
}
