package httpapi

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type fuenteGobiernoRPTPrueba struct {
	cred       application.CredencialesGobiernoCategoriaRPT
	descriptor ports.DescriptorCatalogoRPT
	asignacion domain.InstantaneaAutorizacion
	err        error
	llamadas   int
	cancelar   context.CancelFunc
}

func (f *fuenteGobiernoRPTPrueba) ResolverGobiernoCategoriaRPT(_ context.Context, _ *x509.Certificate) (application.CredencialesGobiernoCategoriaRPT, ports.DescriptorCatalogoRPT, domain.InstantaneaAutorizacion, error) {
	f.llamadas++
	if f.cancelar != nil {
		f.cancelar()
	}
	return f.cred, f.descriptor, f.asignacion, f.err
}

type auditorGobiernoRPTPrueba struct {
	codigos []string
	err     error
}

func (a *auditorGobiernoRPTPrueba) RegistrarDenegacionGobiernoCategoriaRPT(_ context.Context, ruta, codigo, correlacion string) error {
	if !rutaGobiernoRPTValida(ruta) || correlacion == "" {
		return errors.New("auditoria invalida")
	}
	a.codigos = append(a.codigos, codigo)
	return a.err
}

type operadorGobiernoRPTPrueba struct {
	llamadas  int
	cred      application.CredencialesGobiernoCategoriaRPT
	respuesta ports.ResultadoGobiernoCategoriaRPT
	err       error
	ultimo    ports.BorradorPropuestaGobiernoCategoriaRPT
	cancelar  context.CancelFunc
}

type revalidadorGobiernoRPTPrueba struct {
	valor domain.AutenticacionRevalidadaV1
}

func (r revalidadorGobiernoRPTPrueba) RevalidarAutenticacionActorV1(context.Context, domain.SolicitudRevalidacionAutenticacionActorV1) (domain.AutenticacionRevalidadaV1, error) {
	return r.valor, nil
}

type resolutorGobiernoRPTPrueba struct {
	valor domain.ResultadoContextoActorRegistradoV2
}

func (r resolutorGobiernoRPTPrueba) ResolverContextoActorRegistradoV2(context.Context, domain.SolicitudContextoActor) (domain.ResultadoContextoActorRegistradoV2, error) {
	return r.valor, nil
}

type relojGobiernoRPTPrueba struct{ ahora time.Time }

func (r relojGobiernoRPTPrueba) Ahora() time.Time { return r.ahora }

type correlacionGobiernoRPTPrueba struct{}

type lectorGobiernoRPTCancela struct {
	io.Reader
	cancelar context.CancelFunc
}

func (l *lectorGobiernoRPTCancela) Read(p []byte) (int, error) {
	n, err := l.Reader.Read(p)
	if n > 0 {
		l.cancelar()
	}
	return n, err
}

func (*lectorGobiernoRPTCancela) Close() error { return nil }

func (correlacionGobiernoRPTPrueba) NuevaReferenciaCorrelacionAutorizacionV2(context.Context) (string, error) {
	return "correlacion_0123456789abcdef0123456789abcdef", nil
}

const versionRolGobiernoRPTPrueba = "rol:configuracion_rpt:v1"

func credencialesGobiernoRPTPrueba(t *testing.T) application.CredencialesGobiernoCategoriaRPT {
	return credencialesGobiernoRPTActorPrueba(t, "0123456789abcdefghijkl", domain.SuperficieAutenticacionAdministracionPrivilegiadaV1, 0)
}

