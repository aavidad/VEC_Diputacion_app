package bootstrap

import (
	"errors"
	"net/http"

	"vec-diputacion-granada/internal/app/composicion/interna/contrataciontemporal/firmavec"
)

var errIdentidadCertificadoFirmaVecNoDisponible = errors.New("contratacion temporal: identidad certificado firma vec no disponible")

// La composición local conserva su API mientras el runtime interno monta el
// mismo puente desde firmavec. Este wrapper sólo añade el contenedor AUT56.
type ConfiguracionIdentidadCertificadoFirmaVecV2 = firmavec.ConfiguracionIdentidadCertificadoFirmaVecV2
type AcreditacionCertificadoFirmaVecV2 = firmavec.AcreditacionCertificadoFirmaVecV2
type AcreditadorCertificadoFirmaVecV2 = firmavec.AcreditadorCertificadoFirmaVecV2
type EmisorAsercionCertificadoFirmaVecV2 = firmavec.EmisorAsercionCertificadoFirmaVecV2
type VinculadorCertificadoFirmaVecV2 = firmavec.VinculadorCertificadoFirmaVecV2

type EntornoIdentidadCertificadoFirmaVecV2 struct {
	puente    *firmavec.EntornoIdentidadCertificadoFirmaVecV2
	autoridad *autoridadSesionFirmanteV2
}

func NuevaIdentidadCertificadoFirmaVecV2(c ConfiguracionIdentidadCertificadoFirmaVecV2) (*EntornoIdentidadCertificadoFirmaVecV2, error) {
	puente, err := firmavec.NuevaIdentidadCertificadoFirmaVecV2(c)
	if err != nil {
		return nil, errIdentidadCertificadoFirmaVecNoDisponible
	}
	autoridad, err := nuevaAutoridadSesionFirmanteV2Certificado(puente.FuenteComun(), c.Reloj)
	if err != nil {
		return nil, errIdentidadCertificadoFirmaVecNoDisponible
	}
	return &EntornoIdentidadCertificadoFirmaVecV2{puente: puente, autoridad: autoridad}, nil
}

func (e *EntornoIdentidadCertificadoFirmaVecV2) Envolver(siguiente http.Handler) (http.Handler, error) {
	if e == nil || e.puente == nil || e.autoridad == nil || dependenciaEsNulaContratacionTemporalDesarrollo(siguiente) {
		return nil, errIdentidadCertificadoFirmaVecNoDisponible
	}
	return e.puente.Envolver(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		preparada, err := prepararPeticionFirmaVecV2(r)
		if err != nil || preparada == nil {
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		siguiente.ServeHTTP(w, preparada)
	}))
}
