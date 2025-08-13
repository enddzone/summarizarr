import { Suspense } from 'react'
import { SummaryDashboard } from '@/components/summary-dashboard'
import { LoadingSpinner } from '@/components/ui/loading-spinner'

export default function Home() {
  return (
    <main className="min-h-screen bg-background overflow-x-hidden">
      <Suspense fallback={<LoadingSpinner />}>
        <SummaryDashboard />
      </Suspense>
    </main>
  )
}
