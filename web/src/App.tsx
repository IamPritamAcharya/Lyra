import { useState, type FormEvent } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createTrack, deleteTrack, identify, listTracks, login, logout, uploadTrackAudio, type IdentifyResponse, type Track } from "./api";

type View = "identify" | "admin";

export function App() {
  const [view, setView] = useState<View>("identify");
  const [csrf, setCSRF] = useState<string | null>(null);
  return <div className="app-shell">
    <header className="site-header">
      <a className="brand" href="#top" aria-label="Lyra home" onClick={() => setView("identify")}>
        <img src="/brand/lyra-mark.svg" alt="" /><span>LYRA</span>
      </a>
      <nav className="site-nav" aria-label="Primary navigation">
        <button className={view === "identify" ? "nav-link is-active" : "nav-link"} onClick={() => setView("identify")}>Identify</button>
        <button className={view === "admin" ? "nav-link is-active" : "nav-link"} onClick={() => setView("admin")}>Catalog</button>
      </nav>
    </header>
    <main id="top" className="page">{view === "identify" ? <Identify /> : <Admin csrf={csrf} setCSRF={setCSRF} />}</main>
    <footer className="site-footer"><span>Lyra · acoustic landmark identification</span><span>Developed by Pritam · Query recordings are never retained.</span></footer>
  </div>;
}

function Identify() {
  const [file, setFile] = useState<File | null>(null);
  const mutation = useMutation({ mutationFn: identify });
  const submit = (event: FormEvent<HTMLFormElement>) => { event.preventDefault(); if (file) mutation.mutate(file); };
  return <>
    <section className="hero" aria-labelledby="identify-title">
      <div className="hero-copy"><p className="kicker"><i /> LANDMARK RECOGNITION</p><h1 id="identify-title">Find the music<br /><em>in the noise.</em></h1><p>Upload a short recording. Lyra compares its acoustic landmarks against your private reference catalog—fast, deterministic, and without retaining query audio.</p></div>
      <div className="hero-signal" aria-hidden="true"><SignalGraphic /></div>
    </section>
    <form className="identify-card" onSubmit={submit}>
      <label className={file ? "drop-zone has-file" : "drop-zone"}>
        <input accept="audio/*" type="file" onChange={(event) => { setFile(event.target.files?.[0] ?? null); mutation.reset(); }} />
        <span className="drop-icon">⌁</span><span className="drop-title">{file ? file.name : "Choose an audio recording"}</span><span className="drop-copy">{file ? `${formatBytes(file.size)} · ready to identify` : "MP3, WAV, AAC and other FFmpeg-supported audio"}</span>
      </label>
      <div className="identify-actions"><p><strong>Best results:</strong> a clear 5–10 second excerpt from an indexed reference.</p><button className="button button-primary" disabled={!file || mutation.isPending} type="submit">{mutation.isPending ? <><Spinner /> Reading landmarks…</> : <>Identify recording <Arrow /></>}</button></div>
    </form>
    {mutation.isError && <Notice kind="error" title="Could not process that recording">Try another supported audio file under 10 MB.</Notice>}
    {mutation.data && <Result response={mutation.data} />}
    <section className="trust-row" aria-label="Lyra principles"><TrustItem icon="◈" title="Private by default" text="Query audio is processed temporarily, then removed." /><TrustItem icon="⌁" title="Deterministic" text="Acoustic landmarks, not opaque model guesses." /><TrustItem icon="↗" title="Built for excerpts" text="Designed for short recordings of indexed audio." /></section>
  </>;
}

function Admin({ csrf, setCSRF }: { csrf: string | null; setCSRF: (value: string | null) => void }) {
  if (!csrf) return <Login setCSRF={setCSRF} />;
  return <Catalog csrf={csrf} signOut={() => { void logout().finally(() => setCSRF(null)); }} />;
}

function Login({ setCSRF }: { setCSRF: (value: string) => void }) {
  const [username, setUsername] = useState("admin"); const [password, setPassword] = useState("");
  const mutation = useMutation({ mutationFn: () => login(username, password), onSuccess: (session) => setCSRF(session.csrf_token) });
  return <section className="admin-intro"><p className="kicker"><i /> PRIVATE WORKSPACE</p><h1>Reference<br /><em>catalog.</em></h1><form className="auth-card" onSubmit={(event) => { event.preventDefault(); mutation.mutate(); }}><div><h2>Admin sign in</h2><p>Manage the private recordings Lyra can identify.</p></div><label>Username<input value={username} autoComplete="username" onChange={(event) => setUsername(event.target.value)} required /></label><label>Password<input type="password" value={password} autoComplete="current-password" onChange={(event) => setPassword(event.target.value)} required /></label><button className="button button-primary" disabled={mutation.isPending} type="submit">{mutation.isPending ? <><Spinner /> Signing in…</> : <>Sign in <Arrow /></>}</button>{mutation.isError && <p className="field-error">{loginErrorMessage(mutation.error)}</p>}<small>Sessions are HttpOnly, server-side, and expire automatically.</small></form></section>;
}

