package repo

import (
	"context"
	"demo/config"
	"demo/domain"
	"demo/pkg/log"
	"demo/pkg/store"
	"fmt"

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
func (r *UserRepo) vertifyUserPassword(ctx context.Context, user domain.User, password string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return fmt.Errorf("failed to vertify password: %w", err)
	}
	return nil
}
