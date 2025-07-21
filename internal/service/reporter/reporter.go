package reporter

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"baliance.com/gooxml/document"
	"baliance.com/gooxml/schema/soo/wml"
	"github.com/ScrpTrx-Go/GoTGParse/internal/domain/contracts"
	"github.com/ScrpTrx-Go/GoTGParse/internal/domain/model"
	pkg "github.com/ScrpTrx-Go/GoTGParse/pkg/logger"
	"github.com/xuri/excelize/v2"
)

type Reporter struct {
	log pkg.Logger
	db  contracts.SaverPostgres
}

func NewReporter(log pkg.Logger, db contracts.SaverPostgres) *Reporter {
	return &Reporter{
		log: log,
		db:  db,
	}
}

func (r *Reporter) GenerateFullReport(ctx context.Context, from, to time.Time) error {
	r.log.Info("Generating full report", "from", from, "to", to)

	posts, err := r.db.GetPostsByPeriod(ctx, from, to)
	if err != nil {
		r.log.Error("Failed to fetch posts from DB by period", "err", err)
		return err
	}

	if len(posts) == 0 {
		r.log.Warn("No posts found for report period", "from", from, "to", to)
		return nil
	}

	rd := NewReportData(r.log)
	rd.Process(posts)

	if err := rd.SaveAll(); err != nil {
		r.log.Error("Failed to save report", "err", err)
		return err
	}

	r.log.Info("Report generation completed successfully")
	return nil
}

type ReportData struct {
	log    pkg.Logger
	sled   []*SledcomPress
	ic     []*RegionCounter
	errors map[string][]*model.Post
}

type RegionCounter struct {
	RegionName          string
	CasualErrandCounter int
	SpecErrandCounter   int
}

type SledcomPress struct {
	Info  RegionCounter
	Posts []*model.Post
}

func NewReportData(log pkg.Logger) *ReportData {
	return &ReportData{
		log:    log,
		errors: make(map[string][]*model.Post),
	}
}

func (r *ReportData) Process(posts []*model.Post) {
	r.log.Info("Processing posts", "total", len(posts))
	for _, post := range posts {
		if checkError(post) {
			r.errors[post.ErrorType] = append(r.errors[post.ErrorType], post)
			continue
		}
		switch post.Username {
		case "sledcom_press":
			r.addSledcom(post)
		case "infocentrskrf":
			r.addIC(post)
		}
	}
	r.log.Info("Finished processing posts", "sledcom", len(r.sled), "infocentrskrf", len(r.ic), "errors", len(r.errors))
}

func checkError(post *model.Post) bool {
	if post.ErrorType != "" {
		return true
	}
	if len(post.Regions) == 0 {
		post.ErrorType = "Not found region!"
		return true
	}
	if strings.TrimSpace(post.Text) == "" {
		post.ErrorType = "Empty text!"
		return true
	}
	if len(post.Regions) > 1 && post.ErrandType {
		post.ErrorType = "more than 1 region, but errand type is true"
		return true
	}
	return false
}

func (r *ReportData) addSledcom(post *model.Post) {
	for _, region := range post.Regions {
		found := false
		for _, entry := range r.sled {
			if entry.Info.RegionName == region {
				entry.Posts = append(entry.Posts, post)
				if post.ErrandType {
					entry.Info.SpecErrandCounter++
				} else {
					entry.Info.CasualErrandCounter++
				}
				found = true
				break
			}
		}
		if !found {
			e := &SledcomPress{
				Info:  RegionCounter{RegionName: region},
				Posts: []*model.Post{post},
			}
			if post.ErrandType {
				e.Info.SpecErrandCounter++
			} else {
				e.Info.CasualErrandCounter++
			}
			r.sled = append(r.sled, e)
		}
	}
}

func (r *ReportData) addIC(post *model.Post) {
	for _, region := range post.Regions {
		found := false
		for _, entry := range r.ic {
			if entry.RegionName == region {
				if post.ErrandType {
					entry.SpecErrandCounter++
				} else {
					entry.CasualErrandCounter++
				}
				found = true
				break
			}
		}
		if !found {
			e := &RegionCounter{RegionName: region}
			if post.ErrandType {
				e.SpecErrandCounter++
			} else {
				e.CasualErrandCounter++
			}
			r.ic = append(r.ic, e)
		}
	}
}

