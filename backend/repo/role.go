package repo

import (
	"context"
	"demo/config"
	"demo/domain"
	"demo/pkg/log"
	"demo/pkg/store"
	"fmt"
)

type RoleRepo struct {
	log    *log.Logger
	config *config.Config
	db     *store.MySQL
}

func NewRoleRepo(log *log.Logger, config *config.Config, db *store.MySQL) *RoleRepo {
	return &RoleRepo{
		log:    log.WithModule("RoleRepo"),
		config: config,
		db:     db,
	}
}

func (r *RoleRepo) CreateRole(ctx context.Context, role domain.Role) error {
	if _, err := r.GetRoleByName(ctx, role.Name); err == nil {
		return fmt.Errorf("role already exists")
	}
	return r.db.WithContext(ctx).Create(&role).Error
}

func (r *RoleRepo) GetRoleByName(ctx context.Context, name string) (domain.Role, error) {
	var role domain.Role
	if err := r.db.WithContext(ctx).Where("name = ?", name).First(&role).Error; err != nil {
		return domain.Role{}, fmt.Errorf("failed to get role by name: %w", err)
	}
	return role, nil
}

func (r *RoleRepo) UpdateRole(ctx context.Context, role domain.Role) error {
	return r.db.WithContext(ctx).Save(&role).Error
}

func (r *RoleRepo) DeleteRole(ctx context.Context, id int) error {
	return r.db.WithContext(ctx).Delete(&domain.Role{}, id).Error
}

func (r *RoleRepo) ListRoles(ctx context.Context) ([]domain.Role, error) {
	var roles []domain.Role
	if err := r.db.WithContext(ctx).Find(&roles).Error; err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}
	return roles, nil
}
