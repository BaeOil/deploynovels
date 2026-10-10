import React, { useState, useRef, useEffect } from "react";
import ReactDOM from "react-dom";
import "./WriterRegisterPage.css";
import { useNavigate } from "react-router-dom";
import ReactQuill from "react-quill-new";
import "quill/dist/quill.snow.css";
import DOMPurify from "dompurify"; // npm install dompurify — sanitize bio HTML before ever rendering it

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || "http://localhost:8080";

const STEPS = [
    { num: 1, label: "ข้อมูลส่วนตัว" },
    { num: 2, label: "แนะนำตัว" },
    { num: 3, label: "ช่องทางติดต่อ" },
    { num: 4, label: "ยืนยันข้อมูล" },
];

// ─────────────────────────────────────────────
//  Shared fetch helper (ลดโค้ดซ้ำเรื่อง token + error handling
//  ที่เดิมเขียนแยกกันทั้งใน fetchWriterApplication และ handleSubmit)
// ─────────────────────────────────────────────
async function authFetch(path, options = {}) {
    const token = localStorage.getItem("token");
    const res = await fetch(`${API_BASE_URL}${path}`, {
        ...options,
        headers: {
            ...(options.headers || {}),
            ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
    });

    if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        const error = new Error(data.message || data.error || `HTTP ${res.status}`);
        error.status = res.status;
        throw error;
    }

    return res.json().catch(() => ({}));
}

// ─────────────────────────────────────────────
//  Draft storage: ต้องผูกกับ user_id เสมอ ห้ามใช้ key กลางๆ
//  เพราะบนเครื่องที่ใช้ร่วมกัน user A เขียนร่างค้างไว้แล้วปิดแท็บ (ไม่ยกเลิก/ไม่ส่ง)
//  จากนั้น user B ล็อกอินเครื่องเดียวกัน จะเห็นข้อมูลของ user A ทันทีถ้า key ไม่แยกตาม user
// ─────────────────────────────────────────────
function getCurrentUserId() {
    try {
        const userJson = localStorage.getItem("user");
        if (!userJson) return null;
        const user = JSON.parse(userJson);
        return user.user_id ?? user.id ?? user.email ?? null;
    } catch (e) {
        return null;
    }
}

function getDraftKeys() {
    const uid = getCurrentUserId();
    // ไม่รู้ว่าเป็น user ไหน -> ไม่บันทึกร่างลง localStorage เลย (กันรั่วดีกว่าเสียความสะดวก)
    if (!uid) return { stepKey: null, formKey: null };
    return {
        stepKey: `writerRegStep:${uid}`,
        formKey: `writerRegForm:${uid}`,
    };
}

// ล้าง key รุ่นเก่าที่ไม่ได้ผูกกับ user (ของค้างจากก่อนแก้บั๊กนี้) ทิ้งทันทีที่เจอ
// เพื่อไม่ให้ user คนถัดไปที่เข้ามาในเครื่องเดียวกันเห็นข้อมูลนี้อีก
function purgeLegacyUnscopedDraft() {
    localStorage.removeItem("writerRegStep");
    localStorage.removeItem("writerRegForm");
}

const QUILL_MODULES = {
    toolbar: [
        [{ header: [1, 2, 3, false] }],
        ["bold", "italic", "underline", "strike"],
        [{ list: "ordered" }, { list: "bullet" }],
        ["blockquote"],
        ["link"],
        ["clean"]
    ]
};

const QUILL_FORMATS = [
    "header",
    "bold",
    "italic",
    "underline",
    "strike",
    "list",
    "blockquote",
    "link"
];

// ─────────────────────────────────────────────
//  Sub Component: Cancel Confirmation Modal
// ─────────────────────────────────────────────
const CancelConfirmModal = ({ isOpen, onConfirm, onCancel }) => {
    if (!isOpen) return null;

    // แก้ไข Class Name ให้ตรงกับไฟล์ CSS เพื่อให้ Modal เด้งขึ้นมากลางจออย่างถูกต้อง
    return ReactDOM.createPortal(
        <div className="modal-overlay">
            <div className="modal-content">
                <div className="modal-header">
                    <h2>ยกเลิกการสมัคร</h2>
                    <button className="modal-close" onClick={onCancel} aria-label="ปิด">&times;</button>
                </div>
                <div className="modal-body">
                    <p>คุณแน่ใจหรือไม่ว่าต้องการยกเลิกการสมัครเป็นนักเขียน? ข้อมูลทั้งหมดที่กรอกมาจะไม่ถูกบันทึก</p>
                </div>
                <div className="modal-footer">
                    <button type="button" className="modal-btn modal-btn--cancel" onClick={onCancel}>
                        กรอกข้อมูลต่อ
                    </button>
                    <button type="button" className="modal-btn modal-btn--delete" onClick={onConfirm}>
                        ใช่, ยกเลิกเลย
                    </button>
                </div>
            </div>
        </div>,
        document.body
    );
};

