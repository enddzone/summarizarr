# Date Range Picker Filtering Issue - Investigation Summary

## Issue Description
When users select a date range manually using the date picker calendar (e.g., Aug 12-14), the summaries are not being returned correctly. However, preset buttons like "Yesterday" work fine.

## Root Cause Identified
The issue is a **timezone conversion problem** in the date range picker. When users select dates manually:

1. The calendar creates Date objects in local timezone
2. These get converted to Unix timestamps incorrectly
3. The resulting timestamp range doesn't match the intended date range

## Evidence from Console Logs
User selected "Aug 12-14" but the actual timestamps sent to API were:
- Start: `1755147600` = Aug 14 05:00:00 UTC (should be Aug 12 00:00:00 UTC = `1754956800`)
- End: `1755231210` = Aug 15 04:13:30 UTC (should be Aug 14 23:59:59 UTC = `1755215999`)

## Current State
- **Environment**: Docker development stack running on localhost:8081
- **Backend**: Working correctly - filtering logic is sound
- **Database**: Contains summaries from Aug 13-14, 2025 (not 2024)
- **Frontend**: Has debug logging enabled in date picker components

## Files Modified
1. `/web/src/components/ui/date-range-picker.tsx` - Added debug logging and end-of-day fix
2. `/web/src/components/filter-panel.tsx` - Added debug logging to handleDateRangeChange

## Debug Features Added
- Console logging in `handleSelect` function shows date selection process
- Console logging in `handleDateRangeChange` shows filter updates
- Modified logic to set end time to 23:59:59.999 for full day inclusion

## What Works
- Preset buttons ("Yesterday", "Today", etc.) work correctly
- Backend API filtering works correctly when given proper timestamps
- Manual date selection triggers the right functions and API calls

## What Doesn't Work
- Manual date range selection creates wrong timestamps due to timezone handling
- The calendar component may be interpreting dates in local timezone vs UTC

## Technical Details
- Project: Summarizarr (AI-powered Signal message summarizer)
- Stack: Go backend, Next.js frontend, SQLite database
- Date picker: shadcn/ui calendar component with custom range handling
- Backend filtering: Uses Unix timestamps converted to milliseconds

## Services Running
- Docker compose stack on localhost:8081
- Signal CLI container
- Backend container with debug logging
- Database with summaries from 2025-08-13 to 2025-08-14

## Missing Console Logs
The user's browser console showed `fetchSummaries Debug` logs but was missing the expected `DatePickerWithRange handleSelect` logs, suggesting either:
1. The handleSelect function isn't being called
2. Console logs were cleared or not visible
3. Different code path being taken

## Immediate Investigation Needed
Check browser console for missing debug logs from:
- `=== DatePickerWithRange handleSelect ===`
- `=== handleDateRangeChange called ===`

These will show the actual Date objects being created during manual selection.