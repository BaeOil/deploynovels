/**
 * ปรับ URL รูปภาพ/ไฟล์ media จาก internal host (ใน docker) ให้กลายเป็น public host ที่ browser เข้าถึงได้
 *
 * เดิมโค้ดในแต่ละไฟล์ hardcode `.replace("http://minio:9000", "http://localhost:9000")` ไว้ตรงๆ
 * ซึ่งพังทันทีถ้า deploy เพราะ browser จะเรียก localhost:9000 กลับมาที่เครื่องตัวเอง
 *
 * ตอนนี้ใช้ env แทน (ตั้งใน Vercel → Settings → Environment Variables):
 *   - VITE_MEDIA_INTERNAL_HOST = host ที่ MinIO ส่งมาใน URL (เช่น http://minio:9000)
 *   - VITE_MEDIA_PUBLIC_HOST   = host ที่ browser เรียกได้จริง (เช่น https://files.deploynovels.com หรือ https://cdn.railway.app)
 *
 * ถ้าไม่ตั้งค่าไว้ ค่า default ยังเหมือนของเดิมทุกอย่าง (localhost) เพื่อไม่ให้ local dev พัง
 */

const MEDIA_INTERNAL_HOST =
  import.meta.env.VITE_MEDIA_INTERNAL_HOST || "http://minio:9000";

const MEDIA_PUBLIC_HOST =
  import.meta.env.VITE_MEDIA_PUBLIC_HOST || "http://localhost:9000";

// host ภายในที่เคยถูก hardcode ไว้ในโค้ดเดิม / อยู่ใน seed data ของ DB
// ต้องแปลงทุกตัวนี้เป็น public host ตอน deploy ไม่งั้นรูปจะพ้างตอน production
const LEGACY_INTERNAL_HOSTS = [
  "http://minio:9000",
  "http://localhost:9000",
  "http://127.0.0.1:9000",
  "https://minio:9000",
  "https://localhost:9000",
  "https://127.0.0.1:9000",
];

// สร้าง list เต็มของ host ที่ต้องแปลง (รวมค่าที่มาจาก env เข้าไปก่อน ไม่ให้ซ้ำ)
function buildInternalHosts() {
  const hosts = new Set();
  if (MEDIA_INTERNAL_HOST) hosts.add(MEDIA_INTERNAL_HOST);
  if (MEDIA_PUBLIC_HOST) hosts.add(MEDIA_PUBLIC_HOST);
  LEGACY_INTERNAL_HOSTS.forEach((h) => hosts.add(h));
  return Array.from(hosts).filter(Boolean);
}

const INTERNAL_HOSTS = buildInternalHosts();

/**
 * แปลง URL media จาก internal → public host
 * @param {string|null|undefined} url — URL ที่ backend ส่งมา (อาจมี scheme http://minio:9000/... หรือ http://localhost:9000/...)
 * @returns {string|null} — URL ที่ browser เรียกได้จริง หรือ null ถ้าไม่มี input
 */
export function resolveMediaUrl(url) {
  if (!url || typeof url !== "string") return null;

  let result = url;

  // แทน host ฝั่ง legacy/internal ทั้งหมดเป็น public host
  // sort ยาว→สั้นกันปัญหา localhost:9000 ถูกแทนก่อน localhost:90001 แล้ว match พลาด
  const sortedHosts = [...INTERNAL_HOSTS].sort((a, b) => b.length - a.length);
  for (const host of sortedHosts) {
    if (!host) continue;
    if (result.includes(host)) {
      result = result.split(host).join(MEDIA_PUBLIC_HOST);
    }
  }

  // ถ้า URL ไม่มี scheme (เช่น /novels-images/xxx.jpg หรือ novels-images/xxx.jpg)
  // ให้เติม public host เข้าไปด้านหน้า
  if (result && !/^https?:\/\//i.test(result) && !result.startsWith("data:") && !result.startsWith("blob:")) {
    const base = MEDIA_PUBLIC_HOST.replace(/\/+$/, "");
    const path = result.startsWith("/") ? result : `/${result}`;
    result = `${base}${path}`;
  }

  return result;
}
