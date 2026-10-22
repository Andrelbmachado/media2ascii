# PLANO — media2ascii

Plano executável por fases. Cada tarefa tem **onde mexer** e **pronto quando** (critério verificável).
Ordem recomendada: Fase 1 → 2 → 3 → 4. Fases 2 e 3 podem andar em paralelo depois da CI (1.4).

Legenda: `[x]` feito · `[ ]` a fazer · **(bug)** comportamento atual diverge do prometido.

---

## Fase 1 — Funcionar 100% no Mac e no Windows

**Meta:** qualquer pessoa baixa o binário do Release e usa imagem + vídeo + áudio sem surpresas.

- [x] **1.1** Cross-compile `darwin/arm64`, `windows/amd64`, `linux/amd64` (verificado em 21/10/2026).
- [x] **1.2** `go test ./...` passa no Linux.
- [ ] **1.3 Roteiro de teste manual** — rodar em macOS (Terminal.app, iTerm2) e Windows (Windows Terminal, PowerShell, cmd). Registrar resultado numa tabela em `docs/TESTES.md`.

  | # | Comando | Esperado |
  |---|---|---|
  | a | `media2ascii docs/images/lufei.jpg` | imagem colorida, cabe na tela |
  | b | `media2ascii docs/images/lufei.jpg 100 30 BW` | imagem P&B |
  | c | `media2ascii video.mp4` | vídeo centralizado, com áudio |
  | d | ESPAÇO → `ENTER` | pausa e continua do mesmo ponto |
  | e | ESPAÇO → `audio OFF` | áudio para na hora |
  | f | fim do vídeo → `ENTER` | replay do início |
  | g | `Q` e `Ctrl+C` durante o vídeo | sai e o terminal volta ao normal (eco, cursor) |
  | h | sem ffmpeg no PATH | mensagem clara com comando de instalação **do sistema atual** |
  | i | `media2ascii video.mp4 100 24 --export` | gera `video_ascii.mp4` reproduzível |

  **Pronto quando:** todas as linhas ✅ nos 5 terminais, ou com issue aberta para cada ❌.

- [ ] **1.4 CI** — criar `.github/workflows/ci.yml` com matriz `macos-latest`, `windows-latest`, `ubuntu-latest`: `go vet ./...`, `go test ./...`, `go build .`.
  **Pronto quando:** PR mostra os 3 checks verdes.
- [ ] **1.5 Mensagem do ffmpeg por sistema** — `missingToolGuidance` (`main.go:807`) só cita brew/apt. Usar `runtime.GOOS` e sugerir `winget install Gyan.FFmpeg` no Windows. Teste em `main_test.go`.
  **Pronto quando:** teste cobre os 3 sistemas.
- [ ] **1.6 Áudio no Windows robusto** — caminho temporário é interpolado entre aspas simples no comando PowerShell (`main.go:1068`); usuário com `'` no nome quebra. Escapar `'` → `''`.
  **Pronto quando:** teste unitário da string gerada.

## Fase 2 — Corrigir o que o README marca como planejado

**Meta:** zerar a seção "Limitações conhecidas" do README.

- [ ] **2.1 (bug) `qualidade` não faz nada** — é lida em `parseArgs` (`main.go:143`) e no menu, mas `buildConvertOptions` (`main.go:196`) e `applyVideoSettingsForScreen` (`main.go:604`) a ignoram. Proposta: mapear 0–100 para a fração da área do terminal usada (`scaleBetween`, `main.go:625`, já existe, hoje só usada em testes), ex.: 0 → 25% da largura, 100 → 100%.
  **Pronto quando:** `media2ascii foto.jpg 30` gera saída visivelmente menor que `... 100`, com teste em `main_test.go` checando a largura resultante.
- [ ] **2.2 (bug) Mudanças na pausa só valem no replay** — `fps`/`cores` alterados na pausa não recalculam `frameInterval` nem `opts` (`main.go:302`, `main.go:349`). Ou aplicar na hora (recriar o stream a partir do quadro atual com `-ss`), ou deixar explícito no prompt.
  **Pronto quando:** comportamento escolhido documentado no README e coberto por teste de `parsePlaybackCommand`.
- [ ] **2.3 Tela responsiva** — o conversor já mede o terminal a cada quadro (`convert/resize.go:58`), mas a centralização usa `screenWidth/screenHeight` medidos uma vez (`main.go:277`). Medir dentro de `playFrames` a cada quadro (custo baixo) ou via `SIGWINCH` em Unix + polling em Windows.
  **Pronto quando:** redimensionar a janela durante o vídeo mantém o quadro centralizado e sem lixo na tela, nos 3 sistemas.
