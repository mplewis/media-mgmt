export interface AudioTrack {
  readonly index: number
  readonly codec: string
  readonly bitrate: number
  readonly language: string
  readonly channels: number
}

export interface SubtitleTrack {
  readonly index: number
  readonly codec: string
  readonly language: string
}

export interface MediaFile {
  readonly file_path: string
  readonly file_size: number
  readonly duration: number
  readonly video_codec: string
  readonly video_bitrate: number
  readonly video_width: number
  readonly video_height: number
  readonly audio_tracks: readonly AudioTrack[]
  readonly subtitle_tracks: readonly SubtitleTrack[]
  readonly analyzed_at: string
}

export interface MediaData {
  readonly mediaFiles: readonly MediaFile[]
  readonly totalFiles: number
  readonly generatedAt: string
}

export interface SortConfig {
  readonly key: string | null
  readonly direction: 'asc' | 'desc'
}

export interface ColumnVisibility {
  readonly file: boolean
  readonly size: boolean
  readonly duration: boolean
  readonly videoCodec: boolean
  readonly bitrate: boolean
  readonly resolution: boolean
  readonly audioTracks: boolean
  readonly subtitleTracks: boolean
}

export type SortableColumn = 
  | 'file'
  | 'size' 
  | 'duration'
  | 'videoCodec'
  | 'bitrate'
  | 'resolution'
  | 'audioTracks'
  | 'subtitleTracks'

export interface CodecCounts {
  readonly [codec: string]: number
}