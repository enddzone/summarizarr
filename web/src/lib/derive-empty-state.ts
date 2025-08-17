import { EmptyStateVariant } from '@/components/empty-state'
import type { Summary, Group, FilterOptions, SignalConfig } from '@/types'

export interface FetchErrorInfo {
    scope: 'summaries' | 'groups' | 'signal'
    message: string
    status?: number
}

interface DeriveParams {
    signalConfig: Partial<SignalConfig & { connected?: boolean; status?: string }>
    summaries: Summary[]
    groups: Group[]
    filters: FilterOptions
    fetchError?: FetchErrorInfo
}

export interface DerivedEmptyStateMeta {
    errorDetail?: string
    status?: number
    nextSummaryEta?: string
}

export interface DerivedEmptyState {
    variant: EmptyStateVariant
    primaryLabel?: string
    secondaryLabel?: string
    meta?: DerivedEmptyStateMeta
    reason: string
}

// Pure logic function mapping current dashboard state to an EmptyState variant
export function deriveEmptyState(params: DeriveParams): DerivedEmptyState | null {
    const { signalConfig, summaries, groups, filters, fetchError } = params

    // If there is data, no empty state
    if (summaries.length > 0) return null

    // Error takes precedence
    if (fetchError) {
        return {
            variant: 'error',
            primaryLabel: 'Retry',
            meta: { errorDetail: fetchError.message, status: fetchError.status },
            reason: 'Fetch error present',
        }
    }

    // Needs Signal registration
    if (!signalConfig?.isRegistered) {
        return {
            variant: 'needs-signal',
            primaryLabel: 'Set Up Signal',
            secondaryLabel: 'Learn More',
            reason: 'Signal not registered',
        }
    }

    // Registered but filtered out results
    const isAllTime = filters.timeRange.start.getTime() === 0 && filters.activePreset === 'all-time'
    if (signalConfig.isRegistered && summaries.length === 0 && !isAllTime && (filters.groups.length > 0 || filters.searchQuery.trim() || filters.activePreset !== 'today')) {
        return {
            variant: 'filters-empty',
            primaryLabel: 'Clear Filters',
            secondaryLabel: 'All Time',
            reason: 'Filters excluded results',
        }
    }

    // Registered & have groups but no summaries yet
    if (signalConfig.isRegistered && groups.length > 0) {
        return {
            variant: 'no-results',
            primaryLabel: 'Refresh',
            secondaryLabel: 'All Time',
            reason: 'No summaries for time range',
        }
    }

    // Registered but no groups yet
    if (signalConfig.isRegistered && groups.length === 0) {
        return {
            variant: 'waiting',
            primaryLabel: 'Refresh',
            reason: 'Waiting for first messages',
        }
    }

    // Fallback (should not normally reach)
    return {
        variant: 'no-results',
        primaryLabel: 'Refresh',
        reason: 'Fallback path',
    }
}
