import {
  AuthResponse,
  ApiResponse,
  ApiError,
  User,
  Peer,
  Loan,
  Repayment,
  LedgerEvent,
  BorrowerLinkResponse,
  BorrowerAccessResponse,
  BorrowerLoanViewResponse,
} from '@/types/api';

const API_BASE = process.env.NEXT_PUBLIC_API_URL || '';

class ApiClient {
  private getAccessToken(): string | null {
    if (typeof window === 'undefined') return null;
    return localStorage.getItem('tavrocred_access_token');
  }

  public setTokens(accessToken: string, refreshToken: string, user: User) {
    if (typeof window === 'undefined') return;
    localStorage.setItem('tavrocred_access_token', accessToken);
    localStorage.setItem('tavrocred_refresh_token', refreshToken);
    localStorage.setItem('tavrocred_user', JSON.stringify(user));
  }

  public clearAuth() {
    if (typeof window === 'undefined') return;
    localStorage.removeItem('tavrocred_access_token');
    localStorage.removeItem('tavrocred_refresh_token');
    localStorage.removeItem('tavrocred_user');
  }

  public getCurrentUser(): User | null {
    if (typeof window === 'undefined') return null;
    const raw = localStorage.getItem('tavrocred_user');
    if (!raw) return null;
    try {
      return JSON.parse(raw);
    } catch {
      return null;
    }
  }

  public isAuthenticated(): boolean {
    return !!this.getAccessToken();
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit = {},
    customBearerToken?: string
  ): Promise<T> {
    const url = endpoint.startsWith('http') ? endpoint : `${API_BASE}/api/v1${endpoint}`;
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      Accept: 'application/json',
      ...(options.headers as Record<string, string>),
    };

