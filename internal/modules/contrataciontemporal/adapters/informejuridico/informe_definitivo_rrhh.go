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

// RenderizadorBorradorDesarrollo representa la consulta ya autorizada.
// No consulta más datos, registra propuestas ni sustituye un modelo oficial.
type RenderizadorBorradorDesarrollo struct {
	PDF vecports.RenderizadorDocumento
}

func (r RenderizadorBorradorDesarrollo) RenderizarBorrador(
	ctx context.Context, tipo ports.TipoBorradorRRHH, detalle ports.DetalleExpedienteRRHH,
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
		return nil, ports.ErrBorradorRRHHNoDisponible
	}
	hito := detalle.Hitos[6]
	if hito.VersionExpediente != 7 || hito.AccionClave != "registrar_propuesta_formalizacion" ||
		hito.FaseDestino != "nombramiento" || hito.EstadoDestino != domain.EstadoEnCurso {
		return nil, ports.ErrBorradorRRHHNoDisponible
	}
	var documento vecdomain.ContenidoDocumento
	switch tipo {
	case ports.BorradorInformeDefinitivo:
		documento = contenidoInformeDefinitivoDesarrollo(detalle)
	case ports.BorradorResolucion:
		documento = contenidoResolucionDesarrollo(detalle)
	case ports.BorradorDiligencia:
		documento = contenidoDiligenciaDesarrollo(detalle)
	case ports.BorradorTomaPosesion:
		documento = contenidoTomaPosesionDesarrollo(detalle)
	default:
		return nil, ports.ErrBorradorRRHHNoDisponible
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

func contenidoResolucionDesarrollo(d ports.DetalleExpedienteRRHH) vecdomain.ContenidoDocumento {
	r, a := d.Resumen, d.Analisis
	return vecdomain.ContenidoDocumento{Titulo: "Resolución — borrador de desarrollo", Parrafos: []string{
		"BORRADOR PREPARATORIO DE DESARROLLO — NO FIRMADO NI VALIDADO. Datos sintéticos. No constituye una resolución aprobada, un nombramiento efectivo ni una orden de incorporación.",
		fmt.Sprintf("Expediente: %s\nReferencia: %s\nVersión de origen: %d · Fase: nombramiento en curso", r.NumeroVisible, r.ExpedienteRef, r.Version),
		"1. Antecedentes disponibles",
		fmt.Sprintf("La propuesta de formalización figura en el historial del expediente, actuación 7, de %s UTC. Este borrador se prepara desde ese detalle persistido y autorizado; no añade una nueva propuesta.", d.Hitos[6].RealizadaEn.UTC().Format(time.RFC3339Nano)),
		fmt.Sprintf("Centro (referencia): %s\nCategoría (referencia): %s\nGrupo/subgrupo: %s\nModalidad registrada: %s", r.CentroRef, r.CategoriaRef, d.Solicitud.GrupoSubgrupo, modalidadInformeDefinitivo(a.ModalidadClave)),
		fmt.Sprintf("Periodo previsto de la necesidad: del %s al %s. Jornada registrada: %d,%02d %%. Estos datos no fijan la fecha de efectos de un nombramiento.", a.PeriodoInicio.Format("02/01/2006"), a.PeriodoFin.Format("02/01/2006"), a.PorcentajeJornada/100, a.PorcentajeJornada%100),
		"2. Contenido resolutivo pendiente",
		"Órgano competente: pendiente de determinar y validar. Persona propuesta: identificación autorizada pendiente de incorporar. No se infieren ni se asignan nombres, competencias o firmantes desde esta consulta.",
		"Acuerdos, fundamento jurídico, fecha de efectos, recursos y destinatarios: pendientes del modelo oficial y de la validación competente. No se inventa redacción jurídica ni se declara aprobado ningún acuerdo.",
		"3. Firma y efectos pendientes",
		"Número y fecha de resolución, firmas y evidencia de validación: pendientes. La descarga no acredita firma, aprobación, custodia documental, notificación ni toma de posesión; tampoco modifica el expediente o envía comunicaciones.",
		"BORRADOR DE DESARROLLO SIN EFECTOS ADMINISTRATIVOS. Debe completarse y validarse por el circuito competente antes de cualquier uso real.",
	}}
}

func contenidoDiligenciaDesarrollo(d ports.DetalleExpedienteRRHH) vecdomain.ContenidoDocumento {
	r := d.Resumen
	return vecdomain.ContenidoDocumento{Titulo: "Diligencia — borrador de desarrollo", Parrafos: []string{
		"BORRADOR PREPARATORIO DE DESARROLLO — NO FIRMADO NI VALIDADO. Datos sintéticos. No es una diligencia extendida por una persona competente ni certifica un hecho administrativo.",
		fmt.Sprintf("Expediente: %s\nReferencia: %s\nVersión de origen: %d · Fase: nombramiento en curso", r.NumeroVisible, r.ExpedienteRef, r.Version),
		"1. Referencias disponibles para su preparación",
		fmt.Sprintf("Centro (referencia): %s\nCategoría (referencia): %s\nUnidad asignada (referencia): %s", r.CentroRef, r.CategoriaRef, d.Asignacion.UnidadRef),
		fmt.Sprintf("El historial del expediente registra la propuesta de formalización, actuación 7, de %s UTC. Es la fecha de esa actuación, no la fecha de una comparecencia, firma o notificación.", d.Hitos[6].RealizadaEn.UTC().Format(time.RFC3339Nano)),
		"2. Objeto y hechos pendientes de incorporar",
		"Objeto específico de la diligencia: pendiente del modelo oficial y de la validación competente. Hechos que deban hacerse constar, documentación que los acredite y fecha y lugar de realización: pendientes. No se infieren de la propuesta de nombramiento.",
		"Comparecencia e identificación autorizada de las personas intervinientes, cuando correspondan: pendientes. No se afirma que ninguna persona haya comparecido, firmado, recibido una notificación o tomado posesión.",
		"3. Autoría y validación pendientes",
		"Órgano y persona competente para extender la diligencia, fecha, firma y evidencia de validación: pendientes. Este documento no sustituye esas comprobaciones ni acredita la custodia de sus justificantes.",
		"Copia preparatoria obtenida del detalle persistido y autorizado. Su descarga no registra hechos, modifica el expediente ni realiza envíos. BORRADOR DE DESARROLLO SIN EFECTOS ADMINISTRATIVOS.",
	}}
}

func contenidoTomaPosesionDesarrollo(d ports.DetalleExpedienteRRHH) vecdomain.ContenidoDocumento {
	r, a := d.Resumen, d.Analisis
	return vecdomain.ContenidoDocumento{Titulo: "Toma de posesión — borrador de desarrollo", Parrafos: []string{
		"BORRADOR PREPARATORIO DE DESARROLLO — NO FIRMADO NI VALIDADO. Datos sintéticos. No acredita una toma de posesión, un nombramiento eficaz ni una incorporación al puesto.",
		fmt.Sprintf("Expediente: %s\nReferencia: %s\nVersión de origen: %d · Fase: nombramiento en curso", r.NumeroVisible, r.ExpedienteRef, r.Version),
		"1. Datos disponibles del expediente",
		fmt.Sprintf("Centro (referencia): %s\nCategoría (referencia): %s\nGrupo/subgrupo: %s\nModalidad registrada: %s", r.CentroRef, r.CategoriaRef, d.Solicitud.GrupoSubgrupo, modalidadInformeDefinitivo(a.ModalidadClave)),
		fmt.Sprintf("Periodo previsto de la necesidad: del %s al %s. Jornada registrada: %d,%02d %%. No se utiliza este periodo como fecha efectiva de posesión o incorporación.", a.PeriodoInicio.Format("02/01/2006"), a.PeriodoFin.Format("02/01/2006"), a.PorcentajeJornada/100, a.PorcentajeJornada%100),
		fmt.Sprintf("Antecedente: propuesta de formalización registrada en el historial, actuación 7, de %s UTC. Esa fecha no acredita comparecencia ni toma de posesión.", d.Hitos[6].RealizadaEn.UTC().Format(time.RFC3339Nano)),
		"2. Comparecencia y formalización pendientes",
		"Identificación autorizada de la persona interesada y de la persona competente que intervenga: pendientes. Resolución de nombramiento válida y su evidencia de firma: pendientes de incorporar y comprobar; una propuesta no las sustituye.",
		"Lugar, fecha y hora de comparecencia; manifestaciones, juramento o promesa cuando correspondan según el modelo oficial: pendientes. No se afirma que estos hechos hayan ocurrido ni se inventa su redacción.",
		"3. Firmas e incorporación pendientes",
		"Firmas, evidencia de validación, fecha efectiva de toma de posesión y confirmación de incorporación: pendientes. Deben completarse mediante el circuito competente, con el modelo oficial y los hechos acreditados.",
		"Copia preparatoria del detalle persistido y autorizado. Su descarga no registra la posesión, modifica el expediente, da de alta en Personal ni envía información a GINPIX. BORRADOR DE DESARROLLO SIN EFECTOS ADMINISTRATIVOS.",
	}}
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
