package listbuckets

import (
	"sort"
	"time"

	"github.com/egose/s3proxy/internal/auth"
	"github.com/egose/s3proxy/internal/config"
)

type BucketView struct {
	Name         string
	CreationDate time.Time
}

type Service interface {
	List(principal *auth.Principal) []BucketView
}

func New(buckets []config.VirtualBucket, startupTime time.Time) Service {
	views := make([]BucketView, 0, len(buckets))
	for _, bucket := range buckets {
		views = append(views, BucketView{Name: bucket.VisibleName, CreationDate: startupTime})
	}
	return &service{
		buckets: views,
	}
}

type service struct {
	buckets []BucketView
}

func (s *service) List(principal *auth.Principal) []BucketView {
	var views []BucketView
	for _, b := range s.buckets {
		if !isVisible(principal, b.Name) {
			continue
		}
		views = append(views, b)
	}
	sort.Slice(views, func(i, j int) bool {
		return views[i].Name < views[j].Name
	})
	return views
}

func isVisible(p *auth.Principal, visibleName string) bool {
	if p == nil {
		return true
	}
	for _, v := range p.VisibleBuckets {
		if v == "*" || v == visibleName {
			return true
		}
	}
	return false
}
