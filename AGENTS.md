# Agent Guidelines (AGENTS.md)

## What This Project Is

Interactive Fiction Platform (Final Year Project - IT)

เว็บแอปพลิเคชันสำหรับอ่านและเขียนนิยายทางเลือกแบบเลือกเส้นทาง โดยนำเสนอโครงสร้างเนื้อเรื่องในรูปแบบ Directed Acyclic Graph (DAG)

สถานะปัจจุบัน: MVP ทำงานได้แล้ว อยู่ในช่วงเก็บตกความเสถียร แก้ Bug และพัฒนาฟีเจอร์เพิ่มเติม เช่น Profile, Settings, Admin Tools และ Notifications

## Tech Stack & Architecture

- **Backend:** Go 1.22+ Standard Library (`net/http`, `http.ServeMux`) — ห้ามใช้ Web Framework เช่น Gin/Fiber
- **Database:** PostgreSQL (`database/sql` + `github.com/lib/pq`)
- **Storage:** MinIO สำหรับเก็บรูปภาพและไฟล์สื่อ
- **Auth & Config:** JWT (`golang-jwt/jwt/v5`) + Viper (`spf13/viper`)
- **Frontend:** React + Vite + JavaScript (JSX) + React Router + React Flow (`@xyflow/react`) + Tailwind CSS / Custom CSS
- **Deployment Stack:** Docker Compose (Local/Infra Stack), Railway (Backend Server), Vercel (Frontend)

ห้ามเปลี่ยน Tech Stack หรือเพิ่ม Dependency โดยไม่มีเหตุผลและการอนุมัติจาก Human ตามกฎด้านล่าง

---

## Domain & UX Philosophy

- **Platform Type:** Interactive Novel Platform ไม่ใช่เกม
- **Terminology:** ใช้คำว่า นิยาย (Novel), ตอน (Chapter), ฉาก (Scene), ตัวเลือก (Choice), ตอนจบ (Ending), ผู้อ่าน (Reader) และนักเขียน (Writer)
- หลีกเลี่ยงคำศัพท์เกม เช่น Level, Exp, Inventory และ Combat
- **UI Principles:**
  - Minimalist, Soft Pink Theme, Clean และสบายตา
  - ใช้งานง่ายสำหรับนักเขียนมือใหม่
  - Prefer: Cards, Rounded corners, Large spacing และ Simple hierarchy
  - Avoid: Complex dashboards, Dense tables, Tiny buttons และ Hidden actions
- ห้ามปรับ UX หรือเปลี่ยนพฤติกรรมที่ผู้ใช้มองเห็นนอกเหนือจาก Scope ของ Task ที่ได้รับอนุมัติ

---

## Story & Graph Rules

1. **DAG Constraint:** โครงสร้างฉากและตัวเลือกต้องเป็น Directed Acyclic Graph (DAG) ห้ามเกิด Cycle และต้องตรวจสอบความถูกต้องก่อนบันทึกตัวเลือก
2. **Hierarchy & Relationship:** `Novel` → `Chapter` → `Scene` → `Choice`
   - **Chapter:** ใช้จัดกลุ่มฉาก ไม่มี Text Editor ใน Chapter และหนึ่ง Chapter มีได้หลาย Scene
   - **Scene:** เป็น Node ใน Graph เก็บ Title, Content, Images, Choices และ Ending Settings โดยหนึ่ง Scene สังกัดได้เพียงหนึ่ง Chapter
   - **Choice:** เป็น Edge เชื่อมจาก `from_scene` ไป `to_scene` ในระดับ Scene ไม่ใช่ Chapter
3. **Ending Types:** `Good Ending`, `Bad Ending`, `True Ending`, `Secret Ending`
   - ข้อมูลตอนจบที่เกี่ยวข้อง ได้แก่ `ending_title`, `ending_type` และ `ending_description`
   - ก่อนเปลี่ยนกฎ Ending ให้ตรวจ SRS, Spec และ Implementation ที่เกี่ยวข้อง
