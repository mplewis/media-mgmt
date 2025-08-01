import { useState, useMemo } from 'react'
import type { SortConfig, ColumnVisibility, SortableColumn } from '../types/media'
import { useMediaData } from '../hooks/useMediaData'
import { sortMediaFiles } from '../utils/sorting'
import { SummaryCards } from './SummaryCards'
import { SearchBar } from './SearchBar'
import { PathToggle } from './PathToggle'
import { ColumnMenu } from './ColumnMenu'
import { DataTable } from './DataTable'
import { Footer } from './Footer'

export const MediaAnalysisReport = (): JSX.Element => {
  const data = useMediaData()
  const [searchTerm, setSearchTerm] = useState('')
  const [sortConfig, setSortConfig] = useState<SortConfig>({ key: null, direction: 'asc' })
  const [showRelativePaths, setShowRelativePaths] = useState(false)
  const [columnVisibility, setColumnVisibility] = useState<ColumnVisibility>({
    file: true,
    size: true,
    duration: true,
    videoCodec: true,
    bitrate: true,
    resolution: true,
    audioTracks: true,
    subtitleTracks: true
  })
  const [showColumnMenu, setShowColumnMenu] = useState(false)

  const filteredAndSortedData = useMemo(() => {
    const filtered = data.mediaFiles.filter(item => {
      const searchLower = searchTerm.toLowerCase()
      return (
        item.file_path.toLowerCase().includes(searchLower) ||
        item.video_codec.toLowerCase().includes(searchLower)
      )
    })

    return sortMediaFiles(filtered, sortConfig, showRelativePaths)
  }, [data.mediaFiles, searchTerm, sortConfig, showRelativePaths])

  const handleSort = (key: SortableColumn): void => {
    setSortConfig(prevConfig => ({
      key,
      direction: prevConfig.key === key && prevConfig.direction === 'asc' ? 'desc' : 'asc'
    }))
  }

  const toggleColumnVisibility = (column: keyof ColumnVisibility): void => {
    setColumnVisibility(prev => ({
      ...prev,
      [column]: !prev[column]
    }))
  }

  return (
    <div className="min-h-screen bg-gray-50 py-8">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="bg-white shadow-xl rounded-lg overflow-hidden">
          <SummaryCards data={data} />

          <div className="px-6 py-4 bg-gray-50 border-b border-gray-200">
            <div className="flex flex-col sm:flex-row gap-4 items-start sm:items-center justify-between">
              <SearchBar searchTerm={searchTerm} onSearchChange={setSearchTerm} />

              <div className="flex flex-col sm:flex-row gap-4">
                <PathToggle
                  showRelativePaths={showRelativePaths}
                  onToggle={setShowRelativePaths}
                />

                <ColumnMenu
                  columnVisibility={columnVisibility}
                  showMenu={showColumnMenu}
                  onToggleMenu={() => { setShowColumnMenu(!showColumnMenu) }}
                  onToggleColumn={toggleColumnVisibility}
                />
              </div>
            </div>
          </div>

          <DataTable
            data={filteredAndSortedData}
            columnVisibility={columnVisibility}
            sortConfig={sortConfig}
            showRelativePaths={showRelativePaths}
            onSort={handleSort}
          />

          <Footer
            generatedAt={data.generatedAt}
            totalFiles={data.totalFiles}
            filteredCount={filteredAndSortedData.length}
          />
        </div>
      </div>
    </div>
  )
}