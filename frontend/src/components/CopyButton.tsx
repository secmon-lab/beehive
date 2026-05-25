import { Check, Copy } from "lucide-react";
import { useState } from "react";
import { useT } from "../i18n";

interface CopyButtonProps {
  value: string;
  label?: string;
  ariaLabel?: string;
}

export function CopyButton({ value, label, ariaLabel }: CopyButtonProps) {
  const t = useT();
  const [copied, setCopied] = useState(false);
  const baseLabel = label ?? t("common.copy");
  const aria = ariaLabel ?? baseLabel;

  const onClick = () => {
    if (navigator.clipboard?.writeText) {
      navigator.clipboard.writeText(value).catch(() => undefined);
    }
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1400);
  };

  return (
    <button
      type="button"
      className="btn btn-sm btn-ghost"
      aria-label={aria}
      onClick={onClick}
    >
      {copied ? <Check size={12} /> : <Copy size={12} />}
      {copied ? t("common.copied") : baseLabel}
    </button>
  );
}
