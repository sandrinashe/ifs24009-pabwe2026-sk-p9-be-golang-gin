package post

var samples = []CreateRequest{
	{
		Title: "Memulai Backend dengan Golang dan Gin", Category: "Teknologi", IsPublished: true,
		Excerpt: "Golang menawarkan performa tinggi dengan sintaks yang ringkas; Gin membuatnya praktis untuk REST API.",
		Content: "## Mengapa Golang?\n\nGolang dikompilasi menjadi satu berkas biner sehingga mudah dideploy. Dipadukan dengan **Gin**, kita bisa membangun REST API yang cepat dengan middleware yang rapi.\n\n## Langkah awal\n\n1. Inisialisasi modul dengan `go mod init`.\n2. Pasang Gin dan GORM.\n3. Pisahkan kode per fitur: handler, service, repository.\n\nPola *feature-first* menjaga kode tetap mudah diuji dan dirawat.",
	},
	{
		Title: "Belajar Efektif di Era Kecerdasan Buatan", Category: "Pendidikan", IsPublished: true,
		Excerpt: "AI bukan pengganti belajar, melainkan alat bantu untuk memahami konsep lebih cepat.",
		Content: "Gunakan AI untuk menjelaskan ulang konsep sulit, membuat soal latihan, dan memeriksa pemahamanmu. Namun tetap tulis ringkasan dengan kata-katamu sendiri.\n\n- Tanyakan *mengapa*, bukan hanya *bagaimana*.\n- Verifikasi jawaban dengan sumber tepercaya.\n- Latih diri tanpa bantuan secara berkala.",
	},
	{
		Title: "Rutinitas Pagi Sederhana yang Bertahan Lama", Category: "Gaya Hidup", IsPublished: true,
		Excerpt: "Tiga kebiasaan kecil yang bisa dilakukan siapa saja tanpa perlu bangun jam empat pagi.",
		Content: "Mulailah dari yang kecil: minum air, rapikan tempat tidur, tulis tiga prioritas hari ini. Konsistensi lebih penting daripada intensitas.",
	},
	{
		Title: "Membangun Karier Lewat Proyek Nyata", Category: "Bisnis", IsPublished: true,
		Excerpt: "Portofolio proyek nyata sering lebih meyakinkan daripada daftar sertifikat.",
		Content: "Rekruter ingin melihat bagaimana kamu memecahkan masalah. Bangun satu proyek utuh, deploy, dokumentasikan keputusan teknisnya, lalu bagikan.",
	},
	{
		Title: "Mengapa Menulis Itu Cara Berpikir", Category: "Opini", IsPublished: true,
		Excerpt: "Menulis memaksa kita merapikan gagasan yang tadinya hanya samar di kepala.",
		Content: "Ketika menulis, celah logika langsung terlihat. Karena itu menulis bukan sekadar menyampaikan hasil berpikir, melainkan bagian dari proses berpikir itu sendiri.",
	},
	{
		Title: "Catatan Draf: Rencana Konten Bulan Depan", Category: "Umum", IsPublished: false,
		Excerpt: "Draf internal.", Content: "Daftar ide yang belum siap terbit.",
	},
}
