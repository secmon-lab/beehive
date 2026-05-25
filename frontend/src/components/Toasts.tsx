import { Check } from "lucide-react";
import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useState,
  type ReactNode,
} from "react";

interface Toast {
  id: string;
  msg: string;
  icon?: ReactNode;
}

interface ToastsContextValue {
  toasts: Toast[];
  push: (msg: string, opts?: { icon?: ReactNode; duration?: number }) => void;
}

const ToastsContext = createContext<ToastsContextValue | null>(null);

export function ToastsProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([]);

  const push = useCallback(
    (msg: string, opts?: { icon?: ReactNode; duration?: number }) => {
      const id = Math.random().toString(36).slice(2);
      setToasts((curr) => [...curr, { id, msg, icon: opts?.icon }]);
      const duration = opts?.duration ?? 2400;
      window.setTimeout(() => {
        setToasts((curr) => curr.filter((x) => x.id !== id));
      }, duration);
    },
    [],
  );

  const value = useMemo(() => ({ toasts, push }), [toasts, push]);

  return (
    <ToastsContext.Provider value={value}>
      {children}
      <ToastsViewport toasts={toasts} />
    </ToastsContext.Provider>
  );
}

export function useToasts(): ToastsContextValue {
  const ctx = useContext(ToastsContext);
  if (!ctx) {
    throw new Error("useToasts must be used within a ToastsProvider");
  }
  return ctx;
}

function ToastsViewport({ toasts }: { toasts: Toast[] }) {
  return (
    <div className="toasts" aria-live="polite">
      {toasts.map((t) => (
        <div className="toast" key={t.id}>
          {t.icon ?? <Check size={14} />}
          {t.msg}
        </div>
      ))}
    </div>
  );
}
