package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/bolsa/application/inscripcion"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type sesionInscripcionPrueba struct {
	ctx          contextoSeguridadComunDesarrollo
	acreditacion AcreditacionSesionInscripcionBolsa
}

func (s sesionInscripcionPrueba) ResolverInscripcion(*http.Request) (contextoSeguridadComunDesarrollo, AcreditacionSesionInscripcionBolsa, error) {
	return s.ctx, s.acreditacion, nil
}

type autoridadInscripcionPrueba struct {
	alterarRecurso bool
	huella, canal  string
}

func (a autoridadInscripcionPrueba) CapturarLectura(_ context.Context, ctx contextoSeguridadComunDesarrollo, _ AcreditacionSesionInscripcionBolsa, accion, recurso string, filtro inscripcion.Filtro) (inscripcion.CapturaLectura, error) {
	v, _ := ctx.Vinculo.Datos()
	if a.alterarRecurso {
		recurso = "otro_recurso"
	}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	huella := a.huella
	if huella == "" {
		huella = "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	}
	canal := a.canal
	if canal == "" {
		canal = "interna_corporativa"
	}
	return inscripcion.CapturaLectura{PersonaRef: ctx.Resultado.Contexto.PersonaRef, PerfilRef: ctx.Resultado.Contexto.PerfilActivoRef,
		CuentaRef: v.CuentaRef, SesionRef: v.SesionRef, AutenticacionRef: v.AutenticacionRef,
		CertificadoHuellaSHA256: huella,
		Canal:                   canal, Accion: accion, RecursoRef: recurso, Filtro: filtro,
		Finalidad: "revision_inscripciones", CorrelacionRef: "cor_prueba_001", RevisionPermisos: 1,
		EmitidaEn: ahora, ValidaHasta: ahora.Add(20 * time.Second)}, nil
}

type selectorCanalInscripcionPrueba struct{ canal string }

func (s selectorCanalInscripcionPrueba) SeleccionarCanalAspirante(*http.Request) (string, error) {
	return s.canal, nil
}

type acreditadorEmpleadoInscripcionPrueba struct {
	identidad AcreditacionEmpleadoInscripcionBolsa
}

func (a acreditadorEmpleadoInscripcionPrueba) AcreditarEmpleadoInscripcion(context.Context, contextoSeguridadComunDesarrollo) (AcreditacionEmpleadoInscripcionBolsa, error) {
	return a.identidad, nil
}

func contextoInscripcionCanalPrueba(t *testing.T, ahora time.Time, exterior, empleado bool) (contextoSeguridadComunDesarrollo, string) {
	t.Helper()
	principal := vecdomain.Principal{ID: "certificado_sintetico_inscripcion", Roles: []string{"empleado"},
		AuthMethod: vecdomain.AuthMethodCertificate, AuthAssurance: vecdomain.AuthAssuranceHigh,
		Attributes: map[string]string{"autoridad": AutoridadNoAutoritativa, "perfil_ejecucion": config.ExecutionProfileDevelopment,
			"certificate_sha256": "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"}}
	base, err := nuevoContextoSinteticoContratacionTemporalDesarrollo(principal, ahora)
	if err != nil {
		t.Fatal(err)
	}
	resultado := base.Resultado
	empleadoRef := ""
	if empleado {
		resultado = resultadoConVinculoEmpleadoF1(t, resultado)
		empleadoRef = resultado.Contexto.Instantanea.Vinculos[0].Referencia
	}
	v, err := base.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	superficie := vecdomain.SuperficieAutenticacionInternaCorporativaV1
	if exterior {
		superficie = vecdomain.SuperficieAutenticacionExternaPersonalV1
	}
	autenticacion := vecdomain.AutenticacionRevalidadaV1{
		AutenticacionRef: v.AutenticacionRef, AutenticacionHuellaSHA256: v.AutenticacionHuellaSHA256,
		AsercionRef: v.AsercionRef, SesionRef: v.SesionRef, ControlSesionRef: v.ControlSesionRef,
		ControlSesionRevision: v.ControlSesionRevision, ControlSesionHuellaSHA256: v.ControlSesionHuellaSHA256,
		CuentaRef: v.CuentaRef, CuentaOrdinariaRef: v.CuentaOrdinariaRef, CuentaPrivilegiada: v.CuentaPrivilegiada,
		Superficie: superficie, MetodoObservado: v.MetodoObservado, GarantiaObservada: v.GarantiaObservada,
		PoliticaGarantiaRef: v.PoliticaGarantiaRef, PoliticaGarantiaHuellaSHA256: v.PoliticaGarantiaHuellaSHA256,
		AutenticacionVerificadaEn: v.AutenticacionVerificadaEn, SesionEmitidaEn: v.SesionEmitidaEn,
		SesionValidaHasta: v.SesionValidaHasta, SesionRevalidadaEn: v.SesionRevalidadaEn,
	}
	ligadura, resuelto, err := vecdomain.CrearVinculoAutenticacionActorV2ConResultado(context.Background(),
		revalidadorAutenticacionAltaContratacionTemporalDesarrollo{valor: autenticacion},
		vecdomain.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: v.AutenticacionRef, SesionRef: v.SesionRef},
		resolutorContextoAltaContratacionTemporalDesarrollo{valor: resultado},
		vecdomain.SolicitudContextoActor{Cuenta: vecdomain.CuentaAutenticadaContextoActor{
			CuentaRef: v.CuentaRef, Metodo: vecdomain.AuthMethodCertificate, Garantia: vecdomain.AuthAssuranceHigh},
			PerfilActivoRef: v.PerfilActivoRef}, relojFijoAltaContratacionTemporalDesarrollo{ahora: ahora})
	if err != nil {
		t.Fatal(err)
	}
	return contextoSeguridadComunDesarrollo{Vinculo: ligadura, Resultado: resuelto}, empleadoRef
}

