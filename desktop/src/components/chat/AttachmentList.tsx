import type * as React from "react";

import { FileText, Download } from "lucide-react";
import { openExternal } from "@/lib/tauri/desktop";
import { formatBytes, isImageAttachment } from "@/lib/utils";
import type { Attachment } from "@/types";

export interface AttachmentListProps {
  attachments: Attachment[];
}

/** Renders images inline and everything else as a download card. */
export function AttachmentList({ attachments }: AttachmentListProps): React.JSX.Element | null {
  if (attachments.length === 0) return null;

  return (
    <div className="mt-2 flex flex-wrap gap-2">
      {attachments.map((attachment) => {
        if (isImageAttachment(attachment.contentType)) {
          return (
            <a
              key={attachment.id}
              href={attachment.url}
              target="_blank"
              rel="noreferrer"
              className="block max-w-sm overflow-hidden rounded-lg border border-white/10"
            >
              <img
                src={attachment.url}
                alt={attachment.filename}
                loading="lazy"
                className="max-h-80 w-auto object-contain"
              />
            </a>
          );
        }

        return (
          <button
            key={attachment.id}
            type="button"
            onClick={() => void openExternal(attachment.url)}
            className="flex max-w-sm items-center gap-3 rounded-lg border border-white/10 bg-black/20 px-3 py-2 text-left transition hover:bg-black/30"
          >
            <FileText className="h-8 w-8 shrink-0 text-cordis-brand" />
            <span className="min-w-0 flex-1">
              <span className="block truncate text-sm font-medium">{attachment.filename}</span>
              <span className="block text-xs text-muted-foreground">
                {formatBytes(attachment.size)}
              </span>
            </span>
            <Download className="h-4 w-4 shrink-0 text-muted-foreground" />
          </button>
        );
      })}
    </div>
  );
}