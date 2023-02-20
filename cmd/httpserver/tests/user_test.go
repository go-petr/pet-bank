//go:build integration

package tests

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-petr/pet-bank/pkg/web"
)

func TestCreateUserAPI(t *testing.T) {

	testCases := []struct {
		name           string
		requestBody    gin.H
		wantStatusCode int
		checkResponse  func(got web.Response)
	}{}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

		})
	}

}
