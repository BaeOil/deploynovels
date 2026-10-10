// src/components/StatBadge/StatBadge.jsx
import React from "react";
import "./StatBadge.css";

/**
 * StatBadge — แสดงตัวเลขสถิติ (ยอดอ่าน, เส้นทาง, จุดเลือก, ตอนจบ)
 * @param {string|number} value - ค่าตัวเลข
 * @param {string} label - ป้ายกำกับ
 */
const StatBadge = ({ value, label }) => {
  // Format large numbers (e.g. 5900 → 5.9K)
  const formatValue = (val) => {
    if (typeof val === "number") {
      if (val >= 1000) return `${(val / 1000).toFixed(1)}K`;
    }
    return val;
  };

  return (
    <div className="stat-badge" role="figure" aria-label={`${label}: ${value}`}>
      <span className="stat-badge__value">{formatValue(value)}</span>
      <span className="stat-badge__label">{label}</span>
    </div>
  );
};

export default StatBadge;
