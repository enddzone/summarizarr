export interface Summary {
  id: number
  group_id: number
  group_name: string
  text: string
  start: string
  end: string
  created_at: string
}

export interface Group {
  id: number
  name: string
  description?: string
}

export interface User {
  id: number
  uuid: string
  name: string
  phone_number?: string
}

export interface Message {
  id: number
  group_id: number
  user_id: number
  content: string
  timestamp: number
  message_type: 'regular' | 'quote' | 'reaction'
  quote_id?: number
  quote_author_uuid?: string
  quote_text?: string
  reaction_emoji?: string
  reaction_target_author?: string
}

export interface FilterOptions {
  groups: number[]
  timeRange: {
    start: Date
    end: Date
  }
  searchQuery: string
}

export interface SignalConfig {
  phoneNumber: string
  isRegistered: boolean
  qrCodeUrl?: string
}

export interface APIResponse<T> {
  data: T
  error?: string
  message?: string
}

export type ViewMode = 'timeline' | 'cards'
export type SortOrder = 'newest' | 'oldest'
