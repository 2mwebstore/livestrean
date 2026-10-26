// Response envelope shared by every Go API endpoint.

export interface ApiResponse<T> {
  success: true
  message?: string
  data: T
}

export interface PaginationMeta {
  page: number
  limit: number
  total: number
  total_pages: number
}

export interface PaginatedResponse<T> {
  success: true
  data: T[]
  meta: PaginationMeta
}

export interface ApiErrorBody {
  success: false
  message: string
  errors?: Record<string, string>
}

export interface PageQuery {
  page?: number
  limit?: number
  q?: string
}
