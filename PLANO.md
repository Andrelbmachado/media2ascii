# PLANO — media2ascii

## Fase 1 — Funcionar 100% no Mac e no Windows
- [x] Cross-compile ok: `darwin/arm64`, `windows/amd64`, `linux/amd64` (verificado em 21/10/2026).
- [x] `go test ./...` passa no Linux.
- [ ] Testar de verdade no Mac (Terminal, iTerm2) — imagem, vídeo, colorido, controles.
- [ ] Testar no Windows (Windows Terminal, PowerShell, cmd) — cores ANSI, `Ctrl+C`, ffmpeg no PATH.
- [ ] CI com matriz `macos-latest`, `windows-latest`, `ubuntu-latest` rodando build + testes.

## Fase 2 — Tela responsiva
- [ ] Recalcular o tamanho do quadro a cada frame (ou ao receber `SIGWINCH` no Mac/Linux; polling de `term.GetSize` no Windows).
- [ ] Imagem estática: redesenhar ao redimensionar.
- [ ] Teste automatizado do cálculo de tamanho com terminal simulado.

## Fase 3 — Distribuição
- [ ] GoReleaser + GitHub Actions publicando Releases a cada tag (tirar `dist/` do git).
- [ ] Atualizar a fórmula Homebrew automaticamente (hoje está na v1.1.0, o Makefile já está na v2.0.0).
- [ ] Winget / Scoop para Windows.
- [ ] Pacote npm (`npm/`) apontando para os binários do Release.

## Fase 4 — Página atraente
- [ ] GIFs novos de demonstração (foto e vídeo, colorido e P&B).
- [ ] GitHub Pages com player ASCII no navegador e botões de download Mac/Windows.

## Merge com homebrew-media2ascii (aguardando confirmação)
O Homebrew exige que um tap se chame `homebrew-<nome>`, então o repo da fórmula não pode simplesmente sumir sem quebrar `brew tap Andrelbmachado/media2ascii`. Opções:
1. Manter o tap, mas atualizá-lo automaticamente a partir deste repo (recomendado).
2. Mover `Formula/media2ascii.rb` para cá e instalar com `brew tap Andrelbmachado/media2ascii https://github.com/Andrelbmachado/media2ascii` (URL explícita) e só então arquivar o tap.
