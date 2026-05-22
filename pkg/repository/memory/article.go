package memory

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/utils/errutil"
)

func (m *Memory) GetArticleByURL(_ context.Context, url string) (*model.Article, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	id, ok := m.articleByURL[url]
	if !ok {
		return nil, notFound("article", url)
	}
	return cloneArticle(m.articles[id]), nil
}

func (m *Memory) CreateArticle(_ context.Context, a *model.Article) error {
	if a == nil || a.ID == "" {
		return goerr.New("article id is empty", goerr.T(errutil.TagInvalidInput))
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.articleByURL[a.URL]; exists {
		return goerr.New("article already exists",
			goerr.V("url", a.URL),
			goerr.T(errutil.TagConflict))
	}
	c := *a
	m.articles[a.ID] = &c
	m.articleByURL[a.URL] = a.ID
	return nil
}

func (m *Memory) UpdateArticle(_ context.Context, a *model.Article) error {
	if a == nil || a.ID == "" {
		return goerr.New("article id is empty", goerr.T(errutil.TagInvalidInput))
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.articles[a.ID]; !ok {
		return notFound("article", a.ID)
	}
	c := *a
	m.articles[a.ID] = &c
	m.articleByURL[a.URL] = a.ID
	return nil
}
