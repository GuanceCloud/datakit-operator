package selector

import (
	"k8s.io/apimachinery/pkg/labels"
)

type Selector interface {
	Matches(labels map[string]string) bool
}

func ParseSelector(s string) (Selector, error) {
	parsedSelector, err := labels.Parse(s)
	if err != nil {
		return nil, err
	}
	return &selector{parsedSelector}, nil
}

type selector struct {
	p labels.Selector
}

func (s *selector) Matches(m map[string]string) bool {
	return s.p.Matches(labels.Set(m))
}
