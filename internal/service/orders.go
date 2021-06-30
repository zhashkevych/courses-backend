package service

import (
	"context"
	"time"

	"github.com/zhashkevych/creatly-backend/internal/domain"
	"github.com/zhashkevych/creatly-backend/internal/repository"
	"github.com/zhashkevych/creatly-backend/pkg/logger"
	"github.com/zhashkevych/creatly-backend/pkg/payment"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type OrdersService struct {
	offersService     Offers
	promoCodesService PromoCodes
	studentsService   Students

	repo            repository.Orders
	paymentProvider payment.Provider

	callbackURL, responseURL string
}

func NewOrdersService(repo repository.Orders, offersService Offers, promoCodesService PromoCodes, studentsService Students, paymentProvider payment.Provider, callbackURL, responseURL string) *OrdersService {
	return &OrdersService{
		repo:              repo,
		offersService:     offersService,
		promoCodesService: promoCodesService,
		studentsService:   studentsService,
		paymentProvider:   paymentProvider,
		callbackURL:       callbackURL,
		responseURL:       responseURL,
	}
}

func (s *OrdersService) Create(ctx context.Context, studentId, offerId, promocodeId primitive.ObjectID) (string, error) { //nolint:funlen
	offer, err := s.offersService.GetById(ctx, offerId)
	if err != nil {
		return "", err
	}

	promocode, err := s.getOrderPromocode(ctx, offer.SchoolID, promocodeId)
	if err != nil {
		return "", err
	}

	student, err := s.studentsService.GetById(ctx, studentId)
	if err != nil {
		return "", err
	}

	orderAmount := s.calculateOrderPrice(offer.Price.Value, promocode)

	id := primitive.NewObjectID()

	paymentLink, err := s.paymentProvider.GeneratePaymentLink(payment.GeneratePaymentLinkInput{
		OrderId:     id.Hex(),
		Amount:      orderAmount,
		Currency:    offer.Price.Currency,
		OrderDesc:   offer.Description, // TODO proper order description
		CallbackURL: s.callbackURL,
		ResponseURL: s.responseURL,
	})
	if err != nil {
		logger.Error("Failed to generate payment link: ", err.Error())

		return "", err
	}

	order := domain.Order{
		ID:       id,
		SchoolId: offer.SchoolID,
		Student: domain.OrderStudentInfo{
			ID:    student.ID,
			Name:  student.Name,
			Email: student.Email,
		},
		Offer: domain.OrderOfferInfo{
			ID:   offer.ID,
			Name: offer.Name,
		},
		Amount:       orderAmount,
		Currency:     offer.Price.Currency,
		CreatedAt:    time.Now(),
		Status:       domain.OrderStatusCreated,
		Transactions: make([]domain.Transaction, 0),
	}

	if !promocode.ID.IsZero() {
		order.Promo = domain.OrderPromoInfo{
			ID:   promocode.ID,
			Code: promocode.Code,
		}
	}

	if err := s.repo.Create(ctx, order); err != nil {
		return "", err
	}

	return paymentLink, nil
}

func (s *OrdersService) AddTransaction(ctx context.Context, id primitive.ObjectID, transaction domain.Transaction) (domain.Order, error) {
	return s.repo.AddTransaction(ctx, id, transaction)
}

func (s *OrdersService) GetBySchool(ctx context.Context, schoolId primitive.ObjectID) ([]domain.Order, error) {
	return s.repo.GetBySchool(ctx, schoolId)
}

func (s *OrdersService) getOrderPromocode(ctx context.Context, schoolId, promocodeId primitive.ObjectID) (domain.PromoCode, error) {
	var (
		promocode domain.PromoCode
		err       error
	)

	if !promocodeId.IsZero() {
		promocode, err = s.promoCodesService.GetById(ctx, schoolId, promocodeId)
		if err != nil {
			return promocode, err
		}

		if promocode.ExpiresAt.Unix() < time.Now().Unix() {
			return promocode, domain.ErrPromocodeExpired
		}
	}

	return promocode, nil
}

func (s *OrdersService) calculateOrderPrice(price uint, promocode domain.PromoCode) uint {
	if promocode.ID.IsZero() {
		return price
	} else {
		return (price * uint(100-promocode.DiscountPercentage)) / 100
	}
}
