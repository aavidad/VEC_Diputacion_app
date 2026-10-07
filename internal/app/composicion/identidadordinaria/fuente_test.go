package identidadordinaria

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	core "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/pruebas"
)

var instantePrueba = time.Date(2026, 10, 7, 10, 0, 0, 0, time.UTC)

const cuentaPrueba = "cta_0123456789abcdefghijkl"
const perfilPrueba = "prf_0123456789abcdefghijkl"
const personaPrueba = "per_0123456789abcdefghijkl"

type relojPrueba struct{ ahora time.Time }

func (r *relojPrueba) Ahora() time.Time { return r.ahora }

type verificadorPrueba struct {
	asercion httpseguridad.AsercionProxyIdentidad
}

func (v verificadorPrueba) Verificar(context.Context, []byte) (httpseguridad.AsercionProxyIdentidad, error) {
	return v.asercion, nil
}

type evaluadorPrueba struct{ garantia core.AuthAssurance }

func (e evaluadorPrueba) Evaluar(context.Context, httpseguridad.EntradaEvaluacionGarantia) (httpseguridad.ResultadoEvaluacionGarantia, error) {
	return httpseguridad.ResultadoEvaluacionGarantia{Garantia: e.garantia, PoliticaRef: "pga_0123456789abcdefghijkl", HuellaPolitica: "sha256:" + strings.Repeat("a", 64)}, nil
}

type registroPrueba struct {
	alta     httpseguridad.ConfirmacionAltaSesion
	revocada bool
}

func (r *registroPrueba) ConsumirAsercionYRegistrar(_ context.Context, a httpseguridad.AltaSesionAtomica) (httpseguridad.ConfirmacionAltaSesion, error) {
	r.alta = httpseguridad.ConfirmacionAltaSesion{
		AutenticacionRef: "aut_0123456789abcdefghijkl", AsercionRef: "ase_0123456789abcdefghijkl",
		SesionRef: "ses_0123456789abcdefghijkl", ControlSesionRef: "cse_0123456789abcdefghijkl",
		ControlSesionRevision: 1, ControlSesionEstado: httpseguridad.EstadoControlSesionActiva,
		ControlSesionHuellaSHA256: strings.Repeat("c", 64), CuentaRef: cuentaPrueba,
		CuentaOrdinariaRef: cuentaPrueba, SesionRevalidadaEn: a.SesionEmitidaEn,
		SesionValidaHasta: a.AsercionExpiraEn, AltaConfirmada: a,
	}
	return r.alta, nil
}
func (r *registroPrueba) ComprobarSesionYCuentaActivas(_ context.Context, q httpseguridad.ConsultaSesionActiva) error {
	if r.revocada || q.Validar() != nil || q.SesionRef != r.alta.SesionRef {
		return errors.New("sesion revocada")
	}
	return nil
}

type revalidadorPrueba struct {
	alta     httpseguridad.ConfirmacionAltaSesion
	llamadas int
	err      error
}

func (r *revalidadorPrueba) RevalidarAutenticacionActorV1(_ context.Context, s core.SolicitudRevalidacionAutenticacionActorV1) (core.AutenticacionRevalidadaV1, error) {
	r.llamadas++
	if r.err != nil {
		return core.AutenticacionRevalidadaV1{}, r.err
	}
	if s.AutenticacionRef != r.alta.AutenticacionRef || s.SesionRef != r.alta.SesionRef {
		return core.AutenticacionRevalidadaV1{}, errors.New("sesion distinta")
	}
	a := r.alta.AltaConfirmada
	return core.AutenticacionRevalidadaV1{
		AutenticacionRef: r.alta.AutenticacionRef, AutenticacionHuellaSHA256: a.AutenticacionHuellaSHA256,
		AsercionRef: r.alta.AsercionRef, SesionRef: r.alta.SesionRef,
		ControlSesionRef: r.alta.ControlSesionRef, ControlSesionRevision: r.alta.ControlSesionRevision,
		ControlSesionHuellaSHA256: r.alta.ControlSesionHuellaSHA256,
		CuentaRef:                 r.alta.CuentaRef, CuentaOrdinariaRef: r.alta.CuentaOrdinariaRef,
		Superficie:      core.SuperficieAutenticacionInternaCorporativaV1,
		MetodoObservado: a.MetodoObservado, GarantiaObservada: a.GarantiaObservada,
		PoliticaGarantiaRef: a.PoliticaGarantiaRef, PoliticaGarantiaHuellaSHA256: a.PoliticaGarantiaHuellaSHA256,
		AutenticacionVerificadaEn: a.AutenticacionVerificadaEn, SesionEmitidaEn: a.SesionEmitidaEn,
		SesionRevalidadaEn: r.alta.SesionRevalidadaEn, SesionValidaHasta: r.alta.SesionValidaHasta,
	}, nil
}

