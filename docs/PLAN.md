# PLAN — novel-BE (เอกสารประกอบ Workflow ตาม `AGENTS.md`)

> **เอกสารนี้สร้างใหม่จากหลักฐานของโปรเจกต์ ณ สถานะปัจจุบัน** เพื่อให้กลับมาใช้ workflow
> ตาม `AGENTS.md` (หมวด `Documentation` และ `Rules` ข้อ 1–2) ได้ (ไฟล์ `docs/PLAN.md` และ `docs/TASKS.md` เคยมีใน Git
> — เพิ่มที่ commit `27c648b`, แก้ที่ `93a64ef`, ถูกลบที่ commit `96189a0` ซึ่งเป็น HEAD)
> จึงถูกสร้างใหม่โดย **ไม่กู้คืนจาก Git** และ **ไม่ถือว่า Task เคยได้รับอนุมัติมาก่อน**

**ขอบเขตเอกสาร:** อ่านสถานะ + จัดลำดับงานเท่านั้น ไม่แก้ Production Code / Test Code / SRS / OpenAPI / Spec

---

## 1. Baseline และสภาพแวดล้อม

| รายการ | ค่า | หลักฐาน |
|---|---|---|
| สาขา / HEAD | `BE` / `96189a0` | `git rev-parse`, `git log -3` |
| Working Tree | **79 รายการ** (69 ` M` + 10 `??`) ณ ก่อนแก้เอกสารรอบนี้ (เดิม 76 — เพิ่มจาก `AGENTS.md` ที่ถูกแก้นอกรอบ + `PLAN.md`/`TASKS.md`) | `git status --short` |
| Diff ที่ค้างอยู่ | **64 files, +1637 / −765** (เดิม 63 files, +1428 / −693 ก่อน `AGENTS.md` ถูกแก้) | `git diff --shortstat` |
| 8 ไฟล์ untracked | `dto/validation_rules_test.go`, `handlers/error_disclosure_test.go`, `handlers/media_upload_test.go`, `handlers/validation_rules_test.go`, `service/choice_label_validation_test.go`, `service/media_service_test.go`, `service/novel_category_validation_test.go`, `service/writer_apply_interests_test.go` | `git status --short` |
| Backend | Go + Docker container `novel-be-app-1` (ปัจจุบัน Exited), Postgres `novels_postgres` (:5435), OpenAPI `docs/openapi.yaml` | `docker ps`, `.env` |
| DB ที่แอปอ้างอิง | `POSTGRES_HOST=db`, `PORT=5432`, `DBNAME=database`, `USER=novel_user`; container `novels_postgres` มี alias `db` บน network `novels_network` | `.env`, `docker inspect` |

> ⚠️ **ข้อจำกัดการยืนยันฐานข้อมูล:** จากการ inspect ยืนยันได้เพียงว่า `db` = `novels_postgres`
> = ฐานข้อมูลที่แอปและ integration test ใช้ร่วมกัน (ใบเดียว) และในนั้นมีข้อมูลผสมทั้งที่มีรูปแบบ
> ทดสอบ (`@test.local`) และข้อมูลอื่น **ยังไม่พิสูจน์ได้ว่าเป็น "Test DB" หรือ "Production DB"**
> จึงห้ามสรุปว่าเป็นฐานข้อมูลทดสอบจากชื่อหรือรูปแบบข้อมูลเพียงอย่างเดียว

---

## 2. สถานะงานจำแนกตามหลักฐาน (แยกหมวด ห้ามปนกัน)

### 2.1 Implemented (โค้ดมีอยู่แล้ว — ยืนยันจากไฟล์)
- **FR-STRUCT-04** — `internal/service/scene_publish_service.go`: `SCENE_UNREACHABLE` และ
  `MISSING_START_SCENE` = `severity="warning"`; มีเพียง `ENDING_HAS_OUTGOING_CHOICE` +
  `COMPLETED_WITHOUT_ENDING` (+ error โหลดข้อมูล `VALIDATION_ERROR`) = `blocking`;
  gate การเผยแพร่ที่ `severity == "blocking"` — **ตรงมติเจ้าของ: Warning อย่างเดียวไม่บล็อก Publish**
