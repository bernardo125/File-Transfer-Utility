import logo from '../../assets/photos/magpie.webp';

interface BrandMarkProps {
  size?: 'sm' | 'lg';
}

/** The FTU magpie with the "File Throw" wordmark. */
export function BrandMark({ size = 'sm' }: BrandMarkProps) {
  if (size === 'lg') {
    return (
      <div className="flex flex-col items-center text-center">
        <img src={logo} alt="" width={720} height={443} className="w-[min(320px,60vw)] h-auto drop-shadow-[0_18px_30px_rgba(0,0,0,0.25)]" />
        <h1 className="mt-4 text-3xl font-extrabold tracking-tight text-fg">File Throw</h1>
        <p className="mt-1 text-sm text-muted">Direct peer-to-peer file transfer. No servers in between.</p>
      </div>
    );
  }
  return (
    <div className="flex items-center gap-2">
      <img src={logo} alt="" width={720} height={443} className="h-6 w-auto" />
      <span className="text-sm font-bold tracking-tight text-fg">File Throw</span>
    </div>
  );
}