type resolutorPrueba struct {
	resultado core.ResultadoContextoActorRegistradoV2
	llamadas  int
}

func (r *resolutorPrueba) ResolverContextoActorRegistradoV2(_ context.Context, s core.SolicitudContextoActor) (core.ResultadoContextoActorRegistradoV2, error) {
	r.llamadas++
	if s.PerfilActivoRef != r.resultado.Contexto.PerfilActivoRef || s.Cuenta.CuentaRef != r.resultado.Contexto.Instantanea.CuentaRef {
		return core.ResultadoContextoActorRegistradoV2{}, errors.New("contexto ajeno")
	}
	return r.resultado.Clonar()
}

type autorizacionPrueba struct {
	snapshot core.InstantaneaAutorizacion
	llamadas int
	err      error
}

func (a *autorizacionPrueba) ObtenerInstantaneaAutorizacion(_ context.Context, principal, perfil string) (core.InstantaneaAutorizacion, error) {
	a.llamadas++
	if a.err != nil {
		return core.InstantaneaAutorizacion{}, a.err
	}
	if principal != personaPrueba || perfil != perfilPrueba {
		return core.InstantaneaAutorizacion{}, errors.New("selector ajeno")
	}
	return a.snapshot, nil
}

func snapshotPrueba() core.InstantaneaAutorizacion {
	rol := core.VersionRol{RolID: "tecnico_rpt", Version: 2, Nombre: "Técnico RPT",
		Estado: core.EstadoVersionRolPublicada, PublicadaPor: "seguridad", PublicadaEn: instantePrueba.Add(-time.Hour),
		Concesiones: []core.ConcesionRol{{Accion: "rpt.consultar", ModuloID: "rpt", TipoRecurso: "categoria",
			Finalidades: []string{"gestion_rpt"}, GarantiaMinima: core.AuthAssuranceHigh}},
	}
	asignacion := core.AsignacionPerfil{AsignacionID: "asig-rpt-1", Version: 1, PerfilActivoRef: perfilPrueba,
		PrincipalID: personaPrueba, VersionRolRef: rol.Referencia(), Estado: core.EstadoAsignacionPerfilActiva,
		Ambitos:      []core.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{"org_0123456789abcdefghijkl"}}},
		VigenteDesde: instantePrueba.Add(-30 * time.Minute), VigenteHasta: instantePrueba.Add(time.Hour),
		EmitidaPor: "seguridad", EmitidaEn: instantePrueba.Add(-time.Hour),
	}
	huella, _ := core.HuellaCatalogoPoliticasAutorizacion(nil)
	return core.InstantaneaAutorizacion{AsignacionPerfil: asignacion, VersionRol: rol,
		ControlVigenciaVersionRol: core.ControlVigenciaVersionRol{VersionRolRef: rol.Referencia(), Revision: 1,
			Estado: core.EstadoControlVigenciaVersionRolHabilitada, ActualizadoPor: "seguridad", ActualizadoEn: instantePrueba.Add(-30 * time.Minute)},
		RevisionCatalogoPoliticas: 1, CatalogoPoliticasHuellaSHA256: huella}
}

type entornoPrueba struct {
	servicio     *httpseguridad.ServicioIdentidad
	ctx          context.Context
	reloj        *relojPrueba
	registro     *registroPrueba
	revalidador  *revalidadorPrueba
	resolutor    *resolutorPrueba
	autorizacion *autorizacionPrueba
	config       ConfiguracionFuente
}

func nuevoEntornoPrueba(t *testing.T) *entornoPrueba {
	return nuevoEntornoConPoliticaPrueba(t, false)
}

