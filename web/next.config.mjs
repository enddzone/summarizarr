/** @type {import('next').NextConfig} */
const nextConfig = {
  // Development mode configuration
  // For production builds, the Dockerfile copies next.config.prod.mjs over this file
  trailingSlash: false,
  turbopack: {},
  images: {
    unoptimized: true,
    remotePatterns: [
      {
        protocol: 'https',
        hostname: '**',
      },
    ],
  },
  async rewrites() {
    // Development mode - rewrite API calls to backend
    return [
      {
        source: '/api/:path*',
        destination: 'http://localhost:8081/api/:path*',
      },
    ]
  },
}

export default nextConfig