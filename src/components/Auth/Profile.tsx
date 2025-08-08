import React, { useState } from 'react';
import { useCurrentUser, useUpdateProfile, useChangePassword } from '../../hooks/queries/useAuth';
import { Button } from '../ui/button';
import { Input } from '../ui/input';
import { Label } from '../ui/label';
import { Card } from '../ui/card';
import { 
  User, 
  Mail, 
  Calendar, 
  Shield, 
  Save, 
  Loader2,
  CheckCircle,
  AlertCircle,
  Lock
} from 'lucide-react';

export const Profile: React.FC = () => {
  const { data: user } = useCurrentUser();
  const updateProfileMutation = useUpdateProfile();
  const changePasswordMutation = useChangePassword();
  const [activeTab, setActiveTab] = useState<'profile' | 'security'>('profile');
  const [successMessage, setSuccessMessage] = useState('');
  const [error, setError] = useState<string | null>(null);
  
  // Profile form state
  const [profileData, setProfileData] = useState({
    fullName: user?.fullName || '',
    avatarUrl: user?.avatarUrl || '',
  });

  // Password form state
  const [passwordData, setPasswordData] = useState({
    currentPassword: '',
    newPassword: '',
    confirmPassword: '',
  });

  const [passwordErrors, setPasswordErrors] = useState<Record<string, string>>({});

  const handleProfileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setProfileData({
      ...profileData,
      [e.target.name]: e.target.value,
    });
    if (error) setError(null);
    setSuccessMessage('');
  };

  const handlePasswordChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setPasswordData({
      ...passwordData,
      [e.target.name]: e.target.value,
    });
    setPasswordErrors({
      ...passwordErrors,
      [e.target.name]: '',
    });
    if (error) setError(null);
    setSuccessMessage('');
  };

  const handleProfileSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    updateProfileMutation.mutate(
      {
        full_name: profileData.fullName,
        avatar_url: profileData.avatarUrl || undefined,
      },
      {
        onSuccess: () => {
          setSuccessMessage('Profile updated successfully!');
          setTimeout(() => setSuccessMessage(''), 3000);
        },
        onError: (err: any) => {
          setError(err.message || 'Profile update failed');
        },
      }
    );
  };

  const validatePasswordForm = (): boolean => {
    const errors: Record<string, string> = {};

    if (!passwordData.currentPassword) {
      errors.currentPassword = 'Current password is required';
    }

    if (!passwordData.newPassword) {
      errors.newPassword = 'New password is required';
    } else if (passwordData.newPassword.length < 8) {
      errors.newPassword = 'Password must be at least 8 characters';
    }

    if (!passwordData.confirmPassword) {
      errors.confirmPassword = 'Please confirm your new password';
    } else if (passwordData.newPassword !== passwordData.confirmPassword) {
      errors.confirmPassword = 'Passwords do not match';
    }

    setPasswordErrors(errors);
    return Object.keys(errors).length === 0;
  };

  const handlePasswordSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!validatePasswordForm()) {
      return;
    }

    changePasswordMutation.mutate(
      {
        current_password: passwordData.currentPassword,
        new_password: passwordData.newPassword,
      },
      {
        onSuccess: () => {
          setPasswordData({
            currentPassword: '',
            newPassword: '',
            confirmPassword: '',
          });
          setSuccessMessage('Password changed successfully!');
          setTimeout(() => setSuccessMessage(''), 3000);
        },
        onError: (err: any) => {
          setError(err.message || 'Password change failed');
        },
      }
    );
  };

  if (!user) return null;

  return (
    <div className="max-w-4xl mx-auto p-6">
      <h1 className="text-3xl font-bold text-foreground-primary mb-8">
        Account Settings
      </h1>

      <div className="flex space-x-1 mb-6">
        <Button
          variant={activeTab === 'profile' ? 'default' : 'ghost'}
          onClick={() => setActiveTab('profile')}
          className="flex items-center"
        >
          <User className="h-4 w-4 mr-2" />
          Profile
        </Button>
        <Button
          variant={activeTab === 'security' ? 'default' : 'ghost'}
          onClick={() => setActiveTab('security')}
          className="flex items-center"
        >
          <Lock className="h-4 w-4 mr-2" />
          Security
        </Button>
      </div>

      {successMessage && (
        <div className="mb-4 bg-accent-green/10 border border-accent-green/30 rounded-md p-4">
          <div className="flex">
            <CheckCircle className="h-5 w-5 text-accent-green mr-2" />
            <p className="text-sm text-accent-green">
              {successMessage}
            </p>
          </div>
        </div>
      )}

      {error && (
        <div className="mb-4 bg-accent-red/10 border border-accent-red/30 rounded-md p-4">
          <div className="flex">
            <AlertCircle className="h-5 w-5 text-accent-red mr-2" />
            <p className="text-sm text-accent-red">
              {error}
            </p>
          </div>
        </div>
      )}

      {activeTab === 'profile' && (
        <Card className="p-6">
          <h2 className="text-xl font-semibold mb-6">Profile Information</h2>
          
          <div className="space-y-4 mb-6">
            <div className="flex items-center space-x-3 p-3 bg-background-tertiary rounded-lg">
              <Mail className="h-5 w-5 text-foreground-tertiary" />
              <div>
                <p className="text-sm text-foreground-tertiary">Email</p>
                <p className="font-medium">{user.email}</p>
              </div>
            </div>

            <div className="flex items-center space-x-3 p-3 bg-background-tertiary rounded-lg">
              <User className="h-5 w-5 text-foreground-tertiary" />
              <div>
                <p className="text-sm text-foreground-tertiary">Username</p>
                <p className="font-medium">{user.username}</p>
              </div>
            </div>

            <div className="flex items-center space-x-3 p-3 bg-background-tertiary rounded-lg">
              <Shield className="h-5 w-5 text-foreground-tertiary" />
              <div>
                <p className="text-sm text-foreground-tertiary">Role</p>
                <p className="font-medium capitalize">{user.role}</p>
              </div>
            </div>

            <div className="flex items-center space-x-3 p-3 bg-background-tertiary rounded-lg">
              <Calendar className="h-5 w-5 text-foreground-tertiary" />
              <div>
                <p className="text-sm text-foreground-tertiary">Member Since</p>
                <p className="font-medium">
                  {new Date(user.createdAt).toLocaleDateString('en-US', {
                    year: 'numeric',
                    month: 'long',
                    day: 'numeric',
                  })}
                </p>
              </div>
            </div>
          </div>

          <form onSubmit={handleProfileSubmit} className="space-y-4">
            <div>
              <Label htmlFor="fullName">Full Name</Label>
              <Input
                id="fullName"
                name="fullName"
                type="text"
                value={profileData.fullName}
                onChange={handleProfileChange}
                className="mt-1"
                placeholder="John Doe"
              />
            </div>

            <div>
              <Label htmlFor="avatarUrl">Avatar URL</Label>
              <Input
                id="avatarUrl"
                name="avatarUrl"
                type="url"
                value={profileData.avatarUrl}
                onChange={handleProfileChange}
                className="mt-1"
                placeholder="https://example.com/avatar.jpg"
              />
              <p className="text-xs text-foreground-tertiary mt-1">
                Enter a URL to your profile picture
              </p>
            </div>

            <Button
              type="submit"
              disabled={updateProfileMutation.isPending}
              className="flex items-center"
            >
              {updateProfileMutation.isPending ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Saving...
                </>
              ) : (
                <>
                  <Save className="mr-2 h-4 w-4" />
                  Save Changes
                </>
              )}
            </Button>
          </form>
        </Card>
      )}

      {activeTab === 'security' && (
        <Card className="p-6">
          <h2 className="text-xl font-semibold mb-6">Change Password</h2>
          
          <form onSubmit={handlePasswordSubmit} className="space-y-4">
            <div>
              <Label htmlFor="currentPassword">Current Password</Label>
              <Input
                id="currentPassword"
                name="currentPassword"
                type="password"
                value={passwordData.currentPassword}
                onChange={handlePasswordChange}
                className={`mt-1 ${passwordErrors.currentPassword ? 'border-accent-red' : ''}`}
              />
              {passwordErrors.currentPassword && (
                <p className="mt-1 text-sm text-accent-red">{passwordErrors.currentPassword}</p>
              )}
            </div>

            <div>
              <Label htmlFor="newPassword">New Password</Label>
              <Input
                id="newPassword"
                name="newPassword"
                type="password"
                value={passwordData.newPassword}
                onChange={handlePasswordChange}
                className={`mt-1 ${passwordErrors.newPassword ? 'border-accent-red' : ''}`}
              />
              {passwordErrors.newPassword && (
                <p className="mt-1 text-sm text-accent-red">{passwordErrors.newPassword}</p>
              )}
            </div>

            <div>
              <Label htmlFor="confirmPassword">Confirm New Password</Label>
              <Input
                id="confirmPassword"
                name="confirmPassword"
                type="password"
                value={passwordData.confirmPassword}
                onChange={handlePasswordChange}
                className={`mt-1 ${passwordErrors.confirmPassword ? 'border-accent-red' : ''}`}
              />
              {passwordErrors.confirmPassword && (
                <p className="mt-1 text-sm text-accent-red">{passwordErrors.confirmPassword}</p>
              )}
            </div>

            <Button
              type="submit"
              disabled={changePasswordMutation.isPending}
              className="flex items-center"
            >
              {changePasswordMutation.isPending ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Changing Password...
                </>
              ) : (
                <>
                  <Lock className="mr-2 h-4 w-4" />
                  Change Password
                </>
              )}
            </Button>
          </form>
        </Card>
      )}
    </div>
  );
};