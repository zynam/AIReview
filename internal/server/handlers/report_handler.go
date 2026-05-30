package handlers

import (
	"bytes"
	"net/http"

	"aireview/internal/github"
	reportout "aireview/internal/report"
	"aireview/internal/review"
	"aireview/internal/session"

	"github.com/gin-gonic/gin"
)

type ReportHandler struct {
	Service session.Service
}

func (h ReportHandler) Markdown(c *gin.Context) {
	detail, err := h.Service.Detail(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeStoreError(c, err)
		return
	}

	var body bytes.Buffer
	ref := github.PRRef{Owner: detail.Owner, Repo: detail.Repo, Number: detail.PRNumber}
	report := review.ReviewReport{
		Summary:        detail.Summary,
		Impact:         detail.Impact,
		Findings:       detail.Findings,
		TestAssessment: detail.TestAssessment,
		SkippedFiles:   detail.SkippedFiles,
	}
	if err := reportout.WriteMarkdown(&body, ref, report, nil); err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.Data(http.StatusOK, "text/markdown; charset=utf-8", body.Bytes())
}
