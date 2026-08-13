# Lyra brand system

## Position

Lyra identifies the music hiding in real-world recordings. The brand should feel precise, calm, and quietly technical: an acoustic instrument translated into a reliable signal system.

## Identity

The native SVG mark is a single refined waveform with one highlighted landmark peak. It references acoustic signal analysis and the temporal consistency at the heart of Lyra’s matcher without relying on a generic microphone or music-note symbol.

Use [`web/public/brand/lyra-mark.svg`](../web/public/brand/lyra-mark.svg) where space is constrained, [`web/public/brand/lyra-lockup.svg`](../web/public/brand/lyra-lockup.svg) for repository and product headings, and [`web/public/brand/lyra-banner.svg`](../web/public/brand/lyra-banner.svg) for wide repository/social contexts. Keep clear space around the mark equal to its smallest node; do not recolor, stretch, or add effects beyond the supplied artwork.

## Palette

| Token | Value | Use |
| --- | --- | --- |
| Midnight | `#080B14` | Base background |
| Surface | `#12182A` | Cards and panels |
| Indigo | `#A5B4FC` | Primary actions |
| Violet | `#C084FC` | Matching / intelligence accent |
| Aqua | `#67E8F9` | Signal / indexing accent |
| Success | `#4ADE80` | Ready and confirmed states |
| Warning | `#FBBF24` | Pending states |
| Danger | `#FB7185` | Errors and destructive actions |

## Tone

Use direct, clear language: “Track is ready”, “No confident match”, “Reference audio accepted”. Avoid exaggerated AI claims, unverifiable performance promises, and noisy decorative copy.

## Performance rule

The browser interface must remain lightweight. Use native SVG and CSS tokens; avoid large raster assets, repeated backdrop filters, permanent blur effects, and required third-party font requests. Brand polish must not make local development or identification workflows feel slow.
