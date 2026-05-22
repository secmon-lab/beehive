package usecase

import (
	"github.com/secmon-lab/beehive/pkg/domain/interfaces"
	"github.com/secmon-lab/beehive/pkg/service/extractor"
)

// importExtractorPostprocess is a tiny indirection so fetch.go does not
// have to import the extractor package directly — keeping the dep
// graph straight at the usecase boundary.
func importExtractorPostprocess(seeds []interfaces.IoCSeed) []*interfaces.IoCSeed {
	return extractor.Postprocess(seeds)
}