4. **React Flow / Story Tree:**
   - Node Types ที่ระบุไว้ ได้แก่ `Chapter`, `Scene` และ `Ending` ให้ตรวจสอบกับ Implementation ปัจจุบันก่อนแก้ไข
   - ห้ามทำให้ Layout Logic เดิมของ Node เสียหาย หากต้องเพิ่มความสามารถ ให้ต่อยอดจากโครงสร้างเดิมและตรวจสอบผลกระทบก่อน

---

## Reading, Preview & State Rules

- `reading_progress` ต้องเก็บ `current_scene_id` ซึ่งเป็นฉากปัจจุบันที่อ่านถึง ไม่ใช่ฉากล่าสุดที่เคยแวะไป
- ปุ่ม **อ่านต่อ (Continue Reading)** ต้องใช้ `reading_progress.current_scene_id`
- **History Page:** แสดง Progress, Chapter, Scene และ Choice ล่าสุด การเปิดดู History ต้องไม่เปลี่ยน Progress
- **Preview Mode:** ต้องจำลองพฤติกรรมผู้อ่านโดยไม่บันทึกหรือกระทบข้อมูล Production จริงใน DB
- ตรวจความสอดคล้องของความคืบหน้ากับตาราง `reading_progress`, `user_scene_history`, `user_choice_history` และ `user_endings` ก่อนแก้ Flow ที่เกี่ยวข้อง
- ห้ามสรุปว่าพฤติกรรมข้างต้นตรงกับ Implementation ปัจจุบันโดยไม่ตรวจโค้ดที่เกี่ยวข้อง

---

## Project Layout

### Backend (`novel-BE/`)

- `cmd/main.go` — Entry point
- `config/` — Viper configuration
- `internal/db/` — DB Connection Setup
- `internal/dto/` — Request/Response DTOs
- `internal/handlers/` — HTTP Handlers
- `internal/middleware/` — JWT Auth & CORS Middleware
- `internal/models/` — Domain Models
- `internal/repository/` — SQL Queries & MinIO Integration
- `internal/routes/` — Route Registration (`http.ServeMux`)
- `internal/service/` — Business Logic (DAG Validation, Progress, Auth)

### Frontend (`novel-FE/`)

- `src/components/` — Reusable UI Components
- `src/data/` — Static & Mock Data
- `src/pages/` — Role-based Pages (`Admin/`, `Auth/`, `Reader/`, `Writer/`)
- `src/style/` — App.css, index.css
- `src/utils/` — Helper functions
- `src/App.jsx` และ `src/main.jsx`

### Documentation

ตรวจสอบว่าไฟล์มีอยู่จริงก่อนอ่านหรืออ้างอิง และอ่านเฉพาะเอกสารที่เกี่ยวข้องกับ Task:

- `docs/SRS.md` — System Requirements Specification และนโยบาย Review/Versioning
- `docs/specs/` — Feature Specs รายรายการ
- `docs/PLAN.md` — แผนดำเนินงานระดับโปรเจกต์ หากมีอยู่
- `docs/TASKS.md` — รายการ Task ระดับโปรเจกต์ หากมีอยู่
- `docs/openapi.yaml` — API Contract หากมีอยู่

ห้ามสมมติว่าไฟล์ที่ระบุมีอยู่จริง หากไม่พบ ให้รายงานตามจริงและตรวจสอบโครงสร้างเอกสารปัจจุบันก่อน

ห้ามสร้าง `docs/SPEC.md` หรือสร้าง ย้าย หรือลบโครงสร้างเอกสารเพื่อให้ตรงกับคำสั่งเก่าที่ไม่สอดคล้องกับ Repository ปัจจุบัน โดยไม่มีเหตุผลและการอนุมัติ

---

## Requirements Engineering Workflow

ใช้ Workflow นี้เมื่อต้องวางแผนหรือพัฒนาฟีเจอร์ โดยปรับให้เหมาะกับ Repository ปัจจุบัน ไม่สร้างเอกสารซ้ำโดยไม่จำเป็น

ลำดับงาน:

`/clarify` → `/plan` → `/tasks` → `/implement T-xx`

