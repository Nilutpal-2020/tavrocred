'use client';

import React, { useEffect, useState, Suspense } from 'react';
import Link from 'next/link';
import { useRouter, useSearchParams } from 'next/navigation';
import { api } from '@/lib/api';
import { Peer } from '@/types/api';
import { ArrowLeft, PlusCircle, AlertCircle } from 'lucide-react';

function NewLoanContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const preselectedPeerID = searchParams.get('peer_id') || '';

  const [peers, setPeers] = useState<Peer[]>([]);
  const [loadingPeers, setLoadingPeers] = useState(true);

  // Form state
  const [peerID, setPeerID] = useState(preselectedPeerID);
  const [principalAmount, setPrincipalAmount] = useState('');
  const [currency, setCurrency] = useState('INR');
  const [interestType, setInterestType] = useState<'NONE' | 'SIMPLE'>('NONE');
  const [annualRate, setAnnualRate] = useState('0');
  const [startDate, setStartDate] = useState(new Date().toISOString().split('T')[0]);
  const [dueDate, setDueDate] = useState('');
  const [notes, setNotes] = useState('');

  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!api.isAuthenticated()) {
      router.replace('/login');
      return;
    }

    const fetchPeers = async () => {
      try {
        const data = await api.listPeers();
        setPeers(data);
        if (!peerID && data.length > 0) {
          setPeerID(preselectedPeerID || data[0].id);
        }
      } catch (err: any) {
        setError(err?.message || 'Failed to load borrower contacts');
      } finally {
        setLoadingPeers(false);
      }
    };

    fetchPeers();
  }, [router, preselectedPeerID, peerID]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setSubmitting(true);

    try {
      const loan = await api.createLoan({
        peer_id: peerID,
        principal_amount: principalAmount,
        currency,
        interest_type: interestType,
        annual_interest_rate: interestType === 'SIMPLE' ? annualRate : '0.00',
        start_date: startDate,
        due_date: dueDate || undefined,
        notes: notes || undefined,
      });

      router.push(`/loans/${loan.id}`);
    } catch (err: any) {
      setError(err?.message || 'Failed to record loan agreement');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      <div className="flex items-center gap-3">
        <Link
          href="/loans"
          className="p-2 text-slate-400 hover:text-slate-600 rounded-lg hover:bg-white border border-transparent hover:border-slate-200 transition"
        >
          <ArrowLeft className="w-4 h-4" />
        </Link>
        <div>
          <h1 className="text-2xl font-bold text-slate-900 tracking-tight">Record New Loan</h1>
          <p className="text-sm text-slate-500 mt-0.5">
            Create a formal agreement draft for personal lending
          </p>
        </div>
      </div>

      {error && (
        <div className="p-4 bg-rose-50 border border-rose-200 rounded-xl flex items-start text-rose-700 text-sm">
          <AlertCircle className="w-5 h-5 mr-3 flex-shrink-0 mt-0.5" />
          <div>
            <p className="font-semibold">Unable to record loan</p>
            <p className="text-rose-600 mt-0.5">{error}</p>
          </div>
        </div>
      )}

      <form onSubmit={handleSubmit} className="bg-white p-6 sm:p-8 rounded-2xl border border-slate-200 shadow-sm space-y-6">
        {/* Borrower selection */}
        <div>
          <div className="flex items-center justify-between mb-1.5">
            <label className="block text-xs font-semibold text-slate-700 uppercase tracking-wider">
              Borrower Contact *
            </label>
            <Link
              href="/peers"
              className="text-xs font-semibold text-indigo-600 hover:text-indigo-700"
            >
              + Add New Borrower
            </Link>
          </div>

          {loadingPeers ? (
            <div className="text-xs text-slate-400 py-2">Loading borrowers...</div>
          ) : peers.length === 0 ? (
            <div className="p-4 bg-amber-50 border border-amber-200 rounded-lg text-amber-800 text-xs">
              No borrower contacts found.{' '}
              <Link href="/peers" className="font-bold underline">
                Please create a borrower contact
              </Link>{' '}
              first.
            </div>
          ) : (
            <select
              required
              value={peerID}
              onChange={(e) => setPeerID(e.target.value)}
              className="w-full px-3.5 py-2.5 bg-slate-50 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
            >
              {peers.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.name} {p.phone ? `(${p.phone})` : ''}
                </option>
              ))}
            </select>
          )}
        </div>

        {/* Amount and Currency */}
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <div className="sm:col-span-2">
            <label className="block text-xs font-semibold text-slate-700 uppercase tracking-wider mb-1.5">
              Principal Amount *
            </label>
            <div className="relative">
              <span className="absolute inset-y-0 left-0 pl-3.5 flex items-center text-slate-400 font-semibold text-sm">
                ₹
              </span>
              <input
                type="number"
                step="0.01"
                min="1"
                required
                value={principalAmount}
                onChange={(e) => setPrincipalAmount(e.target.value)}
                placeholder="50000.00"
                className="w-full pl-8 pr-3.5 py-2.5 bg-slate-50 border border-slate-200 rounded-lg text-sm font-medium focus:outline-none focus:ring-2 focus:ring-indigo-500"
              />
            </div>
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-700 uppercase tracking-wider mb-1.5">
              Currency
            </label>
            <input
              type="text"
              disabled
              value={currency}
              className="w-full px-3.5 py-2.5 bg-slate-100 border border-slate-200 rounded-lg text-sm text-slate-500 cursor-not-allowed"
            />
          </div>
        </div>

        {/* Interest Type */}
        <div className="space-y-3 pt-2 border-t border-slate-100">
          <label className="block text-xs font-semibold text-slate-700 uppercase tracking-wider">
            Interest Structure
          </label>
          <div className="grid grid-cols-2 gap-4">
            <label
              className={`p-3.5 rounded-xl border cursor-pointer flex flex-col transition ${
                interestType === 'NONE'
                  ? 'border-indigo-600 bg-indigo-50/50 text-indigo-900'
                  : 'border-slate-200 hover:bg-slate-50 text-slate-700'
              }`}
            >
              <div className="flex items-center gap-2">
                <input
                  type="radio"
                  name="interestType"
                  value="NONE"
                  checked={interestType === 'NONE'}
                  onChange={() => setInterestType('NONE')}
                  className="text-indigo-600 focus:ring-indigo-500"
                />
                <span className="font-semibold text-sm">Interest-free</span>
              </div>
              <span className="text-xs text-slate-500 mt-1 pl-5">
                Principal returned with zero interest
              </span>
            </label>

            <label
              className={`p-3.5 rounded-xl border cursor-pointer flex flex-col transition ${
                interestType === 'SIMPLE'
                  ? 'border-indigo-600 bg-indigo-50/50 text-indigo-900'
                  : 'border-slate-200 hover:bg-slate-50 text-slate-700'
              }`}
            >
              <div className="flex items-center gap-2">
                <input
                  type="radio"
                  name="interestType"
                  value="SIMPLE"
                  checked={interestType === 'SIMPLE'}
                  onChange={() => setInterestType('SIMPLE')}
                  className="text-indigo-600 focus:ring-indigo-500"
                />
                <span className="font-semibold text-sm">Simple Interest</span>
              </div>
              <span className="text-xs text-slate-500 mt-1 pl-5">
                ACTUAL/365 annual accrual convention
              </span>
            </label>
          </div>

          {interestType === 'SIMPLE' && (
            <div className="pt-2">
              <label className="block text-xs font-semibold text-slate-700 mb-1.5">
                Annual Interest Rate (% p.a.) *
              </label>
              <input
                type="number"
                step="0.01"
                min="0.01"
                max="100"
                required
                value={annualRate}
                onChange={(e) => setAnnualRate(e.target.value)}
                placeholder="12.00"
                className="w-full sm:w-1/2 px-3.5 py-2.5 bg-slate-50 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
              />
            </div>
          )}
        </div>

        {/* Start and Due Date */}
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 pt-2 border-t border-slate-100">
          <div>
            <label className="block text-xs font-semibold text-slate-700 uppercase tracking-wider mb-1.5">
              Start / Disbursal Date *
            </label>
            <input
              type="date"
              required
              value={startDate}
              onChange={(e) => setStartDate(e.target.value)}
              className="w-full px-3.5 py-2.5 bg-slate-50 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
            />
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-700 uppercase tracking-wider mb-1.5">
              Target Due Date (Optional)
            </label>
            <input
              type="date"
              value={dueDate}
              min={startDate}
              onChange={(e) => setDueDate(e.target.value)}
              className="w-full px-3.5 py-2.5 bg-slate-50 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
            />
          </div>
        </div>

        {/* Notes */}
        <div className="pt-2 border-t border-slate-100">
          <label className="block text-xs font-semibold text-slate-700 uppercase tracking-wider mb-1.5">
            Agreement Notes / Purpose
          </label>
          <textarea
            rows={3}
            value={notes}
            onChange={(e) => setNotes(e.target.value)}
            placeholder="e.g. Lent for emergency hospital medical expenses; promised repayment upon monthly salary"
            className="w-full px-3.5 py-2.5 bg-slate-50 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
          />
        </div>

        {/* Submit button */}
        <div className="flex items-center justify-end gap-3 pt-4 border-t border-slate-100">
          <Link
            href="/loans"
            className="px-4 py-2.5 text-xs font-semibold text-slate-700 border border-slate-200 rounded-lg hover:bg-slate-50 transition"
          >
            Cancel
          </Link>

          <button
            type="submit"
            disabled={submitting || peers.length === 0}
            className="inline-flex items-center px-5 py-2.5 text-xs font-semibold text-white bg-indigo-600 hover:bg-indigo-700 rounded-lg shadow-sm disabled:opacity-60 transition"
          >
            {submitting ? 'Recording Draft...' : 'Record Agreement'}
          </button>
        </div>
      </form>
    </div>
  );
}

export default function NewLoanPage() {
  return (
    <Suspense
      fallback={
        <div className="flex items-center justify-center min-h-[40vh]">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-600"></div>
        </div>
      }
    >
      <NewLoanContent />
    </Suspense>
  );
}
