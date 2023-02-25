package sessiondelivery

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-petr/pet-bank/pkg/errorspkg"
	"github.com/go-petr/pet-bank/pkg/randompkg"
	"github.com/go-petr/pet-bank/pkg/tokenpkg"
	"github.com/go-petr/pet-bank/pkg/web"
	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestRenewAccessToken(t *testing.T) {
	symmetricKey := randompkg.String(32)

	tokenMaker, err := tokenpkg.NewPasetoMaker(symmetricKey)
	if err != nil {
		t.Fatalf("tokenpkg.NewPasetoMaker(%v) returned error: %v", symmetricKey, err)
	}

	type requestBody struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	username := randompkg.Owner()
	duration := time.Minute

	token, payload, err := tokenMaker.CreateToken(username, duration)
	if err != nil {
		t.Fatalf("tokenMaker.CreateToken(%v, %v) returned error: %v", username, duration, err)
	}

	testCases := []struct {
		name           string
		requestBody    requestBody
		buildStubs     func(service *MockService)
		wantStatusCode int
		checkData      func(t *testing.T, res web.Response)
		wantError      string
	}{
		{
			name: "OK",
			requestBody: requestBody{
				RefreshToken: token,
			},
			buildStubs: func(service *MockService) {
				service.EXPECT().
					RenewAccessToken(gomock.Any(), token).
					Times(1).
					Return(token, payload.ExpiredAt, nil)
			},
			wantStatusCode: http.StatusOK,
			checkData: func(t *testing.T, got web.Response) {
				t.Helper()

				want := web.Response{
					AccessToken:          token,
					AccessTokenExpiresAt: payload.ExpiredAt,
				}

				compareCreatedAt := cmpopts.EquateApproxTime(time.Second)
				if diff := cmp.Diff(want, got, compareCreatedAt); diff != "" {
					t.Errorf("res.Data mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			name: "RequiredRefreshToken",
			requestBody: requestBody{
				RefreshToken: "",
			},
			buildStubs: func(service *MockService) {
				service.EXPECT().
					RenewAccessToken(gomock.Any(), token).
					Times(0)
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "RefreshToken field is required",
		},
		{
			name: "InternalServiceError",
			requestBody: requestBody{
				RefreshToken: token,
			},
			buildStubs: func(service *MockService) {
				service.EXPECT().
					RenewAccessToken(gomock.Any(), token).
					Times(1).
					Return("", time.Now(), errorspkg.ErrInternal)
			},
			wantStatusCode: http.StatusInternalServerError,
			wantError:      errorspkg.ErrInternal.Error(),
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Set up mocks
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			sessionService := NewMockService(ctrl)
			sessionHandler := NewHandler(sessionService)

			gin.SetMode(gin.ReleaseMode)
			server := gin.New()
			url := "/sessions"

			server.POST(url, sessionHandler.RenewAccessToken)

			tc.buildStubs(sessionService)

			// Send request
			body, err := json.Marshal(tc.requestBody)
			if err != nil {
				t.Fatalf("Encoding request body error: %v", err)
			}

			req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
			if err != nil {
				t.Fatalf("Creating request error: %v", err)
			}

			w := httptest.NewRecorder()
			server.ServeHTTP(w, req)

			// Test response
			if got := w.Code; got != tc.wantStatusCode {
				t.Errorf("Status code: got %v, want %v", got, tc.wantStatusCode)
			}

			res := web.Response{}

			if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
				t.Errorf("Decoding response body error: %v", err)
			}

			if tc.wantStatusCode != http.StatusOK {
				if res.Error != tc.wantError {
					t.Errorf(`res.Error=%q, want %q`, res.Error, tc.wantError)
				}
			} else {
				tc.checkData(t, res)
			}
		})
	}
}
