import type { ApiErrorBody, ApiErrorCode } from "@/types";
import { useSettingsStore } from "@/stores/settings.store";

export interface RequestOptions {
  method?: "GET" | "POST" | "PATCH" | "PUT" | "DELETE";
  body?: unknown;
  /** Skips the Authorization header, used by login/register/refresh. */
  anonymous?: boolean;
  /** Internal: prevents an infinite refresh recursion. */
  skipRefresh?: boolean;
  signal?: AbortSignal;
}

/** Error carrying the backend's `{ error: { code, message, fields } }` body. */
export class ApiError extends Error {
  readonly code: ApiErrorCode;
  readonly status: number;
  readonly fields: Record<string, string>;

  constructor(status: number, body: ApiErrorBody) {
    super(body.message || "Request failed");
    this.name = "ApiError";
    this.status = status;
    this.code = body.code;
    this.fields = body.fields ?? {};
  }
}

/**
 * The auth store registers a refresher here so the client never has to import
 * the store (which would create a cycle through the api modules).
 */
type Refresher = () => Promise<boolean>;

let refreshHandler: Refresher | null = null;
let refreshPromise: Promise<boolean> | null = null;

export function setRefreshHandler(handler: Refresher): void {
  refreshHandler = handler;
}

export function setApiBaseUrl(url: string): void {
  useSettingsStore.getState().setApiBaseUrl(url);
}

export function getApiBaseUrl(): string {
  return useSettingsStore.getState().apiBaseUrl;
}

function readToken(): string | null {
  try {
    const raw = localStorage.getItem("cordis.auth");
    if (!raw) return null;
    const parsed = JSON.parse(raw) as { accessToken?: string | null };
    return parsed.accessToken ?? null;
  } catch {
    return null;
  }
}

/** Normalises anything thrown by fetch into the contract's error shape. */
function toApiError(error: unknown): ApiError {
  if (error instanceof ApiError) return error;
  const message =
    error instanceof Error && error.message ? error.message : "cannot reach server";
  return new ApiError(0, { code: "network_error", message });
}

/**
 * Performs the refresh exactly once for concurrent 401s by sharing a single
 * in-flight promise, so a burst of failing requests triggers one rotation.
 */
async function refreshOnce(): Promise<boolean> {
  if (refreshPromise) return refreshPromise;
  if (!refreshHandler) return false;

  refreshPromise = refreshHandler()
    .catch(() => false)
    .finally(() => {
      refreshPromise = null;
    });

  return refreshPromise;
}

export async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const {
    method = "GET",
    body,
    anonymous = false,
    skipRefresh = false,
    signal,
  } = options;

  const headers: Record<string, string> = { Accept: "application/json" };
  let payload: BodyInit | undefined;

  if (body instanceof FormData) {
    // The browser must set the multipart boundary itself.
    payload = body;
  } else if (body !== undefined) {
    headers["Content-Type"] = "application/json";
    payload = JSON.stringify(body);
  }

  if (!anonymous) {
    const token = readToken();
    if (token) headers.Authorization = `Bearer ${token}`;
  }

  let response: Response;
  try {
    response = await fetch(`${getApiBaseUrl()}${path}`, {
      method,
      headers,
      body: payload,
      signal,
    });
  } catch (error) {
    if (error instanceof DOMException && error.name === "AbortError") throw error;
    throw new ApiError(0, {
      code: "network_error",
      message: `cannot reach server at ${getApiBaseUrl()}`,
    });
  }

  if (response.status === 204) {
    return undefined as T;
  }

  if (response.status === 401 && !anonymous && !skipRefresh) {
    const refreshed = await refreshOnce();
    if (refreshed) {
      return request<T>(path, { ...options, skipRefresh: true });
    }
  }

  const text = await response.text();
  let parsed: unknown = null;
  if (text) {
    try {
      parsed = JSON.parse(text);
    } catch {
      parsed = null;
    }
  }

  if (!response.ok) {
    const body = (parsed as { error?: ApiErrorBody } | null)?.error;
    throw newApiError(response.status, body, text);
  }

  return parsed as T;
}

/**
 * Falls back to a status-derived code when the backend answers with something
 * other than the documented envelope, for example an HTML proxy error page.
 */
function newApiError(status: number, body: ApiErrorBody | undefined, text: string): ApiError {
  if (body && body.code) return new ApiError(status, body);

  const fallback: Record<number, ApiErrorCode> = {
    400: "validation_error",
    401: "unauthorized",
    403: "forbidden",
    404: "not_found",
    409: "conflict",
    413: "payload_too_large",
    415: "unsupported_media_type",
    429: "rate_limited",
    500: "internal_error",
  };

  const message = text?.slice(0, 200) || `request failed with status ${status}`;
  return new ApiError(status, { code: fallback[status] ?? "internal_error", message });
}

export function get<T>(path: string, options: Omit<RequestOptions, "method" | "body"> = {}) {
  return request<T>(path, { ...options, method: "GET" });
}

export function post<T>(
  path: string,
  body?: unknown,
  options: Omit<RequestOptions, "method" | "body"> = {},
) {
  return request<T>(path, { ...options, method: "POST", body });
}

export function patch<T>(
  path: string,
  body?: unknown,
  options: Omit<RequestOptions, "method" | "body"> = {},
) {
  return request<T>(path, { ...options, method: "PATCH", body });
}

export function del<T>(path: string, options: Omit<RequestOptions, "method" | "body"> = {}) {
  return request<T>(path, { ...options, method: "DELETE" });
}

export { toApiError };