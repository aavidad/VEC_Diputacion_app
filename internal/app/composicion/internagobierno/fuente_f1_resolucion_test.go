package internagobierno

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
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

const (
	personaF1Prueba = "per_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	perfilF1Prueba  = "prf_0123456789abcdefghijkl"
	cuentaF1Prueba  = "cta_0123456789abcdefghijkl"
)

type evaluadorGarantiaF1Prueba struct{ llamadas int }

func (e *evaluadorGarantiaF1Prueba) Evaluar(context.Context, httpseguridad.EntradaEvaluacionGarantia) (httpseguridad.ResultadoEvaluacionGarantia, error) {
	e.llamadas++
	return httpseguridad.ResultadoEvaluacionGarantia{
		Garantia:    core.AuthAssuranceSubstantial,
		PoliticaRef: "pga_0123456789abcdefghijkl", HuellaPolitica: "sha256:" + strings.Repeat("a", 64),
	}, nil
}

type registroSesionF1Prueba struct {
	cuentaRef    string
	confirmacion httpseguridad.ConfirmacionAltaSesion
	consultas    int
	activo       bool
}

func (r *registroSesionF1Prueba) ConsumirAsercionYRegistrar(_ context.Context, alta httpseguridad.AltaSesionAtomica) (httpseguridad.ConfirmacionAltaSesion, error) {
	if alta.Validar() != nil {
		return httpseguridad.ConfirmacionAltaSesion{}, errors.New("alta invalida")
	}
	c := httpseguridad.ConfirmacionAltaSesion{
		AutenticacionRef: "aut_0123456789abcdefghijkl", AsercionRef: "ase_0123456789abcdefghijkl",
		SesionRef: "ses_0123456789abcdefghijkl", ControlSesionRef: "cse_0123456789abcdefghijkl",
		ControlSesionRevision: 1, ControlSesionEstado: httpseguridad.EstadoControlSesionActiva,
		ControlSesionHuellaSHA256: strings.Repeat("b", 64),
		CuentaRef:                 r.cuentaRef, CuentaOrdinariaRef: r.cuentaRef,
		SesionRevalidadaEn: alta.SesionEmitidaEn, SesionValidaHasta: alta.AsercionExpiraEn,
		AltaConfirmada: alta,
	}
	if c.ValidarPara(alta) != nil {
		return httpseguridad.ConfirmacionAltaSesion{}, errors.New("recibo invalido")
	}
	r.confirmacion = c
	return c, nil
}
func (r *registroSesionF1Prueba) ComprobarSesionYCuentaActivas(_ context.Context, consulta httpseguridad.ConsultaSesionActiva) error {
	r.consultas++
	if !r.activo || consulta.Validar() != nil || consulta.AutenticacionRef != r.confirmacion.AutenticacionRef ||
		consulta.SesionRef != r.confirmacion.SesionRef || consulta.CuentaRef != r.cuentaRef {
		return errors.New("sesion retirada")
	}
	return nil
}

type revalidadorF1Prueba struct {
	registro *registroSesionF1Prueba
	llamadas int
}

func (r *revalidadorF1Prueba) RevalidarAutenticacionActorV1(_ context.Context, s core.SolicitudRevalidacionAutenticacionActorV1) (core.AutenticacionRevalidadaV1, error) {
	r.llamadas++
	c := r.registro.confirmacion
	if !r.registro.activo || s.AutenticacionRef != c.AutenticacionRef || s.SesionRef != c.SesionRef {
		return core.AutenticacionRevalidadaV1{}, errors.New("autenticacion retirada")
	}
	a := c.AltaConfirmada
	return core.AutenticacionRevalidadaV1{
		AutenticacionRef: c.AutenticacionRef, AutenticacionHuellaSHA256: a.AutenticacionHuellaSHA256,
		AsercionRef: c.AsercionRef, SesionRef: c.SesionRef, ControlSesionRef: c.ControlSesionRef,
		ControlSesionRevision: c.ControlSesionRevision, ControlSesionHuellaSHA256: c.ControlSesionHuellaSHA256,
		CuentaRef: c.CuentaRef, CuentaOrdinariaRef: c.CuentaOrdinariaRef,
		Superficie:      core.SuperficieAutenticacionInternaCorporativaV1,
		MetodoObservado: core.AuthMethodCertificate, GarantiaObservada: core.AuthAssuranceSubstantial,
		PoliticaGarantiaRef: a.PoliticaGarantiaRef, PoliticaGarantiaHuellaSHA256: a.PoliticaGarantiaHuellaSHA256,
		AutenticacionVerificadaEn: a.AutenticacionVerificadaEn, SesionEmitidaEn: a.SesionEmitidaEn,
		SesionRevalidadaEn: c.SesionRevalidadaEn, SesionValidaHasta: c.SesionValidaHasta,
	}, nil
}

