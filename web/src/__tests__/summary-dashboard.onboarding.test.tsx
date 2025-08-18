import { render, screen, waitFor } from '@testing-library/react'
import { SummaryDashboard } from '@/components/summary-dashboard'

// Mock dependencies
jest.mock('@/hooks/use-toast', () => ({
    useToast: () => ({ toast: jest.fn() }),
}))

jest.mock('@/components/header', () => ({
    Header: () => <div data-testid="header">Header</div>,
}))

jest.mock('@/components/filter-panel', () => ({
    FilterPanel: () => <div data-testid="filter-panel">Filter Panel</div>,
}))

jest.mock('@/components/summary-list', () => ({
    SummaryList: () => <div data-testid="summary-list">Summary List</div>,
}))

jest.mock('@/components/summary-cards', () => ({
    SummaryCards: () => <div data-testid="summary-cards">Summary Cards</div>,
}))

jest.mock('@/components/export-dialog', () => ({
    ExportDialog: () => <div data-testid="export-dialog">Export Dialog</div>,
}))

jest.mock('@/components/signal-setup-dialog', () => ({
    SignalSetupDialog: ({ open }: { open: boolean }) =>
        open ? <div data-testid="signal-setup-dialog">Signal Setup Dialog</div> : null,
}))

jest.mock('@/components/empty-state', () => ({
    EmptyState: ({ variant }: { variant: string }) =>
        <div data-testid="empty-state" data-variant={variant}>Empty State: {variant}</div>,
}))

// Mock fetch
const mockFetch = jest.fn()
global.fetch = mockFetch

// Mock sessionStorage
const mockSessionStorage = {
    getItem: jest.fn(),
    setItem: jest.fn(),
    removeItem: jest.fn(),
    clear: jest.fn(),
}
Object.defineProperty(window, 'sessionStorage', {
    value: mockSessionStorage,
})

describe('SummaryDashboard Auto-Open Signal Setup', () => {
    beforeEach(() => {
        jest.clearAllMocks()

        // Default mock responses
        mockFetch.mockImplementation((url: string) => {
            if (url.includes('/api/summaries')) {
                return Promise.resolve({
                    ok: true,
                    json: () => Promise.resolve([]),
                })
            }
            if (url.includes('/api/groups')) {
                return Promise.resolve({
                    ok: true,
                    json: () => Promise.resolve([]),
                })
            }
            if (url.includes('/api/signal/config')) {
                return Promise.resolve({
                    ok: true,
                    json: () => Promise.resolve({
                        phoneNumber: '',
                        isRegistered: false,
                    }),
                })
            }
            return Promise.reject(new Error('Unknown URL'))
        })
    })

    test('auto-opens Signal setup dialog when not registered and not shown this session', async () => {
        mockSessionStorage.getItem.mockReturnValue(null)

        render(<SummaryDashboard />)

        await waitFor(() => {
            expect(screen.getByTestId('signal-setup-dialog')).toBeInTheDocument()
        })

        // Should set the session flag
        expect(mockSessionStorage.setItem).toHaveBeenCalledWith('summarizarr-signal-setup-shown', 'true')
    })

    test('does not auto-open Signal setup dialog when already shown this session', async () => {
        mockSessionStorage.getItem.mockReturnValue('true')

        render(<SummaryDashboard />)

        await waitFor(() => {
            expect(screen.queryByTestId('signal-setup-dialog')).not.toBeInTheDocument()
        })

        // Should not set the session flag again
        expect(mockSessionStorage.setItem).not.toHaveBeenCalled()
    })

    test('does not auto-open Signal setup dialog when already registered', async () => {
        mockSessionStorage.getItem.mockReturnValue(null)

        // Mock signal as registered
        mockFetch.mockImplementation((url: string) => {
            if (url.includes('/api/signal/config')) {
                return Promise.resolve({
                    ok: true,
                    json: () => Promise.resolve({
                        phoneNumber: '+1234567890',
                        isRegistered: true,
                    }),
                })
            }
            // Other mocks remain the same
            if (url.includes('/api/summaries')) {
                return Promise.resolve({
                    ok: true,
                    json: () => Promise.resolve([]),
                })
            }
            if (url.includes('/api/groups')) {
                return Promise.resolve({
                    ok: true,
                    json: () => Promise.resolve([]),
                })
            }
            return Promise.reject(new Error('Unknown URL'))
        })

        render(<SummaryDashboard />)

        await waitFor(() => {
            expect(screen.queryByTestId('signal-setup-dialog')).not.toBeInTheDocument()
        })

        // Should not set the session flag when already registered
        expect(mockSessionStorage.setItem).not.toHaveBeenCalled()
    })

    test('shows error empty state when fetch error occurs', async () => {
        mockFetch.mockImplementation((url: string) => {
            if (url.includes('/api/summaries')) {
                return Promise.resolve({
                    ok: false,
                    status: 500,
                    statusText: 'Internal Server Error',
                })
            }
            if (url.includes('/api/groups')) {
                return Promise.resolve({
                    ok: true,
                    json: () => Promise.resolve([]),
                })
            }
            if (url.includes('/api/signal/config')) {
                return Promise.resolve({
                    ok: true,
                    json: () => Promise.resolve({
                        phoneNumber: '+1234567890',
                        isRegistered: true,
                    }),
                })
            }
            return Promise.reject(new Error('Unknown URL'))
        })

        render(<SummaryDashboard />)

        await waitFor(() => {
            const emptyState = screen.getByTestId('empty-state')
            expect(emptyState).toHaveAttribute('data-variant', 'error')
        })
    })
})
