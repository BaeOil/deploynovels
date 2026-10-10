// src/components/Toggle/Toggle.jsx
import React from "react";
import "./Toggle.css";

/**
 * Toggle — สวิตช์ on/off
 * @param {boolean}  checked   - สถานะ
 * @param {function} onChange  - callback(newValue)
 * @param {string}   label     - ป้ายกำกับ (optional)
 * @param {string}   id        - id สำหรับ accessibility
 */
const Toggle = ({ checked = false, onChange, label, id, disabled = false }) => {
  const handleClick = () => {
    if (disabled) return;
    onChange?.(!checked);
  };

  return (
    <label className="toggle-wrap" htmlFor={id}>
      <button
        id={id}
        type="button"
        role="switch"
        aria-checked={checked}
        aria-disabled={disabled}
        aria-label={label}
        disabled={disabled}
        className={`toggle ${checked ? "toggle--on" : "toggle--off"} ${disabled ? "toggle--disabled" : ""}`}
        onClick={handleClick}
      >
        <span className="toggle__thumb" />
      </button>
      {label ? (
        <span className={`toggle__label ${checked ? "toggle__label--on" : ""}`}>
          {label}
        </span>
      ) : null}
    </label>
  );
};

export default Toggle;