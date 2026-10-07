package httpseguridad

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

type certificadorPresentacionPrueba struct {
	err                 error
	consultas           int
	crlCero             bool
	verificacionAntigua bool
}

func (c *certificadorPresentacionPrueba) AcreditarCertificadoActual(
	_ context.Context, cadena []*x509.Certificate, ahora time.Time,
) (AcreditacionCertificadoActual, error) {
	c.consultas++
	if c.err != nil {
		return AcreditacionCertificadoActual{}, c.err
	}
	if len(cadena) < 2 || cadena[0] == nil || cadena[1] == nil {
		return AcreditacionCertificadoActual{}, ErrPresentacionCertificadoNoValida
	}
	certificado, autoridad := cadena[0], cadena[1]
	caHasta := autoridad.NotAfter.UTC().Truncate(time.Microsecond)
	for _, eslabon := range cadena[2:] {
		if eslabon == nil {
			return AcreditacionCertificadoActual{}, ErrPresentacionCertificadoNoValida
		}
		if eslabon.NotAfter.Before(caHasta) {
			caHasta = eslabon.NotAfter.UTC().Truncate(time.Microsecond)
		}
	}
	crlSiguiente := ahora.Add(30 * time.Minute)
	if c.crlCero {
		crlSiguiente = time.Time{}
	}
	verificada := ahora
	if c.verificacionAntigua {
		verificada = ahora.Add(-time.Minute)
	}
	huellaCertificado := sha256.Sum256(certificado.Raw)
	huellaCA := sha256.Sum256(autoridad.Raw)
	return AcreditacionCertificadoActual{
		SujetoID: "per_" + strings.Repeat("a", 22), CuentaID: "Cuenta-Tecnica",
		CertificadoSHA256:      "sha256:" + hex.EncodeToString(huellaCertificado[:]),
		CASHA256:               "sha256:" + hex.EncodeToString(huellaCA[:]),
		CertificadoValidoHasta: certificado.NotAfter.UTC().Truncate(time.Microsecond),
		CAValidaHasta:          caHasta,
		CRLSiguienteEn:         crlSiguiente, RevocacionVerificadaEn: verificada,
	}, nil
}

func TestPresentacionCertificadoCRLOFechaDeComprobacionAusenteCierra(t *testing.T) {
	for nombre, alterar := range map[string]func(*certificadorPresentacionPrueba){
		"crl_sin_proxima":      func(c *certificadorPresentacionPrueba) { c.crlCero = true },
		"comprobacion_antigua": func(c *certificadorPresentacionPrueba) { c.verificacionAntigua = true },
	} {
		t.Run(nombre, func(t *testing.T) {
			s, _, certificador, registro, _, estado := entornoPresentacionCertificadoPrueba(t)
			alterar(certificador)
			_, _, _, err := s.AcreditarCertificadoActual(context.Background(), estado)
			if err == nil || certificador.consultas != 1 || registro.inicios != 0 || registro.reanudaciones != 0 {
				t.Fatalf("certificado sin CRL actual avanzo a registro: %v", err)
			}
		})
	}
}

func TestPresentacionCertificadoNotAfterVencidoCierraAntesDeCRL(t *testing.T) {
	s, _, certificador, registro, reloj, estado := entornoPresentacionCertificadoPrueba(t)
	reloj.fijar(estado.VerifiedChains[0][0].NotAfter.Add(time.Second).UTC().Truncate(time.Microsecond))
	if _, _, _, err := s.AcreditarCertificadoActual(context.Background(), estado); err == nil ||
		certificador.consultas != 0 || registro.inicios != 0 || registro.reanudaciones != 0 {
		t.Fatalf("NotAfter vencido alcanzo autoridad: %v", err)
	}
}

func TestPresentacionCertificadoIgnoraFechaMutadaFueraDelDER(t *testing.T) {
	s, _, _, _, _, estado := entornoPresentacionCertificadoPrueba(t)
	original := estado.VerifiedChains[0][0].NotAfter.UTC().Truncate(time.Microsecond)
	estado.VerifiedChains[0][0].NotAfter = original.Add(24 * time.Hour)
	_, _, prueba, err := s.AcreditarCertificadoActual(context.Background(), estado)
	if err != nil || prueba.datos == nil || !prueba.datos.certificadoHasta.Equal(original) {
		t.Fatalf("fecha no firmada sustituyo vencimiento DER: %v", err)
	}
}

