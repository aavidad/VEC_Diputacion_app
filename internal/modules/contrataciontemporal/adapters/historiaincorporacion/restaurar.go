package historiaincorporacion

import (
	"bytes"
	"context"
	"encoding/json"

	dom "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// DTO de MaterialCanonico, NO Material/Vinculo nominal. Sólo se usa para
// seleccionar las lecturas históricas y comprobar igualdad tras las fábricas.
type materialJSON struct {
	Esquema                                                  string
	Confirmacion                                             ct.DatosConfirmacionIncorporacion
	VersionActualExpediente                                  uint64
	Preparacion                                              ct.PreparacionSeguimientoConfirmacionIncorporacion
	SolicitudContexto                                        ct.SolicitudResolverContextoAutorizacionAltaV3
	Vinculo                                                  core.DatosVinculoAutenticacionActorV2
	ContextoCanonico, ProcedenciaCanonica                    []byte
	MotivoV3                                                 core.ReferenciaEntradaCatalogo
	CorrelacionV3                                            string
	Personal                                                 ct.RegistroPersonalEjercicio
	EjercicioSintetico, FirmaOficial, EficaciaAdministrativa bool
}

// Restaurar no valida permiso ACTUAL. Reproduce nominalmente la historia en
// sus fechas originales y comprueba recibo/consumos/seguimiento del mismo
// registro leído. Una entrada gobernada se elige por referencia y dos hashes;
// su procedencia depende del lector propietario, no del DTO ni del constructor.
func (r *Restaurador) Restaurar(ctx context.Context, selector Selector) (salida Restauracion, err error) {
	defer func() {
		if recover() != nil {
			salida = Restauracion{}
			err = ErrHistoria
		}
		if ctx != nil && ctx.Err() != nil {
			salida = Restauracion{}
			err = ctx.Err()
		} else if err != nil {
			salida = Restauracion{}
			err = ErrHistoria
		}
	}()
	if r == nil || ctx == nil {
		return Restauracion{}, ErrHistoria
	}
	if e := ctx.Err(); e != nil {
		return Restauracion{}, e
	}
	if !dom.ReferenciaOpacaValida(selector.ReciboRef) || !sha(selector.MaterialSHA256) || !sha(selector.IntencionSHA256) {
		return Restauracion{}, ErrHistoria
	}
	inicio := r.reloj.Ahora()
	if !dom.InstanteUTCCanonico(inicio) {
		return Restauracion{}, ErrHistoria
	}
	b, e := r.registros.LeerRegistroOriginal(ctx, selector)
	if e != nil || ctx.Err() != nil {
		return Restauracion{}, ErrHistoria
	}
	d, e := decodificarDocumento(b)
	if e != nil {
		return Restauracion{}, e
	}
	ev := d.Evidencia
	if d.Esquema != EsquemaDocumento || ev.Esquema != ct.EsquemaEvidenciaOrdenOriginalIncorporacionV2 ||
		d.Recibo.Transicion.ReciboRef != selector.ReciboRef || ev.MaterialSHA256 != selector.MaterialSHA256 || ev.IntencionSHA256 != selector.IntencionSHA256 ||
		d.Recibo.MaterialOriginalSHA256 != ev.MaterialSHA256 || d.Recibo.IntencionSHA256 != ev.IntencionSHA256 || !bytes.Equal(d.Recibo.MaterialOriginalCanonico, ev.MaterialCanonico) ||
		!dom.InstanteUTCCanonico(ev.PreparadoEn) || !dom.InstanteUTCCanonico(ev.EvaluadaEn) || !dom.InstanteUTCCanonico(ev.ConcesionRegistradaEn) || ev.EvaluadaEn.Before(ev.PreparadoEn) || ev.EvaluadaEn.After(inicio) ||
		d.Recibo.Transicion.RegistradaEn.After(inicio) || hash(ev.MaterialCanonico) != ev.MaterialSHA256 {
		return Restauracion{}, ErrHistoria
	}
	if jsonValido(ev.MaterialCanonico, 8<<20) != nil || jsonValido(ev.DecisionCanonica, vp.TamanoMaximoDecisionCanonicaV3) != nil {
		return Restauracion{}, ErrHistoria
	}
	var m materialJSON
	dec := json.NewDecoder(bytes.NewReader(ev.MaterialCanonico))
	dec.DisallowUnknownFields()
	if dec.Decode(&m) != nil || m.Esquema != ct.EsquemaMaterialConfirmacionIncorporacionV2 || m.Vinculo.Validar() != nil {
		return Restauracion{}, ErrHistoria
	}
	aRef := ReferenciaAutenticacion{m.Vinculo.AutenticacionRef, m.Vinculo.SesionRef, m.Vinculo.AutenticacionHuellaSHA256}
	cRef := ReferenciaContexto{ev.ContextoOriginalRef, hash(m.ContextoCanonico), hash(m.ProcedenciaCanonica)}
	v, resultado, e := core.CrearVinculoAutenticacionActorV2ConResultado(ctx, autenticacionHistorica{r.autenticaciones, aRef},
		core.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: aRef.AutenticacionRef, SesionRef: aRef.SesionRef},
		contextoHistorico{r.contextos, cRef}, core.SolicitudContextoActor{Cuenta: core.CuentaAutenticadaContextoActor{CuentaRef: m.Vinculo.CuentaRef, Metodo: m.Vinculo.MetodoObservado, Garantia: m.Vinculo.GarantiaObservada}, PerfilActivoRef: m.Vinculo.PerfilActivoRef}, relojOriginal{ev.PreparadoEn})
	if e != nil || ctx.Err() != nil {
		return Restauracion{}, ErrHistoria
	}
	vd, e := v.Datos()
	if e != nil || vd != m.Vinculo || resultado.RegistroContextoRef != cRef.RegistroRef || !bytes.Equal(resultado.RepresentacionCanonica, m.ContextoCanonico) || !bytes.Equal(resultado.ManifiestoProcedenciaCanonico, m.ProcedenciaCanonica) {
		return Restauracion{}, ErrHistoria
	}
	cor, e := core.GenerarReferenciaCorrelacionAutorizacionV2(ctx, correlacionHistorica(m.CorrelacionV3))
	if e != nil {
		return Restauracion{}, ErrHistoria
	}
	material, e := ct.NuevoMaterialConfirmacionIncorporacionV2(ct.DatosMaterialConfirmacionIncorporacionV2{
		Confirmacion: m.Confirmacion, VersionActualExpediente: m.VersionActualExpediente, Preparacion: m.Preparacion, SolicitudContexto: m.SolicitudContexto,
		Contexto: ct.ContextoAutorizacionAltaV3{Vinculo: v, Resultado: resultado}, MotivoV3: m.MotivoV3, CorrelacionV3: cor, Personal: m.Personal,
		EjercicioSintetico: m.EjercicioSintetico, FirmaOficial: m.FirmaOficial, EficaciaAdministrativa: m.EficaciaAdministrativa}, ev.PreparadoEn)
	if e != nil {
		return Restauracion{}, ErrHistoria
	}
	canon, e := material.MaterialCanonico()
	if e != nil || !bytes.Equal(canon, ev.MaterialCanonico) {
		return Restauracion{}, ErrHistoria
	}
	recurso, e := ct.RecursoConfirmacionIncorporacionV2(material)
	if e != nil {
		return Restauracion{}, ErrHistoria
	}
	solicitud, e := core.NuevaSolicitudAutorizacionLigadaV3(core.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: v, Accion: ct.AccionConfirmarIncorporacion, Finalidad: ct.FinalidadConfirmarIncorporacion, Recurso: recurso, ReferenciaMotivo: m.MotivoV3, Correlacion: cor})
	if e != nil {
		return Restauracion{}, ErrHistoria
	}
	hs, e := core.HuellaSHA256SolicitudAutorizacionV3(solicitud)
	if e != nil || hs != ev.SolicitudSHA256 {
		return Restauracion{}, ErrHistoria
	}
	// Sólo proyecta selector/ventana de la decisión. TODO el documento será
	// reconstruido por evaluación de snapshot histórico y cotejado byte a byte.
	var cab struct {
		DecisionRef string `json:"decision_ref"`
		EmitidaEn   string `json:"emitida_en"`
		ValidaHasta string `json:"valida_hasta"`
	}
	if json.Unmarshal(ev.DecisionCanonica, &cab) != nil {
		return Restauracion{}, ErrHistoria
	}
	desde, e := fecha(cab.EmitidaEn)
	if e != nil {
		return Restauracion{}, e
	}
	hasta, e := fecha(cab.ValidaHasta)
	if e != nil {
		return Restauracion{}, e
	}
	snapshot, e := r.evaluaciones.LeerEvaluacionOriginal(ctx, ReferenciaEvaluacion{cab.DecisionRef, hash(ev.DecisionCanonica), hs})
	if e != nil || ctx.Err() != nil {
		return Restauracion{}, ErrHistoria
	}
	evaluacion, e := core.NuevaEvidenciaEvaluacionAutorizacionV3(solicitud, snapshot, cab.DecisionRef, desde, hasta)
	if e != nil {
		return Restauracion{}, ErrHistoria
	}
	decision, e := core.NuevaDecisionAutorizacionLigadaV3(solicitud, evaluacion)
	if e != nil {
		return Restauracion{}, ErrHistoria
	}
	dc, e := core.RepresentacionCanonicaDecisionAutorizacionV3(decision)
	if e != nil || !bytes.Equal(dc, ev.DecisionCanonica) {
		return Restauracion{}, ErrHistoria
	}
	candidata, e := vp.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(solicitud, decision, m.MotivoV3, resultado)
	if e != nil {
		return Restauracion{}, ErrHistoria
	}
	concesion, e := vp.RecuperarConcesionHistoricaAutorizacionLigadaV3(ctx, r.concesiones, candidata)
	if e != nil {
		return Restauracion{}, e
	}
	cd, e := concesion.Datos()
	if e != nil || cd.RegistradaEn != ev.ConcesionRegistradaEn {
		return Restauracion{}, ErrHistoria
	}
	exportacion, e := recuperarExportacion(ev)
	if e != nil {
		return Restauracion{}, e
	}
	orden, e := ct.NuevaOrdenConfirmacionIncorporacionV2(material, ct.AutorizacionConfirmacionIncorporacionV2{Solicitud: solicitud, Decision: decision, Confirmacion: concesion, Exportacion: exportacion}, ev.EvaluadaEn)
	if e != nil || ev.CotejarOrden(ctx, orden) != nil {
		return Restauracion{}, ErrHistoria
	}
	historia, e := ct.NuevaHistoriaRegistroIncorporacionV2(orden, d.Recibo, d.Publicacion, d.Anterior, d.Posterior)
	if e != nil {
		return Restauracion{}, ErrHistoria
	}
	fin := r.reloj.Ahora()
	if !dom.InstanteUTCCanonico(fin) || fin.Before(inicio) || ctx.Err() != nil {
		return Restauracion{}, ErrHistoria
	}
	return Restauracion{orden, historia}, nil
}

