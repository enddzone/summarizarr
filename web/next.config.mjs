/** @type {import('next').NextConfig} */
const nextConfig = {
  // Static export for embedding in Go backend
  output: process.env.NODE_ENV === 'production' ? 'export' : undefined,
  trailingSlash: false,
  images: {
    unoptimized: true,
  },
  // Development proxy configuration
  async rewrites() {
    if (process.env.NODE_ENV !== 'production') {
      return [
        {
          source: '/api/:path*',
          destination: 'http://localhost:8081/api/:path*',
        },
      ];
    }
    return [];
  },
}

export default nextConfig