type resolutorF1Prueba struct {
	resultado core.ResultadoContextoActorRegistradoV2
	llamadas  int
}

func (r *resolutorF1Prueba) ResolverContextoActorRegistradoV2(_ context.Context, s core.SolicitudContextoActor) (core.ResultadoContextoActorRegistradoV2, error) {
	r.llamadas++
	if s.Cuenta.CuentaRef != r.resultado.Contexto.Instantanea.CuentaRef || s.PerfilActivoRef != r.resultado.Contexto.PerfilActivoRef {
		return core.ResultadoContextoActorRegistradoV2{}, errors.New("cuenta o perfil ajeno")
	}
	return r.resultado.Clonar()
}

type entornoResolucionF1Prueba struct {
	fuente      *FuenteF1
	ctx         context.Context
	registro    *registroSesionF1Prueba
	evaluador   *evaluadorGarantiaF1Prueba
	revalidador *revalidadorF1Prueba
	resolutor   *resolutorF1Prueba
	corporativo *autoridadCorporativaPrueba
}

func nuevoEntornoResolucionF1Prueba(t *testing.T, personaF1, cuentaSesion string) entornoResolucionF1Prueba {
	t.Helper()
	ahora := time.Date(2026, 10, 8, 9, 0, 0, 0, time.UTC)
	config := httpseguridad.ConfiguracionSuperficie{
		Superficie: httpseguridad.SuperficieInternaCorporativa, ZonaRed: httpseguridad.ZonaRedInterna,
		DireccionEscucha: "127.0.0.1:8443", Audiencia: "vec-interna", EmisorIdentidad: "https://idp.identidad.test",
		RedesPermitidas: []string{"127.0.0.0/8"}, IdentidadesSANProxyPermitidas: []string{"dns:proxy.test"},
		DuracionMaximaAsercion: 3 * time.Minute, EdadMaximaAutenticacion: 15 * time.Minute, ToleranciaReloj: 20 * time.Second,
		MetodosAdmitidos:          []httpseguridad.MetodoAutenticacion{httpseguridad.MetodoCertificado},
		FactoresRequeridos:        []httpseguridad.MetodoAutenticacion{httpseguridad.MetodoCertificado},
		MinimoFactoresVerificados: 1, MinimoGruposCriptograficosDistintos: 1, GarantiaMinima: core.AuthAssuranceSubstantial,
		PoliticaInterna:           httpseguridad.PoliticaInternaDesarrolloCertificadoPersonal,
		RetiradaPoliticaInternaEn: ahora.Add(time.Hour),
	}
	registro := &registroSesionF1Prueba{cuentaRef: cuentaSesion, activo: true}
	evaluador := &evaluadorGarantiaF1Prueba{}
	var asercion httpseguridad.AsercionProxyIdentidad
	servicio, err := httpseguridad.NuevoServicioIdentidad(config, verificadorAsercionF1Mutable{asercion: &asercion}, evaluador, registro, relojPrueba{ahora})
	if err != nil {
		t.Fatalf("servicio real de identidad: %v", err)
	}
	estado := estadoTLSF1Prueba(t)
	canal, err := servicio.AutenticarCanalTLSMutuo(estado)
	if err != nil {
		t.Fatalf("canal TLS real: %v", err)
	}
	certSHA := sha256.Sum256(estado.PeerCertificates[0].Raw)
	certRef := "cert:sha256:" + hex.EncodeToString(certSHA[:])
	emitida := ahora.Add(-2 * time.Minute)
	asercion = httpseguridad.AsercionProxyIdentidad{
		ID: "asercion-f1-prueba", Emisor: config.EmisorIdentidad, Audiencia: config.Audiencia,
		Superficie: config.Superficie, SujetoID: personaF1,
		Cuenta:   httpseguridad.CuentaAcceso{ID: "cuenta-f1-prueba", SujetoVinculadoID: personaF1},
		SesionID: "sesion-f1-prueba", CanalVinculadoRef: canal.ReferenciaVinculacion(),
		AutenticacionVerificadaEn: emitida, EmitidaEn: emitida, NoAntesDe: emitida,
		ExpiraEn: ahora.Add(time.Minute), MetodoPrimario: httpseguridad.MetodoCertificado,
		ACRVerificado: httpseguridad.ACRCertificadoPersonalDesarrolloProtegido,
		Factores: []httpseguridad.FactorAutenticacion{{Metodo: httpseguridad.MetodoCertificado,
			SujetoVinculadoID: personaF1, CredencialRef: certRef, EvidenciaRef: "cert:validacion:001",
			GrupoCriptograficoRef: "grupo:certificado:001", VerificadoEn: emitida}},
	}
	// El verificador es el puerto de aserción protegida; su afirmación sintética
	// permanece ligada al canal de un handshake mTLS realmente ejecutado.
	credencial, err := httpseguridad.NuevaCredencialProxy([]byte("asercion-protegida-sintetica"), canal)
	if err != nil {
		t.Fatal(err)
	}
	identidad, err := servicio.Resolver(context.Background(), credencial)
	if err != nil {
		t.Fatalf("alta real via puertos: %v", err)
	}
	capsula, err := servicio.ProyectarCapsulaIdentidadPeticion(context.Background(), identidad, canal)
	if err != nil {
		t.Fatalf("capsula real: %v", err)
	}
	ctx, err := servicio.VincularCapsulaIdentidadPeticion(context.Background(), capsula, canal)
	if err != nil {
		t.Fatalf("capsula vinculada: %v", err)
	}
	resultado, _, err := pruebas.NuevoContextoRegistradoYVinculoV2(ahora, personaF1, perfilF1Prueba, core.AuthMethodCertificate, core.AuthAssuranceSubstantial)
	if err != nil {
		t.Fatalf("resultado F1 registrado: %v", err)
	}
	revalidador := &revalidadorF1Prueba{registro: registro}
	resolutor := &resolutorF1Prueba{resultado: resultado}
	corporativo := &autoridadCorporativaPrueba{vigente: true}
	c := configuracionFuentePrueba()
	c.Identidad = servicio
	c.Revalidador = revalidador
	c.Resolutor = resolutor
	c.VinculoCorporativo = corporativo
	c.Reloj = relojPrueba{ahora}
	c.Politica.RetiradaEn = config.RetiradaPoliticaInternaEn
	c.Politica.Referencia = "pga_0123456789abcdefghijkl"
	c.PorCuenta = map[string]ContextoNominal{cuentaSesion: {PerfilActivoRef: perfilF1Prueba,
		OrganizacionRef: "ref:" + strings.Repeat("a", 64), UnidadRef: "ref:" + strings.Repeat("b", 64)}}
	fuente, err := NuevaFuenteF1(c)
	if err != nil {
		t.Fatalf("fuente F1: %v", err)
	}
	return entornoResolucionF1Prueba{fuente, ctx, registro, evaluador, revalidador, resolutor, corporativo}
}

