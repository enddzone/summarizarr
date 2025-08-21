import { Suspense } from 'react'
import { ProtectedRoute } from '@/components/auth/protected-route'
import { SummaryDashboard } from '@/components/summary-dashboard'
import { LoadingSpinner } from '@/components/ui/loading-spinner'

export default function Home() {
  return (
    <ProtectedRoute>
      <Suspense fallback={<LoadingSpinner />}>
        <SummaryDashboard />
      </Suspense>
    </ProtectedRoute>
  )
}