func TestPresentacionCertificadoRechazaCruceYElevacionAntesDeSQL(t *testing.T) {
	casos := []struct {
		nombre  string
		alterar func(*ServicioPresentacionCertificado, *verificadorFalso)
	}{
		{"persona_ajena", func(_ *ServicioPresentacionCertificado, v *verificadorFalso) {
			a := v.asercion
			ajena := "per_" + strings.Repeat("b", 22)
			a.SujetoID = ajena
			a.Cuenta.SujetoVinculadoID = ajena
			a.Factores[0].SujetoVinculadoID = ajena
			v.fijarAsercion(a)
		}},
		{"cuenta_ajena", func(_ *ServicioPresentacionCertificado, v *verificadorFalso) {
			a := v.asercion
			a.Cuenta.ID = "Otra-Cuenta"
			v.fijarAsercion(a)
		}},
		{"acr_ajeno", func(_ *ServicioPresentacionCertificado, v *verificadorFalso) {
			a := v.asercion
			a.ACRVerificado = "urn:vec:acr:otro"
			v.fijarAsercion(a)
		}},
		{"high_por_evaluador", func(s *ServicioPresentacionCertificado, _ *verificadorFalso) {
			s.identidad.evaluadorGarantia = evaluadorValido(domain.AuthAssuranceHigh)
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			s, v, _, registro, _, estado := entornoPresentacionCertificadoPrueba(t)
			ctx, credencial, prueba := credencialPresentacionCertificadoPrueba(t, s, v, estado)
			caso.alterar(s, v)
			if _, err := s.IniciarYConsumirPresentacion(ctx, credencial, prueba); err == nil ||
				registro.inicios != 0 || registro.reanudaciones != 0 {
				t.Fatalf("identidad/garantia ajena alcanzo SQL: %v", err)
			}
		})
	}
}

type registroPresentacionPrueba struct {
	modo                   string
	inicios, reanudaciones int
	err                    error
	original               domain.AutenticacionRevalidadaV1
}

func (r *registroPresentacionPrueba) IniciarYConsumirPresentacion(
	_ context.Context, orden OrdenInicioCertificado,
) (ResultadoRegistroPresentacionCertificado, error) {
	r.inicios++
	alta, datos, err := orden.Datos()
	if err != nil || r.err != nil {
		return ResultadoRegistroPresentacionCertificado{}, errors.Join(err, r.err)
	}
	return r.responder(datos, alta), nil
}

func (r *registroPresentacionPrueba) ReanudarYConsumirPresentacion(
	_ context.Context, orden OrdenReanudacionCertificado,
) (ResultadoRegistroPresentacionCertificado, error) {
	r.reanudaciones++
	datos, err := orden.Datos()
	if err != nil || r.err != nil {
		return ResultadoRegistroPresentacionCertificado{}, errors.Join(err, r.err)
	}
	return r.responder(datos, AltaSesionAtomica{}), nil
}

func (r *registroPresentacionPrueba) responder(d DatosPresentacionCertificado, alta AltaSesionAtomica) ResultadoRegistroPresentacionCertificado {
	modo := r.modo
	if modo == "" {
		modo = "abierta"
	}
	original := r.original
	if modo == "abierta" || modo == "renovada" {
		original = domain.AutenticacionRevalidadaV1{
			AutenticacionRef: "aut_0123456789abcdefghijkl", AsercionRef: "ase_0123456789abcdefghijkl",
			SesionRef: "ses_0123456789abcdefghijkl", ControlSesionRef: "cse_0123456789abcdefghijkl",
			AutenticacionHuellaSHA256: alta.AutenticacionHuellaSHA256,
			ControlSesionRevision:     1, ControlSesionHuellaSHA256: strings.Repeat("b", 64),
			CuentaRef: "cta_0123456789abcdefghijkl", CuentaOrdinariaRef: "cta_0123456789abcdefghijkl",
			Superficie:      domain.SuperficieAutenticacionInternaCorporativaV1,
			MetodoObservado: domain.AuthMethodCertificate, GarantiaObservada: domain.AuthAssuranceSubstantial,
			PoliticaGarantiaRef: d.PoliticaGarantiaRef, PoliticaGarantiaHuellaSHA256: d.PoliticaGarantiaHuellaSHA256,
			AutenticacionVerificadaEn: alta.AutenticacionVerificadaEn, SesionEmitidaEn: alta.SesionEmitidaEn,
			SesionRevalidadaEn: d.CertificadoVerificadoEn, SesionValidaHasta: d.AsercionActualExpiraEn,
		}
	}
	return ResultadoRegistroPresentacionCertificado{
		SesionOriginal: original,
		Recibo: ReciboPresentacionCertificado{
			ModoInicio: modo, OperacionRef: d.OperacionRef, PresentacionRef: "prs_0123456789abcdefghijkl",
			Generacion: 1, SesionGeneracion: 1, HuellaSHA256: strings.Repeat("c", 64),
			RegistradaEn: d.CertificadoVerificadoEn, ValidaHasta: d.CertificadoVerificadoEn.Add(30 * time.Second),
			CanalSHA256: d.CanalSHA256, AsercionActualHuellaSHA256: d.AsercionActualHuellaSHA256,
		},
	}
}

