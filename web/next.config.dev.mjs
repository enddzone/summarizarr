/** @type {import('next').NextConfig} */
const nextConfig = {
  // Development mode - no static export, enable API proxying
  trailingSlash: true,
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
    return [
      {
        source: '/api/:path*',
        destination: 'http://localhost:8081/api/:path*',
      },
    ]
  },
}

export default nextConfig