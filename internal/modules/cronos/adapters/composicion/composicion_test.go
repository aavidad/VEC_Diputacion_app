package composicion

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/application"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type revalidadorPrueba struct {
	a vecdomain.AutenticacionRevalidadaV1
}

func (r revalidadorPrueba) RevalidarAutenticacionActorV1(context.Context, vecdomain.SolicitudRevalidacionAutenticacionActorV1) (vecdomain.AutenticacionRevalidadaV1, error) {
	return r.a, nil
}

type resolutorContextoPrueba struct {
	r vecdomain.ResultadoContextoActorRegistradoV2
}

func (r resolutorContextoPrueba) ResolverContextoActorRegistradoV2(context.Context, vecdomain.SolicitudContextoActor) (vecdomain.ResultadoContextoActorRegistradoV2, error) {
	return r.r, nil
}

type relojPrueba struct{ ahora time.Time }

func (r relojPrueba) Ahora() time.Time { return r.ahora }

// identidadPrueba construye una identidad registrada sintética con los
// empleados indicados en el contexto, como la produce ContextoActor V2.
func identidadPrueba(t *testing.T, empleados ...string) IdentidadRegistradaCronos {
	t.Helper()
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	z := strings.Repeat("a", 24)
	cuenta := vecdomain.CuentaAutenticadaContextoActor{CuentaRef: "cta_" + z, Metodo: vecdomain.AuthMethodCertificate, Garantia: vecdomain.AuthAssuranceHigh}
	ac := vecdomain.AcreditacionProcedenciaComponenteContextoActorV1{ProcedenciaRef: "prc_" + z, ProcedenciaVersion: 1, ProcedenciaHuellaSHA256: strings.Repeat("a", 64), ProcedenciaAutoridad: vecdomain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1}
	vinculos := []vecdomain.VinculoReferenciaContextoActor{}
	procedencias := []vecdomain.ProcedenciaVinculoReferenciaContextoActorV1{}
	for i, e := range empleados {
		ref := "vin_" + strings.Repeat(string(rune('a'+i)), 24)
		vinculos = append(vinculos, vecdomain.VinculoReferenciaContextoActor{VinculoRef: ref, Version: 1, Tipo: vecdomain.TipoReferenciaContextoActorEmpleado, Referencia: e, Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour)})
		procedencias = append(procedencias, vecdomain.ProcedenciaVinculoReferenciaContextoActorV1{VinculoRef: ref, Version: 1, Tipo: vecdomain.TipoReferenciaContextoActorEmpleado, Referencia: e, AcreditacionProcedenciaComponenteContextoActorV1: ac})
	}
	in := vecdomain.InstantaneaContextoActor{VinculoRef: "vca_" + z, VinculoVersion: 1, CuentaRef: cuenta.CuentaRef, CuentaVersion: 1, PersonaRef: "per_" + z, PersonaVersion: 1, PerfilActivoRef: "prf_" + z, PerfilVersion: 1, Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour), Vinculos: vinculos}
	actor, err := vecdomain.NuevoContextoActor(cuenta, in, ahora)
	if err != nil {
		t.Fatal(err)
	}
	canon, _ := actor.RepresentacionCanonicaVinculadaV2()
	huella, _ := actor.HuellaSHA256VinculadaV2()
	man := vecdomain.ManifiestoProcedenciaContextoActorV1{Esquema: vecdomain.EsquemaManifiestoProcedenciaContextoActorV1, AutoridadEfectiva: vecdomain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1, Cuenta: vecdomain.ProcedenciaCuentaContextoActorV1{CuentaRef: cuenta.CuentaRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: ac}, Persona: vecdomain.ProcedenciaPersonaContextoActorV1{PersonaRef: actor.PersonaRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: ac}, Perfil: vecdomain.ProcedenciaPerfilContextoActorV1{PerfilRef: actor.PerfilActivoRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: ac}, Contexto: vecdomain.ProcedenciaVinculoContextoActorV1{VinculoRef: in.VinculoRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: ac}, Vinculos: procedencias}
	bm, _ := man.RepresentacionCanonicaV1()
	hm, _ := vecdomain.HuellaSHA256ManifiestoProcedenciaContextoActorV1(bm)
	res := vecdomain.ResultadoContextoActorRegistradoV2{RegistroContextoRef: "rca_" + z, Contexto: actor, RepresentacionCanonica: canon, HuellaSHA256: huella, ManifiestoProcedenciaCanonico: bm, ManifiestoProcedenciaHuellaSHA256: hm, AutoridadEfectiva: vecdomain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1, ResueltoEnAutoritativo: ahora}
	if err := res.Validar(); err != nil {
		t.Fatal(err)
	}
	auth := vecdomain.AutenticacionRevalidadaV1{AutenticacionRef: "aut_" + z, AutenticacionHuellaSHA256: strings.Repeat("a", 64), AsercionRef: "ase_" + z, SesionRef: "ses_" + z, ControlSesionRef: "cse_" + z, ControlSesionRevision: 1, ControlSesionHuellaSHA256: strings.Repeat("b", 64), CuentaRef: cuenta.CuentaRef, CuentaOrdinariaRef: cuenta.CuentaRef, Superficie: vecdomain.SuperficieAutenticacionInternaCorporativaV1, MetodoObservado: vecdomain.AuthMethodCertificate, GarantiaObservada: vecdomain.AuthAssuranceHigh, PoliticaGarantiaRef: "pga_" + z, PoliticaGarantiaHuellaSHA256: strings.Repeat("c", 64), AutenticacionVerificadaEn: ahora.Add(-time.Minute), SesionEmitidaEn: ahora.Add(-time.Minute), SesionRevalidadaEn: ahora.Add(-time.Second), SesionValidaHasta: ahora.Add(time.Minute)}
	v, err := vecdomain.CrearVinculoAutenticacionActorV2(context.Background(), revalidadorPrueba{auth}, vecdomain.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: auth.AutenticacionRef, SesionRef: auth.SesionRef}, resolutorContextoPrueba{res}, vecdomain.SolicitudContextoActor{Cuenta: cuenta, PerfilActivoRef: actor.PerfilActivoRef}, relojPrueba{ahora})
	if err != nil {
		t.Fatal(err)
	}
	return IdentidadRegistradaCronos{Contexto: res, Vinculo: v}
}

