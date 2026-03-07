package biz

import "testing"

func TestParseExpandedKeywordsFromAIResponse_Array(t *testing.T) {
	resp := `["AI自动化办公 在物流报关单据自动识别与录入中的实际应用","AI自动化办公 在口腔诊所预约管理中的实际应用"]`
	keywords := parseExpandedKeywordsFromAIResponse(resp)
	if len(keywords) != 2 {
		t.Fatalf("expected 2 keywords, got %d", len(keywords))
	}
}

func TestParseExpandedKeywordsFromAIResponse_Object(t *testing.T) {
	resp := `{"keywords":["A","B"]}`
	keywords := parseExpandedKeywordsFromAIResponse(resp)
	if len(keywords) != 2 {
		t.Fatalf("expected 2 keywords, got %d", len(keywords))
	}
}

func TestParseExpandedKeywordsFromAIResponse_MarkdownList(t *testing.T) {
	resp := `
1. AI自动化办公 在物流报关单据自动识别与录入中的实际应用
2. AI自动化办公 在口腔诊所预约管理中的实际应用
`
	keywords := parseExpandedKeywordsFromAIResponse(resp)
	if len(keywords) != 2 {
		t.Fatalf("expected 2 keywords, got %d", len(keywords))
	}
}
