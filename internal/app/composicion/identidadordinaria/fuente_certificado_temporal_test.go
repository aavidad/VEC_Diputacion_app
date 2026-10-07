package identidadordinaria

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	core "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/pruebas"
)

const empleadoTemporalPrueba = "emp_0123456789abcdefghijkl"
const cuentaAjenaTemporalPrueba = "cta_ajena0123456789abcdefghijkl"

type registroDosCuentasTemporalPrueba struct {
	base            *registroPrueba
	cuentaSiguiente string
}

func (r *registroDosCuentasTemporalPrueba) ConsumirAsercionYRegistrar(ctx context.Context, alta httpseguridad.AltaSesionAtomica) (httpseguridad.ConfirmacionAltaSesion, error) {
	c, err := r.base.ConsumirAsercionYRegistrar(ctx, alta)
	if err != nil {
		return c, err
	}
	if r.cuentaSiguiente != "" {
		c.AutenticacionRef = "aut_ajena0123456789abcdefghijkl"
		c.AsercionRef = "ase_ajena0123456789abcdefghijkl"
		c.SesionRef = "ses_ajena0123456789abcdefghijkl"
		c.ControlSesionRef = "cse_ajena0123456789abcdefghijkl"
		c.CuentaRef = r.cuentaSiguiente
		c.CuentaOrdinariaRef = r.cuentaSiguiente
		r.base.alta = c
		r.cuentaSiguiente = ""
	}
	return c, nil
}

func (r *registroDosCuentasTemporalPrueba) ComprobarSesionYCuentaActivas(ctx context.Context, consulta httpseguridad.ConsultaSesionActiva) error {
	return r.base.ComprobarSesionYCuentaActivas(ctx, consulta)
}

type controlTemporalPrueba struct {
	verificador *verificadorPrueba
	registro    *registroDosCuentasTemporalPrueba
	canal       httpseguridad.CanalProxyAutenticado
}

type resolutorCancelaTemporalPrueba struct {
	base     *resolutorPrueba
	cancelar context.CancelFunc
}

func (r resolutorCancelaTemporalPrueba) ResolverContextoActorRegistradoV2(ctx context.Context, solicitud core.SolicitudContextoActor) (core.ResultadoContextoActorRegistradoV2, error) {
	resultado, err := r.base.ResolverContextoActorRegistradoV2(ctx, solicitud)
	if err == nil {
		r.cancelar()
	}
	return resultado, err
}

type autorizacionCancelaTemporalPrueba struct {
	base    *autorizacionPrueba
	despues func(context.Context)
}

func (a autorizacionCancelaTemporalPrueba) ObtenerInstantaneaAutorizacion(ctx context.Context, principal, perfil string) (core.InstantaneaAutorizacion, error) {
	snapshot, err := a.base.ObtenerInstantaneaAutorizacion(ctx, principal, perfil)
	if err == nil {
		a.despues(ctx)
	}
	return snapshot, err
}

type relojCancelaTemporalPrueba struct {
	base     *relojPrueba
	cancelar context.CancelFunc
	llamadas int
}

func (r *relojCancelaTemporalPrueba) Ahora() time.Time {
	r.llamadas++
	if r.llamadas == 3 {
		r.cancelar()
	}
	return r.base.Ahora()
}

func contextoConEmpleadoTemporalPrueba(t *testing.T) core.ResultadoContextoActorRegistradoV2 {
	return contextoConEmpleadoTemporalPersonaPrueba(t, personaPrueba)
}

