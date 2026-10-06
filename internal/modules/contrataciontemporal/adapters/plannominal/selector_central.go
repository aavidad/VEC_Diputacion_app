package plannominal

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
)

// SelectorCentralFirmanteV2 resuelve la selección del firmante de un paso del
// plan con la fuente central (AUT56): persona del certificado, enlace del
// cargo por tipo y perfil activo con el rol del paso. El recurso histórico se
// construye con los datos exactos del material (documento original, revisión
// firmada y entrada); la decisión V3 queda ligada a ese documento. No crea
// cargos, enlaces ni asignaciones.
type SelectorCentralFirmanteV2 struct {
	fuente ports.FuenteSeleccionFirmanteV2
	motivo vd.ReferenciaEntradaCatalogo
}

var _ SelectorCentralDescriptorFirmaV2 = (*SelectorCentralFirmanteV2)(nil)

// NuevoSelectorCentralFirmanteV2 recibe el motivo de la firma del catálogo de
// motivos, de configuración privada.
func NuevoSelectorCentralFirmanteV2(fuente ports.FuenteSeleccionFirmanteV2, motivo vd.ReferenciaEntradaCatalogo) (*SelectorCentralFirmanteV2, error) {
	if nula(fuente) || motivo.Validar() != nil {
		return nil, ports.ErrCompetenciaFirmanteNoDisponible
	}
	return &SelectorCentralFirmanteV2{fuente: fuente, motivo: motivo}, nil
}

func solicitudSeleccionDelPaso(certificado string, paso ct.CompetenciaPasoFirmaV2) ports.SolicitudSeleccionFirmanteV2 {
	return ports.SolicitudSeleccionFirmanteV2{CertificadoHuella: certificado, CargoRef: paso.CargoRef, RolID: paso.RolID,
		Accion: paso.Accion, TipoRecurso: paso.TipoRecurso, Finalidad: paso.Finalidad,
		OrganizacionRef: paso.OrganizacionRef, UnidadRef: paso.UnidadRef}
}

