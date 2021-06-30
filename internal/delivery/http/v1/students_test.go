package v1

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/assert/v2"
	"github.com/golang/mock/gomock"
	"github.com/zhashkevych/creatly-backend/internal/domain"
	"github.com/zhashkevych/creatly-backend/internal/service"
	mock_service "github.com/zhashkevych/creatly-backend/internal/service/mocks"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	paymentLink = "http://payment.link/"
)

func TestHandler_studentCreateOrder(t *testing.T) {
	type mockBehavior func(r *mock_service.MockOrders, studentId, offerId, promoId primitive.ObjectID)

	studentId := primitive.NewObjectID()
	offerId := primitive.NewObjectID()
	promoId := primitive.NewObjectID()

	tests := []struct {
		name         string
		body         string
		studentId    primitive.ObjectID
		offerId      primitive.ObjectID
		promoId      primitive.ObjectID
		mockBehavior mockBehavior
		statusCode   int
		responseBody string
	}{
		{
			name:      "ok",
			body:      fmt.Sprintf(`{"offerId": "%s"}`, offerId.Hex()),
			studentId: studentId,
			offerId:   offerId,
			mockBehavior: func(r *mock_service.MockOrders, studentId, offerId, promoId primitive.ObjectID) {
				r.EXPECT().Create(context.Background(), studentId, offerId, promoId).Return(paymentLink, nil)
			},
			statusCode:   200,
			responseBody: fmt.Sprintf(`{"paymentLink":"%s"}`, paymentLink),
		},
		{
			name:      "ok w/ promocode",
			body:      fmt.Sprintf(`{"offerId": "%s", "promoId": "%s"}`, offerId.Hex(), promoId.Hex()),
			studentId: studentId,
			offerId:   offerId,
			promoId:   promoId,
			mockBehavior: func(r *mock_service.MockOrders, studentId, offerId, promoId primitive.ObjectID) {
				r.EXPECT().Create(context.Background(), studentId, offerId, promoId).Return(paymentLink, nil)
			},
			statusCode:   200,
			responseBody: fmt.Sprintf(`{"paymentLink":"%s"}`, paymentLink),
		},
		{
			name:         "offerId missing",
			body:         fmt.Sprintf(`{"offerId": "", "promoId": "%s"}`, promoId.Hex()),
			mockBehavior: func(r *mock_service.MockOrders, studentId, offerId, promoId primitive.ObjectID) {},
			statusCode:   400,
			responseBody: `{"message":"invalid input body"}`,
		},
		{
			name:         "invalid offerId",
			body:         fmt.Sprintf(`{"offerId": "123", "promoId": "%s"}`, promoId.Hex()),
			mockBehavior: func(r *mock_service.MockOrders, studentId, offerId, promoId primitive.ObjectID) {},
			statusCode:   400,
			responseBody: `{"message":"invalid offer id"}`,
		},
		{
			name:         "invalid promoId",
			body:         fmt.Sprintf(`{"offerId": "%s", "promoId": "123"}`, offerId.Hex()),
			mockBehavior: func(r *mock_service.MockOrders, studentId, offerId, promoId primitive.ObjectID) {},
			statusCode:   400,
			responseBody: `{"message":"invalid promo id"}`,
		},
		{
			name:      "service error",
			body:      fmt.Sprintf(`{"offerId": "%s", "promoId": "%s"}`, offerId.Hex(), promoId.Hex()),
			studentId: studentId,
			offerId:   offerId,
			promoId:   promoId,
			mockBehavior: func(r *mock_service.MockOrders, studentId, offerId, promoId primitive.ObjectID) {
				r.EXPECT().Create(context.Background(), studentId, offerId, promoId).Return("", errors.New("failed to create order"))
			},
			statusCode:   500,
			responseBody: `{"message":"failed to create order"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Init Dependencies
			c := gomock.NewController(t)
			defer c.Finish()

			s := mock_service.NewMockOrders(c)
			tt.mockBehavior(s, tt.studentId, tt.offerId, tt.promoId)

			services := &service.Services{Orders: s}
			handler := Handler{services: services}

			// Init Endpoint
			r := gin.New()
			r.POST("/order", func(c *gin.Context) {
				c.Set(studentCtx, tt.studentId.Hex())
			}, handler.studentCreateOrder)

			// Create Request
			w := httptest.NewRecorder()
			req := httptest.NewRequest("POST", "/order",
				bytes.NewBufferString(tt.body))

			// Make Request
			r.ServeHTTP(w, req)

			// Assert
			assert.Equal(t, w.Code, tt.statusCode)
			assert.Equal(t, w.Body.String(), tt.responseBody)
		})
	}
}

func TestHandler_studentGetModuleOffers(t *testing.T) {
	type mockBehavior func(r *mock_service.MockOffers, schoolId, moduleId primitive.ObjectID, offers []domain.Offer)

	schoolId := primitive.NewObjectID()
	moduleId := primitive.NewObjectID()

	packageIds := []primitive.ObjectID{
		primitive.NewObjectID(), primitive.NewObjectID(),
	}

	tests := []struct {
		name         string
		moduleId     string
		schoolId     primitive.ObjectID
		offers       []domain.Offer
		mockBehavior mockBehavior
		statusCode   int
		responseBody string
	}{
		{
			name:     "ok",
			moduleId: moduleId.Hex(),
			schoolId: schoolId,
			offers: []domain.Offer{
				{
					Name:        "test offer",
					Description: "description",
					SchoolID:    schoolId,
					PackageIDs:  packageIds,
					Benefits: []string{
						"benefit 1",
						"benefit 2",
					},
					Price: domain.Price{
						Value:    6900,
						Currency: "USD",
					},
				},
			},
			mockBehavior: func(r *mock_service.MockOffers, schoolId, moduleId primitive.ObjectID, offers []domain.Offer) {
				r.EXPECT().GetByModule(context.Background(), schoolId, moduleId).Return(offers, nil)
			},
			statusCode:   200,
			responseBody: `{"data":[{"id":"000000000000000000000000","name":"test offer","description":"description","price":{"value":6900,"currency":"USD"},"benefits":["benefit 1","benefit 2"]}]}`,
		},
		{
			name:         "invalid module id",
			moduleId:     "123",
			schoolId:     schoolId,
			mockBehavior: func(r *mock_service.MockOffers, schoolId, moduleId primitive.ObjectID, offers []domain.Offer) {},
			statusCode:   400,
			responseBody: `{"message":"invalid id param"}`,
		},
		{
			name:     "service error",
			moduleId: moduleId.Hex(),
			schoolId: schoolId,
			mockBehavior: func(r *mock_service.MockOffers, schoolId, moduleId primitive.ObjectID, offers []domain.Offer) {
				r.EXPECT().GetByModule(context.Background(), schoolId, moduleId).Return(nil, errors.New("failed to get offers"))
			},
			statusCode:   500,
			responseBody: `{"message":"failed to get offers"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Init Dependencies
			c := gomock.NewController(t)
			defer c.Finish()

			s := mock_service.NewMockOffers(c)

			id, _ := primitive.ObjectIDFromHex(tt.moduleId)
			tt.mockBehavior(s, tt.schoolId, id, tt.offers)

			services := &service.Services{Offers: s}
			handler := Handler{services: services}

			// Init Endpoint
			r := gin.New()
			r.GET("/modules/:id/offers", func(c *gin.Context) {
				c.Set(schoolCtx, domain.School{
					ID: schoolId,
				})
			}, handler.studentGetModuleOffers)

			// Create Request
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", fmt.Sprintf("/modules/%s/offers", tt.moduleId), nil)

			// Make Request
			r.ServeHTTP(w, req)

			// Assert
			assert.Equal(t, w.Code, tt.statusCode)
			assert.Equal(t, w.Body.String(), tt.responseBody)
		})
	}
}

