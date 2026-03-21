#!/usr/bin/env node

const https = require("https");
const fs = require("fs");
const path = require("path");
const { execSync } = require("child_process");
const os = require("os");
const zlib = require("zlib");

const VERSION = "v1.0.0";
const REPO = "Andrelbmachado/media2ascii";
const BIN_DIR = path.join(__dirname, "bin");
const IS_WINDOWS = os.platform() === "win32";
const BIN_PATH = path.join(BIN_DIR, IS_WINDOWS ? "media2ascii_windows_amd64.exe" : "media2ascii");

function getPlatformAsset() {
  const platform = os.platform();
  const arch = os.arch();

  if (platform === "darwin") {
    if (arch === "arm64") return "media2ascii_darwin_arm64.tar.gz";
    return "media2ascii_darwin_amd64.tar.gz";
  }

  if (platform === "win32") {
    return "media2ascii_windows_amd64.zip";
  }

  throw new Error(
    `Plataforma não suportada: ${platform}/${arch}.\n` +
    `Por favor, compile manualmente: https://github.com/${REPO}`
  );
}

function download(url, dest) {
  return new Promise((resolve, reject) => {
    const follow = (u) => {
      https.get(u, { headers: { "User-Agent": "media2ascii-npm-installer" } }, (res) => {
        if (res.statusCode === 301 || res.statusCode === 302) {
          follow(res.headers.location);
          return;
        }
        if (res.statusCode !== 200) {
          reject(new Error(`Download falhou: HTTP ${res.statusCode} em ${u}`));
          return;
        }
        const file = fs.createWriteStream(dest);
        res.pipe(file);
        file.on("finish", () => file.close(resolve));
        file.on("error", reject);
      }).on("error", reject);
    };
    follow(url);
  });
}

function extractArchive(archivePath, destDir) {
  if (IS_WINDOWS) {
    execSync(`powershell -Command "Expand-Archive -Path '${archivePath}' -DestinationPath '${destDir}' -Force"`);
  } else {
    execSync(`tar -xzf "${archivePath}" -C "${destDir}"`);
  }
}

async function install() {
  if (!fs.existsSync(BIN_DIR)) fs.mkdirSync(BIN_DIR, { recursive: true });

  const asset = getPlatformAsset();
  const url = `https://github.com/${REPO}/releases/download/${VERSION}/${asset}`;
  const archivePath = path.join(os.tmpdir(), asset);

  console.log(`Baixando media2ascii ${VERSION}...`);
  await download(url, archivePath);

  console.log("Extraindo...");
  extractArchive(archivePath, BIN_DIR);
  fs.unlinkSync(archivePath);

  if (!IS_WINDOWS) {
    fs.chmodSync(BIN_PATH, 0o755);
  }
  console.log("media2ascii instalado com sucesso!");
}

install().catch((err) => {
  console.error("Erro na instalação:", err.message);
  process.exit(1);
});
