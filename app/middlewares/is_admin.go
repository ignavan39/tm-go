package middlewares

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/ignavan39/ucrm-go/app/auth"
	"github.com/ignavan39/ucrm-go/app/repository"
	"github.com/ignavan39/ucrm-go/pkg/httpext"
)

func IsAdminGuard(repo repository.DashboardRepository) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			id := chi.URLParam(r, "dashboardId")
			if len(id) == 0 {
				var payload struct {
					DashboardId string `json:"dashboard_id"`
				}

				body, err := ioutil.ReadAll(r.Body)
				if err != nil {
					httpext.JSON(w, httpext.CommonError{
						Error: "failed decode payload",
						Code:  http.StatusBadRequest,
					}, http.StatusBadRequest)
					return
				}

				reader := ioutil.NopCloser(bytes.NewReader(body))
				r.Body = reader
				err = json.Unmarshal(body, &payload)
				if err != nil {
					httpext.JSON(w, httpext.CommonError{
						Error: "failed decode payload",
						Code:  http.StatusBadRequest,
					}, http.StatusBadRequest)
					return
				}
				id = payload.DashboardId
			}

			if len(id) == 0 {
				httpext.JSON(w, httpext.CommonError{
					Error: "wrong dashboard id",
					Code:  http.StatusBadRequest,
				}, http.StatusBadRequest)
				return
			}

			userId := auth.GetUserIdFromContext(ctx)
			dashboard, err := repo.GetOneDashboardWithUserAccess(id, userId, "rw")
			if err != nil {
				httpext.JSON(w, httpext.CommonError{
					Error: err.Error(),
					Code:  http.StatusInternalServerError,
				}, http.StatusInternalServerError)
				return
			}

			if dashboard.AuthorId == userId {
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("Forbidden"))
		})
	}
}