func nuevoEntornoConPoliticaPrueba(t *testing.T, temporal bool) *entornoPrueba {
	t.Helper()
	r := &relojPrueba{ahora: instantePrueba}
	registro := &registroPrueba{}
	cfg := httpseguridad.ConfiguracionSuperficie{Superficie: httpseguridad.SuperficieInternaCorporativa,
		ZonaRed: httpseguridad.ZonaRedInterna, DireccionEscucha: "127.0.0.1:8443",
		Audiencia: "vec-interna", EmisorIdentidad: "https://idp.test", RedesPermitidas: []string{"127.0.0.0/8"},
		IdentidadesSANProxyPermitidas: []string{"dns:proxy.test"}, DuracionMaximaAsercion: 3 * time.Minute,
		EdadMaximaAutenticacion: 15 * time.Minute, ToleranciaReloj: 20 * time.Second,
		MetodosAdmitidos:          []httpseguridad.MetodoAutenticacion{httpseguridad.MetodoKerberos, httpseguridad.MetodoCertificado},
		FactoresRequeridos:        []httpseguridad.MetodoAutenticacion{httpseguridad.MetodoKerberos, httpseguridad.MetodoCertificado},
		MinimoFactoresVerificados: 2, MinimoGruposCriptograficosDistintos: 2, GarantiaMinima: core.AuthAssuranceHigh,
	}
	garantia := core.AuthAssuranceHigh
	if temporal {
		cfg.PoliticaInterna = httpseguridad.PoliticaInternaDesarrolloCertificadoPersonal
		cfg.RetiradaPoliticaInternaEn = instantePrueba.Add(time.Hour)
		cfg.MetodosAdmitidos = []httpseguridad.MetodoAutenticacion{httpseguridad.MetodoCertificado}
		cfg.FactoresRequeridos = []httpseguridad.MetodoAutenticacion{httpseguridad.MetodoCertificado}
		cfg.MinimoFactoresVerificados = 1
		cfg.MinimoGruposCriptograficosDistintos = 1
		cfg.GarantiaMinima = core.AuthAssuranceSubstantial
		garantia = core.AuthAssuranceSubstantial
	}
	// El canal procede de un handshake mTLS local; el servicio exige la misma instancia.
	estado := estadoTLSPrueba(t)
	var asercion httpseguridad.AsercionProxyIdentidad
	verificador := &verificadorPrueba{}
	servicio, err := httpseguridad.NuevoServicioIdentidad(cfg, verificador, evaluadorPrueba{garantia}, registro, r)
	if err != nil {
		t.Fatal(err)
	}
	canal, err := servicio.AutenticarCanalTLSMutuo(estado)
	if err != nil {
		t.Fatal(err)
	}
	verificada := instantePrueba.Add(-time.Minute)
	asercion = httpseguridad.AsercionProxyIdentidad{ID: "ase-id-1", Emisor: cfg.EmisorIdentidad, Audiencia: cfg.Audiencia,
		Superficie: cfg.Superficie, SujetoID: "sujeto-1", Cuenta: httpseguridad.CuentaAcceso{ID: "cuenta-1", SujetoVinculadoID: "sujeto-1"},
		SesionID: "ses-id-1", CanalVinculadoRef: canal.ReferenciaVinculacion(),
		AutenticacionVerificadaEn: verificada, EmitidaEn: verificada, NoAntesDe: verificada, ExpiraEn: instantePrueba.Add(time.Minute),
		MetodoPrimario: httpseguridad.MetodoKerberos, ACRVerificado: "urn:vec:acr:alto",
		Factores: []httpseguridad.FactorAutenticacion{
			{Metodo: httpseguridad.MetodoKerberos, SujetoVinculadoID: "sujeto-1", Principal: "cuenta-1@TEST", EvidenciaRef: "krb:1", GrupoCriptograficoRef: "grupo:clave", VerificadoEn: verificada},
			{Metodo: httpseguridad.MetodoCertificado, SujetoVinculadoID: "sujeto-1", CredencialRef: "cert:1", EvidenciaRef: "cert:1", GrupoCriptograficoRef: "grupo:tarjeta", VerificadoEn: verificada},
		},
	}
	if temporal {
		asercion.MetodoPrimario = httpseguridad.MetodoCertificado
		asercion.ACRVerificado = httpseguridad.ACRCertificadoPersonalDesarrolloProtegido
		asercion.Factores = asercion.Factores[1:]
	}
	verificador.asercion = asercion
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
	resultado, _, err := pruebas.NuevoContextoRegistradoYVinculoV2(instantePrueba.Add(time.Minute), personaPrueba, perfilPrueba, core.AuthMethodKerberos, core.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	revalidador := &revalidadorPrueba{alta: registro.alta}
	resolutor := &resolutorPrueba{resultado: resultado}
	autorizacion := &autorizacionPrueba{snapshot: snapshotPrueba()}
	config := ConfiguracionFuente{Identidad: servicio, Revalidador: revalidador, Resolutor: resolutor,
		Autorizacion: autorizacion, Reloj: r, PorCuenta: map[string]PerfilNominal{cuentaPrueba: {PerfilActivoRef: perfilPrueba, VersionRolRef: "rol:tecnico_rpt:v2"}}}
	return &entornoPrueba{servicio, ctx, r, registro, revalidador, resolutor, autorizacion, config}
}

func TestFuenteResuelveUnaVezYConservaResultado(t *testing.T) {
	e := nuevoEntornoPrueba(t)
	f, err := NuevaFuente(e.config)
	if err != nil {
		t.Fatal(err)
	}
	e.config.PorCuenta[cuentaPrueba] = PerfilNominal{PerfilActivoRef: "prf_ajeno0123456789abcdefghijkl", VersionRolRef: "rol:ajeno:v1"}
	v, r, s, err := f.Resolver(e.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if v.ValidarPara(r) != nil || r.Validar() != nil || s.Validar() != nil ||
		e.revalidador.llamadas != 1 || e.resolutor.llamadas != 1 || e.autorizacion.llamadas != 1 {
		t.Fatal("vínculo, contexto, instantánea o número de consultas incorrectos")
	}
}

func TestFuenteAceptaRelojEquivalenteConNanosegundos(t *testing.T) {
	for _, ahora := range []time.Time{
		instantePrueba.Add(123 * time.Nanosecond),
		instantePrueba.Add(123 * time.Nanosecond).In(time.FixedZone("zona_equivalente", 2*60*60)),
	} {
		e := nuevoEntornoPrueba(t)
		e.reloj.ahora = ahora
		f, err := NuevaFuente(e.config)
		if err != nil {
			t.Fatal(err)
		}
		v, resultado, snapshot, err := f.Resolver(e.ctx)
		if err != nil || v.ValidarPara(resultado) != nil || snapshot.Validar() != nil {
			t.Fatalf("reloj equivalente denegado: %v", err)
		}
	}
}

func TestFuenteDeniegaRevocacionPerfilYRolRetirado(t *testing.T) {
	casos := []struct {
		name string
		muta func(*entornoPrueba)
	}{
		{"sesion revocada", func(e *entornoPrueba) { e.registro.revocada = true }},
		{"cuenta ajena", func(e *entornoPrueba) {
			e.config.PorCuenta = map[string]PerfilNominal{"cta_ajena0123456789abcdefghijk": {PerfilActivoRef: perfilPrueba, VersionRolRef: "rol:tecnico_rpt:v2"}}
		}},
		{"perfil ajeno", func(e *entornoPrueba) {
			e.config.PorCuenta[cuentaPrueba] = PerfilNominal{PerfilActivoRef: "prf_ajeno0123456789abcdefghijkl", VersionRolRef: "rol:tecnico_rpt:v2"}
		}},
		{"rol retirado", func(e *entornoPrueba) {
			e.autorizacion.snapshot.ControlVigenciaVersionRol.Estado = core.EstadoControlVigenciaVersionRolRetirada
			e.autorizacion.snapshot.ControlVigenciaVersionRol.ActoRef = "acto:retirada"
			e.autorizacion.snapshot.ControlVigenciaVersionRol.MotivoCodigo = "baja"
		}},
		{"version de rol ajena", func(e *entornoPrueba) {
			e.config.PorCuenta[cuentaPrueba] = PerfilNominal{PerfilActivoRef: perfilPrueba, VersionRolRef: "rol:tecnico_rpt:v1"}
		}},
	}
	for _, c := range casos {
		t.Run(c.name, func(t *testing.T) {
			e := nuevoEntornoPrueba(t)
			c.muta(e)
			f, err := NuevaFuente(e.config)
			if err != nil {
				t.Fatal(err)
			}
			if _, _, _, err := f.Resolver(e.ctx); !errors.Is(err, ErrIdentidadOrdinariaNoDisponible) {
				t.Fatalf("acceso admitido: %v", err)
			}
		})
	}
}

func TestFuenteDeniegaSinCapsulaOGarantia(t *testing.T) {
	e := nuevoEntornoPrueba(t)
	f, err := NuevaFuente(e.config)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := f.Resolver(context.Background()); !errors.Is(err, ErrIdentidadOrdinariaNoDisponible) {
		t.Fatal("sin cápsula")
	}
	if factoresCorporativosValidos(httpseguridad.ContextoAuditoriaAutenticada{}) {
		t.Fatal("factores ausentes")
	}
	temporal := nuevoEntornoConPoliticaPrueba(t, true)
	fTemporal, err := NuevaFuente(temporal.config)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := fTemporal.Resolver(temporal.ctx); !errors.Is(err, ErrIdentidadOrdinariaNoDisponible) {
		t.Fatalf("el certificado temporal abrió la fuente: %v", err)
	}
}

type causaPrivadaPrueba struct{ marcador string }

func (e *causaPrivadaPrueba) Error() string { return "dsn_privada=" + e.marcador }

func TestFuentePropagaCausaSinExponerMensajeNiContinuar(t *testing.T) {
	for _, caso := range []struct {
		nombre                                  string
		fallar                                  func(*entornoPrueba, error)
		esperadoResolutor, esperadoAutorizacion int
	}{
		{"revalidacion", func(e *entornoPrueba, err error) { e.revalidador.err = err }, 0, 0},
		{"autorizacion", func(e *entornoPrueba, err error) { e.autorizacion.err = err }, 1, 1},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			e := nuevoEntornoPrueba(t)
			causa := &causaPrivadaPrueba{marcador: "NO_MOSTRAR_SECRETO"}
			caso.fallar(e, causa)
			f, err := NuevaFuente(e.config)
			if err != nil {
				t.Fatal(err)
			}
			v, r, s, err := f.Resolver(e.ctx)
			var extraida *causaPrivadaPrueba
			if !errors.Is(err, ErrIdentidadOrdinariaNoDisponible) || !errors.As(err, &extraida) || extraida != causa ||
				err.Error() != ErrIdentidadOrdinariaNoDisponible.Error() || strings.Contains(err.Error(), causa.marcador) {
				t.Fatalf("causa o mensaje opaco incorrectos: %v", err)
			}
			if v.Validar() == nil || r.Validar() == nil || s.Validar() == nil ||
				e.revalidador.llamadas != 1 || e.resolutor.llamadas != caso.esperadoResolutor ||
				e.autorizacion.llamadas != caso.esperadoAutorizacion {
				t.Fatal("una autoridad posterior fue llamada o salió material parcial")
			}
		})
	}
}

