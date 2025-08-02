import type { MediaData, CodecCounts } from '../types/media'
import { formatTotalSize, formatTotalDuration } from '../utils/formatters'

interface SummaryCardsProps {
  readonly data: MediaData
}

export const SummaryCards = ({ data }: SummaryCardsProps): JSX.Element => {
  const totalSize = data.mediaFiles.reduce((sum, item) => sum + item.file_size, 0)
  const totalDuration = data.mediaFiles.reduce((sum, item) => sum + item.duration, 0)

  const codecCounts: CodecCounts = data.mediaFiles.reduce<CodecCounts>((acc, item) => {
    const count = acc[item.video_codec] ?? 0
    return { ...acc, [item.video_codec]: count + 1 }
  }, {})

  return (
    <div className="px-6 py-8 border-b border-gray-200">
      <h1 className="text-3xl font-bold text-gray-900 text-center mb-6">
        Media Analysis Report
      </h1>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-6">
        <div className="bg-blue-50 rounded-lg p-4 text-center">
          <div className="text-2xl font-bold text-blue-600">{data.totalFiles}</div>
          <div className="text-sm text-gray-600">Total Files</div>
        </div>
        <div className="bg-green-50 rounded-lg p-4 text-center">
          <div className="text-2xl font-bold text-green-600">
            {formatTotalSize(totalSize)} GB
          </div>
          <div className="text-sm text-gray-600">Total Size</div>
        </div>
        <div className="bg-purple-50 rounded-lg p-4 text-center">
          <div className="text-2xl font-bold text-purple-600">
            {formatTotalDuration(totalDuration)} hrs
          </div>
          <div className="text-sm text-gray-600">Total Duration</div>
        </div>
      </div>

      <div className="bg-gray-50 rounded-lg p-4 mb-6">
        <h3 className="text-sm font-medium text-gray-700 mb-2">Video Codecs</h3>
        <div className="flex flex-wrap gap-2">
          {Object.entries(codecCounts).map(([codec, count]) => (
            <span
              key={codec}
              className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800"
            >
              {codec}: {count} files
            </span>
          ))}
        </div>
      </div>
    </div>
  )
}