func credencialesGobiernoRPTActorPrueba(t *testing.T, id string, superficie domain.SuperficieAutenticacionActorV1, desfase time.Duration) application.CredencialesGobiernoCategoriaRPT {
	t.Helper()
	ahora := time.Now().Add(desfase).UTC().Truncate(time.Microsecond)
	cuenta := domain.CuentaAutenticadaContextoActor{CuentaRef: "cta_" + id, Metodo: domain.AuthMethodCertificate, Garantia: domain.AuthAssuranceHigh}
	instantanea := domain.InstantaneaContextoActor{
		VinculoRef: "vca_" + id, VinculoVersion: 5,
		CuentaRef: cuenta.CuentaRef, CuentaVersion: 7,
		PersonaRef: "per_" + id, PersonaVersion: 3,
		PerfilActivoRef: "prf_" + id, PerfilVersion: 4,
		Estado:       domain.EstadoVinculoContextoActorActivo,
		VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour),
	}
	actor, err := domain.NuevoContextoActor(cuenta, instantanea, ahora.Add(-2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	representacion, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	huella, err := actor.HuellaSHA256VinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	acreditacion := domain.AcreditacionProcedenciaComponenteContextoActorV1{
		ProcedenciaRef: "prc_0123456789abcdefghijkl", ProcedenciaVersion: 1,
		ProcedenciaHuellaSHA256: strings.Repeat("a", 64),
		ProcedenciaAutoridad:    domain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
	}
	manifiesto := domain.ManifiestoProcedenciaContextoActorV1{
		Esquema:           domain.EsquemaManifiestoProcedenciaContextoActorV1,
		AutoridadEfectiva: domain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		Cuenta:            domain.ProcedenciaCuentaContextoActorV1{CuentaRef: actor.Instantanea.CuentaRef, Version: actor.Instantanea.CuentaVersion, AcreditacionProcedenciaComponenteContextoActorV1: acreditacion},
		Persona:           domain.ProcedenciaPersonaContextoActorV1{PersonaRef: actor.PersonaRef, Version: actor.Instantanea.PersonaVersion, AcreditacionProcedenciaComponenteContextoActorV1: acreditacion},
		Perfil:            domain.ProcedenciaPerfilContextoActorV1{PerfilRef: actor.PerfilActivoRef, Version: actor.Instantanea.PerfilVersion, AcreditacionProcedenciaComponenteContextoActorV1: acreditacion},
		Contexto:          domain.ProcedenciaVinculoContextoActorV1{VinculoRef: actor.Instantanea.VinculoRef, Version: actor.Instantanea.VinculoVersion, AcreditacionProcedenciaComponenteContextoActorV1: acreditacion},
		Vinculos:          []domain.ProcedenciaVinculoReferenciaContextoActorV1{},
	}
	canonManifiesto, err := manifiesto.RepresentacionCanonicaV1()
	if err != nil {
		t.Fatal(err)
	}
	huellaManifiesto, err := domain.HuellaSHA256ManifiestoProcedenciaContextoActorV1(canonManifiesto)
	if err != nil {
		t.Fatal(err)
	}
	resultado := domain.ResultadoContextoActorRegistradoV2{
		RegistroContextoRef: "rca_0123456789abcdefghijklmn", Contexto: actor,
		RepresentacionCanonica: representacion, HuellaSHA256: huella,
		ManifiestoProcedenciaCanonico:     canonManifiesto,
		ManifiestoProcedenciaHuellaSHA256: huellaManifiesto,
		AutoridadEfectiva:                 domain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		ResueltoEnAutoritativo:            actor.ResueltoEn,
	}
	autenticacion := domain.AutenticacionRevalidadaV1{
		AutenticacionRef: "aut_0123456789abcdefghijkl", AutenticacionHuellaSHA256: strings.Repeat("1", 64),
		AsercionRef: "ase_0123456789abcdefghijkl", SesionRef: "ses_0123456789abcdefghijkl",
		ControlSesionRef: "cse_0123456789abcdefghijkl", ControlSesionRevision: 7,
		ControlSesionHuellaSHA256: strings.Repeat("2", 64),
		CuentaRef:                 cuenta.CuentaRef, CuentaOrdinariaRef: cuenta.CuentaRef,
		Superficie:      superficie,
		MetodoObservado: domain.AuthMethodCertificate, GarantiaObservada: domain.AuthAssuranceHigh,
		PoliticaGarantiaRef:          "pga_0123456789abcdefghijkl",
		PoliticaGarantiaHuellaSHA256: strings.Repeat("3", 64),
		AutenticacionVerificadaEn:    ahora.Add(-5 * time.Minute), SesionEmitidaEn: ahora.Add(-4 * time.Minute),
		SesionRevalidadaEn: ahora.Add(-3 * time.Minute), SesionValidaHasta: ahora.Add(10 * time.Minute),
	}
	if superficie == domain.SuperficieAutenticacionAdministracionPrivilegiadaV1 {
		autenticacion.CuentaPrivilegiada = true
		autenticacion.CuentaOrdinariaRef = "cta_ordinaria0123456789abcdef"
	}
	vinculo, err := domain.CrearVinculoAutenticacionActorV2(t.Context(),
		revalidadorGobiernoRPTPrueba{autenticacion},
		domain.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: autenticacion.AutenticacionRef, SesionRef: autenticacion.SesionRef},
		resolutorGobiernoRPTPrueba{resultado},
		domain.SolicitudContextoActor{Cuenta: cuenta, PerfilActivoRef: actor.PerfilActivoRef},
		relojGobiernoRPTPrueba{ahora})
	if err != nil {
		t.Fatal(err)
	}
	correlacion, err := domain.GenerarReferenciaCorrelacionAutorizacionV2(t.Context(), correlacionGobiernoRPTPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	return application.CredencialesGobiernoCategoriaRPT{Actor: actor, Vinculo: vinculo, ResultadoContexto: resultado,
		Motivo:      domain.ReferenciaEntradaCatalogo{CatalogoID: "motivos_autorizacion", CatalogoVersion: 3, CatalogoHuellaSHA256: strings.Repeat("a", 64), EntradaClave: "motivo_0123456789abcdef0123456789abcdef"},
		Correlacion: correlacion}
}

func (o *operadorGobiernoRPTPrueba) Proponer(_ context.Context, orden application.OrdenProponerGobiernoCategoriaRPT) (ports.ResultadoGobiernoCategoriaRPT, error) {
	o.llamadas++
	if o.cancelar != nil {
		o.cancelar()
	}
	o.cred = orden.Credenciales
	o.ultimo = orden.Borrador
	return o.respuesta, o.err
}
func (o *operadorGobiernoRPTPrueba) Aprobar(_ context.Context, orden application.OrdenAvanzarGobiernoCategoriaRPT) (ports.ResultadoGobiernoCategoriaRPT, error) {
	o.llamadas++
	o.cred = orden.Credenciales
	return o.respuesta, o.err
}
func (o *operadorGobiernoRPTPrueba) Confirmar(_ context.Context, orden application.OrdenAvanzarGobiernoCategoriaRPT) (ports.ResultadoGobiernoCategoriaRPT, error) {
	o.llamadas++
	o.cred = orden.Credenciales
	return o.respuesta, o.err
}

func caGobiernoRPTPrueba(t *testing.T) (*x509.CertPool, tls.Certificate, tls.Certificate) {
	t.Helper()
	claveCA, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ahora := time.Now()
	ca := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "CA ADMIN sintética"}, NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign}
	derCA, err := x509.CreateCertificate(rand.Reader, ca, ca, &claveCA.PublicKey, claveCA)
	if err != nil {
		t.Fatal(err)
	}
	caParseada, err := x509.ParseCertificate(derCA)
	if err != nil {
		t.Fatal(err)
	}
	claveCliente, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	cliente := &x509.Certificate{SerialNumber: big.NewInt(2), Subject: pkix.Name{CommonName: "actor sintético ADMIN"}, NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}}
	derCliente, err := x509.CreateCertificate(rand.Reader, cliente, caParseada, &claveCliente.PublicKey, claveCA)
	if err != nil {
		t.Fatal(err)
	}
	clavePEM, err := x509.MarshalECPrivateKey(claveCliente)
	if err != nil {
		t.Fatal(err)
	}
	p := x509.NewCertPool()
	p.AddCert(caParseada)
	material, err := tls.X509KeyPair(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derCliente}), pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: clavePEM}))
	if err != nil {
		t.Fatal(err)
	}
	claveServidor, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	servidor := &x509.Certificate{SerialNumber: big.NewInt(3), Subject: pkix.Name{CommonName: "admin.ejemplo.test"}, DNSNames: []string{"admin.ejemplo.test"}, NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	derServidor, err := x509.CreateCertificate(rand.Reader, servidor, caParseada, &claveServidor.PublicKey, claveCA)
	if err != nil {
		t.Fatal(err)
	}
	claveServidorPEM, err := x509.MarshalECPrivateKey(claveServidor)
	if err != nil {
		t.Fatal(err)
	}
	materialServidor, err := tls.X509KeyPair(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derServidor}), pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: claveServidorPEM}))
	if err != nil {
		t.Fatal(err)
	}
	return p, material, materialServidor
}

