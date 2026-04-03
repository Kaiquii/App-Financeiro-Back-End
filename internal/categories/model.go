package categories

type Category struct {
	ID     uint   `json:"id" gorm:"primaryKey"`
	UserID uint   `json:"user_id"`
	Name   string `json:"name"`
}

type CreateCategoryRequest struct {
	Name string `json:"name" binding:"required"`
}
