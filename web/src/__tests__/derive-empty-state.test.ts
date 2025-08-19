import { deriveEmptyState } from '@/lib/derive-empty-state'
import type { Summary, Group, FilterOptions } from '@/types'

const createSummary = (id: number, group: Group): Summary => ({
    id,
    group_id: group.id,
    group_name: group.name,
    text: 'sample',
    start: Date.now().toString(),
    end: Date.now().toString(),
    created_at: Date.now().toString(),
})

const base: { summaries: Summary[]; groups: Group[]; filters: FilterOptions } = {
    summaries: [],
    groups: [],
    filters: {
        groups: [],
        timeRange: { start: new Date(0), end: new Date() },
        searchQuery: '',
        activePreset: 'all-time',
    },
}

describe('deriveEmptyState', () => {
    test('returns needs-signal when not registered', () => {
        const res = deriveEmptyState({
            ...base,
            signalConfig: { isRegistered: false },
        })
        expect(res?.variant).toBe('needs-signal')
    })

    test('returns waiting when registered no groups', () => {
        const res = deriveEmptyState({
            ...base,
            signalConfig: { isRegistered: true },
        })
        expect(res?.variant).toBe('waiting')
    })

    test('returns no-results when registered and groups but no summaries all-time', () => {
        const res = deriveEmptyState({
            ...base,
            groups: [{ id: 1, name: 'Test Group' }],
            signalConfig: { isRegistered: true },
        })
        expect(res?.variant).toBe('no-results')
    })

    test('returns filters-empty when filters exclude results', () => {
        const res = deriveEmptyState({
            ...base,
            groups: [{ id: 1, name: 'Test Group' }],
            signalConfig: { isRegistered: true },
            filters: {
                ...base.filters,
                timeRange: { start: new Date(Date.now() - 3600_000), end: new Date() },
                activePreset: 'today',
                searchQuery: 'foo',
            },
        })
        expect(res?.variant).toBe('filters-empty')
    })

    test('returns error variant when fetch error', () => {
        const res = deriveEmptyState({
            ...base,
            signalConfig: { isRegistered: true },
            fetchError: { scope: 'summaries', message: 'boom' },
        })
        expect(res?.variant).toBe('error')
    })

    test('returns null when summaries present', () => {
        const res = deriveEmptyState({
            ...base,
            signalConfig: { isRegistered: true },
            summaries: [createSummary(1, { id: 1, name: 'Test Group' } as Group)],
        })
        expect(res).toBeNull()
    })
})
