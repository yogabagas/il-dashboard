#!/bin/bash

###############################################################################
# PDAM Knowledge Base Seeder Script
#
# Description:
#   This script seeds AI knowledge base with PDAM-related Q&A data
#   by calling the create API endpoint for each knowledge entry.
#
# Usage:
#   ./seed_pdam_knowledge.sh <base_url> <client_id> <auth_token>
#
# Example:
#   ./seed_pdam_knowledge.sh http://localhost:8080 pdam-jakarta eyJhbGciOiJIUzI1NiIs...
#
###############################################################################

# Configuration
BASE_URL="${1:-https://ilbedev.kitamandiri.com}"
CLIENT_ID="${2:-01JPETM6DBJ1TPSKZNKNSE5HB6}"
AUTH_TOKEN="${3:-eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJjbGllbnRJZCI6IjAxSlBFVE02REJKMVRQU0taTktOU0U1SEI2IiwiZXhwIjoxNzY0NzI4NjM5LCJpYXQiOjE3NjQ2NDIyMzksIm5iZiI6MTc2NDY0MjIzOSwic2Vzc2lvbklkIjoiMDFLQkVEWkNOWDBIRks1Q0pGRjE4RFI4R0giLCJ1c2VySWQiOiIwMUtBQTg1UURWR1dXM1FYUVlHU1dWUTU4MiJ9.AzdnwpBMxYUtazg72jFPmyNf__V4GLzTE8kqQAqZlgBCQqhXjmxQy5K3f-C3bsEnPQTJyv_Jl93IGxLKBRuRGipcU3xI1MadKW5Z9FEEtXD7aBxF_jkM6mRM9qFNTaroZD5O4LEtjjIjdDVomiu9vlT6bCL76dRPl2iOD0Flx46CskcZSmuTxZ-Q74dPT6g1jH_4gBgR9niQYUlR5T_S3L-4m2J_8yN4o_qz2kH3IhQKHqpwtu5x-zXJZC4g922r6zMHYJz5eu557dZ-YdYGyorY0tDMV_NZP9KBBgEZUb8ixtUGrSbab3pIaJzwEyKAxZwReQxTHcs2RmM9z3xZEw}"  # Pass token as third argument
API_ENDPOINT="${BASE_URL}/api/v1/instant-link/ai/knowledge-base/create"

# Check if AUTH_TOKEN is provided
if [ -z "$AUTH_TOKEN" ]; then
  echo "ERROR: Authentication token is required!"
  echo "Usage: $0 <base_url> <client_id> <auth_token>"
  echo "Example: $0 http://localhost:8080 pdam-jakarta eyJhbGciOiJIUzI1NiIs..."
  exit 1
fi

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Counters
SUCCESS_COUNT=0
FAILED_COUNT=0
TOTAL_COUNT=0

echo "========================================="
echo "PDAM Knowledge Base Seeder"
echo "========================================="
echo "Base URL: $BASE_URL"
echo "Client ID: $CLIENT_ID"
echo ""

