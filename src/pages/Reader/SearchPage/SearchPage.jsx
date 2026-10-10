import React, { useEffect, useMemo, useState, useCallback } from "react";
import axios from "axios";
import { useNavigate, useLocation } from "react-router-dom";
import { getNovelStatusInfo } from "../../../utils/novelStatus";
import { Eye, Heart, GitBranch, BookmarkPlus, SlidersHorizontal, Check } from "lucide-react";
import LoadingScreen from "../../../components/LoadingScreen/LoadingScreen";
import "./SearchPage.css";
import { resolveMediaUrl } from "../../../utils/mediaUrl";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || "http://localhost:8080";

const normalizeNovel = (data) => {
  const rawCats = data.categories ?? data.Categories ?? data.category_ids ?? data.CategoryIDs ?? [];
  const statusInfo = getNovelStatusInfo(data);

  const cleanCategories = Array.isArray(rawCats) 
    ? rawCats.map(c => {
        if (!c) return "";
        if (typeof c === "string") return c.trim();
        return String(c.name || c.Title || c.title || c.label || "").trim();
      }).filter(Boolean) 
    : [];
    
  const uniqueCategories = [...new Set(cleanCategories)];

  return {
    id: data.id || data.novel_id,
    title: data.title || "ไม่มีชื่อเรื่อง",
    categories: uniqueCategories,
    coverImage: data.cover_image || data.coverImage || null,
    synopsis: data.captions || data.introduction || data.description || "",
    author: data.pen_name || data.penName || data.author_pen_name || data.author_penName || data.author_name || data.authorName || "ไม่ทราบผู้แต่ง",
    stats: {
      views: data.views || data.view_count || 0,
      likes: data.like_count || data.likes || 0,
      chaptersCount: data.chapters_count ?? data.chaptersCount ?? (data.chapters ? data.chapters.length : 0),
      bookshelfCount: data.bookshelf_count || data.bookshelfCount || data.saved_count || data.added_count || 0,
    },
    status: data.status || "draft",
    isPublished: statusInfo.isPublished,
    isCompleted: statusInfo.isCompleted,
  };
};