ทุกขั้นต้องหยุดให้ Human ตรวจสอบหรืออนุมัติตามกฎของขั้นนั้น ห้ามข้ามไป Implement โดยอัตโนมัติ

### 1. Clarify (`/clarify`)

- อ่าน SRS และ Spec ที่เกี่ยวข้องก่อนเริ่มวางแผน
- ตรวจสอบ Implementation ปัจจุบันเมื่อจำเป็น เพื่อแยกสิ่งที่ระบบทำอยู่จริงออกจากสิ่งที่เอกสารกำหนด
- ระบุความกำกวม ความขัดแย้ง ข้อมูลที่ขาด และคำถามที่ต้องให้ Human ตัดสินใจ
- ถามคำถามที่จำเป็นอย่างกระชับ โดยรวมคำถามที่เกี่ยวข้องกันและหลีกเลี่ยงการถามสิ่งที่เอกสารตอบไว้แล้ว
- ห้ามเดา Business Rules ขยาย Scope เพิ่ม Requirement หรือเลือกแนวทางที่มีผลต่อผู้ใช้และข้อมูลโดยไม่มีหลักฐานหรือการอนุมัติ
- หากยังไม่ได้คำตอบ ให้บันทึกเป็น Open Question และระบุ Task ที่ขึ้นกับคำตอบนั้นว่า `รอคำตอบ` หรือ `Blocked`
- การตัดสินใจที่ Human ยืนยันแล้วสามารถบันทึกเป็น Decision/Assumption ตามรูปแบบเอกสารที่มีอยู่
- เมื่อมีข้อขัดแย้งระหว่าง SRS, Spec, API Contract และโค้ด ให้รายงานก่อนเลือกว่าจะเปลี่ยนส่วนใด

**ผลลัพธ์:** รายการความกำกวม คำถาม และการตัดสินใจที่ต้องยืนยัน หากมีการแก้ Spec ต้องรอ Human Review ก่อนเข้าสู่ขั้นถัดไป

### 2. Plan (`/plan`)

ก่อนแก้โค้ด ต้องเสนอแผนให้ Human ตรวจและอนุมัติ โดยแผนควรครอบคลุม:

1. แนวทางแก้และเหตุผล
2. Requirement และ Spec ที่เกี่ยวข้อง
3. ไฟล์ที่คาดว่าจะเปลี่ยนและ Scope ของการเปลี่ยนแปลง
4. Data Model, API หรือ UI ที่ได้รับผลกระทบ หากเกี่ยวข้อง
5. Constraints และผลกระทบต่อระบบเดิม
6. Test Plan และ Acceptance Criteria ที่ใช้ยืนยันผล
7. ลำดับงานและ Dependencies
8. Open Questions และ Assumptions ที่ยังเหลือ

- หากต้องเลือก Technology หรือแนวทางที่เอกสารไม่ได้กำหนด ให้ถาม Human ก่อน
- ข้อเสนอทางเทคนิคต้องแยกให้ชัดว่าเป็นข้อกำหนดจากเอกสารหรือเป็นข้อเสนอของ Agent
- ห้ามเขียนโค้ดระหว่างขั้น Plan

**ผลลัพธ์:** แผนที่ Human ตรวจและอนุมัติแล้วก่อนเริ่มทำ Tasks

### 3. Tasks (`/tasks`)

- แตกแผนเป็น Task ขนาดเล็กที่ทำและตรวจสอบได้ทีละ Task
- แต่ละ Task ต้องระบุ:
  - Task ID
  - Requirement/Spec ที่รองรับ
  - Acceptance Criteria (AC) ที่ใช้ตรวจ
  - ไฟล์ที่คาดว่าจะเปลี่ยน
  - Dependencies
  - เงื่อนไข Done
  - สถานะ
