import { post } from "./client";
import type { AuthResponse, RefreshResponse } from "@/types";

export interface RegisterInput {
  username: string;
  email: string;
  password: string;
  displayName?: string;
}

export interface LoginInput {
  login: string;
  password: string;
}

export function register(input: RegisterInput): Promise<AuthResponse> {
  return post<AuthResponse>("/api/auth/register", input, { anonymous: true });
}

export function login(input: LoginInput): Promise<AuthResponse> {
  return post<AuthResponse>("/api/auth/login", input, { anonymous: true });
}

/** Called by the API client through the registered refresh handler. */
export function refresh(refreshToken: string): Promise<RefreshResponse> {
  return post<RefreshResponse>("/api/auth/refresh", { refreshToken }, { anonymous: true });
}

export function logout(refreshToken: string): Promise<void> {
  return post<void>("/api/auth/logout", { refreshToken }, { anonymous: true });
}