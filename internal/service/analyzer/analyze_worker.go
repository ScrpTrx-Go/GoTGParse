package analyzer

import (
	"context"
	"strings"
	"time"

	"github.com/ScrpTrx-Go/GoTGParse/internal/domain/model"
	pkg "github.com/ScrpTrx-Go/GoTGParse/pkg/logger"
	"github.com/cloudflare/ahocorasick"
)

type AnalyzeWorker struct {
	matchers Matchers
	log      pkg.Logger
	regions  []string
	dict     Dictionaries
	count    int
	skipped  int
}

func NewAnalyzeWorker(factory MatchersCreator, log pkg.Logger, regions []string, dict Dictionaries) AnalyzePostWorker {
	return &AnalyzeWorker{
		matchers: factory.CreateMatchers(),
		log:      log,
		regions:  regions,
		dict:     dict,
	}
}

func (a *AnalyzeWorker) Run(ctx context.Context, in <-chan *model.Post, out chan<- *model.Post) {
	start := time.Now()
	for post := range in {
		select {
		case <-ctx.Done():
			a.log.Warn("Context canceled in analyzer worker")
			return
		default:
		}

		if !a.IsErrand(post) {
			a.skipped++
			continue
		}

		a.count++
		postRegions := a.ExtractRegions(post)
		post.Regions = postRegions

		errandType := a.ErrandType(post)
		post.ErrandType = errandType

		select {
		case <-ctx.Done():
			a.log.Warn("Context canceled during post output")
			return
		case out <- post:
		}
	}
	duration := time.Since(start)
	a.log.Info("AnalyzeWorker completed", "matched", a.count, "skipped", a.skipped, "duration", duration.String())
}

func (a *AnalyzeWorker) IsErrand(post *model.Post) bool {
	switch post.Username {
	case "sledcom_press":
		if a.CheckErrandTitle(post) {
			return true
		}
	case "infocentrskrf":
		if a.TitleHasPrefix(post) {
			return true
		}
	}
	return false
}

func (a *AnalyzeWorker) TitleHasPrefix(post *model.Post) bool {
	matches := a.matchers.PrefixICMatcher.Match([]byte(post.Text))
	return len(matches) > 0
}

func (a *AnalyzeWorker) CheckErrandTitle(post *model.Post) bool {
	errandMatchers := []*ahocorasick.Matcher{a.matchers.PrefixMatcher, a.matchers.VerbMatcher, a.matchers.PSKMatcher}
	errandTitle := a.GetLowTitle(post.Text)
	matchesCounter := 0
	for _, errandMatch := range errandMatchers {
		matches := errandMatch.Match([]byte(errandTitle))
		if len(matches) > 0 {
			matchesCounter++
		}
	}

	if matchesCounter == 2 {
		post.ErrorType = "maybe errand"
		return true
	}
	return matchesCounter == 3
}

func (a *AnalyzeWorker) GetLowTitle(text string) string {
	split := strings.SplitN(text, "\n", 2)
	title := strings.ToLower(strings.Join(strings.Fields(split[0]), " "))
	return title
}

func (a *AnalyzeWorker) ExtractRegions(post *model.Post) []string {
	text := a.FindErrandBody(post)
	matches := a.matchers.Regions.Match([]byte(text))

	if len(matches) == 0 {
		loweredText := strings.ToLower(post.Text)
		matchesFull := a.matchers.Regions.Match([]byte(loweredText))
		errandRegions := a.FoundRegionsName(matchesFull)
		result := a.CheckException(errandRegions)
		if len(matchesFull) > 1 {
			return nil
		}
		return result
	}
	errandRegions := a.FoundRegionsName(matches)
	result := a.CheckException(errandRegions)
	return result
}

func (a *AnalyzeWorker) CheckException(errandRegions []string) []string {
	for _, region := range errandRegions {
		for _, exception := range a.dict.ExceptionsDictonary {
			if region == exception {
				errandRegions = []string{exception}
				break
			}
		}
	}
	return errandRegions
}

func (a *AnalyzeWorker) FindErrandBody(post *model.Post) string {
	loweredText := strings.ToLower(post.Text)
	paragraphs := strings.Split(loweredText, "\n")
	var maxLen int
	var errandBody string
	for idx, para := range paragraphs {
		if idx == 0 {
			continue
		}
		currentLen := len(a.matchers.ErrandBodyMatcher.Match([]byte(para)))
		if currentLen > maxLen {
			maxLen = currentLen
			errandBody = para
		}
	}
	normalizedErrandBody := strings.TrimSpace(errandBody)
	if normalizedErrandBody == "" {
		normalizedErrandBody = strings.ToLower(strings.TrimSpace(post.Text))
	}
	return normalizedErrandBody
}

func (a *AnalyzeWorker) FoundRegionsName(matches []int) []string {
	seen := make(map[string]struct{})
	errandRegions := make([]string, 0)

	for _, match := range matches {
		if region, ok := a.dict.RegionsAllias[a.regions[match]]; ok {
			if _, ok := seen[region]; !ok {
				seen[region] = struct{}{}
				errandRegions = append(errandRegions, region)
			}
		}
	}
	return errandRegions
}

func (a *AnalyzeWorker) ErrandType(post *model.Post) bool {
	text := a.FindErrandBody(post)
	matches := a.matchers.ErrandType.Match([]byte(text))
	return len(matches) > 0
}