type verificadorAsercionF1Mutable struct {
	asercion *httpseguridad.AsercionProxyIdentidad
}

func (v verificadorAsercionF1Mutable) Verificar(context.Context, []byte) (httpseguridad.AsercionProxyIdentidad, error) {
	return *v.asercion, nil
}

func estadoTLSF1Prueba(t *testing.T) tls.ConnectionState {
	t.Helper()
	_, claveCA, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ahora := time.Now()
	ca := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "CA prueba F1"},
		NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour), IsCA: true, BasicConstraintsValid: true,
		KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature}
	derCA, err := x509.CreateCertificate(rand.Reader, ca, ca, claveCA.Public(), claveCA)
	if err != nil {
		t.Fatal(err)
	}
	ca, err = x509.ParseCertificate(derCA)
	if err != nil {
		t.Fatal(err)
	}
	crear := func(serial int64, nombre string, uso x509.ExtKeyUsage) tls.Certificate {
		_, clave, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		c := &x509.Certificate{SerialNumber: big.NewInt(serial), Subject: pkix.Name{CommonName: nombre},
			DNSNames: []string{nombre}, NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour),
			KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{uso}}
		der, err := x509.CreateCertificate(rand.Reader, c, ca, clave.Public(), claveCA)
		if err != nil {
			t.Fatal(err)
		}
		return tls.Certificate{Certificate: [][]byte{der, derCA}, PrivateKey: clave}
	}
	raices := x509.NewCertPool()
	raices.AddCert(ca)
	servidor := tls.Config{Certificates: []tls.Certificate{crear(2, "server.test", x509.ExtKeyUsageServerAuth)},
		ClientAuth: tls.RequireAndVerifyClientCert, ClientCAs: raices, MinVersion: tls.VersionTLS13}
	cliente := tls.Config{Certificates: []tls.Certificate{crear(3, "proxy.test", x509.ExtKeyUsageClientAuth)},
		RootCAs: raices, ServerName: "server.test", MinVersion: tls.VersionTLS13}
	uno, dos := net.Pipe()
	_ = uno.SetDeadline(time.Now().Add(5 * time.Second))
	_ = dos.SetDeadline(time.Now().Add(5 * time.Second))
	s := tls.Server(uno, &servidor)
	c := tls.Client(dos, &cliente)
	terminados := make(chan error, 2)
	go func() { terminados <- s.Handshake() }()
	go func() { terminados <- c.Handshake() }()
	for i := 0; i < 2; i++ {
		if err := <-terminados; err != nil {
			_ = uno.Close()
			_ = dos.Close()
			t.Fatalf("mTLS de prueba: %v", err)
		}
	}
	estado := s.ConnectionState()
	_ = uno.Close()
	_ = dos.Close()
	return estado
}