- **FR-STRUCT-09** — `internal/repository/chapter_repo.go`: guard ลบ (`hasStartScene && deletedEpisode != 1` →
  `ErrChapterHasStartScene`), promote จำกัดเฉพาะ `deletedEpisode == 1`, reindex ด้วย `+100000 / -100001`,
  ปฏิเสธ `ErrChapterHasIncomingChoices`
- **VAL-01..VAL-11** (ยกเว้น VAL-05) — implement ครบ ชี้ file:line ได้:
  VAL-01 `novel_requests.go:16`, VAL-02 `novel_requests.go:27`, VAL-03 `auth_dto.go:21`,
  VAL-04 `choice_requests.go:9` และ `scene_choice_service.go:16`, VAL-06 `novel_requests.go:35`,
  VAL-07 `writer_dto.go:22`, VAL-08 `media_service.go:19`, VAL-09 `media_service.go:21`,
  VAL-10 `media_service.go:105`, VAL-11 = ไม่มี cap ความยาว (สอดคล้อง)
- **VAL-05 (`hint`)** — **ยกเลิกแล้ว**: ตรวจแล้วว่าไม่มี `hint` ทั้งในโค้ด Go และ `openapi.yaml` (0 hits) → ห้ามนำกลับมา

### 2.2 Acceptance-Tested (มีรายงานเดิม — ยังไม่มี artifact ในไฟล์)
- **Spec 001** — มีคำว่า "17 PASS / 0 FAIL / 0 UNABLE" อ้างจาก**รายงานสนทนาเดิม** — **ไม่ใช่ผลที่ยืนยันแล้ว ห้ามอ้างเป็นผลที่ผ่าน** (ไม่มีไฟล์บันทึกผลใน repo, ไม่ทราบที่มาของการนับ — Spec 001 มี 6 AC)
- **Spec 002** — รายงานเดิมอ้างว่า `AC-PROG-01..03` ผ่าน (**ยังไม่มี artifact → ยังไม่ถือเป็นหลักฐาน**); **fallback และ start-position ยังเป็น Open Question**
- **Spec 011 / 019 / 020** — รายงานเดิมอ้างว่าพฤติกรรมหลักผ่าน (**ยังไม่มี artifact → ยังไม่ถือเป็นหลักฐาน**); **ยังมี Open Questions**
- ⚠️ การที่ `go test ./...` ผ่าน **ไม่เท่ากับ** Acceptance Test ของทุก Spec ผ่าน
- ⚠️ ค้น `*.md/*.txt` ทั้ง repo แล้ว **ไม่พบไฟล์ใดบันทึกผล PASS/FAIL/UNABLE** → ผล acceptance ยังอยู่เฉพาะรายงานสนทนา
- ⚠️ **AC ของอีก 21 Specs (003–010, 012–018, 021–026 — ประมาณ 90 AC นับแบบประมาณจากหัวข้อใน §4, ยังไม่ได้ตรวจรายข้อ; โดย Spec 017 มีงานครอบเพียงบางส่วนตาม T1/T2) ไม่มีงาน/วิธีตรวจในเอกสารนี้ → ยังไม่มีหลักฐานตรวจสอบ และอยู่นอกขอบเขตแผนปัจจุบัน (ดู §6)**

### 2.3 Verified (มีหลักฐานยืนยันแล้วในรอบ inspect)
- สถานะ Working Tree = baseline 76 รายการ (ตรวจก่อน/หลังเขียนเอกสาร ณ ตอนนั้น; ปัจจุบัน 79 รายการ — ดู §1)
- FR-STRUCT-04 ฝั่งโค้ดสอดคล้องมติ (severity + gate อ่านจากไฟล์จริง)
- VAL-05 ไม่มีจริงในโค้ด/Contract

### 2.4 Needs Fix (มีหลักฐานว่าต้องแก้ แต่ยังไม่ได้แก้)
- **Test Infrastructure รั่ว**: integration test ที่ใช้ `openTestDB` (`auth_restore_integration_test.go`,
  `choice_graph_lock_integration_test.go`, `reading_progress_provenance_integration_test.go`,
  `scene_repo_test.go`) อ่าน `POSTGRES_HOST` แล้วเขียนข้อมูลจริงโดยไม่มี rollback →
  สร้างบัญชี `restore_user_*` มี timestamp สะสม (ตรวจพบ 28 บัญชี) ทุกครั้งที่รัน

