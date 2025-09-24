package usecase

import (
	"context"
	"demo/domain"
	"demo/repo"

	"github.com/samber/lo"
)

type RoleUsecase interface {
	ListRoles(ctx context.Context) ([]domain.RoleWithoutPrompt, error)
}

type roleUsecase struct {
	roleRepo *repo.RoleRepo
}

func NewRoleUsecase(roleRepo *repo.RoleRepo) RoleUsecase {
	return &roleUsecase{
		roleRepo: roleRepo,
	}
}

func (u *roleUsecase) ListRoles(ctx context.Context) ([]domain.RoleWithoutPrompt, error) {
	roles, err := u.roleRepo.ListRoles(ctx)
	if err != nil {
		return nil, err
	}

	result := lo.Map(roles, func(role domain.Role, _ int) domain.RoleWithoutPrompt {
		return domain.RoleWithoutPrompt{
			ID:   role.ID,
			Name: role.Name,
		}
	})
	return result, nil
}
