import { NextRequest, NextResponse } from 'next/server'

const BACKEND_URL = process.env.BACKEND_URL || 'http://localhost:8081'

export async function GET(request: NextRequest) {
  try {
    const { searchParams } = new URL(request.url)
    
    // Forward all query parameters to the backend
    const backendUrl = new URL('/export', BACKEND_URL)
    searchParams.forEach((value, key) => {
      backendUrl.searchParams.append(key, value)
    })

    const response = await fetch(backendUrl.toString(), {
      method: 'GET',
    })

    if (!response.ok) {
      throw new Error(`Backend responded with status: ${response.status}`)
    }

    // Get the content type from the backend response
    const contentType = response.headers.get('content-type') || 'application/octet-stream'
    
    // Stream the response body
    const body = await response.arrayBuffer()
    
    return new NextResponse(body, {
      headers: {
        'Content-Type': contentType,
        'Content-Disposition': response.headers.get('content-disposition') || 'attachment',
      },
    })
  } catch (error) {
    console.error('Error exporting data:', error)
    return NextResponse.json(
      { error: 'Failed to export data' },
      { status: 500 }
    )
  }
}
