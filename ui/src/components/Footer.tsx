import { formatDate } from '../utils/formatters'

interface FooterProps {
  readonly generatedAt: string
  readonly totalFiles: number
  readonly filteredCount: number
}

export const Footer = ({ generatedAt, totalFiles, filteredCount }: FooterProps): JSX.Element => {
  return (
    <div className="px-6 py-4 bg-gray-50 border-t border-gray-200">
      <div className="text-sm text-gray-500 text-center">
        Generated on {formatDate(generatedAt)} •{' '}
        Showing {filteredCount} of {totalFiles} files
      </div>
    </div>
  )
}