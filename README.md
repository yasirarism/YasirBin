# YasirBin

> _$ ./\<YasirBin{0x1337}/>

A simple, free, and open-source pastebin for your text files. Built with Go.

## Features

- ✨ **Syntax Highlighting** — Automatic language detection with highlight.js
- 🔒 **Password Protection** — Secure pastes with bcrypt-hashed passwords
- ⏳ **Expiring Pastes** — 1 day, 7 days, 30 days, or never
- 🌙 **Dark/Light Mode** — Dracula theme with toggle
- 📋 **Copy URL** — One-click clipboard copy
- 📄 **Raw View** — Plain text output for piping
- 🚀 **REST API** — JSON and form-data support
- ⌨️ **Ctrl+S Save** — Keyboard shortcut to save paste
- 🧹 **Auto Cleanup** — Expired pastes are automatically purged
- 🐳 **Docker Ready** — Single container deployment

## Quick Start

### Run with Docker

```bash
docker build -t yasirbin .
docker run -d -p 3000:3000 -v yasirbin-data:/app/data yasirbin
```

### Run from Source

```bash
# Requires Go 1.22+ and CGO (for SQLite)
go mod download
CGO_ENABLED=1 go build -o yasirbin .
./yasirbin
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `3000` | Server port |
| `DB_PATH` | `./data/yasirbin.db` | SQLite database path |
| `BASE_URL` | (auto-detect) | Base URL for generated links |
| `MAX_SIZE` | `1048576` | Max paste size in bytes (1MB) |

## API Usage

### Create a Paste

```bash
# Form data
curl -X POST -d "content=Hello World" https://your-domain/api/document

# JSON
curl -X POST -H "Content-Type: application/json" \
  -d '{"content":"Hello","password":"secret","expires":"7d"}' \
  https://your-domain/api/document

# From file
curl -X POST --data-urlencode "content@file.txt" https://your-domain/api/document
```

### Get Raw Content

```bash
curl https://your-domain/raw/SLUG
curl "https://your-domain/raw/SLUG?password=secret"
```

### API Response

```json
{
  "ok": true,
  "message": "succesfully created document",
  "data": {
    "url": "https://your-domain/abc123",
    "length": 11,
    "date": "2025-01-01T00:00:00Z",
    "expire_at": "2025-01-31T00:00:00Z"
  }
}
```

## License

MIT
