import { NextRequest, NextResponse } from 'next/server'

const BACKEND_URL = process.env.BACKEND_URL || 'http://localhost:8081'

export async function GET(request: NextRequest) {
  try {
    const { searchParams } = new URL(request.url)
    
    // Forward all query parameters to the backend
    const backendUrl = new URL('/summaries', BACKEND_URL)
    searchParams.forEach((value, key) => {
      backendUrl.searchParams.append(key, value)
    })

    const response = await fetch(backendUrl.toString(), {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
    })

    if (!response.ok) {
      throw new Error(`Backend responded with status: ${response.status}`)
    }

    const data = await response.json()
    
    return NextResponse.json(data)
  } catch (error) {
    console.error('Error fetching summaries:', error)
    return NextResponse.json(
      { error: 'Failed to fetch summaries' },
      { status: 500 }
    )
  }
}

export async function DELETE(request: NextRequest) {
  try {
    const { searchParams } = new URL(request.url)
    const id = searchParams.get('id')
    if (!id) return NextResponse.json({ error: 'Missing id' }, { status: 400 })
    const url = `${BACKEND_URL}/summaries/${id}`
    const resp = await fetch(url, { method: 'DELETE' })
    if (!resp.ok) return NextResponse.json({ error: 'Failed to delete' }, { status: resp.status })
    return NextResponse.json({ ok: true })
  } catch (e) {
    console.error('Error deleting summary:', e)
    return NextResponse.json({ error: 'Failed to delete summary' }, { status: 500 })
  }
}
