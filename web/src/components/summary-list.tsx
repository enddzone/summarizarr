'use client'

import { formatDistanceToNow } from 'date-fns'
import { MessageSquare, Clock } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import ReactMarkdown from 'react-markdown'
import { cleanSummaryText } from '@/lib/summary-utils'
import type { Summary } from '@/types'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'

interface SummaryListProps {
  summaries: Summary[]
  onDelete?: (id: number) => Promise<void> | void
}

// Shared header component factory to eliminate duplication
const createHeaderComponent = (
  level: 'h1' | 'h2' | 'h3',
  containerClass: string,
  textClass: string,
  isFirst = false
) => {
  const Component = (props: { children?: React.ReactNode }) => (
    <div className={`${containerClass} ${isFirst ? 'first:border-t-0 first:pt-0 first:mt-0' : ''}`}>
      <h3 className={textClass}>{props.children}</h3>
    </div>
  )
  Component.displayName = `HeaderComponent${level.toUpperCase()}`
  return Component
}

// Shared markdown component configurations
const createMarkdownComponents = () => {
  return {
    p: (props: { children?: React.ReactNode }) => (
      <p className="text-sm leading-relaxed mb-3">
        {props.children}
      </p>
    ),
    ul: (props: { children?: React.ReactNode }) => (
      <ul className="text-sm space-y-2 mb-4 pl-4">
        {props.children}
      </ul>
    ),
    li: (props: { children?: React.ReactNode }) => (
      <li className="text-sm leading-relaxed text-muted-foreground">{props.children}</li>
    ),
    h1: createHeaderComponent(
      'h1',
      'border-t border-border pt-3 mt-4',
      'text-base font-semibold mb-2 text-foreground',
      true
    ),
    h2: createHeaderComponent(
      'h2',
      'border-t border-border pt-3 mt-4',
      'text-base font-semibold mb-2 text-foreground',
      true
    ),
    h3: createHeaderComponent(
      'h3',
      'border-t border-border pt-3 mt-4',
      'text-base font-semibold mb-2 text-foreground',
      true
    ),
  }
}

export function SummaryList({ summaries, onDelete }: SummaryListProps) {
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
  return (
    <div className="space-y-4">
      {summaries.map((summary) => (
  <Card key={summary.id} className="relative hover:shadow-[0_8px_28px_rgba(52,152,219,0.18)] transition-shadow">
          {onDelete && (
            <AlertDialog>
              <AlertDialogTrigger asChild>
                <button
                  className="absolute right-2 top-2 z-10 rounded p-1 text-muted-foreground/80 hover:text-foreground hover:bg-muted/50"
                  aria-label="Delete summary"
                  onClick={(e) => {
                    e.preventDefault();
                    e.stopPropagation();
                  }}
                >
                  <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" className="h-4 w-4" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>
                </button>
              </AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>Delete summary?</AlertDialogTitle>
                  <AlertDialogDescription>This action cannot be undone.</AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>Cancel</AlertDialogCancel>
                  <AlertDialogAction 
                    onClick={async () => {
                      try {
                        await onDelete(summary.id)
                      } catch (error) {
                        console.error('Delete failed:', error)
                      }
                    }} 
                    className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                  >
                    Delete
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          )}
          <CardHeader className="pb-3">
            <div className="flex items-center justify-between">
              <CardTitle className="text-lg flex items-center gap-2">
                <MessageSquare className="h-5 w-5" style={{ color: colorForGroup(summary.group_id) }} />
                {summary.group_name || `Group ${summary.group_id}`}
              </CardTitle>
              <div className="flex items-center gap-1 text-sm text-muted-foreground">
                <Clock className="h-4 w-4" />
                {formatDistanceToNow(new Date(summary.created_at), { addSuffix: true })}
              </div>
            </div>
          </CardHeader>
          <CardContent>
            <div className="prose prose-sm max-w-none dark:prose-invert">
              <ReactMarkdown components={createMarkdownComponents()}>
                {cleanSummaryText(summary.text)}
              </ReactMarkdown>
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  )
}
