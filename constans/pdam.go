package constans

// PDAMKeywords contains all PDAM-related keywords for message filtering
// These keywords are used to determine if a message is related to PDAM services
var PDAMKeywords = []string{
	// ============================================================
	// CORE PDAM KEYWORDS
	// ============================================================
	"pdam",
	"perusahaan daerah air minum",
	"air",
	"tagihan",
	"pelanggan",
	"customer",

	// ============================================================
	// BILLING & PAYMENT
	// ============================================================
	"bayar",
	"pembayaran",
	"cek",
	"rekening",
	"bill",
	"invoice",
	"lunas",
	"denda",
	"tarif",
	"harga",
	"biaya",
	"virtual account",
	"va",
	"transfer",
	"tunggakan",
	"cicilan",
	"angsuran",
	"nomor",
	"nomer",
	"no",
	"struk",
	"bukti bayar",

	// ============================================================
	// METER & USAGE
	// ============================================================
	"meter",
	"meteran",
	"kubik",
	"m3",
	"stand meter",
	"pemakaian",
	"boros",
	"hemat",
	"irit",
	"konsumsi",
	"kwh", // sometimes confused with water meter

	// ============================================================
	// SERVICES & INSTALLATION
	// ============================================================
	"sambungan",
	"pasang",
	"instalasi",
	"pemasangan",
	"balik nama",
	"ganti nama",
	"pindah nama",
	"tutup sambungan",
	"cabut",
	"nonaktif",
	"aktifkan",
	"daftar",
	"pendaftaran",
	"registrasi",

	// ============================================================
	// LOCATION & INFO
	// ============================================================
	"kantor",
	"cabang",
	"lokasi",
	"alamat",
	"jam buka",
	"operasional",
	"call center",
	"telepon",
	"kontak",
	"layanan",
	"pelayanan",
	"customer service",
	"cs",

	// ============================================================
	// COMPLAINTS & ISSUES
	// ============================================================
	"bocor",
	"kebocoran",
	"pipa",
	"mati",
	"macet",
	"keruh",
	"kotor",
	"berbau",
	"kuning",
	"merah",
	"lapor",
	"pengaduan",
	"gangguan",
	"rusak",
	"mampet",
	"tersumbat",
	"kecil",
	"lemah",
	"tidak mengalir",
	"tidak keluar",
	"kering",
	"error",
	"masalah",
	"problem",
	"keluhan",
	"komplain",

	// ============================================================
	// WATER QUALITY & SAFETY
	// ============================================================
	"kualitas",
	"bersih",
	"sehat",
	"aman",
	"minum",
	"jernih",
	"bau",
	"amis",
	"kaporit",
	"klorin",
	"chlorine",

	// ============================================================
	// FACILITIES & FIXTURES
	// ============================================================
	"kran",
	"shower",
	"bak",
	"wc",
	"toilet",
	"closet",
	"sumur",
	"tandon",
	"toren",
	"pompa",
	"filter",
	"saringan",

	// ============================================================
	// PROMOTIONS & SUBSIDIES
	// ============================================================
	"subsidi",
	"diskon",
	"keringanan",
	"bantuan",
	"gratis",
	"murah",
	"promo",
	"potongan",

	// ============================================================
	// COMPLAINT RELATED (PRICE)
	// ============================================================
	"mahal",
	"naik",
	"tinggi",
	"besar",
	"membengkak",
	"melonjak",

	// ============================================================
	// ADMINISTRATIVE
	// ============================================================
	"dokumen",
	"persyaratan",
	"syarat",
	"ktp",
	"kartu keluarga",
	"kk",
	"surat",
	"formulir",
	"berkas",
	"administrasi",
	"data",

	// ============================================================
	// TECHNICAL TERMS
	// ============================================================
	"tekanan",
	"pressure",
	"debit",
	"aliran",
	"volume",
	"kapasitas",
	"perbaikan",
	"maintenance",
	"perawatan",
	"servis",
}

// PDAMGreetingWords contains greeting words that initiate AI conversation
var PDAMGreetingWords = []string{
	"hallo",
	"halo",
	"hello",
	"hi",
	"hai",
	"hey",
	"assalamualaikum",
	"assalamu'alaikum",
	"selamat pagi",
	"selamat siang",
	"selamat sore",
	"selamat malam",
	"pagi",
	"siang",
	"sore",
	"malam",
	"permisi",
	"maaf",
	"mohon",
}

// PDAMClosingConfirmations contains words/phrases that confirm conversation closing
var PDAMClosingConfirmations = []string{
	// Exact match
	"iya",
	"ya",
	"yes",
	"yup",
	"yap",
	"ok",
	"oke",
	"okay",
	"okey",
	"baik",
	"siap",
	"betul",
	"benar",
	"setuju",
	"done",
	"selesai",
	"sudah",
	"cukup",
	"thanks",
	"thank you",
	"terima kasih",
	"makasih",
	"makasi",
}

// PDAMClosingPhrases contains phrases in AI response that indicate conversation should be closed
var PDAMClosingPhrases = []string{
	"bisa kami close",
	"bisa kami tutup",
	"dapat kami close",
	"dapat kami tutup",
	"bisa ditutup",
	"dapat ditutup",
	"sudah selesai kah",
	"sudah selesaikah",
	"apakah sudah selesai",
	"mau di close",
	"mau ditutup",
	"ingin menutup",
	"sudah cukup",
	"ada lagi yang bisa",
	"ada yang bisa kami bantu lagi",
}

// CustomerNumberMinLength minimum length for customer number
const CustomerNumberMinLength = 6

// CustomerNumberMaxLength maximum length for customer number
const CustomerNumberMaxLength = 12

// AlphanumericThreshold percentage threshold for alphanumeric characters in customer number
const AlphanumericThreshold = 0.8
