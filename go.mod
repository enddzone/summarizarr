module summarizarr

go 1.24.5

require (
	github.com/alexedwards/scs/sqlite3store v0.0.0-20231113091146-cef4b05350c8
	github.com/alexedwards/scs/v2 v2.9.0
	github.com/coder/websocket v1.8.14
	github.com/mattn/go-sqlite3 v1.14.17-0.20240122133042-fb824c8e339e
	github.com/sashabaranov/go-openai v1.41.2
	golang.org/x/crypto v0.47.0
)

replace github.com/mattn/go-sqlite3 => github.com/jgiannuzzi/go-sqlite3 v1.14.17-0.20240122133042-fb824c8e339e
