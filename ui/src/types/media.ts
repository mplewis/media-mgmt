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
  readonly video_profile?: string
  readonly video_level?: string
  readonly pixel_format?: string
  readonly is_vbr?: boolean
  readonly color_space?: string
  readonly color_transfer?: string
  readonly has_dolby_vision?: boolean
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
  readonly videoProfile: boolean
  readonly videoLevel: boolean
  readonly pixelFormat: boolean
  readonly colorInfo: boolean
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
  | 'videoProfile'
  | 'videoLevel'
  | 'pixelFormat'
  | 'colorInfo'
  | 'audioTracks'
  | 'subtitleTracks'

export interface CodecCounts {
  readonly [codec: string]: number
}