func recuperarExportacion(e ct.EvidenciaOrdenOriginalIncorporacionV2) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	cero := vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	if jsonValido(e.CapacidadCanonica, vp.TamanoMaximoCapacidadCanonicaV3) != nil {
		return cero, ErrHistoria
	}
	// Sólo transporte estructural: NO verifica MAC, consume ni hace vigente la
	// capacidad. El hash de conjunto original compromete también este resumen.
	var c struct {
		Decision  string `json:"decision_ref"`
		HD        string `json:"huella_decision_sha256"`
		HM        string `json:"huella_motivo_sha256"`
		Contexto  string `json:"contexto_ref"`
		HC        string `json:"huella_contexto_sha256"`
		Operacion string `json:"operacion"`
		Efecto    string `json:"efecto_ref"`
		HE        string `json:"huella_efecto_sha256"`
		Audiencia string `json:"audiencia_consumo"`
		Desde     string `json:"emitida_en"`
		Hasta     string `json:"expira_en"`
	}
	if json.Unmarshal(e.CapacidadCanonica, &c) != nil {
		return cero, ErrHistoria
	}
	desde, err := fechaCapacidad(c.Desde)
	if err != nil {
		return cero, err
	}
	hasta, err := fechaCapacidad(c.Hasta)
	if err != nil {
		return cero, err
	}
	resumen, err := vp.NuevoResumenCapacidadAtestacionAutorizacionV3(c.Decision, c.HD, c.HM, c.Contexto, c.HC, c.Operacion, c.Efecto, c.HE, c.Audiencia, desde, hasta)
	if err != nil {
		return cero, ErrHistoria
	}
	x, err := vp.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(e.CapacidadCanonica, resumen, e.DecisionCanonica, e.MotivoCanonico, e.ContextoActorCanonico, e.PersonaVersion, e.PerfilVersion, e.PayloadVECAD3, e.SobreCOSESign1, e.EvidenciaVerificacion, e.RaizPublicaSPKI)
	if err != nil {
		return cero, ErrHistoria
	}
	h, err := x.HuellaConjuntoSHA256()
	if err != nil || h != e.HuellaExportacionSHA256 {
		return cero, ErrHistoria
	}
	return x, nil
}