const SearchPage = () => {
  const navigate = useNavigate();
  const location = useLocation();

  // ดึงคำค้นหาเริ่มต้นจาก URL Query (?search=...)
  const getSearchFromUrl = useCallback(() => {
    const params = new URLSearchParams(location.search);
    return params.get("search") || "";
  }, [location.search]);

  const [categories, setCategories] = useState([]);
  const [novels, setNovels] = useState([]);
  const [activeCategory, setActiveCategory] = useState(null);
  const [searchQuery, setSearchQuery] = useState(getSearchFromUrl());
  const [searchType, setSearchType] = useState("all"); // all | title | author | synopsis
  const [showFilterPanel, setShowFilterPanel] = useState(false);
  const [sortBy, setSortBy] = useState("relevant"); // relevant | most_read | latest | thai
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  // ฟังการสลับ Search Query จาก URL ตลอดเวลา
  useEffect(() => {
    setSearchQuery(getSearchFromUrl());
  }, [location.search, getSearchFromUrl]);

  // รับ Event จาก Navbar เมื่อพิมพ์ค้นหา
  useEffect(() => {
    const handleSearchChange = (e) => {
      if (e.detail !== undefined) {
        setSearchQuery(e.detail);
      }
    };
    window.addEventListener("search-change", handleSearchChange);
    return () => window.removeEventListener("search-change", handleSearchChange);
  }, []);

  const loadData = async () => {
    setLoading(true);
    setError(null);
    try {
      const [catRes, novelRes] = await Promise.all([
        axios.get(`${API_BASE_URL}/categories`),
        axios.get(`${API_BASE_URL}/novels`),
      ]);

      const catData = catRes.data?.data || catRes.data || [];
      const novelData = novelRes.data?.data?.novels
                     || novelRes.data?.novels
                     || novelRes.data?.data
                     || novelRes.data
                     || [];

      const allNovels = Array.isArray(novelData) ? novelData.map(normalizeNovel) : [];
      const publishedNovels = allNovels.filter(n => n.isPublished);

      const dbCats = Array.isArray(catData)
        ? catData.map(c => ({ id: c.category_id || c.id, name: String(c.name || c.title || "").trim() }))
        : [];

      setNovels(publishedNovels);
      setCategories(dbCats);
    } catch (err) {
      console.error(err);
      setError("ไม่สามารถโหลดข้อมูลนิยายได้ในขณะนี้");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, []);

  // 🎯 กรองและเรียงลำดับนิยาย (Search + Category Filter + Sort Dropdown)
  const filteredAndSortedNovels = useMemo(() => {
    let result = [...novels];

    // 1. กรองด้วยหมวดหมู่
    if (activeCategory) {
      result = result.filter(n => n.categories.includes(activeCategory));
    }

    // 2. กรองด้วยคำค้นหาแยกตามประเภทการค้นหา
    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase().trim();
      if (searchType === "title") {
        result = result.filter(n => n.title.toLowerCase().includes(q));
      } else if (searchType === "author") {
        result = result.filter(n => n.author.toLowerCase().includes(q));
      } else if (searchType === "synopsis") {
        result = result.filter(n => n.synopsis.toLowerCase().includes(q));
      } else {
        // all: ค้นหารวมทั้งหมด
        result = result.filter(n => 
          n.title.toLowerCase().includes(q) ||
          n.author.toLowerCase().includes(q) ||
          n.synopsis.toLowerCase().includes(q)
        );
      }
    }

    // 3. เรียงลำดับนิยาย
    if (sortBy === "most_read") {
      result.sort((a, b) => b.stats.views - a.stats.views);
    } else if (sortBy === "latest") {
      result.sort((a, b) => (b.id || 0) - (a.id || 0));
    } else if (sortBy === "thai") {
      result.sort((a, b) => a.title.localeCompare(b.title, 'th'));
    }

    return result;
  }, [novels, activeCategory, searchQuery, sortBy, searchType]);

  // ตัวเลือกสำหรับประเภทการค้นหาแถบด้านบน
  const searchTypeTabs = [
    { name: "ทั้งหมด", value: "all" },
    { name: "ชื่อเรื่อง", value: "title" },
    { name: "นามปากกา", value: "author" },
    { name: "คำโปรย", value: "synopsis" }
  ];

  return (
    <div className="search-page">
      <div className="search-page-container">
        
        {/* Header ส่วนหัวผลการค้นหา */}
        <header className="search-header">
          <h1 className="search-title">
            {searchQuery.trim() ? (
              activeCategory ? (
                <>
                  ผลการค้นหา: <span className="highlight-text">"{searchQuery}"</span>{" "}
                  <span className="search-category-in">ในหมวดหมู่</span>{" "}
                  <span className="highlight-text">{activeCategory}</span>
                </>
              ) : (
                <>ผลการค้นหา: <span className="highlight-text">"{searchQuery}"</span></>
              )
            ) : activeCategory ? (
              <>หมวดหมู่: <span className="highlight-text">"{activeCategory}"</span></>
            ) : (
              "นิยายทั้งหมด"
            )}
          </h1>
          <p className="search-count">พบ {filteredAndSortedNovels.length} เรื่อง</p>
        </header>

        {/* แถบตัวกรอง Filters Bar */}
        <div className="filter-bar">
          
          {/* ส่วนซ้าย: การเลือกประเภทที่ค้นหา (แทนที่หมวดหมู่เดิม) */}
          <div className="search-type-tabs" role="tablist" aria-label="ประเภทการค้นหา">
            {searchTypeTabs.map((tab, idx) => {
              const isActive = searchType === tab.value;
              return (
                <button
                  key={idx}
                  type="button"
                  role="tab"
                  aria-selected={isActive}
                  className={`search-type-tab-btn ${isActive ? "active" : ""}`}
                  onClick={() => setSearchType(tab.value)}
                >
                  {tab.name}
                </button>
              );
            })}
          </div>

          {(searchQuery || activeCategory || sortBy !== "relevant" || searchType !== "all") && (
            <button
              type="button"
              className="btn-clear-filters"
              onClick={() => {
                setSearchQuery("");
                setActiveCategory(null);
                setSearchType("all");
                setSortBy("relevant");
                setShowFilterPanel(false);
              }}
            >
              ล้างตัวกรอง
            </button>
          )}

          {/* ส่วนขวา: ปุ่มเปิด-ปิด จัดเรียง/ตัวกรอง */}
          <div className="filter-toggle-wrap">
            <button 
              type="button"
              className={`btn-toggle-filters ${showFilterPanel ? "panel-open" : ""}`}
              aria-expanded={showFilterPanel}
              aria-controls="search-filter-panel"
              onClick={() => setShowFilterPanel(!showFilterPanel)}
            >
              <SlidersHorizontal size={16} />
              <span>จัดเรียง / ตัวกรอง</span>
            </button>
          </div>
        </div>

        {/* แผงควบคุมจัดเรียง/ตัวกรอง Dropdown Panel */}
        {showFilterPanel && (
          <div id="search-filter-panel" className="dropdown-filters-panel">
            <div className="filter-panel-grid">
              
              {/* คอลัมน์เลือกหมวดหมู่ */}
              <div className="panel-filter-column">
                <span className="panel-column-title">🗂️ กรองตามหมวดหมู่</span>
                <div className="panel-categories-list">
                  <button 
                    type="button"
                    className={`panel-cat-btn ${activeCategory === null ? "active" : ""}`}
                    onClick={() => setActiveCategory(null)}
                  >
                    ทั้งหมด
                  </button>
                  {categories.map((cat) => {
                    const isActive = activeCategory === cat.name;
                    return (
                      <button
                        key={cat.id}
                        type="button"
                        className={`panel-cat-btn ${isActive ? "active" : ""}`}
                        onClick={() => setActiveCategory(prev => prev === cat.name ? null : cat.name)}
                      >
                        {cat.name}
                        {isActive && <Check size={12} style={{ marginLeft: "4px" }} />}
                      </button>
                    );
                  })}
                </div>
              </div>

              {/* คอลัมน์จัดเรียงลำดับ */}
              <div className="panel-filter-column sort-column">
                <span className="panel-column-title">↕️ เรียงลำดับตาม</span>
                <div className="panel-sort-list">
                  {[
                    { label: "เกี่ยวข้องมากสุด", value: "relevant" },
                    { label: "ยอดเข้าชมสูงสุด", value: "most_read" },
                    { label: "อัปเดตล่าสุด", value: "latest" },
                    { label: "ก-ฮ (ตามตัวอักษร)", value: "thai" }
                  ].map((opt) => (
                    <button
                      key={opt.value}
                      type="button"
                      className={`panel-sort-btn ${sortBy === opt.value ? "active" : ""}`}
                      onClick={() => setSortBy(opt.value)}
                    >
                      {opt.label}
                    </button>
                  ))}
                </div>
                <button
                  type="button"
                  className="panel-clear-btn"
                  onClick={() => {
                    setSearchQuery("");
                    setActiveCategory(null);
                    setSearchType("all");
                    setSortBy("relevant");
                  }}
                >
                  ล้างตัวกรองทั้งหมด
                </button>
              </div>

            </div>
          </div>
        )}

        {/* Layout แสดงผลลัพธ์แบบเดี่ยวเต็ม 1 คอลัมน์ความกว้าง (ลบ Sidebar ออก) */}
        <div className="search-body-layout-full">
          
          <main className="search-results-area-full">
            {loading ? (
              <LoadingScreen compact message="กำลังค้นหานิยาย..." />
            ) : error ? (
              <div className="search-error">{error}</div>
            ) : filteredAndSortedNovels.length === 0 ? (
              <div className="search-empty-state">
                <div className="empty-icon">🔍</div>
                <h3>ไม่พบผลการค้นหา</h3>
                <p>ลองใช้คำค้นหาอื่น หรือเปลี่ยนตัวกรองหมวดหมู่ดูสิ</p>
              </div>
            ) : (
              <div className="novel-list-vertical-full">
                {filteredAndSortedNovels.map((novel) => {
                  const isFinished = novel.isCompleted;

                  return (
                    <article 
                      key={novel.id} 
                      className="novel-horiz-card-full"
                      onClick={() => navigate(`/novel/${novel.id}`)}
                      onKeyDown={(event) => {
                        if (event.key === "Enter" || event.key === " ") {
                          event.preventDefault();
                          navigate(`/novel/${novel.id}`);
                        }
                      }}
                      role="link"
                      tabIndex={0}
                    >
                      {/* หน้าปก */}
                      <div className="novel-horiz-cover-full">
                        {novel.coverImage ? (
                          <img 
                            src={resolveMediaUrl(novel.coverImage)} 
                            alt={novel.title} 
                            loading="lazy"
                          />
                        ) : (
                          <div className="novel-cover-placeholder-full">📘</div>
                        )}
                      </div>

                      {/* รายละเอียด */}
                      <div className="novel-horiz-details-full">
                        <div className="novel-horiz-header-full">
                          <h3 className="novel-horiz-title-full">{novel.title}</h3>
                          <span className="novel-horiz-author-full">✍️ {novel.author}</span>
                        </div>

                        <p className="novel-horiz-synopsis-full">{novel.synopsis}</p>

                        <div className="novel-horiz-footer-full">
                          {/* Tags หมวดหมู่ */}
                          <div className="novel-tags-full">
                            {novel.categories.map((cat, cIdx) => (
                              <span key={cIdx} className="novel-tag-item-full">{cat}</span>
                            ))}
                          </div>

                          {/* สถิติใช้งาน Lucide-react เหมือนหน้าหมวดหมู่ */}
                          <div className="novel-meta-info-full">
                            <div className="novel-horiz-stat-item-full" title="เพิ่มเข้าชั้นหนังสือ">
                              <BookmarkPlus size={15} color="#db2777" />
                              <span>{novel.stats.bookshelfCount.toLocaleString()}</span>
                            </div>
                            <div className="novel-horiz-stat-item-full" title="ยอดเข้าชม">
                              <Eye size={15} color="#db2777" />
                              <span>{novel.stats.views.toLocaleString()}</span>
                            </div>
                            <div className="novel-horiz-stat-item-full" title="ยอดถูกใจ">
                              <Heart size={15} color="#db2777" />
                              <span>{novel.stats.likes.toLocaleString()}</span>
                            </div>
                            <span className={`status-badge-full ${isFinished ? "finished" : "writing"}`}>
                              {isFinished ? "จบแล้ว" : "กำลังเขียน"}
                            </span>
                          </div>
                        </div>
                      </div>
                    </article>
                  );
                })}
              </div>
            )}
          </main>

        </div>
      </div>
    </div>
  );
};

export default SearchPage;