#!/usr/bin/env node

const { execFileSync } = require("child_process");
const path = require("path");
const os = require("os");

const platform = os.platform();
const arch = os.arch();

let bin;
if (platform === "darwin" && arch === "arm64") {
  bin = path.join(__dirname, "media2ascii_darwin_arm64");
} else if (platform === "darwin") {
  bin = path.join(__dirname, "media2ascii_darwin_amd64");
} else {
  console.error("Plataforma não suportada: " + platform + "/" + arch);
  process.exit(1);
}

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
