import { useEffect, useRef, useState, type CSSProperties, type FormEvent } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createTrack, deleteTrack, identify, listTracks, login, logout, uploadTrackAudio, type IdentifyResponse, type Track } from "./api";

type View = "identify" | "admin";

type LiveCapture = { context: AudioContext; source: MediaStreamAudioSourceNode; processor: ScriptProcessorNode };
type LiveResult = { run: number; response: IdentifyResponse };

export function App() {
  const [view, setView] = useState<View>("identify");
  const [csrf, setCSRF] = useState<string | null>(null);
  return <div className="app-shell">
    <header className="site-header">
      <a className="brand" href="#top" aria-label="Lyra home" onClick={() => setView("identify")}>
        <img src="/brand/lyra-mark.svg" alt="" /><span>LYRA</span>
      </a>
      <nav className="site-nav" aria-label="Primary navigation"><button className="nav-link" onClick={() => setView("admin")}>Private catalog <span>↗</span></button></nav>
    </header>
    <main id="top" className="page">{view === "identify" ? <Identify /> : <Admin csrf={csrf} setCSRF={setCSRF} />}</main>
    <footer className="site-footer"><span>Lyra · acoustic landmark identification</span><span>Developed by Pritam · Query recordings are never retained.</span></footer>
  </div>;
}

