import { useEffect, useState } from 'react'

interface PaginationProps {
  currentPage: number
  totalPages: number
  totalItems: number
  pageSize: number
  onPageChange: (page: number) => void
}

export const Pagination = ({ 
  currentPage, 
  totalPages, 
  totalItems, 
  pageSize, 
  onPageChange 
}: PaginationProps): JSX.Element => {
  const [pageInput, setPageInput] = useState(currentPage.toString())

  // Update input when currentPage changes externally
  useEffect(() => {
    setPageInput(currentPage.toString())
  }, [currentPage])

  // Keyboard navigation
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Only handle if not typing in an input field
      if (e.target instanceof HTMLInputElement) return

      if (e.key === 'ArrowLeft') {
        e.preventDefault()
        if (currentPage > 1) onPageChange(currentPage - 1)
      } else if (e.key === 'ArrowRight') {
        e.preventDefault()
        if (currentPage < totalPages) onPageChange(currentPage + 1)
      } else if (e.key === 'Home' || (e.key === 'ArrowLeft' && (e.ctrlKey || e.metaKey))) {
        e.preventDefault()
        onPageChange(1)
      } else if (e.key === 'End' || (e.key === 'ArrowRight' && (e.ctrlKey || e.metaKey))) {
        e.preventDefault()
        onPageChange(totalPages)
      }
    }

    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [currentPage, totalPages, onPageChange])

  const handlePageInputSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    const page = parseInt(pageInput, 10)
    if (page >= 1 && page <= totalPages) {
      onPageChange(page)
    } else {
      setPageInput(currentPage.toString()) // Reset invalid input
    }
  }

  const handlePageInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setPageInput(e.target.value)
  }

  if (totalItems === 0) {
    return <></>
  }

  const startItem = (currentPage - 1) * pageSize + 1
  const endItem = Math.min(currentPage * pageSize, totalItems)

  // Generate page numbers to display (max 5)
  const getPageNumbers = (): number[] => {
    const pages: number[] = []
    const maxPages = Math.min(5, totalPages)
    
    if (totalPages <= 5) {
      for (let i = 1; i <= totalPages; i++) {
        pages.push(i)
      }
    } else if (currentPage <= 3) {
      for (let i = 1; i <= 5; i++) {
        pages.push(i)
      }
    } else if (currentPage >= totalPages - 2) {
      for (let i = totalPages - 4; i <= totalPages; i++) {
        pages.push(i)
      }
    } else {
      for (let i = currentPage - 2; i <= currentPage + 2; i++) {
        pages.push(i)
      }
    }
    
    return pages
  }

  return (
    <div className="px-6 py-4 bg-gray-50 border-t border-gray-200">
      <div className="flex flex-col sm:flex-row items-center justify-between gap-4">
        <div className="text-sm text-gray-500">
          Showing {startItem} to {endItem} of {totalItems} filtered results
        </div>
        
        <div className="flex items-center justify-center gap-2">
          <button
            onClick={() => onPageChange(1)}
            disabled={currentPage === 1}
            className="w-10 h-8 text-sm border border-gray-300 rounded-md disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-100 flex items-center justify-center"
            title="First page (Home)"
          >
            «
          </button>
          
          <button
            onClick={() => onPageChange(currentPage - 1)}
            disabled={currentPage === 1}
            className="w-10 h-8 text-sm border border-gray-300 rounded-md disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-100 flex items-center justify-center"
            title="Previous page (←)"
          >
            ‹
          </button>
          
          <div className="flex items-center gap-1">
            {getPageNumbers().map((pageNum) => (
              <button
                key={pageNum}
                onClick={() => onPageChange(pageNum)}
                className={`w-10 h-8 text-sm border rounded-md flex items-center justify-center ${
                  pageNum === currentPage
                    ? 'bg-blue-500 text-white border-blue-500'
                    : 'border-gray-300 hover:bg-gray-100'
                }`}
              >
                {pageNum}
              </button>
            ))}
          </div>
          
          <form onSubmit={handlePageInputSubmit} className="flex items-center gap-1">
            <span className="text-sm text-gray-500 whitespace-nowrap">Go to:</span>
            <input
              type="number"
              min="1"
              max={totalPages}
              value={pageInput}
              onChange={handlePageInputChange}
              className="w-16 px-2 py-1 text-sm border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500 text-center"
              title="Enter page number"
            />
          </form>
          
          <button
            onClick={() => onPageChange(currentPage + 1)}
            disabled={currentPage === totalPages}
            className="w-10 h-8 text-sm border border-gray-300 rounded-md disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-100 flex items-center justify-center"
            title="Next page (→)"
          >
            ›
          </button>
          
          <button
            onClick={() => onPageChange(totalPages)}
            disabled={currentPage === totalPages}
            className="w-10 h-8 text-sm border border-gray-300 rounded-md disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-100 flex items-center justify-center"
            title="Last page (End)"
          >
            »
          </button>
        </div>
      </div>
    </div>
  )
}