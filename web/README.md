# Summarizarr Web Interface

A modern, responsive web interface for Summarizarr built with Next.js 15, featuring real-time AI-powered Signal message summaries.

## Features

### 🎨 Modern UI/UX
- **Dark/Light Mode**: Automatic theme switching with system preference detection
- **Responsive Design**: Mobile-first approach with Tailwind CSS
- **Timeline & Cards View**: Switch between detailed timeline and compact card layouts
- **Real-time Updates**: Live summary updates via WebSocket connections
- **Accessibility**: WCAG compliant with keyboard navigation and screen reader support

### 🔍 Advanced Filtering
- **Multi-select Group Filters**: Filter summaries by Signal groups
- **Date Range Picker**: Custom time period selection
- **Search Functionality**: Full-text search across summary content
- **Sort Options**: Sort by newest or oldest first

### 📊 Export & Analytics
- **Multiple Export Formats**: JSON, CSV, and PDF export options
- **Filtered Exports**: Export based on current filter selections
- **Summary Statistics**: Overview of message activity and trends

### 🔧 Signal Integration
- **QR Code Setup**: Easy Signal registration wizard
- **Connection Status**: Real-time Signal connection monitoring
- **Phone Number Management**: Secure phone number registration

## Tech Stack

- **Framework**: Next.js 15 with App Router
- **Styling**: Tailwind CSS with custom design system
- **UI Components**: Radix UI primitives with custom styling
- **Type Safety**: TypeScript throughout
- **Date Handling**: date-fns for reliable date operations
- **Icons**: Lucide React for consistent iconography
- **State Management**: React hooks with optimistic updates

## Quick Start

### Development Setup

1. **Install Dependencies**
   ```bash
   cd web
   npm install
   ```

2. **Environment Configuration**
   ```bash
   cp .env.local.example .env.local
   # Edit .env.local with your configuration
   ```

3. **Start Development Server**
   ```bash
   npm run dev
   ```

4. **Access the Interface**
   Open [http://localhost:3000](http://localhost:3000)

### Production Deployment

#### Docker (Recommended)

1. **Build and Start All Services**
   ```bash
   # From project root
   docker-compose up -d
   ```

2. **Access the Application**
   - Frontend: [http://localhost:3000](http://localhost:3000)
   - Backend API: [http://localhost:8081](http://localhost:8081)
   - Signal CLI: [http://localhost:8080](http://localhost:8080)

#### Production with Nginx

1. **Start with Production Profile**
   ```bash
   docker-compose --profile production up -d
   ```

2. **Access via Nginx**
   - Application: [http://localhost](http://localhost)
   - HTTPS: [https://localhost](https://localhost) (with SSL setup)

### Manual Build

1. **Build Frontend**
   ```bash
   cd web
   npm run build
   npm start
   ```

2. **Build Backend**
   ```bash
   # From project root
   go build -o summarizarr ./cmd/summarizarr
   ./summarizarr
   ```

## Environment Variables

### Frontend (.env.local)
```bash
# Backend API URL
BACKEND_URL=http://localhost:8081

# Public app URL for client-side requests
NEXT_PUBLIC_APP_URL=http://localhost:3000

# Disable Next.js telemetry (optional)
NEXT_TELEMETRY_DISABLED=1
```

### Backend (docker-compose.yml)
```bash
# Signal configuration
SIGNAL_PHONE_NUMBER=+1234567890

# AI backend settings
AI_BACKEND=local  # or 'openai'
OLLAMA_HOST=127.0.0.1:11434
OPENAI_API_KEY=your_openai_key

# Database
DATABASE_PATH=/app/data/summarizarr.db

# Logging
LOG_LEVEL=DEBUG

# Summarization schedule
SUMMARIZATION_INTERVAL=1h
```

## API Integration

The frontend communicates with the Go backend through these API endpoints:

### Summaries
- `GET /api/summaries` - Fetch summaries with filters
- Query parameters: `groups`, `start_time`, `end_time`, `search`, `sort`

### Groups
- `GET /api/groups` - Fetch available Signal groups

### Export
- `GET /api/export` - Export summaries in various formats
- Query parameters: `format` (json|csv|pdf), filters

### Signal Management
- `GET /api/signal/config` - Get Signal registration status
- `POST /api/signal/register` - Register phone number
- `GET /api/signal/status` - Check registration status

## Component Architecture

```
src/
├── app/                    # Next.js App Router
│   ├── api/               # API route handlers
│   ├── globals.css        # Global styles
│   ├── layout.tsx         # Root layout
│   └── page.tsx           # Home page
├── components/            # React components
│   ├── ui/               # Reusable UI components
│   ├── header.tsx        # Main navigation
│   ├── filter-panel.tsx  # Filtering interface
│   ├── summary-list.tsx  # Timeline view
│   ├── summary-cards.tsx # Cards view
│   └── ...               # Dialog components
├── hooks/                # Custom React hooks
├── lib/                  # Utility functions
└── types/                # TypeScript type definitions
```

## Customization

### Theming
Customize the design system in `tailwind.config.ts`:
```typescript
theme: {
  extend: {
    colors: {
      // Add custom colors
    },
    fontFamily: {
      // Add custom fonts
    }
  }
}
```

### UI Components
All UI components are built with Radix UI and can be customized in `src/components/ui/`.

### API Routes
Add new API endpoints in `src/app/api/` following the existing pattern.

## Performance Optimizations

- **Static Generation**: Pages pre-rendered at build time
- **Code Splitting**: Automatic route-based code splitting
- **Image Optimization**: Next.js automatic image optimization
- **Bundle Analysis**: Built-in bundle analyzer
- **Caching**: Optimistic updates and request caching

## Troubleshooting

### Common Issues

1. **Build Errors**
   ```bash
   # Clear Next.js cache
   rm -rf .next
   npm run build
   ```

2. **TypeScript Errors**
   ```bash
   # Check types without building
   npm run type-check
   ```

3. **API Connection Issues**
   - Verify backend is running on port 8081
   - Check BACKEND_URL in .env.local
   - Ensure CORS is configured correctly

4. **Signal Integration**
   - Verify Signal CLI is running on port 8080
   - Check QR code generation
   - Ensure phone number format is correct

### Development Tips

- Use `npm run dev` for hot reloading
- Enable debug logging: `LOG_LEVEL=DEBUG`
- Monitor API calls in browser DevTools
- Use React DevTools for component debugging

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.
