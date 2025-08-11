import React from 'react'
import { render, screen, cleanup } from '@testing-library/react'
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

describe('SummaryCards Performance Tests', () => {
  let mockSummary: Summary
  let mockResizeObserver: {
    observe: jest.Mock
    unobserve: jest.Mock
    disconnect: jest.Mock
  }

  beforeEach(() => {
    mockSummary = {
      id: 1,
      group_id: 123,
      group_name: 'Test Group',
      text: '## Key topics discussed\n- Test topic 1\n- Test topic 2\n\n## Important decisions\n- Test decision',
      created_at: '2024-01-01T10:00:00Z',
      start: '2024-01-01T08:00:00Z',
      end: '2024-01-01T10:00:00Z',
    }

    mockResizeObserver = {
      observe: jest.fn(),
      unobserve: jest.fn(),
      disconnect: jest.fn(),
    }
    
    global.ResizeObserver = jest.fn().mockImplementation(() => mockResizeObserver)
  })

  afterEach(() => {
    cleanup()
  })

  it('renders large number of summaries without performance degradation', () => {
    // Generate 100 summaries to test performance with many cards
    const largeSummaryList: Summary[] = []
    for (let i = 1; i <= 100; i++) {
      largeSummaryList.push({
        ...mockSummary,
        id: i,
        group_id: i % 10, // Vary group IDs
        group_name: `Group ${i % 10}`,
      })
    }

    const startTime = performance.now()
    
    render(<SummaryCards summaries={largeSummaryList} />)
    
    const endTime = performance.now()
    const renderTime = endTime - startTime

    // Should render within reasonable time (less than 200ms for 100 cards in test environment)
    expect(renderTime).toBeLessThan(200)
    
    // Verify all cards are rendered
    const cards = screen.getAllByRole('button')
    expect(cards).toHaveLength(100)
  })

  it('handles rapid re-renders efficiently', () => {
    const { rerender } = render(<SummaryCards summaries={[mockSummary]} />)
    
    const startTime = performance.now()
    
    // Perform 50 rapid re-renders
    for (let i = 0; i < 50; i++) {
      const updatedSummary = { ...mockSummary, id: i + 1 }
      rerender(<SummaryCards summaries={[updatedSummary]} />)
    }
    
    const endTime = performance.now()
    const rerenderTime = endTime - startTime

    // Should complete all re-renders within reasonable time
    expect(rerenderTime).toBeLessThan(100) // Less than 2ms per re-render in test environment

    // Verify final render is correct
    expect(screen.getByText('Test Group')).toBeInTheDocument()
  })

  it('efficiently manages ResizeObserver instances', () => {
    // Test with multiple summaries to ensure ResizeObserver is managed properly
    const multipleSummaries: Summary[] = []
    for (let i = 1; i <= 20; i++) {
      multipleSummaries.push({
        ...mockSummary,
        id: i,
        group_name: `Group ${i}`,
      })
    }

    const { unmount } = render(<SummaryCards summaries={multipleSummaries} />)

    // Should have created ResizeObservers for each card
    expect(global.ResizeObserver).toHaveBeenCalledTimes(20)
    expect(mockResizeObserver.observe).toHaveBeenCalledTimes(20)

    // Cleanup should disconnect all observers
    unmount()
    expect(mockResizeObserver.disconnect).toHaveBeenCalledTimes(20)
  })

  it('handles long text content efficiently', () => {
    const longTextSummary: Summary = {
      ...mockSummary,
      text: `## Key topics discussed
${Array(100).fill('- Very long topic with extensive details that goes on and on with lots of information').join('\n')}

## Important decisions or conclusions
${Array(100).fill('- Another very long decision with comprehensive explanation and detailed rationale').join('\n')}

## Action items or next steps
${Array(100).fill('- Detailed action item with specific requirements and extensive implementation details').join('\n')}

## Notable reactions or responses
${Array(100).fill('- Comprehensive reaction with detailed analysis and extensive feedback from stakeholders').join('\n')}`
    }

    const startTime = performance.now()
    
    render(<SummaryCards summaries={[longTextSummary]} />)
    
    const endTime = performance.now()
    const renderTime = endTime - startTime

    // Should render long content efficiently
    expect(renderTime).toBeLessThan(50)
    
    // Verify content is rendered
    expect(screen.getByText('Test Group')).toBeInTheDocument()
  })

  it('maintains performance with frequent prop changes', () => {
    let currentSummaries = [mockSummary]
    const { rerender } = render(<SummaryCards summaries={currentSummaries} />)

    const startTime = performance.now()

    // Simulate frequent updates (like from real-time data)
    for (let i = 0; i < 100; i++) {
      currentSummaries = currentSummaries.map(summary => ({
        ...summary,
        text: `Updated text ${i}`,
        created_at: new Date().toISOString(),
      }))
      rerender(<SummaryCards summaries={currentSummaries} />)
    }

    const endTime = performance.now()
    const updateTime = endTime - startTime

    // Should handle frequent updates efficiently
    expect(updateTime).toBeLessThan(100)
  })

  it('cleans up resources properly to prevent memory leaks', () => {
    // Track resource usage
    const clearTimeoutSpy = jest.spyOn(global, 'clearTimeout')
    
    // Create and destroy many components
    for (let i = 0; i < 50; i++) {
      const { unmount } = render(<SummaryCards summaries={[mockSummary]} />)
      unmount()
    }

    // Should have called cleanup functions
    expect(clearTimeoutSpy).toHaveBeenCalledTimes(50)
    expect(mockResizeObserver.disconnect).toHaveBeenCalledTimes(50)

    clearTimeoutSpy.mockRestore()
  })

  it('handles edge cases without performance issues', () => {
    const edgeCases: Summary[] = [
      // Empty text
      { ...mockSummary, id: 1, text: '' },
      // Very short text
      { ...mockSummary, id: 2, text: 'Short' },
      // Text with special characters
      { ...mockSummary, id: 3, text: '## Special\n- Content with émojis 🎉 and unicode ™®©' },
      // No group name
      { ...mockSummary, id: 4, group_name: null },
    ]

    const startTime = performance.now()
    
    render(<SummaryCards summaries={edgeCases} />)
    
    const endTime = performance.now()
    const renderTime = endTime - startTime

    // Should handle edge cases efficiently
    expect(renderTime).toBeLessThan(20)
    
    // Verify all cards are rendered
    const cards = screen.getAllByRole('button')
    expect(cards).toHaveLength(4)
  })

  it('maintains consistent performance with different card counts', () => {
    const renderTimes: number[] = []
    
    // Test with different numbers of cards
    const cardCounts = [1, 5, 10, 25, 50]
    
    for (const count of cardCounts) {
      const summaries: Summary[] = []
      for (let i = 1; i <= count; i++) {
        summaries.push({ ...mockSummary, id: i })
      }

      const startTime = performance.now()
      const { unmount } = render(<SummaryCards summaries={summaries} />)
      const endTime = performance.now()
      
      renderTimes.push(endTime - startTime)
      unmount()
    }

    // Performance should scale reasonably (not exponentially)
    // Time per card should remain relatively constant
    const timePerCard1 = renderTimes[0] / cardCounts[0]
    const timePerCard50 = renderTimes[4] / cardCounts[4]
    
    // Time per card shouldn't increase more than 3x as we scale
    expect(timePerCard50).toBeLessThan(timePerCard1 * 3)
  })
})