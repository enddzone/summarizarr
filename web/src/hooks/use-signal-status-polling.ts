import { useState, useEffect, useCallback, useRef } from 'react'

interface SignalStatusResponse {
    phoneNumber: string
    isRegistered: boolean
    connected?: boolean
    status?: string
}

interface UseSignalStatusPollingReturn {
    isPolling: boolean
    error: string | null
    statusData: SignalStatusResponse | null
}

/**
 * Custom hook to poll Signal registration status
 * @param active - Whether polling should be active
 * @param onSuccess - Callback when registration is detected
 * @param interval - Polling interval in milliseconds (default: 5000)
 * @param timeout - Total timeout in milliseconds (default: 60000)
 */
export function useSignalStatusPolling(
    active: boolean,
    onSuccess?: (data: SignalStatusResponse) => void,
    interval: number = 5000,
    timeout: number = 60000
): UseSignalStatusPollingReturn {
    const [isPolling, setIsPolling] = useState(false)
    const [error, setError] = useState<string | null>(null)
    const [statusData, setStatusData] = useState<SignalStatusResponse | null>(null)

    // Use ref to track if component is mounted to prevent race conditions
    const isMountedRef = useRef(true)

    const pollStatus = useCallback(async (): Promise<SignalStatusResponse | null> => {
        try {
            const response = await fetch('/api/signal/status', {
                credentials: 'include'
            })
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`)
            }

            const data = await response.json()

            // Check if component is still mounted before updating state
            if (isMountedRef.current) {
                setStatusData(data)
                setError(null)
            }
            return data
        } catch (err) {
            const errorMsg = err instanceof Error ? err.message : 'Failed to check signal status'

            // Check if component is still mounted before updating state
            if (isMountedRef.current) {
                setError(errorMsg)
            }
            return null
        }
    }, [])

    useEffect(() => {
        // Reset mounted flag when effect runs
        isMountedRef.current = true

        if (!active) {
            if (isMountedRef.current) {
                setIsPolling(false)
                setError(null)
            }
            return
        }

        if (isMountedRef.current) {
            setIsPolling(true)
            setError(null)
        }

        const startTime = Date.now()
        let timeoutId: ReturnType<typeof setTimeout> | null = null
        let intervalId: ReturnType<typeof setInterval> | null = null
        let isCleanedUp = false

        const performPoll = async (): Promise<boolean> => {
            // Early exit if cleaned up or unmounted
            if (isCleanedUp || !isMountedRef.current) {
                return true
            }

            const data = await pollStatus()

            // Check again after async operation
            if (isCleanedUp || !isMountedRef.current) {
                return true
            }

            // Check if registration is successful
            if (data?.isRegistered) {
                if (isMountedRef.current) {
                    setIsPolling(false)
                }
                if (onSuccess && !isCleanedUp) {
                    onSuccess(data)
                }
                return true // Stop polling
            }

            // Check if timeout exceeded
            if (Date.now() - startTime >= timeout) {
                if (isMountedRef.current) {
                    setIsPolling(false)
                    setError('Polling timeout - registration not detected')
                }
                return true // Stop polling
            }

            return false // Continue polling
        }

        // Initial poll
        performPoll().then((shouldStop) => {
            if (shouldStop || isCleanedUp || !isMountedRef.current) return

            // Set up interval polling
            intervalId = setInterval(async () => {
                const shouldStop = await performPoll()
                if (shouldStop && intervalId && !isCleanedUp) {
                    clearInterval(intervalId)
                    intervalId = null
                }
            }, interval)

            // Set up timeout
            timeoutId = setTimeout(() => {
                if (isCleanedUp || !isMountedRef.current) return

                if (intervalId) {
                    clearInterval(intervalId)
                    intervalId = null
                }
                if (isMountedRef.current) {
                    setIsPolling(false)
                    setError('Polling timeout - registration not detected')
                }
            }, timeout)
        }).catch((err) => {
            if (isMountedRef.current && !isCleanedUp) {
                setError(err instanceof Error ? err.message : 'Failed to start polling')
                setIsPolling(false)
            }
        })

        // Cleanup function
        return () => {
            isCleanedUp = true
            isMountedRef.current = false

            if (intervalId) {
                clearInterval(intervalId)
                intervalId = null
            }
            if (timeoutId) {
                clearTimeout(timeoutId)
                timeoutId = null
            }
        }
    }, [active, pollStatus, onSuccess, interval, timeout])

    return {
        isPolling,
        error,
        statusData,
    }
}
