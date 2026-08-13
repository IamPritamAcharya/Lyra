import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { identify, type IdentifyResponse } from "./api";

export function App() {
  const [file, setFile] = useState<File | null>(null);
  const mutation = useMutation({ mutationFn: identify });

  function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (file) mutation.mutate(file);
  }

  return (
    <main className="page">
      <header><p className="eyebrow">LYRA</p><h1>Identify a song from a recording.</h1><p className="intro">Upload a short audio clip. Lyra compares its acoustic landmarks against indexed reference tracks.</p></header>
      <form className="card" onSubmit={submit}>
        <label className="upload"><span>Audio recording</span><input accept="audio/*" type="file" onChange={(event) => setFile(event.target.files?.[0] ?? null)} /><small>{file ? file.name : "Choose a 2–20 second audio clip"}</small></label>
        <button disabled={!file || mutation.isPending} type="submit">{mutation.isPending ? "Identifying…" : "Identify song"}</button>
      </form>
      {mutation.isError && <p className="error" role="alert">{mutation.error.message}</p>}
      {mutation.data && <Result response={mutation.data} />}
      <p className="privacy">Your query audio is processed temporarily and is not retained.</p>
    </main>
  );
}

function Result({ response }: { response: IdentifyResponse }) {
  if (response.reason === "insufficient_audio_signal") return <section className="card result"><h2>Not enough usable audio</h2><p>Try a clearer recording with more music and less silence.</p></section>;
  if (!response.matched || !response.match) return <section className="card result"><h2>No match found</h2><p>Lyra could not confidently identify an indexed track from this recording.</p><small>Processed in {response.processing_ms} ms</small></section>;
  const { match } = response;
  return <section className="card result"><p className="eyebrow">MATCH FOUND</p><h2>{match.title}</h2><p>{match.artist}{match.album ? ` · ${match.album}` : ""}</p><small>Processed in {response.processing_ms} ms · request {response.request_id}</small></section>;
}
