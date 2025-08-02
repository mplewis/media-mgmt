import type { MediaFile, ColumnVisibility, SortableColumn, SortConfig } from '../types/media'
import { formatFileSize, formatDuration, formatAudioTracks, formatSubtitleTracks } from '../utils/formatters'
import { getDisplayPath } from '../utils/pathUtils'

interface DataTableProps {
  readonly data: readonly MediaFile[]
  readonly columnVisibility: ColumnVisibility
  readonly sortConfig: SortConfig
  readonly showRelativePaths: boolean
  readonly inputDir?: string
  readonly onSort: (column: SortableColumn) => void
}

const getSortIcon = (columnKey: SortableColumn, sortConfig: SortConfig): string => {
  if (sortConfig.key === columnKey) {
    return sortConfig.direction === 'asc' ? ' ↑' : ' ↓'
  }
  return ' ↕'
}

export const DataTable = ({
  data,
  columnVisibility,
  sortConfig,
  showRelativePaths,
  inputDir,
  onSort
}: DataTableProps): JSX.Element => {
  const handleSort = (column: SortableColumn): void => {
    onSort(column)
  }

  if (data.length === 0) {
    return (
      <div className="text-center py-12">
        <div className="text-gray-500">No files match your search criteria.</div>
      </div>
    )
  }

  return (
    <div className="overflow-x-auto">
      <table className="min-w-full divide-y divide-gray-200">
        <thead className="bg-gray-50">
          <tr>
            {columnVisibility.file && (
              <th
                onClick={() => { handleSort('file') }}
                className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-100 select-none"
              >
                File{getSortIcon('file', sortConfig)}
              </th>
            )}
            {columnVisibility.size && (
              <th
                onClick={() => { handleSort('size') }}
                className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-100 select-none"
              >
                Size (MB){getSortIcon('size', sortConfig)}
              </th>
            )}
            {columnVisibility.duration && (
              <th
                onClick={() => { handleSort('duration') }}
                className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-100 select-none"
              >
                Duration (min){getSortIcon('duration', sortConfig)}
              </th>
            )}
            {columnVisibility.videoCodec && (
              <th
                onClick={() => { handleSort('videoCodec') }}
                className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-100 select-none"
              >
                Video Codec{getSortIcon('videoCodec', sortConfig)}
              </th>
            )}
            {columnVisibility.bitrate && (
              <th
                onClick={() => { handleSort('bitrate') }}
                className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-100 select-none"
              >
                Bitrate (Mbps){getSortIcon('bitrate', sortConfig)}
              </th>
            )}
            {columnVisibility.resolution && (
              <th
                onClick={() => { handleSort('resolution') }}
                className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-100 select-none"
              >
                Resolution{getSortIcon('resolution', sortConfig)}
              </th>
            )}
            {columnVisibility.videoProfile && (
              <th
                onClick={() => { handleSort('videoProfile') }}
                className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-100 select-none"
              >
                Profile{getSortIcon('videoProfile', sortConfig)}
              </th>
            )}
            {columnVisibility.videoLevel && (
              <th
                onClick={() => { handleSort('videoLevel') }}
                className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-100 select-none"
              >
                Level{getSortIcon('videoLevel', sortConfig)}
              </th>
            )}
            {columnVisibility.pixelFormat && (
              <th
                onClick={() => { handleSort('pixelFormat') }}
                className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-100 select-none"
              >
                Pixel Format{getSortIcon('pixelFormat', sortConfig)}
              </th>
            )}
            {columnVisibility.colorInfo && (
              <th
                onClick={() => { handleSort('colorInfo') }}
                className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-100 select-none"
              >
                Color/HDR{getSortIcon('colorInfo', sortConfig)}
              </th>
            )}
            {columnVisibility.audioTracks && (
              <th
                onClick={() => { handleSort('audioTracks') }}
                className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-100 select-none"
              >
                Audio Tracks{getSortIcon('audioTracks', sortConfig)}
              </th>
            )}
            {columnVisibility.subtitleTracks && (
              <th
                onClick={() => { handleSort('subtitleTracks') }}
                className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-100 select-none"
              >
                Subtitle Tracks{getSortIcon('subtitleTracks', sortConfig)}
              </th>
            )}
          </tr>
        </thead>
        <tbody className="bg-white divide-y divide-gray-200">
          {data.map((item, index) => (
            <tr key={index} className="hover:bg-gray-50">
              {columnVisibility.file && (
                <td
                  className="px-6 py-4 text-sm text-gray-900 font-mono"
                  title={item.file_path}
                >
                  {getDisplayPath(item.file_path, showRelativePaths, inputDir)}
                </td>
              )}
              {columnVisibility.size && (
                <td className="px-6 py-4 text-sm text-gray-900 text-right">
                  {formatFileSize(item.file_size)}
                </td>
              )}
              {columnVisibility.duration && (
                <td className="px-6 py-4 text-sm text-gray-900 text-right">
                  {formatDuration(item.duration)}
                </td>
              )}
              {columnVisibility.videoCodec && (
                <td className="px-6 py-4 text-sm text-gray-900">
                  <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800">
                    {item.video_codec}
                  </span>
                </td>
              )}
              {columnVisibility.bitrate && (
                <td className="px-6 py-4 text-sm text-gray-900 text-right">
                  {(item.video_bitrate / 1000000).toFixed(1)}
                </td>
              )}
              {columnVisibility.resolution && (
                <td className="px-6 py-4 text-sm text-gray-900">
                  {item.video_width}×{item.video_height}
                </td>
              )}
              {columnVisibility.videoProfile && (
                <td className="px-6 py-4 text-sm text-gray-900">
                  {item.video_profile || 'N/A'}
                </td>
              )}
              {columnVisibility.videoLevel && (
                <td className="px-6 py-4 text-sm text-gray-900">
                  {item.video_level || 'N/A'}
                </td>
              )}
              {columnVisibility.pixelFormat && (
                <td className="px-6 py-4 text-sm text-gray-900 font-mono text-xs">
                  {item.pixel_format || 'N/A'}
                </td>
              )}
              {columnVisibility.colorInfo && (
                <td className="px-6 py-4 text-sm text-gray-900">
                  <div className="flex flex-col">
                    {item.color_transfer === 'smpte2084' && (
                      <span className="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-yellow-100 text-yellow-800 mb-1">
                        HDR10
                      </span>
                    )}
                    {item.has_dolby_vision && (
                      <span className="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-purple-100 text-purple-800 mb-1">
                        Dolby Vision
                      </span>
                    )}
                    {item.is_vbr && (
                      <span className="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-blue-100 text-blue-800">
                        VBR
                      </span>
                    )}
                  </div>
                </td>
              )}
              {columnVisibility.audioTracks && (
                <td className="px-6 py-4 text-sm text-gray-900">
                  <span className="font-mono text-xs">
                    {formatAudioTracks(item.audio_tracks)}
                  </span>
                </td>
              )}
              {columnVisibility.subtitleTracks && (
                <td className="px-6 py-4 text-sm text-gray-900">
                  <span className="font-mono text-xs">
                    {formatSubtitleTracks(item.subtitle_tracks)}
                  </span>
                </td>
              )}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}