func TestHandler_studentGetModuleLessons(t *testing.T) {
	type mockBehavior func(r *mock_service.MockStudents, schoolId, moduleId, studentId primitive.ObjectID, lessons []domain.Lesson)

	schoolId := primitive.NewObjectID()
	moduleId := primitive.NewObjectID()
	studentId := primitive.NewObjectID()

	tests := []struct {
		name         string
		moduleId     string
		schoolId     primitive.ObjectID
		studentId    primitive.ObjectID
		lessons      []domain.Lesson
		mockBehavior mockBehavior
		statusCode   int
		responseBody string
	}{
		{
			name:      "ok",
			moduleId:  moduleId.Hex(),
			schoolId:  schoolId,
			studentId: studentId,
			lessons: []domain.Lesson{
				{
					Name:      "test lesson",
					Position:  0,
					Published: true,
					Content:   "content",
					SchoolID:  schoolId,
				},
			},
			mockBehavior: func(r *mock_service.MockStudents, schoolId, studentId, moduleId primitive.ObjectID, lessons []domain.Lesson) {
				r.EXPECT().GetModuleLessons(context.Background(), schoolId, studentId, moduleId).Return(lessons, nil)
			},
			statusCode:   200,
			responseBody: fmt.Sprintf(`{"data":[{"id":"000000000000000000000000","name":"test lesson","position":0,"published":true,"content":"content","schoolId":"%s"}]}`, schoolId.Hex()),
		},
		{
			name:      "invalid module id",
			moduleId:  "123",
			schoolId:  schoolId,
			studentId: studentId,
			mockBehavior: func(r *mock_service.MockStudents, schoolId, studentId, moduleId primitive.ObjectID, lessons []domain.Lesson) {
			},
			statusCode:   400,
			responseBody: `{"message":"invalid id param"}`,
		},
		{
			name:      "module is not available",
			moduleId:  moduleId.Hex(),
			schoolId:  schoolId,
			studentId: studentId,
			mockBehavior: func(r *mock_service.MockStudents, schoolId, studentId, moduleId primitive.ObjectID, lessons []domain.Lesson) {
				r.EXPECT().GetModuleLessons(context.Background(), schoolId, studentId, moduleId).Return(lessons, domain.ErrModuleIsNotAvailable)
			},
			statusCode:   403,
			responseBody: fmt.Sprintf(`{"message":"%s"}`, domain.ErrModuleIsNotAvailable.Error()),
		},
		{
			name:      "service error",
			moduleId:  moduleId.Hex(),
			schoolId:  schoolId,
			studentId: studentId,
			mockBehavior: func(r *mock_service.MockStudents, schoolId, studentId, moduleId primitive.ObjectID, lessons []domain.Lesson) {
				r.EXPECT().GetModuleLessons(context.Background(), schoolId, studentId, moduleId).Return(lessons, errors.New("failed to get module"))
			},
			statusCode:   500,
			responseBody: `{"message":"failed to get module"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Init Dependencies
			c := gomock.NewController(t)
			defer c.Finish()

			s := mock_service.NewMockStudents(c)

			id, _ := primitive.ObjectIDFromHex(tt.moduleId)
			tt.mockBehavior(s, tt.schoolId, tt.studentId, id, tt.lessons)

			services := &service.Services{Students: s}
			handler := Handler{services: services}

			// Init Endpoint
			r := gin.New()
			r.GET("/modules/:id/lessons", func(c *gin.Context) {
				c.Set(schoolCtx, domain.School{
					ID: schoolId,
				})
				c.Set(studentCtx, tt.studentId.Hex())
			}, handler.studentGetModuleLessons)

			// Create Request
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", fmt.Sprintf("/modules/%s/lessons", tt.moduleId), nil)

			// Make Request
			r.ServeHTTP(w, req)

			// Assert
			assert.Equal(t, w.Code, tt.statusCode)
			assert.Equal(t, w.Body.String(), tt.responseBody)
		})
	}
}

func TestHandler_studentSetLessonFinished(t *testing.T) {
	type mockBehavior func(r *mock_service.MockStudents, studentId, lessonId primitive.ObjectID)

	lessonId := primitive.NewObjectID()
	schoolId := primitive.NewObjectID()
	studentId := primitive.NewObjectID()

	tests := []struct {
		name         string
		lessonId     string
		studentId    string
		schoolId     primitive.ObjectID
		mockBehavior mockBehavior
		statusCode   int
		responseBody string
	}{
		{
			name:      "ok",
			lessonId:  lessonId.Hex(),
			studentId: studentId.Hex(),
			schoolId:  schoolId,
			mockBehavior: func(r *mock_service.MockStudents, studentId, lessonId primitive.ObjectID) {
				r.EXPECT().SetLessonFinished(context.Background(), studentId, lessonId).Return(nil)
			},
			statusCode:   200,
			responseBody: "",
		},
		{
			name:      "invalid lesson id",
			lessonId:  "123",
			schoolId:  schoolId,
			studentId: studentId.Hex(),
			mockBehavior: func(r *mock_service.MockStudents, studentId, moduleId primitive.ObjectID) {
			},
			statusCode:   400,
			responseBody: `{"message":"invalid id param"}`,
		},
		{
			name:      "module is not available",
			lessonId:  lessonId.Hex(),
			schoolId:  schoolId,
			studentId: studentId.Hex(),
			mockBehavior: func(r *mock_service.MockStudents, studentId, moduleId primitive.ObjectID) {
				r.EXPECT().SetLessonFinished(context.Background(), studentId, moduleId).Return(domain.ErrModuleIsNotAvailable)
			},
			statusCode:   403,
			responseBody: fmt.Sprintf(`{"message":"%s"}`, domain.ErrModuleIsNotAvailable.Error()),
		},
		{
			name:      "service error",
			lessonId:  lessonId.Hex(),
			schoolId:  schoolId,
			studentId: studentId.Hex(),
			mockBehavior: func(r *mock_service.MockStudents, studentId, moduleId primitive.ObjectID) {
				r.EXPECT().SetLessonFinished(context.Background(), studentId, moduleId).Return(errors.New("failed to update student lessons"))
			},
			statusCode:   500,
			responseBody: `{"message":"failed to update student lessons"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Init Dependencies
			c := gomock.NewController(t)
			defer c.Finish()

			s := mock_service.NewMockStudents(c)

			lId, _ := primitive.ObjectIDFromHex(tt.lessonId)
			sId, _ := primitive.ObjectIDFromHex(tt.studentId)

			tt.mockBehavior(s, sId, lId)

			services := &service.Services{Students: s}
			handler := Handler{services: services}

			// Init Endpoint
			r := gin.New()
			r.GET("/lessons/:id/finished", func(c *gin.Context) {
				c.Set(schoolCtx, domain.School{
					ID: schoolId,
				})
				c.Set(studentCtx, tt.studentId)
			}, handler.studentSetLessonFinished)

			// Create Request
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", fmt.Sprintf("/lessons/%s/finished", tt.lessonId), nil)

			// Make Request
			r.ServeHTTP(w, req)

			// Assert
			assert.Equal(t, w.Code, tt.statusCode)
			assert.Equal(t, w.Body.String(), tt.responseBody)
		})
	}
}