    const token = customBearerToken || this.getAccessToken();
    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }

    const response = await fetch(url, {
      ...options,
      headers,
    });

    const data = await response.json().catch(() => null);

    if (!response.ok) {
      const err: ApiError = data?.error || {
        code: 'NETWORK_ERROR',
        message: response.statusText || 'Request failed',
      };
      throw err;
    }

    return data;
  }

  // --- Auth APIs ---
  async login(payload: { email: string; password: string }): Promise<AuthResponse> {
    const res = await this.request<ApiResponse<AuthResponse>>('/auth/login', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
    const accessToken = res?.data?.access_token || res?.data?.tokens?.access_token || '';
    const refreshToken = res?.data?.refresh_token || res?.data?.tokens?.refresh_token || '';
    if (res?.data?.user) {
      this.setTokens(accessToken, refreshToken, res.data.user);
    }
    return res.data;
  }

  async register(payload: { email: string; password: string; full_name: string }): Promise<AuthResponse> {
    const res = await this.request<ApiResponse<AuthResponse>>('/auth/register', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
    const accessToken = res?.data?.access_token || res?.data?.tokens?.access_token || '';
    const refreshToken = res?.data?.refresh_token || res?.data?.tokens?.refresh_token || '';
    if (res?.data?.user) {
      this.setTokens(accessToken, refreshToken, res.data.user);
    }
    return res.data;
  }

  async refreshToken(): Promise<AuthResponse | null> {
    const refreshToken = typeof window !== 'undefined' ? localStorage.getItem('tavrocred_refresh_token') : null;
    if (!refreshToken) return null;
    try {
      const res = await this.request<ApiResponse<AuthResponse>>('/auth/refresh', {
        method: 'POST',
        body: JSON.stringify({ refresh_token: refreshToken }),
      });
      const accessToken = res?.data?.access_token || res?.data?.tokens?.access_token || '';
      const newRefreshToken = res?.data?.refresh_token || res?.data?.tokens?.refresh_token || '';
      if (res?.data?.user) {
        this.setTokens(accessToken, newRefreshToken, res.data.user);
      }
      return res.data;
    } catch {
      this.clearAuth();
      return null;
    }
  }

  async getMe(): Promise<User> {
    const res = await this.request<ApiResponse<{ user: User }>>('/auth/me');
    return res.data.user;
  }

  async logout(): Promise<void> {
    const refreshToken = typeof window !== 'undefined' ? localStorage.getItem('tavrocred_refresh_token') : null;
    try {
      if (refreshToken) {
        await this.request('/auth/logout', {
          method: 'POST',
          body: JSON.stringify({ refresh_token: refreshToken }),
        });
      }
    } finally {
      this.clearAuth();
    }
  }

  // --- Peer APIs ---
  async listPeers(search = ''): Promise<Peer[]> {
    const query = search ? `?search=${encodeURIComponent(search)}` : '';
    const res = await this.request<ApiResponse<Peer[]>>(`/peers${query}`);
    return res.data;
  }

  async createPeer(payload: { name: string; phone?: string; email?: string; notes?: string }): Promise<Peer> {
    const res = await this.request<ApiResponse<Peer>>('/peers', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
    return res.data;
  }

  async updatePeer(id: string, payload: { name: string; phone?: string; email?: string; notes?: string }): Promise<Peer> {
    const res = await this.request<ApiResponse<Peer>>(`/peers/${id}`, {
      method: 'PUT',
      body: JSON.stringify(payload),
    });
    return res.data;
  }

  // --- Loan APIs ---
  async listLoans(params?: { peer_id?: string; status?: string }): Promise<Loan[]> {
    const q = new URLSearchParams();
    if (params?.peer_id) q.set('peer_id', params.peer_id);
    if (params?.status && params.status !== 'ALL') q.set('status', params.status);
    const queryStr = q.toString() ? `?${q.toString()}` : '';
    const res = await this.request<ApiResponse<Loan[]>>(`/loans${queryStr}`);
    return res.data;
  }

  async getLoan(id: string): Promise<Loan> {
    const res = await this.request<ApiResponse<Loan>>(`/loans/${id}`);
    return res.data;
  }

  async createLoan(payload: {
    peer_id: string;
    principal_amount: string;
    currency?: string;
    interest_type?: string;
    annual_interest_rate?: string;
    start_date: string;
    due_date?: string;
    notes?: string;
  }): Promise<Loan> {
    const res = await this.request<ApiResponse<Loan>>('/loans', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
    return res.data;
  }

  async activateLoan(id: string): Promise<Loan> {
    const res = await this.request<ApiResponse<Loan>>(`/loans/${id}/activate`, {
      method: 'POST',
    });
    return res.data;
  }

  async amendLoan(id: string, payload: {
    principal_amount: string;
    annual_interest_rate?: string;
    interest_type?: string;
    due_date?: string;
    reason: string;
    notes?: string;
  }): Promise<Loan> {
    const res = await this.request<ApiResponse<Loan>>(`/loans/${id}/amend`, {
      method: 'POST',
      body: JSON.stringify(payload),
    });
    return res.data;
  }

  async cancelLoan(id: string, reason: string): Promise<Loan> {
    const res = await this.request<ApiResponse<Loan>>(`/loans/${id}/cancel`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    });
    return res.data;
  }

  async disputeLoan(id: string, reason: string): Promise<Loan> {
    const res = await this.request<ApiResponse<Loan>>(`/loans/${id}/dispute`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    });
    return res.data;
  }

  async listLoanEvents(id: string): Promise<LedgerEvent[]> {
    const res = await this.request<ApiResponse<LedgerEvent[]>>(`/loans/${id}/events`);
    return res.data;
  }

  // --- Repayment APIs ---
  async listRepayments(loanID: string): Promise<Repayment[]> {
    const res = await this.request<ApiResponse<Repayment[]>>(`/loans/${loanID}/repayments`);
    return res.data;
  }

  async recordRepayment(loanID: string, payload: {
    amount: string;
    currency?: string;
    payment_date: string;
    payment_method: string;
    external_reference?: string;
    notes?: string;
    confirmed_directly: boolean;
  }): Promise<Repayment> {
    const res = await this.request<ApiResponse<Repayment>>(`/loans/${loanID}/repayments`, {
      method: 'POST',
      body: JSON.stringify(payload),
    });
    return res.data;
  }

  async confirmRepayment(repaymentID: string): Promise<Repayment> {
    const res = await this.request<ApiResponse<Repayment>>(`/repayments/${repaymentID}/confirm`, {
      method: 'POST',
    });
    return res.data;
  }

  async rejectRepayment(repaymentID: string, reason: string): Promise<Repayment> {
    const res = await this.request<ApiResponse<Repayment>>(`/repayments/${repaymentID}/reject`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    });
    return res.data;
  }

  async reverseRepayment(repaymentID: string, reason: string): Promise<Repayment> {
    const res = await this.request<ApiResponse<Repayment>>(`/repayments/${repaymentID}/reverse`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    });
    return res.data;
  }

  // --- Borrower Portal Links (Lender endpoints) ---
  async generateBorrowerLink(loanID: string, expiresInHours = 72): Promise<BorrowerLinkResponse> {
    const res = await this.request<ApiResponse<BorrowerLinkResponse>>(`/loans/${loanID}/borrower-link`, {
      method: 'POST',
      body: JSON.stringify({ expires_in_hours: expiresInHours }),
    });
    return res.data;
  }

  async revokeBorrowerLink(loanID: string): Promise<void> {
    await this.request(`/loans/${loanID}/borrower-link/revoke`, {
      method: 'POST',
    });
  }

  // --- Public Borrower Portal (Borrower endpoints) ---
  async exchangeBorrowerToken(token: string): Promise<BorrowerAccessResponse> {
    const res = await this.request<ApiResponse<BorrowerAccessResponse>>('/public/borrower/access', {
      method: 'POST',
      body: JSON.stringify({ token }),
    });
    return res.data;
  }

  async getBorrowerLoan(sessionToken: string): Promise<BorrowerLoanViewResponse> {
    const res = await this.request<ApiResponse<BorrowerLoanViewResponse>>(
      '/public/borrower/loan',
      { method: 'GET' },
      sessionToken
    );
    return res.data;
  }

  async acknowledgeBorrowerLoan(sessionToken: string): Promise<any> {
    const res = await this.request<ApiResponse<any>>(
      '/public/borrower/acknowledge',
      { method: 'POST' },
      sessionToken
    );
    return res.data;
  }

  async disputeBorrowerLoan(sessionToken: string, reason: string): Promise<any> {
    const res = await this.request<ApiResponse<any>>(
      '/public/borrower/dispute',
      {
        method: 'POST',
        body: JSON.stringify({ reason }),
      },
      sessionToken
    );
    return res.data;
  }

  async reportBorrowerRepayment(
    sessionToken: string,
    payload: {
      amount: string;
      payment_date: string;
      payment_method: string;
      external_reference?: string;
      notes?: string;
    }
  ): Promise<Repayment> {
    const res = await this.request<ApiResponse<Repayment>>(
      '/public/borrower/repayment-reported',
      {
        method: 'POST',
        body: JSON.stringify(payload),
      },
      sessionToken
    );
    return res.data;
  }
}

export const api = new ApiClient();
