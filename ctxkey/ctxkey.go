package ctxkey

const (
	Logger CtxKey = iota
	TestingPeer
	TestingTx
)

type CtxKey int