// ─────────────────────────────────────────────
//  Sub Component: Step Indicator
// ─────────────────────────────────────────────
const StepIndicator = ({ current }) => (
    <div className="wr-steps" role="list" aria-label="ขั้นตอนการสมัคร">
        {STEPS.map((step, i) => {
            const isDone = current > step.num;
            const isActive = current === step.num;
            return (
                <React.Fragment key={step.num}>
                    <div
                        className={`wr-step ${isActive ? "wr-step--active" : ""} ${isDone ? "wr-step--done" : ""}`}
                        role="listitem"
                        aria-current={isActive ? "step" : undefined}
                    >
                        <div className="wr-step__circle">
                            {isDone ? (
                                <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
                                    <path d="M2.5 7l3.5 3.5 5.5-6" stroke="white" strokeWidth="1.8"
                                        strokeLinecap="round" strokeLinejoin="round" />
                                </svg>
                            ) : step.num}
                        </div>
                        <span className="wr-step__label">{step.label}</span>
                    </div>
                    {i < STEPS.length - 1 && (
                        <div className={`wr-step__line ${isDone ? "wr-step__line--done" : ""}`} />
                    )}
                </React.Fragment>
            );
        })}
    </div>
);

// ─────────────────────────────────────────────
//  Sub Component: Avatar Upload
// ─────────────────────────────────────────────
const AvatarUpload = ({ preview, onChange }) => {
    const inputRef = useRef(null);

    const handleFile = (file) => {
        if (!file) return;
        if (!["image/jpeg", "image/png", "image/webp"].includes(file.type)) {
            alert("รองรับไฟล์รูปภาพประเภท PNG, JPG, WEBP เท่านั้น");
            return;
        }
        if (file.size > 5 * 1024 * 1024) {
            alert("ขนาดรูปภาพต้องไม่เกิน 5MB นะ");
            return;
        }
        const url = URL.createObjectURL(file);
        onChange(file, url);
    };

    return (
        <div className="wr-avatar">
            <button
                type="button"
                className="wr-avatar__circle"
                onClick={() => inputRef.current?.click()}
                aria-label="อัปโหลดรูปโปรไฟล์นักเขียน"
            >
                {preview ? (
                    <img src={preview} alt="รูปโปรไฟล์" className="wr-avatar__img" />
                ) : (
                    <div className="wr-avatar__placeholder">
                        <svg width="48" height="48" viewBox="0 0 48 48" fill="none">
                            <circle cx="24" cy="18" r="9" stroke="var(--gray-300)" strokeWidth="2" />
                            <path d="M6 42c0-9.941 8.059-18 18-18s18 8.059 18 18"
                                stroke="var(--gray-300)" strokeWidth="2" strokeLinecap="round" />
                        </svg>
                    </div>
                )}

                <div className="wr-avatar__overlay">
                    <svg width="22" height="22" viewBox="0 0 22 22" fill="none">
                        <path d="M7 9l2-3h4l2 3h2a1 1 0 011 1v7a1 1 0 01-1 1H5a1 1 0 01-1-1v-7a1 1 0 011-1h2z"
                            stroke="white" strokeWidth="1.5" fill="none" />
                        <circle cx="11" cy="13" r="2.5" stroke="white" strokeWidth="1.5" fill="none" />
                    </svg>
                </div>

                <div className="wr-avatar__badge" aria-hidden="true">
                    <svg width="12" height="12" viewBox="0 0 12 12" fill="none">
                        <rect x="1" y="1" width="10" height="10" rx="2" fill="white" />
                        <path d="M4 8l1.5-1.5L8 4l1 1-2.5 2.5L5 8H4z" fill="var(--pink-500)" strokeWidth="0" />
                    </svg>
                </div>
            </button>

            <input
                ref={inputRef}
                type="file"
                accept="image/jpeg,image/png,image/webp"
                className="wr-avatar__input"
                onChange={(e) => handleFile(e.target.files?.[0])}
                aria-hidden="true"
                tabIndex={-1}
            />
            <p className="wr-avatar__label">รูปโปรไฟล์นักเขียน</p>
        </div>
    );
};