type identidadFija struct {
	id  IdentidadRegistradaCronos
	err error
}

func (i identidadFija) ResolverIdentidadRegistradaCronos(context.Context) (IdentidadRegistradaCronos, error) {
	return i.id, i.err
}

type emisorPrueba struct {
	err         error
	solicitudes []vecdomain.DatosSolicitudAutorizacionLigadaV3
}

func (e *emisorPrueba) EmitirMaterialAutorizacionAtestadaV3(_ context.Context, s vecdomain.SolicitudAutorizacionLigadaV3, _ vecdomain.ResultadoContextoActorRegistradoV2) (vecdomain.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	d, err := s.Datos()
	if err == nil {
		e.solicitudes = append(e.solicitudes, d)
	}
	return vecdomain.DecisionAutorizacionLigadaV3{}, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, e.err
}

func motivosPrueba() MotivosCronos {
	m := vecdomain.ReferenciaEntradaCatalogo{CatalogoID: "motivos_cronos", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("d", 64), EntradaClave: "motivo_" + strings.Repeat("5", 32)}
	return MotivosCronos{Saldo: m, Marcaje: m, Disponibilidad: m, Recuperacion: m}
}

func canalPrueba(t *testing.T) domain.AcreditacionCanalMarcaje {
	t.Helper()
	c, err := domain.NuevaAcreditacionCanalMarcaje(domain.DatosAcreditacionCanalMarcaje{PoliticaVersionRef: "politica:canal:cronos:v1", CanalRef: "portal-empleado-web", OrigenRef: domain.OrigenMarcajeRemoto, CalidadRef: "mtls-certificado"})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestComposicionRechazaDependenciasIncompletas(t *testing.T) {
	if _, err := NuevoAutorizadorCronos(nil, motivosPrueba()); err == nil {
		t.Fatal("autorizador sin emisor")
	}
	if _, err := NuevoAutorizadorCronos(&emisorPrueba{}, MotivosCronos{}); err == nil {
		t.Fatal("autorizador sin motivos")
	}
	a, err := NuevoAutorizadorCronos(&emisorPrueba{}, motivosPrueba())
	if err != nil {
		t.Fatal(err)
	}
	terminal, _ := domain.NuevaAcreditacionCanalMarcaje(domain.DatosAcreditacionCanalMarcaje{PoliticaVersionRef: "p", CanalRef: "c", OrigenRef: "terminal", CalidadRef: "q"})
	if _, err := NuevoResolutorPeticionCronos(identidadFija{}, a, terminal); err == nil {
		t.Fatal("canal no remoto aceptado")
	}
	if _, err := NuevoResolutorPeticionCronos(nil, a, canalPrueba(t)); err == nil {
		t.Fatal("sin identidad")
	}
	if _, err := NuevoProveedorDisponibilidadRemota(nil, identidadFija{}); err == nil {
		t.Fatal("proveedor sin autorizador")
	}
}

// Dos empleados no llegan aquí: el dominio de ContextoActor los rechaza y la
// resolución con alcance {empleado} deniega con motivo en la frontera.
func TestResolutorExigeUnEmpleadoConMotivo(t *testing.T) {
	a, _ := NuevoAutorizadorCronos(&emisorPrueba{}, motivosPrueba())
	req := httptest.NewRequest("GET", "/api/interna/cronos/saldos/propio?periodo=hoy", nil)
	for _, caso := range []struct {
		nombre   string
		fuente   identidadFija
		esperado error
	}{
		{"sin identidad", identidadFija{err: errors.New("sin sesión")}, ports.ErrDependenciaNoDisponible},
		{"sin empleado", identidadFija{id: identidadPrueba(t)}, ports.ErrEmpleadoNoAcreditado},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			r, err := NuevoResolutorPeticionCronos(caso.fuente, a, canalPrueba(t))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := r.ResolverConsultaSaldoPropio(req, ports.PeriodoSaldoHoy, "", ""); !errors.Is(err, caso.esperado) {
				t.Fatalf("saldo: %v", err)
			}
			if _, err := r.ResolverMarcajeRemoto(req); !errors.Is(err, caso.esperado) {
				t.Fatalf("remoto: %v", err)
			}
			if _, err := r.ResolverRecuperacionMarcajeRemoto(req); !errors.Is(err, caso.esperado) {
				t.Fatalf("recuperación: %v", err)
			}
		})
	}
}

