export const getDisplayPath = (fullPath: string, showRelativePaths: boolean, inputDir?: string): string => {
  if (showRelativePaths && inputDir) {
    // Calculate relative path from input directory
    if (fullPath.startsWith(inputDir + '/')) {
      return fullPath.substring(inputDir.length + 1)
    }
  }
  // Return just filename
  return fullPath.split('/').pop() ?? fullPath
}