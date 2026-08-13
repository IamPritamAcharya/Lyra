import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createTrack, deleteTrack, identify, listTracks, login, logout, uploadTrackAudio, type IdentifyResponse, type Track } from "./api";

export function App() {
  const [view, setView] = useState<"identify" | "admin">("identify");
  const [csrf, setCSRF] = useState<string | null>(null);
  return <main className="page">
    <header><p className="eyebrow">LYRA</p><h1>{view === "identify" ? "Identify a song from a recording." : "Reference catalog."}</h1><p className="intro">{view === "identify" ? "Upload a short audio clip. Lyra compares its acoustic landmarks against indexed reference tracks." : "Sign in to add, index, and manage the reference recordings Lyra can identify."}</p>
      <nav><button className={view === "identify" ? "tab active" : "tab"} onClick={() => setView("identify")}>Identify</button><button className={view === "admin" ? "tab active" : "tab"} onClick={() => setView("admin")}>Admin</button></nav>
    </header>
    {view === "identify" ? <Identify /> : <Admin csrf={csrf} setCSRF={setCSRF} />}
  </main>;
}

function Identify() {
  const [file, setFile] = useState<File | null>(null);
  const mutation = useMutation({ mutationFn: identify });
  return <>
    <form className="card" onSubmit={(event) => { event.preventDefault(); if (file) mutation.mutate(file); }}>
      <label className="upload"><span>Audio recording</span><input accept="audio/*" type="file" onChange={(event) => setFile(event.target.files?.[0] ?? null)} /><small>{file ? file.name : "Choose a 2–20 second audio clip"}</small></label>
      <button disabled={!file || mutation.isPending} type="submit">{mutation.isPending ? "Identifying…" : "Identify song"}</button>
    </form>
    {mutation.isError && <p className="error" role="alert">Lyra could not process that recording. Please try another audio file.</p>}
    {mutation.data && <Result response={mutation.data} />}
    <p className="privacy">Your query audio is processed temporarily and is not retained.</p>
  </>;
}

function Admin({ csrf, setCSRF }: { csrf: string | null; setCSRF: (value: string | null) => void }) {
  if (!csrf) return <Login setCSRF={setCSRF} />;
  return <Catalog csrf={csrf} logout={() => { void logout().finally(() => setCSRF(null)); }} />;
}

function Login({ setCSRF }: { setCSRF: (value: string) => void }) {
  const [username, setUsername] = useState("admin");
  const [password, setPassword] = useState("");
  const mutation = useMutation({ mutationFn: () => login(username, password), onSuccess: (session) => setCSRF(session.csrf_token) });
  return <form className="card stack" onSubmit={(event) => { event.preventDefault(); mutation.mutate(); }}>
    <h2>Admin sign in</h2><p className="muted">This account manages private reference audio. Your password is never stored in the browser.</p>
    <label>Username<input value={username} autoComplete="username" onChange={(event) => setUsername(event.target.value)} required /></label>
    <label>Password<input type="password" value={password} autoComplete="current-password" onChange={(event) => setPassword(event.target.value)} required /></label>
    <button disabled={mutation.isPending} type="submit">{mutation.isPending ? "Signing in…" : "Sign in"}</button>
    {mutation.isError && <p className="error">{loginErrorMessage(mutation.error)}</p>}
  </form>;
}

function loginErrorMessage(error: Error) {
  if (error.message === "invalid_credentials") return "Invalid username or password.";
  if (error.message === "rate_limited") return "Too many sign-in attempts. Wait one minute, then try again.";
  return "Sign-in is temporarily unavailable. Confirm that the API has started and migrations completed.";
}

