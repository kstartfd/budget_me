/** The longest side a photo is sent at: enough for print, far below a phone camera's output. */
const maxSide = 2000

/**
 * Shrinks a photo and re-encodes it as JPEG before it is sent. A phone camera
 * file is several megabytes and the reader needs a fraction of that; this also
 * turns whatever the browser can open (HEIC on Safari, WebP, PNG) into one
 * format the backend accepts. Falls back to the file as it is when the browser
 * cannot decode it, and the backend then says what it thinks of it.
 */
export async function shrinkImage(file: File): Promise<Blob> {
  try {
    const bitmap = await createImageBitmap(file, {
      imageOrientation: "from-image",
    })
    const scale = Math.min(1, maxSide / Math.max(bitmap.width, bitmap.height))
    const width = Math.round(bitmap.width * scale)
    const height = Math.round(bitmap.height * scale)
    const canvas = document.createElement("canvas")
    canvas.width = width
    canvas.height = height

    const context = canvas.getContext("2d")
    if (!context) return file

    // JPEG has no transparency: a transparent PNG would turn black.
    context.fillStyle = "#fff"
    context.fillRect(0, 0, width, height)
    context.drawImage(bitmap, 0, 0, width, height)
    bitmap.close()

    const blob = await new Promise<Blob | null>((resolve) =>
      canvas.toBlob(resolve, "image/jpeg", 0.85),
    )

    return blob ?? file
  } catch {
    return file
  }
}