func acreditacionSesionInscripcionPrueba(t *testing.T, ctx contextoSeguridadComunDesarrollo, canal string, ahora time.Time) AcreditacionSesionInscripcionBolsa {
	t.Helper()
	v, err := ctx.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	return AcreditacionSesionInscripcionBolsa{CertificadoHuellaSHA256: "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		Canal: canal, PersonaRef: ctx.Resultado.Contexto.PersonaRef, PerfilRef: ctx.Resultado.Contexto.PerfilActivoRef,
		CuentaRef: v.CuentaRef, SesionRef: v.SesionRef, AutenticacionRef: v.AutenticacionRef,
		VerificadaEn: ahora.Add(-time.Second), ValidaHasta: ahora.Add(time.Minute)}
}

func (autoridadInscripcionPrueba) AutorizarEscritura(context.Context, contextoSeguridadComunDesarrollo, AcreditacionSesionInscripcionBolsa, string, string, []byte, []byte) (AutorizacionEscrituraInscripcionBolsa, error) {
	return AutorizacionEscrituraInscripcionBolsa{Material: vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}}, inscripcion.ErrAccesoDenegado
}

func TestInscripcionBolsaContextoSinParticipacionPrevia(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	principal := vecdomain.Principal{ID: "persona_sintetica_sin_participacion", Roles: []string{"tecnico_rrhh"},
		AuthMethod: vecdomain.AuthMethodCertificate, AuthAssurance: vecdomain.AuthAssuranceHigh,
		Attributes: map[string]string{"autoridad": AutoridadNoAutoritativa,
			"perfil_ejecucion":   config.ExecutionProfileDevelopment,
			"certificate_sha256": "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"}}
	base, err := nuevoContextoAltaContratacionTemporalDesarrollo(principal, ahora)
	if err != nil {
		t.Fatal(err)
	}
	ctx := contextoSeguridadComunDesarrollo{Vinculo: base.Vinculo, Resultado: base.Resultado}
	if len(ctx.Resultado.Contexto.Instantanea.Vinculos) != 0 {
		t.Fatal("la prueba requiere persona sin participación")
	}
	persona := ctx.Resultado.Contexto.PersonaRef
	perfil := ctx.Resultado.Contexto.PerfilActivoRef
	cuenta := ctx.Resultado.Contexto.Instantanea.CuentaRef
	if !contextoInscripcionBolsaValido(ctx, persona, perfil, cuenta, true, ahora) {
		t.Fatal("la sesión válida sin participación quedó bloqueada")
	}
	for nombre, datos := range map[string][3]string{
		"persona ajena": {"otra_persona", perfil, cuenta},
		"perfil ajeno":  {persona, "otro_perfil", cuenta},
		"cuenta ajena":  {persona, perfil, "otra_cuenta"},
	} {
		t.Run(nombre, func(t *testing.T) {
			if contextoInscripcionBolsaValido(ctx, datos[0], datos[1], datos[2], true, ahora) {
				t.Fatal("se aceptó suplantación de sesión")
			}
		})
	}
	if contextoInscripcionBolsaValido(ctx, persona, perfil, cuenta, false, ahora) {
		t.Fatal("sesión interna aceptada en superficie exterior")
	}
}