func contextoConEmpleadoTemporalPersonaPrueba(t *testing.T, persona string) core.ResultadoContextoActorRegistradoV2 {
	t.Helper()
	r, _, err := pruebas.NuevoContextoRegistradoYVinculoV2(instantePrueba.Add(time.Minute), persona,
		perfilPrueba, core.AuthMethodCertificate, core.AuthAssuranceSubstantial)
	if err != nil {
		t.Fatal(err)
	}
	v := core.VinculoReferenciaContextoActor{VinculoRef: "vin_0123456789abcdefghijkl", Version: 1,
		Tipo: core.TipoReferenciaContextoActorEmpleado, Referencia: empleadoTemporalPrueba,
		Estado: core.EstadoVinculoContextoActorActivo, VigenteDesde: instantePrueba.Add(-time.Hour),
		VigenteHasta: instantePrueba.Add(30 * time.Minute)}
	r.Contexto.Instantanea.Vinculos = []core.VinculoReferenciaContextoActor{v}
	r.RepresentacionCanonica, err = r.Contexto.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	r.HuellaSHA256, err = r.Contexto.HuellaSHA256VinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	m, err := core.RehidratarManifiestoProcedenciaContextoActorV1(r.ManifiestoProcedenciaCanonico)
	if err != nil {
		t.Fatal(err)
	}
	m.Vinculos = []core.ProcedenciaVinculoReferenciaContextoActorV1{{VinculoRef: v.VinculoRef,
		Version: v.Version, Tipo: v.Tipo, Referencia: v.Referencia,
		AcreditacionProcedenciaComponenteContextoActorV1: m.Contexto.AcreditacionProcedenciaComponenteContextoActorV1}}
	r.ManifiestoProcedenciaCanonico, err = m.RepresentacionCanonicaV1()
	if err != nil {
		t.Fatal(err)
	}
	r.ManifiestoProcedenciaHuellaSHA256, err = core.HuellaSHA256ManifiestoProcedenciaContextoActorV1(r.ManifiestoProcedenciaCanonico)
	if err != nil || r.Validar() != nil {
		t.Fatalf("resultado con empleado no válido: %v, %v", err, r.Validar())
	}
	return r
}

