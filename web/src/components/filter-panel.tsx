'use client'

import { useMemo, useRef, useState, useCallback } from 'react'
import { Search, Filter, X, Check, MessageSquare } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { DatePickerWithRange } from '@/components/ui/date-range-picker'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import type { FilterOptions, Group } from '@/types'

interface FilterPanelProps {
  filters: FilterOptions
  onFiltersChange: (filters: FilterOptions) => void
  groups: Group[]
}

export function FilterPanel({ filters, onFiltersChange, groups }: FilterPanelProps) {
  const [groupQuery, setGroupQuery] = useState('')
  const [open, setOpen] = useState(false)
  const listRef = useRef<HTMLDivElement | null>(null)

  const handleSearchChange = (value: string) => {
    console.log('Search changed:', value)
    onFiltersChange({
      ...filters,
      searchQuery: value,
    })
  }

  const handleGroupToggle = (groupId: number, checked: boolean) => {
    const updatedGroups = checked
      ? [...filters.groups, groupId]
      : filters.groups.filter(id => id !== groupId)

    console.log('Group toggled:', { groupId, checked, updatedGroups })
    onFiltersChange({
      ...filters,
      groups: updatedGroups,
    })
  }

  const handleClearGroups = useCallback(() => {
    onFiltersChange({ ...filters, groups: [] })
  }, [filters, onFiltersChange])

  const filteredGroups = useMemo(() => {
    const q = groupQuery.trim().toLowerCase()
    if (!q) return groups
    return groups.filter(g => g.name.toLowerCase().includes(q))
  }, [groupQuery, groups])

  const palette = [
    '#3498db',
    '#1abc9c',
    '#9b59b6',
    '#e67e22',
    '#e74c3c',
    '#2ecc71',
    '#f1c40f',
    '#34495e',
    '#16a085',
    '#2980b9',
  ] as const
  const colorForGroup = (groupId: number) => palette[Math.abs(groupId) % palette.length]

  const handleDateRangeChange = (range: { start: Date; end: Date } | undefined) => {
    if (range) {
      console.log('Date range changed:', range)
      onFiltersChange({
        ...filters,
        timeRange: range,
        activePreset: undefined, // Clear preset when manually changing dates
      })
    }
  }

  const handlePresetChange = (preset: 'all-time' | 'today' | 'yesterday' | 'morning' | '6h' | '12h' | undefined) => {
    console.log('Preset changed:', preset)

    let newTimeRange = filters.timeRange;

    // Calculate the time range for the preset
    if (preset) {
      switch (preset) {
        case 'all-time':
          newTimeRange = { start: new Date(0), end: new Date() };
          break;
        case 'today': {
          const now = new Date();
          newTimeRange = {
            start: new Date(now.getFullYear(), now.getMonth(), now.getDate()),
            end: now
          };
          break;
        }
        case 'yesterday': {
          const now = new Date();
          const y = new Date(now);
          y.setDate(now.getDate() - 1);
          newTimeRange = {
            start: new Date(y.getFullYear(), y.getMonth(), y.getDate()),
            end: new Date(y.getFullYear(), y.getMonth(), y.getDate(), 23, 59, 59, 999)
          };
          break;
        }
        case 'morning': {
          const now = new Date();
          newTimeRange = {
            start: new Date(now.getFullYear(), now.getMonth(), now.getDate(), 6, 0, 0),
            end: new Date(now.getFullYear(), now.getMonth(), now.getDate(), 12, 0, 0)
          };
          break;
        }
        case '6h': {
          const end = new Date();
          newTimeRange = {
            start: new Date(end.getTime() - 6 * 60 * 60 * 1000),
            end
          };
          break;
        }
        case '12h': {
          const end = new Date();
          newTimeRange = {
            start: new Date(end.getTime() - 12 * 60 * 60 * 1000),
            end
          };
          break;
        }
      }
    }

    onFiltersChange({
      ...filters,
      timeRange: newTimeRange,
      activePreset: preset,
    })
  }

  const clearAllFilters = () => {
    // Default to "Today" instead of "All time"
    const now = new Date();
    onFiltersChange({
      groups: [],
      timeRange: {
        start: new Date(now.getFullYear(), now.getMonth(), now.getDate()), // Start of today
        end: now, // Current time
      },
      searchQuery: '',
      activePreset: 'today',
    })
  }

  const selectedGroupsCount = filters.groups.length
  const hasActiveFilters = selectedGroupsCount > 0 || filters.searchQuery.length > 0

  return (
    <Card>
      <CardContent className="p-4">
        {/* Top Row - Search and Quick Actions */}
        <div className="flex flex-col lg:flex-row items-start lg:items-center gap-3 mb-3">
          <div className="relative flex-1 min-w-0">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <Input
              placeholder="Search summaries..."
              value={filters.searchQuery}
              onChange={(e) => handleSearchChange(e.target.value)}
              className="pl-10"
            />
          </div>

          <div className="flex flex-col sm:flex-row items-start sm:items-center gap-2 w-full lg:w-auto">
            <div className="flex items-center gap-2 flex-wrap">
              <DatePickerWithRange
                value={filters.timeRange}
                onChange={handleDateRangeChange}
                activePreset={filters.activePreset}
                onPresetChange={handlePresetChange}
              />
              {filters.activePreset && (
                <Badge variant="secondary" className="gap-1 whitespace-nowrap">
                  {filters.activePreset === '6h' ? 'Last 6h' :
                    filters.activePreset === '12h' ? 'Last 12h' :
                      filters.activePreset === 'morning' ? 'Morning' :
                        filters.activePreset.charAt(0).toUpperCase() + filters.activePreset.slice(1)}
                  <button
                    onClick={() => handlePresetChange(undefined)}
                    className="ml-1 hover:bg-muted rounded-full p-0.5 transition-colors"
                    aria-label="Clear preset"
                  >
                    <X className="h-2.5 w-2.5" />
                  </button>
                </Badge>
              )}
            </div>

            <Popover open={open} onOpenChange={setOpen}>
              <PopoverTrigger asChild>
                <Button variant="outline" className="gap-2 w-full sm:w-auto" aria-haspopup="listbox" aria-expanded={open}>
                  <Filter className="h-4 w-4" />
                  Groups
                  {hasActiveFilters && (
                    <Badge variant="secondary" className="ml-1">
                      {selectedGroupsCount}
                    </Badge>
                  )}
                </Button>
              </PopoverTrigger>

              <PopoverContent className="w-[320px] p-2 max-h-[60vh] overflow-hidden" align="end">
                <div className="flex flex-col h-full gap-2" role="listbox" aria-label="Groups">
                  {/* Sticky search */}
                  <div className="sticky top-0 bg-popover z-10 pb-2">
                    <div className="relative">
                      <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                      <Input
                        value={groupQuery}
                        onChange={(e) => setGroupQuery(e.target.value)}
                        placeholder="Search groups..."
                        className="pl-9 h-9"
                        aria-label="Search groups"
                      />
                    </div>
                  </div>

                  {/* Scroll area list */}
                  <div ref={listRef} className="overflow-y-auto space-y-1 pr-1" style={{ scrollbarGutter: 'stable both-edges' }}>
                    {filteredGroups.length === 0 ? (
                      <div className="text-sm text-muted-foreground py-6 text-center">No groups found</div>
                    ) : (
                      filteredGroups.map((group) => {
                        const selected = filters.groups.includes(group.id)
                        return (
                          <button
                            key={group.id}
                            role="option"
                            aria-selected={selected}
                            onClick={() => handleGroupToggle(group.id, !selected)}
                            className={`w-full flex items-center justify-between h-10 px-2 rounded-md text-left hover:bg-muted/60 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2`}
                          >
                            <span className="truncate text-sm flex items-center gap-2">
                              <MessageSquare className="h-4 w-4" style={{ color: colorForGroup(group.id) }} />
                              {group.name}
                            </span>
                            {selected && <Check className="h-4 w-4 text-primary" />}
                          </button>
                        )
                      })
                    )}
                  </div>

                  {/* Footer chips and actions */}
                  <div className="sticky bottom-0 bg-popover pt-2 border-t">
                    {filters.groups.length > 0 && (
                      <div className="flex flex-wrap gap-1 mb-2 max-h-16 overflow-y-auto pr-1">
                        {filters.groups.map((groupId) => {
                          const group = groups.find(g => g.id === groupId)
                          if (!group) return null
                          return (
                            <Badge key={groupId} variant="secondary" className="gap-1">
                              {group.name}
                              <button
                                className="h-4 w-4 p-0 ml-1 rounded hover:bg-muted/60"
                                aria-label={`Remove ${group.name}`}
                                onClick={() => handleGroupToggle(groupId, false)}
                              >
                                <X className="h-3 w-3" />
                              </button>
                            </Badge>
                          )
                        })}
                      </div>
                    )}
                    <div className="flex items-center justify-end pb-2">
                      <Button variant="ghost" size="sm" onClick={handleClearGroups} disabled={filters.groups.length === 0}>
                        Clear
                      </Button>
                    </div>
                  </div>
                </div>
              </PopoverContent>
            </Popover>
            <Button variant="outline" onClick={clearAllFilters}>Clear All</Button>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