function Catalog({ csrf, signOut }: { csrf: string; signOut: () => void }) {
  const client = useQueryClient();
  const [title, setTitle] = useState(""); const [artist, setArtist] = useState(""); const [album, setAlbum] = useState(""); const [file, setFile] = useState<File | null>(null); const [created, setCreated] = useState<Track | null>(null);
  const tracks = useQuery({ queryKey: ["tracks"], queryFn: listTracks, retry: false });
  const create = useMutation({ mutationFn: () => createTrack(csrf, title, artist, album), onSuccess: (track) => { setCreated(track); setTitle(""); setArtist(""); setAlbum(""); client.invalidateQueries({ queryKey: ["tracks"] }); } });
  const upload = useMutation({ mutationFn: () => uploadTrackAudio(csrf, created!.PublicID, file!), onSuccess: () => { setFile(null); client.invalidateQueries({ queryKey: ["tracks"] }); } });
  const remove = useMutation({ mutationFn: (id: string) => deleteTrack(csrf, id), onSuccess: () => client.invalidateQueries({ queryKey: ["tracks"] }) });
  const readyCount = tracks.data?.filter((track) => track.Status === "READY").length ?? 0;
  return <section className="catalog-page"><div className="catalog-head"><div><p className="kicker"><i /> PRIVATE WORKSPACE</p><h1>Reference<br /><em>catalog.</em></h1><p>Only tracks in this catalog can be identified by Lyra.</p></div><button className="button button-quiet" onClick={signOut}>Sign out</button></div>
    <div className="catalog-stats"><Stat value={tracks.data?.length ?? "—"} label="Catalog tracks" /><Stat value={readyCount} label="Ready to identify" /><Stat value="v1" label="Fingerprint algorithm" /></div>
    <div className="catalog-grid"><form className="panel form-panel" onSubmit={(event) => { event.preventDefault(); create.mutate(); }}><div className="panel-heading"><span className="step-number">01</span><div><h2>Create a reference</h2><p>Add metadata before uploading the source audio.</p></div></div><label>Track title<input value={title} onChange={(event) => setTitle(event.target.value)} placeholder="e.g. Night Drive" required /></label><label>Artist<input value={artist} onChange={(event) => setArtist(event.target.value)} placeholder="e.g. Lyra Studio" required /></label><label>Album <span>(optional)</span><input value={album} onChange={(event) => setAlbum(event.target.value)} placeholder="e.g. Signals" /></label><button className="button button-primary" disabled={create.isPending} type="submit">{create.isPending ? <><Spinner /> Creating…</> : <>Create reference <Arrow /></>}</button>{create.isError && <p className="field-error">Could not create the reference. Your session may have expired.</p>}</form>
      <section className={created ? "panel upload-panel is-ready" : "panel upload-panel"}><div className="panel-heading"><span className="step-number">02</span><div><h2>Index source audio</h2><p>{created ? `${created.Title} · ${created.ArtistName}` : "Create a reference first to upload audio."}</p></div></div>{created ? <form className="stack" onSubmit={(event) => { event.preventDefault(); if (file) upload.mutate(); }}><label className={file ? "file-picker has-file" : "file-picker"}><input accept="audio/*" type="file" onChange={(event) => { setFile(event.target.files?.[0] ?? null); upload.reset(); }} /><span>{file ? file.name : "Choose full reference audio"}</span><small>{file ? formatBytes(file.size) : "This audio stays private in object storage."}</small></label><button className="button button-aqua" disabled={!file || upload.isPending} type="submit">{upload.isPending ? <><Spinner /> Uploading…</> : <>Upload and index <Arrow /></>}</button>{upload.isSuccess && <Notice kind="success" title="Reference accepted">The indexing worker will make it ready shortly. Refresh the catalog to check status.</Notice>}{upload.isError && <p className="field-error">Upload failed. Please try again.</p>}</form> : <p className="empty-step">Waiting for reference metadata.</p>}</section></div>
    <section className="panel tracks-panel"><div className="tracks-heading"><div><p className="kicker"><i /> CATALOG STATUS</p><h2>Your tracks</h2></div><button className="button button-quiet" onClick={() => tracks.refetch()}>Refresh <span className="refresh-symbol">↻</span></button></div>{tracks.isLoading && <p className="loading-copy">Loading your catalog…</p>}{tracks.isError && <Notice kind="error" title="Could not load catalog">Your session may have expired. Sign in again and retry.</Notice>}{!tracks.isLoading && tracks.data?.length === 0 && <div className="empty-catalog"><span>⌁</span><h3>No references yet</h3><p>Create your first reference above, then upload its complete source audio.</p></div>}<div className="track-list">{tracks.data?.map((track) => <article className="track" key={track.PublicID}><div className="track-art" aria-hidden="true">♫</div><div className="track-meta"><strong>{track.Title}</strong><p>{track.ArtistName}{track.AlbumName ? ` · ${track.AlbumName}` : ""}</p><span className="track-id">{track.PublicID.slice(0, 8)}</span></div><StatusBadge status={track.Status} reason={track.FailureReason} /><button className="icon-button" aria-label={`Delete ${track.Title}`} disabled={remove.isPending} onClick={() => { if (window.confirm(`Delete ${track.Title}? This cannot be undone.`)) remove.mutate(track.PublicID); }}>×</button></article>)}</div></section>
  </section>;
}

