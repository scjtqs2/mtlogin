package flaresolverr

//go:generate go run github.com/abice/go-enum@v0.5.5 --file=$GOFILE --marshal

// ENUM(
// sessions.create
// sessions.list
// sessions.destroy
// request.get
// request.post
// )
type command string
