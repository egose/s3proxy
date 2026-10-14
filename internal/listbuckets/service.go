package listbuckets

import (
	"slices"
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
	sort.Slice(views, func(i, j int) bool {
		return views[i].Name < views[j].Name
	})
	return &service{
		buckets: views,
	}
}

type service struct {
	buckets []BucketView
}

func (s *service) List(principal *auth.Principal) []BucketView {
	if len(s.buckets) == 0 {
		return nil
	}
	if principal == nil || slices.Contains(principal.VisibleBuckets, "*") {
		return append([]BucketView(nil), s.buckets...)
	}
	if len(principal.VisibleBuckets) == 0 {
		return nil
	}
	visible := make(map[string]struct{}, len(principal.VisibleBuckets))
	for _, name := range principal.VisibleBuckets {
		visible[name] = struct{}{}
	}
	var views []BucketView
	for _, b := range s.buckets {
		if _, ok := visible[b.Name]; !ok {
			continue
		}
		if views == nil {
			views = make([]BucketView, 0, min(len(s.buckets), len(visible)))
		}
		views = append(views, b)
	}
	return views
}
