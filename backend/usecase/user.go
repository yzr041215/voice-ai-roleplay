package usecase

import (
	"context"
	"demo/domain"
	"demo/pkg/log"
	"demo/repo"
)

type UserUsecase struct {
	l        *log.Logger
	userrepo *repo.UserRepo
}

func NewUserUsecase(l *log.Logger, userrepo *repo.UserRepo) *UserUsecase {
	return &UserUsecase{
		l:        l.WithModule("UserUsecase"),
		userrepo: userrepo,
	}
}

func (u *UserUsecase) Register(ctx context.Context, req *domain.CreateUserReq) error {
	return u.userrepo.CreateUser(ctx, domain.User{Name: req.Name, Password: req.Password})
}

func (u *UserUsecase) Login(ctx context.Context, req *domain.LoginReq) (*domain.LoginResp, error) {
	token, err := u.userrepo.VertifyUserPasswordAndGenerateToken(ctx, domain.User{Name: req.Name, Password: req.Password})
	if err != nil {
		return nil, err
	}
	return &domain.LoginResp{Token: token}, nil
}
