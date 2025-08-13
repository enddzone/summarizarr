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

  const { toast } = useToast()

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

      console.log('=== fetchSummaries Debug ===')
      console.log('filters.timeRange.start:', filters.timeRange.start)
      console.log('filters.timeRange.start.getTime():', filters.timeRange.start.getTime())
      console.log('filters.timeRange.end:', filters.timeRange.end)
      console.log('filters.activePreset:', filters.activePreset)
      console.log('isAllTime calculated:', isAllTime)
      console.log('=== End Debug ===')

      if (!isAllTime) {
        const startTime = Math.floor(filters.timeRange.start.getTime() / 1000).toString()
        const endTime = Math.floor(filters.timeRange.end.getTime() / 1000).toString()
        console.log('Adding time parameters:', { startTime, endTime })
        params.append('start_time', startTime)
        params.append('end_time', endTime)
      } else {
        console.log('Skipping time parameters (all-time mode)')
      }

      if (filters.groups.length > 0) {
        params.append('groups', filters.groups.join(','))
      }

      console.log('Fetching summaries with params:', params.toString())
      const response = await fetch(`/api/summaries?${params}`)
      if (!response.ok) throw new Error('Failed to fetch summaries')

      const data = await response.json()
      console.log('Raw summaries response:', data)
      const summariesData = Array.isArray(data) ? data : data.summaries || []

      // Group names are now included in the API response
      console.log('Summaries with group names:', summariesData)
      setSummaries(summariesData)
    } catch (error) {
      console.error('Error fetching summaries:', error)
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
      const groupsData = Array.isArray(data) ? data : data.groups || []
      setGroups(groupsData)
    } catch (error) {
      console.error('Error fetching groups:', error)
      toast({
        title: 'Error',
        description: 'Failed to fetch groups',
        variant: 'destructive',
      })
    }
  }, [toast])

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
        await fetchGroups()
        await fetchSummaries()
        await fetchSignalConfig()
      } finally {
        setLoading(false)
      }
    }
    loadData()
  }, [fetchGroups, fetchSummaries, fetchSignalConfig])

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
      setSummaries((prev) => prev.filter((s) => s.id !== id))
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

        {summaries.length === 0 ? (
          <div className="text-center py-8 sm:py-12">
            <p className="text-muted-foreground text-sm sm:text-base">No summaries found</p>
          </div>
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
