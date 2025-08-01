interface PathToggleProps {
  readonly showRelativePaths: boolean
  readonly onToggle: (show: boolean) => void
}

export const PathToggle = ({ showRelativePaths, onToggle }: PathToggleProps): JSX.Element => {
  return (
    <label className="flex items-center">
      <input
        type="checkbox"
        checked={showRelativePaths}
        onChange={(e) => { onToggle(e.target.checked) }}
        className="mr-2 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
      />
      <span className="text-sm text-gray-700">Show relative paths</span>
    </label>
  )
}