func TestGobiernoRPTConstructorDeniegaSinFronteraADMIN(t *testing.T) {
	op, fuente, audit := &operadorGobiernoRPTPrueba{}, &fuenteGobiernoRPTPrueba{}, &auditorGobiernoRPTPrueba{}
	d := ports.DescriptorCatalogoRPT{CatalogoID: "catalogo.rpt", ModuloID: "bolsa"}
	ca, _, _ := caGobiernoRPTPrueba(t)
	for _, tc := range []struct {
		name       string
		op         OperadorGobiernoCategoriaRPT
		fuente     FuenteCredencialesGobiernoCategoriaRPT
		audit      AuditorDenegacionGobiernoCategoriaRPT
		host       string
		ca         *x509.CertPool
		descriptor ports.DescriptorCatalogoRPT
		perfil     string
	}{
		{"operador", nil, fuente, audit, "admin.ejemplo.test", ca, d, versionRolGobiernoRPTPrueba},
		{"fuente", op, nil, audit, "admin.ejemplo.test", ca, d, versionRolGobiernoRPTPrueba},
		{"auditoria", op, fuente, nil, "admin.ejemplo.test", ca, d, versionRolGobiernoRPTPrueba},
		{"CA", op, fuente, audit, "admin.ejemplo.test", nil, d, versionRolGobiernoRPTPrueba},
		{"host", op, fuente, audit, "vec.ejemplo.test:8443", ca, d, versionRolGobiernoRPTPrueba},
		{"descriptor", op, fuente, audit, "admin.ejemplo.test", ca, ports.DescriptorCatalogoRPT{}, versionRolGobiernoRPTPrueba},
		{"perfil", op, fuente, audit, "admin.ejemplo.test", ca, d, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rutas, err := NuevasRutasGobiernoCategoriaRPT(tc.op, tc.fuente, tc.audit, tc.host, tc.ca, tc.descriptor, tc.perfil)
			if !errors.Is(err, ErrHandlerGobiernoCategoriaRPTInvalido) || rutas != nil {
				t.Fatalf("constructor: rutas=%v error=%v", rutas, err)
			}
		})
	}
}

