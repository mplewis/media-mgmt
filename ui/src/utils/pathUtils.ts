export const getDisplayPath = (fullPath: string, showRelativePaths: boolean): string => {
  if (showRelativePaths) {
    // Extract relative path from the full path
    const pathParts = fullPath.split('/')
    const inputDirIndex = pathParts.findIndex(part => 
      part.includes('jpguide') || part.includes('snw-season-1')
    )
    if (inputDirIndex !== -1) {
      return pathParts.slice(inputDirIndex).join('/')
    }
  }
  // Return just filename
  return fullPath.split('/').pop() ?? fullPath
}