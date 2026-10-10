// ══════════════════════════════════════════════════════════════
//  หน้าสร้างนิยายเรื่องใหม่ — ฝั่งนักเขียน (เวอร์ชันสมบูรณ์)
//
//  เชื่อมต่อ API:
//    - POST /novels      -> สร้างนิยายใหม่ (ข้อมูลตรงตาม Go Struct)
//    - POST /upload/image -> อัปโหลดปกนิยายไปยัง MinIO
// ══════════════════════════════════════════════════════════════

import React, { useState, useEffect } from "react";
import ReactQuill from "react-quill-new";
import "quill/dist/quill.snow.css";
import "./CreateNovelPage.css";
import MultiSelect from "../../../components/MultiSelect/MultiSelect";
import CoverUpload from "../../../components/CoverUpload/CoverUpload";
import Toggle from "../../../components/Toggle/Toggle";
import { useNavigate } from "react-router-dom";
import { getNovelStatusInfo } from "../../../utils/novelStatus";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || "http://localhost:8080";

// ── ค่า default ของ form ──────────────────────────────────────
const INITIAL_FORM = {
    title: "",
    tagline: "",          
    categories: [],       
    description: "",      
    coverFile: null,      
    coverPreview: null,
    isPublished: false,
    isCompleted: false,
    statusMode: "draft",
};

const FALLBACK_CATEGORIES = [
    "แฟนตาซี", "โรแมนซ์", "ผจญภัย", "ลึกลับ", "สสยองขวัญ", 
    "ดราม่า", "ตลก", "ชีวิต", "ไซไฟ", "ประวัติศาสตร์",
];

const getStatusFlags = ({ status, is_published, is_completed, isPublished, isCompleted }) => {
    const statusInfo = getNovelStatusInfo({
        status,
        is_published,
        is_completed,
        isPublished,
        isCompleted,
    });
    return { isCompleted: statusInfo.isCompleted, isPublished: statusInfo.isPublished };
};

const getStatusFromFlags = ({ isCompleted, isPublished }) => {
    return getNovelStatusInfo({ is_completed: isCompleted, is_published: isPublished }).mode;
};

const getStatusLabel = ({ isCompleted, isPublished }) => {
    return getNovelStatusInfo({ is_completed: isCompleted, is_published: isPublished }).label;
};

// ── Validation rules ─────────────────────────────────────────
const validate = (form) => {
    const errors = {};
    if (!form.title.trim()) {
        errors.title = "กรุณากรอกชื่อเรื่อง";
    }
    if (!form.tagline.trim()) {
        errors.tagline = "กรุณากรอกคำโปรย";
    }
    if (form.tagline.length > 200) {
        errors.tagline = "คำโปรยต้องไม่เกิน 200 ตัวอักษร";
    }
    if (form.categories.length === 0) {
        errors.categories = "กรุณาเลือกหมวดหมู่อย่างน้อย 1 หมวด";
    }
    
    // 💡 ปรับปรุง: ตรวจสอบความปลอดภัยกรณีที่ฟอร์ม description ยังคงว่างเปล่า
    const plainDescription = form.description ? form.description.replace(/<(.|\n)*?>/g, "").trim() : "";
    if (!plainDescription) {
        errors.description = "กรุณากรอกแนะนำเรื่อง";
    }
    
    return errors;
};

