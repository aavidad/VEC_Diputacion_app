package bootstrap

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	ctapp "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

type pdpFirmaR5Prueba struct {
	llamadas int
	err      error
}

func (p *pdpFirmaR5Prueba) ExigirSolicitudLigadaV3(_ context.Context, s core.SolicitudAutorizacionLigadaV3, r core.ResultadoContextoActorRegistradoV2) (core.DecisionAutorizacionLigadaV3, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3, error) {
	p.llamadas++
	if p.err != nil {
		return core.DecisionAutorizacionLigadaV3{}, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, p.err
	}
	d, err := s.Datos()
	if err != nil || d.VinculoAutenticacionActor.ValidarPara(r) != nil {
		return core.DecisionAutorizacionLigadaV3{}, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, ports.ErrFirmaDocumentoDenegada
	}
	return core.DecisionAutorizacionLigadaV3{}, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil
}

type exportadorFirmaR5Prueba struct {
	accion, audiencia string
	recurso           core.RecursoAutorizable
	ahora             time.Time
	llamadas          int
	err               error
}

func (e *exportadorFirmaR5Prueba) proveerMaterialConfirmacion(_ context.Context, _ core.SolicitudAutorizacionLigadaV3, _ core.DecisionAutorizacionLigadaV3,
	_ vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3, _ core.ReferenciaEntradaCatalogo,
	_ core.ResultadoContextoActorRegistradoV2) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	e.llamadas++
	if e.err != nil {
		return vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, e.err
	}
	huella, err := e.recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, err
	}
	resumen, err := vp.NuevoResumenCapacidadAtestacionAutorizacionV3("decision-firma-r5-001", strings.Repeat("a", 64), strings.Repeat("a", 64),
		"contexto-firma-r5-001", strings.Repeat("a", 64), e.accion, e.recurso.Referencia, huella, e.audiencia,
		e.ahora.Add(-time.Second), e.ahora.Add(3*time.Second))
	if err != nil {
		return vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, err
	}
	clave := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{9}, ed25519.SeedSize))
	raiz, err := x509.MarshalPKIXPublicKey(clave.Public())
	if err != nil {
		return vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, err
	}
	return vp.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte{7}, vp.TamanoMinimoCapacidadCanonicaV3),
		resumen, []byte("decision"), []byte("motivo"), []byte("contexto"), 1, 1,
		[]byte("payload"), []byte("cose"), []byte("evidencia"), raiz)
}

