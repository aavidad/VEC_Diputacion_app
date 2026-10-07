package httpinterno

import (
	"net/http"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// El parser v2 queda disponible para el futuro handler de lectura con sesión.
// No se monta sobre la ruta V3 mientras falte la autoridad y auditoría W.
type filtrosCuadroRRHHSesionV2JSON struct {
	Texto        string   `json:"texto"`
	CentroRef    string   `json:"centro_ref"`
	CategoriaRef string   `json:"categoria_ref"`
	EstadosClave []string `json:"estados_clave"`
	FasesClave   []string `json:"fases_clave"`
}

type consultaCuadroRRHHSesionV2JSON struct {
	Filtros    *filtrosCuadroRRHHSesionV2JSON `json:"filtros"`
	Paginacion *paginacionCuadroRRHHJSON      `json:"paginacion"`
	Resumen    bool                           `json:"resumen,omitempty"`
}

func solicitudCuadroRRHHSesionV2DesdePeticion(
	w http.ResponseWriter,
	r *http.Request,
) (ports.ConsultaCuadroRRHHSesionV2, bool, error) {
	var entrada consultaCuadroRRHHSesionV2JSON
	if err := decodificarConsultaRRHH(w, r, MaximoCuerpoConsultaCuadroRRHHBytes, &entrada); err != nil {
		return ports.ConsultaCuadroRRHHSesionV2{}, false, err
	}
	if entrada.Filtros == nil || entrada.Paginacion == nil || entrada.Paginacion.Limite == nil {
		return ports.ConsultaCuadroRRHHSesionV2{}, false, errContenidoConsultaRRHHNoValido
	}
	estados := make([]domain.EstadoOperativo, len(entrada.Filtros.EstadosClave))
	for i, estado := range entrada.Filtros.EstadosClave {
		estados[i] = domain.EstadoOperativo(estado)
	}
	fases := make([]domain.ClaveFase, len(entrada.Filtros.FasesClave))
	for i, fase := range entrada.Filtros.FasesClave {
		fases[i] = domain.ClaveFase(fase)
	}
	solicitud, err := ports.NuevaConsultaCuadroRRHHSesionV2(
		entrada.Filtros.Texto, entrada.Filtros.CentroRef,
		entrada.Filtros.CategoriaRef, estados, fases,
		*entrada.Paginacion.Limite, entrada.Paginacion.Cursor,
	)
	if err != nil {
		return ports.ConsultaCuadroRRHHSesionV2{}, false, errContenidoConsultaRRHHNoValido
	}
	return solicitud, entrada.Resumen, nil
}
