package bootstrap

import (
	"context"
	"math"
	"strings"
	"time"

	inc "vec-diputacion-granada/internal/app/incorporacionejercicio"
	bp "vec-diputacion-granada/internal/modules/bolsa/ports"
	httpct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	pgct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	appct "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	domct "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	apppersonal "vec-diputacion-granada/internal/modules/personal/application"
	personal "vec-diputacion-granada/internal/modules/personal/domain"
	pp "vec-diputacion-granada/internal/modules/personal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

type fuentesIncorporacionPersonalB2 struct {
	organizacionRef, organismoRef string
	autoridad                     *autoridadIncorporacionPersonalB2
	ct                            *pgct.FuentePlanNominalB2PostgreSQL
	servicioCT                    servicioCTIncorporacionB2
	detalle                       *appct.ServicioConsultaDetalleRRHH
	personal                      pp.ServicioPlanIncorporacionCT
	ficha                         pp.FuenteFichaIncorporacionCT
	hechos                        pp.ConsultaHechosIncorporacionCT
	vacantes                      *apppersonal.ServicioRegistroEmpleadoB2
	catalogos                     *apppersonal.ServicioCatalogosRegistroEmpleadoB2
	externas                      *FuentesContextoIncorporacionB2
	lectorRPT                     vp.LectorCategoriasRPT
	reloj                         ct.Reloj
	clases                        pp.ServicioClasesOcupacionCT
	anclaje                       bp.ConsultaAnclajeAceptacionCT
}

func versionB2DesdeInt64(v int64) (uint64, bool) {
	if v <= 0 {
		return 0, false
	}
	return uint64(v), true
}

func versionB2HaciaInt64(v uint64) (int64, bool) {
	if v == 0 || v > math.MaxInt64 {
		return 0, false
	}
	return int64(v), true
}

func (f *fuentesIncorporacionPersonalB2) antecedentes(ctx context.Context, exp string) (ct.AntecedentesPlanNominalB2, error) {
	unidad, e := f.ResolverUnidadPlanNominalB2(ctx, f.organizacionRef, exp)
	if e != nil {
		return ct.AntecedentesPlanNominalB2{}, e
	}
	b, e := domct.CanonicoPlanPersonalB2(map[string]string{"organizacion_ref": f.organizacionRef, "expediente_ref": exp, "unidad_ref": unidad})
	if e != nil {
		return ct.AntecedentesPlanNominalB2{}, e
	}
	x, e := f.autoridad.AutorizarPlanNominalB2(ctx, ct.AccionLeerPlanNominalB2, b, core.ContextoActor{})
	if e != nil {
		return ct.AntecedentesPlanNominalB2{}, e
	}
	return f.ct.LeerAntecedentesPlanB2(ctx, f.organizacionRef, exp, x, unidad)
}
func (f *fuentesIncorporacionPersonalB2) detalleActual(ctx context.Context, exp string) (ct.DetalleExpedienteRRHH, error) {
	s, e := ct.NuevaSolicitudDetalleRRHH(exp, 0)
	if e != nil {
		return ct.DetalleExpedienteRRHH{}, e
	}
	return f.detalle.Consultar(ctx, s)
}
func (f *fuentesIncorporacionPersonalB2) ResolverPlanNominalB2(ctx context.Context, s ct.SolicitudPlanNominalB2, actor core.ContextoActor) (domct.PlanIncorporacionPersonalB2, error) {
	var cero domct.PlanIncorporacionPersonalB2
	if f == nil || ctx == nil || s.OrganizacionRef != f.organizacionRef {
		return cero, ct.ErrPlanNominalB2Denegado
	}
	if e := f.autoridad.actorCoincide(ctx, ct.AccionRegistrarPlanNominalB2, actor); e != nil {
		return cero, e
	}
	a, e := f.antecedentes(ctx, s.ExpedienteRef)
	if e != nil {
		return cero, e
	}
	d, e := f.detalleActual(ctx, s.ExpedienteRef)
	if e != nil {
		return cero, e
	}
	if a.VersionExpediente != s.VersionExpediente || d.Resumen.Version != s.VersionExpediente || a.Vinculo == nil || a.Vinculo.Validar() != nil {
		return cero, ct.ErrPlanNominalB2Conflicto
	}
	if a.DocumentoRef == "" || a.DocumentoSHA256 == "" {
		return cero, ct.ErrPreparacionIncorporacionPendiente
	}
	if validarSeleccionFuenteCTB2(s, a, d) != nil {
		return cero, ct.ErrPlanNominalB2Conflicto
	}
	// Cambiar el periodo conocido requiere la rectificación propia de CT.
	ap, e := f.autoridad.actor(ctx, "personal.plan_incorporacion_ct.seleccionar")
	if e != nil {
		return cero, e
	}
	seleccion, e := f.personal.ResolverSeleccion(ctx, pp.SeleccionPlanIncorporacionCT{OrganismoRef: f.organismoRef, Actor: ap, SelectorOrganizacionPlanCT: pp.SelectorOrganizacionPlanCT{PlazaRef: s.PlazaRef, PuestoRef: s.PuestoRef, Desde: personal.FechaCivil(s.Desde)}})
	if e != nil {
		return cero, e
	}
	org := seleccion.Seleccion
	if org.VersionPlantillaRef != s.VersionPlantillaRef || org.VersionRPTRef != s.VersionRPTRef {
		return cero, ct.ErrPlanNominalB2Conflicto
	}
	v := a.Vinculo
	a, e = f.acreditarAnclajeB2(ctx, a)
	if e != nil {
		return cero, e
	}
	revisionPlantilla, okPlantilla := versionB2DesdeInt64(org.RevisionPlantilla)
	revisionRPT, okRPT := versionB2DesdeInt64(org.RevisionRPT)
	revisionPlaza, okPlaza := versionB2DesdeInt64(org.RevisionPlaza)
	revisionPuesto, okPuesto := versionB2DesdeInt64(org.RevisionPuesto)
	if !okPlantilla || !okRPT || !okPlaza || !okPuesto {
		return cero, ct.ErrPlanNominalB2NoDisponible
	}
	p := domct.PlanIncorporacionPersonalB2{OrganizacionRef: s.OrganizacionRef, UnidadCTRef: d.Resumen.UnidadRef, ExpedienteRef: s.ExpedienteRef, VersionExpediente: s.VersionExpediente, AnalisisVersion: a.AnalisisVersion, AnalisisReciboRef: a.AnalisisReciboRef, AnalisisSHA256: a.AnalisisSHA256, PropuestaReciboRef: a.PropuestaReciboRef, AceptacionRef: a.AceptacionRef, AceptacionReciboRef: a.AceptacionReciboRef, Bolsa: a.Bolsa, OrganismoRef: f.organismoRef, UnidadRef: org.UnidadRef, FuenteOrganizacion: domct.FuenteSinVersionPlanB2{Ref: org.FuenteOrganizacionRef, SHA256: org.FuenteOrganizacionHuellaSHA256}, FuentePlantilla: domct.InstrumentoPlanPersonalB2{Ref: strings.TrimPrefix(org.VersionPlantillaRef, "plantilla:"), Revision: revisionPlantilla, FuenteRef: org.PlantillaFuenteRef, FuenteSHA256: org.PlantillaHuellaSHA256}, FuenteRPT: domct.InstrumentoPlanPersonalB2{Ref: strings.TrimPrefix(org.VersionRPTRef, "rpt:"), Revision: revisionRPT, FuenteRef: org.RPTFuenteRef, FuenteSHA256: org.RPTHuellaSHA256}, VersionPlazaRef: org.VersionPlantillaRef, VersionPuestoRef: org.VersionRPTRef, RevisionPlaza: revisionPlaza, RevisionPuesto: revisionPuesto, PuestoRef: s.PuestoRef, PlazaRef: s.PlazaRef, CatalogoRPTID: v.CatalogoID, CatalogoRPTModulo: v.ModuloID, CatalogoRPTVersion: v.CatalogoVersion, CatalogoRPTSHA256: v.CatalogoHuellaSHA256, CategoriaRef: a.CategoriaRef, VinculoRevision: v.Revision, VinculoReciboRef: v.ReciboRef, Regimen: s.Regimen, Modalidad: s.Modalidad, ClaseOcupacion: s.ClaseOcupacion, Desde: s.Desde, Hasta: s.Hasta, MotivoClave: s.MotivoClave, DocumentoRef: s.DocumentoRef, DocumentoSHA256: s.DocumentoSHA256, EjercicioSintetico: true}
	// Sólo Bolsa devuelve Persona y su procedencia. CT no usa la persona RRHH.
	preliminar, e := contratoAplicacionDesdeMaterialB2(p)
	if e != nil {
		return cero, e
	}
	persona, e := f.resolverPersonaInicialB2(ctx, preliminar)
	if e != nil {
		return cero, e
	}
	p.PersonaRef, p.PersonaVersion, p.PersonaReciboBolsaRef = persona.PersonaRef, persona.PersonaVersion, persona.ReciboRef
	fuenteVersion, ok := versionB2DesdeInt64(persona.FuenteVersion)
	if !ok {
		return cero, ct.ErrPlanNominalB2NoDisponible
	}
	p.PersonaFuente = domct.FuentePlanPersonalB2{Ref: persona.FuenteRef, Version: fuenteVersion, SHA256: persona.FuenteSHA256}
	preliminar, e = contratoAplicacionDesdeMaterialB2(p)
	if e != nil {
		return cero, e
	}
	if _, e = f.externas.LeerPuestoRPT(ctx, preliminar); e != nil {
		return cero, e
	}
	if e = f.validarClaseOcupacion(ctx, p.ClaseOcupacion); e != nil {
		return cero, e
	}
	if p.Validar() != nil {
		return cero, ct.ErrPlanNominalB2Invalido
	}
	return p, nil
}
func validarSeleccionFuenteCTB2(s ct.SolicitudPlanNominalB2, a ct.AntecedentesPlanNominalB2, d ct.DetalleExpedienteRRHH) error {
	if s.DocumentoRef != a.DocumentoRef || s.DocumentoSHA256 != a.DocumentoSHA256 ||
		d.Analisis == nil || s.Desde != fechaCivilFuenteCT(d.Analisis.PeriodoInicio) ||
		s.Hasta != fechaCivilFuenteCT(d.Analisis.PeriodoFin) || s.MotivoClave != string(d.Solicitud.MotivoClave) {
		return ct.ErrPlanNominalB2Conflicto
	}
	return nil
}
func fechaCivilFuenteCT(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.DateOnly)
}
func contratoAplicacionDesdeMaterialB2(m domct.PlanIncorporacionPersonalB2) (inc.ContratoPlanNominal, error) {
	regimen, okRegimen := versionB2HaciaInt64(m.Regimen.Version)
	modalidad, okModalidad := versionB2HaciaInt64(m.Modalidad.Version)
	plaza, okPlaza := versionB2HaciaInt64(m.RevisionPlaza)
	puesto, okPuesto := versionB2HaciaInt64(m.RevisionPuesto)
	if !okRegimen || !okModalidad || !okPlaza || !okPuesto {
		return inc.ContratoPlanNominal{}, ct.ErrPlanNominalB2Conflicto
	}
	return inc.ContratoPlanNominal{Protocolo: inc.ProtocoloPersonalB2V1, OrganizacionRef: m.OrganizacionRef, ExpedienteRef: m.ExpedienteRef, VersionExpediente: m.VersionExpediente, EjercicioSintetico: m.EjercicioSintetico, AceptacionRef: m.AceptacionRef, AceptacionReciboRef: m.AceptacionReciboRef, LlamamientoRef: m.Bolsa.LlamamientoRef, SelectorBolsa: m.Bolsa, SeleccionReciboRef: m.PersonaReciboBolsaRef, PersonaRef: m.PersonaRef, PersonaVersion: m.PersonaVersion, PersonaFuente: m.PersonaFuente, FuenteRPT: ct.ReferenciaVersionadaPersonalRPT{Referencia: m.VersionPuestoRef, Version: m.FuenteRPT.Revision, HuellaSHA256: m.FuenteRPT.FuenteSHA256}, CategoriaRef: m.CategoriaRef, VinculoRevision: m.VinculoRevision, VinculoReciboRef: m.VinculoReciboRef, PuestoRef: m.PuestoRef, PlazaRef: m.PlazaRef, DatosPersonal: inc.DatosActosPersonalB2{OrganismoRef: m.OrganismoRef, UnidadRef: m.UnidadRef, Regimen: personal.EntradaCatalogoEmpleadoB2{Ref: m.Regimen.Ref, Version: regimen}, Modalidad: personal.EntradaCatalogoEmpleadoB2{Ref: m.Modalidad.Ref, Version: modalidad}, Desde: personal.FechaCivil(m.Desde), Hasta: personal.FechaCivil(m.Hasta), ClaseOcupacion: m.ClaseOcupacion, VersionPlantillaRef: m.VersionPlazaRef, VersionRPTRef: m.VersionPuestoRef, RevisionPlaza: plaza, RevisionPuesto: puesto, FuenteOrganizacionRef: m.FuenteOrganizacion.Ref, FuenteOrganizacionSHA256: m.FuenteOrganizacion.SHA256, CatalogoRPTID: m.CatalogoRPTID, ModuloRPTID: m.CatalogoRPTModulo, CategoriaID: m.CategoriaRef, CatalogoRPTVersion: m.CatalogoRPTVersion, CatalogoRPTHuellaSHA256: m.CatalogoRPTSHA256}}, nil
}
func contratoAplicacionB2(c ct.ContratoPlanNominalB2) (inc.ContratoPlanNominal, error) {
	r, e := contratoAplicacionDesdeMaterialB2(c.Material)
	if e != nil {
		return inc.ContratoPlanNominal{}, e
	}
	planVersion, ok := versionB2HaciaInt64(c.PlanVersion)
	if !ok {
		return inc.ContratoPlanNominal{}, ct.ErrPlanNominalB2Conflicto
	}
	r.ContratoRef = c.PlanRef
	r.ContratoVersion = c.PlanVersion
	r.ContratoReciboRef = c.PlanReciboRef
	r.ContratoSHA256 = c.PlanSHA256
	r.IntencionRef = c.IntencionRef
	r.IntencionReciboRef = c.IntencionReciboRef
	r.IntencionVersion = c.IntencionVersion
	r.SolicitudRef = c.SolicitudPersonalRef
	r.ReservaIdempotente = c.IdempotenciaPersonalUUID
	r.DatosPersonal.Procedencia = personal.ProcedenciaActoEmpleadoB2{ActoRef: c.PlanRef, FuenteRef: c.PlanRef, FuenteVersion: planVersion, FuenteHuellaSHA256: c.PlanSHA256, IdempotenciaRef: c.IdempotenciaPersonalUUID}
	return r, nil
}
func (f *fuentesIncorporacionPersonalB2) LeerContratoPlanNominal(ctx context.Context, org, exp string) (inc.ContratoPlanNominal, error) {
	if org != f.organizacionRef {
		return inc.ContratoPlanNominal{}, ct.ErrPlanNominalB2Denegado
	}
	c, e := f.servicioCT.LeerContratoPlanNominal(ctx, org, exp)
	if e != nil {
		return inc.ContratoPlanNominal{}, e
	}
	return contratoAplicacionB2(c)
}
func (f *fuentesIncorporacionPersonalB2) LeerIntencionCTDurable(ctx context.Context, c inc.ContratoPlanNominal) (inc.IntencionCTDurable, error) {
	actual, e := f.LeerContratoPlanNominal(ctx, c.OrganizacionRef, c.ExpedienteRef)
	if e != nil {
		return inc.IntencionCTDurable{}, e
	}
	if actual.ContratoRef != c.ContratoRef || actual.ContratoSHA256 != c.ContratoSHA256 {
		return inc.IntencionCTDurable{}, ct.ErrConflictoIncorporacionAplicacion
	}
	return inc.IntencionCTDurable{OrganizacionRef: actual.OrganizacionRef, ExpedienteRef: actual.ExpedienteRef, ContratoReciboRef: actual.ContratoReciboRef, IntencionRef: actual.IntencionRef, ReciboRef: actual.IntencionReciboRef, Version: actual.IntencionVersion, ReservaIdempotente: actual.ReservaIdempotente, Confirmada: true}, nil
}
func (f *fuentesIncorporacionPersonalB2) LeerAntecedenteCT124(ctx context.Context, c inc.ContratoPlanNominal) (inc.AntecedenteCT124, error) {
	a, e := f.antecedentes(ctx, c.ExpedienteRef)
	if e != nil {
		return inc.AntecedenteCT124{}, e
	}
	if a.OrganizacionRef != c.OrganizacionRef || a.AceptacionRef != c.AceptacionRef || a.AceptacionReciboRef != c.AceptacionReciboRef || a.Bolsa.LlamamientoRef != c.LlamamientoRef {
		return inc.AntecedenteCT124{}, ct.ErrConflictoIncorporacionAplicacion
	}
	return inc.AntecedenteCT124{OrganizacionRef: c.OrganizacionRef, ExpedienteRef: c.ExpedienteRef, AceptacionRef: a.AceptacionRef, LlamamientoRef: a.Bolsa.LlamamientoRef, VersionExpediente: c.VersionExpediente, ReciboRef: a.AceptacionReciboRef}, nil
}
func (f *fuentesIncorporacionPersonalB2) leerUsoRPT(ctx context.Context, p pp.PlanIncorporacionCT) (vp.ResultadoUsoCategoriaRPT, error) {
	s, x, e := f.autoridad.materialConsultaUsoRPT(ctx, p)
	if e != nil {
		return vp.ResultadoUsoCategoriaRPT{}, e
	}
	return f.lectorRPT.ConsultarUsoCategoriaRPT(ctx, vp.OrdenUsoCategoriaRPT{Consulta: vp.ConsultaUsoCategoriaRPT{Consumidor: "personal", UsoRef: p.UsoRPTRef, ReservaReciboRef: p.ReservaRPTRef}, Solicitud: s, Autorizacion: x})
}
func (f *fuentesIncorporacionPersonalB2) AcreditarReservaPlanCT(ctx context.Context, p pp.PlanIncorporacionCT, actor core.ContextoActor) (pp.EvidenciaReservaRPTPlanCT, error) {
	if e := f.autoridad.actorCoincide(ctx, "personal.plan_incorporacion_ct.ejecutar", actor); e != nil {
		return pp.EvidenciaReservaRPTPlanCT{}, e
	}
	if p.Datos.CatalogoRPTVersion <= 0 || p.Datos.CatalogoRPTVersion > math.MaxInt {
		return pp.EvidenciaReservaRPTPlanCT{}, ct.ErrConflictoIncorporacionAplicacion
	}
	r, e := f.leerUsoRPT(ctx, p)
	if e != nil {
		return pp.EvidenciaReservaRPTPlanCT{}, e
	}
	u := r.Uso
	if !r.Encontrado || u == nil || u.Consumidor != "personal" || u.UsoRef != p.UsoRPTRef || u.ReservaReciboRef != p.ReservaRPTRef || (u.Estado != "reservado" && u.Estado != "confirmado") || u.CategoriaID != p.Datos.CatalogoRPTCategoria || u.Publicacion != (vp.ReferenciaPublicacionRPT{CatalogoID: p.Datos.CatalogoRPTID, Version: int(p.Datos.CatalogoRPTVersion), HuellaSHA256: p.Datos.CatalogoRPTHuellaSHA256}) {
		return pp.EvidenciaReservaRPTPlanCT{}, ct.ErrConflictoIncorporacionAplicacion
	}
	return pp.EvidenciaReservaRPTPlanCT{UsoRPTRef: u.UsoRef, ReservaRPTRef: u.ReservaReciboRef, PlanHuellaSHA256: p.HuellaSHA256, PlazaRef: p.Datos.PlazaRef, PuestoRef: p.Datos.PuestoRef, CatalogoID: u.Publicacion.CatalogoID, Modulo: p.Datos.CatalogoRPTModulo, Categoria: u.CategoriaID, Version: int64(u.Publicacion.Version), HuellaSHA256: u.Publicacion.HuellaSHA256, ReciboRef: u.ReservaReciboRef}, nil
}
func (f *fuentesIncorporacionPersonalB2) VerificarHechosPersonalB2(ctx context.Context, c ct.ContratoPlanNominalB2, h ct.HechosPersonalIncorporacionB2, actor core.ContextoActor) (ct.HechosPersonalIncorporacionB2, error) {
	if e := f.autoridad.actorCoincide(ctx, ct.AccionLeerPlanNominalB2, actor); e != nil {
		return ct.HechosPersonalIncorporacionB2{}, e
	}
	ap, e := f.autoridad.ActorConsultaPlanB2(ctx)
	if e != nil {
		return ct.HechosPersonalIncorporacionB2{}, e
	}
	estado, e := f.personal.ConsultarPlan(ctx, pp.ConsultaPlanIncorporacionCT{PlanRef: h.PersonalPlanRef, OrganismoRef: c.Material.OrganismoRef, Actor: ap})
	if e != nil {
		return ct.HechosPersonalIncorporacionB2{}, e
	}
	p := estado.Plan
	if p.Datos.OrigenCTRef != c.PlanRef || p.Datos.OrigenCTReciboRef != c.PlanReciboRef || p.Datos.OrigenCTHuellaSHA256 != c.PlanSHA256 || estado.Estado != "ejecutado" || estado.ReciboAltaRelacion == nil || estado.ReciboOcupacion == nil {
		return ct.HechosPersonalIncorporacionB2{}, ct.ErrPlanNominalB2Conflicto
	}
	lector, e := f.autoridad.ActorLecturaHechosB2(ctx)
	if e != nil {
		return ct.HechosPersonalIncorporacionB2{}, e
	}
	corte := personal.CorteEmpleadoB2{VigenteEn: p.Datos.Desde, ConocidoEn: f.reloj.Ahora()}
	ficha, e := f.ficha.ConsultarFicha(ctx, personal.SolicitudFichaEmpleadoB2{EmpleadoRef: h.EmpleadoRef, OrganismoRef: p.Datos.OrganismoRef, Corte: corte, Actor: lector})
	if e != nil {
		return ct.HechosPersonalIncorporacionB2{}, e
	}
	relacionVersion, okRelacion := versionB2HaciaInt64(h.RelacionVersion)
	ocupacionVersion, okOcupacion := versionB2HaciaInt64(h.OcupacionVersion)
	if !okRelacion || !okOcupacion {
		return ct.HechosPersonalIncorporacionB2{}, ct.ErrPlanNominalB2Conflicto
	}
	hechos, e := f.hechos.ConsultarHechosIncorporacionCT(ctx, pp.SolicitudHechosIncorporacionCT{Seleccion: pp.SeleccionHechosIncorporacionCT{OrganismoRef: p.Datos.OrganismoRef, PersonaRef: p.Datos.PersonaRef, EmpleadoRef: h.EmpleadoRef, RelacionRef: h.RelacionRef, OcupacionRef: h.OcupacionRef, UnidadRef: p.Datos.UnidadRef, PuestoRef: p.Datos.PuestoRef, PlazaRef: p.Datos.PlazaRef, VersionEmpleado: ficha.Ficha.Version, VersionRelacion: relacionVersion, VersionOcupacion: ocupacionVersion, Corte: corte}, Actor: lector})
	if e != nil {
		return ct.HechosPersonalIncorporacionB2{}, e
	}
	uso, e := f.leerUsoRPT(ctx, p)
	if e != nil {
		return ct.HechosPersonalIncorporacionB2{}, e
	}
	verificado, e := hechosCTDesdePersonalB2(inc.ResultadoConsumidorPersonalB2{Estado: estado, Hechos: hechos, Uso: uso})
	if e != nil {
		return ct.HechosPersonalIncorporacionB2{}, e
	}
	if verificado != h {
		return ct.HechosPersonalIncorporacionB2{}, ct.ErrPlanNominalB2Conflicto
	}
	return verificado, nil
}

