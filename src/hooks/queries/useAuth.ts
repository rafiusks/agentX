import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '../../lib/api-client';
import { useNavigate } from 'react-router-dom';

// Types
export interface User {
  id: string;
  email: string;
  username: string;
  fullName?: string;
  avatarUrl?: string;
  emailVerified: boolean;
  role: 'user' | 'admin' | 'premium';
  createdAt: string;
}

interface LoginResponse {
  user: User;
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

interface LoginRequest {
  email: string;
  password: string;
  device_name?: string;
}

interface SignupRequest {
  email: string;
  password: string;
  full_name?: string;
}

// Query keys
export const authKeys = {
  all: ['auth'] as const,
  user: () => [...authKeys.all, 'user'] as const,
  session: () => [...authKeys.all, 'session'] as const,
};

/**
 * Hook to get current user
 */
export const useCurrentUser = () => {
  return useQuery({
    queryKey: authKeys.user(),
    queryFn: async () => {
      const token = localStorage.getItem('access_token');
      if (!token) {
        return null;
      }
      return apiClient.get<User>('/auth/me');
    },
    staleTime: 5 * 60 * 1000, // Consider fresh for 5 minutes
    retry: false, // Don't retry auth failures
  });
};

/**
 * Hook for login mutation
 */
export const useLogin = () => {
  const queryClient = useQueryClient();
  const navigate = useNavigate();

  return useMutation({
    mutationFn: async (credentials: LoginRequest) => {
      return apiClient.post<LoginResponse>('/auth/login', credentials);
    },
    onSuccess: (data) => {
      // Store tokens
      localStorage.setItem('access_token', data.access_token);
      localStorage.setItem('refresh_token', data.refresh_token);
      
      // Update query cache
      queryClient.setQueryData(authKeys.user(), data.user);
      
      // Navigate to home
      navigate('/');
    },
    onError: (error) => {
      console.error('Login failed:', error);
    },
  });
};

/**
 * Hook for signup mutation
 */
export const useSignup = () => {
  const queryClient = useQueryClient();
  const navigate = useNavigate();

  return useMutation({
    mutationFn: async (data: SignupRequest) => {
      return apiClient.post<LoginResponse>('/auth/signup', data);
    },
    onSuccess: (data) => {
      // Store tokens (auto-login after signup)
      localStorage.setItem('access_token', data.access_token);
      localStorage.setItem('refresh_token', data.refresh_token);
      
      // Update query cache
      queryClient.setQueryData(authKeys.user(), data.user);
      
      // Navigate to home
      navigate('/');
    },
    onError: (error) => {
      console.error('Signup failed:', error);
    },
  });
};

/**
 * Hook for logout mutation
 */
export const useLogout = () => {
  const queryClient = useQueryClient();
  const navigate = useNavigate();

  return useMutation({
    mutationFn: async () => {
      return apiClient.post('/auth/logout');
    },
    onSuccess: () => {
      // Clear tokens
      localStorage.removeItem('access_token');
      localStorage.removeItem('refresh_token');
      
      // Clear all cached data
      queryClient.clear();
      
      // Navigate to login
      navigate('/login');
    },
    onError: (error) => {
      console.error('Logout failed:', error);
      // Even if logout fails, clear local data
      localStorage.removeItem('access_token');
      localStorage.removeItem('refresh_token');
      queryClient.clear();
      navigate('/login');
    },
  });
};

/**
 * Hook for refreshing auth token
 */
export const useRefreshToken = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async () => {
      const refreshToken = localStorage.getItem('refresh_token');
      if (!refreshToken) {
        throw new Error('No refresh token available');
      }
      
      return apiClient.post<{
        access_token: string;
        refresh_token: string;
        expires_in: number;
      }>('/auth/refresh', { refresh_token: refreshToken });
    },
    onSuccess: (data) => {
      // Update tokens
      localStorage.setItem('access_token', data.access_token);
      localStorage.setItem('refresh_token', data.refresh_token);
      
      // Refetch user data
      queryClient.invalidateQueries({ queryKey: authKeys.user() });
    },
    onError: () => {
      // Clear tokens and redirect to login
      localStorage.removeItem('access_token');
      localStorage.removeItem('refresh_token');
      queryClient.clear();
      window.location.href = '/login';
    },
  });
};

/**
 * Hook for updating user profile
 */
export const useUpdateProfile = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: { full_name: string; avatar_url?: string }) => {
      return apiClient.put<User>('/auth/profile', data);
    },
    onSuccess: (data) => {
      // Update cached user data
      queryClient.setQueryData(authKeys.user(), data);
    },
  });
};

/**
 * Hook for changing password
 */
export const useChangePassword = () => {
  return useMutation({
    mutationFn: async (data: { current_password: string; new_password: string }) => {
      return apiClient.put('/auth/password', data);
    },
  });
};

/**
 * Hook to check if user is authenticated
 */
export const useIsAuthenticated = () => {
  const { data: user, isLoading } = useCurrentUser();
  return {
    isAuthenticated: !!user,
    isLoading,
    user,
  };
};