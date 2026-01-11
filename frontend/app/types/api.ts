// API Response Types

export interface User {
  id: string
  email: string
  display_name: string
}

export type AttributeDataType = 'string' | 'number' | 'boolean' | 'text' | 'date'

export interface Attribute {
  id: string
  organization_id: string
  name: string
  key: string
  data_type: AttributeDataType
  created_at: string
  updated_at: string
}

export interface CategoryAttribute {
  id: string
  category_id: string
  attribute_id: string
  required: boolean
  sort_order: number
  created_at: string
  attribute?: Attribute
}

export interface Category {
  id: string
  organization_id: string
  parent_id?: string
  name: string
  description?: string
  icon?: string
  color?: string
  created_at: string
  updated_at: string
  attributes?: CategoryAttribute[]
}

export interface Location {
  id: string
  organization_id: string
  parent_id?: string
  name: string
  description?: string
  created_at: string
  updated_at: string
}

export interface Condition {
  id: string
  organization_id: string
  code: string
  label: string
  description?: string
  sort_order: number
  created_at: string
  updated_at: string
}

export interface Asset {
  id: string
  organization_id: string
  category_id: string
  location_id?: string
  condition_id?: string
  parent_id?: string
  name: string
  description?: string
  quantity: number
  attributes?: Record<string, unknown>
  purchase_at?: string
  purchase_note?: string
  category?: Category
  location?: Location
  condition?: Condition
  created_at: string
  updated_at: string
}

export interface Warranty {
  id: string
  asset_id: string
  provider?: string
  policy_number?: string
  start_date?: string
  end_date?: string
  notes?: string
  created_at: string
  updated_at: string
}

export interface Attachment {
  id: string
  asset_id: string
  filename: string
  content_type: string
  size: number
  storage_key: string
  created_at: string
}

export interface PaginatedResponse<T> {
  items: T[]
  total: number
  limit: number
  offset: number
}

export interface AssetsResponse {
  assets: Asset[]
  total: number
  limit: number
  offset: number
}

export interface AssetFilters {
  q?: string
  category_id?: string
  location_id?: string
  condition_id?: string
  limit?: number
  offset?: number
}
