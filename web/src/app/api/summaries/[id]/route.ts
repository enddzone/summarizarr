import { NextResponse } from 'next/server'

const BACKEND_URL = process.env.BACKEND_URL || 'http://localhost:8081'

export async function DELETE(request: Request) {
  try {
  const { pathname } = new URL(request.url)
  const parts = pathname.split('/').filter(Boolean)
  const id = parts[parts.length - 1]
    const url = `${BACKEND_URL}/summaries/${id}`
    const resp = await fetch(url, { method: 'DELETE' })
    if (!resp.ok) {
      return NextResponse.json({ error: 'Failed to delete summary' }, { status: resp.status })
    }
    return NextResponse.json({ ok: true })
  } catch (e) {
    console.error('Error deleting summary', e)
    return NextResponse.json({ error: 'Failed to delete summary' }, { status: 500 })
  }
}
