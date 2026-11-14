'use client';

import React, { useEffect, useState } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { api } from '@/lib/api';
import { Loan } from '@/types/api';
import { StatusBadge } from '@/components/StatusBadge';
import { 
  CreditCard, 
  TrendingUp, 
  Users, 
  ArrowUpRight, 
  PlusCircle, 
  AlertCircle,
  FileText
} from 'lucide-react';

export default function DashboardPage() {
  const router = useRouter();
  const [loans, setLoans] = useState<Loan[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!api.isAuthenticated()) {
      router.replace('/login');
      return;
    }

    const loadData = async () => {
      try {
        setLoading(true);
        const data = await api.listLoans();
        setLoans(data);
      } catch (err: any) {
        if (err?.code === 'UNAUTHORIZED') {
          api.clearAuth();
          router.replace('/login');
          return;
        }
        setError(err?.message || 'Failed to load loans');
      } finally {
        setLoading(false);
      }
    };

    loadData();
  }, [router]);

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[50vh]">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-600"></div>
      </div>
    );
  }

  // Calculate aggregated metrics from server-calculated balances
  let totalLent = 0;
  let totalRepaid = 0;
  let totalOutstanding = 0;
  let activeCount = 0;

  loans.forEach((l) => {
    const principal = parseFloat(l.principal_amount) || 0;
    const repaid = parseFloat(l.summary?.total_confirmed_repayments) || 0;
    const outstanding = parseFloat(l.summary?.outstanding_balance) || 0;

    totalLent += principal;
    totalRepaid += repaid;
    if (l.status === 'ACTIVE' || l.status === 'DISPUTED') {
      totalOutstanding += outstanding;
      activeCount++;
    }
  });

  return (
    <div className="space-y-8">
      {/* Page Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 tracking-tight">Lending Dashboard</h1>
          <p className="text-sm text-slate-500 mt-1">
            Personal lending ledger &amp; external repayment verification
          </p>
        </div>

        <div className="flex items-center gap-3">
          <Link
            href="/peers"
            className="inline-flex items-center px-4 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-200 rounded-lg hover:bg-slate-50 shadow-sm transition"
          >
            <Users className="w-4 h-4 mr-2 text-slate-500" />
            Borrowers
          </Link>
          <Link
            href="/loans/new"
            className="inline-flex items-center px-4 py-2 text-sm font-semibold text-white bg-indigo-600 rounded-lg hover:bg-indigo-700 shadow-sm transition"
          >
            <PlusCircle className="w-4 h-4 mr-2" />
            New Loan
          </Link>
        </div>
      </div>

      {error && (
        <div className="p-4 bg-rose-50 border border-rose-200 rounded-xl flex items-center text-rose-700 text-sm">
          <AlertCircle className="w-5 h-5 mr-3 flex-shrink-0" />
          <span>{error}</span>
        </div>
      )}

      {/* Metrics Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-5">
        <div className="bg-white p-5 rounded-2xl border border-slate-200 shadow-sm">
          <div className="flex items-center justify-between text-slate-500 mb-2">
            <span className="text-xs font-semibold uppercase tracking-wider">Total Outstanding</span>
            <div className="p-2 bg-indigo-50 rounded-lg text-indigo-600">
              <CreditCard className="w-4 h-4" />
            </div>
          </div>
          <div className="text-2xl font-bold text-slate-900">
            ₹{totalOutstanding.toLocaleString('en-IN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}
          </div>
          <p className="text-xs text-slate-400 mt-1">Across active and disputed agreements</p>
        </div>

        <div className="bg-white p-5 rounded-2xl border border-slate-200 shadow-sm">
          <div className="flex items-center justify-between text-slate-500 mb-2">
            <span className="text-xs font-semibold uppercase tracking-wider">Total Repaid</span>
            <div className="p-2 bg-emerald-50 rounded-lg text-emerald-600">
              <TrendingUp className="w-4 h-4" />
            </div>
          </div>
          <div className="text-2xl font-bold text-emerald-600">
            ₹{totalRepaid.toLocaleString('en-IN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}
          </div>
          <p className="text-xs text-slate-400 mt-1">Confirmed received repayments</p>
        </div>

        <div className="bg-white p-5 rounded-2xl border border-slate-200 shadow-sm">
          <div className="flex items-center justify-between text-slate-500 mb-2">
            <span className="text-xs font-semibold uppercase tracking-wider">Total Lent</span>
            <div className="p-2 bg-slate-50 rounded-lg text-slate-600">
              <FileText className="w-4 h-4" />
            </div>
          </div>
          <div className="text-2xl font-bold text-slate-900">
            ₹{totalLent.toLocaleString('en-IN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}
          </div>
          <p className="text-xs text-slate-400 mt-1">Cumulative principal recorded</p>
        </div>

        <div className="bg-white p-5 rounded-2xl border border-slate-200 shadow-sm">
          <div className="flex items-center justify-between text-slate-500 mb-2">
            <span className="text-xs font-semibold uppercase tracking-wider">Active Loans</span>
            <div className="p-2 bg-amber-50 rounded-lg text-amber-600">
              <Users className="w-4 h-4" />
            </div>
          </div>
          <div className="text-2xl font-bold text-slate-900">{activeCount}</div>
          <p className="text-xs text-slate-400 mt-1">{loans.length} total loan records</p>
        </div>
      </div>

      {/* Loans Table */}
      <div className="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden">
        <div className="p-6 border-b border-slate-100 flex items-center justify-between">
          <div>
            <h2 className="text-base font-semibold text-slate-900">Recent Personal Loans</h2>
            <p className="text-xs text-slate-500 mt-0.5">Agreement terms, version history, and verified balances</p>
          </div>
          <Link
            href="/loans"
            className="text-xs font-semibold text-indigo-600 hover:text-indigo-500 inline-flex items-center"
          >
            View all loans
            <ArrowUpRight className="w-3.5 h-3.5 ml-1" />
          </Link>
        </div>

        {loans.length === 0 ? (
          <div className="p-12 text-center">
            <div className="w-12 h-12 bg-indigo-50 rounded-full flex items-center justify-center mx-auto text-indigo-600 mb-3">
              <FileText className="w-6 h-6" />
            </div>
            <h3 className="text-sm font-semibold text-slate-900">No loans recorded yet</h3>
            <p className="text-xs text-slate-500 max-w-sm mx-auto mt-1 mb-5">
              Record money you lend to friends or family to maintain clear agreements and verified balances.
            </p>
            <Link
              href="/loans/new"
              className="inline-flex items-center px-4 py-2 text-xs font-semibold text-white bg-indigo-600 rounded-lg hover:bg-indigo-700 shadow-sm"
            >
              <PlusCircle className="w-4 h-4 mr-1.5" />
              Record First Loan
            </Link>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead className="bg-slate-50 text-[11px] font-semibold text-slate-500 uppercase tracking-wider border-b border-slate-100">
                <tr>
                  <th className="py-3 px-6">Borrower</th>
                  <th className="py-3 px-6">Principal</th>
                  <th className="py-3 px-6">Interest Rate</th>
                  <th className="py-3 px-6">Outstanding</th>
                  <th className="py-3 px-6">Status</th>
                  <th className="py-3 px-6 text-right">Action</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {loans.slice(0, 10).map((l) => (
                  <tr key={l.id} className="hover:bg-slate-50/70 transition">
                    <td className="py-4 px-6 font-medium text-slate-900">
                      <div>{l.peer_name || 'Borrower'}</div>
                      <div className="text-xs text-slate-400 font-normal">{l.start_date}</div>
                    </td>
                    <td className="py-4 px-6 text-slate-700 font-medium">
                      ₹{parseFloat(l.principal_amount).toLocaleString('en-IN', { minimumFractionDigits: 2 })}
                    </td>
                    <td className="py-4 px-6 text-slate-500">
                      {l.interest_type === 'SIMPLE' ? `${l.annual_interest_rate}% p.a.` : 'Interest-free'}
                    </td>
                    <td className="py-4 px-6 font-semibold text-slate-900">
                      ₹{parseFloat(l.summary?.outstanding_balance || l.principal_amount).toLocaleString('en-IN', { minimumFractionDigits: 2 })}
                    </td>
                    <td className="py-4 px-6">
                      <StatusBadge status={l.status} />
                    </td>
                    <td className="py-4 px-6 text-right">
                      <Link
                        href={`/loans/${l.id}`}
                        className="text-xs font-semibold text-indigo-600 hover:text-indigo-800"
                      >
                        Details &rarr;
                      </Link>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