// ─────────────────────────────────────────────
//  Sub Component: Page Header (เดิมก็อปโครงสร้างนี้ซ้ำ 3 จุด: approved / pending / ฟอร์มหลัก)
// ─────────────────────────────────────────────
const PageHeader = ({ title, subtitle }) => (
    <div className="wr-header-wrapper">
        <div className="wr-header">
            <h1 className="wr-header__title">{title}</h1>
            <p className="wr-header__sub">{subtitle}</p>
        </div>
    </div>
);

// ─────────────────────────────────────────────
//  Sub Component: Genre Pills
// ─────────────────────────────────────────────
const GenrePills = ({ options, selected, onChange, loading, error, onRetry }) => {
    const toggle = (genreId) => {
        if (selected.includes(genreId)) {
            onChange(selected.filter((g) => g !== genreId));
        } else {
            onChange([...selected, genreId]);
        }
    };

    if (loading) {
        return <p className="wr-genres__status">กำลังโหลดประเภทนิยาย...</p>;
    }

    if (error) {
        return (
            <div className="wr-genres__status wr-genres__status--error">
                <p>ไม่สามารถโหลดประเภทนิยายได้: {error}</p>
                <button type="button" className="wr-btn wr-btn--outline" onClick={onRetry}>
                    ลองใหม่
                </button>
            </div>
        );
    }

    return (
        <div className="wr-genres" role="group" aria-label="เลือกประเภทนิยาย">
            {options.map((genre) => (
                <button
                    key={genre.id}
                    type="button"
                    className={`wr-genre-pill ${selected.includes(genre.id) ? "wr-genre-pill--active" : ""}`}
                    onClick={() => toggle(genre.id)}
                    aria-pressed={selected.includes(genre.id)}
                >
                    {genre.name}
                </button>
            ))}
        </div>
    );
};

// ─────────────────────────────────────────────
//  Sub Component: Summary Card
// ─────────────────────────────────────────────
const SummaryCard = ({ data, genreOptions }) => {
    const genreNames = data.genres
        .map((id) => genreOptions.find((g) => g.id === id)?.name)
        .filter(Boolean);

    // sanitize ก่อนโชว์เสมอ เพราะ bio อาจมาจาก endpoint ที่ไม่ได้ผ่าน Quill toolbar
    // (POST /api/writers/apply รับ JSON ตรงได้ — ต้อง sanitize ทั้งฝั่ง client และ backend)
    const safeBio = data.bio
        ? DOMPurify.sanitize(data.bio, {
              ALLOWED_TAGS: ["p", "br", "strong", "em", "u", "s", "ol", "ul", "li", "blockquote", "a", "h1", "h2", "h3"],
              ALLOWED_ATTR: ["href", "target", "rel"],
          })
        : "—";

    const rows = [
        { label: "ชื่อ - นามสกุล", value: data.fullName || "—" },
        { label: "นามปากกา", value: data.penName || "—" },
        { label: "แนะนำตัวนักเขียน", html: safeBio },
        { label: "ประเภทนิยายที่แต่ง", value: genreNames.length ? genreNames.join(", ") : "—" },
        { label: "อีเมลที่ติดต่อได้", value: data.email || "—" },
        { label: "ช่องทางติดต่อหลัก", value: data.mainContact || "—" },
        { label: "ช่องทางอื่นๆ", value: data.otherLinks || "—" },
    ];
    return (
        <div className="wr-summary">
            <div className="wr-summary__avatar">
                {data.avatarPreview ? (
                    <img src={data.avatarPreview} alt="รูปโปรไฟล์" className="wr-summary__avatar-img" />
                ) : (
                    <div className="wr-summary__avatar-placeholder">
                        <span>✍️</span>
                    </div>
                )}
            </div>

            <div className="wr-summary__info">
                {rows.map((row, idx) => (
                    <div key={idx} className="wr-summary__row">
                        <span className="wr-summary__label">{row.label}</span>
                        {row.html !== undefined ? (
                            <span className="wr-summary__value" dangerouslySetInnerHTML={{ __html: row.html }} />
                        ) : (
                            <span className="wr-summary__value">{row.value}</span>
                        )}
                    </div>
                ))}
            </div>
        </div>
    );
};