func TestAutorizadoresFirmaR5ConsultaConDoblesYGuardas(t *testing.T) {
	s, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	ahora := s.reloj.Ahora()
	const ruta = httpinterno.RutaConsultaFirmaDocumento // Ruta reconocida por la sesión de prueba, no montaje R5.
	p, err := nuevoPerfilFijoCTDesarrollo(principal, s.contexto, ahora, "lector-r5-prueba", []string{ruta},
		func(actor, ref string) (core.InstantaneaAutorizacion, error) {
			return nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(actor, ref, ahora,
				"lector-r5-prueba", "Consulta sintética R5", "asignacion-lector-r5-prueba",
				[]core.ConcesionRol{{Accion: ports.AccionConsultarFirmasR5, ModuloID: ports.ModuloContratacion,
					TipoRecurso: ports.TipoRecursoConsultaFirmasR5, Finalidades: []string{ports.FinalidadFirmaDocumento},
					GarantiaMinima: core.AuthAssuranceHigh, CamposPermitidos: ctapp.CamposConsultaFirmasR5()}},
				[]core.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}}})
		})
	if err != nil || s.registrarPerfilFijoCTDesarrollo(p) != nil {
		t.Fatal("perfil fijo de prueba", err)
	}
	p.contextoEsperadoRegistrado, p.sesionOperativa = p.contexto.Resultado, proveedorSesionOperativaCTPrueba{contexto: p.contexto}
	asignaciones := s.autoridadAsignaciones.(*autoridadAsignacionesContratacionTemporalDesarrolloPrueba)
	asignaciones.asignaciones = map[string]instantaneaPublicadaDesarrollo{p.perfilRef(): {
		instantanea: clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(p.plantilla), actoAsignacion: actoAsignacionPerfilFijoCTDesarrollo}}
	m := ports.MaterialConsultaFirmasR5{OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo,
		ExpedienteRef: "expediente:ct:uno", VersionExpediente: 7, Documento: "informe_definitivo",
		FirmantePrincipalCandidatoRef: p.contexto.Resultado.Contexto.PersonaRef, ClaveIdempotencia: "clave-consulta-r5-0001",
		PasoOrden: 1, CatalogoHuella: strings.Repeat("c", 64)}
	recurso, err := ctapp.RecursoConsultaFirmasR5(m)
	if err != nil {
		t.Fatal(err)
	}
	pdp := new(pdpFirmaR5Prueba)
	exportador := &exportadorFirmaR5Prueba{accion: ports.AccionConsultarFirmasR5, audiencia: ports.AudienciaConsultaFirmasR5V3, recurso: recurso, ahora: ahora}
	a := &autorizadoresFirmaR5Desarrollo{soporte: s, pdp: pdp, reloj: s.reloj,
		consulta: operacionAutorizacionFirmaR5Desarrollo{rutas: []string{ruta}, perfil: p,
			motivo: motivoFirmaDocumentoCTDesarrollo(), exportador: exportador}}
	ctx := contextoRutaCoberturaDesarrolloPrueba(s, principal, ruta)
	cap := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	cap.metodo = http.MethodPost
	ctx = context.WithValue(context.Background(), claveCapacidadConsultasContratacionTemporalDesarrollo{}, cap)
	if _, err := a.AutorizarConsultaFirmasR5(ctx, m); err != nil || pdp.llamadas != 1 || exportador.llamadas != 1 {
		t.Fatalf("consulta atestada por dobles: %v, PDP=%d, exportador=%d", err, pdp.llamadas, exportador.llamadas)
	}
	if !perfilFirmaR5Concede(p, ports.AccionConsultarFirmasR5, ports.TipoRecursoConsultaFirmasR5, ctapp.CamposConsultaFirmasR5()) {
		t.Fatal("consulta R5 sin concesión exacta")
	}
	camposAlterados := append([]string(nil), ctapp.CamposConsultaFirmasR5()...)
	camposAlterados[0] = "ActorRef"
	if perfilFirmaR5Concede(p, ports.AccionConsultarFirmasR5, ports.TipoRecursoConsultaFirmasR5, camposAlterados) ||
		perfilFirmaR5Concede(p, ports.AccionRegistrarFirmaVec, ports.TipoRecursoFirmaVec, nil) {
		t.Fatal("perfil amplió campos o acción")
	}
	if fuenteFirmaR5Compuesta(s, p, ruta, ports.AccionConsultarFirmasR5, ports.TipoRecursoConsultaFirmasR5, motivoFirmaDocumentoCTDesarrollo()) {
		t.Fatal("la fuente CT118 antigua se presentó como R5")
	}
	if asignaciones.preparadas != 0 || asignaciones.publicadas != 0 {
		t.Fatal("la petición provisionó una asignación")
	}
	c := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	c.principal.ID = "per_ajena"
	ctxAjeno := context.WithValue(context.Background(), claveCapacidadConsultasContratacionTemporalDesarrollo{}, c)
	if _, err := a.AutorizarConsultaFirmasR5(ctxAjeno, m); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || pdp.llamadas != 1 {
		t.Fatalf("actor ajeno alcanzó el PDP: %v", err)
	}
	c = ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	c.metodo = http.MethodGet
	ctxMetodo := context.WithValue(context.Background(), claveCapacidadConsultasContratacionTemporalDesarrollo{}, c)
	if _, err := a.AutorizarConsultaFirmasR5(ctxMetodo, m); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || pdp.llamadas != 1 {
		t.Fatalf("método ajeno alcanzó el PDP: %v", err)
	}
	cap = ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	cap.certificadoValidoHasta = ahora.Add(-time.Second)
	ctxCaducado := context.WithValue(context.Background(), claveCapacidadConsultasContratacionTemporalDesarrollo{}, cap)
	if _, err := a.AutorizarConsultaFirmasR5(ctxCaducado, m); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || pdp.llamadas != 1 {
		t.Fatalf("certificado caducado alcanzó el PDP: %v", err)
	}
	if _, err := a.autorizar(ctx, a.consulta, ports.AccionConsultarFirmasR5, ports.AudienciaConsultaFirmasR5V3,
		recurso, p.contexto.Resultado.Contexto.PersonaRef, strings.Repeat("0", 64)); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || pdp.llamadas != 1 {
		t.Fatalf("certificado de firma ajeno alcanzó el PDP: %v", err)
	}
	m.Documento = "resolucion"
	if _, err := a.AutorizarConsultaFirmasR5(ctx, m); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) {
		t.Fatalf("material cambiado reutilizó capacidad: %v", err)
	}
}

