package server

import (
	"net/http"
	"vec-diputacion-granada/internal/shared/limiteshttp"
)

// El límite general sigue vigente para el resto de rutas. Esta carga tiene
// un máximo de 16 MiB de PDF y un margen acotado para la envoltura multipart.
func limiteCuerpoSolicitud(r *http.Request, general int64) int64 {
	const ruta = "/api/vec/bolsa/mi-bolsa/justificantes"
	const multipartPDF = int64((16 << 20) + (64 << 10))
	if r != nil && r.URL != nil && r.Method == http.MethodPost && r.URL.Path == ruta &&
		r.URL.EscapedPath() == ruta && r.URL.RawQuery == "" && !r.URL.ForceQuery &&
		limiteshttp.MismoOrigenEstricto(r) && general < multipartPDF {
		return multipartPDF
	}
	return general
}
