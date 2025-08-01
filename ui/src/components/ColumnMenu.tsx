import { useRef, useEffect, useState } from 'react'
import type { ColumnVisibility } from '../types/media'

interface ColumnMenuProps {
  readonly columnVisibility: ColumnVisibility
  readonly showMenu: boolean
  readonly onToggleMenu: () => void
  readonly onToggleColumn: (column: keyof ColumnVisibility) => void
}

interface MenuPosition {
  top: number
  left: number
}

const formatColumnName = (column: string): string => {
  return column.replace(/([A-Z])/g, ' $1').trim()
}

export const ColumnMenu = ({
  columnVisibility,
  showMenu,
  onToggleMenu,
  onToggleColumn
}: ColumnMenuProps): JSX.Element => {
  const buttonRef = useRef<HTMLButtonElement>(null)
  const menuRef = useRef<HTMLDivElement>(null)
  const [menuPosition, setMenuPosition] = useState<MenuPosition>({ top: 0, left: 0 })

  useEffect(() => {
    if (showMenu && buttonRef.current != null) {
      const rect = buttonRef.current.getBoundingClientRect()
      const menuWidth = 192 // w-48 = 12rem = 192px
      const viewportWidth = window.innerWidth
      const viewportHeight = window.innerHeight
      
      // Calculate horizontal position
      let left = rect.right - menuWidth // Align right edge of menu with right edge of button
      
      // If menu would go off the left edge, align with left edge of button instead
      if (left < 8) {
        left = rect.left
      }
      
      // If menu would still go off right edge, constrain it
      if (left + menuWidth > viewportWidth - 8) {
        left = viewportWidth - menuWidth - 8
      }
      
      // Calculate vertical position
      let top = rect.bottom + window.scrollY + 8 // 8px gap below button, account for scroll
      
      // If menu would go off bottom, position above button
      const menuHeight = Object.keys(columnVisibility).length * 40 + 16 // Estimate menu height
      if (rect.bottom + menuHeight > viewportHeight) {
        top = rect.top + window.scrollY - menuHeight - 8
      }
      
      setMenuPosition({ top, left })
    }
  }, [showMenu, columnVisibility])

  // Close menu when clicking outside
  useEffect(() => {
    if (!showMenu) return

    const handleClickOutside = (event: MouseEvent): void => {
      if (
        buttonRef.current != null &&
        menuRef.current != null &&
        !buttonRef.current.contains(event.target as Node) &&
        !menuRef.current.contains(event.target as Node)
      ) {
        onToggleMenu()
      }
    }

    document.addEventListener('mousedown', handleClickOutside)
    return () => { document.removeEventListener('mousedown', handleClickOutside) }
  }, [showMenu, onToggleMenu])

  return (
    <div className="relative">
      <button
        ref={buttonRef}
        onClick={onToggleMenu}
        className="px-3 py-2 text-sm bg-white border border-gray-300 rounded-md shadow-sm hover:bg-gray-50 focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
      >
        Columns ▼
      </button>

      {showMenu && (
        <div
          ref={menuRef}
          className="fixed w-48 bg-white rounded-md shadow-lg border border-gray-200 z-50"
          style={{
            top: `${menuPosition.top}px`,
            left: `${menuPosition.left}px`
          }}
        >
          <div className="py-1">
            {Object.entries(columnVisibility).map(([column, visible]) => (
              <label
                key={column}
                className="flex items-center px-4 py-2 hover:bg-gray-50 cursor-pointer"
              >
                <input
                  type="checkbox"
                  checked={visible}
                  onChange={() => { onToggleColumn(column as keyof ColumnVisibility) }}
                  className="mr-2 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                />
                <span className="text-sm text-gray-700 capitalize">
                  {formatColumnName(column)}
                </span>
              </label>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}