const BYTE_UNITS = ['B', 'KB', 'MB', 'GB', 'TB'];

export function formatBytes(bytes: number): string {
  if (!bytes || bytes <= 0) return '0 B';
  const i = Math.min(BYTE_UNITS.length - 1, Math.floor(Math.log(bytes) / Math.log(1024)));
  const value = bytes / Math.pow(1024, i);
  const decimals = i === 0 || value >= 100 ? 0 : value >= 10 ? 1 : 2;
  return `${value.toFixed(decimals)} ${BYTE_UNITS[i]}`;
}

export function formatSpeed(mbPerSecond: number): string {
  return `${formatBytes(mbPerSecond * 1024 * 1024)}/s`;
}

/** Compact duration: "45s", "3m 05s", "1h 02m". */
export function formatDuration(totalSeconds: number): string {
  const s = Math.max(0, Math.round(totalSeconds));
  if (s < 60) return `${s}s`;
  const h = Math.floor(s / 3600);
  const m = Math.floor((s % 3600) / 60);
  const sec = s % 60;
  const pad = (n: number) => String(n).padStart(2, '0');
  return h > 0 ? `${h}h ${pad(m)}m` : `${m}m ${pad(sec)}s`;
}

/** Shortens a long hex digest for display: "3fa9c1…0b27e4". */
export function shortHash(hash: string, edge = 6): string {
  return hash.length > edge * 2 + 1 ? `${hash.slice(0, edge)}…${hash.slice(-edge)}` : hash;
}

/** Extracts the file extension in upper case ("PDF"), or an empty string. */
export function fileExtension(name: string): string {
  const dot = name.lastIndexOf('.');
  return dot > 0 && dot < name.length - 1 ? name.slice(dot + 1).toUpperCase() : '';
}