var _ ct.FuentePreparacionPlanNominalB2 = (*fuentesIncorporacionPersonalB2)(nil)
var _ inc.FuenteContratoPlanNominal = (*fuentesIncorporacionPersonalB2)(nil)
var _ pp.FuenteReservaRPTPlanCT = (*fuentesIncorporacionPersonalB2)(nil)

func (f *fuentesIncorporacionPersonalB2) ConsultarOpcionesIncorporacionB2(ctx context.Context, exp string) (httpct.ProyeccionIncorporacionPersonalB2HTTP, error) {
	var cero httpct.ProyeccionIncorporacionPersonalB2HTTP
	d, e := f.detalleActual(ctx, exp)
	if e != nil {
		return cero, e
	}
	a, e := f.antecedentes(ctx, exp)
	if e != nil {
		return cero, e
	}
	r := httpct.ProyeccionIncorporacionPersonalB2HTTP{Esquema: httpct.EsquemaConsultaIncorporacionPersonalB2, ExpedienteRef: exp, VersionExpedienteActual: d.Resumen.Version, Estado: "sin_plan", Prerrequisitos: []httpct.PrerrequisitoB2{
		{ClaveI18n: "ct_incorporacion_b2_previo_aceptacion_persona"},
		{ClaveI18n: "ct_incorporacion_b2_previo_documento"},
		{ClaveI18n: "ct_incorporacion_b2_previo_rpt"},
		{ClaveI18n: "ct_incorporacion_b2_previo_plaza"},
		{ClaveI18n: "ct_incorporacion_b2_previo_periodo"},
	}, Opciones: httpct.OpcionesIncorporacionPersonalB2{Vacantes: []httpct.OpcionVacanteB2{}, Regimenes: []httpct.OpcionCatalogoB2{}, Modalidades: []httpct.OpcionCatalogoB2{}, ClasesOcupacion: []httpct.OpcionClaseOcupacionB2{}, Motivos: []string{string(d.Solicitud.MotivoClave)}, Documentos: []httpct.OpcionDocumentoB2{}}}
	r.Prerrequisitos[1].Cumplido = domct.ReferenciaOpacaValida(a.DocumentoRef) && domct.HuellaPlanPersonalB2Valida(a.DocumentoSHA256)
	r.Prerrequisitos[2].Cumplido = a.Vinculo != nil && a.Vinculo.Validar() == nil && domct.ReferenciaOpacaValida(a.CategoriaRef)
	if d.Analisis == nil {
		return r, nil
	}
	desde, hasta := fechaCivilFuenteCT(d.Analisis.PeriodoInicio), fechaCivilFuenteCT(d.Analisis.PeriodoFin)
	r.Opciones.Periodo = httpct.PeriodoOpcionesB2{Desde: desde, Hasta: hasta, FuenteRef: a.AnalisisReciboRef}
	r.Prerrequisitos[4].Cumplido = domct.ReferenciaOpacaValida(a.AnalisisReciboRef) && domct.HuellaPlanPersonalB2Valida(a.AnalisisSHA256) && domct.VersionPlanPersonalB2Valida(a.AnalisisVersion) && a.AnalisisVersion <= d.Resumen.Version && desde != "" && d.Analisis.PeriodoInicio.Year() > 0 && (hasta == "" || d.Analisis.PeriodoFin.Year() > 0 && hasta > desde)
	actor, e := f.autoridad.actor(ctx, personal.AccionVacantesB2)
	if e != nil {
		return cero, e
	}
	vacantes, e := f.vacantes.ConsultarVacantes(ctx, personal.SolicitudVacantesB2{OrganismoRef: f.organismoRef, Corte: personal.CorteEmpleadoB2{VigenteEn: personal.FechaCivil(desde), ConocidoEn: f.reloj.Ahora()}, Limite: 100, Actor: actor})
	if e != nil {
		return cero, e
	}
	for _, v := range vacantes.Pagina.Vacantes {
		if v.PuestoRef == "" || v.VersionRPTRef == "" || v.PuestoDenominacion == "" || v.CodigoPlazaFuente == "" {
			continue
		}
		r.Opciones.Vacantes = append(r.Opciones.Vacantes, httpct.OpcionVacanteB2{PlazaRef: v.PlazaRef, PuestoRef: v.PuestoRef, VersionPlantillaRef: v.VersionPlantillaRef, VersionRPTRef: v.VersionRPTRef, UnidadRef: v.UnidadRef, CategoriaRef: a.CategoriaRef, PlazaEtiqueta: v.CodigoPlazaFuente, PuestoEtiqueta: v.PuestoDenominacion})
	}
	r.Prerrequisitos[3].Cumplido = len(r.Opciones.Vacantes) > 0
	actor, e = f.autoridad.actor(ctx, personal.AccionConsultarCatalogoEmpleadoB2)
	if e != nil {
		return cero, e
	}
	for _, tipo := range []string{"regimen", "modalidad"} {
		catalogo, e := f.catalogos.Consultar(ctx, personal.SolicitudConsultaCatalogoEmpleadoB2{OrganismoRef: f.organismoRef, Tipo: tipo, Estado: "publicada", Limite: 100, Actor: actor})
		if e != nil {
			return cero, e
		}
		lista := []httpct.OpcionCatalogoB2{}
		for _, entrada := range catalogo.Entradas {
			if personal.FechaCivil(desde).AntesDe(entrada.VigenteDesde) || entrada.VigenteHasta != "" && !personal.FechaCivil(desde).AntesDe(entrada.VigenteHasta) {
				continue
			}
			version, ok := versionB2DesdeInt64(entrada.Version)
			if !ok {
				return cero, ct.ErrPlanNominalB2NoDisponible
			}
			lista = append(lista, httpct.OpcionCatalogoB2{Ref: entrada.Ref, Version: version, Denominacion: entrada.Denominacion})
		}
		if tipo == "regimen" {
			r.Opciones.Regimenes = lista
		} else {
			r.Opciones.Modalidades = lista
		}
	}
	actor, e = f.autoridad.actor(ctx, "personal.plan_incorporacion_ct.clases_ocupacion")
	if e != nil {
		return cero, e
	}
	catalogoClases, e := f.clases.ConsultarClasesOcupacion(ctx, pp.ConsultaClasesOcupacionCT{OrganismoRef: f.organismoRef, Actor: actor})
	if e != nil {
		return cero, e
	}
	if catalogoClases.Catalogo.Validar() != nil {
		return cero, ct.ErrPlanNominalB2NoDisponible
	}
	versionClases, ok := versionB2DesdeInt64(catalogoClases.Catalogo.Version)
	if !ok {
		return cero, ct.ErrPlanNominalB2NoDisponible
	}
	r.Opciones.CatalogoClasesOcupacion = httpct.CatalogoClasesOcupacionB2{Ref: catalogoClases.Catalogo.Ref, Version: versionClases, HuellaSHA256: catalogoClases.Catalogo.HuellaSHA256}
	for _, clase := range catalogoClases.Catalogo.Opciones {
		r.Opciones.ClasesOcupacion = append(r.Opciones.ClasesOcupacion, httpct.OpcionClaseOcupacionB2{Valor: clase.Valor, TextoClave: clase.TextoClave})
	}
	if a.DocumentoRef != "" && a.DocumentoSHA256 != "" {
		r.Opciones.Documentos = append(r.Opciones.Documentos, httpct.OpcionDocumentoB2{DocumentoRef: a.DocumentoRef, DocumentoSHA256: a.DocumentoSHA256, EtiquetaClaveI18n: "ct_incorporacion_b2_documento_formalizacion"})
	}
	if a.ExpedienteRef == exp && a.VersionExpediente == d.Resumen.Version && domct.ReferenciaOpacaValida(a.AceptacionRef) && domct.ReferenciaOpacaValida(a.AceptacionReciboRef) {
		anclado, err := f.acreditarAnclajeB2(ctx, a)
		if err != nil {
			if e := errorPrevioPersonaIncorporacionB2(ctx, err); e != nil {
				return cero, e
			}
		} else {
			persona, err := f.resolverPersonaInicialB2(ctx, inc.ContratoPlanNominal{OrganizacionRef: f.organizacionRef, ExpedienteRef: exp, AceptacionRef: a.AceptacionRef, AceptacionReciboRef: a.AceptacionReciboRef, SelectorBolsa: anclado.Bolsa})
			if err != nil {
				if e := errorPrevioPersonaIncorporacionB2(ctx, err); e != nil {
					return cero, e
				}
			} else {
				r.Prerrequisitos[0].Cumplido = persona.ExpedienteRef == exp && persona.AceptacionRef == a.AceptacionRef && persona.LlamamientoRef == anclado.Bolsa.LlamamientoRef && domct.ReferenciaOpacaValida(persona.PersonaRef) && domct.VersionPlanPersonalB2Valida(persona.PersonaVersion) && domct.ReferenciaOpacaValida(persona.ReciboRef) && domct.HuellaPlanPersonalB2Valida(persona.FuenteSHA256)
			}
		}
	}
	return r, nil
}

