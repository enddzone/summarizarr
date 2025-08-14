"use client";

import Image from "next/image";
import {
  Moon,
  Sun,
  LayoutGrid,
  List,
  ArrowUpDown,
  Download,
  Settings,
} from "lucide-react";
import { useTheme } from "next-themes";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Badge } from "@/components/ui/badge";
import type { ViewMode, SortOrder, SignalConfig } from "@/types";

interface HeaderProps {
  viewMode: ViewMode;
  onViewModeChange: (mode: ViewMode) => void;
  sortOrder: SortOrder;
  onSortOrderChange: (order: SortOrder) => void;
  onExport: () => void;
  onSignalSetup: () => void;
  signalConfig: SignalConfig;
}

export function Header({
  viewMode,
  onViewModeChange,
  sortOrder,
  onSortOrderChange,
  onExport,
  onSignalSetup,
  signalConfig,
}: HeaderProps) {
  const { setTheme, theme } = useTheme();

  return (
    <header className="sticky top-0 z-50 w-full border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="container mx-auto px-4 flex h-16 items-center justify-between">
        {/* Left side - pinned logo, title, and status badge */}
        <div className="flex items-center space-x-2 sm:space-x-3 md:space-x-4 flex-shrink-0">
          <div className="flex items-center min-w-0">
            {/* Logo visible on tablets and up with appropriate sizing */}
            <Image
              src="/main.svg"
              alt="Summarizarr Logo"
              width={96}
              height={96}
              className="hidden sm:block w-12 sm:w-14 md:w-16 lg:w-20 xl:w-28 h-12 sm:h-14 md:h-16 lg:h-20 xl:h-28 object-contain mt-2 sm:mt-3 md:mt-4 md:-mr-4 md:-ml-4 -mr-1 sm:-mr-1 md:-mr-2 lg:-mr-6"
            />
            <h1 className="text-base sm:text-lg md:text-xl lg:text-2xl font-bold tracking-tight min-w-0 md:ml-2">
              <span className="text-primary">SUMMARI</span>
              <span className="text-orange-600">ZARR</span>
            </h1>
          </div>
          <Badge
            variant={signalConfig.isRegistered ? "default" : "secondary"}
            className="hidden sm:flex items-center gap-1.5 flex-shrink-0 text-xs"
          >
            {signalConfig.isRegistered && (
              <div className="w-2 h-2 bg-green-500 rounded-full animate-pulse" />
            )}
            <span className="hidden lg:inline">
              {signalConfig.isRegistered
                ? "Signal Connected"
                : "Signal Not Connected"}
            </span>
            <span className="lg:hidden">
              {signalConfig.isRegistered ? "Connected" : "Not Connected"}
            </span>
          </Badge>
        </div>

        {/* Spacer to push controls to the right */}
        <div className="flex-1"></div>

        <div className="flex items-center space-x-1 sm:space-x-2 flex-shrink-0">
          {/* View Mode Toggle - Hidden on mobile, show cards-only on small screens */}
          <div className="hidden sm:flex items-center border rounded-lg p-1">
            <Button
              variant={viewMode === "timeline" ? "default" : "ghost"}
              size="sm"
              onClick={() => onViewModeChange("timeline")}
              className="px-2 sm:px-3"
            >
              <List className="h-4 w-4" />
              <span className="hidden sm:inline ml-1">Timeline</span>
            </Button>
            <Button
              variant={viewMode === "cards" ? "default" : "ghost"}
              size="sm"
              onClick={() => onViewModeChange("cards")}
              className="px-2 sm:px-3"
            >
              <LayoutGrid className="h-4 w-4" />
              <span className="hidden sm:inline ml-1">Cards</span>
            </Button>
          </div>

          {/* Sort Order */}
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="outline" size="sm" className="hidden sm:flex">
                <ArrowUpDown className="h-4 w-4 mr-2" />
                <span className="hidden lg:inline">
                  {sortOrder === "newest" ? "Newest First" : "Oldest First"}
                </span>
                <span className="lg:hidden">Sort</span>
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={() => onSortOrderChange("newest")}>
                Newest First
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => onSortOrderChange("oldest")}>
                Oldest First
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>

          {/* Mobile Sort Button */}
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="outline" size="sm" className="sm:hidden">
                <ArrowUpDown className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={() => onSortOrderChange("newest")}>
                Newest First
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => onSortOrderChange("oldest")}>
                Oldest First
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>

          {/* Actions Menu for smaller screens */}
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="outline" size="sm" className="md:hidden">
                <Settings className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-48">
              <DropdownMenuItem onClick={onExport}>
                <Download className="h-4 w-4 mr-2" />
                Export
              </DropdownMenuItem>
              <DropdownMenuItem onClick={onSignalSetup}>
                <Settings className="h-4 w-4 mr-2" />
                Signal Setup
              </DropdownMenuItem>
              <DropdownMenuItem
                onClick={() => setTheme(theme === "light" ? "dark" : "light")}
              >
                <Sun className="h-4 w-4 mr-2 rotate-0 scale-100 transition-all dark:-rotate-90 dark:scale-0" />
                <Moon className="absolute h-4 w-4 mr-2 rotate-90 scale-0 transition-all dark:rotate-0 dark:scale-100" />
                Toggle Theme
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>

          {/* Desktop Actions */}
          <div className="hidden md:flex items-center space-x-2">
            {/* Export */}
            <Button variant="outline" size="sm" onClick={onExport}>
              <Download className="h-4 w-4 mr-2" />
              <span className="hidden lg:inline">Export</span>
            </Button>

            {/* Signal Setup */}
            <Button variant="outline" size="sm" onClick={onSignalSetup}>
              <Settings className="h-4 w-4 mr-2" />
              <span className="hidden lg:inline">Signal Setup</span>
            </Button>

            {/* Theme Toggle */}
            <Button
              variant="outline"
              size="sm"
              onClick={() => setTheme(theme === "light" ? "dark" : "light")}
            >
              <Sun className="h-4 w-4 rotate-0 scale-100 transition-all dark:-rotate-90 dark:scale-0" />
              <Moon className="absolute h-4 w-4 rotate-90 scale-0 transition-all dark:rotate-0 dark:scale-100" />
              <span className="sr-only">Toggle theme</span>
            </Button>
          </div>
        </div>
      </div>
    </header>
  );
}
