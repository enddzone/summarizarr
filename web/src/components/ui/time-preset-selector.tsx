'use client'

import * as React from "react"
import { Clock, ChevronDown } from "lucide-react"
import { Button } from "@/components/ui/button"
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

interface TimePresetSelectorProps {
    activePreset?: 'all-time' | 'today' | 'yesterday' | 'morning' | '6h' | '12h'
    onPresetChange?: (preset: 'all-time' | 'today' | 'yesterday' | 'morning' | '6h' | '12h') => void
    className?: string
}

const timePresets = [
    { value: 'today' as const, label: 'Today' },
    { value: 'yesterday' as const, label: 'Yesterday' },
    { value: 'morning' as const, label: 'Morning' },
    { value: '6h' as const, label: 'Last 6h' },
    { value: '12h' as const, label: 'Last 12h' },
    { value: 'all-time' as const, label: 'All time' },
]

export function TimePresetSelector({ activePreset, onPresetChange, className }: TimePresetSelectorProps) {
    const currentPreset = timePresets.find(p => p.value === activePreset)

    return (
        <DropdownMenu>
            <DropdownMenuTrigger asChild>
                <Button variant="outline" className={`w-full sm:w-[180px] justify-between ${className}`}>
                    <div className="flex items-center">
                        <Clock className="h-4 w-4 mr-2" />
                        {currentPreset?.label || 'Today'}
                    </div>
                    <ChevronDown className="h-4 w-4" />
                </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="start" className="w-48">
                {timePresets.map((preset) => (
                    <DropdownMenuItem
                        key={preset.value}
                        onClick={() => onPresetChange?.(preset.value)}
                        className={activePreset === preset.value ? 'bg-accent' : ''}
                    >
                        {preset.label}
                    </DropdownMenuItem>
                ))}
            </DropdownMenuContent>
        </DropdownMenu>
    )
}