func TestGobiernoRPTMTLSADMINDeniegaPortalNormalYFuenteSinPerfil(t *testing.T) {
	ca, cert, certServidor := caGobiernoRPTPrueba(t)
	op, fuente, audit := &operadorGobiernoRPTPrueba{}, &fuenteGobiernoRPTPrueba{err: ErrAccesoRutaExactaDenegado}, &auditorGobiernoRPTPrueba{}
	d := ports.DescriptorCatalogoRPT{CatalogoID: "catalogo.rpt", ModuloID: "bolsa"}
	actor := actorOrganizacionHistoricaPrueba(t)
	rutas, err := NuevasRutasGobiernoCategoriaRPT(op, fuente, audit, "admin.ejemplo.test", ca, d, versionRolGobiernoRPTPrueba)
	if err != nil || len(rutas) != 3 {
		t.Fatalf("rutas=%d error=%v", len(rutas), err)
	}
	mux := http.NewServeMux()
	for _, ruta := range rutas {
		mux.Handle(ruta.Ruta, ruta.Manejador)
	}
	servidor := httptest.NewUnstartedServer(mux)
	servidor.TLS = &tls.Config{Certificates: []tls.Certificate{certServidor}, ClientAuth: tls.VerifyClientCertIfGiven, ClientCAs: ca, MinVersion: tls.VersionTLS12}
	servidor.StartTLS()
	defer servidor.Close()
	cliente := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: ca, Certificates: []tls.Certificate{cert}, ServerName: "admin.ejemplo.test", MinVersion: tls.VersionTLS12}}}
	sinCert := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: ca, ServerName: "admin.ejemplo.test", MinVersion: tls.VersionTLS12}}}
	clave := "12345678-1234-4234-8234-123456789abc"
	cuerpo := `{"propuesta_ref":"propuesta:ejemplo","catalogo_id":"catalogo.rpt","modulo_id":"bolsa","revision_esperada":1,"huella_sha256":"` + strings.Repeat("a", 64) + `"}`
	enviar := func(host, cuerpoEnvio string, c *http.Client) int {
		t.Helper()
		r, err := http.NewRequest(http.MethodPost, servidor.URL+RutaAprobarGobiernoCategoriaRPT, strings.NewReader(cuerpoEnvio))
		if err != nil {
			t.Fatal(err)
		}
		r.Host = host
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Idempotency-Key", clave)
		resp, err := c.Do(r)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		return resp.StatusCode
	}
	if estado := enviar("admin.ejemplo.test", cuerpo, sinCert); estado != http.StatusUnauthorized {
		t.Fatalf("sin certificado=%d", estado)
	}
	if estado := enviar("vec.ejemplo.test", cuerpo, cliente); estado != http.StatusUnauthorized {
		t.Fatalf("portal ordinario=%d", estado)
	}
	if estado := enviar("admin.ejemplo.test", cuerpo, cliente); estado != http.StatusForbidden {
		t.Fatalf("fuente sin perfil=%d", estado)
	}
	if estado := enviar("admin.ejemplo.test", `{"actor":"falso"}`, cliente); estado != http.StatusForbidden {
		t.Fatalf("actor del cuerpo con fuente denegada=%d", estado)
	}
	fuente.err = nil
	fuente.descriptor = d
	fuente.asignacion = asignacionGobiernoRPTPrueba(t, credencialesGobiernoRPTPrueba(t))
	fuente.cred.Actor = actor
	fuente.cred.Actor.Principal.AuthAssurance = domain.AuthAssuranceSubstantial
	if estado := enviar("admin.ejemplo.test", cuerpo, cliente); estado != http.StatusForbidden {
		t.Fatalf("perfil substantial=%d", estado)
	}
	if op.llamadas != 0 || fuente.llamadas != 3 || len(audit.codigos) != 5 {
		t.Fatalf("efecto=%d fuente=%d auditoria=%v", op.llamadas, fuente.llamadas, audit.codigos)
	}
}

func TestGobiernoRPTDTORechazaAutoridadYHuellasCliente(t *testing.T) {
	for _, tc := range []struct{ ruta, campo string }{
		{RutaProponerGobiernoCategoriaRPT, "actor"}, {RutaProponerGobiernoCategoriaRPT, "perfil_activo_ref"},
		{RutaProponerGobiernoCategoriaRPT, "version_rol_ref"}, {RutaProponerGobiernoCategoriaRPT, "auth_assurance"}, {RutaProponerGobiernoCategoriaRPT, "preimagenes_huella_sha256"},
		{RutaProponerGobiernoCategoriaRPT, "documento_huella_sha256"}, {RutaProponerGobiernoCategoriaRPT, "huella_sha256"},
		{RutaAprobarGobiernoCategoriaRPT, "fuente_ref"}, {RutaConfirmarGobiernoCategoriaRPT, "motivo_ref"},
	} {
		t.Run(tc.campo, func(t *testing.T) {
			if camposGobiernoRPTAdmitidos(tc.ruta, map[string]json.RawMessage{tc.campo: json.RawMessage(`null`)}) {
				t.Fatal("campo de autoridad admitido")
			}
		})
	}
	for _, cuerpo := range []string{
		`{"propuesta_ref":"propuesta:ejemplo","actor":"falso"}`,
		`{"propuesta_ref":"propuesta:ejemplo","perfil_activo_ref":"falso"}`,
		`{"propuesta_ref":"propuesta:ejemplo","huella_sha256":null}`,
		`{"propuesta_ref":"propuesta:ejemplo","propuesta_ref":"otra"}`,
	} {
		r := httptest.NewRequest(http.MethodPost, RutaProponerGobiernoCategoriaRPT, strings.NewReader(cuerpo))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Idempotency-Key", "12345678-1234-4234-8234-123456789abc")
		if _, _, err := leerEntradaGobiernoRPT(httptest.NewRecorder(), r); !errors.Is(err, ErrHandlerGobiernoCategoriaRPTInvalido) {
			t.Fatalf("entrada de autoridad admitida: %s", cuerpo)
		}
	}
}

func peticionGobiernoRPTPrueba(t *testing.T, ca *x509.CertPool, material tls.Certificate, ruta, cuerpo string) *http.Request {
	t.Helper()
	cert, err := x509.ParseCertificate(material.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	cadenas, err := cert.Verify(x509.VerifyOptions{Roots: ca, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}})
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodPost, ruta, strings.NewReader(cuerpo))
	r.Host = "admin.ejemplo.test"
	r.TLS = &tls.ConnectionState{HandshakeComplete: true, Version: tls.VersionTLS13, ServerName: "admin.ejemplo.test", PeerCertificates: []*x509.Certificate{cert}, VerifiedChains: cadenas}
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Idempotency-Key", "12345678-1234-4234-8234-123456789abc")
	return r
}