func TestAutorizadoresFirmaR5ParoSinDescriptoresYCertificado(t *testing.T) {
	if _, err := nuevosAutorizadoresFirmaR5Desarrollo(configuracionAutorizadoresFirmaR5Desarrollo{}); !errors.Is(err, errAutorizadoresFirmaR5DesarrolloNoDisponibles) {
		t.Fatalf("montaje incompleto: %v", err)
	}
	r := core.RecursoAutorizable{Referencia: "operacion-firma-vec-ct:clave-firma-vec-0001", ModuloID: ports.ModuloContratacion,
		Tipo: ports.TipoRecursoFirmaVec, Ambitos: map[string]string{"organizacion_ref": organizacionAltaContratacionTemporalDesarrollo},
		Atributos: map[string]string{"material_sha256": strings.Repeat("a", 64)}}
	d := core.DatosSolicitudAutorizacionLigadaV3{Accion: ports.AccionRegistrarFirmaVec, Finalidad: ports.FinalidadFirmaDocumento,
		ReferenciaMotivo: motivoFirmaDocumentoCTDesarrollo(), Recurso: r}
	if !solicitudFirmaR5Exacta(d, d.Accion, d.ReferenciaMotivo, r) {
		t.Fatal("solicitud exacta rechazada")
	}
	ctx := context.WithValue(context.Background(), claveSolicitudFirmaR5Desarrollo{},
		solicitudFirmaR5Desarrollo{accion: ports.AccionRegistrarFirmaVec, motivo: motivoFirmaDocumentoCTDesarrollo(), recurso: r})
	if !solicitudAutorizacionFirmaR5DesarrolloValida(ctx, d) ||
		solicitudAutorizacionFirmaR5DesarrolloValida(context.Background(), d) {
		t.Fatal("marcador sellado de material no exigido")
	}
	d.Recurso.Atributos = map[string]string{"material_sha256": strings.Repeat("b", 64)}
	if solicitudFirmaR5Exacta(d, ports.AccionRegistrarFirmaVec, motivoFirmaDocumentoCTDesarrollo(), r) ||
		solicitudAutorizacionFirmaR5DesarrolloValida(ctx, d) {
		t.Fatal("material cambiado admitido")
	}
}

func materialFirmaR5DesarrolloPrueba(persona, certificado string) ports.MaterialFirmaExterna {
	return ports.MaterialFirmaExterna{
		Via: ports.ViaFirmaExternaPortafirmas, OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo,
		ExpedienteRef: "expediente:ct:uno", VersionExpediente: 7, Documento: "informe_definitivo",
		CatalogoRef: "vec.contratacion_temporal.circuito_firma:1", CatalogoHuella: strings.Repeat("1", 64),
		PasoRef: "vec.contratacion_temporal.circuito_firma:1:informe_definitivo.p1", PasoOrden: 1, Secuencia: 1,
		HistoriaRevision: 1, HistoriaHuella: strings.Repeat("2", 64), OriginalRef: "original:firma:001",
		OriginalVersion: 1, OriginalHuella: strings.Repeat("3", 64), FirmadoHuella: strings.Repeat("4", 64),
		CertificadoHuella: certificado, FirmanteRef: "ref:" + certificado,
		FirmantePrincipalRef: persona, PerfilFirmanteRef: "perfil:firmante:001", CargoFirmante: "Jefatura",
		UnidadFirmanteRef: "unidad:rrhh", PerfilActivoFirmanteRef: "perfil-activo:firmante:001",
		PuestoFirmanteRef: "puesto:firma", AmbitoFirmanteRef: "ambito:provincial",
		AsignacionFirmanteRef: "asignacion:firma:001", AsignacionFirmanteVersion: 1,
		AsignacionFirmanteHuella: strings.Repeat("5", 64), VersionRolFirmanteRef: "rol:ct:firmante:v1",
		VersionRolFirmanteHuella: strings.Repeat("6", 64), ControlVigenciaFirmanteRef: "rol:ct:firmante:v1",
		ControlVigenciaFirmanteRevision: 1, ControlVigenciaFirmanteHuella: strings.Repeat("7", 64),
		AsignacionVigenteDesde: "2026-09-01T00:00:00Z", AsignacionVigenteHasta: "2026-12-01T00:00:00Z",
		ActoCompetenciaRef: "acto:competencia:001", PoliticaVerificacion: ports.PoliticaVerificacionFirma,
		RevocacionEstado: "vigente", SelloTiempoEstado: "valido",
		ReferenciaPortafirmasDeclarada: "PF-2026-001", FechaPortafirmasDeclarada: "2026-10-02T10:00:00Z",
		ClaveIdempotencia: "clave-firma-r5-0001", DocumentoCustodiaRef: "documento:custodia:001",
		DocumentoCustodiaVersion: ports.VersionDocumentoCustodiado,
	}
}

