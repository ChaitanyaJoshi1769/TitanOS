import type { Metadata } from 'next'
import './globals.css'

export const metadata: Metadata = {
  title: 'Titan OS Dashboard',
  description: 'AI Agent and Workflow Orchestration Platform',
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="en">
      <body>
        <div className="flex h-screen bg-gray-50">
          {/* Sidebar */}
          <aside className="w-64 bg-white border-r border-gray-200">
            <nav className="p-6 space-y-4">
              <div className="text-2xl font-bold text-blue-600">Titan OS</div>
              <ul className="space-y-2 text-gray-700">
                <li><a href="/dashboard/overview" className="hover:text-blue-600">Overview</a></li>
                <li><a href="/dashboard/agents" className="hover:text-blue-600">Agents</a></li>
                <li><a href="/dashboard/workflows" className="hover:text-blue-600">Workflows</a></li>
                <li><a href="/dashboard/tasks" className="hover:text-blue-600">Tasks</a></li>
                <li><a href="/dashboard/nodes" className="hover:text-blue-600">Nodes</a></li>
                <li><a href="/dashboard/monitoring" className="hover:text-blue-600">Monitoring</a></li>
                <li><a href="/dashboard/settings" className="hover:text-blue-600">Settings</a></li>
              </ul>
            </nav>
          </aside>

          {/* Main content */}
          <main className="flex-1 overflow-auto">
            <header className="bg-white border-b border-gray-200 p-6 flex justify-between items-center">
              <h1 className="text-2xl font-bold text-gray-800">Dashboard</h1>
              <div className="flex items-center space-x-4">
                <div className="w-8 h-8 rounded-full bg-blue-600 text-white flex items-center justify-center">U</div>
              </div>
            </header>
            <div className="p-6">
              {children}
            </div>
          </main>
        </div>
      </body>
    </html>
  )
}