func TestProveedoresEmitenContratoExactoYSoloParaLaPersona(t *testing.T) {
	empleado := "emp_" + strings.Repeat("e", 24)
	id := identidadPrueba(t, empleado)
	emisor := &emisorPrueba{err: errors.Join(errors.New("no disponible"), vecports.ErrDenegacionExplicitaAutorizacionLigadaV3)}
	a, _ := NuevoAutorizadorCronos(emisor, motivosPrueba())
	r, _ := NuevoResolutorPeticionCronos(identidadFija{id: id}, a, canalPrueba(t))
	req := httptest.NewRequest("GET", "/", nil)
	orden, err := r.ResolverConsultaSaldoPropio(req, ports.PeriodoSaldoHoy, "", "")
	if err != nil {
		t.Fatal(err)
	}
	actor := id.Contexto.Contexto
	saldo := domain.MaterialConsultaSaldoPropio{ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, EmpleadoRef: empleado, Desde: "2026-09-25", Hasta: "2026-09-25", ZonaHoraria: domain.ZonaSaldoPeninsula}
	if _, err := orden.ProveedorMaterial().ProveerMaterialConsultaSaldoPropio(context.Background(), saldo); !errors.Is(err, vecdomain.ErrPermissionDenied) {
		t.Fatal("denegación explícita del PDP no conservada", err)
	}
	datos := emisor.solicitudes[0]
	recurso, _ := application.RecursoConsultaSaldoPropio(saldo)
	if datos.Accion != application.AccionConsultarSaldoPropio || datos.Finalidad != application.FinalidadConsultarSaldoPropio || datos.Recurso.Referencia != recurso.Referencia || datos.Recurso.Atributos["material_sha256"] != recurso.Atributos["material_sha256"] {
		t.Fatalf("contrato V3 distinto: %+v", datos)
	}
	ajeno := saldo
	ajeno.EmpleadoRef = "emp_" + strings.Repeat("z", 24)
	if _, err := orden.ProveedorMaterial().ProveerMaterialConsultaSaldoPropio(context.Background(), ajeno); !errors.Is(err, ports.ErrDependenciaNoDisponible) || len(emisor.solicitudes) != 1 {
		t.Fatal("emite para otro empleado", err)
	}
	remoto, err := r.ResolverMarcajeRemoto(req)
	if err != nil || remoto.CanalAcreditado != canalPrueba(t) {
		t.Fatal("canal remoto no acreditado por la composición", err)
	}
	instante := time.Now().UTC().Truncate(time.Microsecond)
	marcaje := domain.MaterialAutorizacionMarcajePropio{ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, EmpleadoRef: empleado, ClaveOperacion: "clave-remota-0001", Movimiento: domain.PunchEntry, InstanteUTC: instante, Canal: remoto.CanalAcreditado}
	emisor.err = errors.New("firmante caído")
	if _, err := remoto.OrdenConsumo.ProveedorMaterial().ProveerMaterialMarcajePropio(context.Background(), marcaje); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal("un fallo del emisor no es dependencia", err)
	}
	if d := emisor.solicitudes[1]; d.Accion != application.AccionRegistrarMarcajePropio || d.Finalidad != application.FinalidadRegistrarMarcajePropio || d.Recurso.Referencia != "marcaje:cronos:clave-remota-0001" {
		t.Fatalf("contrato de marcaje distinto: %+v", d)
	}
	recuperacion, err := r.ResolverRecuperacionMarcajeRemoto(req)
	if err != nil {
		t.Fatal(err)
	}
	material := domain.MaterialRecuperacionMarcajeRemoto{ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, EmpleadoRef: empleado, ClaveOperacion: "clave-remota-0001", Movimiento: domain.PunchEntry, Canal: recuperacion.CanalAcreditado}
	_, _ = recuperacion.OrdenLectura.ProveedorMaterial().ProveerMaterialRecuperacionMarcajeRemoto(context.Background(), material)
	if d := emisor.solicitudes[2]; d.Accion != application.AccionRecuperarMarcajeRemoto || d.Finalidad != application.FinalidadRecuperarMarcajeRemoto {
		t.Fatalf("contrato de recuperación distinto: %+v", d)
	}
	disponibilidad, _ := NuevoProveedorDisponibilidadRemota(a, identidadFija{id: id})
	estado := domain.MaterialDisponibilidadMarcajeRemoto{ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, EmpleadoRef: empleado, InstanteUTC: instante, Canal: remoto.CanalAcreditado}
	_, _ = disponibilidad.ProveerMaterialDisponibilidadMarcajeRemoto(context.Background(), estado)
	if d := emisor.solicitudes[3]; d.Accion != application.AccionConsultarDisponibilidadRemota || d.Recurso.Referencia != "teletrabajo:cronos:"+empleado {
		t.Fatalf("contrato de disponibilidad distinto: %+v", d)
	}
	estado.PerfilRef = "prf_" + strings.Repeat("q", 24)
	if _, err := disponibilidad.ProveerMaterialDisponibilidadMarcajeRemoto(context.Background(), estado); !errors.Is(err, ports.ErrDependenciaNoDisponible) || len(emisor.solicitudes) != 4 {
		t.Fatal("emite para otro perfil", err)
	}
}
