package guide

import (
	"lunabox/internal/enums"
	"lunabox/internal/models"
	"net/http"
	"time"
)

type GuideNavi struct {
	Name string
	Link string
}

// type GuideContent struct {
// 	Name     string
// 	Link     string
// 	Content  string
// 	SaveLink string
// 	Source   enums.GuideSource
// }

type Guider interface {
	FetchGuideList(searchName string, source enums.GuideSource) ([]GuideNavi, error)
	FetchGuide(searchName string, found []GuideNavi, source enums.GuideSource) (models.GuideContent, error)
}

type SeiyaGuider struct {
	client  *http.Client
	timeout time.Duration
}

func NewSeiyaGuider() *SeiyaGuider {
	return &SeiyaGuider{
		client:  &http.Client{},
		timeout: 10 * time.Second,
	}
}

var _ Guider = (*SeiyaGuider)(nil)

func (s *SeiyaGuider) FetchGuide(searchName string, found []GuideNavi, source enums.GuideSource) (models.GuideContent, error) {
	rs := models.GuideContent{}
	return rs, nil
}

func (s *SeiyaGuider) FetchGuideList(searchName string, source enums.GuideSource) ([]GuideNavi, error) {
	rs := []GuideNavi{}
	return rs, nil
}
