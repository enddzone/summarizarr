'use client';

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { CalendarDays, MessageSquare, Users, TrendingUp } from 'lucide-react';

export function StaticDashboardBackground() {
    return (
        <div className="min-h-screen bg-background">
            {/* Header */}
            <header className="sticky top-0 z-50 w-full border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
                <div className="container mx-auto px-4 flex h-16 items-center justify-between">
                    <div className="flex items-center space-x-4">
                        <h1 className="text-2xl font-bold tracking-tight">
                            <span className="text-primary">SUMMARI</span>
                            <span className="text-orange-600">ZARR</span>
                        </h1>
                        <Badge variant="secondary" className="flex items-center gap-1.5">
                            <div className="w-2 h-2 bg-gray-400 rounded-full" />
                            Signal Not Connected
                        </Badge>
                    </div>
                </div>
            </header>

            {/* Main content */}
            <main className="container mx-auto px-4 py-8">
                {/* Filter Panel */}
                <div className="mb-8 p-4 bg-card rounded-lg border">
                    <div className="flex flex-wrap gap-4 items-center">
                        <div className="flex items-center gap-2">
                            <CalendarDays className="h-4 w-4 text-muted-foreground" />
                            <span className="text-sm text-muted-foreground">Today</span>
                        </div>
                        <div className="flex items-center gap-2">
                            <Users className="h-4 w-4 text-muted-foreground" />
                            <span className="text-sm text-muted-foreground">All Groups</span>
                        </div>
                        <div className="flex items-center gap-2">
                            <MessageSquare className="h-4 w-4 text-muted-foreground" />
                            <span className="text-sm text-muted-foreground">Search messages...</span>
                        </div>
                    </div>
                </div>

                {/* Sample Summary Cards */}
                <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
                    <Card>
                        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                            <CardTitle className="text-sm font-medium">Team Updates</CardTitle>
                            <TrendingUp className="h-4 w-4 text-muted-foreground" />
                        </CardHeader>
                        <CardContent>
                            <div className="text-2xl font-bold">12</div>
                            <p className="text-xs text-muted-foreground">
                                messages today
                            </p>
                            <div className="mt-4 p-3 bg-muted/50 rounded-lg">
                                <p className="text-sm text-muted-foreground">
                                    Discussion about new project milestones and upcoming deadlines...
                                </p>
                            </div>
                        </CardContent>
                    </Card>

                    <Card>
                        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                            <CardTitle className="text-sm font-medium">General Chat</CardTitle>
                            <MessageSquare className="h-4 w-4 text-muted-foreground" />
                        </CardHeader>
                        <CardContent>
                            <div className="text-2xl font-bold">8</div>
                            <p className="text-xs text-muted-foreground">
                                messages today
                            </p>
                            <div className="mt-4 p-3 bg-muted/50 rounded-lg">
                                <p className="text-sm text-muted-foreground">
                                    Casual conversations about weekend plans and shared interests...
                                </p>
                            </div>
                        </CardContent>
                    </Card>

                    <Card>
                        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                            <CardTitle className="text-sm font-medium">Daily Standup</CardTitle>
                            <Users className="h-4 w-4 text-muted-foreground" />
                        </CardHeader>
                        <CardContent>
                            <div className="text-2xl font-bold">15</div>
                            <p className="text-xs text-muted-foreground">
                                messages today
                            </p>
                            <div className="mt-4 p-3 bg-muted/50 rounded-lg">
                                <p className="text-sm text-muted-foreground">
                                    Status updates from team members on current sprint progress...
                                </p>
                            </div>
                        </CardContent>
                    </Card>
                </div>

                {/* Sample Timeline */}
                <div className="mt-8">
                    <h2 className="text-lg font-semibold mb-4">Recent Activity</h2>
                    <div className="space-y-4">
                        {[1, 2, 3].map((i) => (
                            <Card key={i} className="p-4">
                                <div className="flex items-start gap-4">
                                    <div className="w-10 h-10 bg-primary/10 rounded-full flex items-center justify-center">
                                        <MessageSquare className="h-5 w-5 text-primary" />
                                    </div>
                                    <div className="flex-1">
                                        <div className="flex items-center gap-2 mb-2">
                                            <span className="font-medium">Sample Group {i}</span>
                                            <Badge variant="outline" className="text-xs">
                                                {i * 3} messages
                                            </Badge>
                                            <span className="text-sm text-muted-foreground">2 hours ago</span>
                                        </div>
                                        <p className="text-sm text-muted-foreground">
                                            This is a sample summary of group activity and key discussion points...
                                        </p>
                                    </div>
                                </div>
                            </Card>
                        ))}
                    </div>
                </div>
            </main>
        </div>
    );
}
