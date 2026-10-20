package informejuridico

import (
	"context"
	"strings"

	ctapplication "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// RenderizadorBorradorDesarrollo representa la consulta ya autorizada.
// No consulta más datos, registra propuestas ni sustituye un modelo oficial.
type RenderizadorBorradorDesarrollo struct {
	PDF vecports.RenderizadorDocumento
	// OmitirReferencias muestra sólo el nombre cuando lo hay. Se activa en
	// descargas; el valor cero conserva los bytes de formalización y custodia.
	OmitirReferencias bool
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

// referenciaConNombre conserva la referencia si falta nombre. Las descargas
// pueden omitirla junto al nombre sin alterar las referencias del expediente.
func referenciaConNombre(etiquetar EtiquetadorReferencias, referencia string, omitirReferencia bool) string {
	if etiquetar == nil {
		return referencia
	}
	nombre := strings.TrimSpace(etiquetar(referencia))
	if nombre == "" {
		return referencia
	}
	if omitirReferencia {
		return nombre
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
	documento, err := contenidoBorradorDesarrollo(tipo, detalle, r.Etiquetas, r.Plantillas, r.OmitirReferencias)
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

// accionCatalogoInformeJuridico es la clave con la que el catálogo de
// plantillas (v4) liga el informe definitivo al informe jurídico vigente. El
// expediente registra esa actuación como domain.AccionEmitirInformeJuridico.
const accionCatalogoInformeJuridico = "generar_informe_juridico"

// AccionRequeridaCumplida dice si el expediente ya tiene la actuación que la
// plantilla exige para ofrecerse. Es la única traducción entre la clave del
// catálogo y la de la actuación registrada: la lista y la descarga la usan
// antes de comprobar con BorradorPreparable si el documento se puede preparar.
func AccionRequeridaCumplida(plantilla PlantillaBorrador, detalle ports.DetalleExpedienteRRHH) bool {
	if plantilla.RequiereAccion == "" {
		return true
	}
	accion := domain.ClaveCatalogo(plantilla.RequiereAccion)
	if plantilla.RequiereAccion == accionCatalogoInformeJuridico {
		accion = domain.AccionEmitirInformeJuridico
	}
	for _, hito := range detalle.Hitos {
		if hito.AccionClave == accion {
			return true
		}
	}
	return false
}

// BorradorPreparable aplica a un tipo las mismas condiciones que su
// generación (fase, propuesta, datos y plantilla) sin renderizar nada. Sirve
// para que la lista de documentos diga cuáles se pueden preparar ya.
func BorradorPreparable(
	plantillas *PlantillasBorrador, tipo ports.TipoBorradorRRHH, detalle ports.DetalleExpedienteRRHH,
) bool {
	if plantillas == nil {
		return false
	}
	_, err := contenidoBorradorDesarrollo(tipo, detalle, nil, plantillas, false)
	return err == nil
}

// contenidoBorradorDesarrollo mantiene una única fuente para todos los
// textos preparatorios: la plantilla del catálogo. PDF y DOCX sólo difieren en
// la representación final.
func contenidoBorradorDesarrollo(
	tipo ports.TipoBorradorRRHH,
	detalle ports.DetalleExpedienteRRHH,
	etiquetar EtiquetadorReferencias,
	plantillas *PlantillasBorrador,
	omitirReferencias bool,
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
	if detalle.Analisis.PeriodoFin.IsZero() &&
		!plantillas.tieneEtiqueta("fin.", string(detalle.Analisis.PeriodoCausaFin)) {
		return vecdomain.ContenidoDocumento{}, ports.ErrBorradorRRHHNoDisponible
	}
	fuente := fuenteCampos{detalle: detalle, etiquetar: etiquetar, plantillas: plantillas, plantilla: plantilla, omitirReferencias: omitirReferencias}
	if tipo == ports.BorradorInformeDefinitivo && plantilla.RequiereAccion == accionCatalogoInformeJuridico {
		hito, vigente := ctapplication.HitoInformeVigenteParaOriginalRRHH(detalle)
		if !vigente {
			return vecdomain.ContenidoDocumento{}, ports.ErrBorradorRRHHNoDisponible
		}
		fuente.requerido = &hito
		return plantilla.Rellenar(fuente.valores())
	}
	if plantilla.RequiereAccion == "" {
		if detalle.Resumen.FaseClave != "nombramiento" || detalle.Resumen.EstadoClave != domain.EstadoEnCurso {
			return vecdomain.ContenidoDocumento{}, ports.ErrBorradorRRHHNoDisponible
		}
		fuente.propuesta, ok = ctapplication.PropuestaFormalizacionBorrador(detalle.Hitos)
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