func TestInscripcionBolsaFiltroYRutaExactos(t *testing.T) {
	casos := []struct {
		nombre, metodo, ruta, accion, recurso string
		rrhh, valido                          bool
		filtro                                inscripcion.Filtro
	}{
		{"abiertas", "GET", "/api/vec/bolsa/inscripciones/convocatorias-abiertas?limite=7&cursor=abc", inscripcion.AccionListarAbiertas, "convocatorias-abiertas", false, true, inscripcion.Filtro{Limite: 7, Cursor: "abc"}},
		{"propias", "GET", "/api/vec/bolsa/mi-bolsa/inscripciones?limite=8", inscripcion.AccionListarPropias, "inscripciones:propias:", false, true, inscripcion.Filtro{Limite: 8}},
		{"rrhh pendiente", "GET", "/api/vec/bolsa/rrhh/inscripciones?convocatoria_ref=cv1_001_v1", inscripcion.AccionListarRRHH, "inscripciones:rrhh", true, true, inscripcion.Filtro{Limite: 20, Estado: inscripcion.EstadoPendiente, ConvocatoriaRef: "cv1_001_v1"}},
		{"rrhh ajeno exterior", "GET", "/api/vec/bolsa/rrhh/inscripciones", "", "", false, false, inscripcion.Filtro{}},
		{"filtro duplicado", "GET", "/api/vec/bolsa/mi-bolsa/inscripciones?limite=8&limite=9", "", "", false, false, inscripcion.Filtro{}},
		{"query oculta", "GET", "/api/vec/bolsa/mi-bolsa/inscripciones?persona_ref=otra", "", "", false, false, inscripcion.Filtro{}},
		{"post con query", "POST", "/api/vec/bolsa/mi-bolsa/inscripciones?persona_ref=otra", "", "", false, false, inscripcion.Filtro{}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			r := httptest.NewRequest(caso.metodo, caso.ruta, nil)
			accion, recurso, filtro, ok := operacionInscripcionBolsa(r, caso.rrhh)
			if ok != caso.valido {
				t.Fatalf("admisión = %v", ok)
			}
			if ok && (accion != caso.accion || recurso != caso.recurso || filtro != caso.filtro) {
				t.Fatalf("acción/recurso/filtro alterados: %q %q %+v", accion, recurso, filtro)
			}
		})
	}
}

func TestInscripcionBolsaConstructorSinAutoridadCierra(t *testing.T) {
	if _, err := NuevoPreparadorInscripcionBolsa(ConfiguracionPreparadorInscripcionBolsa{}); !errors.Is(err, inscripcion.ErrNoDisponible) {
		t.Fatalf("constructor sin autoridad = %v", err)
	}
	r := httptest.NewRequest(http.MethodGet, "/api/vec/bolsa/mi-bolsa/inscripciones", nil)
	if _, err := (&preparadorInscripcionBolsa{}).PrepararLecturaAspirante(r, inscripcion.AccionListarPropias, "", inscripcion.Filtro{Limite: 20}, "es"); !errors.Is(err, inscripcion.ErrNoDisponible) {
		t.Fatalf("preparador sin sesión = %v", err)
	}
}

