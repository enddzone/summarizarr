'use client'

import { Moon, Sun, LayoutGrid, List, ArrowUpDown, Download, Settings } from 'lucide-react'
import { useTheme } from 'next-themes'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Badge } from '@/components/ui/badge'
import type { ViewMode, SortOrder, SignalConfig } from '@/types'

interface HeaderProps {
  viewMode: ViewMode
  onViewModeChange: (mode: ViewMode) => void
  sortOrder: SortOrder
  onSortOrderChange: (order: SortOrder) => void
  onExport: () => void
  onSignalSetup: () => void
  signalConfig: SignalConfig
}

export function Header({
  viewMode,
  onViewModeChange,
  sortOrder,
  onSortOrderChange,
  onExport,
  onSignalSetup,
  signalConfig,
}: HeaderProps) {
  const { setTheme, theme } = useTheme()

  return (
    <header className="sticky top-0 z-50 w-full border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="container mx-auto px-4 flex h-16 items-center justify-between">
        <div className="flex items-center space-x-4">
          <h1 className="text-2xl font-bold text-primary">Summarizarr</h1>
          <Badge variant={signalConfig.isRegistered ? "default" : "secondary"} className="flex items-center gap-1.5">
            {signalConfig.isRegistered && (
              <div className="w-2 h-2 bg-green-500 rounded-full animate-pulse" />
            )}
            {signalConfig.isRegistered ? "Signal Connected" : "Signal Not Connected"}
          </Badge>
        </div>

        <div className="flex items-center space-x-2">
          {/* View Mode Toggle */}
          <div className="flex items-center border rounded-lg p-1">
            <Button
              variant={viewMode === 'timeline' ? 'default' : 'ghost'}
              size="sm"
              onClick={() => onViewModeChange('timeline')}
              className="px-3"
            >
              <List className="h-4 w-4" />
              Timeline
            </Button>
            <Button
              variant={viewMode === 'cards' ? 'default' : 'ghost'}
              size="sm"
              onClick={() => onViewModeChange('cards')}
              className="px-3"
            >
              <LayoutGrid className="h-4 w-4" />
              Cards
            </Button>
          </div>

          {/* Sort Order */}
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="outline" size="sm">
                <ArrowUpDown className="h-4 w-4 mr-2" />
                {sortOrder === 'newest' ? 'Newest First' : 'Oldest First'}
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={() => onSortOrderChange('newest')}>
                Newest First
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => onSortOrderChange('oldest')}>
                Oldest First
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>

          {/* Export */}
          <Button variant="outline" size="sm" onClick={onExport}>
            <Download className="h-4 w-4 mr-2" />
            Export
          </Button>

          {/* Signal Setup */}
          <Button variant="outline" size="sm" onClick={onSignalSetup}>
            <Settings className="h-4 w-4 mr-2" />
            Signal Setup
          </Button>

          {/* Theme Toggle */}
          <Button
            variant="outline"
            size="sm"
            onClick={() => setTheme(theme === 'light' ? 'dark' : 'light')}
          >
            <Sun className="h-4 w-4 rotate-0 scale-100 transition-all dark:-rotate-90 dark:scale-0" />
            <Moon className="absolute h-4 w-4 rotate-90 scale-0 transition-all dark:rotate-0 dark:scale-100" />
            <span className="sr-only">Toggle theme</span>
          </Button>
        </div>
      </div>
    </header>
  )
}
