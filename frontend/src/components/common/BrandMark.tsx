import logo from '../../assets/photos/magpie.webp';
import mark from '../../assets/photos/magpie-pairing.webp';

interface BrandMarkProps {
  size?: 'sm' | 'lg';
}

/** The FTU magpie with the "File Throw" wordmark. */
export function BrandMark({ size = 'sm' }: BrandMarkProps) {
  if (size === 'lg') {
    return (
      <div className="flex flex-col items-center text-center">
        <img src={logo} alt="" width={720} height={443} className="h-auto w-[min(300px,56vw)]" />
        <h1 className="mt-2 text-base font-bold uppercase tracking-tight text-fg">File Throw</h1>
        <p className="mt-0.5 text-[11px] text-muted">Direct peer-to-peer file transfer. No servers in between.</p>
      </div>
    );
  }
  return (
    <div className="flex items-center gap-2">
      <img src={mark} alt="" width={359} height={360} className="size-7" />
      <span className="text-xs font-bold uppercase tracking-tight text-fg">File Throw</span>
    </div>
  );
}