function Catalog({ csrf, logout: signOut }: { csrf: string; logout: () => void }) {
  const client = useQueryClient();
  const [title, setTitle] = useState(""); const [artist, setArtist] = useState(""); const [album, setAlbum] = useState(""); const [file, setFile] = useState<File | null>(null); const [created, setCreated] = useState<Track | null>(null);
  const tracks = useQuery({ queryKey: ["tracks"], queryFn: listTracks, retry: false });
  const create = useMutation({ mutationFn: () => createTrack(csrf, title, artist, album), onSuccess: (track) => { setCreated(track); client.invalidateQueries({ queryKey: ["tracks"] }); } });
  const upload = useMutation({ mutationFn: () => uploadTrackAudio(csrf, created!.PublicID, file!), onSuccess: () => { setFile(null); client.invalidateQueries({ queryKey: ["tracks"] }); } });
  const remove = useMutation({ mutationFn: (id: string) => deleteTrack(csrf, id), onSuccess: () => client.invalidateQueries({ queryKey: ["tracks"] }) });
  return <section className="admin">
    <div className="admin-head"><p className="eyebrow">ADMIN SESSION</p><button className="secondary" onClick={signOut}>Sign out</button></div>
    <form className="card stack" onSubmit={(event) => { event.preventDefault(); create.mutate(); }}><h2>Create reference track</h2><label>Title<input value={title} onChange={(event) => setTitle(event.target.value)} required /></label><label>Artist<input value={artist} onChange={(event) => setArtist(event.target.value)} required /></label><label>Album <span className="muted">(optional)</span><input value={album} onChange={(event) => setAlbum(event.target.value)} /></label><button disabled={create.isPending} type="submit">{create.isPending ? "Creating…" : "Create track"}</button>{create.isError && <p className="error">Could not create track. Please sign in again.</p>}</form>
    {created && <form className="card stack" onSubmit={(event) => { event.preventDefault(); if (file) upload.mutate(); }}><h2>Upload reference audio</h2><p className="muted">{created.Title} · {created.ArtistName}</p><label className="upload"><span>Audio file</span><input accept="audio/*" type="file" onChange={(event) => setFile(event.target.files?.[0] ?? null)} /><small>{file?.name ?? "Choose the full reference recording"}</small></label><button disabled={!file || upload.isPending} type="submit">{upload.isPending ? "Uploading…" : "Upload and queue indexing"}</button>{upload.isSuccess && <p className="success">Uploaded. A worker will index this reference track.</p>}{upload.isError && <p className="error">Upload failed. Please sign in again and retry.</p>}</form>}
    <section className="card"><div className="admin-head"><h2>Tracks</h2><button className="secondary" onClick={() => tracks.refetch()}>Refresh</button></div>{tracks.isLoading && <p className="muted">Loading catalog…</p>}{tracks.isError && <p className="error">Could not load tracks. Your session may have expired.</p>}<div className="track-list">{tracks.data?.map((track) => <article className="track" key={track.PublicID}><div><strong>{track.Title}</strong><p>{track.ArtistName}{track.AlbumName ? ` · ${track.AlbumName}` : ""}</p><small>{track.Status}{track.FailureReason ? ` · ${track.FailureReason}` : ""}</small></div><button className="danger" disabled={remove.isPending} onClick={() => { if (window.confirm(`Delete ${track.Title}?`)) remove.mutate(track.PublicID); }}>Delete</button></article>)}</div></section>
  </section>;
}

function Result({ response }: { response: IdentifyResponse }) {
  if (response.reason === "insufficient_audio_signal") return <section className="card result"><h2>Not enough usable audio</h2><p>Try a clearer recording with more music and less silence.</p></section>;
  if (!response.matched || !response.match) return <section className="card result"><h2>No match found</h2><p>Lyra could not confidently identify an indexed track from this recording.</p><small>Processed in {response.processing_ms} ms</small></section>;
  const { match } = response;
  return <section className="card result"><p className="eyebrow">MATCH FOUND</p><h2>{match.title}</h2><p>{match.artist}{match.album ? ` · ${match.album}` : ""}</p><small>Processed in {response.processing_ms} ms · request {response.request_id}</small></section>;
}