// ════════════════════════════════════════════════════════════
const CreateNovelPage = () => {
    const [form, setForm] = useState(INITIAL_FORM);
    const [errors, setErrors] = useState({});
    const [submissionError, setSubmissionError] = useState(null);
    const [isSubmitting, setIsSubmitting] = useState(false);
    const [categories, setCategories] = useState([]);
    const [categoriesLoading, setCategoriesLoading] = useState(true);
    const [categoriesLoaded, setCategoriesLoaded] = useState(false);
    const [categoriesError, setCategoriesError] = useState(null);
    const navigate = useNavigate();
    const [showCancelModal, setShowCancelModal] = useState(false);

    useEffect(() => {
        const loadCategories = async () => {
            setCategoriesLoading(true);
            setCategoriesError(null);
            try {
                const response = await fetch(`${API_BASE_URL}/categories`);
                if (!response.ok) {
                    throw new Error("failed to fetch categories");
                }
                const result = await response.json();
                const categoriesData = Array.isArray(result)
                    ? result
                    : Array.isArray(result?.data)
                        ? result.data
                        : [];
                if (categoriesData.length === 0) {
                    throw new Error("categories response invalid");
                }
                setCategories(categoriesData.map((category) => ({
                    id: category.category_id || category.id,
                    name: category.name,
                })));
                setCategoriesLoaded(true);
            } catch (err) {
                console.error("Category load error:", err);
                setCategories(FALLBACK_CATEGORIES.map((name, index) => ({ id: index + 1, name })));
                setCategoriesError("ไม่สามารถดึงหมวดหมู่จากระบบได้ กำลังใช้ค่าเริ่มต้นแทน");
            } finally {
                setCategoriesLoading(false);
            }
        };
        loadCategories();
    }, []);

    const categoryOptions = categories.map((cat) => cat.name);

    const updateFormStatus = (nextCompleted, nextPublished) => {
        setForm((prev) => {
            const nextMode = nextCompleted
                ? (nextPublished ? "completed-published" : "completed-draft")
                : (nextPublished ? "published" : "draft");
            return {
                ...prev,
                isCompleted: nextCompleted,
                isPublished: nextPublished,
                statusMode: nextMode,
            };
        });
    };

    // ── Field change helper ──────────────────────────────────
    const setField = (key, value) => {
        setForm((prev) => ({ ...prev, [key]: value }));
        if (errors[key]) {
            setErrors((prev) => {
                const next = { ...prev };
                delete next[key];
                return next;
            });
        }
    };

    // ── Submit handler ───────────────────────────────────────
    const handleSubmit = async (e) => {
        if (e) e.preventDefault(); 

        const validationErrors = validate(form);
        if (Object.keys(validationErrors).length > 0) {
            setErrors(validationErrors);
            setSubmissionError("กรุณากรอกข้อมูลที่จำเป็นให้ครบถ้วนและถูกต้อง");
            return;
        }

        setIsSubmitting(true);
        setSubmissionError(null);

        try {
            const token = localStorage.getItem("token"); 
            if (!token) {
                throw new Error("ไม่พบข้อมูลการเข้าสู่ระบบ (Token) กรุณาล็อกอินใหม่");
            }

            let coverImageUrl = null;

            // 1. อัปโหลดรูปภาพปกไปยังระบบฝากไฟล์ MinIO ก่อน
            if (form.coverFile) {
                const imageFormData = new FormData();
                imageFormData.append("image", form.coverFile ,"cover_novel.jpg"); 

                // 🎯 แก้ไขจุดที่ 1: เปลี่ยนเส้นทาง Endpoint ไปที่ /upload/image
                const uploadRes = await fetch(`${API_BASE_URL}/upload/image`, {
                    method: "POST",
                    headers: { "Authorization": `Bearer ${token}` },
                    body: imageFormData
                });

                if (uploadRes.ok) {
                    const uploadData = await uploadRes.json();
                    console.log("✅ ข้อมูลที่ได้จาก API Upload:", uploadData); 
                    
                    // 🎯 แก้ไขจุดที่ 2: เจาะจงดึงลิงก์รูปแบบยาวจาก full_url ลงสู่ตัวแปรส่งเข้าฐานข้อมูล
                    coverImageUrl = uploadData.data?.full_url || ""; 
                    console.log("🖼️ ลิงก์รูปภาพที่จะใช้บันทึกลงฐานข้อมูล:", coverImageUrl);
                } else {
                    console.error("อัปโหลดภาพปกไม่สำเร็จ แต่ระบบจะดำเนินการสร้างนิยายต่อ");
                }
            }

            // 2. แปลงหมวดหมู่เป็น ID
            const selectedCategoryIds = categories
                .filter(cat => form.categories.includes(cat.name))
                .map(cat => cat.id);

            if (!categoriesLoaded) {
                throw new Error("ไม่สามารถโหลดหมวดหมู่ได้ กรุณารีเฟรชหน้าเพื่ออัปเดต");
            }

            // 3. รวบตรรกะสถานะ (Status) ให้เป็น String เดียว
            const finalStatus = getStatusFromFlags({
                isCompleted: form.isCompleted,
                isPublished: form.isPublished,
            });

            // 4. ประกอบร่าง Payload ให้ตรงกับ Struct Go
            const novelPayload = {
                title: form.title,
                captions: form.tagline,
                introduction: form.description,
                category_ids: selectedCategoryIds,
                cover_image: coverImageUrl,
                status: finalStatus,
                is_published: form.isPublished,
                is_completed: form.isCompleted,
            };

            const response = await fetch(`${API_BASE_URL}/novels`, {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                    "Authorization": `Bearer ${token}`
                },
                body: JSON.stringify(novelPayload)
            });

            if (!response.ok) {
                const errorData = await response.json().catch(() => ({}));
                throw new Error(errorData.message || errorData.error || "ไม่สามารถบันทึกข้อมูลนิยายลงระบบได้");
            }

            alert("✅ สร้างนิยายและบันทึกข้อมูลครบถ้วนเรียบร้อย!");
            navigate("/writer/dashboard");
            
        } catch (error) {
            console.error("Submit Error:", error);
            setSubmissionError(error.message);
            window.scrollTo({ top: 0, behavior: 'smooth' });
        } finally {
            setIsSubmitting(false);
        }
    };

    // ── Cancel with confirmation ───────────────────────
    const handleCancel = () => {
        setShowCancelModal(true);
    };

    const handleConfirmCancel = () => {
        setShowCancelModal(false);
        navigate("/writer/dashboard");
    };

    const taglineLen = form.tagline ? form.tagline.length : 0;

    // ════════════════════════════════════════════════════════
    return (
        <div className="cnp">
            {/* ── Page heading ── */}
            <div className="cnp__header">
                <h1 className="cnp__title">สร้างนิยายเรื่องใหม่</h1>
                <p className="cnp__sub">กรอกข้อมูลเบื้องต้นเพื่อเริ่มต้นเรื่องราวใหม่ของคุณ</p>
            </div>

            {/* ── Main card ── */}
            <div className="cnp__form-wrap">
                <div className="cnp__card">
                    {submissionError && (
                        <div className="cnp__error-banner" role="alert" style={{ marginBottom: "16px" }}>
                            {submissionError}
                        </div>
                    )}

                    {/* ── Card section header ── */}
                    <div className="cnp__section-header">
                        <h2 className="cnp__section-title">รายละเอียดนิยาย</h2>
                        <p className="cnp__section-sub">ข้อมูลเหล่านี้จะแสดงให้นักอ่านเห็นในหน้ารายละเอียดนิยาย</p>
                    </div>

                    {/* ── Two-column layout ── */}
                    <div className="cnp__columns">

                        {/* ════ Left: form fields ════ */}
                        <div className="cnp__left">

                            {/* ── ชื่อเรื่อง ── */}
                            <div className="cnp__field" id="field-title">
                                <label className="cnp__label" htmlFor="inp-title">
                                    ชื่อเรื่อง <span className="cnp__required">*</span>
                                </label>
                                <input
                                    id="inp-title"
                                    type="text"
                                    className={`cnp__input ${errors.title ? "cnp__input--error" : ""}`}
                                    placeholder="ตั้งชื่อเรื่องของคุณ...."
                                    value={form.title}
                                    onChange={(e) => setField("title", e.target.value)}
                                    maxLength={100}
                                    aria-required="true"
                                />
                                {errors.title && (
                                    <p className="cnp__error" role="alert">{errors.title}</p>
                                )}
                            </div>

                            {/* ── คำโปรย ── */}
                            <div className="cnp__field" id="field-tagline">
                                <label className="cnp__label" htmlFor="inp-tagline">
                                    คำโปรย <span className="cnp__required">*</span>
                                </label>
                                <textarea
                                    id="inp-tagline"
                                    className={`cnp__textarea cnp__textarea--sm ${errors.tagline ? "cnp__input--error" : ""}`}
                                    placeholder="บอกเล่าเรื่องราวของคุณสั้นๆ"
                                    value={form.tagline}
                                    onChange={(e) => setField("tagline", e.target.value)}
                                    maxLength={200}
                                    aria-required="true"
                                />
                                <div className="cnp__char-row">
                                    <p className={`cnp__char-count ${taglineLen > 180 ? "cnp__char-count--warn" : ""}`}>
                                        {taglineLen} / 200 ตัวอักษร
                                    </p>
                                    {errors.tagline && (
                                        <p className="cnp__error" role="alert">{errors.tagline}</p>
                                    )}
                                </div>
                            </div>

                            {/* ── หมวดหมู่ ── */}
                            <div className="cnp__field" id="field-categories">
                                <label className="cnp__label">
                                    หมวดหมู่ <span className="cnp__required">*</span>
                                </label>
                                <MultiSelect
                                    options={categoryOptions}
                                    value={form.categories}
                                    onChange={(val) => setField("categories", val)}
                                    placeholder={categoriesLoading ? "กำลังโหลดหมวดหมู่..." : "เลือกหมวดหมู่..."}
                                    max={5}
                                />
                                {categoriesError && (
                                    <p className="cnp__info" style={{ color: "#d97706" }} role="alert">
                                        {categoriesError}
                                    </p>
                                )}
                                {errors.categories && (
                                    <p className="cnp__error" role="alert">{errors.categories}</p>
                                )}
                            </div>

                            {/* ── แนะนำเรื่อง ── */}
                            <div className="cnp__field" id="field-description">
                                <label className="cnp__label" htmlFor="inp-description">
                                    แนะนำเรื่อง <span className="cnp__required">*</span>
                                </label>
                                <div className={`cnp__quill-wrap ${errors.description ? "cnp__quill-wrap--error" : ""}`}>
                                    <ReactQuill
                                        theme="snow"
                                        value={form.description}
                                        onChange={(value) => setField("description", value)}
                                        placeholder="แนะนำเรื่องราวเกี่ยวกับนิยายของคุณ...."
                                        className="cnp__quill"
                                    />
                                </div>
                                {errors.description && (
                                    <p className="cnp__error" role="alert">{errors.description}</p>
                                )}
                            </div>

                        </div>

                        {/* ════ Right: cover + settings ════ */}
                        <div className="cnp__right">

                            {/* ── Cover upload ── */}
                            <div className="cnp__cover-wrap">
                                <CoverUpload
                                    value={form.coverPreview}
                                    onChange={(file, preview) => {
                                        setField("coverFile", file);
                                        setField("coverPreview", preview);
                                    }}
                                />
                            </div>

                            {/* ── การตั้งค่าเบื้องต้น ── */}
                            <div className="cnp__settings">
                                <h3 className="cnp__settings-title">การตั้งค่าเบื้องต้น</h3>

                                {/* Status เรื่อง */}
                                <div className="cnp__setting-row">
                                    <span className="cnp__setting-label">สถานะเรื่อง</span>
                                    <div className="cnp__setting-control">
                                        <Toggle
                                            id="toggle-published"
                                            checked={form.isPublished}
                                            onChange={(val) => updateFormStatus(form.isCompleted, val)}
                                        />
                                        <span className={`cnp__setting-status ${form.isPublished ? "cnp__setting-status--on" : ""}`}>
                                            {form.isPublished ? "เผยแพร่" : "ฉบับร่าง"}
                                        </span>
                                    </div>
                                </div>

                                {/* Status จบ */}
                                <div className="cnp__setting-row">
                                    <span className="cnp__setting-label">สถานะจบ</span>
                                    <div className="cnp__setting-control">
                                        <Toggle
                                            id="toggle-completed"
                                            checked={form.isCompleted}
                                            onChange={(val) => updateFormStatus(val, form.isPublished)}
                                        />
                                        <span className={`cnp__setting-status ${form.isCompleted ? "cnp__setting-status--on" : ""}`}>
                                            {form.isCompleted ? "จบแล้ว" : "ยังไม่จบ"}
                                        </span>
                                    </div>
                                </div>
                                <div className="cnp__setting-row" style={{ marginTop: 8 }}>
                                    <span className="cnp__setting-label">สถานะปัจจุบัน</span>
                                    <span className="cnp__setting-status cnp__setting-status--on" style={{ marginLeft: 8 }}>
                                        {getStatusLabel({ isCompleted: form.isCompleted, isPublished: form.isPublished })}
                                    </span>
                                </div>
                            </div>

                        </div>
                    </div>

                    {/* ── Footer buttons ── */}
                    <div className="cnp__footer">
                        {/* ยกเลิก */}
                        <button
                            type="button"
                            className="cnp__btn cnp__btn--cancel"
                            onClick={handleCancel}
                            disabled={isSubmitting}
                        >
                            ยกเลิก
                        </button>

                        {/* สร้างนิยาย */}
                        <button
                            type="button"
                            className="cnp__btn cnp__btn--submit"
                            onClick={handleSubmit}
                            disabled={isSubmitting}
                            aria-busy={isSubmitting}
                        >
                            {isSubmitting ? (
                                <>
                                    <span className="cnp__spinner" aria-hidden="true" />
                                    กำลังสร้าง...
                                </>
                            ) : (
                                <>
                                    <svg width="14" height="14" viewBox="0 0 14 14" fill="none" aria-hidden="true">
                                        <path d="M2 7l4 4 6-6" stroke="white" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
                                    </svg>
                                    สร้างนิยายเรื่องใหม่
                                </>
                            )}
                        </button>
                    </div>

                </div>
            </div>

            {/* Cancel Confirmation Modal */}
            {showCancelModal && (
                <div className="modal-overlay" onClick={() => setShowCancelModal(false)}>
                    <div className="modal-content" onClick={(e) => e.stopPropagation()}>
                        <h2>ยืนยันการยกเลิก</h2>
                        <p>คุณต้องการยกเลิกการสร้างนิยายใหม่หรือไม่?</p>
                        <p style={{ color: "#999", fontSize: "14px" }}>ข้อมูลที่ยังไม่บันทึกจะสูญหาย</p>
                        
                        <div className="modal-buttons">
                            <button
                                className="btn btn--outline"
                                onClick={() => setShowCancelModal(false)}
                            >
                                ไม่ยกเลิก
                            </button>
                            <button
                                className="btn btn--danger"
                                onClick={handleConfirmCancel}
                            >
                                ยกเลิก
                            </button>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
};

export default CreateNovelPage;
        