- ใช้สถานะให้สอดคล้องกับเอกสารปัจจุบัน เช่น `พร้อมทำ`, `รอคำตอบ`, `รออนุมัติ` หรือ `Blocked`
- ทุก AC ต้องมี Task ที่รองรับ หรือระบุวิธีตรวจสอบอย่างชัดเจน
- ทุก Constraint ต้องมี Task หรือวิธีตรวจสอบ
- งาน Infrastructure, Test Maintenance และ Cleanup ที่ไม่ได้มาจาก FR โดยตรง ต้องระบุประเภทและเหตุผล ห้ามสร้าง Requirement ID ปลอมเพื่อให้ดู Traceable
- ห้ามสร้าง Plan/Tasks แยกให้ทุก Spec หาก Repository ใช้เอกสารส่วนกลางอยู่แล้ว เว้นแต่ Human อนุมัติหรือมีข้อกำหนดรองรับ
- ห้ามเปลี่ยน ID ที่มีอยู่หรือสร้าง ID ใหม่เองโดยไม่ตรวจรูปแบบและกฎในเอกสารปัจจุบัน

**ผลลัพธ์:** Task list ที่ตรวจสอบ Coverage และ Dependencies แล้ว พร้อมให้ Human อนุมัติก่อน Implement

### 4. Implement (`/implement T-xx`)

- เริ่มได้เฉพาะ Task ที่ Human อนุมัติและพร้อมทำ
- ทำเฉพาะ Task ID ที่ระบุ ห้ามทำ Task อื่นควบคู่หรือขยาย Scope เอง
- อ่านโค้ดและสถานะ Git ปัจจุบันก่อนแก้ เพื่อรักษางานเดิมของผู้ใช้
- แก้ไขให้น้อยที่สุด ไม่เขียนไฟล์ใหม่ทั้งไฟล์หรือ Refactor ที่ไม่จำเป็น
- หากพบว่าจำเป็นต้องแก้ไฟล์นอก Scope ให้หยุด แจ้งเหตุผลและรออนุมัติ
- ตรวจ Test ที่เกี่ยวข้อง และเพิ่มหรือปรับ Test สำหรับ Edge Cases ตามความจำเป็น
- หลัง Implement ให้รันการตรวจสอบที่เหมาะสมกับ Task และรายงานผลจริง
- หยุดหลังจบ Task เพื่อให้ Human Review ก่อนเริ่ม Task ถัดไป
- ห้ามรายงานว่า AC ผ่าน หากไม่มีหลักฐานจาก Test หรือการตรวจสอบที่เหมาะสม
- หาก Test รันไม่ได้ ให้รายงานสาเหตุและแยกจากกรณี Test ไม่ผ่าน

**ผลลัพธ์:** การเปลี่ยนแปลงเฉพาะ Task พร้อมหลักฐานการตรวจสอบและรายงานผลให้ Human Review

---

## Traceability & Acceptance Criteria

- ใช้ Requirement IDs, Spec IDs, AC IDs และ Task IDs ที่มีอยู่ ห้ามเปลี่ยนหรือสร้าง ID ใหม่เองโดยไม่มีเหตุผลและการอนุมัติ
- ทุกการเปลี่ยนแปลงต้อง Trace กลับไปยัง Requirement/Spec หรือระบุให้ชัดว่าเป็นงาน Infrastructure/Maintenance
- AC ต้องระบุพฤติกรรมที่สังเกตหรือตรวจสอบได้ โดยใช้ Given/When/Then เมื่อเหมาะสม
- AC ควรครอบคลุมกรณีปกติ กรณีผิดพลาด และ Edge Cases ที่เกี่ยวข้อง
- Test name และผลทดสอบควรอ้างอิง AC ที่ตรวจเมื่อเหมาะสม แต่ไม่ต้องเปลี่ยนชื่อ Test เดิมที่ใช้งานอยู่โดยไม่จำเป็น
- เมื่อจบงานที่ต้องมีหลักฐาน ให้รายงาน AC ที่ผ่าน ไม่ผ่าน หรือยังตรวจไม่ได้ พร้อมคำสั่งทดสอบและผลจริง
- สร้าง `test-run.txt`, `ac-results.md` หรือ `gaps.md` เฉพาะเมื่อ Scope/Workflow กำหนดหรือ Human อนุมัติ ไม่สร้างไฟล์รายงานซ้ำโดยไม่จำเป็น
- หาก AC ไม่ผ่าน ให้ระบุว่าเกิดจาก Spec, Implementation, Test หรือ Environment เท่าที่มีหลักฐานรองรับ ห้ามเดาสาเหตุ
- การเปลี่ยน Requirement ที่ผ่านการยืนยันหรือ Baseline แล้ว ต้องเสนอ Change Request (CR) และขออนุมัติก่อนแก้เอกสารหรือโค้ดตามการเปลี่ยนแปลงนั้น
- ห้ามเปลี่ยน Version หรือ `Last reviewed` เพียงเพราะมีการแก้ไฟล์ ต้องมีหลักฐานการ Review/อนุมัติตามนโยบายใน SRS
- ห้ามสร้าง Quality Requirement หรือตัวเลขเป้าหมายขึ้นเอง หากเอกสารไม่มีข้อมูลเพียงพอ ให้บันทึกเป็น Open Question

