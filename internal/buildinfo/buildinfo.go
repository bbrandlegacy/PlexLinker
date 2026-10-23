package buildinfo

// Version is replaced by release builds through -ldflags. Keeping the current
// release here gives source builds a clean product identity as well.
var Version = "v2.0.0"
