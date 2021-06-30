package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zhashkevych/creatly-backend/internal/domain"
	"github.com/zhashkevych/creatly-backend/pkg/email"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	verificationCode = "CODE1234"
)

func (s *APITestSuite) TestStudentSignUp() {
	router := gin.New()
	s.handler.Init(router.Group("/api"))

	r := s.Require()

	name, studentEmail, password := "Test Student", "test@test.com", "qwerty123"
	signUpData := fmt.Sprintf(`{"name":"%s","email":"%s","password":"%s"}`, name, studentEmail, password)

	s.mocks.otpGenerator.On("RandomSecret", 8).Return(verificationCode)
	s.mocks.emailProvider.On("AddEmailToList", email.AddEmailInput{
		Email:  studentEmail,
		ListID: listId,
		Variables: map[string]string{
			"name": name,
		},
	}).Return(nil)
	s.mocks.emailSender.On("Send", email.SendEmailInput{
		To:      studentEmail,
		Subject: "Спасибо за регистрацию, Test Student!",
		Body: fmt.Sprintf(`<h1>Спасибо за регистрацию!</h1>
<br>
<p>Чтобы подтвердить свой аккаунт, <a href="https://workshop.zhashkevych.com/verification?code=%s">переходи по ссылке</a>.</p>`, verificationCode),
	}).Return(nil)

	req, _ := http.NewRequest("POST", "/api/v1/students/sign-up", bytes.NewBuffer([]byte(signUpData)))
	req.Header.Set("Content-type", "application/json")
	req.Header.Set("Referer", "https://workshop.zhashkevych.com/")

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	r.Equal(http.StatusCreated, resp.Result().StatusCode)

	var student domain.Student
	err := s.db.Collection("students").FindOne(context.Background(), bson.M{"email": studentEmail}).Decode(&student)
	s.NoError(err)

	passwordHash, err := s.hasher.Hash(password)
	s.NoError(err)

	r.Equal(name, student.Name)
	r.Equal(passwordHash, student.Password)
	r.Equal(false, student.Verification.Verified)
	r.Equal(verificationCode, student.Verification.Code)
}

func (s *APITestSuite) TestStudentSignInNotVerified() {
	router := gin.New()
	s.handler.Init(router.Group("/api"))
	r := s.Require()

	// populate DB data
	studentEmail, password := "test2@test.com", "qwerty123"
	passwordHash, err := s.hasher.Hash(password)
	s.NoError(err)

	_, err = s.db.Collection("students").InsertOne(context.Background(), domain.Student{
		Email:    studentEmail,
		Password: passwordHash,
	})
	s.NoError(err)

	signUpData := fmt.Sprintf(`{"email":"%s","password":"%s"}`, studentEmail, password)
	req, _ := http.NewRequest("POST", "/api/v1/students/sign-in", bytes.NewBuffer([]byte(signUpData)))
	req.Header.Set("Content-type", "application/json")

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	r.Equal(http.StatusBadRequest, resp.Result().StatusCode)
}

func (s *APITestSuite) TestStudentVerify() {
	router := gin.New()
	s.handler.Init(router.Group("/api"))
	r := s.Require()

	// populate DB data
	studentEmail, password := "test3@test.com", "qwerty123"
	passwordHash, err := s.hasher.Hash(password)
	s.NoError(err)

	_, err = s.db.Collection("students").InsertOne(context.Background(), domain.Student{
		Email:        studentEmail,
		Password:     passwordHash,
		Verification: domain.Verification{Code: "CODE4321"},
	})
	s.NoError(err)

	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/students/verify/%s", "CODE4321"), nil)
	req.Header.Set("Content-type", "application/json")

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	r.Equal(http.StatusOK, resp.Result().StatusCode)

	var student domain.Student
	err = s.db.Collection("students").FindOne(context.Background(), bson.M{"email": studentEmail}).Decode(&student)
	s.NoError(err)

	r.Equal(true, student.Verification.Verified)
	r.Equal("", student.Verification.Code)
}

func (s *APITestSuite) TestStudentSignInVerified() {
	router := gin.New()
	s.handler.Init(router.Group("/api"))
	r := s.Require()

	// populate DB data
	studentEmail, password := "test4@test.com", "qwerty123"
	passwordHash, err := s.hasher.Hash(password)
	s.NoError(err)

	_, err = s.db.Collection("students").InsertOne(context.Background(), domain.Student{
		Email:        studentEmail,
		Password:     passwordHash,
		SchoolID:     school.ID,
		Verification: domain.Verification{Verified: true},
	})
	s.NoError(err)

	signUpData := fmt.Sprintf(`{"email":"%s","password":"%s"}`, studentEmail, password)

	req, _ := http.NewRequest("POST", "/api/v1/students/sign-in", bytes.NewBuffer([]byte(signUpData)))
	req.Header.Set("Content-type", "application/json")

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	r.Equal(http.StatusOK, resp.Result().StatusCode)
}

