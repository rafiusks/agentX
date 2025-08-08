import React, { useState } from 'react';
import { Link } from 'react-router-dom';
import { useLogin } from '../../hooks/queries/useAuth';
import { Button } from '../ui/button';
import { Input } from '../ui/input';
import { Label } from '../ui/label';
import { Card } from '../ui/card';
import { AlertCircle, Loader2, Eye, EyeOff } from 'lucide-react';

export const Login: React.FC = () => {
  const loginMutation = useLogin();
  const [error, setError] = useState<string | null>(null);
  const [showPassword, setShowPassword] = useState(false);
  
  const [formData, setFormData] = useState({
    email: '',
    password: '',
  });

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setFormData({
      ...formData,
      [e.target.name]: e.target.value,
    });
    // Clear error when user types
    if (error) setError(null);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    loginMutation.mutate(
      {
        email: formData.email,
        password: formData.password,
      },
      {
        onError: (err: any) => {
          setError(err.message || 'Login failed. Please check your credentials.');
        },
      }
    );
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-background-primary py-12 px-4">
      <Card className="w-full max-w-md p-8 space-y-6 bg-background-secondary border-border-subtle">
        <div>
          <h2 className="text-center text-3xl font-semibold text-foreground-primary">
            Sign in to AgentX
          </h2>
          <p className="mt-2 text-center text-sm text-foreground-secondary">
            Or{' '}
            <Link
              to="/signup"
              className="font-medium text-accent-blue hover:text-accent-blue/80 transition-colors"
            >
              create a new account
            </Link>
          </p>
        </div>
        
        <form className="mt-8 space-y-6" onSubmit={handleSubmit}>
          {error && (
            <div className="bg-accent-red/10 border border-accent-red/30 rounded-lg p-4">
              <div className="flex">
                <div className="flex-shrink-0">
                  <AlertCircle className="h-5 w-5 text-accent-red" />
                </div>
                <div className="ml-3">
                  <p className="text-sm text-accent-red">
                    {error}
                  </p>
                </div>
              </div>
            </div>
          )}
          
          <div className="space-y-4">
            <div>
              <Label htmlFor="email">Email address</Label>
              <Input
                id="email"
                name="email"
                type="email"
                autoComplete="email"
                required
                value={formData.email}
                onChange={handleChange}
                className="mt-1"
                placeholder="you@example.com"
              />
            </div>
            
            <div>
              <Label htmlFor="password">Password</Label>
              <div className="relative mt-1">
                <Input
                  id="password"
                  name="password"
                  type={showPassword ? "text" : "password"}
                  autoComplete="current-password"
                  required
                  value={formData.password}
                  onChange={handleChange}
                  className="pr-10"
                  placeholder="••••••••"
                />
                <button
                  type="button"
                  className="absolute inset-y-0 right-0 flex items-center pr-3 text-foreground-tertiary hover:text-foreground-secondary transition-colors"
                  onClick={() => setShowPassword(!showPassword)}
                >
                  {showPassword ? (
                    <EyeOff className="h-4 w-4" />
                  ) : (
                    <Eye className="h-4 w-4" />
                  )}
                </button>
              </div>
            </div>
          </div>

          <div className="flex items-center justify-between">
            <div className="flex items-center">
              <input
                id="remember-me"
                name="remember-me"
                type="checkbox"
                className="h-4 w-4 text-accent-blue focus:ring-accent-blue/50 border-border-default rounded accent-accent-blue"
              />
              <label htmlFor="remember-me" className="ml-2 block text-sm text-foreground-primary">
                Remember me
              </label>
            </div>

            <div className="text-sm">
              <a href="#" className="font-medium text-accent-blue hover:text-accent-blue/80 transition-colors">
                Forgot your password?
              </a>
            </div>
          </div>

          <Button
            type="submit"
            className="w-full"
            disabled={loginMutation.isPending}
          >
            {loginMutation.isPending ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Signing in...
              </>
            ) : (
              'Sign in'
            )}
          </Button>
        </form>
      </Card>
    </div>
  );
};