func TestGobiernoRPTFronteraValidaYDenegacionesAuditadas(t *testing.T) {
	ca, cert, _ := caGobiernoRPTPrueba(t)
	cred := credencialesGobiernoRPTPrueba(t)
	d := ports.DescriptorCatalogoRPT{CatalogoID: "catalogo.rpt", ModuloID: "bolsa"}
	fuente := &fuenteGobiernoRPTPrueba{cred: cred, descriptor: d, asignacion: asignacionGobiernoRPTPrueba(t, cred)}
	audit := &auditorGobiernoRPTPrueba{}
	resultado := ports.ResultadoGobiernoCategoriaRPT{PropuestaRef: "propuesta:ejemplo", ReciboRef: "12345678-1234-4234-8234-123456789abc", HuellaSHA256: strings.Repeat("a", 64), Revision: 1, Estado: "propuesta", Evidencia: ports.EvidenciaGobiernoCategoriaRPT{AuditoriaRef: "aud:prueba", ConsumoNuevo: true}}
	op := &operadorGobiernoRPTPrueba{respuesta: resultado}
	rutas, err := NuevasRutasGobiernoCategoriaRPT(op, fuente, audit, "admin.ejemplo.test", ca, d, versionRolGobiernoRPTPrueba)
	if err != nil {
		t.Fatal(err)
	}
	h := rutas[0].Manejador
	cuerpo := `{"propuesta_ref":"propuesta:ejemplo","accion":"deshabilitar","catalogo_id":"catalogo.rpt","modulo_id":"bolsa","version":1,"preimagenes_control":{"categoria.enfermeria":{"version":1,"huella_sha256":"` + strings.Repeat("a", 64) + `","revision":1,"estado":"habilitada"},"enfermeria-general":{"version":1,"huella_sha256":"` + strings.Repeat("b", 64) + `","revision":2,"estado":"habilitada"}},"categoria_id":"categoria.enfermeria","revision_esperada":1,"fuente_ref":"fuente:prueba"}`
	cuerpo = strings.Replace(cuerpo, `"enfermeria-general":{`, `"categoria:enfermeria":{"version":1,"huella_sha256":"`+strings.Repeat("c", 64)+`","revision":3,"estado":"habilitada"},"enfermeria-general":{`, 1)
	enviar := func(r *http.Request) int { t.Helper(); w := httptest.NewRecorder(); h.ServeHTTP(w, r); return w.Code }
	for i := 0; i < 2; i++ {
		r := peticionGobiernoRPTPrueba(t, ca, cert, RutaProponerGobiernoCategoriaRPT, cuerpo)
		if estado := enviar(r); estado != http.StatusOK {
			t.Fatalf("acto/replay %d: %d", i, estado)
		}
	}
	rOrigen := peticionGobiernoRPTPrueba(t, ca, cert, RutaProponerGobiernoCategoriaRPT, cuerpo)
	rOrigen.Header.Set("Origin", "https://admin.ejemplo.test")
	rOrigen.Header.Set("Sec-Fetch-Site", "same-origin")
	if estado := enviar(rOrigen); estado != http.StatusOK {
		t.Fatalf("origen ADMIN=%d", estado)
	}
	if op.llamadas != 3 || len(op.ultimo.Contenido.PreimagenesControl) != 3 || len(audit.codigos) != 0 {
		t.Fatalf("camino válido: op=%d preimagenes=%v audit=%v", op.llamadas, op.ultimo.Contenido.PreimagenesControl, audit.codigos)
	}
	preflight := peticionGobiernoRPTPrueba(t, ca, cert, RutaProponerGobiernoCategoriaRPT, cuerpo)
	preflight.Method = http.MethodOptions
	preflight.Header.Set("Origin", "https://vec.ejemplo.test")
	preflight.Header.Set("Access-Control-Request-Method", http.MethodPost)
	wPreflight := httptest.NewRecorder()
	h.ServeHTTP(wPreflight, preflight)
	if wPreflight.Code != http.StatusMethodNotAllowed || wPreflight.Header().Get("Access-Control-Allow-Origin") != "" || op.llamadas != 3 {
		t.Fatalf("preflight permitido: estado=%d cabeceras=%v op=%d", wPreflight.Code, wPreflight.Header(), op.llamadas)
	}
	casos := []struct {
		name     string
		preparar func(*http.Request)
		cuerpo   string
		estado   int
	}{
		{"authorization vacia", func(r *http.Request) { r.Header["Authorization"] = []string{""} }, cuerpo, http.StatusBadRequest},
		{"authorization duplicada", func(r *http.Request) { r.Header["Authorization"] = []string{"", "Bearer falso"} }, cuerpo, http.StatusBadRequest},
		{"cookie", func(r *http.Request) { r.Header.Set("Cookie", "") }, cuerpo, http.StatusBadRequest},
		{"proxy authorization", func(r *http.Request) { r.Header.Set("Proxy-Authorization", "") }, cuerpo, http.StatusBadRequest},
		{"origin cruzado", func(r *http.Request) { r.Header.Set("Origin", "https://vec.ejemplo.test") }, cuerpo, http.StatusForbidden},
		{"origin null", func(r *http.Request) { r.Header.Set("Origin", "null") }, cuerpo, http.StatusForbidden},
		{"origin duplicado", func(r *http.Request) {
			r.Header["Origin"] = []string{"https://admin.ejemplo.test", "https://admin.ejemplo.test"}
		}, cuerpo, http.StatusForbidden},
		{"fetch cross site", func(r *http.Request) { r.Header.Set("Sec-Fetch-Site", "cross-site") }, cuerpo, http.StatusForbidden},
		{"clave inyectada", nil, strings.Replace(cuerpo, `"categoria.enfermeria":{`, `"categoria.enfermeria' OR 1=1 --":{`, 1), http.StatusBadRequest},
		{"preimagen duplicada", nil, strings.Replace(cuerpo, `"enfermeria-general":{`, `"categoria.enfermeria":{`, 1), http.StatusBadRequest},
		{"preimagen duplicada escapada", nil, strings.Replace(cuerpo, `"enfermeria-general":{`, `"categoria\u002eenfermeria":{`, 1), http.StatusBadRequest},
		{"campo preimagen desconocido", nil, strings.Replace(cuerpo, `"estado":"habilitada"`, `"estado":"habilitada","actor":"falso"`, 1), http.StatusBadRequest},
		{"tipo preimagen", nil, strings.Replace(cuerpo, `"categoria.enfermeria":{"version":1`, `"categoria.enfermeria":{"version":"1"`, 1), http.StatusBadRequest},
		{"tamano", nil, cuerpo + strings.Repeat(" ", maximoCuerpoGobiernoCategoriaRPT), http.StatusBadRequest},
	}
	for _, tc := range casos {
		t.Run(tc.name, func(t *testing.T) {
			r := peticionGobiernoRPTPrueba(t, ca, cert, RutaProponerGobiernoCategoriaRPT, tc.cuerpo)
			if tc.preparar != nil {
				tc.preparar(r)
			}
			antes := op.llamadas
			if estado := enviar(r); estado != tc.estado || op.llamadas != antes {
				t.Fatalf("estado=%d op=%d antes=%d", estado, op.llamadas, antes)
			}
		})
	}
	if len(audit.codigos) != len(casos) {
		t.Fatalf("fallos sin auditar: %v", audit.codigos)
	}
	for _, tc := range []struct {
		name   string
		err    error
		estado int
	}{
		{"invalida", ports.ErrGobiernoCategoriaRPTInvalido, http.StatusBadRequest},
		{"denegada", ports.ErrGobiernoCategoriaRPTDenegado, http.StatusForbidden},
		{"conflicto", ports.ErrGobiernoCategoriaRPTConflicto, http.StatusConflict},
		{"no disponible", ports.ErrGobiernoCategoriaRPTNoDisponible, http.StatusServiceUnavailable},
		{"cancelada", context.Canceled, http.StatusServiceUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			op.err = tc.err
			antes := len(audit.codigos)
			if estado := enviar(peticionGobiernoRPTPrueba(t, ca, cert, RutaProponerGobiernoCategoriaRPT, cuerpo)); estado != tc.estado || len(audit.codigos) != antes+1 {
				t.Fatalf("estado=%d auditoria=%v", estado, audit.codigos)
			}
		})
	}
	op.err = nil
	for _, tc := range []struct {
		name   string
		err    error
		estado int
	}{{"fuente 401", ErrAutenticacionRutaExactaRequerida, http.StatusUnauthorized}, {"fuente 403", ErrAccesoRutaExactaDenegado, http.StatusForbidden}, {"fuente 503", errors.New("fallo sintético"), http.StatusServiceUnavailable}, {"fuente cancelada", context.Canceled, http.StatusServiceUnavailable}} {
		t.Run(tc.name, func(t *testing.T) {
			fuente.err = tc.err
			antes := len(audit.codigos)
			llamadas := op.llamadas
			if estado := enviar(peticionGobiernoRPTPrueba(t, ca, cert, RutaProponerGobiernoCategoriaRPT, cuerpo)); estado != tc.estado || len(audit.codigos) != antes+1 || op.llamadas != llamadas {
				t.Fatalf("estado=%d auditoria=%v op=%d", estado, audit.codigos, op.llamadas)
			}
		})
	}
	fuente.err = nil
	rLecturaCancelada := peticionGobiernoRPTPrueba(t, ca, cert, RutaProponerGobiernoCategoriaRPT, cuerpo)
	ctxLectura, cancelarLectura := context.WithCancel(rLecturaCancelada.Context())
	rLecturaCancelada = rLecturaCancelada.WithContext(ctxLectura)
	rLecturaCancelada.Body = &lectorGobiernoRPTCancela{Reader: strings.NewReader(cuerpo), cancelar: cancelarLectura}
	antesAuditoria, antesOperador := len(audit.codigos), op.llamadas
	if estado := enviar(rLecturaCancelada); estado != http.StatusServiceUnavailable || len(audit.codigos) != antesAuditoria+1 || op.llamadas != antesOperador || audit.codigos[len(audit.codigos)-1] != "servicio_no_disponible" {
		t.Fatalf("lectura cancelada: estado=%d auditoria=%v operador=%d", estado, audit.codigos, op.llamadas)
	}
	rCancelado := peticionGobiernoRPTPrueba(t, ca, cert, RutaProponerGobiernoCategoriaRPT, cuerpo)
	ctxFuente, cancelarFuente := context.WithCancel(rCancelado.Context())
	rCancelado = rCancelado.WithContext(ctxFuente)
	fuente.cancelar = cancelarFuente
	antesAuditoria, antesOperador = len(audit.codigos), op.llamadas
	if estado := enviar(rCancelado); estado != http.StatusServiceUnavailable || len(audit.codigos) != antesAuditoria+1 || op.llamadas != antesOperador {
		t.Fatalf("fuente cancelada: estado=%d auditoria=%v operador=%d", estado, audit.codigos, op.llamadas)
	}
	fuente.cancelar = nil
	rCancelado = peticionGobiernoRPTPrueba(t, ca, cert, RutaProponerGobiernoCategoriaRPT, cuerpo)
	ctxOperador, cancelarOperador := context.WithCancel(rCancelado.Context())
	rCancelado = rCancelado.WithContext(ctxOperador)
	op.cancelar = cancelarOperador
	antesAuditoria = len(audit.codigos)
	if estado := enviar(rCancelado); estado != http.StatusServiceUnavailable || len(audit.codigos) != antesAuditoria+1 {
		t.Fatalf("operador cancelado: estado=%d auditoria=%v", estado, audit.codigos)
	}
	op.cancelar = nil
	rCancelado = peticionGobiernoRPTPrueba(t, ca, cert, RutaProponerGobiernoCategoriaRPT, cuerpo)
	ctxErrorOperador, cancelarErrorOperador := context.WithCancel(rCancelado.Context())
	rCancelado = rCancelado.WithContext(ctxErrorOperador)
	op.cancelar = cancelarErrorOperador
	op.err = ports.ErrGobiernoCategoriaRPTInvalido
	antesAuditoria = len(audit.codigos)
	if estado := enviar(rCancelado); estado != http.StatusServiceUnavailable || len(audit.codigos) != antesAuditoria+1 || audit.codigos[len(audit.codigos)-1] != "servicio_no_disponible" {
		t.Fatalf("error con cancelacion: estado=%d auditoria=%v", estado, audit.codigos)
	}
	op.cancelar = nil
	op.err = nil
	audit.err = errors.New("auditoria caída")
	op.err = ports.ErrGobiernoCategoriaRPTConflicto
	if estado := enviar(peticionGobiernoRPTPrueba(t, ca, cert, RutaProponerGobiernoCategoriaRPT, cuerpo)); estado != http.StatusServiceUnavailable {
		t.Fatalf("fallo auditor=%d", estado)
	}
}

