package analyzer

import (
	"github.com/cloudflare/ahocorasick"
)

type DefaultMatchersCreator struct {
	dict    *Dictionaries
	regions []string
}

type Matchers struct {
	PrefixMatcher     *ahocorasick.Matcher
	PrefixICMatcher   *ahocorasick.Matcher
	VerbMatcher       *ahocorasick.Matcher
	PSKMatcher        *ahocorasick.Matcher
	ErrandBodyMatcher *ahocorasick.Matcher
	Regions           *ahocorasick.Matcher
	ErrandType        *ahocorasick.Matcher
}

func NewMatcherCreator(dict *Dictionaries, regions []string) MatchersCreator {
	return &DefaultMatchersCreator{
		dict:    dict,
		regions: regions,
	}
}

func (m *DefaultMatchersCreator) CreateMatchers() Matchers {
	return Matchers{
		PrefixMatcher:     ahocorasick.NewStringMatcher(m.dict.PrefixDictionary),
		PrefixICMatcher:   ahocorasick.NewStringMatcher(m.dict.PrefixDictionaryIC),
		VerbMatcher:       ahocorasick.NewStringMatcher(m.dict.VerbsDictionary),
		PSKMatcher:        ahocorasick.NewStringMatcher(m.dict.PSKDictionary),
		ErrandBodyMatcher: ahocorasick.NewStringMatcher(m.dict.ErrandBodyDictionary),
		Regions:           ahocorasick.NewStringMatcher(m.regions),
		ErrandType:        ahocorasick.NewStringMatcher(m.dict.ErrandTypesDictionary),
	}
}
