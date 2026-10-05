package ports

import (
	"context"
	"time"
	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// FuenteDescriptorFirmaV2 selecciona el paso y cargo/enlace desde el plan
// gobernado de CT antes del PDP. AUT35 resuelve sus fuentes en la TX final.
type FuenteDescriptorFirmaV2 interface {
	DescriptorFirmaV2(context.Context, MaterialFirmaVerificadaV2) (DescriptorConstructorFirmaV2, error)
}

type SeleccionConstructorFirmaV2 struct {
	PerfilEsperadoRef  string `json:"perfil_esperado_ref"`
	PerfilActivoRef    string `json:"perfil_activo_ref"`
	RolID              string `json:"rol_id"`
	CargoRef           string `json:"cargo_ref"`
	EnlaceEjercicioRef string `json:"enlace_ejercicio_ref"`
}

type DescriptorConstructorFirmaV2 struct {
	Esquema              string                              `json:"esquema"`
	CertificadoDERSHA256 string                              `json:"certificado_der_sha256"`
	Seleccion            SeleccionConstructorFirmaV2         `json:"seleccion"`
	Recurso              vd.RecursoFirmaHistoricaV1          `json:"recurso"`
	Accion               string                              `json:"accion"`
	Finalidad            string                              `json:"finalidad"`
	Motivo               vd.ReferenciaEntradaCatalogo        `json:"motivo"`
	Circuito             vd.ReferenciaHistoricaCompetenciaV1 `json:"circuito"`
	PasoRef              string                              `json:"paso_ref"`
	PasoOrden            uint64                              `json:"paso_orden"`
	// CT reemplaza el null del descriptor exterior por el instante original.
	FechaHistorica *time.Time `json:"fecha_historica"`
}

// EmisorMaterialFirmaVerificadaV2 usa el PDP común y devuelve su exportación.
// La implementación de composición acredita identidad, perfil y canal reales.
type EmisorMaterialFirmaVerificadaV2 interface {
	AutorizarMaterialFirmaVerificadaV2(context.Context, MaterialFirmaVerificadaV2, vd.RecursoAutorizable) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
	FuentePerfilActivoOperadorFirmaV2
	FuenteAmbitosOperadorFirmaV2
}

// AmbitosOperadorFirmaV2 son los ámbitos de la asignación vigente de quien
// actúa: la organización y, sólo si la asignación la tiene, la unidad. El PDP
// exige que el recurso tenga exactamente las dimensiones de esa asignación,
// y AD206 las vuelve a leer en el consumo para calcular la huella.
type AmbitosOperadorFirmaV2 struct {
	OrganizacionRef, UnidadRef string
}

// Mapa devuelve los ámbitos del recurso; la unidad vacía no es una dimensión.
func (a AmbitosOperadorFirmaV2) Mapa() map[string]string {
	m := map[string]string{"organizacion_ref": a.OrganizacionRef}
	if a.UnidadRef != "" {
		m["unidad_ref"] = a.UnidadRef
	}
	return m
}

// FuenteAmbitosOperadorFirmaV2 lee la asignación vigente del perfil activo
// revalidado. No acepta ámbitos del material ni de la petición.
type FuenteAmbitosOperadorFirmaV2 interface {
	ObtenerAmbitosOperadorFirmaV2(context.Context) (AmbitosOperadorFirmaV2, error)
}