function Result({ response }: { response: IdentifyResponse }) {
  if (response.reason === "insufficient_audio_signal") return <Notice kind="warning" title="Not enough usable signal">Try a clearer recording with more music and less silence.</Notice>;
  if (!response.matched || !response.match) return <section className="result-card no-match"><span className="result-icon">⌁</span><div><p className="kicker"><i /> NO CONFIDENT MATCH</p><h2>Nothing matched this recording.</h2><p>Lyra only identifies audio that has been added to your private reference catalog.</p><small>Processed in {response.processing_ms} ms</small></div></section>;
  const { match } = response;
  return <section className="result-card matched"><span className="result-icon">✓</span><div><p className="kicker"><i /> MATCH FOUND</p><h2>{match.title}</h2><p>{match.artist}{match.album ? ` · ${match.album}` : ""}</p><small>Reference offset {formatTime(match.reference_offset_ms)} · processed in {response.processing_ms} ms</small></div><div className="match-evidence"><span>Evidence</span><strong>{match.match_strength === "timing_aligned" ? "Timing aligned" : "Match found"}</strong></div></section>;
}

function StatusBadge({ status, reason }: { status: string; reason: string | null }) { const tone = status === "READY" ? "ready" : status === "FAILED" ? "failed" : "pending"; return <div className={`status status-${tone}`} title={reason ?? undefined}><i /> {status}</div>; }
function Notice({ kind, title, children }: { kind: "error" | "warning" | "success"; title: string; children: string }) { return <section className={`notice notice-${kind}`} role={kind === "error" ? "alert" : undefined}><span>{kind === "success" ? "✓" : "!"}</span><div><strong>{title}</strong><p>{children}</p></div></section>; }
function TrustItem({ icon, title, text }: { icon: string; title: string; text: string }) { return <article><span>{icon}</span><div><strong>{title}</strong><p>{text}</p></div></article>; }
function Stat({ value, label }: { value: string | number; label: string }) { return <div><strong>{value}</strong><span>{label}</span></div>; }
function Spinner() { return <i className="spinner" aria-hidden="true" />; }
function Arrow() { return <span aria-hidden="true">→</span>; }
function formatBytes(bytes: number) { return `${(bytes / (1024 * 1024)).toFixed(bytes < 1024 * 1024 ? 2 : 1)} MB`; }
function formatTime(milliseconds: number) { const seconds = Math.max(0, Math.round(milliseconds / 1000)); return `${Math.floor(seconds / 60)}:${String(seconds % 60).padStart(2, "0")}`; }
function loginErrorMessage(error: Error) { if (error.message === "invalid_credentials") return "Invalid username or password."; if (error.message === "rate_limited") return "Too many sign-in attempts. Wait one minute, then try again."; return "Sign-in is temporarily unavailable. Confirm that the API has started and migrations completed."; }
function SignalGraphic() { return <svg viewBox="0 0 360 250" fill="none"><defs><linearGradient id="signal" x1="0" y1="0" x2="1" y2="1"><stop stopColor="#A5B4FC"/><stop offset=".55" stopColor="#C084FC"/><stop offset="1" stopColor="#67E8F9"/></linearGradient></defs>{[38, 73, 108, 143, 178, 213].map((y, index) => <path key={y} d={`M0 ${y} C 55 ${y - 25 + index * 3}, 85 ${y + 30 - index * 4}, 130 ${y} S 215 ${y - 28 + index * 3}, 280 ${y} S 328 ${y + 15}, 360 ${y - 2}`} stroke="url(#signal)" strokeOpacity={0.22 + index * 0.09} strokeWidth="1.5" />)}<circle cx="106" cy="108" r="5" fill="#67E8F9"/><circle cx="211" cy="72" r="5" fill="#C084FC"/><circle cx="278" cy="178" r="5" fill="#A5B4FC"/></svg>; }