func TestAutorizadoresFirmaR5EscriturasConDobles(t *testing.T) {
	s, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	ahora := s.reloj.Ahora()
	const ruta = httpinterno.RutaFirmaDocumento // Solo ejercicio del proveedor; la factoría R5 la veta.
	p, err := nuevoPerfilFijoCTDesarrollo(principal, s.contexto, ahora, "firmante-r5-prueba", []string{ruta},
		func(actor, ref string) (core.InstantaneaAutorizacion, error) {
			return nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(actor, ref, ahora,
				"firmante-r5-prueba", "Firmas sintéticas R5", "asignacion-firmante-r5-prueba",
				[]core.ConcesionRol{
					{Accion: ports.AccionRegistrarFirmaExterna, ModuloID: ports.ModuloContratacion, TipoRecurso: ports.TipoRecursoFirmaExterna,
						Finalidades: []string{ports.FinalidadFirmaDocumento}, GarantiaMinima: core.AuthAssuranceHigh},
					{Accion: ports.AccionRegistrarFirmaVec, ModuloID: ports.ModuloContratacion, TipoRecurso: ports.TipoRecursoFirmaVec,
						Finalidades: []string{ports.FinalidadFirmaDocumento}, GarantiaMinima: core.AuthAssuranceHigh}},
				[]core.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}}})
		})
	if err != nil || s.registrarPerfilFijoCTDesarrollo(p) != nil {
		t.Fatal("perfil de escrituras", err)
	}
	p.contextoEsperadoRegistrado, p.sesionOperativa = p.contexto.Resultado, proveedorSesionOperativaCTPrueba{contexto: p.contexto}
	asignaciones := s.autoridadAsignaciones.(*autoridadAsignacionesContratacionTemporalDesarrolloPrueba)
	asignaciones.asignaciones = map[string]instantaneaPublicadaDesarrollo{p.perfilRef(): {
		instantanea: clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(p.plantilla), actoAsignacion: actoAsignacionPerfilFijoCTDesarrollo}}
	ctx := contextoRutaCoberturaDesarrolloPrueba(s, principal, ruta)
	cap := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	cap.metodo = http.MethodPost
	ctx = context.WithValue(context.Background(), claveCapacidadConsultasContratacionTemporalDesarrollo{}, cap)
	mExterna := materialFirmaR5DesarrolloPrueba("per_firmante_externo_001", strings.Repeat("e", 64))
	if err := mExterna.Validar(); err != nil {
		t.Fatalf("material externo sintético: %v", err)
	}
	mVec := ports.MaterialFirmaVec(materialFirmaR5DesarrolloPrueba(p.contexto.Resultado.Contexto.PersonaRef, s.certificadoSHA256))
	mVec.Via, mVec.ReferenciaPortafirmasDeclarada, mVec.FechaPortafirmasDeclarada = ports.ViaFirmaCertificadoVEC, "", ""
	mVec.ClaveIdempotencia = "clave-firma-vec-0001"
	if err := mVec.Validar(); err != nil {
		t.Fatalf("material VEC sintético: %v", err)
	}
	rExterna, _ := ctapp.RecursoFirmaExterna(mExterna)
	rVec, _ := ctapp.RecursoFirmaVec(mVec)
	pdp := new(pdpFirmaR5Prueba)
	eExterna := &exportadorFirmaR5Prueba{accion: ports.AccionRegistrarFirmaExterna, audiencia: ports.AudienciaFirmaExternaV3, recurso: rExterna, ahora: ahora}
	eVec := &exportadorFirmaR5Prueba{accion: ports.AccionRegistrarFirmaVec, audiencia: ports.AudienciaFirmaVecV3, recurso: rVec, ahora: ahora}
	a := &autorizadoresFirmaR5Desarrollo{soporte: s, pdp: pdp, reloj: s.reloj,
		externa: operacionAutorizacionFirmaR5Desarrollo{rutas: []string{ruta}, perfil: p, motivo: motivoFirmaDocumentoCTDesarrollo(), exportador: eExterna},
		vec:     operacionAutorizacionFirmaR5Desarrollo{rutas: []string{ruta}, perfil: p, motivo: motivoFirmaDocumentoCTDesarrollo(), exportador: eVec}}
	if _, err := a.AutorizarRegistroFirmaExterna(ctx, mExterna); err != nil {
		t.Fatalf("registro externo con firmante distinto del registrador: %v", err)
	}
	if _, err := a.AutorizarFirmaVec(ctx, mVec); err != nil || pdp.llamadas != 2 || eExterna.llamadas != 1 || eVec.llamadas != 1 {
		t.Fatalf("firma VEC con certificado del canal: %v, llamadas %d/%d/%d", err, pdp.llamadas, eExterna.llamadas, eVec.llamadas)
	}
	if asignaciones.preparadas != 0 || asignaciones.publicadas != 0 {
		t.Fatal("escritura provisionó permisos por petición")
	}
	mVec.CertificadoHuella = strings.Repeat("a", 64)
	mVec.FirmanteRef = "ref:" + mVec.CertificadoHuella
	if _, err := a.AutorizarFirmaVec(ctx, mVec); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || pdp.llamadas != 2 {
		t.Fatalf("certificado ajeno alcanzó PDP: %v", err)
	}
	mVec.CertificadoHuella = s.certificadoSHA256
	mVec.FirmanteRef = "ref:" + mVec.CertificadoHuella
	mVec.FirmantePrincipalRef = "per_firmante_ajeno_001"
	if _, err := a.AutorizarFirmaVec(ctx, mVec); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || pdp.llamadas != 2 {
		t.Fatalf("principal firmante ajeno alcanzó PDP: %v", err)
	}
	cap.principal.ID = "certificado_ajeno"
	ctxAjeno := context.WithValue(context.Background(), claveCapacidadConsultasContratacionTemporalDesarrollo{}, cap)
	if _, err := a.AutorizarRegistroFirmaExterna(ctxAjeno, mExterna); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || pdp.llamadas != 2 {
		t.Fatalf("registrador ajeno alcanzó PDP: %v", err)
	}
	pdp.err = errors.New("pdp caído")
	if _, err := a.AutorizarRegistroFirmaExterna(ctx, mExterna); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || eExterna.llamadas != 1 {
		t.Fatalf("fallo PDP invocó exportador: %v", err)
	}
	pdp.err = nil
	eExterna.err = errors.New("exportador caído")
	if _, err := a.AutorizarRegistroFirmaExterna(ctx, mExterna); !errors.Is(err, ports.ErrRegistroFirmaDocumentoNoDisponible) {
		t.Fatalf("fallo exportador no cerró: %v", err)
	}
	if asignaciones.preparadas != 0 || asignaciones.publicadas != 0 {
		t.Fatal("fallos provisionaron permisos")
	}
}