func TestFuenteF1ResolverContextoUsaUnaResolucionViva(t *testing.T) {
	e := nuevoEntornoResolucionF1Prueba(t, personaF1Prueba, cuentaF1Prueba)
	consultasAntes, evaluacionesAntes := e.registro.consultas, e.evaluador.llamadas
	resuelto, err := e.fuente.ResolverContexto(e.ctx)
	if err != nil || resuelto.Resultado.Validar() != nil || resuelto.Vinculo.ValidarPara(resuelto.Resultado) != nil ||
		e.revalidador.llamadas != 1 || e.resolutor.llamadas != 1 || len(e.corporativo.solicitudes) != 1 ||
		e.registro.consultas-consultasAntes != 2 || e.evaluador.llamadas-evaluacionesAntes != 2 {
		t.Fatalf("F1 no uso una resolucion coherente: err=%v auth=%d F1=%d corp=%d sesion=%d garantia=%d", err,
			e.revalidador.llamadas, e.resolutor.llamadas, len(e.corporativo.solicitudes),
			e.registro.consultas-consultasAntes, e.evaluador.llamadas-evaluacionesAntes)
	}
	if resuelto.Resultado.Contexto.PersonaRef != personaF1Prueba || resuelto.Resultado.Contexto.Instantanea.CuentaRef != cuentaF1Prueba {
		t.Fatal("resolucion no conserva persona y cuenta registradas")
	}
}

func TestFuenteF1PeticionVerificadaConservaInterfazYUnaResolucion(t *testing.T) {
	e := nuevoEntornoResolucionF1Prueba(t, personaF1Prueba, cuentaF1Prueba)
	peticion, err := e.fuente.PeticionVerificada(e.ctx)
	if err != nil || peticion.PreparacionCT.ActorRef != personaF1Prueba ||
		peticion.Contexto.Cuenta.CuentaRef != cuentaF1Prueba ||
		e.revalidador.llamadas != 1 || e.resolutor.llamadas != 1 ||
		len(e.corporativo.solicitudes) != 1 {
		t.Fatalf("peticion nominal no conserva contrato: %v", err)
	}
}

func TestFuenteF1DeniegaPersonaCuentaOCorporativoDivergentes(t *testing.T) {
	for _, caso := range []struct {
		nombre, persona, cuenta string
		retirarCorporativo      bool
	}{
		{"persona_ajena", "per_" + strings.Repeat("b", 32), cuentaF1Prueba, false},
		{"cuenta_ajena", personaF1Prueba, "cta_" + strings.Repeat("z", 22), false},
		{"corporativo_retirado", personaF1Prueba, cuentaF1Prueba, true},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			e := nuevoEntornoResolucionF1Prueba(t, caso.persona, caso.cuenta)
			consultasAntes := e.registro.consultas
			if caso.retirarCorporativo {
				e.corporativo.vigente = false
			}
			// El certificado queda ligado a B y F1 resuelve A: ambos son
			// individualmente válidos, pero el cotejo de sujeto debe denegar.
			if caso.nombre == "persona_ajena" {
				resultado, _, err := pruebas.NuevoContextoRegistradoYVinculoV2(
					time.Date(2026, 10, 8, 9, 0, 0, 0, time.UTC), personaF1Prueba, perfilF1Prueba,
					core.AuthMethodCertificate, core.AuthAssuranceSubstantial)
				if err != nil {
					t.Fatal(err)
				}
				e.resolutor.resultado = resultado
			}
			if _, err := e.fuente.ResolverContexto(e.ctx); !errors.Is(err, ErrGobiernoInternoNoDisponible) ||
				e.revalidador.llamadas != 1 || e.resolutor.llamadas != 1 {
				t.Fatalf("identidad/contexto adverso admitido o duplico consultas: %v", err)
			}
			esperadasSesion, esperadasCorporativo := 2, 0
			if caso.nombre == "cuenta_ajena" {
				esperadasSesion = 1
			}
			if caso.retirarCorporativo {
				esperadasCorporativo = 1
			}
			if e.registro.consultas-consultasAntes != esperadasSesion ||
				len(e.corporativo.solicitudes) != esperadasCorporativo {
				t.Fatalf("la guarda adversa no se alcanzo en su fase: sesion=%d corporativo=%d",
					e.registro.consultas-consultasAntes, len(e.corporativo.solicitudes))
			}
		})
	}
}
