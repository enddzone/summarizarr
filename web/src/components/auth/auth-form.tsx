'use client';

import { useState } from 'react';
import Image from 'next/image';
import { Eye, EyeOff, Mail, Lock, UserPlus, LogIn } from 'lucide-react';
import { useAuth } from '@/contexts/auth-context';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';

export function AuthForm() {
    const [email, setEmail] = useState('');
    const [password, setPassword] = useState('');
    const [confirmPassword, setConfirmPassword] = useState('');
    const [error, setError] = useState('');
    const [isLoading, setIsLoading] = useState(false);
    const [isRegistering, setIsRegistering] = useState(false);
    const [showPassword, setShowPassword] = useState(false);
    const [showConfirmPassword, setShowConfirmPassword] = useState(false);
    const { login, register } = useAuth();

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setError('');
        setIsLoading(true);

        try {
            if (isRegistering) {
                if (password !== confirmPassword) {
                    throw new Error('Passwords do not match');
                }
                if (password.length < 8) {
                    throw new Error('Password must be at least 8 characters long');
                }
                const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
                if (!emailRegex.test(email)) {
                    throw new Error('Please enter a valid email address');
                }
                await register(email, password);
            } else {
                if (!email || !password) {
                    throw new Error('Please fill in all fields');
                }
                await login(email, password);
            }
        } catch (err) {
            // Log the original error for debugging purposes
            console.error('Authentication error:', err);
            const errorMessage = err instanceof Error ? err.message : 'An error occurred';
            if (errorMessage.includes('Registration failed')) {
                setError('Email address already exists or invalid. Please try a different email.');
            } else if (errorMessage.includes('Login failed')) {
                setError('Invalid email or password. Please check your credentials.');
            } else {
                setError(errorMessage);
            }
        } finally {
            setIsLoading(false);
        }
    };

    const toggleMode = () => {
        setIsRegistering(!isRegistering);
        setError('');
        setConfirmPassword('');
        setShowPassword(false);
        setShowConfirmPassword(false);
    };

    return (
        <Card className="w-full max-w-md mx-auto bg-background/90 backdrop-blur-md border border-border shadow-2xl dark:bg-background/80 dark:border-border/50 animate-fade-in">
            <CardHeader className="text-center space-y-2 pb-4">
                <div className="flex items-center justify-center mb-2">
                    <Image
                        src="/main.svg"
                        alt="Summarizarr Logo"
                        width={96}
                        height={96}
                        className="w-24 h-24 object-contain"
                    />
                </div>
                <CardTitle className="text-2xl -mt-2">
                    <span className="text-primary">SUMMARI</span>
                    <span className="text-orange-600">ZARR</span>
                </CardTitle>
            </CardHeader>
            <CardContent className="pt-0 px-6 pb-6">
                <form onSubmit={handleSubmit} className="space-y-4">
                    <div className="space-y-2">
                        <Label htmlFor="email" className="text-sm font-medium">Email</Label>
                        <div className="relative">
                            <Mail className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
                            <Input
                                type="email"
                                id="email"
                                value={email}
                                onChange={(e) => setEmail(e.target.value)}
                                placeholder="Enter your email"
                                className="pl-10"
                                required
                                disabled={isLoading}
                            />
                        </div>
                    </div>
                    <div className="space-y-2">
                        <Label htmlFor="password" className="text-sm font-medium">Password</Label>
                        <div className="relative">
                            <Lock className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
                            <Input
                                type={showPassword ? 'text' : 'password'}
                                id="password"
                                value={password}
                                onChange={(e) => setPassword(e.target.value)}
                                placeholder="Enter your password"
                                className="pl-10 pr-10"
                                required
                                disabled={isLoading}
                            />
                            <Button
                                type="button"
                                variant="ghost"
                                size="sm"
                                className="absolute right-1 top-1 h-7 w-7 p-0"
                                onClick={() => setShowPassword(!showPassword)}
                                disabled={isLoading}
                            >
                                {showPassword ? (
                                    <EyeOff className="h-4 w-4" />
                                ) : (
                                    <Eye className="h-4 w-4" />
                                )}
                            </Button>
                        </div>
                    </div>
                    {isRegistering && (
                        <div className="space-y-2">
                            <Label htmlFor="confirmPassword" className="text-sm font-medium">Confirm Password</Label>
                            <div className="relative">
                                <Lock className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
                                <Input
                                    type={showConfirmPassword ? 'text' : 'password'}
                                    id="confirmPassword"
                                    value={confirmPassword}
                                    onChange={(e) => setConfirmPassword(e.target.value)}
                                    placeholder="Confirm your password"
                                    className="pl-10 pr-10"
                                    required
                                    disabled={isLoading}
                                />
                                <Button
                                    type="button"
                                    variant="ghost"
                                    size="sm"
                                    className="absolute right-1 top-1 h-7 w-7 p-0"
                                    onClick={() => setShowConfirmPassword(!showConfirmPassword)}
                                    disabled={isLoading}
                                >
                                    {showConfirmPassword ? (
                                        <EyeOff className="h-4 w-4" />
                                    ) : (
                                        <Eye className="h-4 w-4" />
                                    )}
                                </Button>
                            </div>
                        </div>
                    )}
                    {error && (
                        <div className="text-sm text-destructive bg-destructive/10 p-3 rounded-md border border-destructive/20 animate-slide-up">
                            {error}
                        </div>
                    )}
                    <Button type="submit" className="w-full" disabled={isLoading}>
                        {isLoading ? (
                            <div className="flex items-center gap-2">
                                <div className="w-4 h-4 border-2 border-current border-t-transparent rounded-full animate-spin" />
                                {isRegistering ? 'Creating Account...' : 'Signing In...'}
                            </div>
                        ) : (
                            <div className="flex items-center gap-2">
                                {isRegistering ? (
                                    <UserPlus className="h-4 w-4" />
                                ) : (
                                    <LogIn className="h-4 w-4" />
                                )}
                                {isRegistering ? 'Create Account' : 'Sign In'}
                            </div>
                        )}
                    </Button>
                </form>

                <div className="mt-6 text-center">
                    <p className="text-sm text-muted-foreground">
                        {isRegistering
                            ? 'Already have an account? '
                            : "Don't have an account? "
                        }
                        <button
                            type="button"
                            onClick={toggleMode}
                            className="font-medium text-primary hover:underline transition-colors"
                            disabled={isLoading}
                        >
                            {isRegistering ? 'Sign in' : 'Sign up'}
                        </button>
                    </p>
                </div>
            </CardContent>
        </Card>
    );
}
