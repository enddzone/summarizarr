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
