package plannominal

import (
	"bytes"
	"context"
	"math"

	firma "vec-diputacion-granada/internal/modules/contrataciontemporal/application/firmaautorizacionv2"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
)

// SelectorCentralDescriptorFirmaV2 resuelve una selección única autorizada
// desde identidad, AUT, Personal y el recurso propietario de CT. Debe cotejar
// el firmante y certificado del material, su perfil activo, el cargo del plan
// y su enlace de ejercicio existente. El material y el paso no conceden acceso.
// Sin estas autoridades la implementación debe devolver un error; no publica
// mapeos, crea cargos ni fabrica enlaces para un documento original.
type SelectorCentralDescriptorFirmaV2 interface {
	ResolverSeleccionDescriptorFirmaV2(context.Context, ports.MaterialFirmaVerificadaV2, ct.CompetenciaPasoFirmaV2) (SeleccionCentralDescriptorFirmaV2, error)
}

// El selector entrega el recurso y motivo ya autorizados. EsquemaContexto
// identifica la definición gobernada usada para obtener la huella del recurso;
// no es la huella del efecto V3. CT172/AUT35 revalidan las fuentes en su TX.
type SeleccionCentralDescriptorFirmaV2 struct {
	Seleccion       ports.SeleccionConstructorFirmaV2
	Recurso         vd.RecursoFirmaHistoricaV1
	EsquemaContexto string
	Motivo          vd.ReferenciaEntradaCatalogo
}

type FuenteDescriptorFirmaV2 struct {
	plan     *Fuente
	selector SelectorCentralDescriptorFirmaV2
}

var _ ports.FuenteDescriptorFirmaV2 = (*FuenteDescriptorFirmaV2)(nil)
var _ ports.FuenteDescriptorPlanFijadoFirmaV2 = (*FuenteDescriptorFirmaV2)(nil)

func NuevaFuenteDescriptorFirmaV2(plan *Fuente, selector SelectorCentralDescriptorFirmaV2) (*FuenteDescriptorFirmaV2, error) {
	if plan == nil || plan.resolutor == nil || nula(plan.publicacion) || !plan.version.Valida() || nula(selector) {
		return nil, ports.ErrCompetenciaFirmanteNoDisponible
	}
	return &FuenteDescriptorFirmaV2{plan: plan, selector: selector}, nil
}

// DescriptorFirmaV2 relee la publicación en cada solicitud y fija todos los
// datos del paso antes del PDP. No conserva una selección positiva anterior.
// Esta lectura no sustituye el pin y la revalidación del plan publicado en la
// transacción final; su composición operativa requiere esa dependencia.
func (f *FuenteDescriptorFirmaV2) DescriptorFirmaV2(ctx context.Context, m ports.MaterialFirmaVerificadaV2) (ports.DescriptorConstructorFirmaV2, error) {
	d, err := f.DescriptorPlanFijadoFirmaV2(ctx, m)
	return d.Descriptor, err
}

// DescriptorPlanFijadoFirmaV2 devuelve el pin de la misma lectura y selección
// que produjo el descriptor. No reconstruye una entrada desde fuentes actuales.
func (f *FuenteDescriptorFirmaV2) DescriptorPlanFijadoFirmaV2(ctx context.Context, m ports.MaterialFirmaVerificadaV2) (ports.DescriptorPlanFijadoFirmaV2, error) {
	var cero ports.DescriptorPlanFijadoFirmaV2
	if ctx == nil || f == nil || f.plan == nil || nula(f.selector) {
		return cero, ports.ErrCompetenciaFirmanteNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	// La cota local protege también la conversión al orden uint64 del plan.
	if m.Validar() != nil || m.PasoOrden < 1 || m.PasoOrden > 2 {
		return cero, ports.ErrSolicitudFirmaDocumentoInvalida
	}
	plan, err := f.plan.Plan(ctx)
	if err != nil {
		return cero, errorDescriptor(ctx, ports.ErrCompetenciaFirmanteNoDisponible)
	}
	if plan.Version.Version > math.MaxInt {
		return cero, ports.ErrCompetenciaFirmanteNoAcreditada
	}
	circuito := ct.VersionPlanFirmaV2{Referencia: m.CatalogoRef, Version: m.CatalogoVersion, HuellaSHA256: m.CatalogoHuella}
	paso, err := plan.Seleccionar(circuito, m.Documento, m.PasoRef, m.PerfilFirmanteRef, m.OrganizacionRef, m.UnidadFirmanteRef)
	if err != nil || paso.PasoOrden != uint64(m.PasoOrden) || paso.RolID != m.RolIDFirmante {
		return cero, ports.ErrCompetenciaFirmanteNoAcreditada
	}
	// El selector no puede modificar los bytes que luego se cotejan con el
	// descriptor. El plan contiene valores sin referencias mutables.
	solicitud := m
	solicitud.EvidenciaFirmasCanonica = bytes.Clone(m.EvidenciaFirmasCanonica)
	seleccion, err := f.selector.ResolverSeleccionDescriptorFirmaV2(ctx, solicitud, paso)
	clear(solicitud.EvidenciaFirmasCanonica)
	if err != nil {
		return cero, errorDescriptor(ctx, ports.ErrCompetenciaFirmanteNoDisponible)
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	s := seleccion.Seleccion
	if s.PerfilEsperadoRef != paso.PerfilEsperadoRef || s.PerfilActivoRef != m.PerfilActivoFirmanteRef ||
		s.RolID != paso.RolID || s.CargoRef != paso.CargoRef ||
		seleccion.Recurso.TipoRecurso != paso.TipoRecurso || seleccion.EsquemaContexto != paso.EsquemaContexto {
		return cero, ports.ErrCompetenciaFirmanteNoAcreditada
	}
	r := seleccion.Recurso
	if r.EntradaRevision != nil {
		entrada := *r.EntradaRevision
		r.EntradaRevision = &entrada
	}
	d := ports.DescriptorConstructorFirmaV2{
		Esquema: "vec.competencia-firmante.constructor-ct.v1", CertificadoDERSHA256: m.CertificadoHuella,
		Seleccion: s, Recurso: r, Accion: paso.Accion, Finalidad: paso.Finalidad, Motivo: seleccion.Motivo,
		Circuito: vd.ReferenciaHistoricaCompetenciaV1{Referencia: paso.Circuito.Referencia, Version: paso.Circuito.Version, HuellaSHA256: paso.Circuito.HuellaSHA256},
		PasoRef:  paso.PasoRef, PasoOrden: paso.PasoOrden,
	}
	canon, err := firma.CanonicoDescriptorFirmaVerificadaV2(m, d)
	clear(canon)
	if err != nil {
		return cero, ports.ErrCompetenciaFirmanteNoAcreditada
	}
	resultado := ports.DescriptorPlanFijadoFirmaV2{
		Descriptor: d,
		Plan: vd.ReferenciaEntradaCatalogo{CatalogoID: plan.Version.Referencia, CatalogoVersion: int(plan.Version.Version),
			CatalogoHuellaSHA256: plan.Version.HuellaSHA256, EntradaClave: paso.EntradaClave},
	}
	if resultado.Plan.Validar() != nil {
		return cero, ports.ErrCompetenciaFirmanteNoAcreditada
	}
	return resultado, nil
}

func errorDescriptor(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}
