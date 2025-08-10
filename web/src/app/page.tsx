import { Suspense } from 'react'
import { SummaryDashboard } from '@/components/summary-dashboard'
import { LoadingSpinner } from '@/components/ui/loading-spinner'

export default function Home() {
  return (
    <main className="min-h-screen bg-background">
      {/* Local Development Hot Reload Test - Modified */}
      <Suspense fallback={<LoadingSpinner />}>
        <SummaryDashboard />
      </Suspense>
    </main>
  )
}
