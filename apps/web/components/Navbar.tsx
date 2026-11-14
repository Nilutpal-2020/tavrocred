'use client';

import React, { useEffect, useState } from 'react';
import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import { api } from '@/lib/api';
import { User } from '@/types/api';
import { 
  CreditCard, 
  Users, 
  FileText, 
  BookOpen, 
  LogOut, 
  PlusCircle, 
  Menu, 
  X 
} from 'lucide-react';

export function Navbar() {
  const pathname = usePathname();
  const router = useRouter();
  const [user, setUser] = useState<User | null>(null);
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  useEffect(() => {
    setUser(api.getCurrentUser());
  }, [pathname]);

  // Don't show lender navbar on the borrower portal or auth pages
  if (pathname.startsWith('/portal') || pathname === '/login' || pathname === '/register') {
    return null;
  }

  const handleLogout = async () => {
    await api.logout();
    router.push('/login');
  };

  const navItems = [
    { label: 'Dashboard', href: '/dashboard', icon: CreditCard },
    { label: 'Loans', href: '/loans', icon: FileText },
    { label: 'Borrowers', href: '/peers', icon: Users },
  ];

  return (
    <nav className="bg-white border-b border-slate-200 sticky top-0 z-40">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex justify-between h-16">
          <div className="flex items-center">
            <Link href="/dashboard" className="flex items-center gap-2">
              <div className="w-8 h-8 rounded-lg bg-indigo-600 flex items-center justify-center text-white font-bold text-lg shadow-sm">
                T
              </div>
              <span className="font-bold text-xl tracking-tight text-slate-900">
                Tavro<span className="text-indigo-600">cred</span>
              </span>
            </Link>

            <div className="hidden md:ml-8 md:flex md:space-x-1">
              {navItems.map((item) => {
                const Icon = item.icon;
                const isActive = pathname === item.href || (item.href !== '/dashboard' && pathname.startsWith(item.href));
                return (
                  <Link
                    key={item.href}
                    href={item.href}
                    className={`inline-flex items-center px-3 py-2 text-sm font-medium rounded-md transition-colors ${
                      isActive
                        ? 'bg-indigo-50 text-indigo-700'
                        : 'text-slate-600 hover:text-slate-900 hover:bg-slate-50'
                    }`}
                  >
                    <Icon className="w-4 h-4 mr-2" />
                    {item.label}
                  </Link>
                );
              })}
            </div>
          </div>

          <div className="hidden md:flex md:items-center md:space-x-4">
            <Link
              href="/loans/new"
              className="inline-flex items-center px-3.5 py-1.5 border border-transparent text-xs font-semibold rounded-md shadow-sm text-white bg-indigo-600 hover:bg-indigo-700 transition"
            >
              <PlusCircle className="w-4 h-4 mr-1.5" />
              New Loan
            </Link>

            <a
              href="http://localhost:8080/docs"
              target="_blank"
              rel="noreferrer"
              className="inline-flex items-center px-3 py-1.5 text-xs font-medium text-slate-600 hover:text-indigo-600 border border-slate-200 rounded-md hover:bg-slate-50 transition"
            >
              <BookOpen className="w-3.5 h-3.5 mr-1.5" />
              API Docs
            </a>

            {user ? (
              <div className="flex items-center gap-3 pl-2 border-l border-slate-200">
                <span className="text-xs font-medium text-slate-700">
                  {user.full_name}
                </span>
                <button
                  onClick={handleLogout}
                  className="p-1.5 text-slate-400 hover:text-rose-600 rounded-md hover:bg-slate-50 transition"
                  title="Logout"
                >
                  <LogOut className="w-4 h-4" />
                </button>
              </div>
            ) : (
              <Link
                href="/login"
                className="text-sm font-medium text-indigo-600 hover:text-indigo-500"
              >
                Sign In
              </Link>
            )}
          </div>

          {/* Mobile menu button */}
          <div className="flex items-center md:hidden">
            <button
              onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
              className="p-2 rounded-md text-slate-400 hover:text-slate-500 hover:bg-slate-100"
            >
              {mobileMenuOpen ? <X className="w-6 h-6" /> : <Menu className="w-6 h-6" />}
            </button>
          </div>
        </div>
      </div>

      {/* Mobile menu */}
      {mobileMenuOpen && (
        <div className="md:hidden border-t border-slate-200 bg-white px-2 pt-2 pb-3 space-y-1">
          {navItems.map((item) => {
            const Icon = item.icon;
            const isActive = pathname === item.href;
            return (
              <Link
                key={item.href}
                href={item.href}
                onClick={() => setMobileMenuOpen(false)}
                className={`flex items-center px-3 py-2 rounded-md text-base font-medium ${
                  isActive
                    ? 'bg-indigo-50 text-indigo-700'
                    : 'text-slate-600 hover:bg-slate-50'
                }`}
              >
                <Icon className="w-5 h-5 mr-3" />
                {item.label}
              </Link>
            );
          })}
          <div className="pt-2 border-t border-slate-100 flex flex-col gap-2">
            <Link
              href="/loans/new"
              onClick={() => setMobileMenuOpen(false)}
              className="flex items-center justify-center px-4 py-2 text-sm font-semibold rounded-md text-white bg-indigo-600"
            >
              <PlusCircle className="w-4 h-4 mr-2" />
              New Loan
            </Link>
            {user && (
              <button
                onClick={handleLogout}
                className="flex items-center justify-center px-4 py-2 text-sm font-medium text-rose-600 hover:bg-rose-50 rounded-md"
              >
                <LogOut className="w-4 h-4 mr-2" />
                Logout ({user.full_name})
              </button>
            )}
          </div>
        </div>
      )}
    </nav>
  );
}
