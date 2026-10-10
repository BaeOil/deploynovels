import React, { useState } from "react";
import { useNavigate } from "react-router-dom";
import "./NovelCard.css";
import { getNovelStatusInfo } from "../../utils/novelStatus";

/**
 * NovelCard — การ์ดนิยายมาตรฐาน
 *
 * ขนาด cover จะสม่ำเสมอทุกหน้า เพราะใช้ aspect-ratio: 2/3
 * ความสูงของ cover = column width × (3/2) โดยอัตโนมัติ
 *
 * ✅ วิธีใช้งานใน grid — ต้องเหมือนกันทุกหน้าเพื่อให้การ์ดดูสม่ำเสมอ:
 *
 *   <div style={{
 *     display: "grid",
 *     gridTemplateColumns: "repeat(auto-fill, minmax(180px, 1fr))",
 *     gap: "20px",
 *   }}>
 *     {novels.map(n => <NovelCard key={n.id} novel={n} onClick={...} />)}
 *   </div>
 *
 * Props:
 *   novel  — object (required)
 *   onClick — function (optional)
 */
const NovelCard = ({ novel, onClick, showLike = true }) => {
  const navigate = useNavigate();
  const [liked, setLiked] = useState(novel?.isLiked || false);
  const authorId = novel?.author?.id || novel?.author_id || novel?.author?.writer_id || novel?.author?.user_id;

  const handleAuthorClick = (e) => {
    if (authorId) {
      e.stopPropagation();
      navigate(`/writer/profile/${authorId}`);
    }
  };

  const authorDisplayName =
    novel?.author?.penName ||
    novel?.author?.pen_name ||
    novel?.author?.displayName ||
    novel?.author?.name ||
    novel?.pen_name ||
    novel?.penName ||
    novel?.author_name ||
    novel?.authorName ||
    "ไม่ระบุ";

  // รองรับ field name หลายรูปแบบจาก API
  const views = novel?.views ?? novel?.view_count ?? novel?.stats?.views ?? 0;
  const likes = novel?.like_count ?? novel?.total_likes ?? novel?.stats?.likes ?? 0;
  const bookmarks = novel?.bookshelf_count ?? novel?.bookmark_count ?? novel?.total_bookmarks ?? novel?.stats?.bookmarks ?? 0;

  const fmt = (v = 0) =>
    v >= 1_000_000 ? `${(v / 1_000_000).toFixed(1)}M`
      : v >= 1_000 ? `${(v / 1_000).toFixed(1)}K`
        : String(v);

  const statusInfo = getNovelStatusInfo(novel);
  const shouldShowBadge = statusInfo.isCompleted;

  const handleLike = (e) => {
    e.stopPropagation();
    setLiked(p => !p);
    // TODO: POST /api/novels/:id/like
  };

  const handleKey = (e) => {
    if (e.key === "Enter" || e.key === " ") onClick?.();
  };

  return (
    <article
      className="novel-card"
      onClick={onClick}
      onKeyDown={handleKey}
      role="button"
      tabIndex={0}
      aria-label={`อ่านนิยาย ${novel.title}`}
    >

      {/* ── Cover ── */}
      <div
        className="novel-card__cover"
        style={{ background: novel.coverBg || "var(--pink-50, #fdf2f8)" }}
      >
        {novel.coverImage
          ? (
            <img
              src={novel.coverImage}
              alt={novel.title}
              className="novel-card__cover-img"
              loading="lazy"
            />
          ) : (
            <span className="novel-card__cover-emoji" aria-hidden="true">
              {novel.coverEmoji || "📘"}
            </span>
          )
        }

        {shouldShowBadge && (
          <span className={`novel-card__status novel-card__status--${statusInfo.variant || "completed"}`}>
            {statusInfo.badgeLabel}
          </span>
        )}
      </div>

      {/* ── Body ── */}
      <div className="novel-card__body">

        {/* Categories */}
        <div className="novel-card__categories">
          {(novel.categories || []).slice(0, 2).map((cat, i) => (
            <span key={i} className="novel-card__tag">{typeof cat === "object" ? cat.name : cat}</span>
          ))}
        </div>

        {/* Title (clamp 2 lines) */}
        <h3 className="novel-card__title">{novel.title}</h3>

        {/* Synopsis (clamp 2 lines) */}
        {novel.synopsis && (
          <p className="novel-card__synopsis">
            {novel.synopsis || "\u00A0"}
          </p>
        )}

        {/* Stats */}
        <div className="novel-card__stats-bar">
          <div className="novel-card__stat">
            <div className="novel-card__stat-value">{fmt(views)}</div>
            <div className="novel-card__stat-label">ยอดชม</div>
          </div>
          <div className="novel-card__stat">
            <div className="novel-card__stat-value">{fmt(likes)}</div>
            <div className="novel-card__stat-label">ถูกใจ</div>
          </div>
          <div className="novel-card__stat">
            <div className="novel-card__stat-value">{fmt(bookmarks)}</div>
            <div className="novel-card__stat-label">ชั้นหนังสือ</div>
          </div>
        </div>

        {/* Footer */}
        <div className="novel-card__footer">
          <div className="novel-card__author">
            <span className="novel-card__author-avatar" aria-hidden="true">
              {novel.author?.avatarEmoji || "✍️"}
            </span>
            <span className="novel-card__author-name">
              {authorDisplayName}
            </span>
          </div>

          {showLike && (
            <button
              className={`novel-card__like-btn ${liked ? "novel-card__like-btn--on" : ""}`}
              onClick={handleLike}
              aria-label={liked ? "ยกเลิกถูกใจ" : "กดถูกใจ"}
              aria-pressed={liked}
            >
              <svg width="14" height="14" viewBox="0 0 14 14" fill="none" aria-hidden="true">
                <path
                  d="M7 12S2 8.5 2 5.5A2.5 2.5 0 017 4a2.5 2.5 0 015 1.5C12 8.5 7 12 7 12z"
                  stroke="currentColor"
                  strokeWidth="1.4"
                  fill={liked ? "currentColor" : "none"}
                />
              </svg>
            </button>
          )}
        </div>

      </div>
    </article>
  );
};

export default NovelCard;