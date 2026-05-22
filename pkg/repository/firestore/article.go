package firestore

import (
	"context"
	"errors"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/utils/errutil"
)

func (f *Firestore) GetArticleByURL(ctx context.Context, url string) (*model.Article, error) {
	iter := f.client.Collection(collectionArticles).Where("URL", "==", url).Limit(1).Documents(ctx)
	defer iter.Stop()
	doc, err := iter.Next()
	if errors.Is(err, ErrIteratorDone()) {
		return nil, errutil.NotFound("article",
			goerr.V("url", url),
		)
	}
	if err != nil {
		return nil, goerr.Wrap(err, "query article by url", goerr.V("url", url))
	}
	var a model.Article
	if err := doc.DataTo(&a); err != nil {
		return nil, goerr.Wrap(err, "decode article", goerr.V("id", doc.Ref.ID))
	}
	return &a, nil
}

func (f *Firestore) CreateArticle(ctx context.Context, a *model.Article) error {
	if a == nil || a.ID == "" {
		return goerr.New("article id is empty", goerr.T(errutil.TagInvalidInput))
	}
	docRef := f.client.Collection(collectionArticles).Doc(string(a.ID))
	if _, err := docRef.Create(ctx, a); err != nil {
		return goerr.Wrap(err, "create article", goerr.V("id", a.ID))
	}
	return nil
}

func (f *Firestore) UpdateArticle(ctx context.Context, a *model.Article) error {
	if a == nil || a.ID == "" {
		return goerr.New("article id is empty", goerr.T(errutil.TagInvalidInput))
	}
	docRef := f.client.Collection(collectionArticles).Doc(string(a.ID))
	if _, err := docRef.Set(ctx, a); err != nil {
		return goerr.Wrap(err, "update article", goerr.V("id", a.ID))
	}
	return nil
}
