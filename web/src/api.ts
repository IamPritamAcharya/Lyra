const apiBaseURL = import.meta.env.VITE_LYRA_API_BASE_URL ?? "http://localhost:8080";

export type IdentifyResponse = {
  request_id: string;
  matched: boolean;
  reason?: "insufficient_audio_signal";
  match: null | {
    track_id: string;
    title: string;
    artist: string;
    album: string | null;
    confidence: number;
    match_strength: "timing_aligned";
    reference_offset_ms: number;
  };
  processing_ms: number;
};

export type AdminSession = { username: string; csrf_token: string; expires_at: string };
export type Track = {
  PublicID: string;
  Title: string;
  ArtistName: string;
  AlbumName: string | null;
  Status: string;
  FailureReason: string | null;
};

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const response = await fetch(`${apiBaseURL}${path}`, { credentials: "include", ...init });
  if (!response.ok) {
    const body = await response.json().catch(() => null) as { error?: string } | null;
    throw new Error(body?.error ?? "request_failed");
  }
  return response.json() as Promise<T>;
}

export async function identify(file: File): Promise<IdentifyResponse> {
  const form = new FormData();
  form.append("audio", file);
  return request<IdentifyResponse>("/v1/identify", { method: "POST", body: form });
}

export function login(username: string, password: string): Promise<AdminSession> {
  return request<AdminSession>("/v1/admin/auth/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username, password }),
  });
}

export async function logout(): Promise<void> {
  const response = await fetch(`${apiBaseURL}/v1/admin/auth/logout`, { method: "POST", credentials: "include" });
  if (!response.ok) throw new Error("logout_failed");
}

function adminHeaders(csrf: string): HeadersInit {
  return { "X-CSRF-Token": csrf };
}

export function listTracks(): Promise<Track[]> {
  return request<Track[]>("/v1/admin/tracks?limit=50&offset=0");
}

export function createTrack(csrf: string, title: string, artist: string, album: string): Promise<Track> {
  return request<Track>("/v1/admin/tracks", {
    method: "POST",
    headers: { ...adminHeaders(csrf), "Content-Type": "application/json" },
    body: JSON.stringify({ title, artist, album: album || null }),
  });
}

export function uploadTrackAudio(csrf: string, trackID: string, file: File): Promise<Track> {
  const form = new FormData();
  form.append("audio", file);
  return request<Track>(`/v1/admin/tracks/${trackID}/audio`, { method: "POST", headers: adminHeaders(csrf), body: form });
}

export function deleteTrack(csrf: string, trackID: string): Promise<void> {
  return fetch(`${apiBaseURL}/v1/admin/tracks/${trackID}`, { method: "DELETE", credentials: "include", headers: adminHeaders(csrf) }).then(async (response) => {
    if (!response.ok) throw new Error("delete_failed");
  });
}
