package httpinterno

import (
	"context"
	"errors"
	"net/http"
)

// CodigoCierreSinCeseNoContemplado responde el cierre administrativo sin cese
// cuando la regla de cierre del catálogo (c10) no lo contempla: la pantalla
// no lo ofrece y el servidor no lo registra.
const CodigoCierreSinCeseNoContemplado = "cierre_sin_cese_no_contemplado"

var errCierreSinCeseNoContemplado = errors.New("contratacion temporal http: cierre sin cese no contemplado por la regla de cierre")

// ExigirAdmisionCierreSinCese antepone al manejador del cierre sin cese (y a
// su preparación) la admisión que decide la regla vigente. Sin decisión
// (nil) se conserva la conducta anterior. Una regla ilegible no se
// interpreta como admisión.
func ExigirAdmisionCierreSinCese(h http.Handler, admitido func(context.Context) (bool, error)) http.Handler {
	if h == nil || admitido == nil {
		return h
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r == nil {
			responderErrorCierreAdministrativo(w, r, errorServicioCierreAdministrativoNoDisponible)
			return
		}
		ok, err := admitido(r.Context())
		if err != nil {
			responderErrorCierreAdministrativo(w, r, errorServicioCierreAdministrativoNoDisponible, err)
			return
		}
		if !ok {
			responderErrorCierreAdministrativo(w, r, nuevoErrorCierreAdministrativo(http.StatusConflict, CodigoCierreSinCeseNoContemplado),
				errCierreSinCeseNoContemplado)
			return
		}
		h.ServeHTTP(w, r)
	})
}
