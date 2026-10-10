package models

// Category โครงสร้างข้อมูลสำหรับตาราง categories
type Category struct {
	CategoryID int    `json:"category_id"`
	Name       string `json:"name"`
}
