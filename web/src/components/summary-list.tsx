'use client'

import { formatDistanceToNow } from 'date-fns'
import { MessageSquare, Clock } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import ReactMarkdown from 'react-markdown'
import { cleanSummaryText } from '@/lib/summary-utils'
import type { Summary } from '@/types'

interface SummaryListProps {
  summaries: Summary[]
}

export function SummaryList({ summaries }: SummaryListProps) {
  return (
    <div className="space-y-4">
      {summaries.map((summary) => (
        <Card key={summary.id} className="hover:shadow-md transition-shadow">
          <CardHeader className="pb-3">
            <div className="flex items-center justify-between">
              <CardTitle className="text-lg flex items-center gap-2">
                <MessageSquare className="h-5 w-5 text-primary" />
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
              <ReactMarkdown>{cleanSummaryText(summary.text)}</ReactMarkdown>
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  )
}
