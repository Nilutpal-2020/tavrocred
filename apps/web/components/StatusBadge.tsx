import React from 'react';
import { LoanStatus, RepaymentStatus } from '@/types/api';

interface StatusBadgeProps {
  status: LoanStatus | RepaymentStatus | string;
  className?: string;
}

export function StatusBadge({ status, className = '' }: StatusBadgeProps) {
  let badgeClass = 'badge-draft';
  let label = status;

  switch (status) {
    case 'ACTIVE':
      badgeClass = 'badge-active';
      label = 'Active';
      break;
    case 'COMPLETED':
      badgeClass = 'badge-completed';
      label = 'Completed';
      break;
    case 'DRAFT':
      badgeClass = 'badge-draft';
      label = 'Draft';
      break;
    case 'CANCELLED':
      badgeClass = 'badge-cancelled';
      label = 'Cancelled';
      break;
    case 'DISPUTED':
      badgeClass = 'badge-disputed';
      label = 'Disputed';
      break;
    case 'CONFIRMED':
      badgeClass = 'badge-confirmed';
      label = 'Confirmed';
      break;
    case 'PENDING_VERIFICATION':
      badgeClass = 'badge-pending';
      label = 'Pending Verification';
      break;
    case 'REJECTED':
      badgeClass = 'badge-rejected';
      label = 'Rejected';
      break;
    case 'REVERSED':
      badgeClass = 'badge-reversed';
      label = 'Reversed';
      break;
    case 'ACKNOWLEDGED':
      badgeClass = 'badge-active';
      label = 'Acknowledged';
      break;
    default:
      badgeClass = 'badge-draft';
      label = status;
  }

  return (
    <span className={`badge ${badgeClass} ${className}`}>
      {label}
    </span>
  );
}
