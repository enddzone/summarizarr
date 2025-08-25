module summarizarr

go 1.24.5

require (
	github.com/alexedwards/scs/sqlite3store v0.0.0-20231113091146-cef4b05350c8
	github.com/alexedwards/scs/v2 v2.9.0
	github.com/coder/websocket v1.8.13
	github.com/mattn/go-sqlite3 v1.14.32
	github.com/sashabaranov/go-openai v1.41.1
	golang.org/x/crypto v0.41.0
)

replace github.com/mattn/go-sqlite3 => github.com/mattn/go-sqlite3 v1.14.32
