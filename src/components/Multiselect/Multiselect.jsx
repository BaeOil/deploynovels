// src/components/MultiSelect/MultiSelect.jsx
import React, { useState, useRef, useEffect } from "react";
import "./MultiSelect.css";

/**
 * MultiSelect — เลือกหมวดหมู่หลายอย่างพร้อมกัน
 * แสดงเป็น tags/pills + dropdown
 *
 * @param {string[]}  options      - ตัวเลือกทั้งหมด
 * @param {string[]}  value        - ค่าที่เลือกอยู่
 * @param {function}  onChange     - callback(newSelected[])
 * @param {string}    placeholder  - placeholder
 * @param {number}    max          - จำนวนสูงสุดที่เลือกได้
 */
const MultiSelect = ({
  options = [],
  value = [],
  onChange,
  placeholder = "เลือกหมวดหมู่...",
  max = 5,
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const wrapRef = useRef(null);

  // ปิด dropdown เมื่อคลิกนอก
  useEffect(() => {
    const handleClickOutside = (e) => {
      if (wrapRef.current && !wrapRef.current.contains(e.target)) {
        setIsOpen(false);
      }
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  const toggle = (option) => {
    if (value.includes(option)) {
      // deselect
      onChange(value.filter((v) => v !== option));
    } else {
      // select (ถ้ายังไม่ถึง max)
      if (value.length < max) {
        onChange([...value, option]);
      }
    }
  };

  const remove = (e, option) => {
    e.stopPropagation();
    onChange(value.filter((v) => v !== option));
  };

  return (
    <div className="msel" ref={wrapRef}>
      {/* ── Input box ── */}
      <div
        className={`msel__box ${isOpen ? "msel__box--open" : ""}`}
        onClick={() => setIsOpen(!isOpen)}
        role="combobox"
        aria-expanded={isOpen}
        aria-haspopup="listbox"
        tabIndex={0}
        onKeyDown={(e) => e.key === "Enter" && setIsOpen(!isOpen)}
        aria-label="เลือกหมวดหมู่"
      >
        {/* Tags ที่เลือกแล้ว */}
        <div className="msel__tags">
          {value.length === 0 && (
            <span className="msel__placeholder">{placeholder}</span>
          )}
          {value.map((v) => (
            <span key={v} className="msel__tag">
              {v}
              <button
                type="button"
                className="msel__tag-remove"
                onClick={(e) => remove(e, v)}
                aria-label={`ลบ ${v}`}
              >
                ×
              </button>
            </span>
          ))}
        </div>

        {/* Chevron icon */}
        <svg
          className={`msel__chevron ${isOpen ? "msel__chevron--up" : ""}`}
          width="16" height="16" viewBox="0 0 16 16" fill="none"
          aria-hidden="true"
        >
          <path d="M4 6l4 4 4-4" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/>
        </svg>
      </div>

      {/* ── Dropdown ── */}
      {isOpen && (
        <div className="msel__dropdown" role="listbox" aria-multiselectable="true">
          {options.map((option) => {
            const selected = value.includes(option);
            const disabled = !selected && value.length >= max;
            return (
              <div
                key={option}
                className={`msel__option ${selected ? "msel__option--selected" : ""} ${disabled ? "msel__option--disabled" : ""}`}
                role="option"
                aria-selected={selected}
                onClick={() => !disabled && toggle(option)}
              >
                <span className="msel__option-check" aria-hidden="true">
                  {selected ? "✓" : ""}
                </span>
                {option}
              </div>
            );
          })}
        </div>
      )}

      {/* Count hint */}
      {value.length > 0 && (
        <div className="msel__hint">เลือกแล้ว {value.length}/{max}</div>
      )}
    </div>
  );
};

export default MultiSelect;