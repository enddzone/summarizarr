"use client"

import { ReactNode } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { AlertCircle, Smartphone, Hourglass, FilterX, Inbox } from 'lucide-react'
import type { DerivedEmptyStateMeta } from '@/lib/derive-empty-state'

export type EmptyStateVariant =
    | 'needs-signal'
    | 'waiting'
    | 'no-results'
    | 'filters-empty'
    | 'error'

interface EmptyStateProps {
    variant: EmptyStateVariant
    title?: string
    message?: string | ReactNode
    primaryLabel?: string
    secondaryLabel?: string
    onPrimary?: () => void
    onSecondary?: () => void
    meta?: DerivedEmptyStateMeta
    className?: string
}

const DEFAULT_COPY: Record<EmptyStateVariant, { title: string; message: ReactNode; primary: string; secondary?: string; icon: ReactNode; }> = {
    'needs-signal': {
        title: 'Connect Signal to Get Started',
        message: (
            <>
                Register your phone number to start ingesting group messages. Once connected, summaries will appear automatically.
            </>
        ),
        primary: 'Set Up Signal',
        secondary: 'Learn More',
        icon: <Smartphone className="h-8 w-8 text-primary" />,
    },
    waiting: {
        title: 'Waiting for First Messages',
        message: 'Signal is connected. Send some messages in your Signal group and the next summary cycle will capture them.',
        primary: 'Refresh',
        secondary: undefined,
        icon: <Hourglass className="h-8 w-8 text-primary" />,
    },
    'no-results': {
        title: 'No Summaries Yet',
        message: 'You have groups configured, but no summaries exist for the selected time range.',
        primary: 'Refresh',
        secondary: 'All Time',
        icon: <Inbox className="h-8 w-8 text-primary" />,
    },
    'filters-empty': {
        title: 'Nothing Matches Your Filters',
        message: 'Try clearing filters or expanding the time range to see summaries.',
        primary: 'Clear Filters',
        secondary: 'All Time',
        icon: <FilterX className="h-8 w-8 text-primary" />,
    },
    error: {
        title: 'Unable to Load Summaries',
        message: 'An error occurred while fetching data. You can retry below.',
        primary: 'Retry',
        secondary: undefined,
        icon: <AlertCircle className="h-8 w-8 text-destructive" />,
    },
}

export function EmptyState({
    variant,
    title,
    message,
    primaryLabel,
    secondaryLabel,
    onPrimary,
    onSecondary,
    meta,
    className = '',
}: EmptyStateProps) {
    const copy = DEFAULT_COPY[variant]
    const resolvedTitle = title || copy.title
    const resolvedMessage = message || copy.message
    const resolvedPrimary = primaryLabel || copy.primary
    const resolvedSecondary = secondaryLabel || copy.secondary

    // Generate unique IDs for ARIA relationships
    const titleId = `empty-state-title-${variant}`
    const descriptionId = `empty-state-description-${variant}`

    return (
        <div
            className={`w-full flex justify-center py-10 sm:py-16 ${className}`}
            role="status"
            aria-live="polite"
            aria-labelledby={titleId}
            aria-describedby={descriptionId}
        >
            <Card className="max-w-lg w-full mx-4">
                <CardContent className="pt-8 pb-10 flex flex-col items-center text-center space-y-6">
                    <div className="flex flex-col items-center space-y-4">
                        <div aria-hidden="true">
                            {copy.icon}
                        </div>
                        <h2 id={titleId} className="text-xl font-semibold tracking-tight">
                            {resolvedTitle}
                        </h2>
                        <div id={descriptionId} className="text-sm text-muted-foreground leading-relaxed max-w-md">
                            {resolvedMessage}
                            {variant === 'waiting' && meta?.nextSummaryEta && (
                                <div className="mt-2 text-xs text-muted-foreground/80">Next summary in about {meta.nextSummaryEta}.</div>
                            )}
                            {variant === 'error' && meta?.errorDetail && (
                                <div className="mt-2 text-xs text-destructive/80">{meta.errorDetail}</div>
                            )}
                        </div>
                    </div>
                    <div className="flex flex-col sm:flex-row gap-3 w-full justify-center">
                        {resolvedPrimary && (
                            <Button
                                onClick={onPrimary}
                                className="sm:min-w-[140px]"
                                aria-label={`${resolvedPrimary} - ${variant === 'needs-signal' ? 'Opens Signal setup' : resolvedPrimary}`}
                            >
                                {resolvedPrimary}
                            </Button>
                        )}
                        {resolvedSecondary && onSecondary && (
                            <Button
                                variant="outline"
                                onClick={onSecondary}
                                className="sm:min-w-[140px]"
                                aria-label={`${resolvedSecondary} - ${variant === 'needs-signal' ? 'Opens documentation' : resolvedSecondary}`}
                            >
                                {resolvedSecondary}
                            </Button>
                        )}
                    </div>
                </CardContent>
            </Card>
        </div>
    )
}
