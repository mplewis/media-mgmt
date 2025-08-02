import { createRoot } from 'react-dom/client'
import { MediaAnalysisReport } from './components/MediaAnalysisReport'

// Declare globals that will be available from CDN
declare global {
  interface Window {
    React: typeof import('react')
    ReactDOM: typeof import('react-dom') & typeof import('react-dom/client')
  }
}

const container = document.getElementById('root')
if (container != null) {
  const root = createRoot(container)
  root.render(<MediaAnalysisReport />)
} else {
  console.error('Root element not found')
}