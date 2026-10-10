# 🚀 DeployNovels — Production Deployment Guide

**Architecture:**
- **Frontend** → Vercel (Vite build)
- **Backend** → Railway (Docker container)
- **Database** → Supabase (PostgreSQL)
- **Storage** → Supabase Storage (bucket: `image`)

---

## ✅ What Changed

| Area | File | Change |
|------|------|--------|
| BE | `config/config.go` | เพิ่ม `DATABASE_URL`, `MINIO_REGION`, `MINIO_PUBLIC_ENDPOINT`, `ALLOWED_ORIGINS`, อ่าน `PORT` จาก env |
| BE | `internal/repository/media_repo.go` | `NewMinIOMediaRepository` รับ `publicEndpoint` param, สร้าง URL เต็ม |
| BE | `internal/service/media_service.go` | `NovelImagesBucket` = `"image"` |
| BE | `cmd/main.go` | เพิ่ม `Region` ใน minio client, เพิ่ม `/health` endpoint, `EnsureBucketExists` ใช้ `service.NovelImagesBucket` |
| BE | `.env` | เขียนใหม่เป็น production env (Supabase + Railway placeholder) |
| FE | `src/utils/mediaUrl.js` | ปรับ `resolveMediaUrl` ให้แทนทั้ง `minio:9000` / `localhost:9000` / `127.0.0.1:9000` |
| FE | `src/pages/Writer/Chaptermanagerpage/Chaptermanagerpage.jsx` | ลบ hardcode `minio:9000`, ใช้ `resolveMediaUrl` |
| FE | `.env.production` | ชี้ไปที่ Railway API + Supabase Storage |
| FE | `vercel.json` | SPA rewrite + asset caching |
| FE | `.gitignore` | เพิ่ม `.env*` (ยกเว้น `.env.example`) |
| BE | `.gitignore` | มี `.env` อยู่แล้ว |
| BE | `railway.json` | ระบุ Dockerfile builder + healthcheck `/health` |

---

## 📋 Step-by-Step Deployment

### Step 1: Supabase Setup

