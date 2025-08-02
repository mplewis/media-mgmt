export const formatFileSize = (bytes: number): string => {
  return (bytes / (1024 * 1024)).toFixed(1)
}

export const formatDuration = (seconds: number): string => {
  return (seconds / 60).toFixed(1)
}

export const formatTotalSize = (bytes: number): string => {
  return (bytes / (1024 * 1024 * 1024)).toFixed(1)
}

export const formatTotalDuration = (seconds: number): string => {
  return (seconds / 3600).toFixed(1)
}

export const formatDate = (dateString: string): string => {
  return new Date(dateString).toLocaleString()
}

export const formatAudioTracks = (tracks: readonly { codec: string, language: string, channels: number }[]): string => {
  if (tracks.length === 0) return '0'
  if (tracks.length === 1) {
    const track = tracks[0]
    if (track != null) {
      return `${track.codec} (${track.language}, ${track.channels}ch)`
    }
  }
  return `${tracks.length} tracks: ${tracks.map(t => `${t.codec} (${t.language})`).join(', ')}`
}

export const formatSubtitleTracks = (tracks: readonly { codec: string, language: string }[]): string => {
  if (tracks.length === 0) return '0'
  if (tracks.length === 1) {
    const track = tracks[0]
    if (track != null) {
      return `${track.codec} (${track.language})`
    }
  }
  return `${tracks.length}: ${tracks.map(t => `${t.language}`).join(', ')}`
}