function Identify() {
  const [file, setFile] = useState<File | null>(null);
  const [isListening, setIsListening] = useState(false);
  const [listenSeconds, setListenSeconds] = useState(0);
  const [isCheckingLive, setIsCheckingLive] = useState(false);
  const [liveError, setLiveError] = useState<string | null>(null);
  const [liveResult, setLiveResult] = useState<LiveResult | null>(null);
  const [visibleLiveRun, setVisibleLiveRun] = useState(0);
  const [liveSearchExhausted, setLiveSearchExhausted] = useState(false);
  const [audioLevel, setAudioLevel] = useState(0);
  const mutation = useMutation({ mutationFn: (queryFile: File) => identify(queryFile) });
  const liveCapture = useRef<LiveCapture | null>(null);
  const finishListening = useRef<(() => void) | null>(null);
  const stream = useRef<MediaStream | null>(null);
  const stopTimer = useRef<number | null>(null);
  const elapsedTimer = useRef<number | null>(null);
  const firstCheckTimer = useRef<number | null>(null);
  const checkTimer = useRef<number | null>(null);
  const liveCheck = useRef<Promise<void> | null>(null);
  const listeningRun = useRef(0);
  const matchFound = useRef(false);
  const reachedListenLimit = useRef(false);
  const maximumListenSeconds = 15;
  const firstCheckMilliseconds = 5_000;
  const checkIntervalMilliseconds = 3_000;

  const clearListeningTimers = () => {
    if (stopTimer.current !== null) window.clearTimeout(stopTimer.current);
    if (elapsedTimer.current !== null) window.clearInterval(elapsedTimer.current);
    if (firstCheckTimer.current !== null) window.clearTimeout(firstCheckTimer.current);
    if (checkTimer.current !== null) window.clearInterval(checkTimer.current);
    stopTimer.current = null;
    elapsedTimer.current = null;
    firstCheckTimer.current = null;
    checkTimer.current = null;
  };
  const stopMicrophone = () => {
    stream.current?.getTracks().forEach((track) => track.stop());
    stream.current = null;
  };
  const stopListening = () => {
    finishListening.current?.();
  };
  const checkCapture = async (samples: Float32Array[], sampleRate: number, run: number, finalCheck: boolean, captureMilliseconds: number) => {
    if (liveCheck.current) {
      if (finalCheck) await liveCheck.current;
      else return;
    }
    if (run !== listeningRun.current || matchFound.current) return;
    const audio = encodeWAV(samples, sampleRate);
    if (audio.size === 0) return;
    const request = (async () => {
      setIsCheckingLive(true);
      try {
        const response = await identify(new File([audio], "lyra-live-query.wav", { type: audio.type }), captureMilliseconds);
        if (run !== listeningRun.current) return;
        if (response.matched) {
          matchFound.current = true;
          setLiveResult({ run, response });
          clearListeningTimers();
          stopListening();
        } else if (finalCheck) {
          setLiveResult({ run, response });
          setLiveSearchExhausted(reachedListenLimit.current);
        }
      } catch {
        if (run === listeningRun.current && finalCheck) setLiveError("Could not check the live recording. Try again or upload an audio file.");
      } finally {
        if (run === listeningRun.current) setIsCheckingLive(false);
      }
    })();
    liveCheck.current = request;
    await request;
    if (liveCheck.current === request) liveCheck.current = null;
  };
  const startListening = async () => {
    if (!navigator.mediaDevices?.getUserMedia || !window.AudioContext) {
      setLiveError("Live listening is not supported by this browser. Upload an audio file instead.");
      return;
    }
    setLiveError(null);
    mutation.reset();
    setLiveResult(null);
    setLiveSearchExhausted(false);
    listeningRun.current += 1;
    const run = listeningRun.current;
    setVisibleLiveRun(run);
    matchFound.current = false;
    reachedListenLimit.current = false;
    try {
      const microphone = await navigator.mediaDevices.getUserMedia({ audio: { autoGainControl: false, echoCancellation: false, noiseSuppression: false } });
      stream.current = microphone;
      const context = new AudioContext();
      const source = context.createMediaStreamSource(microphone);
      const processor = context.createScriptProcessor(4096, 1, 1);
      const samples: Float32Array[] = [];
      processor.onaudioprocess = (event) => {
        const chunk = new Float32Array(event.inputBuffer.getChannelData(0));
        samples.push(chunk);
        let power = 0;
        for (const sample of chunk) power += sample * sample;
        setAudioLevel(Math.min(1, Math.sqrt(power / chunk.length) * 8));
      };
      source.connect(processor);
      processor.connect(context.destination);
      liveCapture.current = { context, source, processor };
      await context.resume();
      const startedAt = Date.now();
      const finish = () => {
        if (run !== listeningRun.current || finishListening.current !== finish) return;
        clearListeningTimers();
        finishListening.current = null;
        liveCapture.current?.source.disconnect();
        liveCapture.current?.processor.disconnect();
        void liveCapture.current?.context.close();
        liveCapture.current = null;
        stopMicrophone();
        setIsListening(false);
        setAudioLevel(0);
        if (samples.length === 0) {
          setLiveError("No usable audio was captured. Try again with music playing nearby.");
          return;
        }
        if (!matchFound.current) void checkCapture(samples, context.sampleRate, run, true, Math.min(maximumListenSeconds * 1_000, Date.now() - startedAt));
      };
      finishListening.current = finish;
      setListenSeconds(0);
      setIsListening(true);
      elapsedTimer.current = window.setInterval(() => setListenSeconds(Math.min(maximumListenSeconds, Math.floor((Date.now() - startedAt) / 1000))), 250);
      firstCheckTimer.current = window.setTimeout(() => {
        if (run !== listeningRun.current || finishListening.current !== finish) return;
        void checkCapture(samples, context.sampleRate, run, false, firstCheckMilliseconds);
        checkTimer.current = window.setInterval(() => {
          // Reserve the final second for the definitive 15-second check. The
          // public identify limiter needs two seconds between requests.
          if (Date.now() - startedAt > maximumListenSeconds * 1_000 - checkIntervalMilliseconds) return;
          void checkCapture(samples, context.sampleRate, run, false, Date.now() - startedAt);
        }, checkIntervalMilliseconds);
      }, firstCheckMilliseconds);
      stopTimer.current = window.setTimeout(() => {
        reachedListenLimit.current = true;
        stopListening();
      }, maximumListenSeconds * 1000);
    } catch {
      liveCapture.current?.source.disconnect();
      liveCapture.current?.processor.disconnect();
      void liveCapture.current?.context.close();
      liveCapture.current = null;
      stopMicrophone();
      setAudioLevel(0);
      setLiveError("Could not access the microphone. Allow microphone access, then try again.");
    }
  };
  useEffect(() => () => {
    listeningRun.current += 1;
    clearListeningTimers();
    finishListening.current = null;
    liveCapture.current?.source.disconnect();
    liveCapture.current?.processor.disconnect();
    void liveCapture.current?.context.close();
    liveCapture.current = null;
    stopMicrophone();
  }, []);
  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!file) return;
    // A file-based identify is a new run too — clear any stale live state so
    // it can't be shown, or preferred over this run's own result, once this
    // finishes. See the `result` derivation below for the other half of this.
    setLiveError(null);
    setLiveResult(null);
    setLiveSearchExhausted(false);
    mutation.mutate(file);
  };
  const isProcessing = isCheckingLive || mutation.isPending;
  // A completed result belongs to one run only. `mutation.isPending` has to
  // be in this guard, not just `isListening`/`isCheckingLive`: without it, a
  // truthy `liveResult` from a prior live run wins the `??` below unconditionally
  // while a fresh file upload is still in flight, so the old result stays on
  // screen straight through the "processing" state and never gets replaced.
  const isBusy = isListening || isCheckingLive || mutation.isPending;
  const currentLiveResult = liveResult?.run === visibleLiveRun ? liveResult.response : null;
  const result = isBusy ? null : currentLiveResult ?? mutation.data ?? null;
  return <>
    <section className="listening-hero" aria-labelledby="identify-title">
      <p className="kicker"><i /> LIVE MUSIC FINDER</p>
      <h1 id="identify-title">What’s playing?</h1>
      <p className="listening-subtitle">Bring the music close. Lyra compares what it hears against your private catalog—without keeping the recording.</p>
      <p className="hero-meta"><span>Private catalog</span><i aria-hidden="true" /><span>Up to 15 seconds</span></p>
      <div className="orb-stage">
        <ListeningOrb active={isListening} checking={isProcessing} level={audioLevel} seconds={listenSeconds} maximum={maximumListenSeconds} response={result} onClick={isListening ? stopListening : () => { void startListening(); }} disabled={mutation.isPending || (!isListening && isCheckingLive)} />
        {result && <MatchNode response={result} />}
      </div>
      <p className="orb-status" role="status">{isListening ? <><span>Listening</span><strong>{listenSeconds}<i>/</i>{maximumListenSeconds}s</strong><button onClick={stopListening} type="button">Stop</button></> : isProcessing ? "Finding a match…" : result && !result.matched ? "No match in your catalog" : "Tap the orb to start listening"}</p>
    </section>
    <form className="identify-card upload-card" onSubmit={submit}>
      <label className={file ? "drop-zone has-file" : "drop-zone"}>
        <input accept="audio/*" type="file" disabled={isListening || mutation.isPending} onChange={(event) => { setFile(event.target.files?.[0] ?? null); setLiveError(null); mutation.reset(); }} />
        <span className="drop-icon">⌁</span><span className="drop-title">{file ? file.name : "Choose an audio recording"}</span><span className="drop-copy">{file ? `${formatBytes(file.size)} · ready to identify` : "MP3, WAV, AAC and other FFmpeg-supported audio"}</span>
      </label>
      <div className="identify-actions"><p><strong>Prefer a file?</strong> Upload a clear excerpt from an indexed reference.</p><button className="button button-primary" disabled={!file || mutation.isPending || isListening || isCheckingLive} type="submit">{mutation.isPending ? <><Spinner /> Reading landmarks…</> : <>Identify recording <Arrow /></>}</button></div>
    </form>
    {isListening && <p className="live-note" role="status">Listening from your microphone. Lyra checks the growing capture and stops as soon as it finds a match; otherwise it keeps listening for up to 15 seconds.</p>}
    {!isListening && isCheckingLive && <p className="live-note" role="status">Checking the final live capture…</p>}
    {liveError && <Notice kind="error" title="Live listening unavailable">{liveError}</Notice>}
    {mutation.isError && <Notice kind="error" title="Could not process that recording">Try another supported audio file under 10 MB.</Notice>}
    <section className="trust-row" aria-label="How Lyra works"><TrustItem icon="01" title="Listen" text="Play a clear excerpt near your microphone." /><TrustItem icon="02" title="Align" text="Landmarks are aligned against your private catalog." /><TrustItem icon="03" title="Find" text="A match ends listening early; otherwise it stops at 15 seconds." /></section>
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

