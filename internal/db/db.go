package db

import (
	"database/sql"
	"time"

	_ "github.com/lib/pq"
	"novel-be/config"
)

// Open เปิด connection ไปยัง Postgres/Supabase พร้อมตั้งค่า connection pool
// ที่ปลอดภัยสำหรับ serverless (Railway) — ถ้าไม่ตั้งค่า default จะเปิด connection
// ไม่จำกัดและไม่รีไซเคิล ทำให้ browser ยิงหลาย request พร้อมกันแล้วท่วม connection
// จน Supabase ปฏิเสธ → handler ทุกตัวตอบ 500 ทั้งที่ endpoint เองไม่มีบั๊ก
func Open(cfg config.Config) (*sql.DB, error) {
	dbConn, err := sql.Open("postgres", cfg.GetConnectionString())
	if err != nil {
		return nil, err
	}

	// จำกัดจำนวน connection ที่เปิดพร้อมกัน กันเกิน connection limit ของ Supabase
	// (Supabase free/pro tier ปกติรับได้ ~60 connections จึงตั้งไว้ต่ำกว่าง่ายๆ)
	dbConn.SetMaxOpenConns(20)

	// เก็บ idle connection ไว้ใช้ซ้ำได้บ้าง แต่ไม่ต้องเก็บเยอะ
	dbConn.SetMaxIdleConns(10)

	// บังคับรีไซเคิล connection เป็นระยะ กัน connection ค้างหลัง failover / network reset
	dbConn.SetConnMaxLifetime(30 * time.Minute)

	// idle connection อย่ายืนเกิน 5 นาที ปิดไปคืน resource กับ Supabase
	dbConn.SetConnMaxIdleTime(5 * time.Minute)

	return dbConn, nil
}

