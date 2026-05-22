package fetcher_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/beehive/pkg/domain/interfaces"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/domain/types"
	"github.com/secmon-lab/beehive/pkg/service/fetcher"
	"github.com/secmon-lab/beehive/pkg/utils/errutil"
	"github.com/secmon-lab/beehive/pkg/utils/id"
)

// fakeBlogProvider is a no-op BlogProvider used to verify registration.
type fakeBlogProvider struct{}

func (fakeBlogProvider) Kind() types.SourceKind { return types.KindBlog }
func (fakeBlogProvider) Fetch(_ context.Context, _ *model.Source) ([]*interfaces.FetchedArticle, error) {
	return nil, nil
}

func TestRegistry_RegisterAndResolve(t *testing.T) {
	reg := fetcher.NewRegistry()
	typeID := "test_blog_" + id.NewULID()
	gt.NoError(t, reg.Register(typeID, fakeBlogProvider{}))

	prov, err := reg.Resolve(typeID)
	gt.NoError(t, err)
	gt.Equal(t, prov.Kind(), types.KindBlog)
}

func TestRegistry_UnknownTypeIsNotFound(t *testing.T) {
	reg := fetcher.NewRegistry()
	_, err := reg.Resolve("nope_" + id.NewULID())
	gt.True(t, errutil.IsNotFound(err))
}

func TestRegistry_DuplicateRegisterReturnsError(t *testing.T) {
	reg := fetcher.NewRegistry()
	typeID := "dup_" + id.NewULID()
	gt.NoError(t, reg.Register(typeID, fakeBlogProvider{}))
	err := reg.Register(typeID, fakeBlogProvider{})
	gt.Error(t, err)
}

func TestRegistry_TypesIsSorted(t *testing.T) {
	reg := fetcher.NewRegistry()
	a := "aaa_" + id.NewULID()
	b := "bbb_" + id.NewULID()
	gt.NoError(t, reg.Register(b, fakeBlogProvider{}))
	gt.NoError(t, reg.Register(a, fakeBlogProvider{}))

	ts := reg.Types()
	gt.A(t, ts).Length(2)
	for i := 1; i < len(ts); i++ {
		gt.True(t, ts[i-1] <= ts[i])
	}
}

func TestRegistry_NilSafe(t *testing.T) {
	var reg *fetcher.Registry
	_, err := reg.Resolve("x")
	gt.Error(t, err)
	gt.Equal(t, reg.Len(), 0)
	gt.A(t, reg.Types()).Length(0)
}

func TestRegistry_MustRegisterPanicsOnDuplicate(t *testing.T) {
	reg := fetcher.NewRegistry()
	typeID := "must_" + id.NewULID()
	reg.MustRegister(typeID, fakeBlogProvider{})
	defer func() {
		r := recover()
		gt.NotNil(t, r)
	}()
	reg.MustRegister(typeID, fakeBlogProvider{})
}