### 2.5 Open Question (ยังไม่ตัดสินใจ — ห้ามแอบสรุป)
- **Spec 017 OQ (L138)** — ยังเขียนว่า "ยังไม่ตัดสินใจ" ทั้งที่มติ FR-STRUCT-04 แล้ว → **ต้องปิดแยกต่างหาก**
- **Spec 017 OQ อีก 5 ข้อ (L136, L137, L139, L140, L141)** — ตรวจพบเป็น bullet ใน section Open Questions (จำนวนฉากเริ่มต้น / ผล reindex ต่อตอนอื่น / การกำหนดฉากเริ่มต้นใหม่ / เงื่อนไขก่อนเผยแพร่ / ข้อมูลที่แสดงโครงสร้าง): **ไม่พบข้อความตอบรับกำกับใน bullet → ยังไม่อนุมานว่าเปิดหรือปิด ต้องตรวจต่อและรอเจ้าของยืนยันรายข้อ**
- **Spec 002 OQ** — fallback (scene ที่บันทึกไว้เข้าถึงไม่ได้) และ start-position (ยังไม่มี progress) ยังไม่ระบุใน SRS
- **Spec 011 OQ** — section ที่ L75 (ตรวจพบ 5 bullet — ยังไม่ได้อ่านทุกข้อ จึงยังไม่ยืนยันสถานะรายข้อ); **Spec 019 OQ** — (L70-71) ขอบเขตการ preview; **Spec 020 OQ** — (L64) แสดงสถิตินิยายที่ยังไม่เผยแพร่หรือไม่
- **13 Specs ที่ยังไม่ได้ Review และมี bullet ใน Open Questions ที่ยังไม่ได้ลงรายละเอียดในเอกสารนี้** — รายชื่อและจำนวน bullet ที่ตรวจพบ: 009 (4), 010 (4), 012 (4), 013 (5), 014 (5), 015 (5), 016 (5), 018 (7), 021 (4), 022 (5), 024 (8), 025 (7), 026 (6) — สุ่มอ่านข้อความแล้ว 4 ฉบับ (009/014/018/024): 009/014/024 เป็นคำถาม ไม่พบคำตอบกำกับ, **018 ที่ L110 มีข้อยุติกำกับแล้ว (ขีดฆ่า + ข้อความ "ได้รับข้อยุติแล้ว")**; **ฉบับอื่นยังไม่ได้อ่านทีละข้อ → ยังไม่ยืนยันสถานะรายข้อ (เป็นงานตรวจต่อ — ไม่ตอบคำถามแทนเจ้าของ)**
- **XA-08** — 26 Spec มี `Version`/`Last reviewed` แต่ **17 ฉบับยัง "ยังไม่ได้ Review"** → รอนโยบายจากเจ้าของ

### 2.6 Awaiting Approval (ต้องอนุมัติก่อนเริ่ม)
- การ cleanup ข้อมูล `@test.local` (ทำลายข้อมูล)
- การแก้ Test Infrastructure (code change)
- การสร้าง/รัน Integration Test ของ FR-STRUCT-09
- การ re-verify + บันทึก Acceptance Test
- การเติม Version/Last reviewed (XA-08)
- การยืนยัน/isolate ฐานข้อมูลสำหรับ test

---

## 3. ข้อจำกัดและความเสี่ยงที่ยังไม่ยืนยัน

1. **ฐานข้อมูล** — ยังไม่แยกได้ว่าเป็น Test หรือ Production (ดู §1) → ห้ามรัน test ที่เขียนข้อมูลจนกว่าจะยืนยัน/จัด isolate
2. **Foreign Key จริง** ของ `reading_progress`, `user_scene_history` → `users` ยังไม่ถูกอ่านจาก schema (schema ภายนอก ไม่มี AutoMigrate) → ต้องตรวจสอบก่อน cleanup
3. **ผลรัน test ทุกชุด** — ไม่ได้รันในรอบ inspect → ยังยืนยัน PASS ไม่ได้
4. **ผล Acceptance Test** ของ 001/002/011/019/020 — เดิมไม่มี artifact → ต้อง re-verify
5. **8 ไฟล์ untracked** เป็น test ของ VAL แต่ยังไม่ review/commit และยังไม่ยืนยันว่าผ่าน

---

## 4. แนวทางและมติที่ใช้ (Design Decisions)