func (s *APITestSuite) TestStudentGetPaidLessonsWithoutPurchase() {
	router := gin.New()
	s.handler.Init(router.Group("/api"))
	r := s.Require()

	// populate DB data
	id := primitive.NewObjectID()
	studentEmail, password := "test4@test.com", "qwerty123"
	passwordHash, err := s.hasher.Hash(password)
	s.NoError(err)

	_, err = s.db.Collection("students").InsertOne(context.Background(), domain.Student{
		ID:           id,
		Email:        studentEmail,
		Password:     passwordHash,
		SchoolID:     school.ID,
		Verification: domain.Verification{Verified: true},
	})
	s.NoError(err)

	jwt, err := s.getJwt(id)
	s.NoError(err)

	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/students/modules/%s/lessons", modules[1].(domain.Module).ID.Hex()), nil)
	req.Header.Set("Content-type", "application/json")
	req.Header.Set("Authorization", "Bearer "+jwt)

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	r.Equal(http.StatusForbidden, resp.Result().StatusCode)
}

func (s *APITestSuite) TestStudentGetModuleOffers() {
	router := gin.New()
	s.handler.Init(router.Group("/api"))
	r := s.Require()

	// populate DB data
	id := primitive.NewObjectID()
	studentEmail, password := "test4@test.com", "qwerty123"
	passwordHash, err := s.hasher.Hash(password)
	s.NoError(err)

	_, err = s.db.Collection("students").InsertOne(context.Background(), domain.Student{
		ID:           id,
		Email:        studentEmail,
		Password:     passwordHash,
		SchoolID:     school.ID,
		Verification: domain.Verification{Verified: true},
	})
	s.NoError(err)

	jwt, err := s.getJwt(id)
	s.NoError(err)

	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/students/modules/%s/offers", modules[1].(domain.Module).ID.Hex()), nil)
	req.Header.Set("Content-type", "application/json")
	req.Header.Set("Authorization", "Bearer "+jwt)

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	r.Equal(http.StatusOK, resp.Result().StatusCode)

	var respOffers struct {
		Data []offerResponse `json:"data"`
	}

	respData, err := ioutil.ReadAll(resp.Body)
	s.NoError(err)

	err = json.Unmarshal(respData, &respOffers)
	s.NoError(err)

	r.Equal(1, len(respOffers.Data))
	r.Equal(offers[0].(domain.Offer).Name, respOffers.Data[0].Name)
	r.Equal(offers[0].(domain.Offer).Description, respOffers.Data[0].Description)
	r.Equal(offers[0].(domain.Offer).Price.Value, respOffers.Data[0].Price.Value)
	r.Equal(offers[0].(domain.Offer).Price.Currency, respOffers.Data[0].Price.Currency)
}

func (s *APITestSuite) TestStudentCreateOrderWithoutPromocode() {
	router := gin.New()
	s.handler.Init(router.Group("/api"))
	r := s.Require()

	// populate DB data
	id := primitive.NewObjectID()
	studentEmail, password := "test4@test.com", "qwerty123"
	passwordHash, err := s.hasher.Hash(password)
	s.NoError(err)

	_, err = s.db.Collection("students").InsertOne(context.Background(), domain.Student{
		ID:           id,
		Email:        studentEmail,
		Password:     passwordHash,
		SchoolID:     school.ID,
		Verification: domain.Verification{Verified: true},
	})
	s.NoError(err)

	jwt, err := s.getJwt(id)
	s.NoError(err)

	orderData := fmt.Sprintf(`{"offerId":"%s"}`, offers[0].(domain.Offer).ID.Hex())

	req, _ := http.NewRequest("POST", "/api/v1/students/order", bytes.NewBuffer([]byte(orderData)))
	req.Header.Set("Content-type", "application/json")
	req.Header.Set("Authorization", "Bearer "+jwt)

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	r.Equal(http.StatusOK, resp.Result().StatusCode)

	var order domain.Order
	err = s.db.Collection("orders").FindOne(context.Background(), bson.M{
		"student.id": id,
	}).Decode(&order)
	s.NoError(err)

	r.Equal(offers[0].(domain.Offer).Price.Value, order.Amount)
	r.Equal(offers[0].(domain.Offer).Price.Currency, order.Currency)
}