# Function to insert knowledge
insert_knowledge() {
    local category=$1
    local question=$2
    local answer=$3
    local keywords=$4
    local priority=$5

    TOTAL_COUNT=$((TOTAL_COUNT + 1))

    # Escape JSON special characters
    question=$(echo "$question" | sed 's/"/\\"/g' | sed "s/'/\\'/g")
    answer=$(echo "$answer" | sed 's/"/\\"/g' | sed "s/'/\\'/g")
    keywords=$(echo "$keywords" | sed 's/"/\\"/g' | sed "s/'/\\'/g")

    # Create JSON payload
    payload=$(cat <<EOF
{
  "client_id": "$CLIENT_ID",
  "category": "$category",
  "question": "$question",
  "answer": "$answer",
  "keywords": "$keywords",
  "priority": $priority
}
EOF
)

    # Make API call
    response=$(curl -s -w "\n%{http_code}" -X POST "$API_ENDPOINT" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $AUTH_TOKEN" \
        -d "$payload")

    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')

    if [ "$http_code" -eq 200 ]; then
        echo -e "${GREEN}✓${NC} [$TOTAL_COUNT] $category: ${question:0:50}..."
        SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
    else
        echo -e "${RED}✗${NC} [$TOTAL_COUNT] Failed: $question"
        echo "  HTTP $http_code: $body"
        FAILED_COUNT=$((FAILED_COUNT + 1))
    fi
}

echo "Starting seeding process..."
echo ""

###############################################################################
# KATEGORI: LAYANAN
###############################################################################

insert_knowledge "layanan" \
    "Apa saja jam operasional pelayanan PDAM?" \
    "Pelayanan PDAM buka:\n- Senin-Jumat: 08:00 - 16:00 WIB\n- Sabtu: 08:00 - 12:00 WIB\n- Minggu & Libur: Tutup\n\nUntuk pengaduan darurat 24 jam hubungi call center 1500-123" \
    "jam buka, jam operasional, jam kerja, buka tutup, operasional, jadwal" \
    100

insert_knowledge "layanan" \
    "Bagaimana cara memasang sambungan air baru?" \
    "Untuk pemasangan sambungan baru:\n1. Datang ke kantor PDAM terdekat dengan membawa:\n   - KTP asli + fotocopy\n   - KK (Kartu Keluarga)\n   - IMB/Surat Keterangan Rumah\n   - Bukti pembayaran PBB tahun terakhir\n2. Isi formulir pendaftaran\n3. Lakukan pembayaran biaya pasang (Rp 2.500.000 - 5.000.000 tergantung jenis)\n4. Petugas akan survei lokasi (3-5 hari kerja)\n5. Pemasangan dilakukan 7-14 hari setelah survei disetujui" \
    "pasang baru, sambungan baru, daftar air, instalasi, pemasangan, persyaratan" \
    95

insert_knowledge "layanan" \
    "Dimana lokasi kantor PDAM dan kantor cabang?" \
    "Kantor PDAM Kabupaten Tangerang:\n\n📍 Kantor Pusat:\nJl. Raya Serang Km 11, Balaraja, Tangerang\nTelp: (021) 5951234\n\n📍 Cabang Serpong:\nJl. Raya Serpong No. 25\nTelp: (021) 5371234\n\n📍 Cabang Cikupa:\nJl. Raya Cikupa No. 15\nTelp: (021) 5961234\n\nUntuk lokasi cabang lainnya, silakan cek website: www.pdamkabtgr.co.id/cabang" \
    "lokasi, alamat, kantor, cabang, dimana, tempat, maps, letak, posisi, di mana, ada dimana, cari kantor, kantor terdekat, pdam dimana" \
    90

###############################################################################
# KATEGORI: TAGIHAN
###############################################################################

insert_knowledge "tagihan" \
    "Bagaimana cara cek tagihan air PDAM?" \
    "Anda bisa cek tagihan dengan cara:\n\n💬 Via WhatsApp (Chat ini):\n   Cukup kirim nomor pelanggan Anda langsung\n   Contoh: A532159 atau 1234567890\n   \n   Saya akan langsung carikan informasi tagihan Anda!\n   \n📱 Via Website:\n   Kunjungi: www.pdamkabtgr.co.id/cek-tagihan\n\n🏢 Via Kantor:\n   Datang ke kantor PDAM Kabupaten Tangerang terdekat\n\n⚠️ Format Nomor Pelanggan:\n   - Biasanya 6-10 digit angka atau kombinasi huruf+angka\n   - Cek di lembar tagihan bulan lalu\n   - Contoh: A532159, 1234567890" \
    "cek, tagihan, lihat, air, pdam, cara, bagaimana, info, informasi, bayar, pembayaran, bill, check, pelanggan, nomor, bulan ini, berapa, rekening" \
    100

insert_knowledge "tagihan" \
    "Dimana saja tempat pembayaran tagihan air PDAM?" \
    "Tagihan PDAM Kabupaten Tangerang bisa dibayar di:\n\n💬 Via WhatsApp (Chat ini):\n   Kirim nomor pelanggan Anda, lalu pilih metode pembayaran Virtual Account yang tersedia\n\n💳 Mobile Banking:\n- BCA Mobile, BRI Mobile, Mandiri Online\n- GoPay, OVO, Dana, ShopeePay\n\n🏪 Merchant:\n- Indomaret, Alfamart, Alfamidi\n- Kantor Pos\n- ATM (BCA, BRI, Mandiri, BNI)\n\n🏢 Kantor PDAM:\n- Semua kantor cabang PDAM Kabupaten Tangerang\n- Payment Point PDAM\n\nGunakan nomor pelanggan Anda untuk pembayaran." \
    "bayar, pembayaran, dimana bayar, cara bayar, tempat bayar, payment, virtual account, va" \
    95

insert_knowledge "tagihan" \
    "Berapa tarif air PDAM per meter kubik?" \
    "Tarif air PDAM Kabupaten Tangerang (per m³):\n\n🏠 Rumah Tangga (Domestik):\n- 0-10 m³: Rp 1.650/m³\n- 11-20 m³: Rp 4.100/m³\n- 21-30 m³: Rp 6.200/m³\n- >30 m³: Rp 8.300/m³\n\n🏢 Niaga/Usaha:\n- 0-30 m³: Rp 10.400/m³\n- >30 m³: Rp 11.800/m³\n\n🏭 Industri:\n- Semua pemakaian: Rp 13.500/m³\n\n*Tarif sudah termasuk PPN 11%\n*Biaya admin & beban tetap: Rp 15.000/bulan" \
    "tarif, harga, biaya, tarif air, harga air, biaya per meter, m3" \
    85

insert_knowledge "tagihan" \
    "Kenapa tagihan air saya naik drastis bulan ini?" \
    "Tagihan naik bisa disebabkan:\n\n1️⃣ Pemakaian naik (musim kemarau, tamu, dll)\n2️⃣ Kebocoran pipa dalam rumah\n3️⃣ Keran tidak tertutup rapat\n4️⃣ Closet bocor/mengalir terus\n5️⃣ Meteran rusak/tidak akurat\n\n✅ Solusi:\n- Cek semua keran & pipa di rumah\n- Matikan semua keran, cek meteran masih jalan atau tidak\n- Jika meteran jalan padahal keran mati = ada kebocoran\n- Lapor ke PDAM jika ada kerusakan meteran\n\nHubungi: 1500-123 atau WA 0811-1500-123 untuk pengecekan petugas." \
    "tagihan naik, tagihan tinggi, tagihan mahal, kenapa mahal, tagihan besar" \
    90

###############################################################################
# KATEGORI: PENGADUAN
###############################################################################

insert_knowledge "pengaduan" \
    "Bagaimana cara melapor jika air tidak mengalir?" \
    "Jika air tidak mengalir, segera lapor melalui:\n\n📞 Call Center: (021) 5951234 (24 jam)\n📱 WhatsApp: 0811-5951-234\n💻 Website: www.pdamkabtgr.co.id/pengaduan\n📧 Email: pengaduan@pdamkabtgr.co.id\n\nInfo yang perlu disiapkan:\n- Nomor pelanggan\n- Alamat lengkap\n- Sejak kapan air tidak mengalir\n- Foto meteran (jika perlu)\n\n⏱️ Estimasi penanganan:\n- Gangguan ringan: 2-4 jam\n- Gangguan sedang: 4-8 jam\n- Gangguan berat: 1-2 hari kerja" \
    "air mati, air tidak keluar, tidak ada air, air macet, air kering, lapor" \
    100

insert_knowledge "pengaduan" \
    "Air keruh atau berbau, apa yang harus dilakukan?" \
    "Jika air keruh atau berbau:\n\n✅ Langkah Pertama:\n1. Biarkan keran mengalir 5-10 menit\n2. Jika tetap keruh/berbau, segera lapor\n\n📞 Lapor ke:\n- Call Center: (021) 5951234\n- WhatsApp: 0811-5951-234\n\n⚠️ Sementara waktu:\n- Jangan konsumsi air mentah\n- Masak air hingga mendidih\n- Atau gunakan air galon sementara\n\n🔍 Penyebab umum:\n- Pembilasan pipa\n- Lumpur di reservoir\n- Perbaikan pipa utama\n- Backwash filter\n\nPetugas akan datang untuk pengecekan kualitas air." \
    "air keruh, air kotor, air bau, air berbau, air kuning, air merah, kualitas air" \
    95

insert_knowledge "pengaduan" \
    "Ada kebocoran pipa di jalan depan rumah, kemana melapor?" \
    "Untuk kebocoran pipa PDAM di jalan/area publik:\n\n🚨 Segera lapor ke:\n📞 Hotline Darurat: (021) 5951234 (24 jam)\n📱 WhatsApp: 0811-5951-234\n\nInfo yang dibutuhkan:\n- Lokasi kebocoran (alamat lengkap)\n- Foto lokasi kebocoran\n- Tingkat keparahan (rembes/muncrat/banjir)\n\n⚡ Tim Emergency Response:\n- Akan datang dalam 1-2 jam\n- Untuk kasus darurat < 30 menit\n\n✅ PDAM akan:\n- Isolasi area\n- Perbaiki pipa\n- Pulihkan aliran\n\nTerima kasih telah melaporkan! Ini membantu mencegah pemborosan air." \
    "bocor, kebocoran, pipa bocor, pipa pecah, air meluap, pipa rusak, jalan bocor" \
    100

insert_knowledge "pengaduan" \
    "Meteran air saya rusak atau mati, bagaimana prosedurnya?" \
    "Jika meteran rusak/tidak berputar:\n\n📝 Prosedur:\n1. Lapor ke PDAM via:\n   - Call Center: (021) 5951234\n   - WhatsApp: 0811-5951-234\n   - Datang ke kantor terdekat\n\n2. Petugas akan datang untuk pengecekan (2-3 hari kerja)\n\n3. Jika terbukti rusak:\n   - Penggantian meteran GRATIS (jika kerusakan wajar)\n   - Biaya perbaikan: Rp 250.000 (jika karena kesalahan pelanggan)\n\n4. Selama meteran rusak:\n   - Tagihan dihitung rata-rata 3 bulan terakhir\n   - Atau minimal pemakaian 10m³\n\n⚠️ Jangan membongkar meteran sendiri!\n(Bisa dikenakan denda)" \
    "meteran rusak, meteran mati, meteran error, ganti meteran, meteran tidak jalan" \
    85

###############################################################################
# KATEGORI: ADMINISTRASI
###############################################################################

insert_knowledge "administrasi" \
    "Bagaimana cara balik nama pelanggan PDAM?" \
    "Prosedur balik nama PDAM:\n\n📄 Persyaratan:\n1. Pemilik lama:\n   - KTP asli + fotocopy\n   - Surat pernyataan pelepasan hak\n   - Materai Rp 10.000\n\n2. Pemilik baru:\n   - KTP asli + fotocopy\n   - KK (Kartu Keluarga)\n   - Surat jual beli/hibah/waris\n   - PBB tahun terakhir atas nama baru\n\n💰 Biaya balik nama: Rp 150.000\n\n⏱️ Proses: 3-5 hari kerja\n\n📍 Datang ke kantor PDAM terdekat dengan kedua belah pihak (pemilik lama & baru)\n\n*Tagihan harus lunas sebelum balik nama" \
    "balik nama, ganti nama, pindah nama, tukar nama, nama pelanggan" \
    90

insert_knowledge "administrasi" \
    "Bagaimana cara menutup atau memutus sambungan air PDAM?" \
    "Untuk menutup sambungan air PDAM:\n\n📝 Prosedur:\n1. Datang ke kantor PDAM dengan:\n   - KTP pelanggan\n   - Nomor pelanggan\n   - Bukti tagihan terakhir LUNAS\n\n2. Isi formulir penutupan\n\n3. Biaya:\n   - Penutupan sementara (< 6 bulan): GRATIS\n   - Penutupan permanen: Rp 100.000\n\n4. Proses: 2-3 hari kerja\n\n⚠️ Catatan:\n- Semua tagihan harus lunas\n- Meteran akan dibongkar (penutupan permanen)\n- Untuk buka kembali perlu proses seperti pasang baru\n\n💡 Jika pindah rumah, pertimbangkan balik nama daripada tutup." \
    "tutup, putus, cabut, bongkar, penutupan sambungan, matikan air" \
    75

insert_knowledge "administrasi" \
    "Bagaimana cara mengubah data pelanggan seperti alamat, telepon, atau email?" \
    "Untuk update data pelanggan:\n\n📱 Via Online:\n- Login ke: www.pdamkabtgr.co.id/akun-saya\n- Update data di menu 'Profil'\n- Upload KTP untuk verifikasi\n\n🏢 Via Kantor:\n- Datang ke kantor PDAM terdekat\n- Bawa: KTP asli + fotocopy\n- Isi formulir perubahan data\n\n📞 Via Call Center (untuk no HP/email):\n- Hubungi: (021) 5951234\n- Verifikasi dengan menjawab pertanyaan keamanan\n\n💰 Biaya: GRATIS\n⏱️ Proses: Instant (online) atau 1 hari kerja (kantor)\n\n✅ Data yang bisa diubah:\n- Nomor HP/WA\n- Email\n- Alamat korespondensi (bukan alamat sambungan)" \
    "ubah data, ganti nomor, ganti email, update data, ganti alamat" \
    70

###############################################################################
# KATEGORI: UMUM
###############################################################################

insert_knowledge "umum" \
    "Apa saja tips hemat pemakaian air PDAM?" \
    "💧 Tips hemat air & tagihan:\n\n1️⃣ Kamar Mandi:\n- Mandi pakai gayung, bukan shower (hemat 50%)\n- Matikan keran saat menyabun\n- Perbaiki keran yang menetes\n\n2️⃣ Cuci Pakaian:\n- Cuci dengan beban penuh\n- Pakai ulang air bilasan untuk ngepel\n\n3️⃣ Dapur:\n- Cuci piring sekaligus, jangan satu-satu\n- Rendam dulu sebelum cuci\n\n4️⃣ Umum:\n- Cek kebocoran rutin\n- Pasang aerator di keran (hemat 30%)\n- Tampung air hujan untuk siram tanaman\n\n📊 Rata-rata kebutuhan: 10-15 m³/bulan untuk 4 orang" \
    "hemat air, tips hemat, boros air, irit air, cara hemat" \
    60

insert_knowledge "umum" \
    "Apakah air PDAM aman untuk diminum?" \
    "✅ Air PDAM Kabupaten Tangerang sudah memenuhi standar:\n- Permenkes No. 492/2010\n- WHO (World Health Organization)\n- SNI (Standar Nasional Indonesia)\n\n🔬 Pengujian kualitas:\n- Lab PDAM: Setiap hari\n- Lab Independen: Per bulan\n- Parameter: Bakteriologi, fisika, kimia\n\n💧 Rekomendasi:\n- Air PDAM layak dikonsumsi setelah DIMASAK mendidih\n- Untuk konsumsi langsung, gunakan filter/dispenser dengan UV\n\n📍 Cek kualitas air di daerah Anda:\nwww.pdamkabtgr.co.id/kualitas-air\n\n☎️ Lapor jika ada kejanggalan:\n(021) 5951234" \
    "air minum, aman diminum, bisa diminum, kualitas air, air bersih, sehat" \
    80

insert_knowledge "umum" \
    "Apakah ada program subsidi atau keringanan tagihan PDAM?" \
    "🎁 Program subsidi PDAM Kabupaten Tangerang:\n\n1️⃣ Subsidi Rumah Tangga Kecil:\n- Pemakaian 0-10 m³/bulan\n- Tarif lebih murah: Rp 1.650/m³\n- Otomatis tanpa pendaftaran\n\n2️⃣ Keringanan Tunggakan:\n- Diskon denda 50% (periode tertentu)\n- Cicilan tunggakan 3-6 bulan\n- Hubungi: (021) 5951234 untuk apply\n\n3️⃣ Program Sambungan Gratis:\n- Untuk keluarga prasejahtera (PKH/KIS/DTKS)\n- Gratis biaya pasang Rp 2.500.000\n- Hubungi Dinas Sosial setempat\n\n4️⃣ Air Tangki Gratis (saat gangguan):\n- Jika air mati > 24 jam\n- Request via (021) 5951234\n\nInfo lengkap: www.pdamkabtgr.co.id/subsidi" \
    "subsidi, diskon, keringanan, bantuan, gratis, murah, promo" \
    75

###############################################################################
# Summary
###############################################################################

echo ""
echo "========================================="
echo "Seeding completed!"
echo "========================================="
echo -e "${GREEN}Success:${NC} $SUCCESS_COUNT"
echo -e "${RED}Failed:${NC} $FAILED_COUNT"
echo "Total: $TOTAL_COUNT"
echo ""

# Trigger cache refresh
echo "Refreshing cache for client: $CLIENT_ID..."
REFRESH_ENDPOINT="${BASE_URL}/api/v1/instant-link/ai/knowledge/refresh"
curl -s -X POST "$REFRESH_ENDPOINT" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $AUTH_TOKEN" \
    -d "{\"client_id\":\"$CLIENT_ID\"}" > /dev/null

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓${NC} Cache refreshed successfully"
else
    echo -e "${YELLOW}⚠${NC} Cache refresh may have failed, but data is already in DB"
fi

echo ""
echo "Done! Knowledge base is ready to use."
echo "========================================="

exit 0
