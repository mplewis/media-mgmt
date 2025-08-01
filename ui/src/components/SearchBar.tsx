interface SearchBarProps {
  readonly searchTerm: string
  readonly onSearchChange: (term: string) => void
}

export const SearchBar = ({ searchTerm, onSearchChange }: SearchBarProps): JSX.Element => {
  return (
    <div className="flex-1 max-w-md">
      <input
        type="text"
        placeholder="Search files..."
        value={searchTerm}
        onChange={(e) => { onSearchChange(e.target.value) }}
        className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
      />
    </div>
  )
}