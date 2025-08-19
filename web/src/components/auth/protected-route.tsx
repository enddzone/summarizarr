'use client';

import { useAuth } from '@/contexts/auth-context';
import { AuthForm } from './auth-form';
import { LoadingSpinner } from '@/components/ui/loading-spinner';
import { ThemeToggle } from '@/components/theme-toggle';
import { StaticDashboardBackground } from '@/components/static-dashboard-background';

interface ProtectedRouteProps {
  children: React.ReactNode;
}

export function ProtectedRoute({ children }: ProtectedRouteProps) {
  const { user, loading } = useAuth();

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <LoadingSpinner />
      </div>
    );
  }

  if (!user) {
    return (
      <div className="relative min-h-screen">
        {/* Background app content (blurred and static) */}
        <div className="absolute inset-0 blur-sm opacity-30 pointer-events-none dark:opacity-30">
          <StaticDashboardBackground />
        </div>

        {/* Dark overlay */}
        <div className="absolute inset-0 bg-background/30 dark:bg-background/40" />

        {/* Auth overlay */}
        <div className="relative z-10 min-h-screen flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
          {/* Theme toggle in top right */}
          <div className="absolute top-4 right-4">
            <ThemeToggle />
          </div>

          {/* Auth form */}
          <div className="max-w-md w-full">
            <AuthForm />
          </div>
        </div>
      </div>
    );
  }

  return <>{children}</>;
}