'use client';

import React, { useState } from 'react';
import { api } from '@/lib/api';
import { Loan } from '@/types/api';
import { X, AlertCircle } from 'lucide-react';

interface AmendLoanModalProps {
  loan: Loan;
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

export function AmendLoanModal({ loan, isOpen, onClose, onSuccess }: AmendLoanModalProps) {
  const [principalAmount, setPrincipalAmount] = useState(loan.principal_amount);
  const [interestType, setInterestType] = useState<'NONE' | 'SIMPLE'>(loan.interest_type);
  const [annualRate, setAnnualRate] = useState(loan.annual_interest_rate);
  const [dueDate, setDueDate] = useState(loan.due_date || '');
  const [notes, setNotes] = useState(loan.notes || '');
  const [reason, setReason] = useState('');

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  if (!isOpen) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setLoading(true);

    try {
      await api.amendLoan(loan.id, {
        principal_amount: principalAmount,
        interest_type: interestType,
        annual_interest_rate: interestType === 'SIMPLE' ? annualRate : '0.00',
        due_date: dueDate || undefined,
        reason,
        notes: notes || undefined,
      });

      onSuccess();
      onClose();
    } catch (err: any) {
      setError(err?.message || 'Failed to amend loan agreement');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 backdrop-blur-sm p-4">
      <div className="bg-white rounded-2xl max-w-md w-full p-6 shadow-xl border border-slate-200">
        <div className="flex items-center justify-between pb-4 border-b border-slate-100 mb-4">
          <div>
            <h2 className="text-base font-bold text-slate-900">Amend Loan Agreement</h2>
            <p className="text-xs text-slate-500 mt-0.5">Increments version to v{loan.version + 1} and logs audit record</p>
          </div>
          <button onClick={onClose} className="text-slate-400 hover:text-slate-600 p-1">
            <X className="w-5 h-5" />
          </button>
        </div>

        {error && (
          <div className="mb-4 p-3 bg-rose-50 border border-rose-200 rounded-lg flex items-start text-rose-700 text-xs">
            <AlertCircle className="w-4 h-4 mr-2 flex-shrink-0 mt-0.5" />
            <span>{error}</span>
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-xs font-semibold text-slate-700 mb-1">
              Amendment Reason *
            </label>
            <input
              type="text"
              required
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder="e.g. Mutual agreement to extend due date by 30 days"
              className="w-full px-3 py-2 bg-slate-50 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
            />
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-700 mb-1">
              Principal Amount (₹) *
            </label>
            <input
              type="number"
              step="0.01"
              required
              value={principalAmount}
              onChange={(e) => setPrincipalAmount(e.target.value)}
              className="w-full px-3 py-2 bg-slate-50 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
            />
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-xs font-semibold text-slate-700 mb-1">
                Interest Structure
              </label>
              <select
                value={interestType}
                onChange={(e) => setInterestType(e.target.value as 'NONE' | 'SIMPLE')}
                className="w-full px-3 py-2 bg-slate-50 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
              >
                <option value="NONE">Interest-Free</option>
                <option value="SIMPLE">Simple Interest</option>
              </select>
            </div>

            {interestType === 'SIMPLE' && (
              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">
                  Rate (% p.a.)
                </label>
                <input
                  type="number"
                  step="0.01"
                  required
                  value={annualRate}
                  onChange={(e) => setAnnualRate(e.target.value)}
                  className="w-full px-3 py-2 bg-slate-50 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
                />
              </div>
            )}
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-700 mb-1">
              New Due Date
            </label>
            <input
              type="date"
              value={dueDate}
              min={loan.start_date}
              onChange={(e) => setDueDate(e.target.value)}
              className="w-full px-3 py-2 bg-slate-50 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
            />
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-700 mb-1">
              Notes
            </label>
            <textarea
              rows={2}
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              className="w-full px-3 py-2 bg-slate-50 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
            />
          </div>

          <div className="flex justify-end gap-2 pt-3 border-t border-slate-100">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 border border-slate-200 text-xs font-semibold text-slate-700 rounded-lg hover:bg-slate-50"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={loading}
              className="px-4 py-2 bg-indigo-600 hover:bg-indigo-700 text-white text-xs font-semibold rounded-lg shadow-sm disabled:opacity-60"
            >
              {loading ? 'Amending...' : 'Submit Amendment'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
