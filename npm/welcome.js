#!/usr/bin/env node
'use strict';

// Skip in CI, piped output, or silent installs
if (
  !process.stdout.isTTY ||
  process.env.CI ||
  process.env.TERM === 'dumb' ||
  process.env.npm_config_loglevel === 'silent'
) {
  process.exit(0);
}

const pkg = require('./package.json');

// ─── ANSI primitives ─────────────────────────────────────────────────────────
const R    = '\x1b[0m';
const B    = '\x1b[1m';
const fg   = n      => `\x1b[38;5;${n}m`;
const bg   = n      => `\x1b[48;5;${n}m`;
const vlen = s      => s.replace(/\x1b\[[\d;]*m/g, '').length;
const rpad = (s, w) => s + ' '.repeat(Math.max(0, w - vlen(s)));

// ─── Palette ─────────────────────────────────────────────────────────────────
const GN  = fg(82);   // lime green   — brand
const WH  = fg(231);  // bright white — text
const GR  = fg(245);  // gray         — secondary
const DGR = fg(239);  // dark gray    — dimmed
const YL  = fg(220);  // yellow       — highlights
const FR  = fg(241);  // frame

// Half-block play button palette
// Each char cell = 2 vertical "pixels": ▀ top=fg/bot=bg  ▄ top=bg/bot=fg  █ full=fg
const G  = `${fg(82)}${bg(234)}`;   // green on near-black (filled pixel)
const DK = `${fg(234)}${bg(234)}`;  // dark on dark (empty pixel, invisible)

// ─── Box builder ─────────────────────────────────────────────────────────────
const W   = 74;                         // inner width (total line = 76 chars)
const bar = n => '═'.repeat(n);
const row = s => `${FR}║${R}${rpad(' ' + s, W)}${FR}║${R}`;
const blk = () => row('');
const top = `${FR}╔${bar(W)}╗${R}`;
const sep = `${FR}╠${bar(W)}╣${R}`;
const bot = `${FR}╚${bar(W)}╝${R}`;

// ─── Pixel-art play button (half-block technique, 6 chars × 4 rows) ──────────
//
//  Pixel map — G = green, · = dark background  (6 cols × 8 pixel-rows)
//
//  col:  0  1  2  3  4  5
//  px0:  G  ·  ·  ·  ·  ·
//  px1:  G  G  ·  ·  ·  ·
//  px2:  G  G  G  ·  ·  ·
//  px3:  G  G  G  G  ·  ·
//  px4:  G  G  G  G  ·  ·
//  px5:  G  G  G  ·  ·  ·
//  px6:  G  G  ·  ·  ·  ·
//  px7:  G  ·  ·  ·  ·  ·
//
//  Encoding into 4 character rows (2 pixel-rows each):
//    ▀  →  top pixel = fg color,  bottom pixel = bg color
//    ▄  →  top pixel = bg color,  bottom pixel = fg color
//    █  →  both pixels = fg color
//    ' '→  both pixels = bg color (invisible)
//
//  Row A (px0,1): (G,G)(·,G)(·,·)(·,·)(·,·)(·,·)  →  █ ▄ · · · ·
//  Row B (px2,3): (G,G)(G,G)(G,G)(·,G)(·,·)(·,·)  →  █ █ █ ▄ · ·
//  Row C (px4,5): (G,G)(G,G)(G,G)(G,·)(·,·)(·,·)  →  █ █ █ ▀ · ·
//  Row D (px6,7): (G,G)(G,·)(·,·)(·,·)(·,·)(·,·)  →  █ ▀ · · · ·

const pA = `${G}█${G}▄${DK} ${DK} ${DK} ${DK} `;    // 6 visible chars
const pB = `${G}█${G}█${G}█${G}▄${DK} ${DK} `;        // 6 visible chars
const pC = `${G}█${G}█${G}█${G}▀${DK} ${DK} `;        // 6 visible chars
const pD = `${G}█${G}▀${DK} ${DK} ${DK} ${DK} `;      // 6 visible chars

const gap = '  ';  // space between icon and text

// ─── Welcome screen ──────────────────────────────────────────────────────────
const screen = [
  '',
  top,
  blk(),
  row(`${pA}${gap}${GN}${B}media2ascii${R}  ${DGR}v${pkg.version}${R}`),
  row(`${pB}${gap}${WH}Converta imagens e vídeos em ASCII art no terminal${R}`),
  row(`${pC}${gap}${GR}Requer: ${YL}brew install ffmpeg${GR}  ·  ${YL}apt install ffmpeg${R}`),
  row(`${pD}${gap}${DGR}github.com/Andrelbmachado/media2ascii${R}`),
  blk(),
  sep,
  blk(),
  row(`${WH}${B}Uso${R}   ${GN}media2ascii${R} ${YL}<arquivo>${R}  ${GR}[qualidade 0–100]  [fps]${R}`),
  blk(),
  row(`${GR}Exemplos${R}`),
  row(`  ${GN}media2ascii${R} video.mp4`),
  row(`  ${GN}media2ascii${R} video.mp4 ${YL}70 30${R}`),
  row(`  ${GN}media2ascii${R} imagem.jpg ${YL}90${R}`),
  row(`  ${GN}media2ascii${R} video.mp4 ${DGR}--export${R}`),
  blk(),
  bot,
  '',
].join('\n');

process.stdout.write(screen + '\n');