---

## Rules (Must Follow)

1. **Plan ก่อน Code เสมอ:** ห้ามเขียนโค้ดก่อนเสนอแผนสั้น ๆ ให้ Human ตรวจและอนุมัติ
2. **ทำทีละ 1 Task:** ทำเฉพาะ Task ที่ Human อนุมัติจาก `docs/TASKS.md` หากไม่มี Task ที่เหมาะสม ให้เสนอการเพิ่มหรือปรับ Task ก่อนเริ่มโค้ด
3. **Tests:** เมื่อแก้หรือ Refactor โค้ด MVP เดิม ให้ตรวจ Test ที่เกี่ยวข้องและเพิ่ม/ปรับ Test สำหรับ Edge Cases ตามความจำเป็น ห้ามลบหรือแก้ Test เดิมเพียงเพื่อให้ผ่านโดยไม่มีเหตุผลชัดเจน
4. **Business Rules:** บังคับใช้ในชั้นที่เหมาะสมกับสถาปัตยกรรมและฐานข้อมูลปัจจุบัน ห้ามเพิ่มหรือเปลี่ยน DB Constraint/Schema โดยไม่มีการอนุมัติ
5. **API Contract:** Endpoint และ Payload ต้องสอดคล้องกับ `docs/openapi.yaml` หากต้องเพิ่มหรือเปลี่ยน Contract ให้เสนอการแก้ Contract และขอ Human อนุมัติก่อน
6. **Error Response:** Backend ใช้รูปแบบ `{"error": {"code": "...", "message": "..."}}` ตาม Contract ปัจจุบัน หากพบความไม่สอดคล้อง ให้รายงานก่อนปรับรูปแบบ
7. **Authentication:** ใช้ JWT ผ่าน Middleware ใน `internal/middleware/` ตามโครงสร้างที่มีอยู่ ห้ามสร้าง Auth Check ซ้ำซ้อนใน Handlers โดยไม่จำเป็น
8. **Ambiguity:** หากมีความกำกวมที่ส่งผลต่อ Requirement, Data, Security, API หรือพฤติกรรมผู้ใช้ ให้ถาม Human ก่อน ห้ามเดา
9. **Minimal Diff:** จำกัดการเปลี่ยนแปลงให้เล็กที่สุด ห้ามเขียนโค้ดใหม่ทั้งหมดทั้งไฟล์ เว้นแต่ได้รับคำสั่งอย่างชัดเจน
10. **Scope:** ห้ามแก้ไฟล์นอกขอบเขต Task ที่อนุมัติ หากจำเป็นต้องแก้เพิ่ม ให้แจ้งเหตุผลและรออนุมัติ
11. **Git Safety:** ห้ามใช้ `git reset`, `git checkout`, `git restore` หรือ `git stash` เพื่อทิ้งหรือแทนที่งานเดิม ห้าม Commit หรือ Push อัตโนมัติ
12. **Database Safety:** ห้ามแก้ Schema, Migration หรือข้อมูลจริงในฐานข้อมูลโดยไม่มีคำสั่งชัดเจน ตรวจสอบ DB target และผลกระทบก่อนรันคำสั่งที่เปลี่ยนข้อมูล
13. **Production Safety:** Preview/Test ต้องไม่เขียนหรือกระทบข้อมูล Production จริง หากไม่สามารถยืนยันการแยกฐานข้อมูลได้ ให้หยุดและถามก่อน
14. **Reporting:** แยกให้ชัดว่าอะไรตรวจแล้ว อะไรแก้แล้ว อะไรทดสอบผ่าน และอะไรยังไม่ได้ตรวจ ห้ามอ้างผลที่ไม่มีหลักฐาน

