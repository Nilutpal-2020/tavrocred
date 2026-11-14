'use client';

import React, { useState } from 'react';
import { api } from '@/lib/api';
import { Loan, BorrowerLinkResponse } from '@/types/api';
import { X, Copy, Check, Link as LinkIcon, ShieldCheck, AlertCircle } from 'lucide-react';

interface BorrowerLinkModalProps {
  loan: Loan;
  isOpen: boolean;
  onClose: () => void;
}

export function BorrowerLinkModal({ loan, isOpen, onClose }: BorrowerLinkModalProps) {
  const [expiryHours, setExpiryHours] = useState(72);
  const [linkData, setLinkData] = useState<BorrowerLinkResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [revoking, setRevoking] = useState(false);
  const [copied, setCopied] = useState(false);
  const [error, setError] = useState<string | null>(null);

  if (!isOpen) return null;

  const handleGenerate = async () => {
    setError(null);
    setLoading(true);
    try {
      const res = await api.generateBorrowerLink(loan.id, expiryHours);
      setLinkData(res);
    } catch (err: any) {
      setError(err?.message || 'Failed to generate link');
    } finally {
      setLoading(false);
    }
  };

  const handleRevoke = async () => {
    if (!confirm('Are you sure you want to revoke all active borrower access links for this loan?')) return;
    setError(null);
    setRevoking(true);
    try {
      await api.revokeBorrowerLink(loan.id);
      setLinkData(null);
      alert('All active borrower portal links for this loan have been revoked.');
    } catch (err: any) {
      setError(err?.message || 'Failed to revoke links');
    } finally {
      setRevoking(false);
    }
  };

  const handleCopy = () => {
    if (!linkData) return;
    navigator.clipboard.writeText(linkData.url);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 backdrop-blur-sm p-4">
      <div className="bg-white rounded-2xl max-w-lg w-full p-6 shadow-xl border border-slate-200">
        <div className="flex items-center justify-between pb-4 border-b border-slate-100 mb-4">
          <div className="flex items-center gap-2">
            <div className="p-2 bg-indigo-50 text-indigo-600 rounded-lg">
              <LinkIcon className="w-5 h-5" />
            </div>
            <div>
              <h2 className="text-base font-bold text-slate-900">Borrower Portal Link</h2>
              <p className="text-xs text-slate-500">Secure link for {loan.peer_name || 'borrower'} (no account needed)</p>
            </div>
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

        <div className="space-y-4">
          <div className="p-3.5 bg-slate-50 border border-slate-200 rounded-xl text-xs text-slate-600 space-y-1.5">
            <div className="flex items-center gap-1.5 font-semibold text-slate-800">
              <ShieldCheck className="w-4 h-4 text-emerald-600" />
              Cryptographically Secure Link
            </div>
            <p>
              Generates a single-purpose, 32-byte cryptographic token. The borrower can review loan terms, live balance, acknowledge agreement, dispute details, and report payments.
            </p>
          </div>

          {!linkData ? (
            <div className="space-y-4 pt-2">
              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1.5">
                  Link Expiry Duration
                </label>
                <select
                  value={expiryHours}
                  onChange={(e) => setExpiryHours(Number(e.target.value))}
                  className="w-full px-3 py-2 bg-slate-50 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
                >
                  <option value={24}>24 Hours (1 day)</option>
                  <option value={72}>72 Hours (3 days - Recommended)</option>
                  <option value={168}>7 Days (1 week)</option>
                  <option value={720}>30 Days (1 month)</option>
                </select>
              </div>

              <div className="flex justify-between items-center pt-2">
                <button
                  type="button"
                  onClick={handleRevoke}
                  disabled={revoking}
                  className="text-xs font-semibold text-rose-600 hover:text-rose-700"
                >
                  {revoking ? 'Revoking...' : 'Revoke Existing Links'}
                </button>

                <button
                  type="button"
                  onClick={handleGenerate}
                  disabled={loading}
                  className="px-4 py-2 bg-indigo-600 hover:bg-indigo-700 text-white text-xs font-semibold rounded-lg shadow-sm disabled:opacity-60"
                >
                  {loading ? 'Generating...' : 'Generate New Secure Link'}
                </button>
              </div>
            </div>
          ) : (
            <div className="space-y-4 pt-2">
              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">
                  Shareable Borrower Link
                </label>
                <div className="flex items-center gap-2">
                  <input
                    type="text"
                    readOnly
                    value={linkData.url}
                    className="flex-1 px-3 py-2 bg-slate-50 border border-slate-200 rounded-lg text-xs font-mono text-slate-700 selection:bg-indigo-500 selection:text-white"
                  />
                  <button
                    onClick={handleCopy}
                    className="px-3 py-2 bg-indigo-600 hover:bg-indigo-700 text-white text-xs font-semibold rounded-lg flex items-center gap-1.5 transition"
                  >
                    {copied ? (
                      <>
                        <Check className="w-4 h-4 text-emerald-300" />
                        Copied!
                      </>
                    ) : (
                      <>
                        <Copy className="w-4 h-4" />
                        Copy
                      </>
                    )}
                  </button>
                </div>
                <p className="text-[11px] text-slate-400 mt-1.5">
                  Expires on {new Date(linkData.expires_at).toLocaleString()}
                </p>
              </div>

              <div className="flex justify-between items-center pt-3 border-t border-slate-100">
                <button
                  type="button"
                  onClick={handleRevoke}
                  disabled={revoking}
                  className="text-xs font-semibold text-rose-600 hover:text-rose-700"
                >
                  Revoke Link
                </button>

                <button
                  type="button"
                  onClick={onClose}
                  className="px-4 py-2 border border-slate-200 text-xs font-semibold text-slate-700 rounded-lg hover:bg-slate-50"
                >
                  Done
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
