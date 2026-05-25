interface SourceCountsProps {
  triggered: number;
  skipped: number;
  failed: number;
  total: number;
}

// SourceCounts renders the colored "5 / 1 / 0 of 6" tally shown in the
// Runs table. Each segment carries its semantic color (green/grey/red)
// and a tooltip that explains the bucket.
export function SourceCounts({ triggered, skipped, failed, total }: SourceCountsProps) {
  return (
    <span className="source-counts">
      <span className="triggered" title={`${triggered} triggered`}>
        {triggered}
      </span>
      <span className="sep">/</span>
      <span className="skipped" title={`${skipped} skipped`}>
        {skipped}
      </span>
      <span className="sep">/</span>
      <span
        className={`failed${failed === 0 ? " none" : ""}`}
        title={`${failed} failed`}
      >
        {failed}
      </span>
      <span className="total">of {total}</span>
    </span>
  );
}
