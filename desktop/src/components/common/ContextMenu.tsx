import { useEffect, useRef, useState, type React } from "react";
import { cn } from "@/lib/utils";

export interface ContextMenuItem {
  label: string;
  onSelect: () => void;
  destructive?: boolean;
  disabled?: boolean;
  separatorBefore?: boolean;
}

export interface ContextMenuProps {
  items: ContextMenuItem[];
  children: React.ReactNode;
}

/**
 * Right-click menu rendered inline at the cursor. It closes on outside click,
 * Escape or after a selection, without needing a portal.
 */
export function ContextMenu({ items, children }: ContextMenuProps): React.JSX.Element {
  const [open, setOpen] = useState(false);
  const [position, setPosition] = useState({ x: 0, y: 0 });
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;

    const onPointerDown = (event: MouseEvent) => {
      if (!menuRef.current?.contains(event.target as Node)) setOpen(false);
    };
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") setOpen(false);
    };
    const onResize = () => setOpen(false);

    window.addEventListener("mousedown", onPointerDown);
    window.addEventListener("keydown", onKeyDown);
    window.addEventListener("resize", onResize);

    return () => {
      window.removeEventListener("mousedown", onPointerDown);
      window.removeEventListener("keydown", onKeyDown);
      window.removeEventListener("resize", onResize);
    };
  }, [open]);

  return (
    <div
      className="relative"
      onContextMenu={(event) => {
        event.preventDefault();
        setPosition({ x: event.clientX, y: event.clientY });
        setOpen(true);
      }}
    >
      {children}

      {open ? (
        <div
          ref={menuRef}
          role="menu"
          style={{ left: position.x, top: position.y }}
          className="fixed z-[60] min-w-[11rem] rounded-md border bg-popover p-1 text-popover-foreground shadow-lg"
        >
          {items.map((item) => (
            <div key={item.label}>
              {item.separatorBefore ? <div className="my-1 h-px bg-border" /> : null}
              <button
                type="button"
                role="menuitem"
                disabled={item.disabled}
                onClick={() => {
                  setOpen(false);
                  item.onSelect();
                }}
                className={cn(
                  "flex w-full items-center rounded-sm px-2 py-1.5 text-left text-sm transition-colors",
                  "hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50",
                  item.destructive && "text-destructive hover:bg-destructive/10",
                )}
              >
                {item.label}
              </button>
            </div>
          ))}
        </div>
      ) : null}
    </div>
  );
}