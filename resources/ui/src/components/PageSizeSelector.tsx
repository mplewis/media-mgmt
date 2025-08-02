interface PageSizeSelectorProps {
  pageSize: number
  onPageSizeChange: (size: number) => void
}

export const PageSizeSelector = ({ pageSize, onPageSizeChange }: PageSizeSelectorProps): JSX.Element => {
  return (
    <div className="flex items-center gap-2">
      <span className="text-sm text-gray-700">Show:</span>
      <select
        value={pageSize}
        onChange={(e) => onPageSizeChange(Number(e.target.value))}
        className="px-2 py-1 text-sm border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
      >
        <option value={10}>10</option>
        <option value={25}>25</option>
        <option value={50}>50</option>
        <option value={100}>100</option>
      </select>
    </div>
  )
}