func asignacionGobiernoRPTPrueba(t *testing.T, cred application.CredencialesGobiernoCategoriaRPT) domain.InstantaneaAutorizacion {
	t.Helper()
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	rol := domain.VersionRol{RolID: "configuracion_rpt", Version: 1, Nombre: "Configuracion RPT sintetica", Estado: domain.EstadoVersionRolPublicada,
		PublicadaPor: "autoridad:prueba", PublicadaEn: ahora.Add(-2 * time.Hour)}
	for _, accion := range []string{ports.AccionProponerGobiernoCategoriaRPT, ports.AccionAprobarGobiernoCategoriaRPT, ports.AccionConfirmarGobiernoCategoriaRPT} {
		rol.Concesiones = append(rol.Concesiones, domain.ConcesionRol{Accion: accion, ModuloID: "bolsa", TipoRecurso: ports.TipoRecursoGobiernoCategoriaRPT,
			Finalidades: []string{ports.FinalidadGobiernoCategoriaRPT}, GarantiaMinima: domain.AuthAssuranceHigh, CamposPermitidos: []string{"gobierno", "recibo"}})
	}
	huella, err := domain.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		t.Fatal(err)
	}
	i := domain.InstantaneaAutorizacion{VersionRol: rol,
		AsignacionPerfil: domain.AsignacionPerfil{AsignacionID: "admin:" + cred.Actor.Principal.ID, Version: 1,
			PrincipalID: cred.Actor.Principal.ID, PerfilActivoRef: cred.Actor.PerfilActivoRef, VersionRolRef: rol.Referencia(),
			Estado: domain.EstadoAsignacionPerfilActiva, EmitidaPor: "autoridad:prueba", EmitidaEn: ahora.Add(-time.Hour),
			VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour),
			Ambitos: []domain.AmbitoPerfil{{Clave: "catalogo_id", Valores: []string{"catalogo.rpt"}}, {Clave: "modulo_id", Valores: []string{"bolsa"}}}},
		ControlVigenciaVersionRol: domain.ControlVigenciaVersionRol{VersionRolRef: rol.Referencia(), Revision: 1,
			Estado: domain.EstadoControlVigenciaVersionRolHabilitada, ActualizadoPor: rol.PublicadaPor, ActualizadoEn: rol.PublicadaEn},
		RevisionCatalogoPoliticas: 1, CatalogoPoliticasHuellaSHA256: huella}
	if err := i.Validar(); err != nil {
		t.Fatalf("instantanea: %v rol=%v asignacion=%v control=%v", err, i.VersionRol.Validar(), i.AsignacionPerfil.Validar(), i.ControlVigenciaVersionRol.Validar())
	}
	return i
}

