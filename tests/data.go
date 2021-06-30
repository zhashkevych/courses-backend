package tests

import (
	"time"

	"github.com/zhashkevych/creatly-backend/internal/domain"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	school = domain.School{
		ID: primitive.NewObjectID(),
		Courses: []domain.Course{
			{
				ID:        primitive.NewObjectID(),
				Name:      "Course #1",
				Published: true,
			},
			{
				ID:   primitive.NewObjectID(),
				Name: "Course #2", // Unpublished course, shouldn't be available to student
			},
		},
		Settings: domain.Settings{
			Domains: []string{"http://localhost:1337", "workshop.zhashkevych.com", ""},
		},
	}

	packages = []interface{}{
		domain.Package{
			ID:       primitive.NewObjectID(),
			Name:     "Package #1",
			CourseID: school.Courses[0].ID,
		},
	}

	offers = []interface{}{
		domain.Offer{
			ID:          primitive.NewObjectID(),
			Name:        "Offer #1",
			Description: "Offer #1 Description",
			SchoolID:    school.ID,
			PackageIDs:  []primitive.ObjectID{packages[0].(domain.Package).ID},
			Price:       domain.Price{Value: 6900, Currency: "USD"},
		},
	}

	promocodes = []interface{}{
		domain.PromoCode{
			ID:                 primitive.NewObjectID(),
			Code:               "TEST25",
			DiscountPercentage: 25,
			ExpiresAt:          time.Now().Add(time.Hour),
			OfferIDs:           []primitive.ObjectID{offers[0].(domain.Offer).ID},
			SchoolId:           school.ID,
		},
	}

	modules = []interface{}{
		domain.Module{
			ID:        primitive.NewObjectID(),
			Name:      "Module #1", // Free Module, should be available to anyone
			CourseID:  school.Courses[0].ID,
			Published: true,
			Lessons: []domain.Lesson{
				{
					ID:        primitive.NewObjectID(),
					Name:      "Lesson #1",
					Published: true,
				},
			},
		},
		domain.Module{
			ID:        primitive.NewObjectID(),
			Name:      "Module #2", // Part of paid package, should be available only after purchase
			CourseID:  school.Courses[0].ID,
			Published: true,
			PackageID: packages[0].(domain.Package).ID,
			Lessons: []domain.Lesson{
				{
					ID:        primitive.NewObjectID(),
					Name:      "Lesson #1",
					Published: true,
				},
				{
					ID:        primitive.NewObjectID(),
					Name:      "Lesson #2",
					Published: true,
				},
			},
		},
		domain.Module{
			ID:        primitive.NewObjectID(),
			Name:      "Module #1", // Part of unpublished course
			CourseID:  school.Courses[1].ID,
			Published: true,
			PackageID: packages[0].(domain.Package).ID,
			Lessons: []domain.Lesson{
				{
					ID:        primitive.NewObjectID(),
					Name:      "Lesson #1",
					Published: true,
				},
				{
					ID:        primitive.NewObjectID(),
					Name:      "Lesson #2",
					Published: true,
				},
			},
		},
	}
)
