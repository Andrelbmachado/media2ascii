#!/usr/bin/env node

const https = require("https");
const fs = require("fs");
const path = require("path");
const { execSync } = require("child_process");
const os = require("os");
const zlib = require("zlib");

const VERSION = "v2.0.0";
const REPO = "Andrelbmachado/media2ascii";
const BIN_DIR = path.join(__dirname, "bin");

const ASSETS = {
  darwin:  { arm64: ["media2ascii_darwin_arm64.tar.gz",      false],  x64:   ["media2ascii_darwin_amd64.tar.gz",      false] },
  linux:   { x64:   ["media2ascii_linux_amd64.tar.gz",       false],  arm64: ["media2ascii_linux_arm64.tar.gz",       false], arm: ["media2ascii_linux_arm.tar.gz", false] },
  win32:   { x64:   ["media2ascii_windows_amd64.zip",        true],   arm64: ["media2ascii_windows_arm64.zip",        true] },
  freebsd: { x64:   ["media2ascii_freebsd_amd64.tar.gz",     false] },
};

function getPlatformAsset() {
  const platform = os.platform();
  const arch = os.arch();
  const entry = ASSETS[platform] && (ASSETS[platform][arch] || ASSETS[platform]["x64"]);
  if (!entry) {
    throw new Error(
      `Plataforma não suportada: ${platform}/${arch}.\n` +
      `Por favor, compile manualmente: https://github.com/${REPO}`
    );
  }
  return { asset: entry[0], isZip: entry[1] };
}

function getBinPath(isZip) {
  const platform = os.platform();
  const arch = os.arch();
  const binName = isZip
    ? `media2ascii_${platform === "win32" ? "windows" : platform}_${arch === "x64" ? "amd64" : arch}.exe`
    : "media2ascii";
  return path.join(BIN_DIR, binName);
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

function extractArchive(archivePath, destDir, isZip) {
  if (isZip) {
    execSync(`powershell -Command "Expand-Archive -Path '${archivePath}' -DestinationPath '${destDir}' -Force"`);
  } else {
    execSync(`tar -xzf "${archivePath}" -C "${destDir}"`);
  }
}

async function install() {
  if (!fs.existsSync(BIN_DIR)) fs.mkdirSync(BIN_DIR, { recursive: true });

  const { asset, isZip } = getPlatformAsset();
  const url = `https://github.com/${REPO}/releases/download/${VERSION}/${asset}`;
  const archivePath = path.join(os.tmpdir(), asset);

  console.log(`Baixando media2ascii ${VERSION}...`);
  await download(url, archivePath);

  console.log("Extraindo...");
  extractArchive(archivePath, BIN_DIR, isZip);
  fs.unlinkSync(archivePath);

  if (!isZip) {
    fs.chmodSync(getBinPath(isZip), 0o755);
  }
  console.log("media2ascii instalado com sucesso!");
}

install().catch((err) => {
  console.error("Erro na instalação:", err.message);
  process.exit(1);
});
