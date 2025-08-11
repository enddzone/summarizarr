'use client'

import { format, formatDistanceToNow } from 'date-fns'
import { MessageSquare, Clock } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import ReactMarkdown from 'react-markdown'
import { cleanSummaryText } from '@/lib/summary-utils'
import type { Summary } from '@/types'
import { useRef, useEffect, useState } from 'react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
  DialogDescription,
} from '@/components/ui/dialog'
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

interface SummaryCardsProps {
  summaries: Summary[]
  onDelete?: (id: number) => Promise<void> | void
}

export function SummaryCards({ summaries, onDelete }: SummaryCardsProps) {
  // Stable color palette (Flat UI inspired) to differentiate groups
  const palette = [
    '#3498db', // Peter River
    '#1abc9c', // Turquoise
    '#9b59b6', // Amethyst
    '#e67e22', // Carrot
    '#e74c3c', // Alizarin
    '#2ecc71', // Emerald
    '#f1c40f', // Sun Flower
    '#34495e', // Wet Asphalt
    '#16a085', // Green Sea
    '#2980b9', // Belize Hole
  ] as const

  const colorForGroup = (groupId: number) =>
    palette[Math.abs(groupId) % palette.length]

  // Individual card overflow detection
  const CardWithOverflowDetection = ({ summary, groupColor, onDelete }: { summary: Summary, groupColor: string, onDelete?: (id: number) => void | Promise<void> }) => {
    const contentRef = useRef<HTMLDivElement>(null)
    const [showMore, setShowMore] = useState(false)
    const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)

    useEffect(() => {
      if (contentRef.current) {
        const element = contentRef.current
        
        // Use ResizeObserver to detect when content size changes
        const resizeObserver = new ResizeObserver(() => {
          // Use requestAnimationFrame to ensure measurement happens after CSS is applied
          requestAnimationFrame(() => {
            setShowMore(element.scrollHeight > element.clientHeight)
          })
        })
        
        resizeObserver.observe(element)
        
        // Also check immediately after a short delay to catch initial render
        const timeoutId = setTimeout(() => {
          requestAnimationFrame(() => {
            setShowMore(element.scrollHeight > element.clientHeight)
          })
        }, 100)
        
        return () => {
          resizeObserver.disconnect()
          clearTimeout(timeoutId)
        }
      }
    }, [summary.text])

    const handleDeleteClick = (e: React.MouseEvent) => {
      e.preventDefault()
      e.stopPropagation()
      setDeleteDialogOpen(true)
    }

    const handleDeleteConfirm = async () => {
      try {
        if (onDelete) {
          await onDelete(summary.id)
        }
        setDeleteDialogOpen(false)
      } catch (error) {
        console.error('Delete failed:', error)
        // Keep dialog open on error so user can retry
      }
    }

    const text = cleanSummaryText(summary.text)

    return (
      <Dialog>
        <DialogTrigger asChild>
          <Card
            role="button"
            className="relative min-w-[18rem] h-full max-h-[260px] overflow-hidden flex flex-col transition-all duration-200 hover:scale-[1.01] hover:shadow-[0_10px_44px_rgba(52,152,219,0.25)] hover:ring-2 hover:ring-primary/50 hover:ring-offset-2 hover:ring-offset-background cursor-pointer"
          >
            {onDelete && (
              <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
                <AlertDialogTrigger asChild>
                  <button
                    className="absolute right-2 top-2 z-10 rounded-full p-1.5 text-muted-foreground/70 hover:text-destructive hover:bg-destructive/10 transition-colors"
                    aria-label="Delete summary"
                    onClick={handleDeleteClick}
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
                      onClick={handleDeleteConfirm}
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
                <CardTitle className="text-base flex items-center gap-2 truncate">
                  <MessageSquare className="h-4 w-4 flex-shrink-0" style={{ color: groupColor }} />
                  <span className="truncate">
                    {summary.group_name || `Group ${summary.group_id}`}
                  </span>
                </CardTitle>
              </div>
              <div className="flex items-center gap-2 text-xs text-muted-foreground">
                <Clock className="h-3 w-3" />
                {formatDistanceToNow(new Date(summary.created_at), { addSuffix: true })}
              </div>
            </CardHeader>
            <CardContent className="pt-0 flex-1 relative">
              <div ref={contentRef} className="prose prose-sm max-w-none dark:prose-invert overflow-hidden">
                <ReactMarkdown
                  components={{
                    p: ({ children }) => (
                      <p className="text-sm leading-relaxed line-clamp-4 mb-2">{children}</p>
                    ),
                    ul: ({ children }) => (
                      <ul className="text-sm space-y-1 mb-3 pl-4">{children}</ul>
                    ),
                    li: ({ children }) => (
                      <li className="text-sm leading-relaxed text-muted-foreground">{children}</li>
                    ),
                    h1: ({ children }) => (
                      <div className="border-t border-border pt-2 mt-2">
                        <h3 className="text-sm font-semibold mb-1 text-foreground">{children}</h3>
                      </div>
                    ),
                    h2: ({ children }) => (
                      <div className="border-t border-border pt-2 mt-2">
                        <h3 className="text-sm font-semibold mb-1 text-foreground">{children}</h3>
                      </div>
                    ),
                    h3: ({ children }) => (
                      <div className="border-t border-border pt-2 mt-2">
                        <h3 className="text-sm font-semibold mb-1 text-foreground">{children}</h3>
                      </div>
                    ),
                  }}
                >
                  {text}
                </ReactMarkdown>
              </div>
              {/* Enhanced gradient + more indicator for better visibility */}
              {showMore && (
                <div className="pointer-events-none absolute inset-x-0 bottom-0 h-12 bg-gradient-to-t from-background via-background/90 to-transparent flex items-end justify-center pb-2">
                  <div className="bg-primary/10 text-primary border border-primary/20 px-2 py-1 rounded-full text-xs font-medium shadow-sm">
                    Click to read more...
                  </div>
                </div>
              )}
            </CardContent>
          </Card>
        </DialogTrigger>
        <DialogContent className="w-[92vw] max-w-3xl max-h-[85vh] overflow-y-auto p-0 data-[state=open]:animate-pop-out">
          <DialogHeader className="p-6 pb-3">
            <DialogTitle className="flex items-center gap-2">
              <MessageSquare className="h-5 w-5" style={{ color: groupColor }} />
              {summary.group_name || `Group ${summary.group_id}`}
            </DialogTitle>
            <DialogDescription className="flex items-center gap-2 pt-1">
              <Clock className="h-4 w-4" />
              <span>
                {format(new Date(summary.start), 'LLL dd, y')} – {format(new Date(summary.end), 'LLL dd, y')}
              </span>
            </DialogDescription>
          </DialogHeader>
          <div className="px-6 pb-6">
            <div className="prose max-w-none dark:prose-invert">
              <ReactMarkdown
                components={{
                  p: ({ children }) => (
                    <p className="text-sm leading-relaxed mb-3">{children}</p>
                  ),
                  ul: ({ children }) => (
                    <ul className="text-sm space-y-2 mb-4 pl-4">{children}</ul>
                  ),
                  li: ({ children }) => (
                    <li className="text-sm leading-relaxed text-muted-foreground">{children}</li>
                  ),
                  h1: ({ children }) => (
                    <div className="border-t border-border pt-3 mt-4 first:border-t-0 first:pt-0 first:mt-0">
                      <h3 className="text-base font-semibold mb-2 text-foreground">{children}</h3>
                    </div>
                  ),
                  h2: ({ children }) => (
                    <div className="border-t border-border pt-3 mt-4 first:border-t-0 first:pt-0 first:mt-0">
                      <h3 className="text-base font-semibold mb-2 text-foreground">{children}</h3>
                    </div>
                  ),
                  h3: ({ children }) => (
                    <div className="border-t border-border pt-3 mt-4 first:border-t-0 first:pt-0 first:mt-0">
                      <h3 className="text-base font-semibold mb-2 text-foreground">{children}</h3>
                    </div>
                  ),
                }}
              >
                {text}
              </ReactMarkdown>
            </div>
          </div>
        </DialogContent>
      </Dialog>
    )
  }

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
      {summaries.map((summary) => (
        <CardWithOverflowDetection
          key={summary.id}
          summary={summary}
          groupColor={colorForGroup(summary.group_id)}
          onDelete={onDelete}
        />
      ))}
    </div>
  )
}