function Result({ response, searchExhausted = false }: { response: IdentifyResponse; searchExhausted?: boolean }) {
  if (response.reason === "insufficient_audio_signal") return <Notice kind="warning" title="Not enough usable signal">Try a clearer recording with more music and less silence.</Notice>;
  if (!response.matched || !response.match) return <section className="result-card no-match"><span className="result-icon">⌁</span><div><p className="kicker"><i /> NO CONFIDENT MATCH</p><h2>{searchExhausted ? "No match after 15 seconds." : "Nothing matched this recording."}</h2><p>Lyra only identifies audio that has been added to your private reference catalog.</p><small>Processed in {response.processing_ms} ms</small></div></section>;
  const { match } = response;
  return <section className="result-card matched"><span className="result-icon">✓</span><div><p className="kicker"><i /> MATCH FOUND</p><h2>{match.title}</h2><p>{match.artist}{match.album ? ` · ${match.album}` : ""}</p><small>Reference offset {formatTime(match.reference_offset_ms)} · processed in {response.processing_ms} ms</small></div><div className="match-evidence"><span>Evidence</span><strong>{match.match_strength === "timing_aligned" ? "Timing aligned" : "Match found"}</strong></div></section>;
}

function StatusBadge({ status, reason }: { status: string; reason: string | null }) { const tone = status === "READY" ? "ready" : status === "FAILED" ? "failed" : "pending"; return <div className={`status status-${tone}`} title={reason ?? undefined}><i /> {status}</div>; }
function Notice({ kind, title, children }: { kind: "error" | "warning" | "success"; title: string; children: string }) { return <section className={`notice notice-${kind}`} role={kind === "error" ? "alert" : undefined}><span>{kind === "success" ? "✓" : "!"}</span><div><strong>{title}</strong><p>{children}</p></div></section>; }
function TrustItem({ icon, title, text }: { icon: string; title: string; text: string }) { return <article><span>{icon}</span><div><strong>{title}</strong><p>{text}</p></div></article>; }
function Stat({ value, label }: { value: string | number; label: string }) { return <div><strong>{value}</strong><span>{label}</span></div>; }
function Spinner() { return <i className="spinner" aria-hidden="true" />; }
function ListeningOrb({ active, checking, level, seconds, maximum, response, onClick, disabled }: { active: boolean; checking: boolean; level: number; seconds: number; maximum: number; response: IdentifyResponse | null; onClick: () => void; disabled: boolean }) {
  const scale = 1 + level * .16;
  const match = response?.matched ? response.match : null;
  const noMatch = response && !response.matched;
  return <button className={`listening-orb ${active ? "is-listening" : ""} ${checking ? "is-checking" : ""} ${match ? "has-match" : ""} ${noMatch ? "has-no-match" : ""}`} disabled={disabled} onClick={onClick} type="button" aria-label={active ? "Stop listening" : "Start live listening"} style={{ "--level": level, "--orb-scale": scale } as CSSProperties}>
    <span className="orb-ripple orb-ripple-one" /><span className="orb-ripple orb-ripple-two" />
    <span className="orb-core"><FluidOrbCanvas level={level} active={active} checking={checking} matched={Boolean(match)} noMatch={Boolean(noMatch)} /><span className="orb-glyph">{match ? "✓" : noMatch ? "" : active ? "∿" : "◉"}</span>{match ? <span className="orb-result"><strong>{match.title}</strong><small>{match.artist}</small></span> : active && <small>{seconds}/{maximum}</small>}</span>
  </button>;
}
function FluidOrbCanvas({ level, active, checking, matched, noMatch }: { level: number; active: boolean; checking: boolean; matched: boolean; noMatch: boolean }) {
  const canvas = useRef<HTMLCanvasElement>(null);
  const levelRef = useRef(level);
  levelRef.current = level;
  useEffect(() => {
    const element = canvas.current;
    if (!element) return;
    const context = element.getContext("2d");
    if (!context) return;
    let frame = 0;
    let smoothLevel = 0;
    const draw = (now: number) => {
      const bounds = element.getBoundingClientRect();
      const ratio = Math.min(window.devicePixelRatio || 1, 2);
      const width = Math.max(1, Math.round(bounds.width * ratio)); const height = Math.max(1, Math.round(bounds.height * ratio));
      if (element.width !== width || element.height !== height) { element.width = width; element.height = height; }
      const size = Math.min(width, height); const center = size / 2; const time = now / 1000;
      smoothLevel += (levelRef.current - smoothLevel) * .075;
      const energy = active ? smoothLevel : checking ? .14 : matched ? .08 : noMatch ? .03 : .018;
      const hue = matched ? 155 : noMatch ? 225 : checking ? 278 : 258;
      context.clearRect(0, 0, width, height); context.save(); context.translate((width - size) / 2, (height - size) / 2);
      const glow = context.createRadialGradient(center - size * .13, center - size * .17, size * .025, center, center, size * .56);
      glow.addColorStop(0, `hsla(${hue + 42}, 100%, 94%, .96)`); glow.addColorStop(.16, `hsla(${hue + 22}, 96%, 78%, .92)`); glow.addColorStop(.52, `hsla(${hue}, 82%, 54%, .82)`); glow.addColorStop(1, `hsla(${hue - 30}, 78%, 18%, .14)`);
      context.fillStyle = glow; context.beginPath();
      for (let index = 0; index <= 96; index += 1) { const angle = index / 96 * Math.PI * 2; const wave = Math.sin(angle * 3 + time * .72) * .008 + Math.cos(angle * 5 - time * .42) * .006; const radius = size * (.425 + wave + energy * .035 * Math.sin(angle * 4 + time * 1.3)); const x = center + Math.cos(angle) * radius; const y = center + Math.sin(angle) * radius; index === 0 ? context.moveTo(x, y) : context.lineTo(x, y); }
      context.closePath(); context.shadowBlur = size * (.13 + energy * .14); context.shadowColor = `hsla(${hue}, 90%, 66%, .58)`; context.fill(); context.restore();
      frame = requestAnimationFrame(draw);
    };
    frame = requestAnimationFrame(draw);
    return () => cancelAnimationFrame(frame);
  }, [active, checking, matched, noMatch]);
  return <canvas className="fluid-orb-canvas" ref={canvas} aria-hidden="true" />;
}
function MatchNode({ response }: { response: IdentifyResponse }) {
  const match = response.matched ? response.match : null;
  return <article className={`match-node ${match ? "has-match" : "has-no-match"}`}><span className="node-line" aria-hidden="true" /><div className="node-card"><span className="node-dot">{match ? "✓" : "×"}</span><div><small>{match ? "CATALOG MATCH" : "NO CONFIDENT MATCH"}</small><strong>{match ? match.title : "Nothing matched"}</strong><p>{match ? match.artist : "Try another excerpt"}</p></div></div>
  </article>;
}
function Arrow() { return <span aria-hidden="true">→</span>; }
function encodeWAV(samples: Float32Array[], sampleRate: number) {
  const sampleCount = samples.reduce((total, sample) => total + sample.length, 0);
  const buffer = new ArrayBuffer(44 + sampleCount * 2);
  const view = new DataView(buffer);
  const writeText = (offset: number, value: string) => { for (let index = 0; index < value.length; index += 1) view.setUint8(offset + index, value.charCodeAt(index)); };
  writeText(0, "RIFF"); view.setUint32(4, 36 + sampleCount * 2, true); writeText(8, "WAVE"); writeText(12, "fmt ");
  view.setUint32(16, 16, true); view.setUint16(20, 1, true); view.setUint16(22, 1, true); view.setUint32(24, sampleRate, true);
  view.setUint32(28, sampleRate * 2, true); view.setUint16(32, 2, true); view.setUint16(34, 16, true); writeText(36, "data"); view.setUint32(40, sampleCount * 2, true);
  let offset = 44;
  for (const sample of samples) for (const value of sample) { view.setInt16(offset, Math.max(-1, Math.min(1, value)) * 0x7fff, true); offset += 2; }
  return new Blob([buffer], { type: "audio/wav" });
}
function formatBytes(bytes: number) { return `${(bytes / (1024 * 1024)).toFixed(bytes < 1024 * 1024 ? 2 : 1)} MB`; }
function formatTime(milliseconds: number) { const seconds = Math.max(0, Math.round(milliseconds / 1000)); return `${Math.floor(seconds / 60)}:${String(seconds % 60).padStart(2, "0")}`; }
function loginErrorMessage(error: Error) { if (error.message === "invalid_credentials") return "Invalid username or password."; if (error.message === "rate_limited") return "Too many sign-in attempts. Wait one minute, then try again."; return "Sign-in is temporarily unavailable. Confirm that the API has started and migrations completed."; }
