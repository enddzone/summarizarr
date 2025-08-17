/**
 * Application configuration constants
 * These are not exposed to the client bundle
 */
export const APP_CONFIG = {
    /**
     * Documentation URLs for user guidance
     */
    DOCS: {
        SIGNAL_SETUP: 'https://github.com/enddzone/summarizarr#signal-setup',
        MAIN_README: 'https://github.com/enddzone/summarizarr#readme',
    },

    /**
     * External links with fallback handling
     */
    EXTERNAL_LINKS: {
        GITHUB_REPO: 'https://github.com/enddzone/summarizarr',
    },
} as const;

/**
 * Safely opens an external URL with proper security attributes
 * @param url - The URL to open
 * @param fallbackUrl - Optional fallback URL if primary fails
 */
export function openExternalUrl(url: string, fallbackUrl?: string): void {
    try {
        window.open(url, '_blank', 'noopener,noreferrer');
    } catch (error) {
        console.warn('Failed to open primary URL:', url, error);
        if (fallbackUrl) {
            try {
                window.open(fallbackUrl, '_blank', 'noopener,noreferrer');
            } catch (fallbackError) {
                console.error('Failed to open fallback URL:', fallbackUrl, fallbackError);
            }
        }
    }
}
