package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/zhashkevych/creatly-backend/internal/domain"
	"github.com/zhashkevych/creatly-backend/internal/repository"
	"github.com/zhashkevych/creatly-backend/pkg/auth"
	"github.com/zhashkevych/creatly-backend/pkg/dns"
	"github.com/zhashkevych/creatly-backend/pkg/hash"
	"github.com/zhashkevych/creatly-backend/pkg/otp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UsersService struct {
	repo         repository.Users
	hasher       hash.PasswordHasher
	tokenManager auth.TokenManager
	otpGenerator otp.Generator
	dnsService   dns.DomainManager

	emailService  Emails
	schoolService Schools

	accessTokenTTL         time.Duration
	refreshTokenTTL        time.Duration
	verificationCodeLength int

	domain string
}

func NewUsersService(repo repository.Users, hasher hash.PasswordHasher, tokenManager auth.TokenManager,
	emailService Emails, schoolsService Schools, dnsService dns.DomainManager, accessTTL, refreshTTL time.Duration, otpGenerator otp.Generator,
	verificationCodeLength int, domain string) *UsersService {
	return &UsersService{
		repo:                   repo,
		hasher:                 hasher,
		emailService:           emailService,
		schoolService:          schoolsService,
		tokenManager:           tokenManager,
		accessTokenTTL:         accessTTL,
		refreshTokenTTL:        refreshTTL,
		otpGenerator:           otpGenerator,
		verificationCodeLength: verificationCodeLength,
		dnsService:             dnsService,
		domain:                 domain,
	}
}

func (s *UsersService) SignUp(ctx context.Context, input UserSignUpInput) error {
	passwordHash, err := s.hasher.Hash(input.Password)
	if err != nil {
		return err
	}

	verificationCode := s.otpGenerator.RandomSecret(s.verificationCodeLength)

	user := domain.User{
		Name:         input.Name,
		Password:     passwordHash,
		Phone:        input.Phone,
		Email:        input.Email,
		RegisteredAt: time.Now(),
		LastVisitAt:  time.Now(),
		Verification: domain.Verification{
			Code: verificationCode,
		},
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return err
	}

	// todo. DECIDE ON EMAIL MARKETING STRATEGY

	return s.emailService.SendUserVerificationEmail(VerificationEmailInput{
		Email:            user.Email,
		Name:             user.Name,
		VerificationCode: verificationCode,
	})
}

func (s *UsersService) SignIn(ctx context.Context, input UserSignInInput) (Tokens, error) {
	passwordHash, err := s.hasher.Hash(input.Password)
	if err != nil {
		return Tokens{}, err
	}

	user, err := s.repo.GetByCredentials(ctx, input.Email, passwordHash)
	if err != nil {
		return Tokens{}, err
	}

	return s.createSession(ctx, user.ID)
}

func (s *UsersService) RefreshTokens(ctx context.Context, refreshToken string) (Tokens, error) {
	student, err := s.repo.GetByRefreshToken(ctx, refreshToken)
	if err != nil {
		return Tokens{}, err
	}

	return s.createSession(ctx, student.ID)
}

func (s *UsersService) Verify(ctx context.Context, userId primitive.ObjectID, hash string) error {
	err := s.repo.Verify(ctx, userId, hash)
	if err != nil {
		return err
	}

	return nil
}

func (s *UsersService) CreateSchool(ctx context.Context, userId primitive.ObjectID, schoolName string) (domain.School, error) {
	schoolId, err := s.schoolService.Create(ctx, schoolName)
	if err != nil {
		return domain.School{}, err
	}

	if err := s.repo.AttachSchool(ctx, userId, schoolId); err != nil {
		return domain.School{}, err
	}

	subdomain := generateSubdomain(schoolName)
	if err := s.dnsService.AddCNAMERecord(ctx, subdomain); err != nil {
		return domain.School{}, err
	}

	schoolDomain := s.generateSchoolDomain(subdomain)

	if err := s.schoolService.UpdateSettings(ctx, schoolId, UpdateSchoolSettingsInput{
		Domains: []string{schoolDomain},
	}); err != nil {
		return domain.School{}, err
	}

	// todo create new admin

	// todo send email with info

	// Return school
	return domain.School{ID: schoolId, Settings: domain.Settings{Domains: []string{schoolDomain}}}, nil
}

func (s *UsersService) createSession(ctx context.Context, userId primitive.ObjectID) (Tokens, error) {
	var (
		res Tokens
		err error
	)

	res.AccessToken, err = s.tokenManager.NewJWT(userId.Hex(), s.accessTokenTTL)
	if err != nil {
		return res, err
	}

	res.RefreshToken, err = s.tokenManager.NewRefreshToken()
	if err != nil {
		return res, err
	}

	session := domain.Session{
		RefreshToken: res.RefreshToken,
		ExpiresAt:    time.Now().Add(s.refreshTokenTTL),
	}

	err = s.repo.SetSession(ctx, userId, session)

	return res, err
}

func (s *UsersService) generateSchoolDomain(subdomain string) string {
	return fmt.Sprintf("%s.%s", subdomain, s.domain)
}

// input: Example School Name -> output: example-school-name
func generateSubdomain(schoolName string) string {
	var subdomain string

	parts := strings.Split(schoolName, " ")
	for i, part := range parts {
		if i == len(parts)-1 {
			subdomain += strings.ToLower(part)

			break
		}

		subdomain += strings.ToLower(part) + "-"
	}

	return subdomain
}
