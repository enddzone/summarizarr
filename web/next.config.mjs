/** @type {import('next').NextConfig} */
const nextConfig = {
  // Static export configuration for production builds
  output: 'export',
  trailingSlash: true,
  images: {
    unoptimized: true,
    remotePatterns: [
      {
        protocol: 'https',
        hostname: '**',
      },
    ],
  },
  // Remove rewrites for static export as they're not supported
}

export default nextConfig