| มติ | แนวทาง | ผลกระทบต่อเอกสาร |
|---|---|---|
| FR-STRUCT-04 | `SCENE_UNREACHABLE` = Warning, Warning อย่างเดียวไม่บล็อก, Blocking อื่นบล็อกตามเดิม | ปิด Spec 017 OQ (L138) **แยกต่างหาก**; พิจารณา SRS L353 / RTM L723 |
| FR-STRUCT-09 | Implementation ถูกต้องตามเอกสาร แต่ **ขาดหลักฐาน integration test** | ต้องมี Task สร้าง test บน DB ที่ isolate ปลอดภัย |
| FR-VALID-01 | ใช้ **cross-cutting traceability** ตามแนวทางปัจจุบัน (RTM L766 → Constraints §5 ของ 6 Specs) | **ยังไม่สร้าง Spec 027** |
| VAL-05 | ยกเลิกถาวร ไม่มี `hint` | ห้ามเพิ่มใน Model/DTO/API/DB |
| OpenAPI | `docs/openapi.yaml` เป็น **Contract** ของทุก Endpoint/Payload (`AGENTS.md` `Rules` ข้อ 5) | ถ้าจะเปลี่ยน API ต้องแก้ Contract ก่อนแล้วถามเจ้าของ |

### ตารางตรวจ Constraints (เทียบแหล่งข้อกำหนดจริง — ยังไม่สร้างเกณฑ์ใหม่)

| แหล่ง (หลักฐานจริง) | จำนวนข้อ | งาน/วิธีตรวจที่มีในเอกสารนี้ | สถานะหลักฐาน |
|---|---|---|---|
| SRS §2.4 Constraints (L183–189) | 7 | ยังไม่มีงานตรวจระบุ | ยังไม่ตรวจ |
| SRS §3.3 NFR (L527–529) | ไม่กำหนดตัวเลข | แผนนี้ไม่เพิ่มตัวเลข NFR ใหม่ (สอดคล้อง) | ตรวจแล้วว่าเอกสารนี้ไม่ได้เพิ่มตัวเลข NFR |
| SRS §3.4 Domain (L533–551) | 4 หัวข้อ | ยังไม่มีงานตรวจระบุ | ยังไม่ตรวจ |
| SRS VAL-01..VAL-11 (L477–487) | 11 | T6 (ตรวจ 10 ข้อ + ยืนยัน VAL-05 ไม่มี `hint`) | มีงานแล้ว — ยังไม่รัน ยังไม่มีผลตรวจ |
| §5 Constraints ของ 26 Specs | ≈166 ข้อ (นับ bullet แบบประมาณจาก 26 ไฟล์ — ยังไม่ได้ตรวจรายข้อ) | มีเพียงส่วนที่ถูกอ้างใน T2/T6 | ส่วนใหญ่ยังไม่มีงานตรวจ |
| Spec §8 `Version` / `Last reviewed` | 26 ฉบับ (17 = "ยังไม่ได้ Review") | T7 (รอนโยบาย) | มีงานแล้ว — รอการตัดสินใจ |
| `FR-STRUCT-09` / `AC-STRUCT-09` | 1 เรื่อง (7 กรณี) | T2 (integration test) | มีงานแล้ว — `BLOCKED` รอ T0/T3 |
| `FR-STRUCT-04` / `AC-STRUCT-04` (OQ เอกสาร) | 1 เรื่อง | T1 (ปิด OQ L138) | มีงานแล้ว — เป็นงานเอกสาร ไม่ใช่การตรวจพฤติกรรมโค้ด |

> **หมายเหตุ:** ไม่มีรายการในตารางนี้ที่ถือว่า "ผ่าน" เว้นแต่จะมีหลักฐานตรวจสอบจริงในไฟล์ (ตาม `AGENTS.md`: ห้ามรายงาน AC/Constraint ผ่านโดยไม่มีหลักฐาน)

### ไฟล์ที่คาดว่าจะเปลี่ยนและประเภทงาน (สรุปจาก `docs/TASKS.md`)

