import React from 'react'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { SummaryCards } from '../summary-cards'
import type { Summary } from '@/types'

// Mock ReactMarkdown to avoid ES module issues
jest.mock('react-markdown', () => {
  return function MockReactMarkdown({ children }: { children: string }) {
    return <div data-testid="markdown-content">{children}</div>
  }
})

// Mock the summary utils
jest.mock('@/lib/summary-utils', () => ({
  cleanSummaryText: (text: string) => text,
}))

// Mock date-fns to have predictable outputs
jest.mock('date-fns', () => ({
  format: () => 'Jan 01, 2024',
  formatDistanceToNow: () => '2 hours ago',
}))

const mockSummary: Summary = {
  id: 1,
  group_id: 123,
  group_name: 'Test Group',
  text: '## Key topics discussed\n- Test topic 1\n- Test topic 2\n\n## Important decisions\n- Test decision',
  created_at: '2024-01-01T10:00:00Z',
  start: '2024-01-01T08:00:00Z',
  end: '2024-01-01T10:00:00Z',
}

describe('SummaryCards', () => {
  const user = userEvent.setup()

  beforeEach(() => {
    // Reset all mocks
    jest.clearAllMocks()
    
    // Mock ResizeObserver
    global.ResizeObserver = jest.fn().mockImplementation(() => ({
      observe: jest.fn(),
      unobserve: jest.fn(),
      disconnect: jest.fn(),
    }))
  })

  describe('Basic Rendering', () => {
    it('renders empty state correctly', () => {
      render(<SummaryCards summaries={[]} />)
      expect(screen.queryByRole('button')).toBeNull()
    })

    it('renders single summary card with correct content', () => {
      render(<SummaryCards summaries={[mockSummary]} />)
      
      expect(screen.getByText('Test Group')).toBeInTheDocument()
      expect(screen.getByText('2 hours ago')).toBeInTheDocument()
    })

    it('renders multiple summary cards', () => {
      const summaries = [
        mockSummary,
        { ...mockSummary, id: 2, group_name: 'Another Group' },
      ]
      
      render(<SummaryCards summaries={summaries} />)
      
      expect(screen.getByText('Test Group')).toBeInTheDocument()
      expect(screen.getByText('Another Group')).toBeInTheDocument()
    })

    it('falls back to group ID when group name is not available', () => {
      const summaryWithoutName: Summary = { ...mockSummary, group_name: null }
      render(<SummaryCards summaries={[summaryWithoutName]} />)
      
      expect(screen.getByText('Group 123')).toBeInTheDocument()
    })
  })

  describe('Accessibility Features', () => {
    it('has proper ARIA labels for cards', () => {
      render(<SummaryCards summaries={[mockSummary]} />)
      
      const card = screen.getByRole('button')
      expect(card).toHaveAttribute('aria-label', expect.stringContaining('Open summary for Test Group'))
      expect(card).toHaveAttribute('aria-label', expect.stringContaining('2 hours ago'))
    })

    it('has proper ARIA labels for delete buttons when onDelete provided', () => {
      const mockDelete = jest.fn()
      render(<SummaryCards summaries={[mockSummary]} onDelete={mockDelete} />)
      
      const deleteButton = screen.getByLabelText('Delete summary for Test Group')
      expect(deleteButton).toBeInTheDocument()
    })

    it('supports keyboard navigation for cards', async () => {
      render(<SummaryCards summaries={[mockSummary]} />)
      
      const card = screen.getByRole('button')
      
      // Focus the card
      await user.tab()
      expect(card).toHaveFocus()
      
      // Should have focus ring styles
      expect(card).toHaveClass('focus:ring-2', 'focus:ring-primary')
    })
  })

  describe('Delete Functionality', () => {
    it('shows delete button when onDelete prop is provided', () => {
      const mockDelete = jest.fn()
      render(<SummaryCards summaries={[mockSummary]} onDelete={mockDelete} />)
      
      expect(screen.getByLabelText('Delete summary for Test Group')).toBeInTheDocument()
    })

    it('hides delete button when onDelete prop is not provided', () => {
      render(<SummaryCards summaries={[mockSummary]} />)
      
      expect(screen.queryByLabelText('Delete summary for Test Group')).toBeNull()
    })

    it('opens delete confirmation dialog on delete button click', async () => {
      const mockDelete = jest.fn()
      render(<SummaryCards summaries={[mockSummary]} onDelete={mockDelete} />)
      
      const deleteButton = screen.getByLabelText('Delete summary for Test Group')
      await user.click(deleteButton)
      
      expect(screen.getByText('Delete summary?')).toBeInTheDocument()
      expect(screen.getByText('This action cannot be undone.')).toBeInTheDocument()
    })

    it('calls onDelete when confirmed', async () => {
      const mockDelete = jest.fn()
      render(<SummaryCards summaries={[mockSummary]} onDelete={mockDelete} />)
      
      // Open delete dialog
      const deleteButton = screen.getByLabelText('Delete summary for Test Group')
      await user.click(deleteButton)
      
      // Confirm deletion
      const confirmButton = screen.getByText('Delete')
      await user.click(confirmButton)
      
      expect(mockDelete).toHaveBeenCalledWith(1)
    })

    it('does not call onDelete when cancelled', async () => {
      const mockDelete = jest.fn()
      render(<SummaryCards summaries={[mockSummary]} onDelete={mockDelete} />)
      
      // Open delete dialog
      const deleteButton = screen.getByLabelText('Delete summary for Test Group')
      await user.click(deleteButton)
      
      // Cancel deletion
      const cancelButton = screen.getByText('Cancel')
      await user.click(cancelButton)
      
      expect(mockDelete).not.toHaveBeenCalled()
    })
  })

  describe('Overflow Detection', () => {
    it('handles ResizeObserver cleanup properly', () => {
      const mockDisconnect = jest.fn()
      const mockResizeObserver = {
        observe: jest.fn(),
        unobserve: jest.fn(),
        disconnect: mockDisconnect,
      }
      
      global.ResizeObserver = jest.fn().mockImplementation(() => mockResizeObserver)
      
      const { unmount } = render(<SummaryCards summaries={[mockSummary]} />)
      
      // Unmount component to trigger cleanup
      unmount()
      
      // Should disconnect ResizeObserver
      expect(mockDisconnect).toHaveBeenCalled()
    })

    it('falls back gracefully when ResizeObserver is not supported', () => {
      const consoleSpy = jest.spyOn(console, 'warn').mockImplementation()
      
      // Simulate browser without ResizeObserver
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      global.ResizeObserver = undefined as any
      
      render(<SummaryCards summaries={[mockSummary]} />)
      
      expect(consoleSpy).toHaveBeenCalledWith('ResizeObserver not supported, using fallback')
      
      consoleSpy.mockRestore()
    })
  })

  describe('Dialog Functionality', () => {
    it('opens full content dialog when card is clicked', async () => {
      render(<SummaryCards summaries={[mockSummary]} />)
      
      const card = screen.getByRole('button')
      await user.click(card)
      
      // Should show full dialog with detailed content
      expect(screen.getByText('Jan 01, 2024 – Jan 01, 2024')).toBeInTheDocument()
    })
  })

  describe('Performance and Memory', () => {
    it('cleans up timeouts on unmount', () => {
      jest.spyOn(global, 'clearTimeout')
      
      const { unmount } = render(<SummaryCards summaries={[mockSummary]} />)
      unmount()
      
      expect(clearTimeout).toHaveBeenCalled()
    })

    it('handles rapid re-renders without memory leaks', () => {
      const { rerender } = render(<SummaryCards summaries={[mockSummary]} />)
      
      // Rapid re-renders
      for (let i = 0; i < 10; i++) {
        rerender(<SummaryCards summaries={[{ ...mockSummary, id: i }]} />)
      }
      
      // Should not throw or cause issues
      expect(screen.getByText('Test Group')).toBeInTheDocument()
    })
  })
})