func nuevoEntornoCertificadoTemporalPrueba(t *testing.T) (*entornoPrueba, *FuenteCertificadoTemporal, *controlTemporalPrueba) {
	t.Helper()
	reloj := &relojPrueba{ahora: instantePrueba}
	registro := &registroPrueba{}
	selector := &registroDosCuentasTemporalPrueba{base: registro}
	cfg := httpseguridad.ConfiguracionSuperficie{Superficie: httpseguridad.SuperficieInternaCorporativa,
		ZonaRed: httpseguridad.ZonaRedInterna, DireccionEscucha: "127.0.0.1:8443",
		Audiencia: "vec-interna", EmisorIdentidad: "https://idp.test", RedesPermitidas: []string{"127.0.0.0/8"},
		IdentidadesSANProxyPermitidas: []string{"dns:proxy.test"}, DuracionMaximaAsercion: 3 * time.Minute,
		EdadMaximaAutenticacion: 15 * time.Minute, ToleranciaReloj: 20 * time.Second,
		MetodosAdmitidos:          []httpseguridad.MetodoAutenticacion{httpseguridad.MetodoCertificado},
		FactoresRequeridos:        []httpseguridad.MetodoAutenticacion{httpseguridad.MetodoCertificado},
		MinimoFactoresVerificados: 1, MinimoGruposCriptograficosDistintos: 1,
		GarantiaMinima:            core.AuthAssuranceSubstantial,
		PoliticaInterna:           httpseguridad.PoliticaInternaDesarrolloCertificadoPersonal,
		RetiradaPoliticaInternaEn: instantePrueba.Add(time.Hour)}
	verificador := &verificadorPrueba{}
	servicio, err := httpseguridad.NuevoServicioIdentidad(cfg, verificador, evaluadorPrueba{core.AuthAssuranceSubstantial}, selector, reloj)
	if err != nil {
		t.Fatal(err)
	}
	canal, err := servicio.AutenticarCanalTLSMutuo(estadoTLSPrueba(t))
	if err != nil {
		t.Fatal(err)
	}
	verificada := instantePrueba.Add(-time.Minute)
	verificador.asercion = httpseguridad.AsercionProxyIdentidad{ID: "ase-id-1", Emisor: cfg.EmisorIdentidad,
		Audiencia: cfg.Audiencia, Superficie: cfg.Superficie, SujetoID: personaPrueba,
		Cuenta:   httpseguridad.CuentaAcceso{ID: "cuenta-1", SujetoVinculadoID: personaPrueba},
		SesionID: "ses-id-1", CanalVinculadoRef: canal.ReferenciaVinculacion(),
		AutenticacionVerificadaEn: verificada, EmitidaEn: verificada, NoAntesDe: verificada,
		ExpiraEn: instantePrueba.Add(time.Minute), MetodoPrimario: httpseguridad.MetodoCertificado,
		ACRVerificado: httpseguridad.ACRCertificadoPersonalDesarrolloProtegido,
		Factores: []httpseguridad.FactorAutenticacion{{Metodo: httpseguridad.MetodoCertificado,
			SujetoVinculadoID: personaPrueba, CredencialRef: "cert:1", EvidenciaRef: "cert:1",
			GrupoCriptograficoRef: "grupo:tarjeta", VerificadoEn: verificada}}}
	credencial, err := httpseguridad.NuevaCredencialProxy([]byte("asercion-protegida"), canal)
	if err != nil {
		t.Fatal(err)
	}
	identidad, err := servicio.Resolver(context.Background(), credencial)
	if err != nil {
		t.Fatal(err)
	}
	capsula, err := servicio.ProyectarCapsulaIdentidadPeticion(context.Background(), identidad, canal)
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := servicio.VincularCapsulaIdentidadPeticion(context.Background(), capsula, canal)
	if err != nil {
		t.Fatal(err)
	}
	revalidador := &revalidadorPrueba{alta: registro.alta}
	resolutor := &resolutorPrueba{resultado: contextoConEmpleadoTemporalPrueba(t)}
	autorizacion := &autorizacionPrueba{snapshot: snapshotPrueba()}
	e := &entornoPrueba{servicio: servicio, ctx: ctx, reloj: reloj, registro: registro,
		revalidador: revalidador, resolutor: resolutor, autorizacion: autorizacion}
	f, err := NuevaFuenteCertificadoTemporal(ConfiguracionFuenteCertificadoTemporal{Identidad: servicio,
		Revalidador: revalidador, Resolutor: resolutor, Autorizacion: autorizacion, Reloj: reloj,
		PorCuenta: map[string]PerfilNominal{
			cuentaPrueba:              {PerfilActivoRef: perfilPrueba, VersionRolRef: "rol:tecnico_rpt:v2"},
			cuentaAjenaTemporalPrueba: {PerfilActivoRef: perfilPrueba, VersionRolRef: "rol:tecnico_rpt:v2"},
		},
		Politica: PoliticaCertificadoTemporal{Referencia: "pga_0123456789abcdefghijkl",
			HuellaSHA256: strings.Repeat("a", 64), RetiradaEn: cfg.RetiradaPoliticaInternaEn}})
	if err != nil {
		t.Fatal(err)
	}
	return e, f, &controlTemporalPrueba{verificador: verificador, registro: selector, canal: canal}
}

