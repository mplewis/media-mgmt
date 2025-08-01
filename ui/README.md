# Media Management UI

TypeScript React UI for interactive media analysis reports.

## Development Setup

### Prerequisites

- [Node.js](https://nodejs.org/) (v18 or later)
- [pnpm](https://pnpm.io/) package manager

### Getting Started

1. Install dependencies:
   ```bash
   pnpm install
   ```

2. Start the development server:
   ```bash
   pnpm dev
   ```

3. Open your browser to [http://localhost:3000](http://localhost:3000)

The development server includes:
- Hot module replacement (HMR)
- Sample media data for testing
- TypeScript type checking
- ESLint linting
- Tailwind CSS for styling

### Available Scripts

- `pnpm dev` - Start development server with HMR
- `pnpm build` - Build for production
- `pnpm preview` - Preview production build locally
- `pnpm lint` - Run ESLint
- `pnpm lint:fix` - Run ESLint with auto-fix
- `pnpm format` - Format code with Prettier
- `pnpm format:check` - Check code formatting
- `pnpm type-check` - Run TypeScript type checking

### Development vs Production

- **Development**: Uses Vite dev server with sample data injected via `window.__MEDIA_DATA__`
- **Production**: Compiled by Go application using esbuild with real media analysis data

### Project Structure

```
src/
├── components/          # React components
│   ├── DataTable.tsx    # Interactive data table
│   ├── SearchBar.tsx    # Search functionality
│   ├── ColumnMenu.tsx   # Column visibility controls
│   ├── PathToggle.tsx   # Path display toggle
│   ├── SummaryCards.tsx # Statistics cards
│   └── Footer.tsx       # Report footer
├── hooks/               # Custom React hooks
│   └── useMediaData.ts  # Media data management
├── types/               # TypeScript type definitions
│   └── media.ts         # Media data interfaces
├── utils/               # Utility functions
│   ├── formatters.ts    # Data formatting helpers
│   ├── pathUtils.ts     # Path manipulation
│   └── sorting.ts       # Data sorting logic
├── index.tsx            # Application entry point
└── Report.tsx           # Main report component
```

### Code Standards

- **TypeScript**: Strict mode enabled with comprehensive type checking
- **ESLint**: Standard configuration with TypeScript and React rules
- **Prettier**: Automated code formatting
- **React**: Function components with hooks
- **CSS**: Tailwind CSS utility classes

### Integration with Go Application

The UI is embedded into the Go binary and compiled at runtime using esbuild. The Go application:

1. Embeds all TypeScript source files using `//go:embed`
2. Injects real media analysis data into the JavaScript
3. Compiles TypeScript to optimized JavaScript using esbuild
4. Generates self-contained HTML reports with embedded React application