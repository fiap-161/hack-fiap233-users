package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailTaken         = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound       = errors.New("user not found")
	// ErrDuplicateEmail is returned by UserRepository when a unique constraint fires.
	ErrDuplicateEmail = errors.New("duplicate email")
)

type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type AuthOutput struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// UserRepository is defined here (by the consumer) following Go's interface convention.
type UserRepository interface {
	Create(ctx context.Context, name, email, passwordHash string) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, string, error)
	FindByID(ctx context.Context, id int) (*User, error)
	List(ctx context.Context) ([]*User, error)
	Ping(ctx context.Context) error
}

type UserService interface {
	Register(ctx context.Context, name, email, password string) (*AuthOutput, error)
	Login(ctx context.Context, email, password string) (*AuthOutput, error)
	GetByID(ctx context.Context, id int) (*User, error)
	ListUsers(ctx context.Context, userID int) ([]*User, error)
	ListAllUsers(ctx context.Context) ([]*User, error)
	Health(ctx context.Context) error
}

type userService struct {
	repo      UserRepository
	jwtSecret []byte
}

type UserServiceBuilder struct {
	service *userService
}

func New() *UserServiceBuilder {
	return &UserServiceBuilder{service: &userService{}}
}

func (b *UserServiceBuilder) WithRepository(repo UserRepository) *UserServiceBuilder {
	b.service.repo = repo
	return b
}

func (b *UserServiceBuilder) WithJWTSecret(secret string) *UserServiceBuilder {
	b.service.jwtSecret = []byte(secret)
	return b
}

func (b *UserServiceBuilder) Build() UserService {
	return b.service
}

func (s *userService) Register(ctx context.Context, name, email, password string) (*AuthOutput, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	user, err := s.repo.Create(ctx, name, email, string(hash))
	if err != nil {
		if errors.Is(err, ErrDuplicateEmail) {
			return nil, ErrEmailTaken
		}
		return nil, fmt.Errorf("creating user: %w", err)
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, fmt.Errorf("generating token: %w", err)
	}

	return &AuthOutput{Token: token, User: *user}, nil
}

func (s *userService) Login(ctx context.Context, email, password string) (*AuthOutput, error) {
	user, hash, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("finding user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, fmt.Errorf("generating token: %w", err)
	}

	return &AuthOutput{Token: token, User: *user}, nil
}

func (s *userService) GetByID(ctx context.Context, id int) (*User, error) {
	u, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return u, nil
}

// ListUsers retorna apenas o usuário com o ID informado (listagem segura: um único usuário).
// Em produção o API Gateway injeta X-User-Id; o handler passa esse ID.
func (s *userService) ListUsers(ctx context.Context, userID int) ([]*User, error) {
	u, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return []*User{u}, nil
}

// ListAllUsers retorna todos os usuários. Deve ser usada apenas em rotas protegidas (usuário logado).
func (s *userService) ListAllUsers(ctx context.Context) ([]*User, error) {
	return s.repo.List(ctx)
}

func (s *userService) Health(ctx context.Context) error {
	return s.repo.Ping(ctx)
}

func (s *userService) generateToken(user *User) (string, error) {
	claims := jwt.MapClaims{
		"sub":   fmt.Sprintf("%d", user.ID),
		"email": user.Email,
		"exp":   time.Now().Add(24 * time.Hour).Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
}
