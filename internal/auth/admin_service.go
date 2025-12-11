package auth

import (
	"errors"
	"whistleblower_REST/internal/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AdminAuthService struct {
	db *gorm.DB
}

func NewAdminAuthService(db *gorm.DB) *AdminAuthService {
	return &AdminAuthService{db: db}
}

// InitializeSuperAdmin membuat akun superadmin default jika belum ada
func (s *AdminAuthService) InitializeSuperAdmin() error {
	var count int64
	s.db.Model(&models.Admin{}).Where("role = ?", "superadmin").Count(&count)

	if count == 0 {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		superadmin := models.Admin{
			FullName:   "Super Administrator",
			Email:      "superadmin@system.com",
			Password:   string(hashedPassword),
			Department: "IT Management",
			Role:       "superadmin",
			IsActive:   true,
		}

		return s.db.Create(&superadmin).Error
	}
	return nil
}

// AdminLogin untuk login admin dan superadmin
func (s *AdminAuthService) AdminLogin(email, password string) (string, *models.Admin, error) {
	var admin models.Admin

	// Cari admin by email
	if err := s.db.Where("email = ? AND is_active = ?", email, true).First(&admin).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", nil, errors.New("invalid credentials")
		}
		return "", nil, err
	}

	// Pastikan role adalah admin atau superadmin (investigator tidak bisa login)
	if admin.Role != "admin" && admin.Role != "superadmin" {
		return "", nil, errors.New("unauthorized role")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(password)); err != nil {
		return "", nil, errors.New("invalid credentials")
	}

	// Generate JWT token
	token, err := GenerateTokenForAdmin(admin.ID, admin.Role)
	if err != nil {
		return "", nil, err
	}

	return token, &admin, nil
}

// CreateAdmin - hanya superadmin yang bisa membuat admin baru
func (s *AdminAuthService) CreateAdmin(fullName, email, password, department string) error {
	// Cek apakah email sudah digunakan
	var count int64
	s.db.Model(&models.Admin{}).Where("email = ?", email).Count(&count)
	if count > 0 {
		return errors.New("email already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	admin := models.Admin{
		FullName:   fullName,
		Email:      email,
		Password:   string(hashedPassword),
		Department: department,
		Role:       "admin",
		IsActive:   true,
	}

	return s.db.Create(&admin).Error
}

// CreateInvestigator - superadmin dan admin bisa membuat investigator (tanpa password)
func (s *AdminAuthService) CreateInvestigator(fullName, email, department string) error {
	// Cek apakah email sudah digunakan
	var count int64
	s.db.Model(&models.Admin{}).Where("email = ?", email).Count(&count)
	if count > 0 {
		return errors.New("email already exists")
	}

	investigator := models.Admin{
		FullName:   fullName,
		Email:      email,
		Password:   "", // Investigator tidak perlu password
		Department: department,
		Role:       "investigator",
		IsActive:   true,
	}

	return s.db.Create(&investigator).Error
}

// UpdateAdmin - update data admin/investigator
func (s *AdminAuthService) UpdateAdmin(id uint, fullName, email, department string, isActive bool) error {
	updates := map[string]interface{}{
		"full_name":  fullName,
		"email":      email,
		"department": department,
		"is_active":  isActive,
	}

	return s.db.Model(&models.Admin{}).Where("id = ?", id).Updates(updates).Error
}

// ChangeAdminPassword - ubah password admin
func (s *AdminAuthService) ChangeAdminPassword(id uint, newPassword string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return s.db.Model(&models.Admin{}).Where("id = ?", id).Update("password", string(hashedPassword)).Error
}

// DeleteAdmin - soft delete dengan set is_active = false
func (s *AdminAuthService) DeleteAdmin(id uint) error {
	// Tidak boleh hapus superadmin
	var admin models.Admin
	if err := s.db.First(&admin, id).Error; err != nil {
		return err
	}

	if admin.Role == "superadmin" {
		return errors.New("cannot delete superadmin")
	}

	return s.db.Model(&models.Admin{}).Where("id = ?", id).Update("is_active", false).Error
}

// GetAllAdmins - ambil semua admin dan investigator
func (s *AdminAuthService) GetAllAdmins() ([]models.Admin, error) {
	var admins []models.Admin
	err := s.db.Where("is_active = ?", true).Order("created_at DESC").Find(&admins).Error
	return admins, err
}

// GetAdminByID - ambil admin by ID
func (s *AdminAuthService) GetAdminByID(id uint) (*models.Admin, error) {
	var admin models.Admin
	err := s.db.Where("id = ? AND is_active = ?", id, true).First(&admin).Error
	if err != nil {
		return nil, err
	}
	return &admin, nil
}

func (s *AdminAuthService) GetAllInvestigators() ([]models.Admin, error) {
	var investigators []models.Admin
	if err := s.db.Where("role = ?", "investigator").Find(&investigators).Error; err != nil {
		return nil, err
	}
	return investigators, nil
}
