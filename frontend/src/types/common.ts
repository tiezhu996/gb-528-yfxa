export interface ApiEnvelope<T> {
  data: T
  request_id: string
}

export interface PageMeta {
  page: number
  page_size: number
  total: number
}

export interface PageEnvelope<T> extends ApiEnvelope<T[]> {
  meta: PageMeta
}

export interface ApiErrorBody {
  error: {
    code: string
    message: string
    details?: unknown
  }
  request_id: string
}
