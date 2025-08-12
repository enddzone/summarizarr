import type { Metadata } from 'next'
import { GeistSans } from 'geist/font/sans'
import { GeistMono } from 'geist/font/mono'
import './globals.css'
import { Providers } from '@/components/providers'

export const metadata: Metadata = {
  title: {
    default: 'Summarizarr',
    template: '%s | Summarizarr'
  },
  description: 'AI-powered Signal message summarization with real-time insights',
  keywords: ['Signal', 'AI', 'Summarization', 'Messages', 'Chat'],
  authors: [{ name: 'Summarizarr Team' }],
  creator: 'Summarizarr',
  icons: {
    icon: [
      { url: '/favicon-16x16.png', sizes: '16x16', type: 'image/png' },
      { url: '/favicon-32x32.png', sizes: '32x32', type: 'image/png' },
    ],
    apple: [
      { url: '/apple-touch-icon.png', sizes: '180x180', type: 'image/png' },
    ],
    other: [
      { url: '/android-chrome-192x192.png', sizes: '192x192', type: 'image/png' },
    ],
  },
  openGraph: {
    type: 'website',
    locale: 'en_US',
    url: '/',
    title: 'Summarizarr',
    description: 'AI-powered Signal message summarization with real-time insights',
    siteName: 'Summarizarr',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'Summarizarr',
    description: 'AI-powered Signal message summarization with real-time insights',
  },
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="en" className={`${GeistSans.variable} ${GeistMono.variable}`} suppressHydrationWarning>
      <body className="font-sans antialiased">
        <Providers>
          {children}
        </Providers>
      </body>
    </html>
  )
}