func entornoPresentacionCertificadoPrueba(t *testing.T) (*ServicioPresentacionCertificado, *verificadorFalso, *certificadorPresentacionPrueba, *registroPresentacionPrueba, *relojFijo, tls.ConnectionState) {
	t.Helper()
	ahora := time.Now().UTC().Truncate(time.Second)
	cfg := configuracionCertificadoDesarrollo()
	cfg.RetiradaPoliticaInternaEn = ahora.Add(time.Hour)
	verificador := &verificadorFalso{}
	reloj := &relojFijo{ahora: ahora}
	identidad := debeServicio(t, cfg, verificador, evaluadorValido(domain.AuthAssuranceSubstantial), nuevoRegistroMemoria(), reloj)
	certificador := &certificadorPresentacionPrueba{}
	registro := &registroPresentacionPrueba{}
	servicio, err := NuevoServicioPresentacionCertificado(identidad, certificador, registro)
	if err != nil {
		t.Fatal(err)
	}
	estado := estadoTLSMutuoReal(t, sanDNSConfigurado(cfg))
	return servicio, verificador, certificador, registro, reloj, estado
}

func credencialPresentacionCertificadoPrueba(t *testing.T, s *ServicioPresentacionCertificado, v *verificadorFalso, e tls.ConnectionState) (context.Context, CredencialProxy, PruebaCertificadoActual) {
	t.Helper()
	ctx, canal, prueba, err := s.AcreditarCertificadoActual(context.Background(), e)
	if err != nil {
		t.Fatal(err)
	}
	ahora := s.identidad.reloj.Ahora().UTC().Truncate(time.Microsecond)
	asercion := asercionInternaValida(ahora, s.identidad.configuracion, canal)
	asercion.SujetoID = prueba.datos.sujetoID
	asercion.Cuenta.SujetoVinculadoID = prueba.datos.sujetoID
	asercion.ACRVerificado = ACRCertificadoPersonalDesarrolloProtegido
	asercion.Factores = asercion.Factores[1:]
	asercion.Factores[0].SujetoVinculadoID = prueba.datos.sujetoID
	asercion.Factores[0].CredencialRef = "cert:" + prueba.datos.certificadoSHA256
	asercion.Factores[0].EvidenciaRef = "tls:verified:" + prueba.datos.certificadoSHA256
	v.fijarAsercion(asercion)
	return ctx, debeCredencial(t, []byte("asercion-protegida-sintetica"), canal), prueba
}

func TestPresentacionCertificadoInicioUnaOperacionYCapsulaActual(t *testing.T) {
	s, v, certificador, registro, _, estado := entornoPresentacionCertificadoPrueba(t)
	ctx, credencial, prueba := credencialPresentacionCertificadoPrueba(t, s, v, estado)
	capsula, err := s.IniciarYConsumirPresentacion(ctx, credencial, prueba)
	if err != nil || capsula.datos == nil || registro.inicios != 1 || registro.reanudaciones != 0 || certificador.consultas != 1 {
		t.Fatalf("inicio: err=%v inicio=%d reanudar=%d cert=%d capsula=%t", err, registro.inicios, registro.reanudaciones, certificador.consultas, capsula.datos != nil)
	}
	if _, err := json.Marshal(capsula); err == nil {
		t.Fatal("capsula serializable")
	}
	vinculado, err := s.VincularCapsulaPresentacion(ctx, capsula, credencial.canal)
	if err != nil {
		t.Fatalf("vincular capsula: %v", err)
	}
	cuenta, auditoria, err := s.identidad.ExtraerCapsulaIdentidadPeticion(vinculado)
	if err != nil || cuenta.CuentaRef != "cta_0123456789abcdefghijkl" ||
		auditoria.SesionRef() != "ses_0123456789abcdefghijkl" ||
		auditoria.CanalVinculadoRef() != credencial.canal.ReferenciaVinculacion() {
		t.Fatalf("capsula no conservo identidad original y canal actual: %v", err)
	}
	if err := s.identidad.ExigirSujetoPersonaCertificadoTemporal(vinculado, prueba.datos.sujetoID); err != nil {
		t.Fatalf("F1 no reconocio persona del certificado actual: %v", err)
	}
	if err := s.identidad.ExigirSujetoPersonaCertificadoTemporal(vinculado, "per_"+strings.Repeat("b", 22)); err == nil {
		t.Fatal("F1 acepto persona ajena")
	}
	ambas := context.WithValue(vinculado, claveCapsulaIdentidad{}, capsulaIdentidadVinculada{})
	if _, _, err := s.identidad.ExtraerCapsulaIdentidadPeticion(ambas); err == nil {
		t.Fatal("dos variantes de capsula admitidas")
	}
	if _, err := s.VincularCapsulaPresentacion(ctx, capsula, credencial.canal); err == nil {
		t.Fatal("capsula vinculada dos veces")
	}
}

