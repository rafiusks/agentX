import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useCurrentUser, useLogout } from '../../hooks/queries/useAuth';
import { Button } from '../ui/button';
import {
  User,
  Settings,
  Key,
  LogOut,
  ChevronDown,
  Shield,
  CreditCard,
} from 'lucide-react';

export const UserMenu: React.FC = () => {
  const navigate = useNavigate();
  const { data: user } = useCurrentUser();
  const logoutMutation = useLogout();
  const [isOpen, setIsOpen] = useState(false);

  if (!user) return null;

  const handleLogout = () => {
    logoutMutation.mutate();
  };

  const menuItems = [
    {
      icon: User,
      label: 'Profile',
      onClick: () => navigate('/profile'),
    },
    {
      icon: Settings,
      label: 'Settings',
      onClick: () => navigate('/settings'),
    },
    {
      icon: Key,
      label: 'API Keys',
      onClick: () => navigate('/api-keys'),
    },
    ...(user.role === 'admin' ? [{
      icon: Shield,
      label: 'Admin Panel',
      onClick: () => navigate('/admin'),
    }] : []),
    ...(user.role === 'premium' ? [{
      icon: CreditCard,
      label: 'Subscription',
      onClick: () => navigate('/subscription'),
    }] : []),
  ];

  return (
    <div className="relative">
      <Button
        variant="ghost"
        className="flex items-center space-x-2"
        onClick={() => setIsOpen(!isOpen)}
      >
        <div className="flex items-center space-x-2">
          {user.avatarUrl ? (
            <img
              src={user.avatarUrl}
              alt={user.username}
              className="h-8 w-8 rounded-full"
            />
          ) : (
            <div className="h-8 w-8 rounded-full bg-gray-300 dark:bg-gray-700 flex items-center justify-center">
              <User className="h-5 w-5 text-gray-600 dark:text-gray-400" />
            </div>
          )}
          <span className="text-sm font-medium text-gray-700 dark:text-gray-200">
            {user.username}
          </span>
          <ChevronDown className="h-4 w-4 text-gray-500" />
        </div>
      </Button>

      {isOpen && (
        <>
          <div
            className="fixed inset-0 z-10"
            onClick={() => setIsOpen(false)}
          />
          <div className="absolute right-0 mt-2 w-48 bg-white dark:bg-gray-800 rounded-md shadow-lg py-1 z-20 border border-gray-200 dark:border-gray-700">
            <div className="px-4 py-2 border-b border-gray-200 dark:border-gray-700">
              <p className="text-sm font-medium text-gray-900 dark:text-white">
                {user.fullName || user.username}
              </p>
              <p className="text-xs text-gray-500 dark:text-gray-400">
                {user.email}
              </p>
              {user.role !== 'user' && (
                <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium mt-1 ${
                  user.role === 'admin' 
                    ? 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200'
                    : 'bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-200'
                }`}>
                  {user.role}
                </span>
              )}
            </div>

            {menuItems.map((item, index) => (
              <button
                key={index}
                onClick={() => {
                  item.onClick();
                  setIsOpen(false);
                }}
                className="w-full flex items-center px-4 py-2 text-sm text-gray-700 dark:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700"
              >
                <item.icon className="h-4 w-4 mr-2" />
                {item.label}
              </button>
            ))}

            <div className="border-t border-gray-200 dark:border-gray-700">
              <button
                onClick={handleLogout}
                className="w-full flex items-center px-4 py-2 text-sm text-red-600 dark:text-red-400 hover:bg-gray-100 dark:hover:bg-gray-700"
              >
                <LogOut className="h-4 w-4 mr-2" />
                Sign out
              </button>
            </div>
          </div>
        </>
      )}
    </div>
  );
};