1. สร้าง project ใหม่ใน [supabase.com](https://supabase.com)
2. Project Ref: `vbwfesraeptqlifxekms` (ถ้าใช้ project เดิม)
3. **Database**:
   - ไปที่ SQL Editor
   - Run `D:\novel-DB\docker\init.sql` ก่อน (สร้างตารางทั้งหมด)
   - Run migration ทั้งหมดจาก `D:\novel-DB\docker\migration_*.sql` ตามลำดับ
   - Run `D:\novel-DB\insert.text` (seed data) ถ้าต้องการ
4. **Storage**:
   - สร้าง bucket ชื่อ `image` (Public bucket)
   - ไปที่ Project Settings → Storage → S3 Access Keys
   - สร้าง access key + secret (เก็บไว้ใช้ใน `.env`)

### Step 2: Backend Deployment (Railway)

1. Push code ขึ้น GitHub (`deploynovels-BE`)
2. ไปที่ [railway.app](https://railway.app)
3. New Project → Deploy from GitHub → เลือก `deploynovels-BE`
4. Railway จะอ่าน `railway.json` และใช้ Dockerfile อัตโนมัติ
5. **Environment Variables**:
   ```
   APP_PORT=8080
   DATABASE_URL=postgresql://postgres.vbwfesraeptqlifxekms:<YOUR_PASSWORD>@aws-1-ap-northeast-2.pooler.supabase.com:5432/postgres
   MINIO_ENDPOINT=https://vbwfesraeptqlifxekms.storage.supabase.co/storage/v1/s3
   MINIO_REGION=ap-northeast-2
   MINIO_PUBLIC_ENDPOINT=https://vbwfesraeptqlifxekms.storage.supabase.co/storage/v1/object/public
   MINIO_ACCESS_KEY=<SUPABASE_S3_ACCESS_KEY>
   MINIO_SECRET_KEY=<SUPABASE_S3_SECRET_KEY>
   MINIO_USE_SSL=true
   ALLOWED_ORIGINS=https://*.vercel.app,https://your-app.vercel.app
   JWT_SECRET=<GENERATE_NEW>
   JWT_REFRESH_SECRET=<GENERATE_NEW>
   ```
   - สร้าง JWT secret: `openssl rand -base64 32`
6. Railway จะ deploy container ที่ `https://deploynovels-production.up.railway.app`

### Step 3: Frontend Deployment (Vercel)

1. Push code ขึ้น GitHub (`deploynovels-main`)
2. ไปที่ [vercel.com](https://vercel.com)
3. New Project → Import `deploynovels-main`
4. **Environment Variables** (Production):
   ```
   VITE_API_BASE_URL=https://deploynovels-production.up.railway.app
   VITE_MEDIA_INTERNAL_HOST=http://minio:9000
   VITE_MEDIA_PUBLIC_HOST=https://vbwfesraeptqlifxekms.storage.supabase.co/storage/v1/object/public
   ```
   > ⚠️ มีแค่ 3 ตัวนี้เท่านั้นที่ frontend อ่านจริง (ผ่าน `import.meta.env`)
   > ~~VITE_IMAGE_BASE_URL~~ **ไม่ต้องตั้ง** — โค้ดไม่มีอ่านตัวนี้เลย (รูปถูกแปลงผ่าน `resolveMediaUrl()` ใน `src/utils/mediaUrl.js`)
5. Build Command: `npm run build`
6. Output Directory: `dist`
7. Deploy → Vercel จะสร้าง URL เช่น `https://deploynovels.vercel.app`

### Step 4: Update CORS (ถ้าจำเป็น)

ถ้า Vercel สร้าง URL แบบ `https://deploynovels-xxx.vercel.app` ให้อัปเดต `ALLOWED_ORIGINS` ใน Railway:
```
ALLOWED_ORIGINS=https://*.vercel.app,https://deploynovels-xxx.vercel.app
```

---

## 🔧 Local Development

### Backend
```bash
cd deploynovels-BE
cp .env.example .env  # แก้ค่าให้ตรงกับ local setup
go run ./cmd/main.go
```

### Frontend
```bash
cd deploynovels-main
npm install
npm run dev
```

`.env.local` สำหรับ local dev:
```
VITE_API_BASE_URL=http://localhost:8080
VITE_MEDIA_INTERNAL_HOST=http://minio:9000
VITE_MEDIA_PUBLIC_HOST=http://localhost:9000
```

---

## 🔒 Security Notes

- ❌ **ห้าม commit `.env`** — Railway/Vercel จะ inject env ตอน build/runtime
- ✅ JWT secrets ต้องสร้างใหม่สำหรับ production
- ✅ Supabase RLS policies ต้องตั้งให้ถูกต้อง (ถ้าเปิดใช้)
- ✅ Railway PORT จะ inject อัตโนมัติ (8080 default)

---

## 🐛 Troubleshooting

### รูปไม่ขึ้น (404)
- เช็คว่า bucket `image` เป็น Public bucket หรือเปล่า (Storage → `image` → toggle Public = ON)
- เช็คว่า `MINIO_PUBLIC_ENDPOINT` (BE) ตรงกับ `VITE_MEDIA_PUBLIC_HOST` (FE)
  - ทั้งคู่ต้องเป็น `https://vbwfesraeptqlifxekms.storage.supabase.co/storage/v1/object/public`
  - ⚠️ `MINIO_PUBLIC_ENDPOINT` **ห้าม** มี `/image` ท้าย (รหัสจะเติม bucket เอง)

### CORS error
- เช็คว่า `ALLOWED_ORIGINS` มี domain ของ Vercel ถูกใส่เข้าไป
- ลองเพิ่ม wildcard: `https://*.vercel.app`

### Database connection fail
- เช็คว่า `DATABASE_URL` ใช้ Session Pooler (IPv4 compatible)
- เช็คว่า password ถูก URL-encoded (ถ้ามี special chars)

### Railway deploy fail
- เช็คว่า Dockerfile build ผ่าน: `docker build -t test .`
- เช็คว่า `go.mod` ไม่มี error: `go mod verify`