func TestGobiernoRPTAdmiteAsignacionesIndividualesAlMismoRolADMIN(t *testing.T) {
	ca, cert, _ := caGobiernoRPTPrueba(t)
	actores := []application.CredencialesGobiernoCategoriaRPT{
		credencialesGobiernoRPTPrueba(t),
		credencialesGobiernoRPTActorPrueba(t, "abcdefghijkl0123456789", domain.SuperficieAutenticacionAdministracionPrivilegiadaV1, 0),
	}
	if actores[0].Actor.Principal.ID == actores[1].Actor.Principal.ID || actores[0].Actor.PerfilActivoRef == actores[1].Actor.PerfilActivoRef {
		t.Fatal("el ejercicio requiere personas y perfiles distintos")
	}
	d := ports.DescriptorCatalogoRPT{CatalogoID: "catalogo.rpt", ModuloID: "bolsa"}
	fuente, audit := &fuenteGobiernoRPTPrueba{descriptor: d}, &auditorGobiernoRPTPrueba{}
	op := &operadorGobiernoRPTPrueba{respuesta: ports.ResultadoGobiernoCategoriaRPT{PropuestaRef: "propuesta:ejemplo",
		ReciboRef: "12345678-1234-4234-8234-123456789abc", Evidencia: ports.EvidenciaGobiernoCategoriaRPT{AuditoriaRef: "aud:prueba", ConsumoNuevo: true}}}
	rutas, err := NuevasRutasGobiernoCategoriaRPT(op, fuente, audit, "admin.ejemplo.test", ca, d, versionRolGobiernoRPTPrueba)
	if err != nil {
		t.Fatal(err)
	}
	propuesta := `{"propuesta_ref":"propuesta:ejemplo","accion":"deshabilitar","catalogo_id":"catalogo.rpt","modulo_id":"bolsa","version":1,"preimagenes_control":{},"categoria_id":"categoria:ejemplo","revision_esperada":1,"fuente_ref":"fuente:prueba"}`
	avance := `{"propuesta_ref":"propuesta:ejemplo","catalogo_id":"catalogo.rpt","modulo_id":"bolsa","revision_esperada":1,"huella_sha256":"` + strings.Repeat("a", 64) + `"}`
	for _, cred := range actores {
		fuente.cred, fuente.asignacion = cred, asignacionGobiernoRPTPrueba(t, cred)
		for _, ruta := range rutas {
			cuerpo := avance
			if ruta.Ruta == RutaProponerGobiernoCategoriaRPT {
				cuerpo = propuesta
			}
			w := httptest.NewRecorder()
			ruta.Manejador.ServeHTTP(w, peticionGobiernoRPTPrueba(t, ca, cert, ruta.Ruta, cuerpo))
			if w.Code != http.StatusOK {
				t.Fatalf("actor=%s ruta=%s estado=%d", cred.Actor.Principal.ID, ruta.Ruta, w.Code)
			}
			if op.cred.Actor.Principal.ID != cred.Actor.Principal.ID || op.cred.Actor.PerfilActivoRef != cred.Actor.PerfilActivoRef {
				t.Fatal("la frontera sustituyo la asignacion individual")
			}
		}
	}
	if op.llamadas != 6 || len(audit.codigos) != 0 {
		t.Fatalf("efectos=%d auditoria=%v", op.llamadas, audit.codigos)
	}

	for _, caso := range []struct {
		nombre  string
		cambiar func()
	}{
		{"asignacion ajena", func() { fuente.asignacion.AsignacionPerfil.PrincipalID = actores[1].Actor.Principal.ID }},
		{"perfil ajeno", func() { fuente.asignacion.AsignacionPerfil.PerfilActivoRef = actores[1].Actor.PerfilActivoRef }},
		{"vinculo ajeno", func() { fuente.cred.Vinculo = actores[1].Vinculo }},
		{"actor ajeno", func() { fuente.cred.Actor = actores[1].Actor }},
		{"rol distinto", func() {
			fuente.asignacion.VersionRol.RolID = "otro_rol"
			fuente.asignacion.AsignacionPerfil.VersionRolRef = fuente.asignacion.VersionRol.Referencia()
			fuente.asignacion.ControlVigenciaVersionRol.VersionRolRef = fuente.asignacion.VersionRol.Referencia()
		}},
		{"rol retirado", func() {
			fuente.asignacion.ControlVigenciaVersionRol.Estado = domain.EstadoControlVigenciaVersionRolRetirada
			fuente.asignacion.ControlVigenciaVersionRol.ActoRef = "acto:retirada"
			fuente.asignacion.ControlVigenciaVersionRol.MotivoCodigo = "retirada"
		}},
		{"version de rol retirada", func() {
			fuente.asignacion.VersionRol.Estado = domain.EstadoVersionRolRetirada
			fuente.asignacion.VersionRol.RetiradaPor = "autoridad:prueba"
			fuente.asignacion.VersionRol.RetiradaEn = time.Now().UTC().Add(-time.Minute).Truncate(time.Microsecond)
			fuente.asignacion.VersionRol.RetiradaRef = "acto:retirada"
			fuente.asignacion.VersionRol.MotivoRetiradaCodigo = "retirada"
			fuente.asignacion.ControlVigenciaVersionRol.Estado = domain.EstadoControlVigenciaVersionRolRetirada
			fuente.asignacion.ControlVigenciaVersionRol.ActoRef = "acto:retirada"
			fuente.asignacion.ControlVigenciaVersionRol.MotivoCodigo = "retirada"
		}},
		{"asignacion revocada", func() {
			fuente.asignacion.AsignacionPerfil.Estado = domain.EstadoAsignacionPerfilRevocada
			fuente.asignacion.AsignacionPerfil.RevocadaPor = "autoridad:prueba"
			fuente.asignacion.AsignacionPerfil.RevocadaEn = time.Now().UTC().Add(-time.Minute).Truncate(time.Microsecond)
			fuente.asignacion.AsignacionPerfil.RevocacionRef = "acto:revocacion"
		}},
		{"asignacion vencida", func() {
			fuente.asignacion.AsignacionPerfil.VigenteHasta = time.Now().UTC().Add(-time.Minute).Truncate(time.Microsecond)
		}},
		{"contexto vencido", func() {
			fuente.cred = credencialesGobiernoRPTActorPrueba(t, "0123456789abcdefghijkl", domain.SuperficieAutenticacionAdministracionPrivilegiadaV1, -2*time.Hour)
		}},
		{"superficie ordinaria", func() {
			fuente.cred = credencialesGobiernoRPTActorPrueba(t, "0123456789abcdefghijkl", domain.SuperficieAutenticacionInternaCorporativaV1, 0)
		}},
		{"garantia insuficiente", func() { fuente.cred.Actor.Principal.AuthAssurance = domain.AuthAssuranceSubstantial }},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			fuente.cred, fuente.asignacion = actores[0], asignacionGobiernoRPTPrueba(t, actores[0])
			caso.cambiar()
			antes := op.llamadas
			w := httptest.NewRecorder()
			rutas[1].Manejador.ServeHTTP(w, peticionGobiernoRPTPrueba(t, ca, cert, rutas[1].Ruta, avance))
			if w.Code != http.StatusForbidden || op.llamadas != antes {
				t.Fatalf("estado=%d efectos=%d", w.Code, op.llamadas-antes)
			}
		})
	}
}
