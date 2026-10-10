// src/components/NovelCoverCard/NovelCoverCard.jsx
import React from "react";
import "./NovelCoverCard.css";
import StatBadge from "../StatBadge/StatBadge";

/**
 * NovelCoverCard — แสดงภาพปกนิยายและสถิติ 4 ตัว
 * @param {object} novel - ข้อมูลนิยาย
 */
const NovelCoverCard = ({ novel }) => {
  const { title, coverImage, coverEmoji, stats, chapter_count, chapterCount, userProgress, isCompleted } = novel;
  
  const totalComments = stats?.comments ?? 0;

  return (
    <div className="novel-cover-card">
      {/* Cover image */}
      <div className="novel-cover-card__cover" aria-label={`ภาพปก: ${title}`}>
        {isCompleted && (
          <span className="novel-cover-card__ribbon" aria-label="นิยายจบแล้ว">
            🏁 จบแล้ว
          </span>
        )}
        {coverImage ? (
          <img
            src={coverImage}
            alt={`ภาพปกนิยาย ${title}`}
            className="novel-cover-card__img"
          />
        ) : (
          /* Fallback when no image from DB yet */
          <div className="novel-cover-card__placeholder" aria-hidden="true">
            <span className="novel-cover-card__emoji">{coverEmoji}</span>
          </div>
        )}
      </div>

      {/* Stats grid 2x2 */}
      <div className="novel-cover-card__stats" role="list" aria-label="สถิตินิยาย">
        <div role="listitem">
          <StatBadge value={stats.views} label="ยอดอ่าน" />
        </div>
        <div role="listitem">
          <StatBadge value={stats.likes} label="ถูกใจ" />
        </div>
        <div role="listitem">
          <StatBadge value={stats.bookshelfCount} label="ชั้นหนังสือ" />
        </div>
        <div role="listitem">
          <StatBadge value={totalComments} label="คอมเมนต์" />
        </div>
      </div>
    </div>
  );
};

export default NovelCoverCard;