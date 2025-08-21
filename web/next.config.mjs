/** @type {import('next').NextConfig} */
const nextConfig = {
  // Static export for embedding in Go backend
  output: 'export',
  trailingSlash: false,
  images: {
    unoptimized: true,
  },
}

export default nextConfig