func TestHandler_studentSignUp(t *testing.T) {
	type mockBehavior func(r *mock_service.MockStudents, input service.StudentSignUpInput)

	schoolId := primitive.NewObjectID()

	tests := []struct {
		name         string
		requestBody  string
		schoolId     primitive.ObjectID
		serviceInput service.StudentSignUpInput
		mockBehavior mockBehavior
		statusCode   int
		responseBody string
	}{
		{
			name:        "ok",
			requestBody: `{"name":"Vasya","email":"test@test.com","password":"qwerty123","registerSource":"test-course"}`,
			schoolId:    schoolId,
			serviceInput: service.StudentSignUpInput{
				Name:         "Vasya",
				Email:        "test@test.com",
				Password:     "qwerty123",
				SchoolID:     schoolId,
				SchoolDomain: "localhost",
			},
			mockBehavior: func(r *mock_service.MockStudents, input service.StudentSignUpInput) {
				r.EXPECT().SignUp(context.Background(), input).Return(nil)
			},
			statusCode: 201,
		},
		{
			name:         "missing name",
			requestBody:  `{"name":"","email":"test@test.com","password":"qwerty123","registerSource":"test-course"}`,
			schoolId:     schoolId,
			mockBehavior: func(r *mock_service.MockStudents, input service.StudentSignUpInput) {},
			statusCode:   400,
			responseBody: `{"message":"invalid input body"}`,
		},
		{
			name:         "invalid name",
			requestBody:  `{"name":"q","email":"test@test.com","password":"qwerty123","registerSource":"test-course"}`,
			schoolId:     schoolId,
			mockBehavior: func(r *mock_service.MockStudents, input service.StudentSignUpInput) {},
			statusCode:   400,
			responseBody: `{"message":"invalid input body"}`,
		},
		{
			name:         "invalid name",
			requestBody:  `{"name":"q","email":"test@test.com","password":"qwerty123","registerSource":"test-course"}`,
			schoolId:     schoolId,
			mockBehavior: func(r *mock_service.MockStudents, input service.StudentSignUpInput) {},
			statusCode:   400,
			responseBody: `{"message":"invalid input body"}`,
		},
		{
			name:         "missing email",
			requestBody:  `{"name":"Vasya","email":"","password":"qwerty123","registerSource":"test-course"}`,
			schoolId:     schoolId,
			mockBehavior: func(r *mock_service.MockStudents, input service.StudentSignUpInput) {},
			statusCode:   400,
			responseBody: `{"message":"invalid input body"}`,
		},
		{
			name:         "missing password",
			requestBody:  `{"name":"Vasya","email":"test@test.com","password":"","registerSource":"test-course"}`,
			schoolId:     schoolId,
			mockBehavior: func(r *mock_service.MockStudents, input service.StudentSignUpInput) {},
			statusCode:   400,
			responseBody: `{"message":"invalid input body"}`,
		},
		{
			name:         "password too short",
			requestBody:  `{"name":"Vasya","email":"test@test.com","password":"qwerty","registerSource":"test-course"}`,
			schoolId:     schoolId,
			mockBehavior: func(r *mock_service.MockStudents, input service.StudentSignUpInput) {},
			statusCode:   400,
			responseBody: `{"message":"invalid input body"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Init Dependencies
			c := gomock.NewController(t)
			defer c.Finish()

			s := mock_service.NewMockStudents(c)

			tt.mockBehavior(s, tt.serviceInput)

			services := &service.Services{Students: s}
			handler := Handler{services: services}

			// Init Endpoint
			r := gin.New()
			r.GET("/sign-up", func(c *gin.Context) {
				c.Set(schoolCtx, domain.School{
					ID: tt.schoolId,
					Settings: domain.Settings{
						Domains: []string{"localhost"},
					},
				})
				c.Set(domainCtx, "localhost")
			}, handler.studentSignUp)

			// Create Request
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/sign-up", bytes.NewBufferString(tt.requestBody))

			// Make Request
			r.ServeHTTP(w, req)

			// Assert
			assert.Equal(t, w.Code, tt.statusCode)
			assert.Equal(t, w.Body.String(), tt.responseBody)
		})
	}
}
