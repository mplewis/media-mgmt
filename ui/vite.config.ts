import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// Sample media data for development
const sampleMediaData = {
  mediaFiles: [
    {
      file_path: '/Users/dev/media/sample-video-1.mkv',
      file_size: 1024 * 1024 * 1024, // 1GB
      duration: 3600, // 1 hour
      video_codec: 'hevc',
      video_bitrate: 8000000, // 8Mbps
      video_width: 1920,
      video_height: 1080,
      audio_tracks: [
        { index: 1, codec: 'aac', bitrate: 128000, language: 'eng', channels: 2 }
      ],
      subtitle_tracks: [
        { index: 2, codec: 'srt', language: 'eng' }
      ],
      analyzed_at: new Date().toISOString()
    },
    {
      file_path: '/Users/dev/media/sample-video-2.mp4',
      file_size: 2 * 1024 * 1024 * 1024, // 2GB
      duration: 5400, // 1.5 hours
      video_codec: 'h264',
      video_bitrate: 6000000, // 6Mbps
      video_width: 3840,
      video_height: 2160,
      audio_tracks: [
        { index: 1, codec: 'dts', bitrate: 1536000, language: 'eng', channels: 6 },
        { index: 2, codec: 'aac', bitrate: 256000, language: 'eng', channels: 2 }
      ],
      subtitle_tracks: [
        { index: 3, codec: 'hdmv_pgs_subtitle', language: 'eng' },
        { index: 4, codec: 'hdmv_pgs_subtitle', language: 'fre' },
        { index: 5, codec: 'hdmv_pgs_subtitle', language: 'spa' }
      ],
      analyzed_at: new Date().toISOString()
    }
  ],
  totalFiles: 2,
  generatedAt: new Date().toISOString()
}

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  define: {
    // Inject sample data for development
    'window.__MEDIA_DATA__': JSON.stringify(sampleMediaData)
  },
  server: {
    port: 3000,
    open: true
  },
  build: {
    outDir: 'dist',
    sourcemap: true,
    rollupOptions: {
      output: {
        manualChunks: {
          react: ['react', 'react-dom']
        }
      }
    }
  }
})