// ======================================================
//  Main Component: WriterRegisterPage
// ======================================================
const WriterRegisterPage = ({ onComplete, onBack }) => {
    const navigate = useNavigate();
    
    // 0. เคลียร์ร่างรุ่นเก่าที่ไม่ผูกกับ user (บั๊กเดิม) ทิ้งก่อนเสมอ กันรั่วข้ามบัญชี
    purgeLegacyUnscopedDraft();
    const { stepKey: draftStepKey, formKey: draftFormKey } = getDraftKeys();

    // 1. ดึงข้อมูล Step จาก localStorage (ถ้ามี และผูกกับ user คนปัจจุบันเท่านั้น)
    const [step, setStep] = useState(() => {
        if (!draftStepKey) return 1;
        const savedStep = localStorage.getItem(draftStepKey);
        return savedStep ? Number(savedStep) : 1;
    });

    const [checkingAuth, setCheckingAuth] = useState(true);
    const [cancelModalOpen, setCancelModalOpen] = useState(false);

    // 2. ดึงข้อมูล Form จาก localStorage (ถ้ามี และผูกกับ user คนปัจจุบันเท่านั้น)
    const [form, setForm] = useState(() => {
        const savedForm = draftFormKey ? localStorage.getItem(draftFormKey) : null;
        if (savedForm) {
            try {
                const parsed = JSON.parse(savedForm);
                return {
                    ...parsed,
                    avatarFile: null, // ไฟล์รูปเซฟลง storage ตรงๆ ไม่ได้ ต้องอัปโหลดใหม่
                    avatarPreview: null
                };
            } catch (e) {
                console.error("Failed to parse saved form", e);
            }
        }
        return {
            avatarFile: null,
            avatarPreview: null,
            fullName: "",
            penName: "",
            email: "",
            bio: "",
            genres: [],
            mainContact: "",
            otherLinks: "",
            confirmed: false,
        };
    });

    const [errors, setErrors] = useState({});
    const [isSubmitting, setIsSubmitting] = useState(false);
    const [writerAppStatus, setWriterAppStatus] = useState("none");
    const [writerAppRejectionReason, setWriterAppRejectionReason] = useState(null);
    const [writerAppLoading, setWriterAppLoading] = useState(true);
    const [writerAppError, setWriterAppError] = useState(null);
    const hasPerformedAuthCheck = useRef(false);

    // ประเภทนิยาย: ดึงจาก backend จริง แทนของเดิมที่เป็น array มั่วอยู่ในโค้ด
    const [genreOptions, setGenreOptions] = useState([]);
    const [genresLoading, setGenresLoading] = useState(true);
    const [genresError, setGenresError] = useState(null);
    const [genresReloadKey, setGenresReloadKey] = useState(0);

    // 3. บันทึกข้อมูลลง localStorage ทุกครั้งที่ Step หรือ Form เปลี่ยนแปลง (เฉพาะ key ที่ผูกกับ user แล้วเท่านั้น)
    useEffect(() => {
        if (!draftStepKey) return;
        localStorage.setItem(draftStepKey, step.toString());
    }, [step]);

    useEffect(() => {
        if (!draftFormKey) return;
        // แยกไฟล์รูปออก ไม่เซฟลง localStorage
        const { avatarFile, avatarPreview, ...formToSave } = form;
        localStorage.setItem(draftFormKey, JSON.stringify(formToSave));
    }, [form]);

    // 4. ฟังก์ชันล้างความจำเมื่อทำงานเสร็จสิ้นหรือยกเลิก (ล้างเฉพาะ key ของ user คนปัจจุบัน)
    const clearSavedData = () => {
        if (draftStepKey) localStorage.removeItem(draftStepKey);
        if (draftFormKey) localStorage.removeItem(draftFormKey);
    };

    useEffect(() => {
        const fetchWriterApplication = async () => {
            const token = localStorage.getItem("token");
            if (!token) {
                setWriterAppLoading(false);
                return;
            }

            try {
                const data = await authFetch("/api/writers/me");
                setWriterAppStatus(data.status || "none");
                setWriterAppRejectionReason(data.rejection_reason || null);

                if (data.status === "rejected") {
                    setForm((prev) => ({
                        ...prev,
                        fullName: prev.fullName || data.full_name || data.name_lastname || "",
                        penName: prev.penName || data.pen_name || "",
                        email: prev.email || data.email || data.email_writer || "",
                        bio: prev.bio || data.bio || "",
                        genres: (prev.genres && prev.genres.length > 0) ? prev.genres : (data.category_ids || data.genres || []),
                        mainContact: prev.mainContact || data.main_contact || data.primary_contact || "",
                        otherLinks: prev.otherLinks || data.other_links || data.secondary_contact || "",
                        avatarPreview: prev.avatarPreview || data.avatar_url || null,
                    }));
                }
            } catch (err) {
                // backend ส่ง 200 + {status:"none"} เสมอเมื่อยังไม่เคยสมัคร (sql.ErrNoRows)
                // ดังนั้น catch ตรงนี้คือ error จริงๆ เท่านั้น (401/500 ฯลฯ)
                console.error("Failed to load writer application status:", err);
                setWriterAppError(err.message || "เกิดข้อผิดพลาดในการตรวจสอบสถานะคำขอ");
            } finally {
                setWriterAppLoading(false);
            }
        };

        fetchWriterApplication();
    }, []);

    // ดึงประเภทนิยายจริงจาก backend — GET /categories คืน {status, message, data:[{category_id, name}]}
    useEffect(() => {
        let cancelled = false;

        const fetchGenres = async () => {
            setGenresLoading(true);
            setGenresError(null);
            try {
                const data = await authFetch("/categories");
                const list = Array.isArray(data) ? data : data.data || data.categories || [];
                if (!cancelled) {
                    setGenreOptions(list.map((c) => ({ id: c.category_id, name: c.name })));
                }
            } catch (err) {
                if (!cancelled) {
                    console.error("Failed to load genres:", err);
                    setGenresError(err.message || "โหลดประเภทนิยายไม่สำเร็จ");
                }
            } finally {
                if (!cancelled) setGenresLoading(false);
            }
        };

        fetchGenres();
        return () => { cancelled = true; };
    }, [genresReloadKey]);

    useEffect(() => {
        if (hasPerformedAuthCheck.current) return;
        hasPerformedAuthCheck.current = true;

        const token = localStorage.getItem("token");
        const userJson = localStorage.getItem("user");

        if (userJson) {
            try {
                const user = JSON.parse(userJson);
                if (user.role === "writer" || user.is_writer === true) {
                    alert("คุณเป็นนักเขียนอยู่แล้ว ไม่สามารถสมัครซ้ำได้ 🎉");
                    clearSavedData(); // เคลียร์ข้อมูลทิ้ง
                    navigate("/writer/dashboard");
                    return;
                }
            } catch (e) {
                console.error("Failed to parse local user status:", e);
            }
        }
        setCheckingAuth(false);
    }, [navigate]);

    const setField = (key, value) => {
        setForm((prev) => ({ ...prev, [key]: value }));
        if (errors[key]) setErrors((prev) => { const n = { ...prev }; delete n[key]; return n; });
    };

    const validateStep = (s) => {
        const e = {};
        if (s === 1) {
            if (!form.fullName.trim()) e.fullName = "กรุณากรอกชื่อ-นามสกุล";
            if (!form.penName.trim()) e.penName = "กรุณากรอกนามปากกา";
            if (!form.email.trim()) e.email = "กรุณากรอกอีเมล";
            else if (!/^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$/.test(form.email)) e.email = "กรุณากรอกอีเมลภาษาอังกฤษให้ถูกต้อง";
        }
        if (s === 2) {
            const plainBio = form.bio.replace(/<[^>]*>/g, "").trim();

            if (!plainBio) {
                e.bio = "กรุณาแนะนำตัวตนของคุณสั้นๆ";
            }
            if (form.genres.length === 0) e.genres = "กรุณาเลือกประเภทนิยายอย่างน้อย 1 ประเภท";
        }
        if (s === 3) {
            if (!form.mainContact.trim()) e.mainContact = "กรุณากรอกช่องทางติดต่อหลัก";
        }
        if (s === 4) {
            if (!form.confirmed) e.confirmed = "กรุณากดรับรองว่าข้อมูลทั้งหมดเป็นความจริง";
        }
        return e;
    };

    const handleNext = () => {
        const e = validateStep(step);
        if (Object.keys(e).length) { setErrors(e); return; }
        setErrors({});
        if (step < 4) {
            setStep(step + 1);
            window.scrollTo({ top: 0, behavior: "smooth" });
        } else {
            handleSubmit();
        }
    };

    const handlePrev = () => {
        if (step > 1) {
            setStep(step - 1);
            setErrors({});
            window.scrollTo({ top: 0, behavior: "smooth" });
        } else {
            setCancelModalOpen(true);
        }
    };

    const handleCancelConfirm = () => {
        setCancelModalOpen(false);
        clearSavedData(); // เคลียร์ข้อมูลทิ้งกรณียกเลิก
        navigate("/");
        if (onBack) onBack();
    };

    const handleCancelModal = () => {
        setCancelModalOpen(false);
    };

    const handleSubmit = async () => {
        if (writerAppStatus === "pending") {
            alert("คุณได้ยื่นคำขอเป็นนักเขียนแล้ว กรุณารอการอนุมัติหรือปฏิเสธก่อนสมัครใหม่");
            return;
        }
        if (writerAppStatus === "approved") {
            alert("คำขอของคุณได้รับการอนุมัติแล้ว ไม่สามารถสมัครใหม่ได้อีก");
            navigate("/writer/dashboard");
            return;
        }

        setIsSubmitting(true);
        try {
            const token = localStorage.getItem("token");
            const formData = new FormData();

            formData.append("full_name", form.fullName);
            formData.append("pen_name", form.penName);
            formData.append("email", form.email);
            formData.append("bio", form.bio);
            formData.append("category_ids", JSON.stringify(form.genres));
            formData.append("main_contact", form.mainContact);
            formData.append("other_links", form.otherLinks);

            if (form.avatarFile) {
                formData.append("avatar", form.avatarFile);
            }

            const response = await fetch(`${API_BASE_URL}/api/writers/apply`, {
                method: "POST",
                headers: {
                    "Authorization": `Bearer ${token}`
                },
                body: formData
            });

            if (!response.ok) {
                const errData = await response.json().catch(() => null);
                throw new Error(errData?.error?.message || errData?.message || "การส่งใบสมัครล้มเหลว กรุณาตรวจสอบข้อมูลและลองใหม่อีกครั้ง");
            }

            clearSavedData(); // เคลียร์ข้อมูลทิ้งเมื่อส่งสำเร็จแล้ว

            alert("🎉 ส่งใบสมัครเป็นนักเขียนสำเร็จแล้ว! กรุณารอผู้ดูแลระบบอนุมัติภายใน 1-3 วันทำการ");
            navigate("/");
            onComplete?.();

        } catch (error) {
            console.error("Submission Error:", error);
            alert(error instanceof Error ? error.message : "เกิดข้อผิดพลาดในการติดต่อฐานข้อมูลหลังบ้าน");
        } finally {
            setIsSubmitting(false);
        }
    };

    if (checkingAuth || writerAppLoading) {
        return <div className="wr-loading">กำลังตรวจสอบสิทธิ์และสถานะคำขอ...</div>;
    }

    if (writerAppStatus === "approved") {
        return (
            <div className="wr-page">
                <PageHeader
                    title="คุณได้รับอนุมัติแล้ว"
                    subtitle="คุณสามารถเข้าสู่ระบบและใช้งานพื้นที่นักเขียนได้ทันที"
                />
                <div className="wr-card" style={{ minHeight: 'auto', padding: '40px' }}>
                    <p>คำขอสมัครนักเขียนของคุณได้รับการอนุมัติแล้ว ไม่สามารถยื่นคำขอซ้ำได้อีก</p>
                    <button type="button" className="wr-btn wr-btn--primary" style={{ marginTop: '20px' }} onClick={() => navigate("/writer/dashboard")}>ไปที่ Writer Dashboard</button>
                </div>
            </div>
        );
    }

    if (writerAppStatus === "pending") {
        return (
            <div className="wr-page">
                <PageHeader
                    title="สถานะการสมัคร: รอตรวจสอบ"
                    subtitle="ใบสมัครของคุณกำลังอยู่ในขั้นตอนการพิจารณา"
                />
                <div className="wr-card" style={{ minHeight: 'auto', textAlign: 'center', padding: '60px 20px', display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
                    <div style={{ fontSize: '48px', marginBottom: '16px' }}>⏳</div>
                    <h2 style={{ fontFamily: "'Open Sans', serif", color: 'var(--ink)', marginBottom: '8px' }}>คุณได้ส่งใบสมัครไปแล้ว</h2>
                    <p style={{ color: 'var(--gray-600)', marginBottom: '32px', lineHeight: '1.6' }}>
                        กรุณารอผู้ดูแลระบบตรวจสอบและอนุมัติใบสมัครของคุณภายใน 1-3 วันทำการ<br/>
                        หากคำขอถูกปฏิเสธ คุณจึงจะสามารถกลับมาแก้ไขข้อมูลและส่งใบสมัครใหม่ได้
                    </p>
                    <button type="button" className="wr-btn wr-btn--primary" onClick={() => navigate("/")}>กลับสู่หน้าหลัก</button>
                </div>
            </div>
        );
    }

    return (
        <>
            <div className="wr-page">
                <PageHeader
                    title="สมัครเป็นนักเขียน"
                    subtitle="กรอกข้อมูลเพื่อยืนยันตัวตนของคุณในฐานะนักเขียน"
                />

                {writerAppStatus === "rejected" && (
                    <div className="wr-form-notice wr-form-notice--warning">
                        <p className="wr-form-notice__title">คำขอครั้งก่อนถูกปฏิเสธ</p>
                        {writerAppRejectionReason ? (
                            <p className="wr-form-notice__reason">เหตุผล: {writerAppRejectionReason}</p>
                        ) : (
                            <p className="wr-form-notice__reason wr-form-notice__reason--muted">
                                ผู้ดูแลระบบไม่ได้ระบุเหตุผลเพิ่มเติม
                            </p>
                        )}
                        <p>คุณสามารถแก้ข้อมูลและยื่นสมัครใหม่ได้</p>
                    </div>
                )}

                <StepIndicator current={step} />

                {/* STEP 1: ข้อมูลส่วนตัว */}
                {step === 1 && (
                    <div className="wr-step-content">
                        <div className="wr-card wr-card--compact">
                            <div className="wr-avatar-center">
                                <AvatarUpload
                                    preview={form.avatarPreview}
                                    onChange={(file, url) => { setField("avatarFile", file); setField("avatarPreview", url); }}
                                />
                            </div>

                            <h2 className="wr-card__title wr-text-center">ข้อมูลส่วนตัว</h2>

                            <div className="wr-field">
                                <label className="wr-label" htmlFor="fullName">ชื่อ - นามสกุล</label>
                                <div className={`wr-input-wrap ${errors.fullName ? "wr-input-wrap--error" : ""}`}>
                                    <input id="fullName" className="wr-input" type="text"
                                        placeholder="กรอกชื่อ - นามสกุลของคุณ"
                                        value={form.fullName}
                                        onChange={(e) => setField("fullName", e.target.value)} />
                                </div>
                                {errors.fullName && <p className="wr-field__error" role="alert">{errors.fullName}</p>}
                            </div>

                            <div className="wr-field">
                                <label className="wr-label" htmlFor="penName">นามปากกา</label>
                                <div className={`wr-input-wrap ${errors.penName ? "wr-input-wrap--error" : ""}`}>
                                    <input id="penName" className="wr-input" type="text"
                                        placeholder="กรอกนามปากกา"
                                        value={form.penName}
                                        onChange={(e) => setField("penName", e.target.value)} />
                                </div>
                                {errors.penName && <p className="wr-field__error" role="alert">{errors.penName}</p>}
                            </div>

                            <div className="wr-field">
                                <label className="wr-label" htmlFor="email">อีเมล</label>
                                <div className={`wr-input-wrap ${errors.email ? "wr-input-wrap--error" : ""}`}>
                                    <input id="email" className="wr-input" type="email"
                                        placeholder="กรอกอีเมล"
                                        inputMode="email"
                                        autoComplete="email"
                                        pattern="[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}"
                                        value={form.email}
                                        onChange={(e) => setField("email", e.target.value.replace(/[^A-Za-z0-9._%+@-]/g, ""))} />
                                </div>
                                {errors.email && <p className="wr-field__error" role="alert">{errors.email}</p>}
                            </div>

                            <div className="wr-step-nav" style={{ marginTop: 32 }}>
                                <button type="button" className="wr-btn wr-btn--outline" onClick={handlePrev}>
                                    ยกเลิกการสมัคร
                                </button>
                                <button type="button" className="wr-btn wr-btn--primary" onClick={handleNext}>
                                    ถัดไป
                                </button>
                            </div>
                        </div>
                    </div>
                )}

                {/* STEP 2: แนะนำตัว */}
                {step === 2 && (
                    <div className="wr-step-content">
                        <h2 className="wr-section-title">แนะนำตัว</h2>

                        <div className={`wr-input-wrap ${errors.bio ? "wr-input-wrap--error" : ""}`}>
                            <ReactQuill
                                theme="snow"
                                value={form.bio}
                                onChange={(value) => setField("bio", value)}
                                modules={QUILL_MODULES}
                                formats={QUILL_FORMATS}
                                placeholder="อธิบายความเป็นตัวคุณ สไตล์งานเขียน หรือผลงานของคุณสั้นๆ..."
                            />
                        </div>

                        <div className="wr-field">
                            <label className="wr-label">ประเภทนิยายที่แต่ง (เลือกได้มากกว่า 1 ประเภท)</label>
                            <GenrePills
                                options={genreOptions}
                                selected={form.genres}
                                onChange={(val) => setField("genres", val)}
                                loading={genresLoading}
                                error={genresError}
                                onRetry={() => setGenresReloadKey((k) => k + 1)}
                            />
                            {errors.genres && <p className="wr-field__error" role="alert">{errors.genres}</p>}
                        </div>

                        <div className="wr-step-nav">
                            <button type="button" className="wr-btn wr-btn--outline" onClick={handlePrev}>ย้อนกลับ</button>
                            <button type="button" className="wr-btn wr-btn--primary" onClick={handleNext}>ถัดไป</button>
                        </div>
                    </div>
                )}

                {/* STEP 3: ช่องทางติดต่อ */}
                {step === 3 && (
                    <div className="wr-step-content">
                        <h2 className="wr-section-title">ช่องทางติดต่อ</h2>

                        <div className="wr-field">
                            <label className="wr-label" htmlFor="mainContact">ช่องทางติดต่อหลัก</label>
                            <div className={`wr-input-wrap ${errors.mainContact ? "wr-input-wrap--error" : ""}`}>
                                <input
                                    id="mainContact"
                                    className="wr-input"
                                    type="text"
                                    placeholder="แนบลิงก์โซเชียลมีเดียหลัก เช่น Facebook / IG / Twitter หรือเบอร์โทรติดต่อ"
                                    value={form.mainContact}
                                    onChange={(e) => setField("mainContact", e.target.value)}
                                />
                            </div>
                            {errors.mainContact && <p className="wr-field__error" role="alert">{errors.mainContact}</p>}
                        </div>

                        <div className="wr-field" style={{ marginTop: 20 }}>
                            <label className="wr-label" htmlFor="otherLinks">ช่องทางอื่นๆ</label>
                            <div className="wr-input-wrap">
                                <textarea
                                    id="otherLinks"
                                    className="wr-textarea wr-textarea--sm"
                                    placeholder="แนบลิงก์ผลงานเก่าๆ บล็อก หรือช่องทางการติดตามเพิ่มเติม (ไม่บังคับ)"
                                    rows={4}
                                    value={form.otherLinks}
                                    onChange={(e) => setField("otherLinks", e.target.value)}
                                />
                            </div>
                        </div>

                        <div className="wr-step-nav">
                            <button type="button" className="wr-btn wr-btn--outline" onClick={handlePrev}>ย้อนกลับ</button>
                            <button type="button" className="wr-btn wr-btn--primary" onClick={handleNext}>ถัดไป</button>
                        </div>
                    </div>
                )}

                {/* STEP 4: ยืนยันข้อมูล */}
                {step === 4 && (
                    <div className="wr-step-content">
                        <div className="wr-confirm-header-wrapper">
                            <div className="wr-confirm-header">
                                <h2 className="wr-section-title">ยืนยันข้อมูล</h2>
                                <p className="wr-confirm-sub">ตรวจสอบข้อมูลของคุณ</p>
                            </div>
                        </div>

                        <SummaryCard data={form} genreOptions={genreOptions} />

                        <div className="wr-confirm-check" style={{ marginTop: 20 }}>
                            <label className="wr-checkbox" htmlFor="confirmed">
                                <input
                                    id="confirmed"
                                    type="checkbox"
                                    className="wr-checkbox__input"
                                    checked={form.confirmed}
                                    onChange={(e) => setField("confirmed", e.target.checked)}
                                />
                                <span className={`wr-checkbox__box ${form.confirmed ? "wr-checkbox__box--checked" : ""}`} />
                                <span className="wr-checkbox__label">ฉันยอมรับข้อมูลที่ให้ไว้เป็นความจริง</span>
                            </label>
                            {errors.confirmed && <p className="wr-field__error" role="alert">{errors.confirmed}</p>}
                        </div>

                        <div className="wr-step-nav">
                            <button type="button" className="wr-btn wr-btn--outline" onClick={handlePrev}>ย้อนกลับ</button>
                            <button
                                type="button"
                                className="wr-btn wr-btn--primary"
                                onClick={handleNext}
                                disabled={isSubmitting || writerAppStatus === "pending" || writerAppStatus === "approved"}
                                aria-busy={isSubmitting}
                            >
                                {isSubmitting ? <span className="wr-spinner" /> : "ยืนยันและส่งใบสมัคร"}
                            </button>
                        </div>
                    </div>
                )}

                <CancelConfirmModal
                    isOpen={cancelModalOpen}
                    onConfirm={handleCancelConfirm}
                    onCancel={handleCancelModal}
                />
            </div>
        </>
    );
};

export default WriterRegisterPage;