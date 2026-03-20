# media2ascii

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
# Imagem com configurações padrão
media2ascii imagem.jpg

# Vídeo com 10 fps e qualidade 70
media2ascii video.mp4 10 70

# Vídeo com alta qualidade
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