func estadoTLSPrueba(t *testing.T) tls.ConnectionState {
	t.Helper()
	ahora := time.Now()
	_, claveCA, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ca := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "CA test"},
		NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour), IsCA: true, BasicConstraintsValid: true,
		KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature}
	derCA, err := x509.CreateCertificate(rand.Reader, ca, ca, claveCA.Public(), claveCA)
	if err != nil {
		t.Fatal(err)
	}
	certCA, err := x509.ParseCertificate(derCA)
	if err != nil {
		t.Fatal(err)
	}
	crear := func(serial int64, nombre string, uso x509.ExtKeyUsage) tls.Certificate {
		pub, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		plantilla := &x509.Certificate{SerialNumber: big.NewInt(serial), Subject: pkix.Name{CommonName: nombre}, DNSNames: []string{nombre},
			NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{uso}}
		der, err := x509.CreateCertificate(rand.Reader, plantilla, certCA, pub, claveCA)
		if err != nil {
			t.Fatal(err)
		}
		return tls.Certificate{Certificate: [][]byte{der, derCA}, PrivateKey: priv}
	}
	raices := x509.NewCertPool()
	raices.AddCert(certCA)
	servidor, cliente := net.Pipe()
	ss := tls.Server(servidor, &tls.Config{Certificates: []tls.Certificate{crear(2, "server.test", x509.ExtKeyUsageServerAuth)}, ClientAuth: tls.RequireAndVerifyClientCert, ClientCAs: raices, MinVersion: tls.VersionTLS13})
	cc := tls.Client(cliente, &tls.Config{Certificates: []tls.Certificate{crear(3, "proxy.test", x509.ExtKeyUsageClientAuth)}, RootCAs: raices, ServerName: "server.test", MinVersion: tls.VersionTLS13})
	errores := make(chan error, 2)
	go func() { errores <- ss.Handshake() }()
	go func() { errores <- cc.Handshake() }()
	for i := 0; i < 2; i++ {
		if err := <-errores; err != nil {
			t.Fatal(err)
		}
	}
	estado := ss.ConnectionState()
	_ = ss.Close()
	_ = cc.Close()
	return estado
}