---

## Coding & Safety Rules

### Database & Safety Rules

- ห้ามแก้ Database Schema หรือ Migration เว้นแต่ได้รับคำสั่งชัดเจน ให้พิจารณา Logic ฝั่ง Backend ก่อน
- ห้ามใช้ Logic เพื่อหลบเลี่ยงข้อกำหนดข้อมูลที่ยืนยันแล้ว
- หลีกเลี่ยง Refactor ที่ไม่จำเป็น หากไม่แน่ใจให้ถามก่อนเปลี่ยน Architecture
- ก่อนรัน Test/Script ที่เขียนข้อมูล ต้องยืนยันว่าใช้ฐานข้อมูลทดสอบที่แยกจาก Production และเข้าใจผลกระทบแล้ว
- ห้ามลบ แก้ หรือ Cleanup ข้อมูลจริง รวมถึงบัญชีทดสอบใน DB โดยไม่มีขอบเขตและการอนุมัติชัดเจน

### Backend Rules

- ใช้ RESTful API และ JSON ตาม Contract ที่มีอยู่
- หลีกเลี่ยง SQL ซ้ำซ้อน
- ใช้ DB Transaction เมื่อมีการอัปเดตหลาย Table ที่ต้องสำเร็จหรือล้มเหลวร่วมกัน
- ตรวจ Authorization ในระดับที่เหมาะสม อย่าเชื่อ `user_id` หรือ Owner ID จาก Request โดยไม่ตรวจสิทธิ์
- ใช้ AuditService/Logging ที่มีอยู่ตามรูปแบบของโปรเจกต์เมื่อ Requirement กำหนด ห้ามสร้างกลไกซ้ำโดยไม่จำเป็น

### Frontend Rules

- ใช้ Functional Components และ React Hooks ตามรูปแบบของโปรเจกต์
- แยก Component เมื่อช่วยให้อ่านและดูแลรักษาง่ายขึ้น
- ใช้ CSS Classes (Tailwind/CSS Modules/Custom CSS) ตามโครงสร้างเดิม หลีกเลี่ยง Inline Styles เมื่อไม่จำเป็น
- ห้ามแก้ Frontend ใน Task ที่กำหนดขอบเขตเป็น Backend-only

### Dependency Rules

- ห้ามติดตั้ง Package ใหม่ หากมี Package ที่ใช้งานได้อยู่แล้ว
- ก่อนเพิ่ม Dependency ให้ตรวจ `package.json`, lockfile หรือ `go.mod` ที่เกี่ยวข้อง
- หากจำเป็นต้องเพิ่ม Dependency ต้องอธิบายเหตุผลและรอ Human อนุมัติก่อน

---

## Commands

- Start Infra Stack: `docker compose build && docker compose up -d`
- Run Backend Local: `cd novel-BE && go run ./cmd/main.go` (port 8080)
- Backend test: `cd novel-BE && go test ./...`
- Backend lint: `cd novel-BE && golangci-lint run && gosec ./...`
- Run Frontend Local: `cd novel-FE && npm run dev` (port 5173)
- Frontend check: `cd novel-FE && npm run lint && npm run build`

ก่อนรันคำสั่งที่อาจเปลี่ยนข้อมูล สร้างไฟล์จำนวนมาก หรือกระทบ Environment ให้ตรวจผลกระทบและขออนุมัติตามกฎด้านบน