func TestInscripcionBolsaLecturaNominalSinParticipacionYSuplantacion(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	huella := "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	principal := vecdomain.Principal{ID: "certificado_rrhh_sintetico_inscripcion", Roles: []string{"tecnico_rrhh"},
		AuthMethod: vecdomain.AuthMethodCertificate, AuthAssurance: vecdomain.AuthAssuranceHigh,
		Attributes: map[string]string{"autoridad": AutoridadNoAutoritativa, "perfil_ejecucion": config.ExecutionProfileDevelopment,
			"certificate_sha256": huella}}
	base, err := nuevoContextoAltaContratacionTemporalDesarrollo(principal, ahora)
	if err != nil {
		t.Fatal(err)
	}
	ctx := contextoSeguridadComunDesarrollo{Vinculo: base.Vinculo, Resultado: base.Resultado}
	if len(ctx.Resultado.Contexto.Instantanea.Vinculos) != 0 {
		t.Fatal("el contexto tiene participación")
	}
	v, err := ctx.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	perfil := ctx.Resultado.Contexto.PerfilActivoRef
	acreditacion := AcreditacionSesionInscripcionBolsa{
		CertificadoHuellaSHA256: huella, Canal: "interna_corporativa",
		PersonaRef: ctx.Resultado.Contexto.PersonaRef, PerfilRef: perfil, CuentaRef: v.CuentaRef,
		SesionRef: v.SesionRef, AutenticacionRef: v.AutenticacionRef,
		VerificadaEn: ahora.Add(-time.Second), ValidaHasta: ahora.Add(time.Minute),
	}
	prueba := sesionInscripcionPrueba{ctx: ctx, acreditacion: acreditacion}
	c := ConfiguracionPreparadorInscripcionBolsa{
		RRHH:            []identidadConsultaRRHHDesarrollo{{identidad: identidadCertificadoDesarrollo{principal: principal}, perfilRef: perfil}},
		SesionAspirante: prueba, SesionRRHH: prueba, Autoridad: autoridadInscripcionPrueba{},
		Reloj: relojContratacionTemporalDesarrollo{},
	}
	p, err := NuevoPreparadorInscripcionBolsa(c)
	if err != nil {
		t.Fatal(err)
	}
	filtro := inscripcion.Filtro{Limite: 20, Estado: inscripcion.EstadoPendiente}
	r := httptest.NewRequest(http.MethodGet, "/api/vec/bolsa/rrhh/inscripciones", nil)
	a, err := p.PrepararLecturaRRHH(r, inscripcion.AccionListarRRHH, "", filtro, "es")
	if err != nil || a.PersonaRef != acreditacion.PersonaRef || a.Lectura == nil || !a.LecturaValida(inscripcion.AccionListarRRHH, a.Lectura.RecursoRef, filtro) {
		t.Fatalf("lectura nominal sin participación: %v", err)
	}
	acreditacion.PersonaRef = "per_suplantada"
	c.SesionRRHH = sesionInscripcionPrueba{ctx: ctx, acreditacion: acreditacion}
	p, err = NuevoPreparadorInscripcionBolsa(c)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.PrepararLecturaRRHH(r, inscripcion.AccionListarRRHH, "", filtro, "es"); !errors.Is(err, inscripcion.ErrSesionAusente) {
		t.Fatalf("acreditación de otra persona: %v", err)
	}
	acreditacion.PersonaRef = ctx.Resultado.Contexto.PersonaRef
	c.SesionRRHH = sesionInscripcionPrueba{ctx: ctx, acreditacion: acreditacion}
	c.Autoridad = autoridadInscripcionPrueba{alterarRecurso: true}
	p, err = NuevoPreparadorInscripcionBolsa(c)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.PrepararLecturaRRHH(r, inscripcion.AccionListarRRHH, "", filtro, "es"); !errors.Is(err, inscripcion.ErrAccesoDenegado) {
		t.Fatalf("concesión para otro recurso: %v", err)
	}
}

func TestInscripcionBolsaAspiranteExternoSinCandidatura(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	ctx, _ := contextoInscripcionCanalPrueba(t, ahora, true, false)
	if len(ctx.Resultado.Contexto.Instantanea.Vinculos) != 0 {
		t.Fatal("el externo ya tiene vínculo")
	}
	acreditacion := acreditacionSesionInscripcionPrueba(t, ctx, "externa_personal", ahora)
	sesion := sesionInscripcionPrueba{ctx: ctx, acreditacion: acreditacion}
	c := ConfiguracionPreparadorInscripcionBolsa{SesionAspirante: sesion, SesionRRHH: sesion,
		RRHH:      []identidadConsultaRRHHDesarrollo{{perfilRef: "prf_rrhh_nominal"}},
		Autoridad: autoridadInscripcionPrueba{canal: "externa_personal"}, Reloj: relojContratacionTemporalDesarrollo{}}
	p, err := NuevoPreparadorInscripcionBolsa(c)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodGet, "/api/vec/bolsa/inscripciones/convocatorias-abiertas", nil)
	filtro := inscripcion.Filtro{Limite: 20}
	a, err := p.PrepararLecturaAspirante(r, inscripcion.AccionListarAbiertas, "", filtro, "es")
	if err != nil || a.PersonaRef != acreditacion.PersonaRef || a.Canal != "externa_personal" || a.Lectura == nil {
		t.Fatalf("aspirante externo sin candidatura: %v", err)
	}
}

