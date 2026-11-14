export interface User {
  id: string;
  email: string;
  full_name: string;
  created_at: string;
  updated_at: string;
}

export interface AuthTokens {
  access_token: string;
  refresh_token: string;
  token_type?: string;
  expires_in?: number;
}

export interface AuthResponse {
  user: User;
  access_token: string;
  refresh_token: string;
  expires_in: number;
  tokens?: AuthTokens;
}

export interface Peer {
  id: string;
  user_id: string;
  name: string;
  email?: string | null;
  phone?: string | null;
  notes?: string | null;
  created_at: string;
  updated_at: string;
}

export type LoanStatus = 'DRAFT' | 'ACTIVE' | 'COMPLETED' | 'CANCELLED' | 'DISPUTED';
export type InterestType = 'NONE' | 'SIMPLE';

export interface FinancialSummary {
  principal_amount: string;
  interest_accrued: string;
  total_amount_due: string;
  total_confirmed_repayments: string;
  outstanding_balance: string;
  elapsed_days: number;
  calculation_date: string;
}

export interface Loan {
  id: string;
  user_id: string;
  peer_id: string;
  peer_name?: string;
  peer_email?: string | null;
  principal_amount: string;
  currency: string;
  interest_type: InterestType;
  annual_interest_rate: string;
  day_count_convention: string;
  start_date: string;
  due_date?: string | null;
  status: LoanStatus;
  version: number;
  notes?: string | null;
  created_at: string;
  updated_at: string;
  summary: FinancialSummary;
}

export type RepaymentStatus = 'PENDING_VERIFICATION' | 'CONFIRMED' | 'REJECTED' | 'REVERSED';
export type RepaymentSource = 'LENDER_RECORDED' | 'BORROWER_REPORTED';

export interface Repayment {
  id: string;
  user_id: string;
  loan_id: string;
  amount: string;
  currency: string;
  payment_date: string;
  payment_method: string;
  external_reference?: string | null;
  status: RepaymentStatus;
  source: RepaymentSource;
  confirmed_at?: string | null;
  confirmed_by?: string | null;
  rejection_reason?: string | null;
  reversed_at?: string | null;
  reversal_reason?: string | null;
  notes?: string | null;
  created_at: string;
  updated_at: string;
}

export interface LedgerEvent {
  id: string;
  user_id: string;
  loan_id?: string | null;
  event_type: string;
  actor_type: string;
  actor_id?: string | null;
  payload: Record<string, any>;
  created_at: string;
}

export interface BorrowerLinkResponse {
  token: string;
  url: string;
  expires_at: string;
}

export interface BorrowerAccessResponse {
  session_token: string;
  loan_id: string;
  expires_at: string;
}

export interface BorrowerLoanDetail {
  id: string;
  currency: string;
  principal_amount: string;
  interest_type: string;
  annual_interest_rate: string;
  day_count_convention: string;
  start_date: string;
  due_date?: string | null;
  status: string;
  version: number;
  notes?: string | null;
}

export interface BorrowerLoanViewResponse {
  loan: BorrowerLoanDetail;
  summary: FinancialSummary;
  lender: {
    name: string;
    email: string;
  };
  latest_acknowledgement?: {
    version: number;
    status: string;
    dispute_reason?: string | null;
    responded_at: string;
  } | null;
  repayments: Repayment[];
}

export interface ApiResponse<T> {
  success: boolean;
  data: T;
  meta?: {
    page: number;
    page_size: number;
    total_items: number;
    total_pages: number;
  };
}

export interface ApiError {
  code: string;
  message: string;
  details?: Record<string, string>;
}
