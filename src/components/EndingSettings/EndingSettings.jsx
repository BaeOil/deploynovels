import React from "react";
import "./EndingSettings.css";

const TYPES = [
  {
    value: "good",
    icon: "🌸",
    label: "Good Ending",
    hint: "ฉากจบที่ตัวละครมีความสุขหรือประสบความสำเร็จ",
    className: "good",
  },
  {
    value: "bad",
    icon: "💀",
    label: "Bad Ending",
    hint: "ฉากจบที่หม่นหมอง ตัวละครพบกับความสูญเสียหรือความล้มเหลว",
    className: "bad",
  },
  {
    value: "true",
    icon: "👑",
    label: "True Ending",
    hint: "ฉากจบที่แท้จริง เปิดเผยปมและบทสรุปทั้งหมดของเรื่องราว",
    className: "true",
  },
  {
    value: "secret",
    icon: "🌙",
    label: "Secret Ending",
    hint: "ฉากจบลับที่ซ่อนอยู่หลังตัวเลือกพิเศษ",
    className: "secret",
  },
];

export default function EndingSettings({
  sceneTitle = "แสงสุดท้ายแห่งอาณาจักร",
  isEnding = true,
  endingTitle = "",
  endingType = "true",
  endingDescription = "",
  endingDescriptionEnabled = false,
  onToggleEnding,
  onToggleEndingDescriptionEnabled,
  onChangeEndingTitle,
  onChangeEndingType,
  onChangeEndingDescription,
  onSave,
  onClose,
}) {
  const [descriptionEnabled, setDescriptionEnabled] = React.useState(
    Boolean(endingDescriptionEnabled || endingDescription)
  );

  React.useEffect(() => {
    setDescriptionEnabled(Boolean(endingDescriptionEnabled || endingDescription));
  }, [endingDescriptionEnabled, endingDescription]);

  const handleToggleDescription = () => {
    const next = !descriptionEnabled;
    setDescriptionEnabled(next);
    onToggleEndingDescriptionEnabled?.(next);
  };

  const current = TYPES.find((t) => t.value === endingType) || TYPES[0];
  const previewTitle = endingTitle.trim() || sceneTitle;

  return (
    <div className="ending-page">
      <div className="ending-card">
        {/* 1. Header & Main Toggle */}
        <div className="ending-header">
          <div className="ending-header-text">
            <h3>🏁 ฉากจบ</h3>
            <p>บันทึกฉากนี้ลงในคลังฉากจบของนักอ่าน</p>
          </div>

          <div className="header-toggle">
            <span className="header-toggle-label">ใช้เป็นฉากจบ</span>
            <label className="switch">
              <input
                type="checkbox"
                checked={isEnding}
                onChange={() => onToggleEnding?.(!isEnding)}
              />
              <span className="slider"></span>
            </label>
          </div>
        </div>

        {isEnding && (
          <div className="ending-settings-two-columns" style={{
            display: "grid",
            gridTemplateColumns: "1fr 1.35fr",
            gap: "24px",
            padding: "20px 24px",
            borderTop: "1px solid #f3f4f6"
          }}>
            
            {/* ฝั่งซ้าย (Left Column) - ตัวอย่างคลังฉากจบ */}
            <div className="ending-settings-left-col" style={{ display: "flex", flexDirection: "column", gap: "12px" }}>
              <label style={{ fontWeight: "700", color: "#374151", fontSize: "0.95rem", textAlign: "left" }}>ตัวอย่างคลังฉากจบ</label>
              
              <div className={`preview-card ${current.className}`} style={{ height: "100%", justifyContent: "center" }}>
                <div className="preview-icon">{current.icon}</div>
                <div className="badge">{current.label}</div>
                <h4>{previewTitle}</h4>
                <p className="preview-description">
                  {descriptionEnabled
                    ? endingDescription.trim() || "ยังไม่มีคำอธิบายตอนจบ"
                    : ""}
                </p>
                <small style={{ marginTop: "auto" }}>จะแสดงใน คลังฉากจบ หลังจากนักอ่านค้นพบฉากจบนี้</small>
              </div>
            </div>

            {/* ฝั่งขวา (Right Column) - ตั้งค่าและรายละเอียด */}
            <div className="ending-settings-right-col" style={{ display: "flex", flexDirection: "column", gap: "20px", textAlign: "left" }}>
              
              {/* เลือกประเภท ENDING */}
              <div className="ending-section-box" style={{ textAlign: "left" }}>
                <label style={{ fontWeight: "700", color: "#374151", fontSize: "0.95rem", display: "block", marginBottom: "12px", textAlign: "left" }}>ประเภทฉากจบ</label>
                <div className="type-grid" style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: "12px" }}>
                  {TYPES.map((item) => {
                    const isActive = endingType === item.value;
                    return (
                      <button
                        key={item.value}
                        type="button"
                        className={`type-card ${isActive ? "active" : ""}`}
                        onClick={() => onChangeEndingType?.(item.value)}
                        style={{
                          display: "flex",
                          alignItems: "center",
                          gap: "10px",
                          padding: "14px 16px",
                          borderRadius: "12px",
                          border: isActive ? "2px solid #db2777" : "1.5px solid #e5e7eb",
                          backgroundColor: isActive ? "rgba(219, 39, 119, 0.02)" : "#ffffff",
                          cursor: "pointer",
                          textAlign: "left",
                          position: "relative",
                          transition: "all 0.18s ease"
                        }}
                      >
                        <div className="icon" style={{ fontSize: "1.6rem", marginBottom: 0 }}>{item.icon}</div>
                        <div style={{ fontWeight: "700", color: isActive ? "#db2777" : "#374151", fontSize: "0.88rem" }}>{item.label}</div>
                        
                        {/* ✨ เครื่องหมาย Checkmark เมื่อเลือก */}
                        {isActive && (
                          <div style={{
                            position: "absolute",
                            top: "6px",
                            right: "6px",
                            width: "16px",
                            height: "16px",
                            borderRadius: "50%",
                            backgroundColor: "#db2777",
                            color: "#ffffff",
                            display: "flex",
                            alignItems: "center",
                            justifyContent: "center",
                            fontSize: "9px",
                            fontWeight: "700"
                          }}>
                            ✓
                          </div>
                        )}
                      </button>
                    );
                  })}
                </div>
                <div className="type-hint" style={{ marginTop: "10px", padding: "10px", background: "#f9fafb", borderRadius: "10px", color: "#6b7280", fontSize: "0.82rem", textAlign: "left" }}>
                  {current.hint}
                </div>
              </div>

              {/* ชื่อฉากจบ */}
              <div className="ending-section-box" style={{ display: "flex", flexDirection: "column", gap: "8px", textAlign: "left" }}>
                <label style={{ fontWeight: "700", color: "#374151", fontSize: "0.95rem", textAlign: "left" }}>
                  ชื่อฉากจบ <span className="optional" style={{ marginLeft: "6px", color: "#9ca3af", fontSize: "0.8rem", fontWeight: "400" }}>(เว้นว่างเพื่อใช้ชื่อฉาก)</span>
                </label>
                <input
                  className="input"
                  value={endingTitle}
                  onChange={(e) => onChangeEndingTitle?.(e.target.value)}
                  placeholder={sceneTitle}
                  style={{ width: "100%", padding: "12px", borderRadius: "12px", border: "1px solid #e5e7eb", fontSize: "0.88rem" }}
                />
              </div>

              {/* รายละเอียดฉากจบ & Toggle */}
              <div className="ending-section-box" style={{ textAlign: "left" }}>
                <div className="toggle-row" style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: "10px", width: "100%" }}>
                  <div className="toggle-row-text" style={{ textAlign: "left" }}>
                    <label style={{ fontWeight: "700", color: "#374151", fontSize: "0.95rem", marginBottom: "4px", display: "block", textAlign: "left" }}>รายละเอียดฉากจบ</label>
                    <p style={{ margin: 0, color: "#6b7280", fontSize: "0.82rem", textAlign: "left" }}>แสดงข้อความเพิ่มเติมหลังปลดล็อก</p>
                  </div>
                  <label className="switch" style={{ marginLeft: "auto" }}>
                    <input
                      type="checkbox"
                      checked={descriptionEnabled}
                      onChange={handleToggleDescription}
                    />
                    <span className="slider" />
                  </label>
                </div>
                
                {descriptionEnabled && (
                  <textarea
                    className="input"
                    style={{ minHeight: "100px", resize: "vertical", width: "100%", padding: "12px", borderRadius: "12px", border: "1px solid #e5e7eb", fontSize: "0.88rem", marginTop: "8px" }}
                    value={endingDescription}
                    onChange={(e) => onChangeEndingDescription?.(e.target.value)}
                    placeholder="เขียนคำอธิบายเพิ่มเติมที่นี่..."
                  />
                )}
              </div>

            </div>

          </div>
        )}
      </div>
    </div>
  );
}
