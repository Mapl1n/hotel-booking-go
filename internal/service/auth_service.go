package service

import (
	"context"
	"errors"
	"regexp"
	"time"

	"hotel-booking-go/internal/config"
	"hotel-booking-go/internal/dao"
	"hotel-booking-go/internal/model"
	"hotel-booking-go/pkg/sms"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct {
	userDAO     *dao.UserDAO
	loginLogDAO *dao.LoginLogDAO
	smsService  *sms.Service
	cfg         *config.Config
}

func NewAuthService(userDAO *dao.UserDAO, loginLogDAO *dao.LoginLogDAO, smsService *sms.Service, cfg *config.Config) *AuthService {
	return &AuthService{userDAO: userDAO, loginLogDAO: loginLogDAO, smsService: smsService, cfg: cfg}
}

var (
	ErrPhoneExists      = errors.New("该手机号已注册")
	ErrInvalidLogin     = errors.New("手机号或密码错误")
	ErrAccountDisabled  = errors.New("该账号已被禁用")
	ErrInvalidCode      = errors.New("验证码错误或已过期")
	ErrWeakPassword     = errors.New("密码长度不能少于8位")
	ErrPasswordNoLetter = errors.New("密码必须包含至少一个字母")
	ErrPasswordNoDigit  = errors.New("密码必须包含至少一个数字")
	ErrSmsCooldown      = errors.New("请稍后再试")
)

// ValidatePassword 校验密码强度
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return ErrWeakPassword
	}
	hasLetter, _ := regexp.MatchString(`[a-zA-Z]`, password)
	if !hasLetter {
		return ErrPasswordNoLetter
	}
	hasDigit, _ := regexp.MatchString(`\d`, password)
	if !hasDigit {
		return ErrPasswordNoDigit
	}
	return nil
}

// HashPassword bcrypt 哈希密码
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// VerifyPassword 验证密码
func VerifyPassword(hashed, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain)) == nil
}

// SendCode 发送短信验证码
func (s *AuthService) SendCode(ctx context.Context, phone string) error {
	return s.smsService.SendCode(ctx, phone)
}

// Register 用户注册
func (s *AuthService) Register(ctx context.Context, username, password, code string) error {
	// 校验密码强度
	if err := ValidatePassword(password); err != nil {
		return err
	}
	// 校验验证码
	if !s.smsService.VerifyCode(ctx, username, code) {
		return ErrInvalidCode
	}
	// 检查用户是否已存在
	_, err := s.userDAO.FindByUsername(username)
	if err == nil {
		return ErrPhoneExists
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	// 哈希密码
	hashed, err := HashPassword(password)
	if err != nil {
		return err
	}
	// 创建用户
	user := &model.User{
		Username:     username,
		PasswordHash: hashed,
		Phone:        username,
		Role:         "guest",
		IsActive:     true,
	}
	return s.userDAO.Create(user)
}

// Login 用户登录
func (s *AuthService) Login(username, password, ip string) (string, *model.User, error) {
	user, err := s.userDAO.FindByUsername(username)
	if err != nil {
		return "", nil, ErrInvalidLogin
	}
	if !user.IsActive {
		return "", nil, ErrAccountDisabled
	}
	if !VerifyPassword(user.PasswordHash, password) {
		return "", nil, ErrInvalidLogin
	}
	// 生成 JWT
	token, err := s.generateJWT(user)
	if err != nil {
		return "", nil, err
	}
	// 记录登录日志
	s.loginLogDAO.Create(&model.LoginLog{
		UserID:    user.ID,
		Username:  user.Username,
		IPAddress: ip,
	})
	return token, user, nil
}

func (s *AuthService) generateJWT(user *model.User) (string, error) {
	claims := jwt.MapClaims{
		"sub":      user.Username,
		"role":     user.Role,
		"hotel_id": user.HotelID,
		"user_id":  user.ID,
		"exp":      time.Now().Add(time.Duration(s.cfg.JWTExpireHours) * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.JWTSecretKey))
}

// CreateStaff 创建员工账号
func (s *AuthService) CreateStaff(username, password, role string, hotelID int64, phone string) (*model.User, error) {
	if err := ValidatePassword(password); err != nil {
		return nil, err
	}
	_, err := s.userDAO.FindByUsername(username)
	if err == nil {
		return nil, ErrPhoneExists
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	hashed, err := HashPassword(password)
	if err != nil {
		return nil, err
	}
	if phone == "" {
		phone = username
	}
	user := &model.User{
		Username:     username,
		PasswordHash: hashed,
		Phone:        phone,
		Role:         role,
		HotelID:      &hotelID,
		IsActive:     true,
	}
	if err := s.userDAO.Create(user); err != nil {
		return nil, err
	}
	return user, nil
}

// GetLoginLogs 查询登录日志
func (s *AuthService) GetLoginLogs(userIDs []int64, limit int) ([]model.LoginLog, error) {
	if len(userIDs) > 0 {
		return s.loginLogDAO.ListByUserIDs(userIDs, limit)
	}
	return s.loginLogDAO.ListAll(limit)
}
