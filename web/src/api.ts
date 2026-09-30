// Typed client for the Go JSON API (GET/POST/PUT/DELETE /api/v1/...).

export type Preset = 'static' | 'spa' | 'proxy' | 'php';
export type TlsMode = 'auto' | 'off';

export interface Backend {
  service?: string;
  image: string;
  internalPort?: number;
  env?: Record<string, string>;
}

export interface Site {
  id: string;
  version: number;
  domains: string[];
  preset: Preset;
  tls: TlsMode;
  log: boolean;
  root?: string;
  backend?: Backend;
}

export interface SiteSummary {
  id: string;
  domains: string[];
  preset: Preset;
}

export interface ProjectInfo {
  name: string;
  root: string;
  version: number;
  email: string;
  sites: SiteSummary[];
  caddyFound: boolean;
}

export interface FieldError {
  file: string;
  field: string;
  reason: string;
}

export interface Warning {
  file: string;
  field: string;
  reason: string;
}

export interface PreviewResult {
  caddyfile: string;
  compose: string;
  formatted: boolean;
  warnings: Warning[];
}

export interface StatusResult {
  changedOnDisk: string[];
  caddyFound: boolean;
  generatedStale: boolean;
}

export class ApiError extends Error {
  status: number;
  errors: FieldError[];
  constructor(status: number, errors: FieldError[], fallback: string) {
    super(errors.length > 0 ? errors.map((e) => `${e.field}: ${e.reason}`).join('; ') : fallback);
    this.status = status;
    this.errors = errors;
  }
}

async function req<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...init,
    headers: { 'Content-Type': 'application/json', ...(init?.headers ?? {}) },
  });
  if (res.status === 204) return undefined as T;
  const body = await res.json().catch(() => ({}));
  if (!res.ok) {
    const errors = Array.isArray((body as { errors?: FieldError[] }).errors)
      ? (body as { errors: FieldError[] }).errors
      : [];
    const fallback =
      typeof (body as { error?: string }).error === 'string'
        ? (body as { error: string }).error
        : `request failed (${res.status})`;
    throw new ApiError(res.status, errors, fallback);
  }
  return body as T;
}

export const api = {
  getProject: () => req<ProjectInfo>('/api/v1/project'),
  updateProject: (patch: { name?: string; email?: string }) =>
    req<{ name: string; email: string; version: number }>('/api/v1/project', {
      method: 'PUT',
      body: JSON.stringify(patch),
    }),
  getSite: (id: string) => req<Site>(`/api/v1/sites/${encodeURIComponent(id)}`),
  getSnippet: (id: string) => req<{ snippet: string }>(`/api/v1/sites/${encodeURIComponent(id)}/snippet`),
  createSite: (site: Site) => req<Site>('/api/v1/sites', { method: 'POST', body: JSON.stringify(site) }),
  updateSite: (id: string, site: Site) =>
    req<Site>(`/api/v1/sites/${encodeURIComponent(id)}`, { method: 'PUT', body: JSON.stringify(site) }),
  deleteSite: (id: string) => req<void>(`/api/v1/sites/${encodeURIComponent(id)}`, { method: 'DELETE' }),
  getPreview: () => req<PreviewResult>('/api/v1/preview'),
  generate: () => req<{ caddyfilePath: string; composePath: string; formatted: boolean; warnings: Warning[] }>(
    '/api/v1/generate',
    { method: 'POST' },
  ),
  getStatus: () => req<StatusResult>('/api/v1/status'),
  refreshStatus: () => req<{ changedOnDisk: string[] }>('/api/v1/status/refresh', { method: 'POST' }),
};

export function fieldErrors(errors: FieldError[]): Map<string, string> {
  const m = new Map<string, string>();
  for (const e of errors) {
    if (!m.has(e.field)) m.set(e.field, e.reason);
  }
  return m;
}
