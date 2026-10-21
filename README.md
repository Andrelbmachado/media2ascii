# media2ascii

**Transforme qualquer foto ou vídeo em arte ASCII — direto no seu terminal.**

![demo](docs/images/demo.gif)

- 🖼️ Imagens (JPG, PNG, GIF…) e 🎞️ vídeos (via ffmpeg)
- 🎨 Colorido ou preto e branco
- 📐 Ajusta o tamanho ao terminal
- ⚡ Binário único em Go, sem dependências (exceto ffmpeg para vídeo)

## Download

| Sistema | Arquivo |
|---|---|
| macOS (Apple Silicon) | `media2ascii_darwin_arm64.tar.gz` |
| macOS (Intel) | `media2ascii_darwin_amd64.tar.gz` |
| Windows (x64) | `media2ascii_windows_amd64.zip` |
| Windows (ARM) | `media2ascii_windows_arm64.zip` |
| Linux (x64 / ARM64 / ARM) | `media2ascii_linux_*.tar.gz` |

Baixe na página de **[Releases](https://github.com/Andrelbmachado/media2ascii/releases)**. Atalhos:

```bash
# macOS (Homebrew)
brew tap Andrelbmachado/media2ascii && brew install media2ascii
# Qualquer sistema com Go
go install github.com/Andrelbmachado/media2ascii/v2@latest
```

> **Status multiplataforma (21/10/2026):** o código compila para macOS (arm64), Windows (amd64) e Linux, e os testes passam no Linux. A execução real no Mac e no Windows ainda **precisa ser confirmada** (ver PLANO.md).
> O ajuste ao tamanho do terminal acontece ao abrir; **redimensionar em tempo real durante o vídeo é planejado**.

---

## Detalhes técnicos

Converta imagens e vídeos em arte ASCII diretamente no terminal.

## Instalação via Homebrew

```bash
brew tap Andrelbmachado/media2ascii
brew install media2ascii
```

## Instalação via Go

```bash
go install github.com/Andrelbmachado/media2ascii@latest
```

## Uso

```bash
media2ascii <arquivo> [fps] [qualidade]
```

| Parâmetro  | Descrição                              | Padrão |
|------------|----------------------------------------|--------|
| arquivo    | Caminho para imagem ou vídeo           | —      |
| fps        | Frames por segundo (somente vídeo)     | 8      |
| qualidade  | Nível de detalhe: 0 (baixo) a 100 (alto) | 70   |

## Exemplos

```bash
### Imagem com configurações padrão
media2ascii imagem.jpg

### Vídeo com 10 fps e qualidade 70
media2ascii video.mp4 10 70

### Vídeo com alta qualidade
media2ascii video.mp4 25 90
```

## Requisitos para vídeo

`ffmpeg` deve estar instalado:

```bash
brew install ffmpeg
```

## Controles durante a reprodução de vídeo

| Comando              | Ação                              |
|----------------------|-----------------------------------|
| `Ctrl+C`             | Parar imediatamente               |
| `ENTER`              | Replay                            |
| `fps <n>`            | Alterar FPS (ex: `fps 12`)        |
| `qualidade <0-100>`  | Alterar nível de detalhe          |
| `cores BW`           | Modo preto e branco               |
| `cores Color`        | Modo colorido                     |
| `tela`               | Exibir tamanho atual do terminal  |
| `sair`               | Sair                              |

## Licença

MIT — veja o arquivo [LICENSE](LICENSE).


> Este README e o PLANO.md foram reorganizados em 21/10/2026 a partir da visão do André. Itens marcados como **planejado** ainda não existem no código.