func TestPresentacionCertificadoReanudacionNoAbreTrasDenegacion(t *testing.T) {
	s, v, _, registro, _, estado := entornoPresentacionCertificadoPrueba(t)
	ctx, credencial, prueba := credencialPresentacionCertificadoPrueba(t, s, v, estado)
	registro.err = errors.New("control revocado sintético")
	if _, err := s.ReanudarYConsumirPresentacion(ctx, credencial, prueba); err == nil ||
		registro.reanudaciones != 1 || registro.inicios != 0 {
		t.Fatalf("reanudacion: err=%v inicio=%d reanudar=%d", err, registro.inicios, registro.reanudaciones)
	}
	if _, err := s.IniciarYConsumirPresentacion(ctx, credencial, prueba); err == nil || registro.inicios != 0 {
		t.Fatalf("prueba ya consumida permitió alta: %v", err)
	}
}

func TestPresentacionCertificadoReanudadaConservaHistoriaYSeparaVentanaActual(t *testing.T) {
	s, v, _, registro, reloj, estado := entornoPresentacionCertificadoPrueba(t)
	ctxInicial, credencialInicial, pruebaInicial := credencialPresentacionCertificadoPrueba(t, s, v, estado)
	primera, err := s.IniciarYConsumirPresentacion(ctxInicial, credencialInicial, pruebaInicial)
	if err != nil {
		t.Fatal(err)
	}
	registro.original = primera.datos.resultado.SesionOriginal
	registro.modo = "reanudada"
	ctx, credencial, prueba := credencialPresentacionCertificadoPrueba(t, s, v, estado)
	asercion := v.asercion
	asercion.ID = "asercion-002"
	asercion.AutenticacionVerificadaEn = reloj.Ahora()
	asercion.EmitidaEn = reloj.Ahora()
	asercion.NoAntesDe = reloj.Ahora()
	asercion.Factores[0].VerificadoEn = reloj.Ahora()
	v.fijarAsercion(asercion)
	credencial = debeCredencial(t, []byte("segunda-presentacion-protegida"), credencial.canal)
	segunda, err := s.ReanudarYConsumirPresentacion(ctx, credencial, prueba)
	if err != nil || registro.inicios != 1 || registro.reanudaciones != 1 {
		t.Fatalf("reanudacion no conservo una sesion original: %v", err)
	}
	vinculado, err := s.VincularCapsulaPresentacion(ctx, segunda, credencial.canal)
	if err != nil {
		t.Fatal(err)
	}
	_, auditoria, err := s.identidad.ExtraerCapsulaIdentidadPeticion(vinculado)
	if err != nil || auditoria.AutenticacionHuellaSHA256() != registro.original.AutenticacionHuellaSHA256 ||
		auditoria.PresentacionAsercionHuellaSHA256() == auditoria.AutenticacionHuellaSHA256() ||
		!auditoria.SesionEmitidaEn().Before(auditoria.PresentacionEmitidaEn()) ||
		auditoria.PresentacionRef() == "" || auditoria.PresentacionGeneracion() == 0 {
		t.Fatalf("auditoria mezclo sesion original y presentacion actual: %v", err)
	}
}

