package repo

import (
	"context"
	"demo/config"
	"demo/domain"
	"demo/pkg/log"
	"demo/pkg/store"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type UserRepo struct {
	log    *log.Logger
	config *config.Config
	db     *store.MySQL
}

func NewUserRepo(log *log.Logger, config *config.Config, db *store.MySQL) *UserRepo {
	return &UserRepo{
		log:    log.WithModule("UserRepo"),
		config: config,
		db:     db,
	}
}

func (r *UserRepo) CreateUser(ctx context.Context, user domain.User) error {
	if _, err := r.GetUserByName(ctx, user.Name); err == nil {
		return fmt.Errorf("user already exists")
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	user.Password = string(hashedPassword)
	return r.db.WithContext(ctx).Create(&user).Error
}

func (r *UserRepo) GetUserByName(ctx context.Context, name string) (domain.User, error) {
	var user domain.User
	if err := r.db.WithContext(ctx).Where("name = ?", name).First(&user).Error; err != nil {
		return domain.User{}, fmt.Errorf("failed to get user by name: %w", err)
	}
	return user, nil
}
func (r *UserRepo) UpDataPassword(ctx context.Context, user domain.User, newPassword string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	user.Password = string(hashedPassword)
	return r.db.WithContext(ctx).Save(&user).Error
}
func (r *UserRepo) VertifyUserPasswordAndGenerateToken(ctx context.Context, user domain.User) (string, error) {
	originPassword := user.Password
	user, err := r.GetUserByName(ctx, user.Name)
	if err != nil {
		return "", fmt.Errorf("failed to get user by name: %w", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(originPassword)); err != nil {
		return "", fmt.Errorf("password is not correct")
	}
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte("secret"))
	return tokenString, err
}
