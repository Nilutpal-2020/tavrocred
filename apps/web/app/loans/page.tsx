'use client';

import React, { useEffect, useState, Suspense } from 'react';
import Link from 'next/link';
import { useRouter, useSearchParams } from 'next/navigation';
import { api } from '@/lib/api';
import { Loan, LoanStatus } from '@/types/api';
import { StatusBadge } from '@/components/StatusBadge';
import { PlusCircle, AlertCircle, FileText } from 'lucide-react';

function LoansContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const initialPeerID = searchParams.get('peer_id') || undefined;

  const [loans, setLoans] = useState<Loan[]>([]);
  const [loading, setLoading] = useState(true);
  const [statusFilter, setStatusFilter] = useState<string>('ALL');
  const [error, setError] = useState<string | null>(null);

  const fetchLoans = async (status = statusFilter) => {
    try {
      setLoading(true);
      const data = await api.listLoans({
        peer_id: initialPeerID,
        status: status === 'ALL' ? undefined : status,
      });
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

  useEffect(() => {
    if (!api.isAuthenticated()) {
      router.replace('/login');
      return;
    }
    fetchLoans(statusFilter);
  }, [router, statusFilter, initialPeerID]);

  const tabs: { label: string; value: string }[] = [
    { label: 'All', value: 'ALL' },
    { label: 'Active', value: 'ACTIVE' },
    { label: 'Drafts', value: 'DRAFT' },
    { label: 'Disputed', value: 'DISPUTED' },
    { label: 'Completed', value: 'COMPLETED' },
    { label: 'Cancelled', value: 'CANCELLED' },
  ];

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 tracking-tight">Personal Loans</h1>
          <p className="text-sm text-slate-500 mt-1">
            Track agreed terms, financial breakdown, version amendments, and repayments
          </p>
        </div>

        <Link
          href="/loans/new"
          className="inline-flex items-center px-4 py-2 text-sm font-semibold text-white bg-indigo-600 rounded-lg hover:bg-indigo-700 shadow-sm transition"
        >
          <PlusCircle className="w-4 h-4 mr-2" />
          New Loan
        </Link>
      </div>

      {error && (
        <div className="p-4 bg-rose-50 border border-rose-200 rounded-xl flex items-center text-rose-700 text-sm">
          <AlertCircle className="w-5 h-5 mr-3 flex-shrink-0" />
          <span>{error}</span>
        </div>
      )}

      {/* Tabs */}
      <div className="flex overflow-x-auto border-b border-slate-200 gap-2 pb-px text-sm">
        {tabs.map((tab) => (
          <button
            key={tab.value}
            onClick={() => setStatusFilter(tab.value)}
            className={`px-4 py-2 font-medium border-b-2 whitespace-nowrap transition ${
              statusFilter === tab.value
                ? 'border-indigo-600 text-indigo-600'
                : 'border-transparent text-slate-500 hover:text-slate-700 hover:border-slate-300'
            }`}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {/* Loans Table */}
      <div className="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden">
        {loading ? (
          <div className="flex items-center justify-center min-h-[40vh]">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-600"></div>
          </div>
        ) : loans.length === 0 ? (
          <div className="p-12 text-center">
            <div className="w-12 h-12 bg-indigo-50 rounded-full flex items-center justify-center mx-auto text-indigo-600 mb-3">
              <FileText className="w-6 h-6" />
            </div>
            <h3 className="text-sm font-semibold text-slate-900">No loans found</h3>
            <p className="text-xs text-slate-500 max-w-sm mx-auto mt-1 mb-5">
              {statusFilter !== 'ALL'
                ? `No loans currently in ${statusFilter.toLowerCase()} status.`
                : 'Record your first loan agreement to begin tracking.'}
            </p>
            <Link
              href="/loans/new"
              className="inline-flex items-center px-4 py-2 text-xs font-semibold text-white bg-indigo-600 rounded-lg hover:bg-indigo-700 shadow-sm"
            >
              <PlusCircle className="w-4 h-4 mr-1.5" />
              Create Loan
            </Link>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead className="bg-slate-50 text-[11px] font-semibold text-slate-500 uppercase tracking-wider border-b border-slate-100">
                <tr>
                  <th className="py-3 px-6">Borrower</th>
                  <th className="py-3 px-6">Principal</th>
                  <th className="py-3 px-6">Interest</th>
                  <th className="py-3 px-6">Total Due</th>
                  <th className="py-3 px-6">Outstanding</th>
                  <th className="py-3 px-6">Status</th>
                  <th className="py-3 px-6 text-right">Action</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {loans.map((l) => (
                  <tr key={l.id} className="hover:bg-slate-50/70 transition">
                    <td className="py-4 px-6 font-medium text-slate-900">
                      <div>{l.peer_name || 'Borrower'}</div>
                      <div className="text-xs text-slate-400 font-normal">
                        Starts {l.start_date} {l.due_date ? `· Due ${l.due_date}` : ''}
                      </div>
                    </td>
                    <td className="py-4 px-6 font-medium text-slate-700">
                      ₹{parseFloat(l.principal_amount).toLocaleString('en-IN', { minimumFractionDigits: 2 })}
                    </td>
                    <td className="py-4 px-6 text-slate-500 text-xs">
                      {l.interest_type === 'SIMPLE' ? `${l.annual_interest_rate}% p.a.` : 'Interest-free'}
                    </td>
                    <td className="py-4 px-6 text-slate-700">
                      ₹{parseFloat(l.summary?.total_amount_due || l.principal_amount).toLocaleString('en-IN', { minimumFractionDigits: 2 })}
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

export default function LoansPage() {
  return (
    <Suspense
      fallback={
        <div className="flex items-center justify-center min-h-[40vh]">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-600"></div>
        </div>
      }
    >
      <LoansContent />
    </Suspense>
  );
}
