package informejuridico

import (
	"context"
	"fmt"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// RenderizadorInformeDefinitivoDesarrollo representa la consulta ya autorizada.
// No consulta más datos, registra propuestas ni sustituye un modelo oficial.
type RenderizadorInformeDefinitivoDesarrollo struct {
	PDF vecports.RenderizadorDocumento
}

func (r RenderizadorInformeDefinitivoDesarrollo) RenderizarInforme(
	ctx context.Context, detalle ports.DetalleExpedienteRRHH,
) ([]byte, error) {
	if ctx == nil || r.PDF == nil || r.PDF.Formato() != vecdomain.FormatoDocumentoPDF {
		return nil, ports.ErrConsultaRRHHNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	solicitud, err := ports.NuevaSolicitudDetalleRRHH(detalle.Resumen.ExpedienteRef, 7)
	if err != nil || detalle.ValidarContenidoPublicablePara(solicitud) != nil ||
		detalle.Resumen.FaseClave != "nombramiento" || detalle.Resumen.EstadoClave != domain.EstadoEnCurso ||
		detalle.Analisis == nil || detalle.Cobertura == nil || detalle.Asignacion == nil || len(detalle.Hitos) != 7 {
		return nil, ports.ErrInformeDefinitivoRRHHNoDisponible
	}
	hito := detalle.Hitos[6]
	if hito.VersionExpediente != 7 || hito.AccionClave != "registrar_propuesta_formalizacion" ||
		hito.FaseDestino != "nombramiento" || hito.EstadoDestino != domain.EstadoEnCurso {
		return nil, ports.ErrInformeDefinitivoRRHHNoDisponible
	}
	contenido, err := r.PDF.Renderizar(ctx, contenidoInformeDefinitivoDesarrollo(detalle))
	if err != nil {
		return nil, err
	}
	if err := r.PDF.ValidarSalida(ctx, contenido); err != nil {
		return nil, err
	}
	return contenido, ctx.Err()
}

func contenidoInformeDefinitivoDesarrollo(d ports.DetalleExpedienteRRHH) vecdomain.ContenidoDocumento {
	r, a := d.Resumen, d.Analisis
	parrafos := []string{
		"BORRADOR PREPARATORIO DE DESARROLLO — NO FIRMADO NI VALIDADO. Datos sintéticos. No es una resolución, un nombramiento efectivo ni una redacción jurídica aprobada por Recursos Humanos.",
		fmt.Sprintf("Expediente: %s\nReferencia: %s\nVersión de origen: %d · Fase: nombramiento en curso", r.NumeroVisible, r.ExpedienteRef, r.Version),
		"1. Necesidad y análisis registrados",
		fmt.Sprintf("Centro (referencia): %s\nCategoría (referencia): %s\nGrupo/subgrupo: %s\nModalidad registrada: %s", r.CentroRef, r.CategoriaRef, d.Solicitud.GrupoSubgrupo, modalidadInformeDefinitivo(a.ModalidadClave)),
		fmt.Sprintf("Periodo previsto: del %s al %s. Jornada registrada: %d,%02d %%.\nResultado registrado de retención de crédito: %s.", a.PeriodoInicio.Format("02/01/2006"), a.PeriodoFin.Format("02/01/2006"), a.PorcentajeJornada/100, a.PorcentajeJornada%100, a.ResultadoRC),
	}
	if a.CostePrevisto != nil {
		parrafos = append(parrafos, fmt.Sprintf("Coste previsto registrado: %d,%02d %s. Esta descarga no recalcula ni autoriza gasto.", a.CostePrevisto.Centimos/100, a.CostePrevisto.Centimos%100, a.CostePrevisto.Moneda))
	}
	parrafos = append(parrafos,
		"2. Tramitación registrada",
		fmt.Sprintf("Vía de cobertura (clave registrada): %s. Unidad asignada (referencia): %s. Asignación registrada el %s UTC.", d.Cobertura.ViaClave, d.Asignacion.UnidadRef, d.Asignacion.AsignadaEn.UTC().Format(time.RFC3339Nano)),
		fmt.Sprintf("El historial del expediente contiene el registro de la propuesta de formalización, actuación 7, de %s UTC. Esta consulta no incorpora identidad de la candidatura, contenido del correo, recibos internos ni documentos firmados.", d.Hitos[6].RealizadaEn.UTC().Format(time.RFC3339Nano)),
		"3. Pendiente de completar y validar",
		"Deben incorporarse el modelo oficial y la redacción jurídica competente, las comprobaciones y referencias documentales que correspondan, la identificación autorizada de la persona propuesta y las firmas requeridas. Los campos ausentes no se han inventado. No se certifica el resultado jurídico o de fiscalización mediante esta descarga.",
		"Esta copia se regenera desde el detalle persistido y autorizado del expediente. No guarda un documento firmado, no acredita custodia documental, no modifica el expediente y no realiza ningún envío. BORRADOR DE DESARROLLO SIN EFECTOS ADMINISTRATIVOS.",
	)
	return vecdomain.ContenidoDocumento{Titulo: "Informe definitivo — borrador de desarrollo", Parrafos: parrafos}
}

func modalidadInformeDefinitivo(clave domain.ClaveCatalogo) string {
	switch clave {
	case "sustitucion":
		return "Sustitución"
	case "vacante":
		return "Vacante"
	case "acumulacion_tareas":
		return "Acumulación de tareas"
	case "programa":
		return "Programa"
	case "relevo":
		return "Relevo"
	default:
		return string(clave)
	}
}
