import React from "react";
import "./AdminModeBanner.css";

/**
 * AdminModeBanner
 *
 * แบนเนอร์แสดงสถานะโหมดแอดมินที่ด้านบนของหน้า
 * ใช้ร่วมกันในทุกหน้าที่แอดมินสามารถเข้าถึงได้
 *
 * Props:
 *   page — string ชื่อหน้าปัจจุบัน (optional) เช่น "หน้าอ่านนิยาย", "โปรไฟล์นักเขียน"
 */
const AdminModeBanner = ({ page = "" }) => {
  return (
    <div className="admin-mode-banner" role="status" aria-live="polite">
      <div className="admin-mode-banner__inner">
        <span className="admin-mode-banner__icon">🛡️</span>
        <span className="admin-mode-banner__text">
          <strong>โหมดผู้ดูแลระบบ</strong>
          {page && (
            <span className="admin-mode-banner__page"> — {page}</span>
          )}
          <span className="admin-mode-banner__desc">
            คุณกำลังดูหน้านี้ในฐานะแอดมิน ฟีเจอร์บางส่วนถูกปิดซ่อนสำหรับบัญชีประเภทนี้
          </span>
        </span>
        <span className="admin-mode-banner__badge">ADMIN</span>
      </div>
    </div>
  );
};

export default AdminModeBanner;
