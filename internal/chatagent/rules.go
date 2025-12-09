package chatagent

import "strings"

type Rule struct {
	Keywords []string
	Response string
}

var Rules = []Rule{
	// === FITUR UTAMA ===
	{
		Keywords: []string{"cara lapor", "buat laporan", "laporan gimana", "cara buat laporan", "membuat laporan", "mau lapor"},
		Response: "Untuk membuat laporan:\n1. Buka menu 'Buat Laporan'\n2. Isi detail kejadian yang ingin dilaporkan\n3. Pilih kategori yang sesuai\n4. Upload bukti (foto/dokumen)\n5. Klik 'Kirim Laporan'\n\nLaporan Anda akan diproses oleh tim kami.",
	},
	{
		Keywords: []string{"upload bukti", "cara upload", "upload dokumen", "lampirkan file", "kirim bukti", "attach file"},
		Response: "Untuk upload bukti:\n1. Saat membuat laporan, klik tombol 'Upload Bukti'\n2. Pilih file dari galeri atau kamera\n3. Anda bisa upload foto, PDF, atau dokumen\n4. Maksimal ukuran file 10MB\n\nBukti yang kuat akan mempercepat proses investigasi.",
	},
	{
		Keywords: []string{"status laporan", "cek status", "laporan saya", "track laporan", "lacak laporan", "laporan diproses"},
		Response: "Untuk cek status laporan:\n1. Buka menu 'Laporan Saya'\n2. Pilih laporan yang ingin dicek\n3. Anda akan melihat status terkini: 'Pending', 'Diproses', atau 'Selesai'\n\nAnda juga akan mendapat notifikasi jika ada update.",
	},
	
	// === PRIVASI & KEAMANAN ===
	{
		Keywords: []string{"anonim", "rahasia", "privasi", "keamanan data", "identitas", "nama saya aman", "dilacak"},
		Response: "Laporan Anda dijamin ANONIM dan AMAN. Identitas pelapor dilindungi sepenuhnya sesuai kebijakan privasi kami. Data Anda terenkripsi dan hanya diakses oleh tim investigasi yang berwenang.",
	},
	
	// === FAQ TAMBAHAN ===
	{
		Keywords: []string{"kategori laporan", "jenis laporan", "pilih kategori"},
		Response: "Kategori laporan yang tersedia:\n• Korupsi\n• Pelecehan\n• Pelanggaran Etika\n• Fraud/Penipuan\n• Kesehatan & Keselamatan\n• Lainnya\n\nPilih kategori yang paling sesuai dengan laporan Anda.",
	},
	{
		Keywords: []string{"berapa lama", "waktu proses", "lama diproses", "kapan selesai"},
		Response: "Waktu proses laporan:\n• Verifikasi awal: 1-3 hari kerja\n• Investigasi: 7-14 hari kerja\n• Tindak lanjut: Tergantung kompleksitas kasus\n\nAnda akan mendapat notifikasi setiap ada perkembangan.",
	},
	{
		Keywords: []string{"edit laporan", "ubah laporan", "ganti laporan", "revisi laporan"},
		Response: "Laporan yang sudah terkirim tidak dapat diedit langsung untuk menjaga integritas data. Jika ada informasi tambahan, Anda bisa:\n1. Hubungi admin melalui fitur chat\n2. Atau buat laporan baru dengan keterangan 'Update dari laporan ID: xxx'",
	},
	{
		Keywords: []string{"hapus laporan", "batalkan laporan", "cancel laporan"},
		Response: "Untuk membatalkan laporan, silakan hubungi admin melalui fitur chat dengan menyertakan ID laporan Anda. Admin akan membantu proses pembatalan.",
	},
	{
		Keywords: []string{"notifikasi", "pemberitahuan", "tidak dapat notif", "aktifkan notifikasi"},
		Response: "Untuk mengaktifkan notifikasi:\n1. Buka Pengaturan > Notifikasi\n2. Aktifkan 'Pemberitahuan Laporan'\n3. Izinkan akses notifikasi di pengaturan HP\n\nAnda akan mendapat update real-time tentang status laporan.",
	},
}

func MatchRule(text string) (string, bool) {
	lower := strings.ToLower(text)
	lower = strings.TrimSpace(lower)

	for _, rule := range Rules {
		for _, kw := range rule.Keywords {
			if strings.Contains(lower, kw) {
				return rule.Response, true
			}
		}
	}

	return "", false
}