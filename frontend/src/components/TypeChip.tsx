interface TypeChipProps {
  type: string;
}

export function TypeChip({ type }: TypeChipProps) {
  return <span className={`type-chip type-chip-${type}`}>{type}</span>;
}