func TestCertificadoTemporalAbreYRevalidaMismaSesion(t *testing.T) {
	e, f, _ := nuevoEntornoCertificadoTemporalPrueba(t)
	s, err := f.Abrir(e.ctx)
	if err != nil {
		t.Fatal(err)
	}
	v, r, a, err := s.Contexto()
	if err != nil || v.ValidarPara(r) != nil || a.Validar() != nil || s.EmpleadoRef() != empleadoTemporalPrueba ||
		len(s.CanalSHA256()) != 64 || !s.VigenteHasta().Equal(instantePrueba.Add(time.Minute)) {
		t.Fatalf("sesión temporal incompleta: %v", err)
	}
	a.VersionRol.Concesiones[0].Accion = "accion_ajena"
	e.autorizacion.snapshot.AsignacionPerfil.Ambitos[0].Valores[0] = "organizacion_ajena"
	_, _, intacta, err := s.Contexto()
	if err != nil || intacta.VersionRol.Concesiones[0].Accion != "rpt.consultar" ||
		intacta.AsignacionPerfil.Ambitos[0].Valores[0] != "org_0123456789abcdefghijkl" {
		t.Fatalf("la instantánea de sesión conserva alias mutables: %v", err)
	}
	esperados, err := s.Esperados()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Revalidar(e.ctx, esperados); err != nil || e.revalidador.llamadas != 2 ||
		e.resolutor.llamadas != 2 || e.autorizacion.llamadas != 2 {
		t.Fatalf("revalidación no consultó las autoridades: %v", err)
	}
	for _, mutar := range []func(*EsperadosSesion){
		func(x *EsperadosSesion) { x.personaRef = "per_ajena0123456789abcdefghijkl" },
		func(x *EsperadosSesion) { x.cuentaRef = "cta_ajena0123456789abcdefghijkl" },
		func(x *EsperadosSesion) { x.perfilRef = "prf_ajeno0123456789abcdefghijkl" },
		func(x *EsperadosSesion) { x.sesionRef = "ses_ajena0123456789abcdefghijkl" },
		func(x *EsperadosSesion) { x.empleadoRef = "emp_ajeno0123456789abcdefghijkl" },
		func(x *EsperadosSesion) { x.canalSHA256 = strings.Repeat("f", 64) },
	} {
		ajeno := esperados
		mutar(&ajeno)
		if _, err := f.Revalidar(e.ctx, ajeno); !errors.Is(err, ErrCertificadoTemporalNoDisponible) {
			t.Fatalf("se admitió sesión ajena: %v", err)
		}
	}
}

func TestCertificadoTemporalNoCruzaDosCuentasDelMismoServicio(t *testing.T) {
	e, f, control := nuevoEntornoCertificadoTemporalPrueba(t)
	primera, err := f.Abrir(e.ctx)
	if err != nil {
		t.Fatal(err)
	}
	esperados, err := primera.Esperados()
	if err != nil {
		t.Fatal(err)
	}
	segunda := contextoConEmpleadoTemporalPrueba(t)
	segunda.Contexto.Instantanea.CuentaRef = cuentaAjenaTemporalPrueba
	segunda.RepresentacionCanonica, err = segunda.Contexto.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	segunda.HuellaSHA256, err = segunda.Contexto.HuellaSHA256VinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	manifiesto, err := core.RehidratarManifiestoProcedenciaContextoActorV1(segunda.ManifiestoProcedenciaCanonico)
	if err != nil {
		t.Fatal(err)
	}
	manifiesto.Cuenta.CuentaRef = cuentaAjenaTemporalPrueba
	segunda.ManifiestoProcedenciaCanonico, err = manifiesto.RepresentacionCanonicaV1()
	if err != nil {
		t.Fatal(err)
	}
	segunda.ManifiestoProcedenciaHuellaSHA256, err = core.HuellaSHA256ManifiestoProcedenciaContextoActorV1(segunda.ManifiestoProcedenciaCanonico)
	if err != nil || segunda.Validar() != nil {
		t.Fatalf("segunda cuenta F1 inválida: %v, %v", err, segunda.Validar())
	}
	control.registro.cuentaSiguiente = cuentaAjenaTemporalPrueba
	control.verificador.asercion.ID = "ase-id-2"
	control.verificador.asercion.SesionID = "ses-id-2"
	canal := control.canal
	control.verificador.asercion.CanalVinculadoRef = canal.ReferenciaVinculacion()
	credencial, err := httpseguridad.NuevaCredencialProxy([]byte("asercion-protegida-b"), canal)
	if err != nil {
		t.Fatal(err)
	}
	identidad, err := e.servicio.Resolver(context.Background(), credencial)
	if err != nil {
		t.Fatal(err)
	}
	capsula, err := e.servicio.ProyectarCapsulaIdentidadPeticion(context.Background(), identidad, canal)
	if err != nil {
		t.Fatal(err)
	}
	ctxB, err := e.servicio.VincularCapsulaIdentidadPeticion(context.Background(), capsula, canal)
	if err != nil {
		t.Fatal(err)
	}
	e.revalidador.alta = control.registro.base.alta
	e.resolutor.resultado = segunda
	actual, err := f.Abrir(ctxB)
	if err != nil {
		t.Fatalf("la cuenta B válida debe poder abrir su propia sesión: %v", err)
	}
	if actual.EmpleadoRef() != empleadoTemporalPrueba || actual.CanalSHA256() != primera.CanalSHA256() {
		t.Fatal("la cuenta B legítima comparte canal, pero no debe compartir sesión ni actor")
	}
	if _, err := f.Revalidar(ctxB, esperados); !errors.Is(err, ErrCertificadoTemporalNoDisponible) {
		t.Fatalf("la cuenta B continuó la sesión A: %v", err)
	}
}

