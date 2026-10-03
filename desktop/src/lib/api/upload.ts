import { post } from "./client";
import type { Attachment } from "@/types";

export interface UploadOptions {
  width?: number;
  height?: number;
}

/** The backend expects the binary under the `file` field plus optional integers. */
export function uploadFile(
  file: File,
  options: UploadOptions = {},
): Promise<Attachment> {
  const form = new FormData();
  form.append("file", file);
  if (typeof options.width === "number") form.append("width", String(options.width));
  if (typeof options.height === "number") form.append("height", String(options.height));

  return post<Attachment>("/api/upload", form);
}

/** Turns a dropped/selected file into the AttachmentInput the send endpoint wants. */
export function toAttachmentInput(attachment: Attachment) {
  return {
    url: attachment.url,
    filename: attachment.filename,
    size: attachment.size,
    contentType: attachment.contentType,
    width: attachment.width ?? undefined,
    height: attachment.height ?? undefined,
  };
}