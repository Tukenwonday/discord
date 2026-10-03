import { useEffect, useState, type React } from "react";
import { Minus, Square, X, Download } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import {
  bindResizeListener,
  close,
  minimize,
  subscribeMaximized,
  toggleMaximize,
} from "@/lib/tauri/window";
import { checkForUpdates, installUpdate, isTauri } from "@/lib/tauri/desktop";
import { cn } from "@/lib/utils";

export interface TitlebarProps {
  title: string;
  subtitle?: string;
}

/** Frameless window chrome: drag region, server identity and window buttons. */
export function Titlebar({ title, subtitle }: TitlebarProps): React.JSX.Element {
  const [maximized, setMaximized] = useState(false);
  const [updateVersion, setUpdateVersion] = useState<string | null>(null);

  useEffect(() => {
    const unsubscribe = subscribeMaximized(setMaximized);
    let dispose: (() => void) | undefined;

    void bindResizeListener().then((unlisten) => {
      dispose = unlisten;
    });

    return () => {
      unsubscribe();
      dispose?.();
    };
  }, []);

  useEffect(() => {
    if (!isTauri()) return;
    let cancelled = false;

    void checkForUpdates().then((status) => {
      if (!cancelled && status.available && status.version) {
        setUpdateVersion(status.version);
      }
    });

    return () => {
      cancelled = true;
    };
  }, []);

  const windowControls = (
    <div className="no-drag flex items-center">
      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            variant="ghost"
            size="icon"
            aria-label="Minimise"
            className="h-9 w-11 rounded-none text-muted-foreground hover:bg-white/10 hover:text-foreground"
            onClick={minimize}
          >
            <Minus className="h-4 w-4" />
          </Button>
        </TooltipTrigger>
        <TooltipContent side="bottom">Minimise</TooltipContent>
      </Tooltip>

      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            variant="ghost"
            size="icon"
            aria-label={maximized ? "Restore" : "Maximise"}
            className="h-9 w-11 rounded-none text-muted-foreground hover:bg-white/10 hover:text-foreground"
            onClick={() => void toggleMaximize()}
          >
            <Square className="h-3 w-3" />
          </Button>
        </TooltipTrigger>
        <TooltipContent side="bottom">{maximized ? "Restore" : "Maximise"}</TooltipContent>
      </Tooltip>

      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            variant="ghost"
            size="icon"
            aria-label="Close"
            className="h-9 w-11 rounded-none text-muted-foreground hover:bg-red-600 hover:text-white"
            onClick={close}
          >
            <X className="h-4 w-4" />
          </Button>
        </TooltipTrigger>
        <TooltipContent side="bottom">Close</TooltipContent>
      </Tooltip>
    </div>
  );

  return (
    <header
      data-tauri-drag-region
      className="drag-region flex h-10 shrink-0 items-center justify-between border-b border-black/20 bg-cordis-rail pl-3"
    >
      <div className="flex min-w-0 items-center gap-2">
        <span className="truncate text-sm font-semibold text-foreground">{title}</span>
        {subtitle ? (
          <>
            <span className="text-muted-foreground">/</span>
            <span className="truncate text-sm text-muted-foreground">{subtitle}</span>
          </>
        ) : null}
      </div>

      {updateVersion ? (
        <button
          type="button"
          onClick={() => void installUpdate()}
          className={cn(
            "no-drag mr-2 flex items-center gap-1.5 rounded-full bg-cordis-brand px-2.5 py-0.5",
            "text-[11px] font-semibold text-white hover:bg-cordis-brand-hover",
          )}
        >
          <Download className="h-3 w-3" />
          Update to {updateVersion}
        </button>
      ) : null}

      {windowControls}
    </header>
  );
}