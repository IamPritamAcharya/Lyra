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
    reference_offset_ms: number;
  };
  processing_ms: number;
};

export async function identify(file: File): Promise<IdentifyResponse> {
  const form = new FormData();
  form.append("audio", file);
  const response = await fetch(`${apiBaseURL}/v1/identify`, { method: "POST", body: form });
  if (!response.ok) throw new Error("Lyra could not process that recording. Please try another audio file.");
  return response.json() as Promise<IdentifyResponse>;
}
