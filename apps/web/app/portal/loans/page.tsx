'use client';

import React, { useEffect, useState, Suspense } from 'react';
import { useSearchParams } from 'next/navigation';
import { api } from '@/lib/api';
import { BorrowerLoanViewResponse } from '@/types/api';
import { StatusBadge } from '@/components/StatusBadge';
import { 
  CheckCircle2, 
  AlertTriangle, 
  CreditCard, 
  ShieldCheck, 
  Calendar, 
  Clock, 
  Send, 
  AlertCircle,
  HelpCircle,
  PlusCircle
} from 'lucide-react';

function BorrowerPortalContent() {
  const searchParams = useSearchParams();
  const rawToken = searchParams.get('token');

  const [sessionToken, setSessionToken] = useState<string | null>(null);
  const [data, setData] = useState<BorrowerLoanViewResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Dispute modal
  const [disputeOpen, setDisputeOpen] = useState(false);
  const [disputeReason, setDisputeReason] = useState('');
  const [submittingDispute, setSubmittingDispute] = useState(false);

  // Report payment form
  const [reportOpen, setReportOpen] = useState(false);
  const [reportAmount, setReportAmount] = useState('');
  const [reportDate, setReportDate] = useState(new Date().toISOString().split('T')[0]);
  const [reportMethod, setReportMethod] = useState('UPI');
  const [reportRef, setReportRef] = useState('');
  const [reportNotes, setReportNotes] = useState('');
  const [submittingReport, setSubmittingReport] = useState(false);

  // Success notifications
  const [successMsg, setSuccessMsg] = useState<string | null>(null);

  // 1. Initial exchange: rawToken -> sessionToken
  useEffect(() => {
    if (!rawToken) {
      setError('Access link is missing or invalid. Please check the URL provided by your lender.');
      setLoading(false);
      return;
    }

    const exchange = async () => {
      try {
        setLoading(true);
        const access = await api.exchangeBorrowerToken(rawToken);
        setSessionToken(access.session_token);
        const loanView = await api.getBorrowerLoan(access.session_token);
        setData(loanView);
      } catch (err: any) {
        setError(err?.message || 'Invalid, expired, or revoked access link.');
      } finally {
        setLoading(false);
      }
    };

    exchange();
  }, [rawToken]);

  const refreshData = async () => {
    if (!sessionToken) return;
    try {
      const loanView = await api.getBorrowerLoan(sessionToken);
      setData(loanView);
    } catch (err) {
      // ignore
    }
  };

  const handleAcknowledge = async () => {
    if (!sessionToken) return;
    try {
      await api.acknowledgeBorrowerLoan(sessionToken);
      setSuccessMsg('You have successfully acknowledged this loan agreement.');
      setTimeout(() => setSuccessMsg(null), 5000);
      refreshData();
    } catch (err: any) {
      alert(err?.message || 'Acknowledgement failed');
    }
  };

  const handleDispute = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!sessionToken) return;
    setSubmittingDispute(true);
    try {
      await api.disputeBorrowerLoan(sessionToken, disputeReason);
      setDisputeOpen(false);
      setDisputeReason('');
      setSuccessMsg('Dispute noted. The agreement status has been updated for both parties.');
      setTimeout(() => setSuccessMsg(null), 5000);
      refreshData();
    } catch (err: any) {
      alert(err?.message || 'Dispute failed');
    } finally {
      setSubmittingDispute(false);
    }
  };

  const handleReportPayment = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!sessionToken) return;
    setSubmittingReport(true);
    try {
      await api.reportBorrowerRepayment(sessionToken, {
        amount: reportAmount,
        payment_date: reportDate,
        payment_method: reportMethod,
        external_reference: reportRef || undefined,
        notes: reportNotes || undefined,
      });
      setReportOpen(false);
      setReportAmount('');
      setReportRef('');
      setReportNotes('');
      setSuccessMsg('Your payment report was submitted. Your lender will verify and confirm receipt.');
      setTimeout(() => setSuccessMsg(null), 6000);
      refreshData();
    } catch (err: any) {
      alert(err?.message || 'Failed to report payment');
    } finally {
      setSubmittingReport(false);
    }
  };

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-slate-50">
        <div className="text-center space-y-3">
          <div className="animate-spin rounded-full h-9 w-9 border-b-2 border-indigo-600 mx-auto"></div>
          <p className="text-xs font-semibold text-slate-500 uppercase tracking-wider">
            Verifying secure access link...
          </p>
        </div>
      </div>
    );
  }

  if (error || !data) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-slate-50 p-4">
        <div className="max-w-md w-full bg-white p-8 rounded-2xl border border-slate-200 text-center shadow-sm">
          <div className="w-12 h-12 bg-rose-50 text-rose-600 rounded-full flex items-center justify-center mx-auto mb-4">
            <AlertCircle className="w-6 h-6" />
          </div>
          <h1 className="text-lg font-bold text-slate-900 mb-2">Access Link Error</h1>
          <p className="text-xs text-slate-500 mb-6">{error}</p>
          <div className="p-3 bg-slate-50 rounded-xl text-xs text-slate-600 text-left border border-slate-100">
            <span className="font-semibold text-slate-800 block mb-1">What happened?</span>
            The link you opened may have expired, been revoked by your lender, or contains an invalid security token. Please ask your lender to share a fresh borrower link.
          </div>
        </div>
      </div>
    );
  }

  const { loan, summary, lender, latest_acknowledgement, repayments } = data;

  return (
    <div className="min-h-screen bg-slate-50 py-8 px-4 sm:px-6 lg:px-8">
      <div className="max-w-2xl mx-auto space-y-6">
        {/* Brand & Security Header */}
        <div className="text-center pb-2">
          <div className="inline-flex items-center gap-2 bg-indigo-50 border border-indigo-100 px-3 py-1 rounded-full text-indigo-700 text-xs font-semibold mb-3">
            <ShieldCheck className="w-4 h-4 text-emerald-600" />
            Verified Borrower Portal · Tavrocred
          </div>
          <h1 className="text-2xl font-black text-slate-900 tracking-tight">
            Personal Loan Agreement
          </h1>
          <p className="text-xs text-slate-500 mt-1">
            Shared by <span className="font-semibold text-slate-700">{lender.name}</span> ({lender.email})
          </p>
        </div>

        {successMsg && (
          <div className="p-4 bg-emerald-50 border border-emerald-200 rounded-xl flex items-center text-emerald-800 text-xs">
            <CheckCircle2 className="w-5 h-5 mr-3 text-emerald-600 flex-shrink-0" />
            <span>{successMsg}</span>
          </div>
        )}

        {/* Live Balance Card */}
        <div className="bg-white rounded-2xl border-2 border-indigo-600/30 p-6 shadow-sm">
          <div className="flex items-center justify-between pb-4 border-b border-slate-100">
            <div>
              <span className="text-xs font-semibold text-slate-400 uppercase tracking-wider block">
                Current Outstanding Balance
              </span>
              <div className="text-3xl font-black text-indigo-950 mt-1">
                ₹{parseFloat(summary.outstanding_balance).toLocaleString('en-IN', { minimumFractionDigits: 2 })}
              </div>
            </div>
            <div className="text-right">
              <StatusBadge status={loan.status} />
              <div className="text-[11px] text-slate-400 mt-1 font-mono">v{loan.version} agreement</div>
            </div>
          </div>

          <div className="grid grid-cols-2 sm:grid-cols-4 gap-4 pt-4 text-xs">
            <div>
              <span className="text-slate-400 block mb-0.5">Original Principal</span>
              <span className="font-bold text-slate-800">
                ₹{parseFloat(loan.principal_amount).toLocaleString('en-IN', { minimumFractionDigits: 2 })}
              </span>
            </div>
            <div>
              <span className="text-slate-400 block mb-0.5">Accrued Interest</span>
              <span className="font-bold text-slate-800">
                ₹{parseFloat(summary.interest_accrued).toLocaleString('en-IN', { minimumFractionDigits: 2 })}
              </span>
            </div>
            <div>
              <span className="text-slate-400 block mb-0.5">Total Confirmed Repaid</span>
              <span className="font-bold text-emerald-600">
                ₹{parseFloat(summary.total_confirmed_repayments).toLocaleString('en-IN', { minimumFractionDigits: 2 })}
              </span>
            </div>
            <div>
              <span className="text-slate-400 block mb-0.5">Start Date</span>
              <span className="font-bold text-slate-800">{loan.start_date}</span>
            </div>
          </div>

          {loan.due_date && (
            <div className="mt-4 pt-3 border-t border-slate-100 flex items-center justify-between text-xs text-slate-600">
              <span className="flex items-center gap-1.5 font-medium">
                <Calendar className="w-3.5 h-3.5 text-indigo-600" />
                Target Due Date: {loan.due_date}
              </span>
              <span className="text-slate-400">
                {loan.interest_type === 'SIMPLE' ? `${loan.annual_interest_rate}% p.a. simple interest` : 'Interest-free loan'}
              </span>
            </div>
          )}

          {loan.notes && (
            <div className="mt-4 pt-3 border-t border-slate-100 text-xs">
              <span className="text-slate-400 block mb-1 font-medium">Lender&apos;s Notes:</span>
              <p className="text-slate-600 italic bg-slate-50 p-2.5 rounded-lg border border-slate-100">
                &ldquo;{loan.notes}&rdquo;
              </p>
            </div>
          )}
        </div>

        {/* Action Panel: Acknowledge / Dispute / Report Payment */}
        <div className="bg-white rounded-2xl border border-slate-200 p-5 shadow-sm space-y-3">
          <h2 className="text-xs font-bold text-slate-800 uppercase tracking-wider">
            Agreement Responses &amp; Actions
          </h2>

          <div className="flex flex-col sm:flex-row gap-2">
            {latest_acknowledgement?.status === 'ACKNOWLEDGED' ? (
              <div className="flex-1 p-3 bg-emerald-50 border border-emerald-200 rounded-xl flex items-center text-emerald-800 text-xs">
                <CheckCircle2 className="w-4 h-4 text-emerald-600 mr-2 flex-shrink-0" />
                <span>
                  You acknowledged agreement version <strong>v{latest_acknowledgement.version}</strong> on{' '}
                  {new Date(latest_acknowledgement.responded_at).toLocaleDateString()}.
                </span>
              </div>
            ) : (
              <button
                onClick={handleAcknowledge}
                className="flex-1 py-3 px-4 bg-emerald-600 hover:bg-emerald-700 text-white font-semibold rounded-xl text-xs shadow-sm flex items-center justify-center gap-2 transition"
              >
                <CheckCircle2 className="w-4 h-4" />
                I Acknowledge These Terms
              </button>
            )}

            <button
              onClick={() => setReportOpen(true)}
              className="py-3 px-4 bg-indigo-600 hover:bg-indigo-700 text-white font-semibold rounded-xl text-xs shadow-sm flex items-center justify-center gap-2 transition"
            >
              <Send className="w-4 h-4" />
              Report a Payment
            </button>

            <button
              onClick={() => setDisputeOpen(true)}
              className="py-3 px-3.5 border border-slate-200 text-slate-600 hover:bg-slate-50 font-medium rounded-xl text-xs flex items-center justify-center gap-1.5 transition"
            >
              <AlertTriangle className="w-4 h-4 text-amber-600" />
              Dispute
            </button>
          </div>
        </div>

        {/* Repayment History Table */}
        <div className="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden">
          <div className="p-5 border-b border-slate-100 flex items-center justify-between">
            <h2 className="text-sm font-bold text-slate-900">Payment History &amp; Reports</h2>
            <span className="text-xs text-slate-400">{repayments.length} entries</span>
          </div>

          {repayments.length === 0 ? (
            <div className="p-8 text-center text-xs text-slate-400">
              No payments reported or recorded yet.
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-left text-xs">
                <thead className="bg-slate-50 text-slate-400 font-semibold border-b border-slate-100">
                  <tr>
                    <th className="py-2.5 px-4">Date</th>
                    <th className="py-2.5 px-4">Amount</th>
                    <th className="py-2.5 px-4">Method</th>
                    <th className="py-2.5 px-4">Status</th>
                    <th className="py-2.5 px-4">Reference</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-100 text-slate-700">
                  {repayments.map((r) => (
                    <tr key={r.id}>
                      <td className="py-3 px-4 font-medium">{r.payment_date}</td>
                      <td className="py-3 px-4 font-bold text-slate-900">
                        ₹{parseFloat(r.amount).toLocaleString('en-IN', { minimumFractionDigits: 2 })}
                      </td>
                      <td className="py-3 px-4">{r.payment_method}</td>
                      <td className="py-3 px-4">
                        <StatusBadge status={r.status} />
                      </td>
                      <td className="py-3 px-4 text-slate-400 font-mono text-[11px]">
                        {r.external_reference || '—'}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>

        {/* Disclaimer / Transparency footer */}
        <div className="p-4 bg-slate-100/60 rounded-xl border border-slate-200 text-center text-[11px] text-slate-400 leading-relaxed">
          Tavrocred is a personal recordkeeping and agreement transparency platform. It does not process payments, charge collection fees, or provide credit scoring. Payments are completed externally between lender and borrower.
        </div>
      </div>

      {/* Dispute Modal */}
      {disputeOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 backdrop-blur-sm p-4">
          <div className="bg-white rounded-2xl max-w-md w-full p-6 shadow-xl border border-slate-200">
            <h2 className="text-base font-bold text-slate-900 mb-1">Dispute Loan Details</h2>
            <p className="text-xs text-slate-500 mb-4">
              If any aspect of the principal amount, interest rate, or dates doesn&apos;t match what you agreed with your lender, state the reason below.
            </p>

            <form onSubmit={handleDispute} className="space-y-4">
              <textarea
                required
                rows={3}
                value={disputeReason}
                onChange={(e) => setDisputeReason(e.target.value)}
                placeholder="Explain what detail is incorrect..."
                className="w-full px-3 py-2 bg-slate-50 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-amber-500"
              />

              <div className="flex justify-end gap-2 pt-2">
                <button
                  type="button"
                  onClick={() => setDisputeOpen(false)}
                  className="px-4 py-2 border border-slate-200 text-xs font-semibold text-slate-700 rounded-lg hover:bg-slate-50"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={submittingDispute}
                  className="px-4 py-2 bg-amber-600 hover:bg-amber-700 text-white text-xs font-semibold rounded-lg shadow-sm disabled:opacity-60"
                >
                  {submittingDispute ? 'Submitting...' : 'Submit Dispute'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Report Payment Modal */}
      {reportOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 backdrop-blur-sm p-4">
          <div className="bg-white rounded-2xl max-w-md w-full p-6 shadow-xl border border-slate-200">
            <h2 className="text-base font-bold text-slate-900 mb-1">Report an External Payment</h2>
            <p className="text-xs text-slate-500 mb-4">
              Let your lender know you transferred money (via UPI, bank transfer, or cash). The payment will be placed in pending status until the lender verifies receipt.
            </p>

            <form onSubmit={handleReportPayment} className="space-y-4">
              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">
                  Amount Transferred (₹) *
                </label>
                <div className="relative">
                  <span className="absolute inset-y-0 left-0 pl-3 flex items-center text-slate-400 font-semibold text-sm">
                    ₹
                  </span>
                  <input
                    type="number"
                    step="0.01"
                    min="1"
                    required
                    value={reportAmount}
                    onChange={(e) => setReportAmount(e.target.value)}
                    placeholder="2500.00"
                    className="w-full pl-8 pr-3 py-2 bg-slate-50 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
                  />
                </div>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs font-semibold text-slate-700 mb-1">
                    Transfer Date *
                  </label>
                  <input
                    type="date"
                    required
                    value={reportDate}
                    max={new Date().toISOString().split('T')[0]}
                    onChange={(e) => setReportDate(e.target.value)}
                    className="w-full px-3 py-2 bg-slate-50 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
                  />
                </div>

                <div>
                  <label className="block text-xs font-semibold text-slate-700 mb-1">
                    Method *
                  </label>
                  <select
                    value={reportMethod}
                    onChange={(e) => setReportMethod(e.target.value)}
                    className="w-full px-3 py-2 bg-slate-50 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
                  >
                    <option value="UPI">UPI (GPay / PhonePe / Paytm)</option>
                    <option value="BANK_TRANSFER">Bank Transfer (IMPS/NEFT)</option>
                    <option value="CASH">Cash in Hand</option>
                    <option value="OTHER">Other</option>
                  </select>
                </div>
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">
                  UPI Ref ID / UTR (Optional)
                </label>
                <input
                  type="text"
                  value={reportRef}
                  onChange={(e) => setReportRef(e.target.value)}
                  placeholder="e.g. 411200983210"
                  className="w-full px-3 py-2 bg-slate-50 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">
                  Notes for Lender
                </label>
                <input
                  type="text"
                  value={reportNotes}
                  onChange={(e) => setReportNotes(e.target.value)}
                  placeholder="e.g. Sent to your HDFC account from ICICI"
                  className="w-full px-3 py-2 bg-slate-50 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
                />
              </div>

              <div className="flex justify-end gap-2 pt-2">
                <button
                  type="button"
                  onClick={() => setReportOpen(false)}
                  className="px-4 py-2 border border-slate-200 text-xs font-semibold text-slate-700 rounded-lg hover:bg-slate-50"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={submittingReport}
                  className="px-4 py-2 bg-indigo-600 hover:bg-indigo-700 text-white text-xs font-semibold rounded-lg shadow-sm disabled:opacity-60"
                >
                  {submittingReport ? 'Submitting...' : 'Submit Payment Report'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}

export default function BorrowerPortalPage() {
  return (
    <Suspense
      fallback={
        <div className="min-h-screen flex items-center justify-center bg-slate-50">
          <div className="text-center space-y-3">
            <div className="animate-spin rounded-full h-9 w-9 border-b-2 border-indigo-600 mx-auto"></div>
            <p className="text-xs font-semibold text-slate-500 uppercase tracking-wider">
              Loading borrower portal...
            </p>
          </div>
        </div>
      }
    >
      <BorrowerPortalContent />
    </Suspense>
  );
}
