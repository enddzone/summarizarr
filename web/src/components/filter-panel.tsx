'use client'

import { useState } from 'react'
import { Search, Filter, X, Calendar } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Checkbox } from '@/components/ui/checkbox'
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

  const handleDateRangeChange = (range: { start: Date; end: Date } | undefined) => {
    if (range) {
      console.log('Date range changed:', range)
      onFiltersChange({
        ...filters,
        timeRange: range,
      })
    }
  }

  const clearAllFilters = () => {
    onFiltersChange({
      groups: [],
      timeRange: {
        start: new Date(Date.now() - 24 * 60 * 60 * 1000), // Last 24 hours
        end: new Date(),
      },
      searchQuery: '',
    })
  }

  const selectedGroupsCount = filters.groups.length
  const hasActiveFilters = selectedGroupsCount > 0 || filters.searchQuery.length > 0

  return (
    <Card>
      <CardContent className="p-4">
        {/* Top Row - Search and Quick Actions */}
        <div className="flex flex-col lg:flex-row items-start lg:items-center gap-4 mb-4">
          <div className="relative flex-1 min-w-0">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <Input
              placeholder="Search summaries..."
              value={filters.searchQuery}
              onChange={(e) => handleSearchChange(e.target.value)}
              className="pl-10"
            />
          </div>
          
          <div className="flex flex-col sm:flex-row items-start sm:items-center gap-3 w-full lg:w-auto">
            <DatePickerWithRange
              value={filters.timeRange}
              onChange={handleDateRangeChange}
            />

            <Popover>
              <PopoverTrigger asChild>
                <Button variant="outline" className="gap-2 w-full sm:w-auto">
                  <Filter className="h-4 w-4" />
                  Groups
                  {hasActiveFilters && (
                    <Badge variant="secondary" className="ml-1">
                      {selectedGroupsCount}
                    </Badge>
                  )}
                </Button>
              </PopoverTrigger>

              <PopoverContent className="w-96 p-4" align="end">
                <div className="space-y-4">
                  {/* Group Filters */}
                  <div>
                    <div className="flex items-center justify-between mb-3">
                      <h3 className="text-sm font-medium">Select Groups</h3>
                      {hasActiveFilters && (
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={clearAllFilters}
                          className="h-8 px-2"
                        >
                          <X className="h-3 w-3 mr-1" />
                          Clear All
                        </Button>
                      )}
                    </div>
                    
                    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-3">
                      {groups.map((group) => (
                        <div key={group.id} className="flex items-center space-x-2 p-2 rounded-md hover:bg-muted/50">
                          <Checkbox
                            id={`group-${group.id}`}
                            checked={filters.groups.includes(group.id)}
                            onCheckedChange={(checked) => 
                              handleGroupToggle(group.id, checked as boolean)
                            }
                          />
                          <label
                            htmlFor={`group-${group.id}`}
                            className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70 cursor-pointer flex-1"
                          >
                            {group.name}
                          </label>
                        </div>
                      ))}
                    </div>
                  </div>

                  {/* Active Filter Tags */}
                  {hasActiveFilters && (
                    <div>
                      <h4 className="text-sm font-medium mb-2">Active Filters</h4>
                      <div className="flex flex-wrap gap-2">
                        {filters.groups.map((groupId) => {
                          const group = groups.find(g => g.id === groupId)
                          if (!group) return null
                          
                          return (
                            <Badge key={groupId} variant="secondary" className="gap-1">
                              {group.name}
                              <Button
                                variant="ghost"
                                size="sm"
                                className="h-4 w-4 p-0 hover:bg-transparent"
                                onClick={() => handleGroupToggle(groupId, false)}
                              >
                                <X className="h-3 w-3" />
                              </Button>
                            </Badge>
                          )
                        })}
                      </div>
                    </div>
                  )}
                </div>
              </PopoverContent>
            </Popover>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
