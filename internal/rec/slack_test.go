package rec

import "testing"

func TestSummarySection(t *testing.T) {
	tests := []struct {
		name    string
		minutes string
		want    string
	}{
		{
			name:    "概要の後に次の見出しが続く場合_概要だけを返すこと",
			minutes: "# 議事録\n\n## 概要\n\n設計を決めた。\n\n## 決定事項\n\n- A を採用",
			want:    "## 概要\n\n設計を決めた。",
		},
		{
			name:    "概要が最後の見出しの場合_末尾までを返すこと",
			minutes: "## 決定事項\n\n- A\n\n## 概要\n\n設計を決めた。\n",
			want:    "## 概要\n\n設計を決めた。",
		},
		{
			name:    "見出しレベルが違う場合_そのまま拾うこと",
			minutes: "# 概要\n\n設計を決めた。\n\n# TODO\n\n- B",
			want:    "# 概要\n\n設計を決めた。",
		},
		{
			name:    "概要の見出しが無い場合_空を返すこと",
			minutes: "## 決定事項\n\n- A を採用\n",
			want:    "",
		},
		{
			name:    "見出しが一つも無い場合_空を返すこと",
			minutes: "ただの本文\n",
			want:    "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := summarySection(tt.minutes); got != tt.want {
				t.Errorf("summarySection() = %q, want %q", got, tt.want)
			}
		})
	}
}
