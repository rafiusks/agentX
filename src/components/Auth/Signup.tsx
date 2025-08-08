import React, { useState } from 'react';
import { Link } from 'react-router-dom';
import { useSignup } from '../../hooks/queries/useAuth';
import { Button } from '../ui/button';
import { Input } from '../ui/input';
import { Label } from '../ui/label';
import { Card } from '../ui/card';
import { AlertCircle, Loader2, CheckCircle, Eye, EyeOff } from 'lucide-react';

export const Signup: React.FC = () => {
  const signupMutation = useSignup();
  const [error, setError] = useState<string | null>(null);
  const [showPassword, setShowPassword] = useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = useState(false);
  
  const [formData, setFormData] = useState({
    email: '',
    password: '',
    confirmPassword: '',
    fullName: '',
  });

  const [validationErrors, setValidationErrors] = useState<Record<string, string>>({});

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setFormData({
      ...formData,
      [e.target.name]: e.target.value,
    });
    // Clear validation error for this field
    if (validationErrors[e.target.name]) {
      setValidationErrors({
        ...validationErrors,
        [e.target.name]: '',
      });
    }
    // Clear general error when user types
    if (error) setError(null);
  };

  const validateForm = (): boolean => {
    const errors: Record<string, string> = {};

    // Email validation
    if (!formData.email) {
      errors.email = 'Email is required';
    } else if (!/\S+@\S+\.\S+/.test(formData.email)) {
      errors.email = 'Email is invalid';
    }

    // Password validation - match backend requirements
    if (!formData.password) {
      errors.password = 'Password is required';
    } else if (formData.password.length < 8) {
      errors.password = 'Password must be at least 8 characters';
    } else if (!/[a-z]/.test(formData.password)) {
      errors.password = 'Password must contain at least one lowercase letter';
    } else if (!/[A-Z]/.test(formData.password)) {
      errors.password = 'Password must contain at least one uppercase letter';
    } else if (!/[0-9]/.test(formData.password)) {
      errors.password = 'Password must contain at least one number';
    }

    // Confirm password validation
    if (!formData.confirmPassword) {
      errors.confirmPassword = 'Please confirm your password';
    } else if (formData.password !== formData.confirmPassword) {
      errors.confirmPassword = 'Passwords do not match';
    }

    setValidationErrors(errors);
    return Object.keys(errors).length === 0;
  };

  const validatePassword = (password: string): { isValid: boolean; errors: string[] } => {
    const errors: string[] = [];
    
    if (password.length < 8) {
      errors.push('Password must be at least 8 characters');
    }
    if (!/[a-z]/.test(password)) {
      errors.push('Password must contain at least one lowercase letter');
    }
    if (!/[A-Z]/.test(password)) {
      errors.push('Password must contain at least one uppercase letter');
    }
    if (!/[0-9]/.test(password)) {
      errors.push('Password must contain at least one number');
    }
    
    return {
      isValid: errors.length === 0,
      errors
    };
  };

  const getPasswordStrength = (password: string): { 
    strength: number; 
    label: string; 
    color: string; 
    isValid: boolean;
    requirements: string[];
  } => {
    const validation = validatePassword(password);
    
    if (!validation.isValid) {
      return {
        strength: 0,
        label: 'Invalid',
        color: 'bg-accent-red',
        isValid: false,
        requirements: validation.errors
      };
    }
    
    let strength = 0;
    if (password.length >= 8) strength++;
    if (password.length >= 12) strength++;
    if (/[a-z]/.test(password) && /[A-Z]/.test(password)) strength++;
    if (/[0-9]/.test(password)) strength++;
    if (/[^a-zA-Z0-9]/.test(password)) strength++;

    if (strength <= 2) return { strength: 1, label: 'Weak', color: 'bg-accent-yellow', isValid: true, requirements: [] };
    if (strength === 3) return { strength: 2, label: 'Fair', color: 'bg-accent-yellow', isValid: true, requirements: [] };
    if (strength === 4) return { strength: 3, label: 'Good', color: 'bg-accent-green', isValid: true, requirements: [] };
    return { strength: 4, label: 'Strong', color: 'bg-accent-green', isValid: true, requirements: [] };
  };

  const passwordStrength = formData.password ? getPasswordStrength(formData.password) : null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!validateForm()) {
      return;
    }

    signupMutation.mutate(
      {
        email: formData.email,
        password: formData.password,
        full_name: formData.fullName || undefined,
      },
      {
        onError: (err: any) => {
          setError(err.message || 'Signup failed. Please try again.');
        },
      }
    );
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-background-primary py-12 px-4">
      <Card className="w-full max-w-md p-8 space-y-6 bg-background-secondary border-border-subtle">
        <div>
          <h2 className="text-center text-3xl font-semibold text-foreground-primary">
            Create your account
          </h2>
          <p className="mt-2 text-center text-sm text-foreground-secondary">
            Or{' '}
            <Link
              to="/login"
              className="font-medium text-accent-blue hover:text-accent-blue/80 transition-colors"
            >
              sign in to existing account
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
              <Label htmlFor="email">Email address *</Label>
              <Input
                id="email"
                name="email"
                type="email"
                autoComplete="email"
                required
                value={formData.email}
                onChange={handleChange}
                className={`mt-1 ${validationErrors.email ? 'border-accent-red' : ''}`}
                placeholder="you@example.com"
              />
              {validationErrors.email && (
                <p className="mt-1 text-sm text-accent-red">{validationErrors.email}</p>
              )}
            </div>


            <div>
              <Label htmlFor="fullName">Full Name (optional)</Label>
              <Input
                id="fullName"
                name="fullName"
                type="text"
                autoComplete="name"
                value={formData.fullName}
                onChange={handleChange}
                className="mt-1"
                placeholder="John Doe"
              />
            </div>
            
            <div>
              <Label htmlFor="password">Password *</Label>
              <div className="relative mt-1">
                <Input
                  id="password"
                  name="password"
                  type={showPassword ? "text" : "password"}
                  autoComplete="new-password"
                  required
                  value={formData.password}
                  onChange={handleChange}
                  className={`pr-10 ${validationErrors.password ? 'border-accent-red' : ''}`}
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
              {validationErrors.password && (
                <p className="mt-1 text-sm text-accent-red">{validationErrors.password}</p>
              )}
              {passwordStrength && formData.password && (
                <div className="mt-2">
                  <div className="flex items-center justify-between mb-1">
                    <span className="text-xs text-foreground-secondary">Password strength</span>
                    <span className={`text-xs font-medium ${
                      passwordStrength.label === 'Invalid' ? 'text-accent-red' :
                      passwordStrength.label === 'Weak' ? 'text-accent-yellow' :
                      passwordStrength.label === 'Fair' ? 'text-accent-yellow' :
                      passwordStrength.label === 'Good' ? 'text-accent-green' :
                      'text-accent-green'
                    }`}>
                      {passwordStrength.label}
                    </span>
                  </div>
                  <div className="w-full bg-background-tertiary rounded-full h-1.5">
                    <div
                      className={`h-1.5 rounded-full transition-all ${passwordStrength.color}`}
                      style={{ width: `${(passwordStrength.strength / 4) * 100}%` }}
                    />
                  </div>
                  {passwordStrength.requirements.length > 0 && (
                    <div className="mt-2">
                      <p className="text-xs text-accent-red mb-1">Requirements:</p>
                      <ul className="text-xs text-accent-red space-y-0.5">
                        {passwordStrength.requirements.map((req, index) => (
                          <li key={index} className="flex items-center">
                            <span className="mr-1">•</span>
                            {req}
                          </li>
                        ))}
                      </ul>
                    </div>
                  )}
                </div>
              )}
            </div>

            <div>
              <Label htmlFor="confirmPassword">Confirm Password *</Label>
              <div className="relative mt-1">
                <Input
                  id="confirmPassword"
                  name="confirmPassword"
                  type={showConfirmPassword ? "text" : "password"}
                  autoComplete="new-password"
                  required
                  value={formData.confirmPassword}
                  onChange={handleChange}
                  className={`pr-10 ${validationErrors.confirmPassword ? 'border-accent-red' : ''}`}
                  placeholder="••••••••"
                />
                <button
                  type="button"
                  className="absolute inset-y-0 right-0 flex items-center pr-3 text-foreground-tertiary hover:text-foreground-secondary transition-colors"
                  onClick={() => setShowConfirmPassword(!showConfirmPassword)}
                >
                  {showConfirmPassword ? (
                    <EyeOff className="h-4 w-4" />
                  ) : (
                    <Eye className="h-4 w-4" />
                  )}
                </button>
              </div>
              {validationErrors.confirmPassword && (
                <p className="mt-1 text-sm text-accent-red">{validationErrors.confirmPassword}</p>
              )}
              {formData.confirmPassword && formData.password === formData.confirmPassword && (
                <div className="mt-1 flex items-center text-accent-green">
                  <CheckCircle className="h-4 w-4 mr-1" />
                  <span className="text-sm">Passwords match</span>
                </div>
              )}
            </div>
          </div>

          <div className="flex items-center">
            <input
              id="terms"
              name="terms"
              type="checkbox"
              required
              className="h-4 w-4 text-accent-blue focus:ring-accent-blue/50 border-border-subtle rounded accent-accent-blue"
            />
            <label htmlFor="terms" className="ml-2 block text-sm text-foreground-primary">
              I agree to the{' '}
              <a href="#" className="text-accent-blue hover:text-accent-blue/80 transition-colors">
                Terms and Conditions
              </a>
            </label>
          </div>

          <Button
            type="submit"
            className="w-full"
            disabled={signupMutation.isPending || (formData.password && passwordStrength && !passwordStrength.isValid) || false}
          >
            {signupMutation.isPending ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Creating account...
              </>
            ) : (
              'Create account'
            )}
          </Button>
        </form>
      </Card>
    </div>
  );
};