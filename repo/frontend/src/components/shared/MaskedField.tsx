import { useState } from 'react';
import { maskValue } from '../../utils/maskValue';

interface MaskedFieldProps {
  value: string;
  label?: string;
}

export function MaskedField({ value, label }: MaskedFieldProps) {
  const [revealed, setRevealed] = useState(false);

  if (!value) return null;

  return (
    <div className="flex items-center gap-2">
      {label && <span className="text-sm text-gray-500">{label}:</span>}
      <span className="text-sm font-mono">{revealed ? value : maskValue(value)}</span>
      <button
        onClick={() => setRevealed(!revealed)}
        className="text-xs text-primary-600 hover:underline"
      >
        {revealed ? 'Hide' : 'Reveal'}
      </button>
    </div>
  );
}
