package cli

import "testing"

func TestParseAffidamentiSort(t *testing.T) {
	cases := []struct {
		field, dir string
		wantField  string
		wantDesc   bool
		wantErr    bool
	}{
		{"", "", "", false, false},
		{"data", "", "data", false, false},
		{"Data", "desc", "data", true, false},
		{"importo", "ASC", "importo", false, false},
		{"", "desc", "", false, true},
		{"dataPubblicazione", "", "", false, true},
		{"data", "giu", "", false, true},
	}
	for _, c := range cases {
		f, d, err := parseAffidamentiSort(c.field, c.dir)
		if (err != nil) != c.wantErr || f != c.wantField || d != c.wantDesc {
			t.Errorf("parseAffidamentiSort(%q,%q) = %q,%v,%v; want %q,%v,err=%v", c.field, c.dir, f, d, err, c.wantField, c.wantDesc, c.wantErr)
		}
	}
}

func TestSortAffidamenti(t *testing.T) {
	rows := func() []affidamentoRow {
		return []affidamentoRow{
			{Data: "2025-03-01", Importo: 50, CIG: "b"},
			{Data: "", Importo: 10, CIG: "vuota"},
			{Data: "2024-01-15", Importo: 300, CIG: "a"},
			{Data: "2025-03-01", Importo: 20, CIG: "c"},
		}
	}
	cigs := func(rs []affidamentoRow) string {
		s := ""
		for _, r := range rs {
			s += r.CIG + " "
		}
		return s
	}
	cases := []struct {
		field string
		desc  bool
		want  string
	}{
		{"", false, "b vuota a c "},        // nessun ordinamento
		{"data", false, "a b c vuota "},    // stabile a parità di data, vuota in fondo
		{"data", true, "b c a vuota "},     // vuota in fondo anche in desc
		{"importo", false, "vuota c b a "}, // numerico, non lessicale
		{"importo", true, "a b c vuota "},
	}
	for _, c := range cases {
		rs := rows()
		sortAffidamenti(rs, c.field, c.desc)
		if got := cigs(rs); got != c.want {
			t.Errorf("sortAffidamenti(%q, desc=%v) = %q; want %q", c.field, c.desc, got, c.want)
		}
	}
}
