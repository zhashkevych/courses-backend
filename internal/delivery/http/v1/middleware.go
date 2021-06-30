package v1

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/zhashkevych/creatly-backend/internal/domain"
	"github.com/zhashkevych/creatly-backend/pkg/logger"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	authorizationHeader = "Authorization"

	studentCtx = "studentId"
	adminCtx   = "adminId"
	userCtx    = "userId"
	schoolCtx  = "school"
	domainCtx  = "domain"
)

func (h *Handler) setSchoolFromRequest(c *gin.Context) {
	host := parseRequestHost(c)

	school, err := h.services.Schools.GetByDomain(c.Request.Context(), host)
	if err != nil {
		logger.Error(err)
		c.AbortWithStatus(http.StatusForbidden)

		return
	}

	c.Set(schoolCtx, school)
	c.Set(domainCtx, host)
}

func parseRequestHost(c *gin.Context) string {
	refererHeader := c.Request.Header.Get("Referer")
	refererParts := strings.Split(refererHeader, "/")

	// this logic is used to avoid crashes during integration testing
	if len(refererParts) < 3 {
		return c.Request.Host
	}

	hostParts := strings.Split(refererParts[2], ":")

	return hostParts[0]
}

func getSchoolFromContext(c *gin.Context) (domain.School, error) {
	value, ex := c.Get(schoolCtx)
	if !ex {
		return domain.School{}, errors.New("school is missing from ctx")
	}

	school, ok := value.(domain.School)
	if !ok {
		return domain.School{}, errors.New("failed to convert value from ctx to domain.School")
	}

	return school, nil
}

func (h *Handler) studentIdentity(c *gin.Context) {
	id, err := h.parseAuthHeader(c)
	if err != nil {
		newResponse(c, http.StatusUnauthorized, err.Error())
	}

	c.Set(studentCtx, id)
}

func (h *Handler) adminIdentity(c *gin.Context) {
	id, err := h.parseAuthHeader(c)
	if err != nil {
		newResponse(c, http.StatusUnauthorized, err.Error())
	}

	c.Set(adminCtx, id)
}

func (h *Handler) userIdentity(c *gin.Context) {
	id, err := h.parseAuthHeader(c)
	if err != nil {
		newResponse(c, http.StatusUnauthorized, err.Error())
	}

	c.Set(userCtx, id)
}

func (h *Handler) parseAuthHeader(c *gin.Context) (string, error) {
	header := c.GetHeader(authorizationHeader)
	if header == "" {
		return "", errors.New("empty auth header")
	}

	headerParts := strings.Split(header, " ")
	if len(headerParts) != 2 || headerParts[0] != "Bearer" {
		return "", errors.New("invalid auth header")
	}

	if len(headerParts[1]) == 0 {
		return "", errors.New("token is empty")
	}

	return h.tokenManager.Parse(headerParts[1])
}

func getStudentId(c *gin.Context) (primitive.ObjectID, error) {
	return getIdByContext(c, studentCtx)
}

func getUserId(c *gin.Context) (primitive.ObjectID, error) {
	return getIdByContext(c, userCtx)
}

func getIdByContext(c *gin.Context, context string) (primitive.ObjectID, error) {
	idFromCtx, ok := c.Get(context)
	if !ok {
		return primitive.ObjectID{}, errors.New("studentCtx not found")
	}

	idStr, ok := idFromCtx.(string)
	if !ok {
		return primitive.ObjectID{}, errors.New("studentCtx is of invalid type")
	}

	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		return primitive.ObjectID{}, err
	}

	return id, nil
}

func getDomainFromContext(c *gin.Context) (string, error) {
	val, ex := c.Get(domainCtx)
	if !ex {
		return "", errors.New("domainCtx not found")
	}

	valStr, ok := val.(string)
	if !ok {
		return "", errors.New("domainCtx is of invalid type")
	}

	return valStr, nil
}
