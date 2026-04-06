export interface User {
  id: string;
  username: string;
  display_name: string;
  email?: string;
  is_locked: boolean;
  created_at: string;
  updated_at: string;
}

export interface AuthResponse {
  user: User;
  roles: string[];
}

export interface Role {
  id: string;
  name: string;
  description: string;
}

export interface UserRole {
  user_id: string;
  role_id: string;
  role_name: string;
  granted_by: string;
  granted_at: string;
}

export interface Collectible {
  id: string;
  seller_id: string;
  title: string;
  description: string;
  contract_address?: string;
  chain_id?: number;
  token_id?: string;
  metadata_uri?: string;
  image_url?: string;
  price_cents: number;
  currency: string;
  status: string;
  hidden_by?: string;
  hidden_reason?: string;
  view_count: number;
  created_at: string;
  updated_at: string;
}

export interface CollectibleTxHistory {
  id: string;
  collectible_id: string;
  tx_hash: string;
  from_address: string;
  to_address: string;
  block_number: number;
  timestamp: string;
}

export type OrderStatus = 'pending' | 'confirmed' | 'processing' | 'completed' | 'cancelled';

export interface Order {
  id: string;
  idempotency_key: string;
  buyer_id: string;
  collectible_id: string;
  seller_id: string;
  status: OrderStatus;
  price_snapshot_cents: number;
  cancellation_reason?: string;
  cancelled_by?: string;
  fulfillment_tracking?: {
    carrier?: string;
    tracking_number?: string;
    shipped_at?: string;
    delivered_at?: string;
  };
  created_at: string;
  updated_at: string;
}

export interface Message {
  id: string;
  order_id: string;
  sender_id: string;
  body: string;
  attachment_id?: string;
  attachment_size?: number;
  attachment_mime?: string;
  created_at: string;
}

export interface Notification {
  id: string;
  user_id: string;
  template_id: string;
  template_slug?: string;
  rendered_title: string;
  rendered_body: string;
  is_read: boolean;
  status: string;
  retry_count: number;
  max_retries: number;
  delivered_at?: string;
  created_at: string;
}

export interface NotificationTemplate {
  id: string;
  slug: string;
  title_template: string;
  body_template: string;
}

export interface NotificationPreferences {
  user_id: string;
  preferences: Record<string, boolean>;
  subscription_mode: 'status_only' | 'all_events';
}

export interface ABTest {
  id: string;
  name: string;
  description: string;
  status: string;
  traffic_pct: number;
  start_date: string;
  end_date: string;
  control_variant: string;
  test_variant: string;
  rollback_threshold_pct: number;
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface ABTestResult {
  id: string;
  ab_test_id: string;
  variant: string;
  views: number;
  orders: number;
  conversion_rate: number;
  computed_at: string;
}

export interface ABTestAssignment {
  test_name: string;
  variant: string;
}

export interface AnomalyEvent {
  id: string;
  user_id: string;
  anomaly_type: string;
  details: Record<string, unknown>;
  acknowledged: boolean;
  created_at: string;
}

export interface IPRule {
  id: string;
  cidr: string;
  action: string;
  created_by: string;
  created_at: string;
}

export interface PaginatedResponse<T> {
  data: T[];
  page: number;
  page_size: number;
  total_count: number;
  total_pages: number;
}

export interface FunnelResponse {
  views: number;
  orders: number;
  rate: number;
  days: number;
}

export interface RetentionCohort {
  cohort_date: string;
  cohort_size: number;
  retained_count: number;
  retention_rate: number;
}

export interface ContentPerformance {
  collectible_id: string;
  title: string;
  views: number;
  orders: number;
  conversion_rate: number;
}

export interface ErrorResponse {
  error: {
    code: string;
    message: string;
    request_id?: string;
  };
}
