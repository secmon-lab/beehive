package firestore

import "google.golang.org/api/iterator"

// ErrIteratorDone exposes the upstream sentinel so other files in this
// package can errors.Is against it without re-importing the iterator
// package everywhere.
func ErrIteratorDone() error { return iterator.Done }
