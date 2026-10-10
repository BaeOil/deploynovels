// src/data/mockWriterData.js
// ══════════════════════════════════════════════════════════
//  Mock data ฝั่ง Writer
//  TODO: แทนที่ด้วย API calls เมื่อ Backend พร้อม
//  GET /api/v1/writer/dashboard
//  GET /api/v1/writer/novels
// ══════════════════════════════════════════════════════════

export const mockWriterProfile = {
  id: "writer-001",
  displayName: "makeawish",
  role: "Writer",
  avatarEmoji: "🦉",
  joinedDate: "มกราคม 2568",
};

// ─────────────────────────────────────────────
// Dashboard stats
// ─────────────────────────────────────────────
export const mockDashboardStats = {
  totalNovels:    4,
  totalLikes:     1989,
  totalViews:     8561,
  totalBookmarks: 50,
};

// ─────────────────────────────────────────────
// Writer's novels list
// ─────────────────────────────────────────────
export const mockWriterNovels = [
  {
    id: "novel-001",
    title: "ห้องสมุดปริศนา",
    coverEmoji: "🌸",
    coverBg: "linear-gradient(150deg,#FFF0F5,#FFD6E7)",
    categories: ["แฟนตาซี", "ลึกลับ"],
    synopsis: "นักสืบสาวค้นพบห้องสมุดลับที่ซ่อนความลับนับพันปี",
    status: "published",   // published | draft
    chapterCount: 12,
    sceneCount: 48,
    stats: { views: 5234, likes: 1204, bookmarks: 32 },
    updatedAt: "18 เม.ย. 2568",
  },
  {
    id: "novel-002",
    title: "ราชันย์แห่งเงา",
    coverEmoji: "⚔️",
    coverBg: "linear-gradient(150deg,#F3F0FF,#E0D6FF)",
    categories: ["แฟนตาซี", "ผจญภัย"],
    synopsis: "อัศวินผู้ถูกสาปต้องต่อสู้ในโลกที่ความมืดครองเมือง",
    status: "draft",
    chapterCount: 8,
    sceneCount: 31,
    stats: { views: 3327, likes: 785, bookmarks: 18 },
    updatedAt: "10 เม.ย. 2568",
  },
  {
    id: "novel-003",
    title: "จันทร์เจ็บ",
    coverEmoji: "🌙",
    coverBg: "linear-gradient(150deg,#F0F4FF,#D6E0FF)",
    categories: ["โรแมนซ์", "ดราม่า"],
    synopsis: "ความรักที่เจ็บปวดระหว่างคืนเดือนหงาย",
    status: "published",
    chapterCount: 6,
    sceneCount: 24,
    stats: { views: 2108, likes: 930, bookmarks: 15 },
    updatedAt: "5 เม.ย. 2568",
  },
  {
    id: "novel-004",
    title: "น้ำขอบฟ้าใส",
    coverEmoji: "🌊",
    coverBg: "linear-gradient(150deg,#F0FBFF,#D6F0FF)",
    categories: ["แฟนตาซี"],
    synopsis: "เด็กหญิงผู้ค้นพบแผนที่ในโลกที่น้ำท่วมทุกสิ่ง",
    status: "draft",
    chapterCount: 4,
    sceneCount: 14,
    stats: { views: 412, likes: 98, bookmarks: 5 },
    updatedAt: "2 เม.ย. 2568",
  },
];

// ─────────────────────────────────────────────
// Category options สำหรับ CreateNovel form
// ─────────────────────────────────────────────
export const NOVEL_CATEGORIES = [
  "แฟนตาซี",
  "โรแมนซ์",
  "ผจญภัย",
  "ลึกลับ",
  "สยองขวัญ",
  "ดราม่า",
  "ตลก",
  "ชีวิต",
  "ไซไฟ",
  "ประวัติศาสตร์",
];