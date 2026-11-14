'use client';

import React, { useEffect, useState } from 'react';
import Link from 'next/link';
import { useRouter, useParams } from 'next/navigation';
import { api } from '@/lib/api';
import { Loan, Repayment, LedgerEvent } from '@/types/api';
import { StatusBadge } from '@/components/StatusBadge';
import { RecordRepaymentModal } from '@/components/RecordRepaymentModal';
import { BorrowerLinkModal } from '@/components/BorrowerLinkModal';
import { AmendLoanModal } from '@/components/AmendLoanModal';
import { ReasonActionModal } from '@/components/ReasonActionModal';
import { 
  ArrowLeft, 
  CreditCard, 
  PlusCircle, 
  Share2, 
  CheckCircle, 
  XCircle, 
  RotateCcw, 
  AlertTriangle, 
  History, 
  FileText,
  AlertCircle
} from 'lucide-react';

export default function LoanDetailPage() {
  const router = useRouter();
  const params = useParams();
  const loanID = params?.id as string;

  const [loan, setLoan] = useState<Loan | null>(null);
  const [repayments, setRepayments] = useState<Repayment[]>([]);
  const [events, setEvents] = useState<LedgerEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Modals state
  const [repaymentModalOpen, setRepaymentModalOpen] = useState(false);
  const [linkModalOpen, setLinkModalOpen] = useState(false);
  const [amendModalOpen, setAmendModalOpen] = useState(false);

  // Reason modals
  const [cancelModalOpen, setCancelModalOpen] = useState(false);
  const [disputeModalOpen, setDisputeModalOpen] = useState(false);
  const [rejectRepaymentID, setRejectRepaymentID] = useState<string | null>(null);
  const [reverseRepaymentID, setReverseRepaymentID] = useState<string | null>(null);

  const fetchLoanData = async () => {
    try {
      setLoading(true);
      const [loanData, repData, eventData] = await Promise.all([
        api.getLoan(loanID),
        api.listRepayments(loanID),
        api.listLoanEvents(loanID),
      ]);
      setLoan(loanData);
      setRepayments(repData);
      setEvents(eventData);
    } catch (err: any) {
      if (err?.code === 'UNAUTHORIZED') {
        api.clearAuth();
        router.replace('/login');
        return;
      }
      setError(err?.message || 'Failed to load loan details');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (!api.isAuthenticated()) {
      router.replace('/login');
      return;
    }
    if (loanID) {
      fetchLoanData();
    }
  }, [router, loanID]);

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[50vh]">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-600"></div>
      </div>
    );
  }

  if (!loan) {
    return (
      <div className="p-8 text-center bg-white rounded-2xl border border-slate-200">
        <p className="text-slate-600 mb-4">Loan agreement not found.</p>
        <Link href="/loans" className="text-indigo-600 text-sm font-semibold">
          &larr; Back to Loans
        </Link>
      </div>
    );
  }

  const handleActivate = async () => {
    if (!confirm('Activate this loan? This formalizes the agreement and enables repayment tracking and amendments.')) return;
    try {
      await api.activateLoan(loan.id);
      fetchLoanData();
    } catch (err: any) {
      alert(err?.message || 'Activation failed');
    }
  };

  const handleConfirmRepayment = async (id: string) => {
    if (!confirm('Confirm receipt of this payment? This will reduce the outstanding loan balance.')) return;
    try {
      await api.confirmRepayment(id);
      fetchLoanData();
    } catch (err: any) {
      alert(err?.message || 'Confirmation failed');
    }
  };

  return (
    <div className="space-y-8">
      {/* Back button & Header */}
      <div>
        <Link
          href="/loans"
          className="inline-flex items-center text-xs font-semibold text-slate-500 hover:text-slate-800 mb-3 transition"
        >
          <ArrowLeft className="w-3.5 h-3.5 mr-1" />
          Back to all loans
        </Link>

        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
          <div className="flex items-center gap-3">
            <h1 className="text-2xl font-bold text-slate-900 tracking-tight">
              Loan with {loan.peer_name || 'Borrower'}
            </h1>
            <StatusBadge status={loan.status} />
            <span className="text-xs font-mono bg-slate-100 text-slate-600 px-2 py-0.5 rounded border border-slate-200">
              v{loan.version}
            </span>
          </div>

          {/* Action buttons based on status */}
          <div className="flex items-center flex-wrap gap-2">
            {loan.status === 'DRAFT' && (
              <button
                onClick={handleActivate}
                className="inline-flex items-center px-4 py-2 text-xs font-semibold text-white bg-emerald-600 hover:bg-emerald-700 rounded-lg shadow-sm transition"
              >
                <CheckCircle className="w-4 h-4 mr-1.5" />
                Activate Loan
              </button>
            )}

            {loan.status === 'ACTIVE' && (
              <>
                <button
                  onClick={() => setLinkModalOpen(true)}
                  className="inline-flex items-center px-3.5 py-2 text-xs font-semibold text-indigo-700 bg-indigo-50 border border-indigo-200 hover:bg-indigo-100 rounded-lg shadow-sm transition"
                >
                  <Share2 className="w-3.5 h-3.5 mr-1.5" />
                  Borrower Link
                </button>
                <button
                  onClick={() => setRepaymentModalOpen(true)}
                  className="inline-flex items-center px-4 py-2 text-xs font-semibold text-white bg-indigo-600 hover:bg-indigo-700 rounded-lg shadow-sm transition"
                >
                  <PlusCircle className="w-4 h-4 mr-1.5" />
                  Record Repayment
                </button>
                <button
                  onClick={() => setAmendModalOpen(true)}
                  className="inline-flex items-center px-3 py-2 text-xs font-medium text-slate-700 bg-white border border-slate-200 hover:bg-slate-50 rounded-lg transition"
                >
                  Amend Terms
                </button>
                <button
                  onClick={() => setDisputeModalOpen(true)}
                  className="inline-flex items-center px-3 py-2 text-xs font-medium text-amber-700 bg-amber-50 border border-amber-200 hover:bg-amber-100 rounded-lg transition"
                >
                  <AlertTriangle className="w-3.5 h-3.5 mr-1" />
                  Dispute
                </button>
              </>
            )}

            {loan.status === 'DISPUTED' && (
              <>
                <button
                  onClick={() => setLinkModalOpen(true)}
                  className="inline-flex items-center px-3.5 py-2 text-xs font-semibold text-indigo-700 bg-indigo-50 border border-indigo-200 hover:bg-indigo-100 rounded-lg shadow-sm transition"
                >
                  <Share2 className="w-3.5 h-3.5 mr-1.5" />
                  Borrower Link
                </button>
                <button
                  onClick={() => setAmendModalOpen(true)}
                  className="inline-flex items-center px-3.5 py-2 text-xs font-semibold text-white bg-indigo-600 hover:bg-indigo-700 rounded-lg shadow-sm transition"
                >
                  Amend &amp; Resolve Dispute
                </button>
              </>
            )}

            {(loan.status === 'DRAFT' || loan.status === 'ACTIVE') && (
              <button
                onClick={() => setCancelModalOpen(true)}
                className="inline-flex items-center px-3 py-2 text-xs font-medium text-rose-600 hover:bg-rose-50 border border-transparent rounded-lg transition"
              >
                Cancel
              </button>
            )}
          </div>
        </div>
      </div>

      {error && (
        <div className="p-4 bg-rose-50 border border-rose-200 rounded-xl flex items-center text-rose-700 text-sm">
          <AlertCircle className="w-5 h-5 mr-3 flex-shrink-0" />
          <span>{error}</span>
        </div>
      )}

      {/* Financial Breakdown Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-4">
        <div className="bg-white p-5 rounded-2xl border border-slate-200 shadow-sm">
          <span className="text-xs font-semibold text-slate-500 uppercase tracking-wider block mb-1">
            Principal
          </span>
          <div className="text-xl font-bold text-slate-900">
            ₹{parseFloat(loan.principal_amount).toLocaleString('en-IN', { minimumFractionDigits: 2 })}
          </div>
          <p className="text-xs text-slate-400 mt-1">Disbursed on {loan.start_date}</p>
        </div>

        <div className="bg-white p-5 rounded-2xl border border-slate-200 shadow-sm">
          <span className="text-xs font-semibold text-slate-500 uppercase tracking-wider block mb-1">
            Interest Accrued
          </span>
          <div className="text-xl font-bold text-slate-900">
            ₹{parseFloat(loan.summary?.interest_accrued || '0.00').toLocaleString('en-IN', { minimumFractionDigits: 2 })}
          </div>
          <p className="text-xs text-slate-400 mt-1">
            {loan.interest_type === 'SIMPLE' ? `${loan.annual_interest_rate}% p.a. (${loan.summary?.elapsed_days || 0}d)` : 'Interest-free'}
          </p>
        </div>

        <div className="bg-white p-5 rounded-2xl border border-slate-200 shadow-sm">
          <span className="text-xs font-semibold text-slate-500 uppercase tracking-wider block mb-1">
            Total Amount Due
          </span>
          <div className="text-xl font-bold text-slate-900">
            ₹{parseFloat(loan.summary?.total_amount_due || loan.principal_amount).toLocaleString('en-IN', { minimumFractionDigits: 2 })}
          </div>
          <p className="text-xs text-slate-400 mt-1">Principal + interest</p>
        </div>

        <div className="bg-white p-5 rounded-2xl border border-slate-200 shadow-sm">
          <span className="text-xs font-semibold text-slate-500 uppercase tracking-wider block mb-1">
            Total Repaid
          </span>
          <div className="text-xl font-bold text-emerald-600">
            ₹{parseFloat(loan.summary?.total_confirmed_repayments || '0.00').toLocaleString('en-IN', { minimumFractionDigits: 2 })}
          </div>
          <p className="text-xs text-slate-400 mt-1">Confirmed payments</p>
        </div>

        <div className="bg-white p-5 rounded-2xl border-2 border-indigo-500/40 shadow-sm bg-indigo-50/20">
          <span className="text-xs font-semibold text-indigo-700 uppercase tracking-wider block mb-1">
            Outstanding Balance
          </span>
          <div className="text-2xl font-black text-indigo-950">
            ₹{parseFloat(loan.summary?.outstanding_balance || loan.principal_amount).toLocaleString('en-IN', { minimumFractionDigits: 2 })}
          </div>
          <p className="text-xs text-indigo-600 mt-1">
            {loan.due_date ? `Due by ${loan.due_date}` : 'Open-ended'}
          </p>
        </div>
      </div>

      {/* Agreed Terms & Notes */}
      <div className="bg-white p-6 rounded-2xl border border-slate-200 shadow-sm">
        <h2 className="text-base font-semibold text-slate-900 mb-4">Agreement Specifications</h2>
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-4 text-xs">
          <div>
            <span className="text-slate-400 block mb-0.5">Borrower</span>
            <span className="font-semibold text-slate-800">{loan.peer_name}</span>
          </div>
          <div>
            <span className="text-slate-400 block mb-0.5">Day-Count Convention</span>
            <span className="font-semibold text-slate-800">{loan.day_count_convention}</span>
          </div>
          <div>
            <span className="text-slate-400 block mb-0.5">Currency</span>
            <span className="font-semibold text-slate-800">{loan.currency}</span>
          </div>
          <div>
            <span className="text-slate-400 block mb-0.5">Last Updated</span>
            <span className="font-semibold text-slate-800">{new Date(loan.updated_at).toLocaleString()}</span>
          </div>
        </div>

        {loan.notes && (
          <div className="mt-4 pt-4 border-t border-slate-100 text-xs">
            <span className="text-slate-400 block mb-1 font-medium">Notes &amp; Agreement Terms</span>
            <p className="text-slate-700 bg-slate-50 p-3 rounded-lg border border-slate-100">
              {loan.notes}
            </p>
          </div>
        )}
      </div>

      {/* Repayments Section */}
      <div className="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden">
        <div className="p-6 border-b border-slate-100 flex items-center justify-between">
          <div>
            <h2 className="text-base font-semibold text-slate-900">Repayment Records</h2>
            <p className="text-xs text-slate-500 mt-0.5">
              Only confirmed repayments reduce the verified outstanding balance
            </p>
          </div>
          {loan.status === 'ACTIVE' && (
            <button
              onClick={() => setRepaymentModalOpen(true)}
              className="inline-flex items-center px-3.5 py-1.5 text-xs font-semibold text-white bg-indigo-600 hover:bg-indigo-700 rounded-lg shadow-sm transition"
            >
              <PlusCircle className="w-3.5 h-3.5 mr-1.5" />
              Record Repayment
            </button>
          )}
        </div>

        {repayments.length === 0 ? (
          <div className="p-10 text-center">
            <CreditCard className="w-8 h-8 text-slate-300 mx-auto mb-2" />
            <p className="text-xs text-slate-500">No repayments recorded for this loan yet.</p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead className="bg-slate-50 text-[11px] font-semibold text-slate-500 uppercase tracking-wider border-b border-slate-100">
                <tr>
                  <th className="py-3 px-6">Date</th>
                  <th className="py-3 px-6">Amount</th>
                  <th className="py-3 px-6">Method</th>
                  <th className="py-3 px-6">Source</th>
                  <th className="py-3 px-6">Status</th>
                  <th className="py-3 px-6">Reference / Notes</th>
                  <th className="py-3 px-6 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {repayments.map((r) => (
                  <tr key={r.id} className="hover:bg-slate-50/70 transition">
                    <td className="py-4 px-6 text-slate-800 font-medium whitespace-nowrap">
                      {r.payment_date}
                    </td>
                    <td className="py-4 px-6 font-semibold text-slate-900 whitespace-nowrap">
                      ₹{parseFloat(r.amount).toLocaleString('en-IN', { minimumFractionDigits: 2 })}
                    </td>
                    <td className="py-4 px-6 text-xs text-slate-600">{r.payment_method}</td>
                    <td className="py-4 px-6 text-xs text-slate-500">
                      {r.source === 'BORROWER_REPORTED' ? (
                        <span className="text-indigo-600 font-medium">Borrower Self-Report</span>
                      ) : (
                        <span>Lender Recorded</span>
                      )}
                    </td>
                    <td className="py-4 px-6">
                      <StatusBadge status={r.status} />
                    </td>
                    <td className="py-4 px-6 text-xs text-slate-500 max-w-xs truncate">
                      {r.external_reference && (
                        <span className="font-mono bg-slate-100 px-1 py-0.5 rounded text-[11px] mr-1">
                          {r.external_reference}
                        </span>
                      )}
                      {r.rejection_reason && (
                        <span className="text-rose-600 font-medium block">
                          Reason: {r.rejection_reason}
                        </span>
                      )}
                      {r.reversal_reason && (
                        <span className="text-purple-600 font-medium block">
                          Reversed: {r.reversal_reason}
                        </span>
                      )}
                      {r.notes}
                    </td>
                    <td className="py-4 px-6 text-right whitespace-nowrap space-x-2 text-xs">
                      {r.status === 'PENDING_VERIFICATION' && (
                        <>
                          <button
                            onClick={() => handleConfirmRepayment(r.id)}
                            className="font-semibold text-emerald-600 hover:text-emerald-800"
                          >
                            Confirm
                          </button>
                          <button
                            onClick={() => setRejectRepaymentID(r.id)}
                            className="font-semibold text-rose-600 hover:text-rose-800"
                          >
                            Reject
                          </button>
                        </>
                      )}

                      {r.status === 'CONFIRMED' && (
                        <button
                          onClick={() => setReverseRepaymentID(r.id)}
                          className="font-semibold text-purple-600 hover:text-purple-800"
                        >
                          Reverse
                        </button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Append-only Audit Trail / Ledger Events */}
      <div className="bg-white rounded-2xl border border-slate-200 shadow-sm p-6">
        <div className="flex items-center gap-2 mb-4">
          <History className="w-4 h-4 text-indigo-600" />
          <h2 className="text-base font-semibold text-slate-900">Audit Trail &amp; Ledger Events</h2>
        </div>

        {events.length === 0 ? (
          <p className="text-xs text-slate-400">No events logged yet.</p>
        ) : (
          <div className="relative pl-6 space-y-6 before:absolute before:left-2 before:top-2 before:bottom-2 before:w-0.5 before:bg-slate-200">
            {events.map((e) => (
              <div key={e.id} className="relative group text-xs">
                <div className="absolute -left-6 top-1 w-2.5 h-2.5 rounded-full bg-indigo-600 ring-4 ring-white" />
                <div className="flex items-center gap-2 mb-1">
                  <span className="font-bold text-slate-800 font-mono text-[11px] bg-slate-100 px-1.5 py-0.5 rounded">
                    {e.event_type}
                  </span>
                  <span className="text-slate-400">
                    {new Date(e.created_at).toLocaleString()}
                  </span>
                  <span className="text-slate-500 font-medium">by {e.actor_type}</span>
                </div>
                {e.payload && Object.keys(e.payload).length > 0 && (
                  <pre className="mt-1 p-2 bg-slate-50 rounded border border-slate-100 text-[11px] font-mono text-slate-600 overflow-x-auto">
                    {JSON.stringify(e.payload, null, 2)}
                  </pre>
                )}
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Modals */}
      <RecordRepaymentModal
        loan={loan}
        isOpen={repaymentModalOpen}
        onClose={() => setRepaymentModalOpen(false)}
        onSuccess={fetchLoanData}
      />

      <BorrowerLinkModal
        loan={loan}
        isOpen={linkModalOpen}
        onClose={() => setLinkModalOpen(false)}
      />

      <AmendLoanModal
        loan={loan}
        isOpen={amendModalOpen}
        onClose={() => setAmendModalOpen(false)}
        onSuccess={fetchLoanData}
      />

      {/* Cancel Loan Modal */}
      <ReasonActionModal
        title="Cancel Loan Agreement"
        description="Cancelling will terminate this loan agreement and prevent further transactions."
        actionLabel="Confirm Cancellation"
        isDestructive={true}
        isOpen={cancelModalOpen}
        onClose={() => setCancelModalOpen(false)}
        onSubmit={async (reason) => {
          await api.cancelLoan(loan.id, reason);
          fetchLoanData();
        }}
      />

      {/* Dispute Loan Modal */}
      <ReasonActionModal
        title="Mark Loan as Disputed"
        description="Marking as disputed will signal disagreements in loan details and pause standard operations until resolved."
        actionLabel="Mark Disputed"
        isDestructive={true}
        isOpen={disputeModalOpen}
        onClose={() => setDisputeModalOpen(false)}
        onSubmit={async (reason) => {
          await api.disputeLoan(loan.id, reason);
          fetchLoanData();
        }}
      />

      {/* Reject Repayment Modal */}
      <ReasonActionModal
        title="Reject Repayment Report"
        description="Provide a reason why this borrower payment report was rejected (e.g. funds not credited)."
        actionLabel="Confirm Rejection"
        isDestructive={true}
        isOpen={!!rejectRepaymentID}
        onClose={() => setRejectRepaymentID(null)}
        onSubmit={async (reason) => {
          if (rejectRepaymentID) {
            await api.rejectRepayment(rejectRepaymentID, reason);
            fetchLoanData();
          }
        }}
      />

      {/* Reverse Repayment Modal */}
      <ReasonActionModal
        title="Reverse Confirmed Repayment"
        description="Reversing will restore the loan's outstanding balance and log an auditable reversal reason."
        actionLabel="Confirm Reversal"
        isDestructive={true}
        isOpen={!!reverseRepaymentID}
        onClose={() => setReverseRepaymentID(null)}
        onSubmit={async (reason) => {
          if (reverseRepaymentID) {
            await api.reverseRepayment(reverseRepaymentID, reason);
            fetchLoanData();
          }
        }}
      />
    </div>
  );
}
