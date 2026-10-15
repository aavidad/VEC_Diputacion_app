package ports

import (
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

const (
	accionAlmacenSubirComprobanteMeritos                                     = "meritos.cartera.comprobante.subir_propio"
	accionAlmacenDescargarComprobanteMeritos                                 = "meritos.cartera.comprobante.descargar_propio"
	accionAlmacenSubirJustificanteRenunciaBolsa                              = "bolsa.justificante_renuncia.subir_propio"
	accionAlmacenDescargarJustificanteRenunciaBolsa                          = "bolsa.justificante_renuncia.descargar_propio"
	accionAlmacenDescargarJustificanteRenunciaRRHHBolsa                      = "bolsa.justificante_renuncia.descargar_rrhh"
	accionAlmacenSubirJustificanteDietas                                     = "dietas.justificante.subir_propio"
	accionAlmacenDescargarJustificanteDietas                                 = "dietas.justificante.descargar_propio"
	accionAlmacenSubirJustificacionCronos                                    = "cronos.justificacion.subir_propio"
	accionAlmacenDescargarJustificacionCronos                                = "cronos.justificacion.descargar_propio"
	PasoAlmacenEscribirSubidaLocal                      PasoOperacionAlmacen = "01_escribir_subida_local"
	PasoAlmacenRecuperarEscrituraSubidaLocal            PasoOperacionAlmacen = "01_recuperar_escritura_subida_local"
	PasoAlmacenRecuperarPromocionSubidaLocal            PasoOperacionAlmacen = "01_recuperar_promocion_subida_local"
)

type FaseAlmacenSubidaLocal string

const (
	FaseAlmacenSubidaLocalEscribir           FaseAlmacenSubidaLocal = "escribir"
	FaseAlmacenSubidaLocalAnalizar           FaseAlmacenSubidaLocal = "analizar"
	FaseAlmacenSubidaLocalPromover           FaseAlmacenSubidaLocal = "promover"
	FaseAlmacenSubidaLocalDescargar          FaseAlmacenSubidaLocal = "descargar"
	FaseAlmacenSubidaLocalRecuperarEscritura FaseAlmacenSubidaLocal = "recuperar_escritura"
	FaseAlmacenSubidaLocalRecuperarPromocion FaseAlmacenSubidaLocal = "recuperar_promocion"
)

func accionAlmacenSubidaLocalValida(accion string, fase FaseAlmacenSubidaLocal) bool {
	switch fase {
	case FaseAlmacenSubidaLocalEscribir, FaseAlmacenSubidaLocalAnalizar, FaseAlmacenSubidaLocalPromover,
		FaseAlmacenSubidaLocalRecuperarEscritura, FaseAlmacenSubidaLocalRecuperarPromocion:
		return accion == accionAlmacenSubirComprobanteMeritos ||
			accion == accionAlmacenSubirJustificanteRenunciaBolsa ||
			accion == accionAlmacenSubirJustificanteDietas || accion == accionAlmacenSubirJustificacionCronos
	case FaseAlmacenSubidaLocalDescargar:
		return accion == accionAlmacenDescargarComprobanteMeritos ||
			accion == accionAlmacenDescargarJustificanteRenunciaBolsa ||
			accion == accionAlmacenDescargarJustificanteRenunciaRRHHBolsa ||
			accion == accionAlmacenDescargarJustificanteDietas || accion == accionAlmacenDescargarJustificacionCronos
	default:
		return false
	}
}

func especificacionSubidaLocal(accion string, fase FaseAlmacenSubidaLocal) (especificacionAutorizacionAlmacen, bool) {
	if !accionAlmacenSubidaLocalValida(accion, fase) {
		return especificacionAutorizacionAlmacen{}, false
	}
	s := especificacionAutorizacionAlmacen{accionNegocio: accion}
	switch fase {
	case FaseAlmacenSubidaLocalEscribir:
		s.camposExactos = []string{"contenido", "estado", "recibo"}
		s.pasos = []pasoPlanOperacionAlmacen{{referencia: PasoAlmacenEscribirSubidaLocal, accion: AccionAlmacenEscribir}}
	case FaseAlmacenSubidaLocalAnalizar:
		s.camposExactos = []string{"contenido", "estado", "recibo"}
		s.pasos = []pasoPlanOperacionAlmacen{
			{referencia: PasoAlmacenLeerParaAnalisis, accion: AccionAlmacenLeer},
			{referencia: PasoAlmacenAnalizarContenido, accion: AccionAlmacenAnalizarContenido},
		}
		s.requiereObjeto = true
	case FaseAlmacenSubidaLocalPromover:
		s.camposExactos = []string{"contenido", "estado", "recibo"}
		s.pasos = []pasoPlanOperacionAlmacen{{referencia: PasoAlmacenPromover, accion: AccionAlmacenPromover}}
		s.requiereObjeto = true
	case FaseAlmacenSubidaLocalDescargar:
		s.camposExactos = []string{"contenido"}
		s.pasos = []pasoPlanOperacionAlmacen{{referencia: PasoAlmacenLeerOriginalDocumento, accion: AccionAlmacenLeer}}
		s.requiereObjeto = true
	case FaseAlmacenSubidaLocalRecuperarEscritura:
		s.camposExactos = []string{"contenido", "estado", "recibo"}
		s.pasos = []pasoPlanOperacionAlmacen{{referencia: PasoAlmacenRecuperarEscrituraSubidaLocal, accion: AccionAlmacenLeer}}
	case FaseAlmacenSubidaLocalRecuperarPromocion:
		s.camposExactos = []string{"contenido", "estado", "recibo"}
		s.pasos = []pasoPlanOperacionAlmacen{{referencia: PasoAlmacenRecuperarPromocionSubidaLocal, accion: AccionAlmacenLeer}}
	}
	return s, s.valida()
}

// Cada fase exige una nueva concesión registrada para la misma acción del
// módulo y el objeto/versión exactos cuando ya existe. Nunca se deriva una
// capacidad de análisis o promoción del contexto de escritura anterior.
func NuevoContextoSubidaLocalAlmacenV3(
	solicitud domain.SolicitudAutorizacionLigadaV3,
	decision domain.DecisionAutorizacionLigadaV3,
	confirmacion ConfirmacionRegistroConcesionAutorizacionLigadaV3,
	vinculos VinculosOperacionAlmacen,
	fase FaseAlmacenSubidaLocal,
	verificadaEn time.Time,
) (ContextoOperacionAlmacen, error) {
	datos, err := solicitud.Datos()
	if err != nil {
		return ContextoOperacionAlmacen{}, errorAutorizacionAlmacen()
	}
	especificacion, ok := especificacionSubidaLocal(datos.Accion, fase)
	if !ok {
		return ContextoOperacionAlmacen{}, errorAutorizacionAlmacen()
	}
	return nuevoContextoOperacionAlmacenV3(solicitud, decision, confirmacion, vinculos, verificadaEn, especificacion)
}
