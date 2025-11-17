package chatagent

import "strings"

type Rule struct {
	Keywords []string
	Response string
}

var Rules = []Rule{
	{
		Keywords: []string{"cara lapor", "buat laporan", "laporan gimana"},
		Response: "Untuk membuat laporan, buka menu 'Buat Laporan', isi detail kejadian, pilih kategori, lalu upload bukti.",
	},
	{
		Keywords: []string{"status laporan", "cek status laporan", "laporan saya"},
		Response: "Silakan berikan ID laporan Anda, nanti saya bantu cek statusnya melalui admin.",
	},
	{
		Keywords: []string{"hubungi admin", "bicara admin", "mau admin"},
		Response: "<handoff>true</handoff> Baik, saya akan menghubungkan Anda ke admin.",
	},
	// Tambahkan rule lain sesuai kebutuhan
}

func MatchRule(text string) (string, bool) {
	lower := strings.ToLower(text)

	for _, rule := range Rules {
		for _, kw := range rule.Keywords {
			if strings.Contains(lower, kw) {
				return rule.Response, true
			}
		}
	}

	return "", false
}
