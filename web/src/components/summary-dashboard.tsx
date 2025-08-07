'use client'

import { useState, useEffect } from 'react'
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
  const [filters, setFilters] = useState<FilterOptions>({
    groups: [],
    timeRange: {
      start: new Date(Date.now() - 24 * 60 * 60 * 1000), // Last 24 hours
      end: new Date(),
    },
    searchQuery: '',
  })
  const [signalConfig, setSignalConfig] = useState<SignalConfig>({
    phoneNumber: '',
    isRegistered: false,
  })
  const [showExportDialog, setShowExportDialog] = useState(false)
  const [showSignalSetup, setShowSignalSetup] = useState(false)
  
  const { toast } = useToast()

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
  }, [])

  // Fetch summaries when filters change
  useEffect(() => {
    if (!loading) {
      fetchSummaries()
    }
  }, [filters, sortOrder])

  const fetchSummaries = async () => {
    try {
      const params = new URLSearchParams({
        sort: sortOrder,
        search: filters.searchQuery,
        start_time: Math.floor(filters.timeRange.start.getTime() / 1000).toString(),
        end_time: Math.floor(filters.timeRange.end.getTime() / 1000).toString(),
      })

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
  }

  const fetchGroups = async () => {
    try {
      const response = await fetch('/api/groups')
      if (!response.ok) throw new Error('Failed to fetch groups')
      
      const data = await response.json()
      setGroups(Array.isArray(data) ? data : data.groups || [])
    } catch (error) {
      console.error('Error fetching groups:', error)
      toast({
        title: 'Error',
        description: 'Failed to fetch groups',
        variant: 'destructive',
      })
    }
  }

  const fetchSignalConfig = async () => {
    try {
      const response = await fetch('/api/signal/config')
      if (!response.ok) throw new Error('Failed to fetch Signal config')
      
      const data = await response.json()
      setSignalConfig(data)
    } catch (error) {
      console.error('Error fetching Signal config:', error)
      // Don't show error toast for signal config as it might not be set up yet
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
    <div className="min-h-screen bg-gradient-to-br from-blue-50/30 via-background to-indigo-50/20">
      <Header
        viewMode={viewMode}
        onViewModeChange={setViewMode}
        sortOrder={sortOrder}
        onSortOrderChange={setSortOrder}
        onExport={() => setShowExportDialog(true)}
        onSignalSetup={() => setShowSignalSetup(true)}
        signalConfig={signalConfig}
      />
      
      <div className="container mx-auto px-4 py-6 space-y-6">
        <FilterPanel
          filters={filters}
          onFiltersChange={setFilters}
          groups={groups}
        />

        {summaries.length === 0 ? (
          <div className="text-center py-12">
            <p className="text-muted-foreground">No summaries found</p>
          </div>
        ) : viewMode === 'timeline' ? (
          <SummaryList summaries={summaries} />
        ) : (
          <SummaryCards summaries={summaries} />
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
