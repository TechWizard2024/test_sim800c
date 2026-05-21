package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/sirupsen/logrus"
)

func Recovery(logger *logrus.Logger) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.WithFields(logrus.Fields{
						"error": err,
						"stack": string(debug.Stack()),
						"path":  r.URL.Path,
					}).Error("Panic récupéré")

					http.Error(w, fmt.Sprintf("Erreur interne: %v", err), http.StatusInternalServerError)
				}
			}()

			next(w, r)
		}
	}
}
