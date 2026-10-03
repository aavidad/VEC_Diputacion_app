package application

import (
	"context"
	"time"

	meritos "vec-diputacion-granada/internal/modules/meritos/domain"
	meritosports "vec-diputacion-granada/internal/modules/meritos/ports"
	"vec-diputacion-granada/internal/modules/seleccion/domain"
	"vec-diputacion-granada/internal/modules/seleccion/ports"
)

// PrepararAdmision prepara una lista de revisión sintética. Nunca consulta
// identidad, RUM nominal o registro ni transforma el resumen local S3 en acto.
// La correspondencia y el universo exacto de requisitos quedan pendientes.
func PrepararAdmision(ctx context.Context, m ports.MaterialAdmisionPreparacion) (domain.PreparacionAdmision, error) {
	cero := domain.PreparacionAdmision{}
	if ctx == nil || m.Alcance != domain.AlcanceAdmisionPreparacion ||
		!meritos.ReferenciaValida(m.PreparacionRef) || m.Revision < 1 || m.Revision > 1_000_000 ||
		len(m.Requisitos) > 64 || domain.ValidarBasesAdmision(m.Bases, m.SolicitudContexto) != nil {
		return cero, domain.ErrAdmisionPreparacion
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	exigidos := map[string]int{}
	requisitos := map[string]bool{}
	for _, r := range m.Requisitos {
		if requisitos[r.Referencia] {
			return cero, domain.ErrAdmisionPreparacion
		}
		requisitos[r.Referencia] = true
		if _, err := domain.RevisarRequisitoAdmision(r, nil); err != nil {
			return cero, err
		}
		for _, h := range r.HechosEsperados {
			if previa := exigidos[h.Referencia]; previa != 0 && previa != h.Version {
				return cero, domain.ErrAdmisionPreparacion
			}
			exigidos[h.Referencia] = h.Version
		}
	}
	hechos := map[string]domain.SoporteAdmision{}
	versionPaquete := ""
	if m.Hechos != nil {
		if m.Hechos.Alcance != meritos.AlcanceSintetico || !meritos.ReferenciaValida(m.Hechos.VersionPaquete) || len(m.Hechos.Hechos) > 128 {
			return cero, domain.ErrAdmisionPreparacion
		}
		versionPaquete = m.Hechos.VersionPaquete
		for _, h := range m.Hechos.Hechos {
			if _, repetido := hechos[h.Referencia]; repetido || exigidos[h.Referencia] != h.Version {
				return cero, domain.ErrAdmisionPreparacion
			}
			if err := validarHechoAdmision(h); err != nil {
				return cero, err
			}
			hechos[h.Referencia] = domain.SoporteAdmision{
				ReferenciaHechoAdmision: domain.ReferenciaHechoAdmision{Referencia: h.Referencia, Version: h.Version},
				FuenteRef:               h.Procedencia.FuenteRef, FuenteVersion: h.Procedencia.Version, EstadoAportado: string(h.Estado),
				EvidenciasAportadas: len(h.Evidencias), Desde: h.Vigencia.Desde, Hasta: h.Vigencia.Hasta,
			}
		}
	}
	out := domain.PreparacionAdmision{
		Esquema: "vec.seleccion.admision-preparacion.v1", PreparacionRef: m.PreparacionRef, Revision: m.Revision,
		Alcance: m.Alcance, Estado: "pendiente_revision_competente", UniversoRequisitos: "propuesto_no_cotejado", Bases: m.Bases,
		VersionPaqueteHechos: versionPaquete, Requisitos: []domain.RevisionRequisitoAdmision{},
		Pendientes: []string{"seleccion.admision.pendiente.bases_universo", "seleccion.admision.pendiente.correspondencia_solicitud",
			"seleccion.admision.pendiente.lectura_autorizada", "seleccion.admision.pendiente.identidad_registro_documentos",
			"seleccion.admision.pendiente.subsanacion_bases", "seleccion.admision.pendiente.aprobacion_listas"},
	}
	if m.SolicitudContexto != nil {
		copia := *m.SolicitudContexto
		out.SolicitudContexto = &copia
	}
	for _, r := range m.Requisitos {
		soportes := []domain.SoporteAdmision{}
		for _, h := range r.HechosEsperados {
			if aportado, ok := hechos[h.Referencia]; ok {
				soportes = append(soportes, aportado)
			}
		}
		revision, err := domain.RevisarRequisitoAdmision(r, soportes)
		if err != nil {
			return cero, err
		}
		out.Requisitos = append(out.Requisitos, revision)
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	return out, nil
}

// Comprueba estructura del DTO RUM sin inventar Persona, actor de revisión ni
// acreditación para reconstruir un Hecho completo. Ningún campo es autoridad.
func validarHechoAdmision(h meritosports.HechoPreparado) error {
	if !meritos.ReferenciaValida(h.Referencia) || h.Version < 1 || !meritos.ReferenciaValida(h.ConceptoRef) ||
		!meritos.ReferenciaValida(h.Procedencia.FuenteRef) || !meritos.ReferenciaValida(h.Procedencia.Version) ||
		!meritos.ReferenciaValida(h.Procedencia.HechoOrigenRef) || len(h.Procedencia.CapturadaEn) > 40 ||
		len(h.Evidencias) > 32 || len(h.Pendientes) == 0 || len(h.Pendientes) > 32 {
		return domain.ErrAdmisionPreparacion
	}
	instante, err := time.Parse(time.RFC3339Nano, h.Procedencia.CapturadaEn)
	if err != nil || instante.Year() < 1 {
		return domain.ErrAdmisionPreparacion
	}
	switch h.Tipo {
	case "titulacion", "curso_asistencia", "curso_superacion", "experiencia", "idioma", "otro":
	default:
		return domain.ErrAdmisionPreparacion
	}
	if h.Horas != nil && (*h.Horas < 0 || (h.Tipo != "curso_asistencia" && h.Tipo != "curso_superacion")) {
		return domain.ErrAdmisionPreparacion
	}
	for i, ev := range h.Evidencias {
		if ev.Validar() != nil || !meritos.ReferenciaValida(ev.ID) {
			return domain.ErrAdmisionPreparacion
		}
		for _, anterior := range h.Evidencias[:i] {
			if ev == anterior {
				return domain.ErrAdmisionPreparacion
			}
		}
	}
	for _, p := range h.Pendientes {
		if !meritos.ReferenciaValida(p) {
			return domain.ErrAdmisionPreparacion
		}
	}
	return nil // Vigencia y estado se validan en el dominio antes del resultado.
}
