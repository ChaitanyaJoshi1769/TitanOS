'use client'

import { useQuery } from '@tanstack/react-query'
import axios from 'axios'
import { useState } from 'react'

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8000'

interface Agent {
  id: string
  name: string
  status: 'online' | 'offline' | 'idle'
  createdAt: string
  tasksCompleted: number
  uptime: string
}

export default function AgentsPage() {
  const [agents, setAgents] = useState<Agent[]>([])
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [agentName, setAgentName] = useState('')

  const { isLoading } = useQuery({
    queryKey: ['agents'],
    queryFn: async () => {
      const res = await axios.get(`${API_BASE}/api/v1/agents`)
      setAgents(res.data.agents || [])
      return res.data
    },
    refetchInterval: 5000,
  })

  const handleCreateAgent = async () => {
    try {
      await axios.post(`${API_BASE}/api/v1/agents`, { name: agentName })
      setAgentName('')
      setShowCreateModal(false)
      // Refetch agents
    } catch (error) {
      console.error('Failed to create agent:', error)
    }
  }

  if (isLoading) return <div className="text-center py-12">Loading agents...</div>

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <h2 className="text-2xl font-bold">Agents</h2>
        <button
          onClick={() => setShowCreateModal(true)}
          className="bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700"
        >
          Create Agent
        </button>
      </div>

      {/* Agents Table */}
      <div className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-100 border-b">
            <tr>
              <th className="px-6 py-3 text-left text-sm font-semibold text-gray-700">Name</th>
              <th className="px-6 py-3 text-left text-sm font-semibold text-gray-700">Status</th>
              <th className="px-6 py-3 text-left text-sm font-semibold text-gray-700">Created</th>
              <th className="px-6 py-3 text-left text-sm font-semibold text-gray-700">Tasks Completed</th>
              <th className="px-6 py-3 text-left text-sm font-semibold text-gray-700">Uptime</th>
              <th className="px-6 py-3 text-left text-sm font-semibold text-gray-700">Actions</th>
            </tr>
          </thead>
          <tbody>
            {agents.length === 0 ? (
              <tr>
                <td colSpan={6} className="px-6 py-8 text-center text-gray-500">
                  No agents yet. Create one to get started.
                </td>
              </tr>
            ) : (
              agents.map((agent) => (
                <tr key={agent.id} className="border-b hover:bg-gray-50">
                  <td className="px-6 py-4 font-medium text-gray-900">{agent.name}</td>
                  <td className="px-6 py-4">
                    <span className={`px-3 py-1 rounded-full text-sm font-medium ${
                      agent.status === 'online' ? 'bg-green-100 text-green-800' :
                      agent.status === 'idle' ? 'bg-yellow-100 text-yellow-800' :
                      'bg-red-100 text-red-800'
                    }`}>
                      {agent.status}
                    </span>
                  </td>
                  <td className="px-6 py-4 text-sm text-gray-600">{new Date(agent.createdAt).toLocaleDateString()}</td>
                  <td className="px-6 py-4 text-sm text-gray-600">{agent.tasksCompleted}</td>
                  <td className="px-6 py-4 text-sm text-gray-600">{agent.uptime}</td>
                  <td className="px-6 py-4">
                    <button className="text-blue-600 hover:text-blue-800 text-sm font-medium">View</button>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {/* Create Agent Modal */}
      {showCreateModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-8 max-w-md w-full">
            <h3 className="text-xl font-bold mb-4">Create New Agent</h3>
            <input
              type="text"
              placeholder="Agent Name"
              value={agentName}
              onChange={(e) => setAgentName(e.target.value)}
              className="w-full px-4 py-2 border border-gray-300 rounded mb-4 focus:outline-none focus:border-blue-600"
            />
            <div className="flex justify-end gap-2">
              <button
                onClick={() => setShowCreateModal(false)}
                className="px-4 py-2 border border-gray-300 rounded hover:bg-gray-50"
              >
                Cancel
              </button>
              <button
                onClick={handleCreateAgent}
                className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
              >
                Create
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