func (s *SelectorCentralFirmanteV2) ResolverSeleccionDescriptorFirmaV2(ctx context.Context, m ports.MaterialFirmaVerificadaV2, paso ct.CompetenciaPasoFirmaV2) (SeleccionCentralDescriptorFirmaV2, error) {
	var cero SeleccionCentralDescriptorFirmaV2
	if s == nil || nula(s.fuente) || ctx == nil {
		return cero, ports.ErrCompetenciaFirmanteNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if m.Validar() != nil || paso.Validar() != nil || m.OrganizacionRef != paso.OrganizacionRef || m.UnidadFirmanteRef != paso.UnidadRef ||
		m.RolIDFirmante != paso.RolID || uint64(m.PasoOrden) != paso.PasoOrden {
		return cero, ports.ErrCompetenciaFirmanteNoAcreditada
	}
	r, err := s.fuente.SeleccionarFirmanteV2(ctx, solicitudSeleccionDelPaso(m.CertificadoHuella, paso))
	if err != nil {
		if ctx.Err() != nil {
			return cero, ctx.Err()
		}
		return cero, err
	}
	// La selección es de la persona y el perfil del material, o no vale.
	if r.PersonaRef != m.FirmantePrincipalRef || r.PerfilActivoRef != m.PerfilActivoFirmanteRef || r.RolID != paso.RolID || r.CargoRef != paso.CargoRef {
		return cero, ports.ErrCompetenciaFirmanteNoAcreditada
	}
	recurso := vd.RecursoFirmaHistoricaV1{OrganizacionRef: m.OrganizacionRef, UnidadRef: m.UnidadFirmanteRef, ExpedienteRef: m.ExpedienteRef,
		DocumentoRef: m.OriginalRef, RecursoAutorizableRef: m.OriginalRef, ModuloID: ports.ModuloContratacion, TipoRecurso: paso.TipoRecurso,
		Original:         vd.ReferenciaHistoricaCompetenciaV1{Referencia: m.OriginalRef, Version: m.OriginalVersion, HuellaSHA256: m.OriginalHuella},
		PDFRaizSHA256:    m.OriginalHuella,
		Firmado:          vd.ReferenciaHistoricaCompetenciaV1{Referencia: m.DocumentoCustodiaRef, Version: m.DocumentoCustodiaVersion, HuellaSHA256: m.FirmadoHuella},
		PDFFirmadoSHA256: m.FirmadoHuella, NumeroFirmas: uint64(m.PasoOrden)}
	if m.PasoOrden > 1 {
		recurso.EntradaRevision = &vd.ReferenciaHistoricaCompetenciaV1{Referencia: m.EntradaDocumentoRef, Version: m.EntradaDocumentoVersion,
			HuellaSHA256: m.EntradaDocumentoHuella}
	}
	huella, err := HuellaContextoRecursoFirmaV2(paso.EsquemaContexto, recurso)
	if err != nil {
		return cero, ports.ErrCompetenciaFirmanteNoAcreditada
	}
	recurso.RecursoContextoSHA256 = huella
	return SeleccionCentralDescriptorFirmaV2{
		Seleccion: ports.SeleccionConstructorFirmaV2{PerfilEsperadoRef: paso.PerfilEsperadoRef, PerfilActivoRef: r.PerfilActivoRef,
			RolID: r.RolID, CargoRef: r.CargoRef, EnlaceEjercicioRef: r.EnlaceEjercicioRef},
		Recurso: recurso, EsquemaContexto: paso.EsquemaContexto, Motivo: s.motivo}, nil
}

// HuellaContextoRecursoFirmaV2 es la recurso_contexto_sha256 del descriptor:
// SHA-256 del JSON canónico (claves ordenadas, sin espacios) del esquema de
// contexto del paso y de los datos que identifican el documento firmado.
// SQL solo valida su formato. Lo que ata la decisión al documento exacto es
// CT172, que coteja documento, recurso, original y firmado con el material,
// y la huella del descriptor, que entra en la huella de contexto de la decisión.
func HuellaContextoRecursoFirmaV2(esquema string, r vd.RecursoFirmaHistoricaV1) (string, error) {
	if esquema == "" {
		return "", ct.ErrPlanCompetenciaFirmaV2
	}
	contexto := map[string]any{"esquema_contexto": esquema, "organizacion_ref": r.OrganizacionRef, "unidad_ref": r.UnidadRef,
		"expediente_ref": r.ExpedienteRef, "documento_ref": r.DocumentoRef, "tipo_recurso": r.TipoRecurso,
		"original": r.Original, "firmado": r.Firmado, "entrada_revision": r.EntradaRevision}
	b, err := json.Marshal(contexto)
	if err != nil {
		return "", ct.ErrPlanCompetenciaFirmaV2
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}

// FuenteCompetenciaFirmantePlanV2 acredita la competencia del firmante de un
// paso con el plan publicado y la selección central (AUT56). El paso se elige
// por circuito, documento, paso, perfil y organización; si el plan tiene más
// de una unidad para esos datos, la elección es ambigua y se deniega.
type FuenteCompetenciaFirmantePlanV2 struct {
	plan   *Fuente
	fuente ports.FuenteSeleccionFirmanteV2
	reloj  func() time.Time
}

var _ ports.FuenteCompetenciaFirmante = (*FuenteCompetenciaFirmantePlanV2)(nil)

func NuevaFuenteCompetenciaFirmantePlanV2(plan *Fuente, fuente ports.FuenteSeleccionFirmanteV2, reloj func() time.Time) (*FuenteCompetenciaFirmantePlanV2, error) {
	if plan == nil || nula(fuente) || reloj == nil {
		return nil, ports.ErrCompetenciaFirmanteNoDisponible
	}
	return &FuenteCompetenciaFirmantePlanV2{plan: plan, fuente: fuente, reloj: reloj}, nil
}

func (f *FuenteCompetenciaFirmantePlanV2) AcreditarCompetenciaFirmante(ctx context.Context, q ports.SolicitudCompetenciaFirmante) (ports.EvidenciaCompetenciaFirmante, error) {
	var cero ports.EvidenciaCompetenciaFirmante
	if f == nil || f.plan == nil || nula(f.fuente) || ctx == nil {
		return cero, ports.ErrCompetenciaFirmanteNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	plan, err := f.plan.Plan(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return cero, ctx.Err()
		}
		return cero, ports.ErrCompetenciaFirmanteNoDisponible
	}
	paso, ok := pasoUnicoDeSolicitud(plan, q)
	if !ok {
		return cero, ports.ErrCompetenciaFirmanteNoAcreditada
	}
	r, err := f.fuente.SeleccionarFirmanteV2(ctx, solicitudSeleccionDelPaso(q.CertificadoHuella, paso))
	if err != nil {
		if ctx.Err() != nil {
			return cero, ctx.Err()
		}
		if errors.Is(err, ports.ErrCompetenciaFirmanteNoAcreditada) {
			return cero, err
		}
		return cero, ports.ErrCompetenciaFirmanteNoDisponible
	}
	// El adaptador ya lo coteja; se repite aquí para no depender de él.
	if r.RolID != paso.RolID || r.CargoRef != paso.CargoRef {
		return cero, ports.ErrCompetenciaFirmanteNoDisponible
	}
	desde, err1 := canonicaFechaCompetencia(r.AsignacionVigenteDesde)
	hasta, err2 := canonicaFechaCompetencia(r.AsignacionVigenteHasta)
	if err1 != nil || err2 != nil {
		return cero, ports.ErrCompetenciaFirmanteNoDisponible
	}
	return ports.EvidenciaCompetenciaFirmante{
		CuentaFirmanteRef: r.CuentaRef, VinculoCredencialFirmanteRef: r.VinculoCertificado.Referencia,
		VinculoCredencialFirmanteRevision: r.VinculoCertificado.Version, VinculoCredencialFirmanteHuella: r.VinculoCertificado.HuellaSHA256,
		RolIDFirmante: r.RolID, Solicitud: q, FirmantePrincipalRef: r.PersonaRef, PerfilFirmanteRef: q.PerfilFirmanteRef,
		CargoFirmante: r.CargoRef, UnidadFirmanteRef: paso.UnidadRef, PerfilActivoFirmanteRef: r.PerfilActivoRef,
		AsignacionFirmanteRef: r.Asignacion.Referencia, AsignacionFirmanteVersion: r.Asignacion.Version, AsignacionFirmanteHuella: r.Asignacion.HuellaSHA256,
		VersionRolFirmanteRef: r.RolRef, VersionRolFirmanteHuella: r.RolHuellaSHA256,
		ControlVigenciaFirmanteRef: r.ControlRol.Referencia, ControlVigenciaFirmanteRevision: r.ControlRol.Version,
		ControlVigenciaFirmanteHuella: r.ControlRol.HuellaSHA256,
		AsignacionVigenteDesde:        desde, AsignacionVigenteHasta: hasta,
		CompetenciaComprobadaEn: fechaCompetencia(f.reloj()),
		Vigente:                 true}, nil
}

// pasoUnicoDeSolicitud elige el paso del plan por circuito, documento, paso,
// perfil y organización. Ninguno o más de uno (otra unidad) no acreditan.
func pasoUnicoDeSolicitud(plan ct.PlanCompetenciaFirmaV2, q ports.SolicitudCompetenciaFirmante) (ct.CompetenciaPasoFirmaV2, bool) {
	circuito := ct.VersionPlanFirmaV2{Referencia: q.CatalogoRef, Version: q.CatalogoVersion, HuellaSHA256: q.CatalogoHuella}
	var paso ct.CompetenciaPasoFirmaV2
	encontrados := 0
	for _, p := range plan.Pasos {
		if p.Circuito == circuito && p.Documento == q.Documento && p.PasoRef == q.PasoRef && q.PasoOrden > 0 &&
			p.PasoOrden == uint64(q.PasoOrden) && p.PerfilEsperadoRef == q.PerfilFirmanteRef && p.OrganizacionRef == q.OrganizacionRef {
			paso = p
			encontrados++
		}
	}
	return paso, encontrados == 1
}

// canonicaFechaCompetencia convierte la fecha de la asignación a la forma que
// exigen la aplicación (ports.FechaFirmaExternaCanonica) y CT170/CT172: UTC,
// hasta microsegundos y sin ceros finales en la fracción.
func canonicaFechaCompetencia(v string) (string, error) {
	t, err := time.Parse(time.RFC3339Nano, v)
	if err != nil {
		return "", err
	}
	return fechaCompetencia(t), nil
}

func fechaCompetencia(t time.Time) string {
	return t.UTC().Truncate(time.Microsecond).Format(time.RFC3339Nano)
}