| Task | ประเภทงาน | ไฟล์/ทรัพยากรที่คาดว่าจะแตะ |
|---|---|---|
| T0 | Governance (ไม่แตะโค้ด/DB) | — (บันทึกมติในเอกสาร) |
| T1 | Documentation | `docs/specs/017-structure/spec.md` (ทางเลือกแตะ SRS/RTM — ยังไม่อนุมัติ) |
| T2 | Test Infrastructure (Test Code + รัน test — ต้องอนุมัติ) | ไฟล์เทสต์ใน `internal/repository/` (ใหม่/เพิ่ม — ยังไม่กำหนดชื่อ) |
| T3 | Test Infrastructure (แก้ Test Code — ต้องอนุมัติ) | 4 ไฟล์เทสต์ตาม TASKS T3 |
| T4 | Data Maintenance (ลบข้อมูล — ต้องอนุมัติ) | ไม่ใช่ไฟล์ — ข้อมูล `@test.local` ใน DB |
| T5 | Verification/Read-only + artifact | ไฟล์ artifact ใหม่ (ยังไม่กำหนดชื่อ) |
| T6 | Read-only (review เท่านั้น) | ไม่แก้ไฟล์ — 8 ไฟล์ untracked เป็นวัตถุ review |
| T7 | Documentation | `docs/specs/*/spec.md` บรรทัด `Version`/`Last reviewed` (รอนโยบาย + ยืนยัน `XA-08`) |

---

## 5. ลำดับงานและ Dependency

```
T0 (อนุมัติ: นโยบาย DB + scope) ─┬─> T1 (ปิด OQ FR-STRUCT-04, no behavior — ทำคู่กันได้ ไม่ต้องรอ T0)
                                  ├─> T3 (แก้ test-infra รั่ว)
                                  │        ├─> T4 (cleanup @test.local)   [ต้องทำหลัง T3]
                                  │        └─> T2 (integration test FR-STRUCT-09) [ต้องทำหลัง T3 + ยืนยัน DB]
                                  ├─> T5 (re-verify acceptance 001/002/011/019/020)
                                  ├─> T6 (review 8 VAL test + ยืนยัน VAL — ทำคู่กันได้ ไม่ต้องรอ T0; ต้องขออนุมัติก่อน commit)
                                  └─> T7 (XA-08 Version/Last reviewed ตามนโยบาย)
```

รายละเอียด Acceptance Criteria และวิธี Verify ของแต่ละ Task อยู่ใน **`docs/TASKS.md`**

---

## 6. นอกขอบเขต (Out of Scope)

- ไม่แก้ Production Code / Test Code / SRS / OpenAPI / Spec ใด ๆ ในรอบสร้างเอกสารนี้
- ไม่เพิ่ม `hint` (VAL-05) กลับคืน
- ไม่สร้าง Spec 027
- ไม่รัน test ใด ๆ และไม่เปลี่ยนข้อมูลฐานข้อมูล
- ไม่กู้คืน/checkout/stash เพื่อเปลี่ยน Working Tree เดิม
- ไม่ขยายงานตรวจไปยัง 21 Specs ที่ยังไม่มี Coverage (003–010, 012–018, 021–026) — **อยู่นอกขอบเขตแผนปัจจุบัน และยังไม่มีหลักฐานตรวจสอบ AC ของฉบับเหล่านี้** (เลือกแนวทางคงขอบเขตแบบ risk-based — ดู §2.2)
- ไม่แก้ SRS / RTM / `AGENTS.md` / Spec / OpenAPI ในรอบนี้ (ทางเลือกปรับคำใน §7 ข้อ 7 ยังไม่อนุมัติ)
- ยังไม่เพิ่ม/เลือก env flag สำหรับกันรัน test จนกว่าจะมีข้อเสนอทางเลือกให้เจ้าของพิจารณาก่อน

---

## 7. Open Decisions ที่รออนุมัติจากเจ้าของโครงการ

1. ยืนยัน/จัด isolate ฐานข้อมูลสำหรับ test (ยังไม่สรุปว่าเป็น Test DB)
2. อนุมัติ cleanup ข้อมูล `@test.local` (30 บัญชี) + scope การลบ
3. อนุมัติแก้ Test Infrastructure ให้ไม่รั่ว (code change)
4. อนุมัติสร้าง/รัน Integration Test ของ FR-STRUCT-09
5. ขอบเขต re-verify Acceptance Test + ที่บันทึกผล
6. มติ XA-08 (Version / Last reviewed สำหรับ 17 Spec ที่ยังไม่ Review)
7. (ทางเลือก) แก้ SRS L353 / RTM L723 ให้ warning ชัดเจนขึ้น หรือคงคำเดิม

