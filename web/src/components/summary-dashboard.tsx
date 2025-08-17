'use client'

import { useState, useEffect, useCallback } from 'react'
import { Header } from '@/components/header'
import { FilterPanel } from '@/components/filter-panel'
import { SummaryList } from '@/components/summary-list'
import { SummaryCards } from '@/components/summary-cards'
import { ExportDialog } from '@/components/export-dialog'
import { SignalSetupDialog } from '@/components/signal-setup-dialog'
import { LoadingSpinner } from '@/components/ui/loading-spinner'
import { useToast } from '@/hooks/use-toast'
import type { Summary, Group, FilterOptions, ViewMode, SortOrder, SignalConfig } from '@/types'
import { EmptyState } from '@/components/empty-state'
import { deriveEmptyState, FetchErrorInfo } from '@/lib/derive-empty-state'
import { APP_CONFIG, openExternalUrl } from '@/lib/config'

export function SummaryDashboard() {
  const [summaries, setSummaries] = useState<Summary[]>([])
  const [groups, setGroups] = useState<Group[]>([])
  const [loading, setLoading] = useState(true)
  const [viewMode, setViewMode] = useState<ViewMode>('timeline')
  const [sortOrder, setSortOrder] = useState<SortOrder>('newest')
  const [filters, setFilters] = useState<FilterOptions>(() => {
    // Default to "Today" instead of "All time"
    const now = new Date();
    return {
      groups: [],
      timeRange: {
        start: new Date(now.getFullYear(), now.getMonth(), now.getDate()), // Start of today
        end: now, // Current time
      },
      searchQuery: '',
      activePreset: 'today',
    }
  })
  const [signalConfig, setSignalConfig] = useState<SignalConfig>({
    phoneNumber: '',
    isRegistered: false,
  })
  const [showExportDialog, setShowExportDialog] = useState(false)
  const [showSignalSetup, setShowSignalSetup] = useState(false)
  const [fetchError, setFetchError] = useState<FetchErrorInfo | undefined>(undefined)

  const { toast } = useToast()

  // Helper function to clear filters while preserving time range
  const clearFilters = useCallback(() => {
    setFilters((prev: FilterOptions) => ({
      ...prev,
      groups: [],
      searchQuery: '',
      timeRange: prev.timeRange,
      activePreset: prev.activePreset
    }))
  }, [])

  // Restore view mode from localStorage on mount
  useEffect(() => {
    try {
      const savedViewMode = localStorage.getItem('summarizarr-view-mode') as ViewMode
      if (savedViewMode && (savedViewMode === 'timeline' || savedViewMode === 'cards')) {
        setViewMode(savedViewMode)
      }
    } catch (error) {
      console.warn('Failed to restore view mode from localStorage:', error)
    }
  }, [])

  // Save view mode to localStorage when it changes
  useEffect(() => {
    try {
      localStorage.setItem('summarizarr-view-mode', viewMode)
    } catch (error) {
      console.warn('Failed to save view mode to localStorage:', error)
    }
  }, [viewMode])

  // Define functions with useCallback to stabilize references
  const fetchSummaries = useCallback(async () => {
    try {
      const params = new URLSearchParams({
        sort: sortOrder,
        search: filters.searchQuery,
      })

      // Only add time range filters if not "All time"
      // Check both the epoch date and the active preset to determine if this is "all time"
      const isAllTime = filters.timeRange.start.getTime() === 0 && filters.activePreset === 'all-time'

      if (!isAllTime) {
        const startTime = Math.floor(filters.timeRange.start.getTime() / 1000).toString()
        const endTime = Math.floor(filters.timeRange.end.getTime() / 1000).toString()
        params.append('start_time', startTime)
        params.append('end_time', endTime)
      }

      if (filters.groups.length > 0) {
        params.append('groups', filters.groups.join(','))
      }

      const response = await fetch(`/api/summaries?${params}`)
      if (!response.ok) {
        const msg = `Failed to fetch summaries (${response.status})`
        throw Object.assign(new Error(msg), { status: response.status })
      }

      const data = await response.json()
      const summariesData = Array.isArray(data) ? data : data.summaries || []

      // Group names are now included in the API response
      setSummaries(summariesData)
    } catch (error) {
      const err = error as { message?: string; status?: number }
      console.error('Error fetching summaries:', err)
      setFetchError({ scope: 'summaries', message: err?.message || 'Failed to fetch summaries', status: err?.status })
      toast({
        title: 'Error',
        description: 'Failed to fetch summaries',
        variant: 'destructive',
      })
    }
  }, [sortOrder, filters, toast])

  const fetchGroups = useCallback(async () => {
    try {
      const response = await fetch('/api/groups')
      if (!response.ok) throw new Error('Failed to fetch groups')

      const data = await response.json()
      const groupsData = Array.isArray(data) ? data : (data?.groups || [])
      setGroups(groupsData)
    } catch (error) {
      // Suppress user-facing error toast during initial onboarding when Signal is not yet registered.
      // We only surface the toast if Signal *is* registered (so groups should exist) and we're past initial load.
      console.warn('Groups fetch failed', error)
      if (signalConfig.isRegistered && !loading) {
        toast({
          title: 'Error',
          description: 'Failed to fetch groups',
          variant: 'destructive',
        })
      }
    }
  }, [toast, signalConfig.isRegistered, loading])

  const fetchSignalConfig = useCallback(async () => {
    try {
      const response = await fetch('/api/signal/config')
      if (!response.ok) throw new Error('Failed to fetch signal config')

      const data = await response.json()
      setSignalConfig(data)
    } catch (error) {
      console.error('Error fetching signal config:', error)
      // Signal config is optional, so we don't show an error toast
    }
  }, [])

  // Fetch initial data
  useEffect(() => {
    const loadData = async () => {
      try {
        await Promise.all([
          (async () => { await fetchGroups() })(),
          (async () => { await fetchSummaries() })(),
          (async () => { await fetchSignalConfig() })(),
        ])
      } finally {
        setLoading(false)
      }
    }
    loadData()
  }, [fetchGroups, fetchSummaries, fetchSignalConfig])

  // Auto-open Signal setup dialog once per session if not registered
  useEffect(() => {
    const SESSION_KEY = 'summarizarr-signal-setup-shown'

    if (!loading && !signalConfig.isRegistered) {
      // Check if we've already shown the dialog this session
      const hasShownThisSession = sessionStorage.getItem(SESSION_KEY) === 'true'

      if (!hasShownThisSession) {
        setShowSignalSetup(true)
        sessionStorage.setItem(SESSION_KEY, 'true')
      }
    }
  }, [loading, signalConfig.isRegistered])

  // Fetch summaries when filters change
  useEffect(() => {
    if (!loading) {
      fetchSummaries()
    }
  }, [filters, sortOrder, loading, fetchSummaries])

  const handleDelete = async (id: number) => {
    try {
      const res = await fetch(`/api/summaries/${id}`, { method: 'DELETE' })
      if (!res.ok) throw new Error('Failed to delete summary')
      setSummaries((prev: Summary[]) => prev.filter((s: Summary) => s.id !== id))
      toast({ title: 'Deleted', description: 'Summary removed.' })
    } catch (e) {
      console.error('Delete failed', e)
      toast({ title: 'Delete failed', description: 'Could not delete summary.', variant: 'destructive' })
    }
  }

  const handleExport = async (format: 'json' | 'csv' | 'pdf') => {
    try {
      const params = new URLSearchParams({
        format,
        sort: sortOrder,
        search: filters.searchQuery,
        start_time: Math.floor(filters.timeRange.start.getTime() / 1000).toString(),
        end_time: Math.floor(filters.timeRange.end.getTime() / 1000).toString(),
      })

      if (filters.groups.length > 0) {
        params.append('groups', filters.groups.join(','))
      }

      const response = await fetch(`/api/export?${params}`)
      if (!response.ok) throw new Error('Export failed')

      const blob = await response.blob()
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `summaries-${new Date().toISOString().split('T')[0]}.${format}`
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      URL.revokeObjectURL(url)

      toast({
        title: 'Export successful',
        description: `Summaries exported as ${format.toUpperCase()}`,
      })
    } catch (error) {
      console.error('Export error:', error)
      toast({
        title: 'Export failed',
        description: 'Failed to export summaries',
        variant: 'destructive',
      })
    }
  }

  if (loading) {
    return <LoadingSpinner />
  }

  // Derive empty state (only when not loading and no summaries or error)
  const derived = !loading ? deriveEmptyState({
    signalConfig,
    summaries,
    groups,
    filters,
    fetchError,
  }) : null

  return (
    <div className="min-h-screen">
      <Header
        viewMode={viewMode}
        onViewModeChange={setViewMode}
        sortOrder={sortOrder}
        onSortOrderChange={setSortOrder}
        onExport={() => setShowExportDialog(true)}
        onSignalSetup={() => setShowSignalSetup(true)}
        signalConfig={signalConfig}
      />

      {/* Responsive container with better mobile padding */}
      <div className="mx-auto max-w-screen-2xl px-3 sm:px-4 md:px-6 lg:px-8 py-4 sm:py-6 space-y-4 sm:space-y-6">
        <FilterPanel
          filters={filters}
          onFiltersChange={setFilters}
          groups={groups}
        />

        {derived ? (
          <EmptyState
            variant={derived.variant}
            primaryLabel={derived.primaryLabel}
            secondaryLabel={derived.secondaryLabel}
            meta={derived.meta}
            onPrimary={() => {
              if (derived.variant === 'needs-signal') {
                setShowSignalSetup(true)
              } else if (derived.variant === 'filters-empty') {
                clearFilters()
              } else {
                // Retry/Refresh
                setFetchError(undefined)
                fetchSummaries()
              }
            }}
            onSecondary={derived.secondaryLabel ? () => {
              if (derived.variant === 'filters-empty' || derived.variant === 'no-results') {
                // Set to all time
                setFilters((prev: FilterOptions) => ({
                  ...prev,
                  timeRange: { start: new Date(0), end: new Date() },
                  activePreset: 'all-time',
                }))
              } else if (derived.variant === 'needs-signal') {
                openExternalUrl(APP_CONFIG.DOCS.SIGNAL_SETUP, APP_CONFIG.EXTERNAL_LINKS.GITHUB_REPO)
              }
            } : undefined}
          />
        ) : viewMode === 'timeline' ? (
          <div className="overflow-x-auto">
            <SummaryList summaries={summaries} onDelete={handleDelete} />
          </div>
        ) : (
          <SummaryCards summaries={summaries} onDelete={handleDelete} />
        )}
      </div>

      <ExportDialog
        open={showExportDialog}
        onOpenChange={setShowExportDialog}
        onExport={handleExport}
      />

      <SignalSetupDialog
        open={showSignalSetup}
        onOpenChange={setShowSignalSetup}
        signalConfig={signalConfig}
        onConfigUpdate={setSignalConfig}
      />
    </div>
  )
}
