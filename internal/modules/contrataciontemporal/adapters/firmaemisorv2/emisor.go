// Package firmaemisorv2 adapta la emisión común V3 al registro nominal de CT.
package firmaemisorv2

import (
	"bytes"
	"context"
	"errors"
	"maps"
	"reflect"

	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	vecapp "vec-diputacion-granada/internal/vec/application"
	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// ContextoActorFirmaV2 procede de la sesión y del canal revalidados. La huella
// del certificado del canal nunca se obtiene del material del documento.
type ContextoActorFirmaV2 struct {
	Resultado              vd.ResultadoContextoActorRegistradoV2
	Vinculo                vd.VinculoAutenticacionActorV2
	CertificadoCanalSHA256 string
}

// FuenteContextoActorFirmaV2 debe consultar las autoridades comunes de sesión
// y contexto registrado en cada invocación, con sus lecturas nominales auditadas,
// sin datos HTTP; un perfil fijo sólo vale como asignación publicada y
// consumida tal cual. Revalidar no concede permiso de registro.
type FuenteContextoActorFirmaV2 interface {
	RevalidarContextoActorFirmaV2(context.Context) (ContextoActorFirmaV2, error)
}

// EmisorComunV3 es el emisor existente compuesto con PDP, registro, firmante
// y verificador. La composición conecta su implementación nominal.
type EmisorComunV3 interface {
	EmitirMaterialAutorizacionAtestadaV3(context.Context, vd.SolicitudAutorizacionLigadaV3, vd.ResultadoContextoActorRegistradoV2) (vd.DecisionAutorizacionLigadaV3, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vp.ExportadorMaterialConsumoAutorizacionAtestadaV3, error)
}

type Emisor struct {
	fuente FuenteContextoActorFirmaV2
	emisor EmisorComunV3
	motivo vd.ReferenciaEntradaCatalogo
	reloj  vd.RelojVinculoAutenticacionActorV2
	// autorizacion lee la asignación vigente del perfil activo. Sin ella el
	// emisor no autoriza firmas: sólo consultas y recuperaciones.
	autorizacion vp.FuenteAutorizacion
	// Sólo el constructor DEV instala esta admisión; la vía corporativa
	// conserva su requisito HIGH y no consume excepciones de desarrollo.
	admision *vecapp.AdmisionGarantiaFirmaVecDesarrollo
}

// NuevoEmisor recibe el motivo gobernado y el reloj común en composición.
// No configura credenciales, audiencias, perfiles ni autoridades alternativas.
func NuevoEmisor(f FuenteContextoActorFirmaV2, e EmisorComunV3, motivo vd.ReferenciaEntradaCatalogo, reloj vd.RelojVinculoAutenticacionActorV2) (*Emisor, error) {
	if nulo(f) || nulo(e) || nulo(reloj) || !vd.ReferenciaMotivoAutorizacionV2Valida(motivo) {
		return nil, ports.ErrCompetenciaFirmanteNoDisponible
	}
	return &Emisor{fuente: f, emisor: e, motivo: motivo, reloj: reloj}, nil
}

// NuevoEmisorConAmbitos compone además la fuente común de autorización, de la
// que salen los ámbitos del recurso de firma (los de la asignación vigente).
func NuevoEmisorConAmbitos(f FuenteContextoActorFirmaV2, e EmisorComunV3, motivo vd.ReferenciaEntradaCatalogo, reloj vd.RelojVinculoAutenticacionActorV2, autorizacion vp.FuenteAutorizacion) (*Emisor, error) {
	if nulo(autorizacion) {
		return nil, ports.ErrCompetenciaFirmanteNoDisponible
	}
	x, err := NuevoEmisor(f, e, motivo, reloj)
	if err != nil {
		return nil, err
	}
	x.autorizacion = autorizacion
	return x, nil
}

// NuevoEmisorFirmaVecDesarrollo exige la admisión común del acto VEC. No
// concede permiso: V3 y su consumidor durable conservan sus guardas propias.
func NuevoEmisorFirmaVecDesarrollo(f FuenteContextoActorFirmaV2, e EmisorComunV3,
	motivo vd.ReferenciaEntradaCatalogo, reloj vd.RelojVinculoAutenticacionActorV2,
	autorizacion vp.FuenteAutorizacion, admision *vecapp.AdmisionGarantiaFirmaVecDesarrollo,
) (*Emisor, error) {
	if admision == nil {
		return nil, ports.ErrCompetenciaFirmanteNoDisponible
	}
	x, err := NuevoEmisorConAmbitos(f, e, motivo, reloj, autorizacion)
	if err != nil {
		return nil, err
	}
	x.admision = admision
	return x, nil
}

func (e *Emisor) contexto(ctx context.Context) (ContextoActorFirmaV2, error) {
	return e.contextoParaActo(ctx, false)
}

// Sólo el registro VEC de desarrollo puede intentar la garantía temporal.
// Consulta, recuperación y lecturas previas conservan la frontera corporativa.
func (e *Emisor) contextoParaActo(ctx context.Context, firmaVecDesarrollo bool) (ContextoActorFirmaV2, error) {
	var cero ContextoActorFirmaV2
	if e == nil || ctx == nil || nulo(e.fuente) || nulo(e.emisor) || nulo(e.reloj) || !vd.ReferenciaMotivoAutorizacionV2Valida(e.motivo) {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	if ctx.Err() != nil {
		return cero, opaco(ctx, nil)
	}
	base, err := e.fuente.RevalidarContextoActorFirmaV2(ctx)
	if err != nil || ctx.Err() != nil {
		return cero, opaco(ctx, err)
	}
	base.Resultado, err = base.Resultado.Clonar()
	if err != nil || !base.Vinculo.VigenteEn(e.reloj.Ahora(), base.Resultado) || !ctdomain.HuellaSHA256FirmaValida(base.CertificadoCanalSHA256) {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	d, err := base.Vinculo.Datos()
	if err != nil || d.Superficie != vd.SuperficieAutenticacionInternaCorporativaV1 || d.CuentaPrivilegiada ||
		d.MetodoObservado != vd.AuthMethodCertificate ||
		(e.admision == nil && d.GarantiaObservada != vd.AuthAssuranceHigh) ||
		(e.admision != nil && (!firmaVecDesarrollo || d.GarantiaObservada != vd.AuthAssuranceSubstantial)) {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	if ctx.Err() != nil {
		return cero, opaco(ctx, nil)
	}
	return base, nil
}

func (e *Emisor) ObtenerPerfilActivoOperadorFirmaV2(ctx context.Context) (string, error) {
	base, err := e.contexto(ctx)
	if err != nil {
		return "", err
	}
	return base.Resultado.Contexto.PerfilActivoRef, nil
}

// ObtenerAmbitosOperadorFirmaV2 devuelve los ámbitos de la asignación vigente
// del perfil activo revalidado.
func (e *Emisor) ObtenerAmbitosOperadorFirmaV2(ctx context.Context) (ports.AmbitosOperadorFirmaV2, error) {
	base, err := e.contexto(ctx)
	if err != nil {
		return ports.AmbitosOperadorFirmaV2{}, err
	}
	return e.ambitosAsignacion(ctx, base)
}

// ambitosAsignacion sólo admite organización y, como mucho, unidad, cada una
// con un único valor: cualquier otra forma no puede ser un recurso de firma.
func (e *Emisor) ambitosAsignacion(ctx context.Context, base ContextoActorFirmaV2) (ports.AmbitosOperadorFirmaV2, error) {
	a, _, err := e.ambitosConInstantanea(ctx, base)
	return a, err
}

// La admisión DEV consume la misma captura usada para resolver los ámbitos;
// esta función realiza una sola lectura de autorización.
func (e *Emisor) ambitosConInstantanea(ctx context.Context, base ContextoActorFirmaV2) (ports.AmbitosOperadorFirmaV2, vd.InstantaneaAutorizacion, error) {
	var cero ports.AmbitosOperadorFirmaV2
	var sinInstantanea vd.InstantaneaAutorizacion
	if nulo(e.autorizacion) {
		return cero, sinInstantanea, ports.ErrFirmaDocumentoDenegada
	}
	d, err := base.Vinculo.Datos()
	if err != nil {
		return cero, sinInstantanea, ports.ErrFirmaDocumentoDenegada
	}
	s, err := e.autorizacion.ObtenerInstantaneaAutorizacion(ctx, d.PrincipalID, d.PerfilActivoRef)
	if err != nil || ctx.Err() != nil {
		return cero, sinInstantanea, opaco(ctx, err)
	}
	a := s.AsignacionPerfil
	if s.Validar() != nil || a.PrincipalID != d.PrincipalID || a.PerfilActivoRef != d.PerfilActivoRef || !a.VigenteEn(e.reloj.Ahora()) ||
		len(a.Ambitos) < 1 || len(a.Ambitos) > 2 {
		return cero, sinInstantanea, ports.ErrFirmaDocumentoDenegada
	}
	var r ports.AmbitosOperadorFirmaV2
	for _, ambito := range a.Ambitos {
		if len(ambito.Valores) != 1 {
			return cero, sinInstantanea, ports.ErrFirmaDocumentoDenegada
		}
		switch ambito.Clave {
		case "organizacion_ref":
			r.OrganizacionRef = ambito.Valores[0]
		case "unidad_ref":
			r.UnidadRef = ambito.Valores[0]
		default:
			return cero, sinInstantanea, ports.ErrFirmaDocumentoDenegada
		}
	}
	if r.OrganizacionRef == "" || (len(a.Ambitos) == 2 && r.UnidadRef == "") {
		return cero, sinInstantanea, ports.ErrFirmaDocumentoDenegada
	}
	return r, s, nil
}

func (e *Emisor) AutorizarMaterialFirmaVerificadaV2(ctx context.Context, m ports.MaterialFirmaVerificadaV2, r vd.RecursoAutorizable) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return e.autorizarMaterial(ctx, m, r, "descriptor_firma_sha256")
}

// AutorizarMaterialPlanFirmaV2 usa la misma autoridad nominal. El recurso
// exterior procede de RecursoPlanAutorizadoFirmaV2; el consumidor SQL coteja
// ese envoltorio y su decisión interior antes de confirmar el efecto.
func (e *Emisor) AutorizarMaterialPlanFirmaV2(ctx context.Context, m ports.MaterialFirmaVerificadaV2, r vd.RecursoAutorizable) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return e.autorizarMaterial(ctx, m, r, "plan_firma_sha256")
}

func (e *Emisor) autorizarMaterial(ctx context.Context, m ports.MaterialFirmaVerificadaV2, r vd.RecursoAutorizable, claveHuella string) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	var cero vp.ExportacionMaterialConsumoAutorizacionAtestadaV3
	if ctx == nil || e == nil {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	if ctx.Err() != nil {
		return cero, opaco(ctx, nil)
	}
	m.EvidenciaFirmasCanonica = bytes.Clone(m.EvidenciaFirmasCanonica)
	r.Ambitos, r.Atributos = maps.Clone(r.Ambitos), maps.Clone(r.Atributos)
	if m.Validar() != nil || !recursoExactoConHuella(m, r, claveHuella) {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	var base ContextoActorFirmaV2
	var err error
	if e.admision != nil {
		if m.Via != ports.ViaFirmaCertificadoVEC {
			return cero, ports.ErrFirmaDocumentoDenegada
		}
		base, err = e.contextoParaActo(ctx, true)
	} else {
		base, err = e.contexto(ctx)
	}
	if err != nil {
		return cero, err
	}
	d, err := base.Vinculo.Datos()
	if err != nil || d.PerfilActivoRef != m.PerfilActivoOperadorRef {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	// Las dos decisiones, interior y exterior del plan, llevan los ámbitos de
	// la asignación vigente (AD206 y AD209 los releen en el consumo).
	esperados, instantanea, err := e.ambitosConInstantanea(ctx, base)
	if err != nil {
		return cero, err
	}
	if !maps.Equal(r.Ambitos, esperados.Mapa()) {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	accion, audiencia := ports.AccionRegistrarFirmaExterna, ports.AudienciaFirmaExternaV2
	if m.Via == ports.ViaFirmaCertificadoVEC {
		if d.PrincipalID != m.FirmantePrincipalRef || d.CuentaRef != m.CuentaFirmanteRef ||
			d.PerfilActivoRef != m.PerfilActivoFirmanteRef || base.CertificadoCanalSHA256 != m.CertificadoHuella {
			return cero, ports.ErrFirmaDocumentoDenegada
		}
		accion, audiencia = ports.AccionRegistrarFirmaVec, ports.AudienciaFirmaVecV2
	} else if d.PrincipalID == m.FirmantePrincipalRef {
		// CT172 separa explícitamente al operador RRHH del firmante externo.
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	if e.admision != nil {
		if m.Via != ports.ViaFirmaCertificadoVEC || accion != ports.AccionRegistrarFirmaVec ||
			audiencia != ports.AudienciaFirmaVecV2 {
			return cero, ports.ErrFirmaDocumentoDenegada
		}
		huellaMaterial, err := m.HuellaSHA256()
		if err != nil || r.Atributos["material_sha256"] != huellaMaterial {
			return cero, ports.ErrFirmaDocumentoDenegada
		}
		descriptor := vecapp.DescriptorAdmisionGarantiaActo{
			Accion: accion, Audiencia: audiencia, ModuloID: r.ModuloID,
			TipoRecurso: r.Tipo, Finalidad: ports.FinalidadFirmaDocumento,
			RecursoRef: r.Referencia, MaterialHuellaSHA256: huellaMaterial,
		}
		evidenciaAdmision, err := e.admision.Admitir(ctx, base.Vinculo, base.Resultado, instantanea, descriptor)
		if err != nil || ctx.Err() != nil {
			return cero, opaco(ctx, err)
		}
		resumen, err := evidenciaAdmision.Resumen()
		if err != nil || resumen.GarantiaReal != vd.AuthAssuranceSubstantial ||
			!e.reloj.Ahora().Before(resumen.VigenteHasta) {
			return cero, ports.ErrFirmaDocumentoDenegada
		}
		// ValidarEvidencia requiere la captura usada por el PDP y la ventana
		// de su decisión real. Hasta que esa captura exista en el contrato
		// común, no emitir siquiera una concesión candidata V3.
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	correlacion, err := vp.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	if err != nil {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	solicitud, err := vd.NuevaSolicitudAutorizacionLigadaV3(vd.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: base.Vinculo, ReferenciaMotivo: e.motivo,
		Accion: accion, Recurso: r, Finalidad: ports.FinalidadFirmaDocumento, Correlacion: correlacion,
	})
	if err != nil {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	// El emisor recibe su copia; no puede alterar la preimagen del cotejo.
	resultado, err := base.Resultado.Clonar()
	if err != nil || ctx.Err() != nil {
		return cero, opaco(ctx, err)
	}
	decision, confirmacion, exportador, err := e.emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, resultado)
	if err != nil || ctx.Err() != nil {
		return cero, opaco(ctx, err)
	}
	if decision.ValidarPara(solicitud) != nil || nulo(exportador) {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	restricciones, err := decision.RestriccionesProyeccionPara(solicitud)
	if err != nil || len(restricciones.CamposPermitidos) != 0 || len(restricciones.Obligaciones) != 0 {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || ctx.Err() != nil {
		return cero, opaco(ctx, err)
	}
	if !vp.MaterialAtestadoLigadoV3(solicitud, decision, confirmacion, base.Resultado, e.motivo, material, audiencia) ||
		!base.Vinculo.VigenteEn(e.reloj.Ahora(), base.Resultado) {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	ahora, resumen := e.reloj.Ahora(), material.ResumenCapacidad()
	if ahora.Before(resumen.EmitidaEn()) || !ahora.Before(resumen.ExpiraEn()) {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	if ctx.Err() != nil {
		return cero, opaco(ctx, nil)
	}
	return material, nil
}

// El descriptor ya fue congelado por AutorizadorNominalFirmaV2. Este puerto
// recibe su SHA256 y exige la preimagen exacta de RecursoFirmaVerificadaV2;
// los ámbitos se cotejan después con la asignación vigente.
func recursoExacto(m ports.MaterialFirmaVerificadaV2, r vd.RecursoAutorizable) bool {
	return recursoExactoConHuella(m, r, "descriptor_firma_sha256")
}

func recursoExactoConHuella(m ports.MaterialFirmaVerificadaV2, r vd.RecursoAutorizable, claveHuella string) bool {
	h, err := m.HuellaSHA256()
	tipo := ports.TipoRecursoFirmaExterna
	if m.Via == ports.ViaFirmaCertificadoVEC {
		tipo = ports.TipoRecursoFirmaVec
	}
	return (claveHuella == "descriptor_firma_sha256" || claveHuella == "plan_firma_sha256") && err == nil && r.Validar() == nil && r.Referencia == m.RecursoRef() && r.ModuloID == ports.ModuloContratacion && r.Tipo == tipo &&
		r.Ambitos["organizacion_ref"] == m.OrganizacionRef && len(r.Atributos) == 2 &&
		r.Atributos["material_sha256"] == h && ctdomain.HuellaSHA256FirmaValida(r.Atributos[claveHuella])
}

func opaco(ctx context.Context, causa error) error {
	for _, err := range []error{ctx.Err(), causa} {
		if errors.Is(err, context.Canceled) {
			return errors.Join(ports.ErrFirmaDocumentoDenegada, context.Canceled)
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return errors.Join(ports.ErrFirmaDocumentoDenegada, context.DeadlineExceeded)
		}
	}
	return ports.ErrFirmaDocumentoDenegada
}

func nulo(v any) bool {
	if v == nil {
		return true
	}
	x := reflect.ValueOf(v)
	switch x.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return x.IsNil()
	}
	return false
}

var _ ports.EmisorMaterialFirmaVerificadaV2 = (*Emisor)(nil)
var _ EmisorComunV3 = (*confianzaatestacion.EmisorMaterialAutorizacionAtestadaV3)(nil)

var _ ports.EmisorMaterialPlanFirmaV2 = (*Emisor)(nil)
