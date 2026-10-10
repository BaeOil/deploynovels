import React from "react";
import "./CategoryCard.css";

const CategoryCard = ({ category, active, onClick }) => {
  return (
    <button
      type="button"
      className={`category-card ${active ? "category-card--active" : ""}`}
      onClick={() => onClick(category.name)}
      style={{
        background: active ? category.background : "#fff0f6",
        border: active ? "none" : "1px solid #fce7f3"
      }}
    >
      <div className="category-card__body">
        <div 
          className="category-card__badge"
          style={{ background: active ? "rgba(255, 255, 255, 0.25)" : "rgba(244, 63, 94, 0.08)" }}
        >
          {category.icon}
        </div>
        <div className="category-card__info">
          <h3 className={`category-card__title ${active ? "text-white" : "text-slate-800"}`}>
            {category.name}
          </h3>
          <p className={`category-card__count ${active ? "text-rose-100" : "text-slate-400"}`}>
            {category.count} เรื่อง
          </p>
        </div>
      </div>
    </button>
  );
};

export default CategoryCard;