func TestInscripcionBolsaEmpleadoNominalSinParticipacion(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	ctx, empleadoRef := contextoInscripcionCanalPrueba(t, ahora, false, true)
	if len(ctx.Resultado.Contexto.Instantanea.Vinculos) != 1 ||
		ctx.Resultado.Contexto.Instantanea.Vinculos[0].Tipo != vecdomain.TipoReferenciaContextoActorEmpleado {
		t.Fatal("la prueba requiere empleado sin candidatura ni participación")
	}
	acreditacion := acreditacionSesionInscripcionPrueba(t, ctx, "interna_corporativa", ahora)
	sesion := sesionInscripcionPrueba{ctx: ctx, acreditacion: acreditacion}
	identidad := AcreditacionEmpleadoInscripcionBolsa{EmpleadoRef: empleadoRef,
		PersonaRef: acreditacion.PersonaRef, PerfilRef: acreditacion.PerfilRef,
		CuentaRef: acreditacion.CuentaRef, ValidaHasta: ahora.Add(time.Minute)}
	c := ConfiguracionPreparadorInscripcionBolsa{SesionAspirante: sesion, SesionEmpleado: sesion, SesionRRHH: sesion,
		SelectorCanalAspirante: selectorCanalInscripcionPrueba{canal: "interna_corporativa"},
		AcreditadorEmpleado:    acreditadorEmpleadoInscripcionPrueba{identidad: identidad},
		RRHH: []identidadConsultaRRHHDesarrollo{{perfilRef: "prf_rrhh_nominal", identidad: identidadCertificadoDesarrollo{
			principal: vecdomain.Principal{Attributes: map[string]string{"certificate_sha256": acreditacion.CertificadoHuellaSHA256}}}}},
		Autoridad: autoridadInscripcionPrueba{}, Reloj: relojContratacionTemporalDesarrollo{}}
	p, err := NuevoPreparadorInscripcionBolsa(c)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodGet, "/api/vec/bolsa/mi-bolsa/inscripciones", nil)
	filtro := inscripcion.Filtro{Limite: 20}
	a, err := p.PrepararLecturaAspirante(r, inscripcion.AccionListarPropias, "", filtro, "es")
	if err != nil || a.PersonaRef != identidad.PersonaRef || a.Canal != "interna_corporativa" || a.Lectura == nil {
		t.Fatalf("empleado sin participación: %v", err)
	}
	rRRHH := httptest.NewRequest(http.MethodGet, "/api/vec/bolsa/rrhh/inscripciones", nil)
	if _, err := p.PrepararLecturaRRHH(rRRHH, inscripcion.AccionListarRRHH, "",
		inscripcion.Filtro{Limite: 20, Estado: inscripcion.EstadoPendiente}, "es"); !errors.Is(err, inscripcion.ErrSesionAusente) {
		t.Fatalf("perfil empleado usó la vía RRHH: %v", err)
	}
	// El mismo certificado RRHH actúa como empleado sólo con perfil activo
	// ordinario distinto y concesión nominal de la acción propia.
	c.RRHH[0].perfilRef = identidad.PerfilRef
	p, err = NuevoPreparadorInscripcionBolsa(c)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.PrepararLecturaAspirante(r, inscripcion.AccionListarPropias, "", filtro, "es"); !errors.Is(err, inscripcion.ErrAccesoDenegado) {
		t.Fatalf("perfil RRHH usó la rama empleado: %v", err)
	}
	c.RRHH[0].perfilRef = "prf_rrhh_nominal"
	c.AcreditadorEmpleado = acreditadorEmpleadoInscripcionPrueba{identidad: AcreditacionEmpleadoInscripcionBolsa{
		EmpleadoRef: "emp_ajeno", PersonaRef: identidad.PersonaRef, PerfilRef: identidad.PerfilRef,
		CuentaRef: identidad.CuentaRef, ValidaHasta: ahora.Add(time.Minute)}}
	p, err = NuevoPreparadorInscripcionBolsa(c)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.PrepararLecturaAspirante(r, inscripcion.AccionListarPropias, "", filtro, "es"); !errors.Is(err, inscripcion.ErrAccesoDenegado) {
		t.Fatalf("vínculo de otro empleado: %v", err)
	}
	c.SelectorCanalAspirante = nil
	if _, err := NuevoPreparadorInscripcionBolsa(c); !errors.Is(err, inscripcion.ErrNoDisponible) {
		t.Fatalf("montaje empleado parcial: %v", err)
	}
}
