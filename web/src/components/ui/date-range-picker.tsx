'use client'

import * as React from "react"
import { CalendarIcon } from "lucide-react"
import { format } from "date-fns"
import { DateRange } from "react-day-picker"
import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import { Calendar } from "@/components/ui/calendar"
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover"

interface DatePickerWithRangeProps {
  value: { start: Date; end: Date }
  onChange: (range: { start: Date; end: Date } | undefined) => void
  className?: string
  activePreset?: 'all-time' | 'today' | 'yesterday' | 'morning' | '6h' | '12h'
  onPresetChange?: (preset: 'all-time' | 'today' | 'yesterday' | 'morning' | '6h' | '12h' | undefined) => void
}

export function DatePickerWithRange({ value, onChange, className, activePreset: externalActivePreset, onPresetChange }: DatePickerWithRangeProps) {
  const [date, setDate] = React.useState<DateRange>({
    from: value.start,
    to: value.end,
  })
  const [activePreset, setActivePreset] = React.useState<
    'all-time' | 'today' | 'yesterday' | 'morning' | '6h' | '12h' | undefined
  >(externalActivePreset)

  React.useEffect(() => {
    setDate({ from: value.start, to: value.end })
  }, [value.start, value.end])

  React.useEffect(() => {
    setActivePreset(externalActivePreset)
  }, [externalActivePreset])

  const updateActivePreset = (preset: 'all-time' | 'today' | 'yesterday' | 'morning' | '6h' | '12h' | undefined) => {
    setActivePreset(preset)
    onPresetChange?.(preset)
  }

  const handleSelect = (newDate?: DateRange) => {
    // Update local state eagerly so the UI reflects the first click (from-day)
    if (newDate) {
      setDate(newDate)
    }

    if (!newDate || !newDate.from) {
      // If cleared, reset to controlled value and propagate current value
      setDate({ from: value.start, to: value.end })
      onChange({ start: value.start, end: value.end })
      updateActivePreset(undefined)
      return;
    }

    // When only 'from' is selected (first click), don't collapse both to same day for the parent.
    // Wait until 'to' gets picked to notify the parent, fixing the stuck single-date issue.
    if (newDate.from && newDate.to) {
      onChange({ start: newDate.from, end: newDate.to })
      updateActivePreset(undefined)
    }
  }

  return (
    <div className={cn("grid gap-2", className)}>
      <Popover>
        <PopoverTrigger asChild>
          <Button
            id="date"
            variant={"outline"}
            className={cn(
              "w-full sm:w-[300px] justify-start text-left font-normal text-sm",
              !date && "text-muted-foreground"
            )}
          >
            <CalendarIcon className="mr-2 h-4 w-4 flex-shrink-0" />
            <span className="truncate">
              {date?.from ? (
                date.to ? (
                  <>
                    <span className="hidden sm:inline">
                      {format(date.from, "LLL dd, y")} -{" "}
                      {format(date.to, "LLL dd, y")}
                    </span>
                    <span className="sm:hidden">
                      {format(date.from, "MMM dd")} - {format(date.to, "MMM dd")}
                    </span>
                  </>
                ) : (
                  <>
                    <span className="hidden sm:inline">{format(date.from, "LLL dd, y")}</span>
                    <span className="sm:hidden">{format(date.from, "MMM dd")}</span>
                  </>
                )
              ) : (
                <span>Pick a date range</span>
              )}
            </span>
          </Button>
        </PopoverTrigger>
        <PopoverContent className="w-auto p-0 max-w-[95vw] sm:max-w-none" align="start">
          <div className="p-2 border-b bg-background/80">
            <div className="flex flex-wrap gap-2">
              <Button variant={activePreset === 'all-time' ? 'default' : 'secondary'} size="sm" onClick={() => {
                const end = new Date();
                const start = new Date(0); // Unix epoch start
                const newRange = { from: start, to: end };
                setDate(newRange);
                onChange({ start, end });
                updateActivePreset('all-time')
              }}>All time</Button>
              <Button variant={activePreset === 'today' ? 'default' : 'secondary'} size="sm" onClick={() => {
                const now = new Date();
                const start = new Date(now.getFullYear(), now.getMonth(), now.getDate());
                const newRange = { from: start, to: now };
                setDate(newRange);
                onChange({ start, end: now });
                updateActivePreset('today')
              }}>Today</Button>
              <Button variant={activePreset === 'yesterday' ? 'default' : 'secondary'} size="sm" onClick={() => {
                const now = new Date();
                const y = new Date(now);
                y.setDate(now.getDate() - 1);
                const start = new Date(y.getFullYear(), y.getMonth(), y.getDate());
                const end = new Date(y.getFullYear(), y.getMonth(), y.getDate(), 23, 59, 59, 999);
                const newRange = { from: start, to: end };
                setDate(newRange);
                onChange({ start, end });
                updateActivePreset('yesterday')
              }}>Yesterday</Button>
              <Button variant={activePreset === 'morning' ? 'default' : 'secondary'} size="sm" onClick={() => {
                const now = new Date();
                const start = new Date(now.getFullYear(), now.getMonth(), now.getDate(), 6, 0, 0);
                const end = new Date(now.getFullYear(), now.getMonth(), now.getDate(), 12, 0, 0);
                const newRange = { from: start, to: end };
                setDate(newRange);
                onChange({ start, end });
                updateActivePreset('morning')
              }}>Morning</Button>
              <Button variant={activePreset === '6h' ? 'default' : 'secondary'} size="sm" onClick={() => {
                const end = new Date();
                const start = new Date(end.getTime() - 6 * 60 * 60 * 1000);
                const newRange = { from: start, to: end };
                setDate(newRange);
                onChange({ start, end });
                updateActivePreset('6h')
              }}>Last 6h</Button>
              <Button variant={activePreset === '12h' ? 'default' : 'secondary'} size="sm" onClick={() => {
                const end = new Date();
                const start = new Date(end.getTime() - 12 * 60 * 60 * 1000);
                const newRange = { from: start, to: end };
                setDate(newRange);
                onChange({ start, end });
                updateActivePreset('12h')
              }}>Last 12h</Button>
            </div>
          </div>
          <div className="sm:hidden">
            <Calendar
              initialFocus
              mode="range"
              defaultMonth={date?.from}
              selected={date}
              onSelect={handleSelect}
              numberOfMonths={1}
            />
          </div>
          <div className="hidden sm:block">
            <Calendar
              initialFocus
              mode="range"
              defaultMonth={date?.from}
              selected={date}
              onSelect={handleSelect}
              numberOfMonths={2}
            />
          </div>
        </PopoverContent>
      </Popover>
    </div>
  )
}