func (s *APITestSuite) TestStudentCreateOrderWrongOffer() {
	router := gin.New()
	s.handler.Init(router.Group("/api"))
	r := s.Require()

	// populate DB data
	id := primitive.NewObjectID()
	studentEmail, password := "test4@test.com", "qwerty123"
	passwordHash, err := s.hasher.Hash(password)
	s.NoError(err)

	_, err = s.db.Collection("students").InsertOne(context.Background(), domain.Student{
		ID:           id,
		Email:        studentEmail,
		Password:     passwordHash,
		SchoolID:     school.ID,
		Verification: domain.Verification{Verified: true},
	})
	s.NoError(err)

	jwt, err := s.getJwt(id)
	s.NoError(err)

	orderData := fmt.Sprintf(`{"offerId":"%s"}`, id.Hex())

	req, _ := http.NewRequest("POST", "/api/v1/students/order", bytes.NewBuffer([]byte(orderData)))
	req.Header.Set("Content-type", "application/json")
	req.Header.Set("Authorization", "Bearer "+jwt)

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	r.Equal(http.StatusBadRequest, resp.Result().StatusCode)
}

func (s *APITestSuite) TestStudentCreateOrderWithPromocode() {
	router := gin.New()
	s.handler.Init(router.Group("/api"))
	r := s.Require()

	// populate DB data
	id := primitive.NewObjectID()
	studentEmail, password := "test4@test.com", "qwerty123"
	passwordHash, err := s.hasher.Hash(password)
	s.NoError(err)

	_, err = s.db.Collection("students").InsertOne(context.Background(), domain.Student{
		ID:           id,
		Email:        studentEmail,
		Password:     passwordHash,
		SchoolID:     school.ID,
		Verification: domain.Verification{Verified: true},
	})
	s.NoError(err)

	jwt, err := s.getJwt(id)
	s.NoError(err)

	orderData := fmt.Sprintf(`{"offerId":"%s", "promoId": "%s"}`,
		offers[0].(domain.Offer).ID.Hex(), promocodes[0].(domain.PromoCode).ID.Hex())

	req, _ := http.NewRequest("POST", "/api/v1/students/order", bytes.NewBuffer([]byte(orderData)))
	req.Header.Set("Content-type", "application/json")
	req.Header.Set("Authorization", "Bearer "+jwt)

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	r.Equal(http.StatusOK, resp.Result().StatusCode)

	var order domain.Order
	err = s.db.Collection("orders").FindOne(context.Background(), bson.M{
		"student.id": id,
	}).Decode(&order)
	s.NoError(err)

	offerPrice := offers[0].(domain.Offer).Price.Value
	promocodeDiscount := promocodes[0].(domain.PromoCode).DiscountPercentage
	orderPrice := (offerPrice * uint(100-promocodeDiscount)) / 100

	r.Equal(orderPrice, order.Amount)
	r.Equal(offers[0].(domain.Offer).Price.Currency, order.Currency)
}

func (s *APITestSuite) TestStudentCreateOrderWrongPromo() {
	router := gin.New()
	s.handler.Init(router.Group("/api"))
	r := s.Require()

	// populate DB data
	id := primitive.NewObjectID()
	studentEmail, password := "test4@test.com", "qwerty123"
	passwordHash, err := s.hasher.Hash(password)
	s.NoError(err)

	_, err = s.db.Collection("students").InsertOne(context.Background(), domain.Student{
		ID:           id,
		Email:        studentEmail,
		Password:     passwordHash,
		SchoolID:     school.ID,
		Verification: domain.Verification{Verified: true},
	})
	s.NoError(err)

	jwt, err := s.getJwt(id)
	s.NoError(err)

	orderData := fmt.Sprintf(`{"offerId":"%s", "promoId": "%s"}`,
		offers[0].(domain.Offer).ID.Hex(), id.Hex())

	req, _ := http.NewRequest("POST", "/api/v1/students/order", bytes.NewBuffer([]byte(orderData)))
	req.Header.Set("Content-type", "application/json")
	req.Header.Set("Authorization", "Bearer "+jwt)

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	r.Equal(http.StatusBadRequest, resp.Result().StatusCode)
}

func (s *APITestSuite) getJwt(userId primitive.ObjectID) (string, error) {
	return s.tokenManager.NewJWT(userId.Hex(), time.Hour)
}
