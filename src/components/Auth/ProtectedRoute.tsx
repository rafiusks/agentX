import React, { useEffect } from 'react';
import { Navigate, useLocation } from 'react-router-dom';
import { useIsAuthenticated, useRefreshToken } from '../../hooks/queries/useAuth';
import { Loader2 } from 'lucide-react';

interface ProtectedRouteProps {
  children: React.ReactNode;
  requiredRole?: 'user' | 'admin' | 'premium';
}

export const ProtectedRoute: React.FC<ProtectedRouteProps> = ({ 
  children, 
  requiredRole = 'user' 
}) => {
  const location = useLocation();
  const { isAuthenticated, isLoading, user } = useIsAuthenticated();
  const refreshMutation = useRefreshToken();

  useEffect(() => {
    // Try to refresh session on mount if we have a refresh token
    const refreshToken = localStorage.getItem('refresh_token');
    const accessToken = localStorage.getItem('access_token');
    if (!accessToken && refreshToken) {
      refreshMutation.mutate();
    }
  }, []);

  // Show loading state while checking auth
  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-blue-600" />
      </div>
    );
  }

  // Redirect to login if not authenticated
  if (!isAuthenticated) {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }

  // Check role requirements
  if (requiredRole && user) {
    const roleHierarchy = { user: 1, premium: 2, admin: 3 };
    const userRoleLevel = roleHierarchy[user.role] || 0;
    const requiredRoleLevel = roleHierarchy[requiredRole] || 0;

    if (userRoleLevel < requiredRoleLevel) {
      return (
        <div className="min-h-screen flex items-center justify-center">
          <div className="text-center">
            <h2 className="text-2xl font-bold text-gray-900 dark:text-white mb-2">
              Access Denied
            </h2>
            <p className="text-gray-600 dark:text-gray-400">
              You need {requiredRole} privileges to access this page.
            </p>
          </div>
        </div>
      );
    }
  }

  return <>{children}</>;
};