func TestCertificadoTemporalDeniegaRevocacionRetiradaYEmpleadoAusente(t *testing.T) {
	for _, tc := range []struct {
		nombre string
		mutar  func(*entornoPrueba, *FuenteCertificadoTemporal)
	}{
		{"revocacion", func(e *entornoPrueba, _ *FuenteCertificadoTemporal) { e.registro.revocada = true }},
		{"retirada", func(e *entornoPrueba, f *FuenteCertificadoTemporal) { e.reloj.ahora = f.politica.RetiradaEn }},
		{"empleado ausente", func(e *entornoPrueba, _ *FuenteCertificadoTemporal) {
			e.resolutor.resultado, _, _ = pruebas.NuevoContextoRegistradoYVinculoV2(instantePrueba.Add(time.Minute), personaPrueba,
				perfilPrueba, core.AuthMethodCertificate, core.AuthAssuranceSubstantial)
		}},
		{"empleado ambiguo", func(e *entornoPrueba, _ *FuenteCertificadoTemporal) {
			v := e.resolutor.resultado.Contexto.Instantanea.Vinculos[0]
			v.VinculoRef = "vin_ajeno0123456789abcdefghijkl"
			v.Referencia = "emp_ajeno0123456789abcdefghijkl"
			e.resolutor.resultado.Contexto.Instantanea.Vinculos = append(e.resolutor.resultado.Contexto.Instantanea.Vinculos, v)
		}},
		{"persona F1 distinta", func(e *entornoPrueba, _ *FuenteCertificadoTemporal) {
			e.resolutor.resultado = contextoConEmpleadoTemporalPersonaPrueba(t, "per_ajena0123456789abcdefghijkl")
		}},
		{"politica distinta", func(_ *entornoPrueba, f *FuenteCertificadoTemporal) {
			f.politica.HuellaSHA256 = strings.Repeat("b", 64)
		}},
		{"perfil retirado", func(e *entornoPrueba, _ *FuenteCertificadoTemporal) {
			e.autorizacion.snapshot.ControlVigenciaVersionRol.Estado = core.EstadoControlVigenciaVersionRolRetirada
			e.autorizacion.snapshot.ControlVigenciaVersionRol.ActoRef = "acto:retirada"
			e.autorizacion.snapshot.ControlVigenciaVersionRol.MotivoCodigo = "baja"
		}},
	} {
		t.Run(tc.nombre, func(t *testing.T) {
			e, f, _ := nuevoEntornoCertificadoTemporalPrueba(t)
			tc.mutar(e, f)
			if _, err := f.Abrir(e.ctx); !errors.Is(err, ErrCertificadoTemporalNoDisponible) {
				t.Fatalf("se admitió identidad o vigencia inválida: %v", err)
			}
		})
	}
}

