#!/usr/bin/env node

const { execFileSync } = require("child_process");
const path = require("path");
const os = require("os");

const platform = os.platform();
const arch = os.arch();

const BINARIES = {
  darwin:  { arm64: "media2ascii_darwin_arm64",       x64: "media2ascii_darwin_amd64" },
  linux:   { arm64: "media2ascii_linux_arm64",        x64: "media2ascii_linux_amd64",  arm: "media2ascii_linux_arm" },
  win32:   { arm64: "media2ascii_windows_arm64.exe",  x64: "media2ascii_windows_amd64.exe" },
  freebsd: { x64:   "media2ascii_freebsd_amd64" },
};

const platformBins = BINARIES[platform];
const binName = platformBins && (platformBins[arch] || platformBins["x64"]);
if (!binName) {
  console.error("Plataforma não suportada: " + platform + "/" + arch);
  process.exit(1);
}
const bin = path.join(__dirname, binName);

// Add ffmpeg-static to PATH so the Go binary can find ffmpeg
const ffmpegPath = require("ffmpeg-static");
const ffmpegDir = path.dirname(ffmpegPath);
const env = Object.assign({}, process.env, {
  PATH: ffmpegDir + path.delimiter + (process.env.PATH || ""),
});

try {
  execFileSync(bin, process.argv.slice(2), { stdio: "inherit", env });
} catch (err) {
  process.exit(err.status || 1);
}
