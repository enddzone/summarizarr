/** @type {import('next').NextConfig} */
const nextConfig = {
  // Production mode - static export for embedding in Go binary
  output: 'export',
  trailingSlash: false,
  images: {
    unoptimized: true,
    remotePatterns: [
      {
        protocol: 'https',
        hostname: '**',
      },
    ],
  },
  // No rewrites needed - use same origin for all API calls
}

export default nextConfig