func TestSesionCertificadoTemporalNoFiltraNiSerializaIdentificadores(t *testing.T) {
	e, f, _ := nuevoEntornoCertificadoTemporalPrueba(t)
	s, err := f.Abrir(e.ctx)
	if err != nil {
		t.Fatal(err)
	}
	esperados, err := s.Esperados()
	if err != nil {
		t.Fatal(err)
	}
	for _, valor := range []any{s, esperados} {
		for _, modo := range []string{"%v", "%+v", "%#v"} {
			impreso := fmt.Sprintf(modo, valor)
			if strings.Contains(impreso, personaPrueba) || strings.Contains(impreso, cuentaPrueba) ||
				strings.Contains(impreso, s.CanalSHA256()) {
				t.Fatalf("identificador filtrado en formato %s", modo)
			}
		}
		if _, err := json.Marshal(valor); !errors.Is(err, ErrCertificadoTemporalNoDisponible) {
			t.Fatalf("la sesión interna se serializó: %v", err)
		}
	}
}

func TestCertificadoTemporalConservaCausaDeCancelacionTrasExitoDeAutoridad(t *testing.T) {
	for _, caso := range []struct {
		nombre   string
		preparar func(*entornoPrueba, *FuenteCertificadoTemporal, context.CancelFunc)
	}{
		{"resolver F1", func(e *entornoPrueba, f *FuenteCertificadoTemporal, cancelar context.CancelFunc) {
			f.resolutor = resolutorCancelaTemporalPrueba{base: e.resolutor, cancelar: cancelar}
		}},
		{"fuente V3", func(e *entornoPrueba, f *FuenteCertificadoTemporal, cancelar context.CancelFunc) {
			f.autorizacion = autorizacionCancelaTemporalPrueba{base: e.autorizacion,
				despues: func(context.Context) { cancelar() }}
		}},
		{"reloj tras fuente V3", func(e *entornoPrueba, f *FuenteCertificadoTemporal, cancelar context.CancelFunc) {
			f.reloj = &relojCancelaTemporalPrueba{base: e.reloj, cancelar: cancelar}
		}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			e, f, _ := nuevoEntornoCertificadoTemporalPrueba(t)
			ctx, cancelar := context.WithCancel(e.ctx)
			defer cancelar()
			caso.preparar(e, f, cancelar)
			sesion, err := f.Abrir(ctx)
			if err == nil || !errors.Is(err, context.Canceled) || !errors.Is(err, ErrCertificadoTemporalNoDisponible) ||
				err.Error() != ErrCertificadoTemporalNoDisponible.Error() || sesion.vinculo.Validar() == nil ||
				e.resolutor.llamadas != 1 {
				t.Fatalf("cancelación sin causa, mensaje opaco o salida parcial: %v", err)
			}
		})
	}
}

func TestCertificadoTemporalConservaPlazoTrasExitoDeFuenteV3(t *testing.T) {
	e, f, _ := nuevoEntornoCertificadoTemporalPrueba(t)
	ctx, cancelar := context.WithTimeout(e.ctx, 100*time.Millisecond)
	defer cancelar()
	f.autorizacion = autorizacionCancelaTemporalPrueba{base: e.autorizacion,
		despues: func(ctx context.Context) { <-ctx.Done() }}
	sesion, err := f.Abrir(ctx)
	if err == nil || !errors.Is(err, context.DeadlineExceeded) || !errors.Is(err, ErrCertificadoTemporalNoDisponible) ||
		err.Error() != ErrCertificadoTemporalNoDisponible.Error() || sesion.vinculo.Validar() == nil ||
		e.autorizacion.llamadas != 1 {
		t.Fatalf("vencimiento sin causa, mensaje opaco o salida parcial: %v", err)
	}
}