- [ ] **2.4 Imagem estática redesenha ao redimensionar** — opcional, atrás de flag `--watch` (imagem fica na tela até `Q`).
- [ ] **2.5 (bug) `--export`** — ignora `BW` (`main.go:978` fixa `colored: true`) e o tamanho virtual 200×60 não é usado (`applyVideoSettingsForScreen` descarta os parâmetros), então a resolução depende do terminal. Respeitar `cores` e usar `FixedWidth/FixedHeight`.
  **Pronto quando:** dois exports do mesmo vídeo em terminais de tamanhos diferentes geram MP4 com a mesma resolução.
- [ ] **2.6 Prompt do menu** — diz `FPS = 0->30`, mas aceita qualquer valor > 0. Ajustar texto em `askPlaybackAction` (`main.go:696`) e em `usage()`.
- [ ] **2.7 Áudio no Linux** — tentar `ffplay -nodisp -autoexit`, depois `paplay`/`aplay`; silencioso se nenhum existir.
- [ ] **2.8 Limpeza** — remover código só usado em testes (`centerASCIIFrame`, se 2.3 não o aproveitar) e atualizar o comentário de versão em `main.go:1` (diz v2.0.0; npm está em 2.0.5).

## Fase 3 — Distribuição

**Meta:** uma tag `vX.Y.Z` publica tudo sozinha; nenhum binário versionado no git.

- [ ] **3.1 GoReleaser** — `.goreleaser.yaml` reproduzindo os alvos do `Makefile` (darwin/linux/windows/freebsd, mesmos nomes de arquivo) + `checksums.txt`. Injetar versão via `-ldflags "-X main.version=..."` e adicionar `media2ascii --version`.
- [ ] **3.2 Workflow de release** — `.github/workflows/release.yml` disparado em `push` de tag `v*`.
- [ ] **3.3 Tirar `dist/` e `npm/bin/media2ascii_*` do git** — adicionar ao `.gitignore` e `git rm --cached` (os arquivos ficam no disco).
- [ ] **3.4 Homebrew automático** — bloco `brews:` do GoReleaser atualizando `Andrelbmachado/homebrew-media2ascii` (hoje na v1.1.0). Requer um token com escrita no tap como secret.
  **Pronto quando:** `brew upgrade media2ascii` instala a mesma versão da última tag.
- [ ] **3.5 Windows** — manifesto Scoop (bucket próprio, via GoReleaser `scoops:`) primeiro; Winget depois (PR em `microsoft/winget-pkgs`).
- [ ] **3.6 npm** — `npm/install.js` baixar o binário do Release em vez de empacotar 8 binários; publicar via workflow com `NPM_TOKEN`. Confirmar antes se o pacote `media2ascii` já existe no registro e quem é o dono.
- [ ] **3.7 Assinatura macOS** — avaliar notarização (exige conta Apple Developer); até lá, manter a dica de `xattr` no README.

**Pronto quando (fase):** criar a tag `v2.1.0` gera Release com binários + checksums, atualiza Homebrew e Scoop, sem passos manuais.

## Fase 4 — Página atraente

- [ ] **4.1 GIFs novos** — gravar com `vhs` (charmbracelet) a partir de um `.tape` versionado em `docs/demo.tape`: foto colorida, foto P&B, vídeo com pausa/menu. Substituir `docs/images/demo.gif`; apagar imagens antigas não usadas no README.
- [ ] **4.2 GitHub Pages** — `docs/index.html` estático: hero com GIF, player ASCII no navegador (frames pré-gerados com `--export` em `.txt`, tocados em `<pre>`), botões de download que detectam o sistema e apontam para `releases/latest`.
- [ ] **4.3 README** — badges (CI, release, licença) depois de 1.4 e 3.2.

## Decisão pendente — merge com `homebrew-media2ascii`

O Homebrew exige que um tap se chame `homebrew-<nome>`; apagar o repo da fórmula quebra `brew tap Andrelbmachado/media2ascii`.

1. **Manter o tap e atualizá-lo pelo GoReleaser deste repo (recomendado)** — zero quebra para usuários; é a tarefa 3.4.
2. Mover `Formula/media2ascii.rb` para cá e instruir `brew tap Andrelbmachado/media2ascii https://github.com/Andrelbmachado/media2ascii` (URL explícita); só então arquivar o tap. Quebra quem já usa o comando curto.

**Aguardando confirmação do André** antes de qualquer mudança no repo do tap.