func errorPrevioPersonaIncorporacionB2(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if err == ct.ErrPreparacionIncorporacionPendiente {
		return nil
	}
	return err
}

func (f *fuentesIncorporacionPersonalB2) resolverPersonaInicialB2(ctx context.Context, c inc.ContratoPlanNominal) (inc.PersonaSeleccionadaBolsa, error) {
	x := c.SelectorBolsa
	actor, e := f.externas.c.ActorBolsa(ctx)
	if e != nil {
		return inc.PersonaSeleccionadaBolsa{}, e
	}
	r, e := f.externas.c.Bolsa.ConsultarPersonaAceptacionCT(ctx, bp.SolicitudConsultaPersonaAceptacionCT{ActorConfiable: actor, Selector: bp.SelectorPersonaAceptacionCT{UnidadRef: x.UnidadRef, CategoriaRef: x.CategoriaRef, NecesidadRef: x.NecesidadRef, AceptacionOperacionRef: x.AceptacionOperacionRef, AceptacionRegistroSHA256: x.AceptacionRegistroSHA256, AperturaOperacionRef: x.AperturaOperacionRef, AperturaRegistroSHA256: x.AperturaRegistroSHA256, LlamamientoRef: x.LlamamientoRef, PropuestaRef: x.PropuestaRef}})
	if e != nil {
		return inc.PersonaSeleccionadaBolsa{}, e
	}
	if r.Estado != "acreditado" || r.Persona == nil || r.Vinculo == nil || r.Aceptacion == nil {
		return inc.PersonaSeleccionadaBolsa{}, ct.ErrPreparacionIncorporacionPendiente
	}
	personaVersion, ok := versionB2DesdeInt64(r.Persona.Version)
	if !ok {
		return inc.PersonaSeleccionadaBolsa{}, ct.ErrPlanNominalB2NoDisponible
	}
	return inc.PersonaSeleccionadaBolsa{OrganizacionRef: c.OrganizacionRef, ExpedienteRef: c.ExpedienteRef, AceptacionRef: c.AceptacionRef, LlamamientoRef: r.Aceptacion.LlamamientoRef, PersonaRef: r.Persona.Ref, PersonaVersion: personaVersion, ReciboRef: r.Aceptacion.ReciboRef, FuenteRef: r.Vinculo.ProcedenciaRef, FuenteVersion: r.Vinculo.ProcedenciaVersion, FuenteSHA256: r.Vinculo.ProcedenciaSHA256}, nil
}