func TestFronteraFirmaR5ExigeAccionYPerfil(t *testing.T) {
	s, _, _ := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	const ruta = httpinterno.RutaFirmaDocumento
	perfil := "prf_firma_r5_prueba"
	f := fronteraContratacionTemporalDesarrollo("ct-firma-r5-prueba", ports.AccionRegistrarFirmaVec, ruta, []string{perfil})
	fronteras, err := nuevoCatalogoFronterasComunDesarrollo([]descriptorFronteraComunDesarrollo{f})
	if err != nil {
		t.Fatal(err)
	}
	politica, err := nuevaPoliticaAutorizacionSolicitudLigadaV3Desarrollo(s, s, s, s)
	if err != nil {
		t.Fatal(err)
	}
	catalogo, err := nuevoCatalogoAutorizacionComunDesarrollo(fronteras, []descriptorAutorizacionComunDesarrollo{{
		Accion: ports.AccionRegistrarFirmaVec, ClavePolitica: f.ClavePolitica, ClaveCapacidad: f.ClaveCapacidad,
		Fronteras: []string{f.Clave}, Politica: politica}})
	if err != nil {
		t.Fatal(err)
	}
	pdp := &autorizadorComunDesarrollo{selector: selectorAutorizacionComunDesarrollo{catalogo: catalogo}}
	if !fronteraFirmaR5Compuesta(pdp, fronteras, ruta, ports.AccionRegistrarFirmaVec, perfil) ||
		fronteraFirmaR5Compuesta(pdp, fronteras, ruta, ports.AccionRegistrarFirmaExterna, perfil) ||
		fronteraFirmaR5Compuesta(pdp, fronteras, ruta, ports.AccionRegistrarFirmaVec, "prf_ajeno") {
		t.Fatal("frontera mezcló acción o perfil")
	}
}
