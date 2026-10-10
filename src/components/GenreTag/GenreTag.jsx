// src/components/GenreTag/GenreTag.jsx
import React from "react";
import "./GenreTag.css";

// จับคู่คำในชื่อหมวดหมู่กับกลุ่มสี — ไม่ครบทุกคำก็ไม่เป็นไร มีสี "default"
// (สีเดิมแบบ primary) รองรับหมวดที่เดาไม่ได้อยู่แล้ว
const GENRE_COLOR_RULES = [
  { key: "romance", match: ["โรแมนติก", "รัก", "romance"] },
  { key: "fantasy", match: ["แฟนตาซี", "fantasy"] },
  { key: "adventure", match: ["ผจญภัย", "adventure"] },
  { key: "comedy", match: ["คอมเมดี้", "ตลก", "comedy"] },
  { key: "drama", match: ["ดราม่า", "drama"] },
  { key: "horror", match: ["สยองขวัญ", "ผี", "horror"] },
  { key: "mystery", match: ["ลึกลับ", "สืบสวน", "mystery"] },
  { key: "action", match: ["แอคชั่น", "ต่อสู้", "action"] },
  { key: "scifi", match: ["ไซไฟ", "วิทยาศาสตร์", "sci-fi", "scifi"] },
  { key: "slice", match: ["ชีวิตประจำวัน", "slice"] },
];

const detectGenreColorKey = (label) => {
  if (!label) return "default";
  const lower = String(label).toLowerCase();
  const rule = GENRE_COLOR_RULES.find((r) =>
    r.match.some((keyword) => lower.includes(keyword.toLowerCase()))
  );
  return rule ? rule.key : "default";
};

/**
 * GenreTag — แสดงหมวดหมู่นิยาย
 * @param {string} label - ชื่อหมวดหมู่
 * @param {string} variant - "primary" | "outline" (default: "primary") ใช้เมื่อ colorByCategory เป็น false
 * @param {boolean} colorByCategory - true = เลือกสีอัตโนมัติตามคำในชื่อหมวดหมู่ (default: false เพื่อไม่กระทบที่เรียกใช้เดิม)
 */
const GenreTag = ({ label, variant = "primary", colorByCategory = false }) => {
  const modifier = colorByCategory ? `category-${detectGenreColorKey(label)}` : variant;

  return (
    <span className={`genre-tag genre-tag--${modifier}`} aria-label={`หมวดหมู่: ${label}`}>
      {label}
    </span>
  );
};

export default GenreTag;