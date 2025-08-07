'use client'

import { formatDistanceToNow } from 'date-fns'
import { MessageSquare, Clock } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import ReactMarkdown from 'react-markdown'
import { cleanSummaryText } from '@/lib/summary-utils'
import type { Summary } from '@/types'

interface SummaryCardsProps {
  summaries: Summary[]
}

export function SummaryCards({ summaries }: SummaryCardsProps) {
  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
      {summaries.map((summary) => (
        <Card key={summary.id} className="hover:shadow-lg transition-all duration-200 hover:scale-105">
          <CardHeader className="pb-3">
            <div className="flex items-center justify-between">
              <CardTitle className="text-base flex items-center gap-2 truncate">
                <MessageSquare className="h-4 w-4 text-primary flex-shrink-0" />
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
          <CardContent className="pt-0">
            <div className="prose prose-sm max-w-none dark:prose-invert">
              <ReactMarkdown 
                components={{
                  p: ({ children }) => <p className="text-sm leading-relaxed line-clamp-6">{children}</p>,
                }}
              >
                {cleanSummaryText(summary.text)}
              </ReactMarkdown>
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  )
}
