package informejuridico

import (
	"context"
	"strings"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// RenderizadorBorradorDesarrollo representa la consulta ya autorizada.
// No consulta más datos, registra propuestas ni sustituye un modelo oficial.
type RenderizadorBorradorDesarrollo struct {
	PDF vecports.RenderizadorDocumento
	// Etiquetas devuelve el nombre de catálogo de una referencia (centro,
	// categoría, unidad) o cadena vacía si no lo conoce. Puede ser nil.
	Etiquetas EtiquetadorReferencias
	// Plantillas aporta el texto de cada documento. Sin catálogo no hay
	// borradores: el texto no vive en el código.
	Plantillas *PlantillasBorrador
}

// EtiquetadorReferencias traduce una referencia opaca al nombre con el que la
// conoce RRHH. Es sólo presentación: no altera el detalle ni sus huellas.
type EtiquetadorReferencias func(referencia string) string

// referenciaConNombre imprime «Nombre (referencia)» cuando hay nombre y la
// referencia sola cuando no; nunca oculta la referencia, que es lo que consta
// en el expediente.
func referenciaConNombre(etiquetar EtiquetadorReferencias, referencia string) string {
	if etiquetar == nil {
		return referencia
	}
	nombre := strings.TrimSpace(etiquetar(referencia))
	if nombre == "" {
		return referencia
	}
	return nombre + " (" + referencia + ")"
}

func (r RenderizadorBorradorDesarrollo) RenderizarBorrador(
	ctx context.Context, tipo ports.TipoBorradorRRHH, detalle ports.DetalleExpedienteRRHH,
) ([]byte, error) {
	if ctx == nil || r.PDF == nil || r.PDF.Formato() != vecdomain.FormatoDocumentoPDF || r.Plantillas == nil {
		return nil, ports.ErrConsultaRRHHNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	documento, err := contenidoBorradorDesarrollo(tipo, detalle, r.Etiquetas, r.Plantillas)
	if err != nil {
		return nil, err
	}
	contenido, err := r.PDF.Renderizar(ctx, documento)
	if err != nil {
		return nil, err
	}
	if err := r.PDF.ValidarSalida(ctx, contenido); err != nil {
		return nil, err
	}
	return contenido, ctx.Err()
}

// contenidoBorradorDesarrollo mantiene una única fuente para todos los
// textos preparatorios: la plantilla del catálogo. PDF y DOCX sólo difieren en
// la representación final.
func contenidoBorradorDesarrollo(
	tipo ports.TipoBorradorRRHH,
	detalle ports.DetalleExpedienteRRHH,
	etiquetar EtiquetadorReferencias,
	plantillas *PlantillasBorrador,
) (vecdomain.ContenidoDocumento, error) {
	plantilla, ok := plantillas.Plantilla(tipo)
	if !ok {
		return vecdomain.ContenidoDocumento{}, ports.ErrBorradorRRHHNoDisponible
	}
	solicitud, err := ports.NuevaSolicitudDetalleRRHH(
		detalle.Resumen.ExpedienteRef, detalle.Resumen.Version,
	)
	if err != nil || detalle.ValidarContenidoPublicablePara(solicitud) != nil ||
		detalle.Analisis == nil || detalle.Cobertura == nil || detalle.Asignacion == nil ||
		!plantilla.AdmiteModalidad(detalle.Analisis.ModalidadClave) {
		return vecdomain.ContenidoDocumento{}, ports.ErrBorradorRRHHNoDisponible
	}
	fuente := fuenteCampos{detalle: detalle, etiquetar: etiquetar, plantillas: plantillas, plantilla: plantilla}
	if plantilla.RequiereAccion == "" {
		if detalle.Resumen.FaseClave != "nombramiento" || detalle.Resumen.EstadoClave != domain.EstadoEnCurso {
			return vecdomain.ContenidoDocumento{}, ports.ErrBorradorRRHHNoDisponible
		}
		fuente.propuesta, ok = propuestaFormalizacionBorrador(detalle.Hitos)
	} else {
		fuente.propuesta, fuente.requerido, ok = propuestaAnteriorA(detalle.Hitos, plantilla.RequiereAccion)
	}
	hito := fuente.propuesta
	if !ok || hito.VersionExpediente < 7 ||
		hito.FaseDestino != "nombramiento" || hito.EstadoDestino != domain.EstadoEnCurso {
		return vecdomain.ContenidoDocumento{}, ports.ErrBorradorRRHHNoDisponible
	}
	return plantilla.Rellenar(fuente.valores())
}

// propuestaAnteriorA localiza la última actuación que exige la plantilla
// (cese, modificación…) y la propuesta de formalización que la precede. Los
// documentos posteriores al nombramiento no dependen de que la propuesta sea
// la última actuación.
func propuestaAnteriorA(hitos []ports.HitoExpedienteRRHH, accion string) (ports.HitoExpedienteRRHH, *ports.HitoExpedienteRRHH, bool) {
	for i := len(hitos) - 1; i >= 0; i-- {
		if string(hitos[i].AccionClave) != accion {
			continue
		}
		requerido := hitos[i]
		for j := i - 1; j >= 0; j-- {
			if hitos[j].AccionClave == "registrar_propuesta_formalizacion" {
				return hitos[j], &requerido, true
			}
		}
		return ports.HitoExpedienteRRHH{}, nil, false
	}
	return ports.HitoExpedienteRRHH{}, nil, false
}

// propuestaFormalizacionBorrador conserva el antecedente de la propuesta
// cuando la misma historia ya registra su resolución y la anotación
// administrativa posterior. No interpreta esas actuaciones como firma,
// eficacia o incorporación.
func propuestaFormalizacionBorrador(hitos []ports.HitoExpedienteRRHH) (ports.HitoExpedienteRRHH, bool) {
	indice := len(hitos) - 1
	switch {
	case indice >= 0 && hitos[indice].AccionClave == "registrar_propuesta_formalizacion":
	case indice >= 1 && esResolucionFormalizacionPosterior(hitos[indice]):
		indice--
	case indice >= 2 &&
		esAnotacionAdministrativaPosterior(hitos[indice]) &&
		esResolucionFormalizacionPosterior(hitos[indice-1]):
		indice -= 2
	default:
		return ports.HitoExpedienteRRHH{}, false
	}
	hito := hitos[indice]
	return hito, hito.AccionClave == "registrar_propuesta_formalizacion"
}

func esResolucionFormalizacionPosterior(hito ports.HitoExpedienteRRHH) bool {
	return hito.AccionClave == "registrar_resolucion_formalizacion" &&
		hito.FaseOrigen == "nombramiento" && hito.FaseDestino == "nombramiento" &&
		hito.EstadoOrigen == domain.EstadoEnCurso && hito.EstadoDestino == domain.EstadoEnCurso
}

func esAnotacionAdministrativaPosterior(hito ports.HitoExpedienteRRHH) bool {
	return hito.AccionClave == domain.AccionRegistrarAnotacionAdministrativa &&
		hito.FaseOrigen == "nombramiento" && hito.FaseDestino == "nombramiento" &&
		hito.EstadoOrigen == domain.EstadoEnCurso && hito.EstadoDestino == domain.EstadoEnCurso
}