func estadosTLSCertificadoCompartidoPresentacion(t *testing.T, san string) (tls.ConnectionState, tls.ConnectionState) {
	t.Helper()
	ahora := time.Now()
	publicaCA, privadaCA, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	caPlantilla := &x509.Certificate{SerialNumber: big.NewInt(11), Subject: pkix.Name{CommonName: "CA sintética"},
		NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour), IsCA: true, BasicConstraintsValid: true,
		KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature}
	derCA, err := x509.CreateCertificate(rand.Reader, caPlantilla, caPlantilla, publicaCA, privadaCA)
	if err != nil {
		t.Fatal(err)
	}
	ca, err := x509.ParseCertificate(derCA)
	if err != nil {
		t.Fatal(err)
	}
	crear := func(serial int64, nombre string, uso x509.ExtKeyUsage) tls.Certificate {
		publica, privada, e := ed25519.GenerateKey(rand.Reader)
		if e != nil {
			t.Fatal(e)
		}
		plantilla := &x509.Certificate{SerialNumber: big.NewInt(serial), Subject: pkix.Name{CommonName: nombre},
			DNSNames: []string{nombre}, NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour),
			KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{uso}}
		der, e := x509.CreateCertificate(rand.Reader, plantilla, ca, publica, privadaCA)
		if e != nil {
			t.Fatal(e)
		}
		return tls.Certificate{Certificate: [][]byte{der, derCA}, PrivateKey: privada}
	}
	raices := x509.NewCertPool()
	raices.AddCert(ca)
	servidorCfg := &tls.Config{Certificates: []tls.Certificate{crear(12, "servidor.test", x509.ExtKeyUsageServerAuth)},
		ClientAuth: tls.RequireAndVerifyClientCert, ClientCAs: raices, MinVersion: tls.VersionTLS13, MaxVersion: tls.VersionTLS13}
	clienteCfg := &tls.Config{Certificates: []tls.Certificate{crear(13, san, x509.ExtKeyUsageClientAuth)},
		RootCAs: raices, ServerName: "servidor.test", MinVersion: tls.VersionTLS13, MaxVersion: tls.VersionTLS13}
	conectar := func() tls.ConnectionState {
		parServidor, parCliente := net.Pipe()
		servidor, cliente := tls.Server(parServidor, servidorCfg), tls.Client(parCliente, clienteCfg)
		fallos := make(chan error, 2)
		go func() { fallos <- servidor.Handshake() }()
		go func() { fallos <- cliente.Handshake() }()
		for i := 0; i < 2; i++ {
			if e := <-fallos; e != nil {
				t.Fatal(e)
			}
		}
		estado := servidor.ConnectionState()
		_ = parServidor.Close()
		_ = parCliente.Close()
		return estado
	}
	return conectar(), conectar()
}

func TestPresentacionCertificadoMismoDERDosCanalesSesionOriginal(t *testing.T) {
	s, v, _, registro, _, _ := entornoPresentacionCertificadoPrueba(t)
	primero, segundo := estadosTLSCertificadoCompartidoPresentacion(t, sanDNSConfigurado(s.identidad.configuracion))
	ctxUno, credencialUno, pruebaUno := credencialPresentacionCertificadoPrueba(t, s, v, primero)
	capsulaUno, err := s.IniciarYConsumirPresentacion(ctxUno, credencialUno, pruebaUno)
	if err != nil {
		t.Fatal(err)
	}
	registro.original = capsulaUno.datos.resultado.SesionOriginal
	registro.modo = "reanudada"
	ctxDos, credencialDos, pruebaDos := credencialPresentacionCertificadoPrueba(t, s, v, segundo)
	asercionDos := v.asercion
	asercionDos.ID = "asercion-segunda"
	v.fijarAsercion(asercionDos)
	if pruebaUno.datos.certificadoSHA256 != pruebaDos.datos.certificadoSHA256 ||
		credencialUno.canal.ReferenciaVinculacion() == credencialDos.canal.ReferenciaVinculacion() {
		t.Fatal("escenario no comparte DER o no separa exportador TLS")
	}
	credencialDos = debeCredencial(t, []byte("asercion-segunda-protegida"), credencialDos.canal)
	capsulaDos, err := s.ReanudarYConsumirPresentacion(ctxDos, credencialDos, pruebaDos)
	if err != nil || capsulaDos.datos == nil || registro.inicios != 1 || registro.reanudaciones != 1 ||
		capsulaDos.datos.resultado.SesionOriginal.SesionRef != capsulaUno.datos.resultado.SesionOriginal.SesionRef ||
		capsulaDos.datos.canalRef == capsulaUno.datos.canalRef {
		t.Fatalf("segunda conexion no reuso sesion original con capsula nueva: %v", err)
	}
	if _, err := s.ReanudarYConsumirPresentacion(ctxDos, credencialDos, pruebaUno); err == nil {
		t.Fatal("prueba de otra petición/otro TLS aceptada")
	}
}
