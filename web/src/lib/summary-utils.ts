export function cleanSummaryText(text: string): string {
  // Remove the generic intro line that appears at the start of AI summaries
  const cleanText = text
    .replace(/^Here'?s a concise summary of the Signal group conversation:\s*\n*/i, '')
    .replace(/^Here are the key points from the Signal group conversation:\s*\n*/i, '')
    .replace(/^Summary of the Signal group conversation:\s*\n*/i, '')
    .trim()

  return cleanText
}

export function processMarkdownContent(text: string): string {
  // Clean the text first
  const cleanText = cleanSummaryText(text)
  
  // For now, we'll return the cleaned text
  // Later we can add markdown processing if needed
  return cleanText
}
