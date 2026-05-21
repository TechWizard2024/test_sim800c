import React, { useEffect, useState } from 'react'

export default function App() {
  const [theme, setTheme] = useState(() => localStorage.getItem('theme') || 'dark')

  useEffect(() => {
    document.documentElement.classList.toggle('dark', theme === 'dark')
    localStorage.setItem('theme', theme)
  }, [theme])

  return (
    <div className="min-h-screen">
      <header className="flex items-center justify-between px-4 py-3 border-b border-gray-200 dark:border-gray-800">
        <div className="font-semibold">SIM800C Supervision</div>
        <button
          className="px-3 py-2 rounded bg-gray-200 dark:bg-gray-700"
          onClick={() => setTheme((t) => (t === 'dark' ? 'light' : 'dark'))}
          title="Changer le thème"
        >
          Theme: {theme}
        </button>
      </header>

      <main className="p-4">
        <div className="grid gap-4 md:grid-cols-3">
          <div className="rounded-lg border border-gray-200 dark:border-gray-800 p-4">
            <div className="text-sm opacity-70">Modems</div>
            <div className="text-2xl font-bold">(loading)</div>
          </div>
          <div className="rounded-lg border border-gray-200 dark:border-gray-800 p-4">
            <div className="text-sm opacity-70">USSD</div>
            <div className="text-2xl font-bold">(loading)</div>
          </div>
          <div className="rounded-lg border border-gray-200 dark:border-gray-800 p-4">
            <div className="text-sm opacity-70">SMS</div>
            <div className="text-2xl font-bold">(loading)</div>
          </div>
        </div>

        <div className="mt-6 rounded-lg border border-gray-200 dark:border-gray-800 p-4">
          <div className="font-semibold">Dashboard</div>
          <div className="text-sm opacity-70 mt-2">
            Connexion WebSocket à implémenter. Les widgets temps réel seront branchés ici.
          </div>
        </div>
      </main>
    </div>
  )
}

