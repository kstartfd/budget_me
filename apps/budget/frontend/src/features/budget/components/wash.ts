/**
 * Translucent tint of a category hue, per the design's wash() helper. The
 * alpha comes from the theme (--wash-alpha: 12% light / 16% dark) so tints
 * read the same in both modes.
 */
export function wash(hex: string): string {
  return `color-mix(in srgb, ${hex} var(--wash-alpha), transparent)`
}

/** Stronger tint for selected chip borders (design uses 50% alpha). */
export function washStrong(hex: string): string {
  return `color-mix(in srgb, ${hex} 50%, transparent)`
}