func (r *ReportData) SaveAll() error {
	r.log.Info("Saving all report files")
	if err := r.saveDocSledcom(); err != nil {
		r.log.Error("Failed to save sledcom.docx", "err", err)
		return err
	}
	if err := r.saveErrorDoc(); err != nil {
		r.log.Error("Failed to save errors.docx", "err", err)
		return err
	}
	if err := r.saveExcel(); err != nil {
		r.log.Error("Failed to save Excel report", "err", err)
		return err
	}
	r.log.Info("All report files saved successfully")
	return nil
}

func (r *ReportData) saveDocSledcom() error {
	doc := document.New()
	for _, region := range r.sled {
		para := doc.AddParagraph()
		run := para.AddRun()
		para.Properties().SetAlignment(wml.ST_JcCenter)
		para.SetStyle("Heading1")
		run.Properties().SetBold(true)
		run.AddText(region.Info.RegionName)

		sort.Slice(region.Posts, func(i, j int) bool {
			return region.Posts[i].Timestamp.Before(region.Posts[j].Timestamp)
		})

		for _, post := range region.Posts {
			doc.AddParagraph().AddRun().AddText(fmt.Sprintf("Время публикации: %v", post.Timestamp.Format("2006-01-02 15:04:05")))
			doc.AddParagraph().AddRun().AddText(strings.Join(post.Regions, ", "))

			lines := strings.Split(post.Text, "\n")
			for idx, line := range lines {
				para := doc.AddParagraph()
				run := para.AddRun()
				if idx == 0 {
					run.Properties().SetBold(true)
				}
				para.Properties().SetAlignment(wml.ST_JcBoth)
				run.AddText(strings.TrimSpace(line))
			}

			para := doc.AddParagraph()
			hl := para.AddHyperLink()
			hl.SetTarget(post.Link)
			run := hl.AddRun()
			run.Properties().SetStyle("Hyperlink")
			run.AddText("Открыть пост в Telegram")

			doc.AddParagraph().AddRun().AddText("----------")
		}
	}
	return doc.SaveToFile("reports/sledcom.docx")
}

func (r *ReportData) saveErrorDoc() error {
	doc := document.New()
	for errType, posts := range r.errors {
		doc.AddParagraph().AddRun().AddText(errType)
		for _, post := range posts {
			doc.AddParagraph().AddRun().AddText(fmt.Sprintf("Время публикации: %v", post.Timestamp.Format("2006-01-02 15:04:05")))
			lines := strings.Split(post.Text, "\n")
			for idx, line := range lines {
				para := doc.AddParagraph()
				run := para.AddRun()
				if idx == 0 {
					run.Properties().SetBold(true)
				}
				para.Properties().SetAlignment(wml.ST_JcBoth)
				run.AddText(strings.TrimSpace(line))
			}
			para := doc.AddParagraph()
			hl := para.AddHyperLink()
			hl.SetTarget(post.Link)
			run := hl.AddRun()
			run.Properties().SetStyle("Hyperlink")
			run.AddText("Открыть пост в Telegram")
			doc.AddParagraph().AddRun().AddText("----------")
		}
	}
	return doc.SaveToFile("reports/errors.docx")
}

func (r *ReportData) saveExcel() error {
	src, err := filepath.Abs("reports/template.xlsx")
	if err != nil {
		return err
	}
	dst, err := filepath.Abs("reports/report.xlsx")
	if err != nil {
		return err
	}
	if err := CopyTemplate(src, dst); err != nil {
		return err
	}
	f, err := excelize.OpenFile(dst)
	if err != nil {
		return err
	}
	defer f.Close()

	sheet := "Показатели"
	for row := 6; row <= 95; row++ {
		cell := fmt.Sprintf("D%d", row)
		regionName, err := f.GetCellValue(sheet, cell)
		if err != nil || strings.TrimSpace(regionName) == "" {
			continue
		}
		regionName = strings.TrimSpace(regionName)

		for _, region := range r.sled {
			if region.Info.RegionName == regionName {
				f.SetCellValue(sheet, fmt.Sprintf("E%d", row), region.Info.CasualErrandCounter)
				f.SetCellValue(sheet, fmt.Sprintf("F%d", row), region.Info.SpecErrandCounter)
			}
		}
		for _, region := range r.ic {
			if region.RegionName == regionName {
				f.SetCellValue(sheet, fmt.Sprintf("G%d", row), region.CasualErrandCounter)
				f.SetCellValue(sheet, fmt.Sprintf("H%d", row), region.SpecErrandCounter)
			}
		}
	}

	return f.Save()
}

func CopyTemplate(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}
