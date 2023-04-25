package types

type SyncProvider interface {
	// Synchronising returns whether the downloader is currently synchronising.
	Synchronising() bool
	// FinSynchronising returns whether the downloader is currently retrieving finalized blocks.
	FinSynchronising() bool
	// DagSynchronising returns whether the downloader is currently retrieving dag chain blocks.
	DagSynchronising() bool
}