func (f *fuentesIncorporacionPersonalB2) ResolverUnidadPlanNominalB2(ctx context.Context, org, exp string) (string, error) {
	if f == nil || org != f.organizacionRef {
		return "", ct.ErrPlanNominalB2Denegado
	}
	d, e := f.detalleActual(ctx, exp)
	if e != nil {
		return "", e
	}
	if d.Asignacion == nil || d.Asignacion.UnidadRef == "" || d.Asignacion.UnidadRef != d.Resumen.UnidadRef {
		return "", ct.ErrPreparacionIncorporacionPendiente
	}
	return d.Asignacion.UnidadRef, nil
}

func (f *fuentesIncorporacionPersonalB2) validarClaseOcupacion(ctx context.Context, clase string) error {
	actor, e := f.autoridad.actor(ctx, "personal.plan_incorporacion_ct.clases_ocupacion")
	if e != nil {
		return e
	}
	c, e := f.clases.ConsultarClasesOcupacion(ctx, pp.ConsultaClasesOcupacionCT{OrganismoRef: f.organismoRef, Actor: actor})
	if e != nil {
		return e
	}
	if c.Catalogo.Validar() != nil {
		return ct.ErrPlanNominalB2NoDisponible
	}
	for _, o := range c.Catalogo.Opciones {
		if o.Valor == clase {
			return nil
		}
	}
	return ct.ErrPlanNominalB2Invalido
}
func (f *fuentesIncorporacionPersonalB2) acreditarAnclajeB2(ctx context.Context, a ct.AntecedentesPlanNominalB2) (ct.AntecedentesPlanNominalB2, error) {
	actual, e := f.autoridad.contexto(ctx, bp.AccionConsultaAnclajeAceptacionCT)
	if e != nil {
		return ct.AntecedentesPlanNominalB2{}, e
	}
	x := a.Bolsa
	r, e := f.anclaje.ConsultarAnclajeAceptacionCT(ctx, bp.SolicitudConsultaAnclajeAceptacionCT{ActorConfiable: bp.ActorConfiablePersonaAceptacionCT{Vinculo: actual.Vinculo, Resultado: actual.Resultado}, Selector: bp.SelectorAnclajeAceptacionCT{UnidadRef: x.UnidadRef, CategoriaRef: x.CategoriaRef, NecesidadRef: x.NecesidadRef, AceptacionOperacionRef: x.AceptacionOperacionRef, AceptacionRegistroSHA256: x.AceptacionRegistroSHA256, AperturaOperacionRef: x.AperturaOperacionRef, LlamamientoRef: x.LlamamientoRef, PropuestaRef: x.PropuestaRef}})
	if e != nil {
		return ct.AntecedentesPlanNominalB2{}, e
	}
	if r.Estado != "acreditado" || r.Anclaje == nil {
		return ct.AntecedentesPlanNominalB2{}, ct.ErrPreparacionIncorporacionPendiente
	}
	anclaje := r.Anclaje.SelectorPersonaAceptacionCT
	a.Bolsa = domct.SelectorBolsaPlanB2{UnidadRef: anclaje.UnidadRef, CategoriaRef: anclaje.CategoriaRef, NecesidadRef: anclaje.NecesidadRef, AceptacionOperacionRef: anclaje.AceptacionOperacionRef, AceptacionRegistroSHA256: anclaje.AceptacionRegistroSHA256, AperturaOperacionRef: anclaje.AperturaOperacionRef, AperturaRegistroSHA256: anclaje.AperturaRegistroSHA256, LlamamientoRef: anclaje.LlamamientoRef, PropuestaRef: anclaje.PropuestaRef}
	return a, nil
}

func (f *fuentesIncorporacionPersonalB2) validarFuentesContratoB2(ctx context.Context, c ct.ContratoPlanNominalB2) error {
	contrato, e := contratoAplicacionB2(c)
	if e != nil {
		return e
	}
	persona, e := f.externas.LeerPersonaSeleccionada(ctx, contrato)
	if e != nil {
		return e
	}
	p := c.Material
	fuenteVersion, ok := versionB2DesdeInt64(persona.FuenteVersion)
	if !ok || persona.PersonaRef != p.PersonaRef || persona.PersonaVersion != p.PersonaVersion || persona.FuenteRef != p.PersonaFuente.Ref || fuenteVersion != p.PersonaFuente.Version || persona.FuenteSHA256 != p.PersonaFuente.SHA256 || persona.ReciboRef != p.PersonaReciboBolsaRef {
		return ct.ErrPlanNominalB2Conflicto
	}
	_, e = f.externas.LeerPuestoRPT(ctx, contrato)
	return e
}
