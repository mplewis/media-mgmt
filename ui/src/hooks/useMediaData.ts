import { useState, useEffect } from 'react'
import type { MediaData } from '../types/media'

// Declare global for injected data
declare global {
  interface Window {
    __MEDIA_DATA__?: MediaData
  }
}

export const useMediaData = (): MediaData => {
  const [data, setData] = useState<MediaData>({
    mediaFiles: [],
    totalFiles: 0,
    generatedAt: ''
  })

  useEffect(() => {
    // First try to use injected data from esbuild
    if (window.__MEDIA_DATA__ != null) {
      setData(window.__MEDIA_DATA__)
      return
    }

    // Fallback to reading from DOM (for backwards compatibility)
    const jsonElement = document.getElementById('media-data')
    if (jsonElement?.textContent != null) {
      try {
        const parsedData = JSON.parse(jsonElement.textContent) as MediaData
        setData(parsedData)
      } catch (error) {
        console.error('Failed to parse media data:', error)
      }
    }
  }, [])

  return data
}