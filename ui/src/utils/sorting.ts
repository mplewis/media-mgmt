import type { MediaFile, SortableColumn, SortConfig } from '../types/media'
import { getDisplayPath } from './pathUtils'

export const sortMediaFiles = (
  files: readonly MediaFile[],
  sortConfig: SortConfig,
  showRelativePaths: boolean
): readonly MediaFile[] => {
  if (sortConfig.key === null) {
    return files
  }

  const sortedFiles = [...files].sort((a, b) => {
    let aVal: string | number
    let bVal: string | number

    switch (sortConfig.key as SortableColumn) {
      case 'file':
        aVal = getDisplayPath(a.file_path, showRelativePaths)
        bVal = getDisplayPath(b.file_path, showRelativePaths)
        break
      case 'size':
        aVal = a.file_size
        bVal = b.file_size
        break
      case 'duration':
        aVal = a.duration
        bVal = b.duration
        break
      case 'videoCodec':
        aVal = a.video_codec
        bVal = b.video_codec
        break
      case 'bitrate':
        aVal = a.video_bitrate
        bVal = b.video_bitrate
        break
      case 'resolution':
        aVal = a.video_width * a.video_height
        bVal = b.video_width * b.video_height
        break
      case 'audioTracks':
        aVal = a.audio_tracks.length
        bVal = b.audio_tracks.length
        break
      case 'subtitleTracks':
        aVal = a.subtitle_tracks.length
        bVal = b.subtitle_tracks.length
        break
      default:
        return 0
    }

    if (typeof aVal === 'string') {
      const result = aVal.localeCompare(bVal as string)
      return sortConfig.direction === 'asc' ? result : -result
    } else {
      const result = aVal - (bVal as number)
      return sortConfig.direction === 'asc' ? result : -result
    }
  })

  return sortedFiles
}