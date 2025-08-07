import { NextRequest, NextResponse } from 'next/server'

const BACKEND_URL = process.env.BACKEND_URL || 'http://localhost:8081'

export async function GET(request: NextRequest) {
  try {
    const response = await fetch(`${BACKEND_URL}/signal/config`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
    })

    if (!response.ok) {
      // Return default config if backend doesn't have signal config yet
      return NextResponse.json({
        phoneNumber: '',
        isRegistered: false,
      })
    }

    const data = await response.json()
    
    return NextResponse.json(data)
  } catch (error) {
    console.error('Error fetching Signal config:', error)
    // Return default config on error
    return NextResponse.json({
      phoneNumber: '',
      isRegistered: false,
    })
  }
}
