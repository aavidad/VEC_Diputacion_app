package httpseguridad

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type relojFijo struct {
	mu    sync.RWMutex
	ahora time.Time
}

func (r *relojFijo) Ahora() time.Time {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.ahora
}

func (r *relojFijo) fijar(valor time.Time) {
	r.mu.Lock()
	r.ahora = valor
	r.mu.Unlock()
}

type verificadorFalso struct {
	mu       sync.RWMutex
	asercion AsercionProxyIdentidad
	err      error
	mutar    bool
	ultima   []byte
	llamadas atomic.Int64
}

func (v *verificadorFalso) Verificar(_ context.Context, protegida []byte) (AsercionProxyIdentidad, error) {
	v.llamadas.Add(1)
	v.mu.Lock()
	v.ultima = append([]byte(nil), protegida...)
	asercion := copiarAsercion(v.asercion)
	err := v.err
	mutar := v.mutar
	v.mu.Unlock()
	if mutar && len(protegida) > 0 {
		protegida[0] = 'X'
	}
	return asercion, err
}

func (v *verificadorFalso) fijarAsercion(asercion AsercionProxyIdentidad) {
	v.mu.Lock()
	v.asercion = copiarAsercion(asercion)
	v.mu.Unlock()
}

func (v *verificadorFalso) fijarError(err error) {
	v.mu.Lock()
	v.err = err
	v.mu.Unlock()
}

type evaluadorFalso struct {
	mu        sync.RWMutex
	resultado ResultadoEvaluacionGarantia
	err       error
	llamadas  atomic.Int64
}

func (e *evaluadorFalso) Evaluar(_ context.Context, entrada EntradaEvaluacionGarantia) (ResultadoEvaluacionGarantia, error) {
	e.llamadas.Add(1)
	// Una mutacion del puerto no puede alcanzar los factores conservados por el
	// servicio porque recibe una copia defensiva.
	if len(entrada.Factores) > 0 {
		entrada.Factores[0].EvidenciaRef = "mutada-por-evaluador"
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.resultado, e.err
}

func (e *evaluadorFalso) fijarResultado(resultado ResultadoEvaluacionGarantia) {
	e.mu.Lock()
	e.resultado = resultado
	e.mu.Unlock()
}

func (e *evaluadorFalso) fijarError(err error) {
	e.mu.Lock()
	e.err = err
	e.mu.Unlock()
}

func TestServicioIdentidadResuelveYProyectaContextoAuditable(t *testing.T) {
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	configuracion := configuracionInternaValida()
	verificador := &verificadorFalso{mutar: true}
	evaluador := evaluadorValido(dominiovec.AuthAssuranceHigh)
	registro := nuevoRegistroMemoria()
	reloj := &relojFijo{ahora: ahora}
	servicio := debeServicio(t, configuracion, verificador, evaluador, registro, reloj)
	canal := debeCanalTLS(t, servicio, configuracion)
	verificador.fijarAsercion(asercionInternaValida(ahora, configuracion, canal))

	protegida := []byte("asercion-firmada-opaca")
	credencial := debeCredencial(t, protegida, canal)
	protegida[0] = 'Z'
	identidad, err := servicio.Resolver(context.Background(), credencial)
	if err != nil {
		t.Fatalf("resolver identidad: %v", err)
	}
	if string(verificador.ultima) != "asercion-firmada-opaca" {
		t.Fatalf("el constructor no copio los bytes: %q", verificador.ultima)
	}
	if string(credencial.asercionProtegida) != "asercion-firmada-opaca" {
		t.Fatalf("el verificador pudo mutar el material privado hasheado: %q", credencial.asercionProtegida)
	}
	if identidad.confirmacion.AltaConfirmada.EspacioIdentidad != configuracion.EmisorIdentidad {
		t.Fatalf("alta sin espacio de identidad protegido: %q", identidad.confirmacion.AltaConfirmada.EspacioIdentidad)
	}

	cuenta, auditoria, err := servicio.ProyectarCuentaAutenticada(context.Background(), identidad)
	if err != nil {
		t.Fatalf("proyectar cuenta autenticada: %v", err)
	}
	if cuenta.CuentaRef != identidad.confirmacion.CuentaRef ||
		cuenta.Metodo != dominiovec.AuthMethodCertificate ||
		cuenta.Garantia != dominiovec.AuthAssuranceHigh || cuenta.Validar() != nil {
		t.Fatalf("cuenta autenticada inesperada: %#v", cuenta)
	}
	resultado := fmt.Sprintf("%#v %#v %s %s", cuenta, auditoria, debeJSON(t, cuenta), debeJSON(t, auditoria))
	for _, identificadorIdP := range []string{identidad.estado.cuenta.ID, identidad.estado.sujetoID} {
		if strings.Contains(resultado, identificadorIdP) {
			t.Fatalf("la proyeccion filtro el identificador del IdP %q: %q", identificadorIdP, resultado)
		}
	}
	if auditoria.Superficie() != SuperficieInternaCorporativa ||
		auditoria.Garantia() != dominiovec.AuthAssuranceHigh || len(auditoria.Factores()) != 2 ||
		auditoria.PoliticaGarantiaRef() != "pga_0123456789abcdefghijkl" ||
		!strings.HasPrefix(auditoria.HuellaConfiguracion(), "sha256:") {
		t.Fatalf("contexto de auditoria incompleto: %#v", auditoria)
	}
	huellaProtegida := sha256.Sum256([]byte("asercion-firmada-opaca"))
	if auditoria.AutenticacionHuellaSHA256() != fmt.Sprintf("%x", huellaProtegida) ||
		auditoria.AutenticacionHuellaSHA256() != identidad.confirmacion.AltaConfirmada.AutenticacionHuellaSHA256 ||
		auditoria.MetodoObservado() != dominiovec.AuthMethodCertificate ||
		auditoria.PoliticaGarantiaHuellaSHA256() != strings.Repeat("a", 64) ||
		auditoria.ControlSesionRevision() != 1 || auditoria.ControlSesionEstado() != EstadoControlSesionActiva ||
		auditoria.ControlSesionHuellaSHA256() != identidad.confirmacion.ControlSesionHuellaSHA256 ||
		!auditoria.AutenticacionVerificadaEn().Equal(identidad.estado.autenticacionVerificadaEn) ||
		!auditoria.SesionRevalidadaEn().Equal(identidad.confirmacion.SesionRevalidadaEn) ||
		!auditoria.SesionValidaHasta().Equal(identidad.confirmacion.SesionValidaHasta) {
		t.Fatalf("proyeccion autoritativa incompleta: %#v", auditoria)
	}
	factores := auditoria.Factores()
	factores[0].EvidenciaRef = "mutada"
	if auditoria.Factores()[0].EvidenciaRef == "mutada" {
		t.Fatal("el contexto debe devolver copias de las colecciones")
	}
}

func TestCanalSoloNaceDeHandshakeTLSMutuoVerificado(t *testing.T) {
	configuracion := configuracionInternaValida()
	servicio := debeServicio(t, configuracion, &verificadorFalso{}, evaluadorValido(dominiovec.AuthAssuranceHigh), nuevoRegistroMemoria(), &relojFijo{ahora: time.Now()})
	estado := estadoTLSMutuoReal(t, sanDNSConfigurado(configuracion))
	canal, err := servicio.AutenticarCanalTLSMutuo(estado)
	if err != nil || canal.Tipo() != CanalProxyTLSMutuo || canal.Superficie() != configuracion.Superficie ||
		canal.ReferenciaVinculacion() == "" || canal.IdentidadPar() != configuracion.IdentidadesSANProxyPermitidas[0] {
		t.Fatalf("canal mTLS valido inesperado: %#v, %v", canal, err)
	}
	otroCanal, err := servicio.AutenticarCanalTLSMutuo(estadoTLSMutuoReal(t, sanDNSConfigurado(configuracion)))
	if err != nil || otroCanal.ReferenciaVinculacion() == canal.ReferenciaVinculacion() {
		t.Fatalf("la referencia debe quedar ligada a cada conexion TLS: %v", err)
	}

	pruebas := []struct {
		nombre    string
		modificar func(*tls.ConnectionState)
	}{
		{"sin handshake", func(e *tls.ConnectionState) { e.HandshakeComplete = false }},
		{"tls antiguo", func(e *tls.ConnectionState) { e.Version = tls.VersionTLS11 }},
		{"suite no acreditada", func(e *tls.ConnectionState) { e.CipherSuite = 0 }},
		{"sin cadena verificada", func(e *tls.ConnectionState) { e.VerifiedChains = nil }},
		{"sin certificado par", func(e *tls.ConnectionState) { e.PeerCertificates = nil }},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			invalido := estado
			prueba.modificar(&invalido)
			if _, err := servicio.AutenticarCanalTLSMutuo(invalido); !errors.Is(err, ErrCanalProxyNoAutenticado) {
				t.Fatalf("canal no verificado aceptado: %v", err)
			}
		})
	}
	estadoFabricado := tls.ConnectionState{
		HandshakeComplete: true,
		Version:           estado.Version,
		CipherSuite:       estado.CipherSuite,
		PeerCertificates:  estado.PeerCertificates,
		VerifiedChains:    estado.VerifiedChains,
	}
	if _, err := servicio.AutenticarCanalTLSMutuo(estadoFabricado); !errors.Is(err, ErrCanalProxyNoAutenticado) {
		t.Fatalf("copiar campos publicos sin exportador TLS no debe fabricar un canal: %v", err)
	}

	otraConfiguracion := configuracion
	otraConfiguracion.IdentidadesSANProxyPermitidas = []string{"dns:otro-proxy.corporativa.example"}
	otroServicio := debeServicio(t, otraConfiguracion, &verificadorFalso{}, evaluadorValido(dominiovec.AuthAssuranceHigh), nuevoRegistroMemoria(), &relojFijo{ahora: time.Now()})
	if _, err := otroServicio.AutenticarCanalTLSMutuo(estado); !errors.Is(err, ErrCanalProxyNoAutenticado) {
		t.Fatalf("SAN no permitido aceptado: %v", err)
	}

	configuracionConPin := configuracion
	huellaReal := sha256.Sum256(estado.PeerCertificates[0].Raw)
	configuracionConPin.HuellasProxyTLSPermitidas = []string{fmt.Sprintf("sha256:%x", huellaReal)}
	servicioConPin := debeServicio(t, configuracionConPin, &verificadorFalso{}, evaluadorValido(dominiovec.AuthAssuranceHigh), nuevoRegistroMemoria(), &relojFijo{ahora: time.Now()})
	if _, err := servicioConPin.AutenticarCanalTLSMutuo(estado); err != nil {
		t.Fatalf("SAN y huella permitidos deben autenticar: %v", err)
	}
	configuracionConPin.HuellasProxyTLSPermitidas = []string{"sha256:" + strings.Repeat("0", 64)}
	servicioConPinIncorrecto := debeServicio(t, configuracionConPin, &verificadorFalso{}, evaluadorValido(dominiovec.AuthAssuranceHigh), nuevoRegistroMemoria(), &relojFijo{ahora: time.Now()})
	if _, err := servicioConPinIncorrecto.AutenticarCanalTLSMutuo(estado); !errors.Is(err, ErrCanalProxyNoAutenticado) {
		t.Fatalf("una huella incorrecta no se compensa con SAN correcto: %v", err)
	}
}

func TestCanalYSesionQuedanLigadosAUnaInstancia(t *testing.T) {
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	configuracion := configuracionInternaValida()
	v1 := &verificadorFalso{}
	v2 := &verificadorFalso{}
	s1 := debeServicio(t, configuracion, v1, evaluadorValido(dominiovec.AuthAssuranceHigh), nuevoRegistroMemoria(), &relojFijo{ahora: ahora})
	s2 := debeServicio(t, configuracion, v2, evaluadorValido(dominiovec.AuthAssuranceHigh), nuevoRegistroMemoria(), &relojFijo{ahora: ahora})
	canal1 := debeCanalTLS(t, s1, configuracion)
	asercion := asercionInternaValida(ahora, configuracion, canal1)
	v1.fijarAsercion(asercion)
	v2.fijarAsercion(asercion)
	credencial := debeCredencial(t, []byte("opaca"), canal1)
	identidad, err := s1.Resolver(context.Background(), credencial)
	if err != nil {
		t.Fatalf("resolver con instancia emisora: %v", err)
	}
	if _, err := s2.Resolver(context.Background(), credencial); !errors.Is(err, ErrCanalProxyNoAutenticado) || v2.llamadas.Load() != 0 {
		t.Fatalf("otra instancia no debe usar el canal: %v", err)
	}
	if _, _, err := s2.ProyectarCuentaAutenticada(context.Background(), identidad); !errors.Is(err, ErrSesionNoValida) {
		t.Fatalf("otra instancia no debe proyectar la sesion: %v", err)
	}
	if _, _, err := s1.ProyectarCuentaAutenticada(context.Background(), IdentidadSesion{}); !errors.Is(err, ErrSesionNoValida) {
		t.Fatalf("el valor cero opaco debe fallar cerrado: %v", err)
	}
	copiaServicio := *s1
	if _, err := copiaServicio.Resolver(context.Background(), credencial); !errors.Is(err, ErrCanalProxyNoAutenticado) {
		t.Fatalf("una copia estructural no es la instancia que autentico el canal: %v", err)
	}
	if _, _, err := copiaServicio.ProyectarCuentaAutenticada(context.Background(), identidad); !errors.Is(err, ErrSesionNoValida) {
		t.Fatalf("una copia estructural no debe proyectar la sesion: %v", err)
	}
}

func TestServicioIdentidadFallaCerradoAnteAsercionesInvalidas(t *testing.T) {
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	configuracion := configuracionInternaValida()
	verificador := &verificadorFalso{}
	servicio := debeServicio(t, configuracion, verificador, evaluadorValido(dominiovec.AuthAssuranceHigh), nuevoRegistroMemoria(), &relojFijo{ahora: ahora})
	canal := debeCanalTLS(t, servicio, configuracion)
	base := asercionInternaValida(ahora, configuracion, canal)
	credencial := debeCredencial(t, []byte("opaca"), canal)

	pruebas := []struct {
		nombre    string
		modificar func(*AsercionProxyIdentidad)
	}{
		{"id ausente", func(a *AsercionProxyIdentidad) { a.ID = "" }},
		{"id con control", func(a *AsercionProxyIdentidad) { a.ID = "id\ninyectado" }},
		{"id enorme", func(a *AsercionProxyIdentidad) { a.ID = strings.Repeat("a", longitudMaximaID+1) }},
		{"emisor distinto", func(a *AsercionProxyIdentidad) { a.Emisor = "otro" }},
		{"audiencia distinta", func(a *AsercionProxyIdentidad) { a.Audiencia = "otra" }},
		{"superficie distinta", func(a *AsercionProxyIdentidad) { a.Superficie = SuperficieExternaPersonal }},
		{"sujeto ausente", func(a *AsercionProxyIdentidad) { a.SujetoID = "" }},
		{"cuenta de otro sujeto", func(a *AsercionProxyIdentidad) { a.Cuenta.SujetoVinculadoID = "persona-otra" }},
		{"sesion ausente", func(a *AsercionProxyIdentidad) { a.SesionID = "" }},
		{"acr ausente", func(a *AsercionProxyIdentidad) { a.ACRVerificado = "" }},
		{"metodo primario ausente", func(a *AsercionProxyIdentidad) { a.MetodoPrimario = "" }},
		{"metodo primario sin factor", func(a *AsercionProxyIdentidad) { a.MetodoPrimario = MetodoDNIe }},
		{"canal no vinculado", func(a *AsercionProxyIdentidad) { a.CanalVinculadoRef = "tls:otro" }},
		{"autenticacion verificada ausente", func(a *AsercionProxyIdentidad) { a.AutenticacionVerificadaEn = time.Time{} }},
		{"autenticacion posterior a sesion", func(a *AsercionProxyIdentidad) {
			a.AutenticacionVerificadaEn = a.EmitidaEn.Add(time.Microsecond)
		}},
		{"autenticacion demasiado antigua", func(a *AsercionProxyIdentidad) {
			a.AutenticacionVerificadaEn = ahora.Add(-configuracion.EdadMaximaAutenticacion)
		}},
		{"autenticacion fuera de UTC canonico", func(a *AsercionProxyIdentidad) {
			a.AutenticacionVerificadaEn = a.AutenticacionVerificadaEn.In(time.FixedZone("UTC-no-canonico", 0))
		}},
		{"autenticacion sin precision PostgreSQL", func(a *AsercionProxyIdentidad) {
			a.AutenticacionVerificadaEn = a.AutenticacionVerificadaEn.Add(time.Nanosecond)
		}},
		{"sesion emitida sin precision PostgreSQL", func(a *AsercionProxyIdentidad) {
			a.EmitidaEn = a.EmitidaEn.Add(time.Nanosecond)
		}},
		{"emitida en futuro", func(a *AsercionProxyIdentidad) { a.EmitidaEn = ahora.Add(time.Minute) }},
		{"aun no vigente", func(a *AsercionProxyIdentidad) { a.NoAntesDe = ahora.Add(time.Minute) }},
		{"caducada", func(a *AsercionProxyIdentidad) { a.ExpiraEn = ahora }},
		{"duracion excesiva", func(a *AsercionProxyIdentidad) {
			a.ExpiraEn = a.EmitidaEn.Add(configuracion.DuracionMaximaAsercion + time.Second)
		}},
		{"falta kerberos", func(a *AsercionProxyIdentidad) { a.Factores = a.Factores[1:] }},
		{"falta certificado", func(a *AsercionProxyIdentidad) { a.Factores = a.Factores[:1] }},
		{"principal kerberos ausente", func(a *AsercionProxyIdentidad) { a.Factores[0].Principal = "" }},
		{"certificado sin referencia", func(a *AsercionProxyIdentidad) { a.Factores[1].CredencialRef = "" }},
		{"evidencia repetida", func(a *AsercionProxyIdentidad) { a.Factores[1].EvidenciaRef = a.Factores[0].EvidenciaRef }},
		{"mismo grupo criptografico PKINIT", func(a *AsercionProxyIdentidad) {
			a.Factores[0].GrupoCriptograficoRef = a.Factores[1].GrupoCriptograficoRef
		}},
		{"mismo grupo con caja distinta", func(a *AsercionProxyIdentidad) {
			a.Factores[0].GrupoCriptograficoRef = strings.ToUpper(a.Factores[1].GrupoCriptograficoRef)
		}},
		{"factor previo", func(a *AsercionProxyIdentidad) {
			a.Factores[0].VerificadoEn = a.EmitidaEn.Add(-configuracion.ToleranciaReloj - time.Second)
		}},
		{"factor de otro sujeto", func(a *AsercionProxyIdentidad) { a.Factores[0].SujetoVinculadoID = "persona-otra" }},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			invalida := copiarAsercion(base)
			prueba.modificar(&invalida)
			verificador.fijarAsercion(invalida)
			if _, err := servicio.Resolver(context.Background(), credencial); !errors.Is(err, ErrAsercionNoValida) {
				t.Fatalf("se esperaba asercion invalida, recibido %v", err)
			}
		})
	}
}

func TestPuertosExternosFallanCerradosYSaneanErrores(t *testing.T) {
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	configuracion := configuracionInternaValida()
	verificador := &verificadorFalso{}
	evaluador := evaluadorValido(dominiovec.AuthAssuranceHigh)
	registro := nuevoRegistroMemoria()
	servicio := debeServicio(t, configuracion, verificador, evaluador, registro, &relojFijo{ahora: ahora})
	canal := debeCanalTLS(t, servicio, configuracion)
	verificador.fijarAsercion(asercionInternaValida(ahora, configuracion, canal))
	credencial := debeCredencial(t, []byte("opaca"), canal)
	secreto := "detalle-secreto-del-proveedor"

	verificador.fijarError(errors.New(secreto))
	if _, err := servicio.Resolver(context.Background(), credencial); !errors.Is(err, ErrAsercionNoValida) || strings.Contains(err.Error(), secreto) {
		t.Fatalf("error del verificador filtrado o aceptado: %v", err)
	}
	verificador.fijarError(nil)
	evaluador.fijarError(errors.New(secreto))
	if _, err := servicio.Resolver(context.Background(), credencial); !errors.Is(err, ErrAsercionNoValida) || strings.Contains(err.Error(), secreto) {
		t.Fatalf("error del evaluador filtrado o aceptado: %v", err)
	}
	evaluador.fijarError(nil)
	registro.inactivar("cuenta-tecnica")
	if _, err := servicio.Resolver(context.Background(), credencial); !errors.Is(err, ErrSesionNoValida) || strings.Contains(err.Error(), "inactiva") {
		t.Fatalf("error del registro filtrado o aceptado: %v", err)
	}

	contexto, cancelar := context.WithCancel(context.Background())
	cancelar()
	llamadasAntes := verificador.llamadas.Load()
	if _, err := servicio.Resolver(contexto, credencial); !errors.Is(err, context.Canceled) || verificador.llamadas.Load() != llamadasAntes {
		t.Fatalf("un contexto cancelado debe parar antes del proveedor: %v", err)
	}
}

func TestGarantiaSeCalculaYSeLigaALaPolitica(t *testing.T) {
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	configuracion := configuracionInternaValida()
	verificador := &verificadorFalso{}
	evaluador := evaluadorValido(dominiovec.AuthAssuranceSubstantial)
	servicio := debeServicio(t, configuracion, verificador, evaluador, nuevoRegistroMemoria(), &relojFijo{ahora: ahora})
	canal := debeCanalTLS(t, servicio, configuracion)
	verificador.fijarAsercion(asercionInternaValida(ahora, configuracion, canal))
	if _, err := servicio.Resolver(context.Background(), debeCredencial(t, []byte("opaca"), canal)); !errors.Is(err, ErrAsercionNoValida) {
		t.Fatalf("la garantia calculada insuficiente debe denegarse: %v", err)
	}

	for _, resultado := range []ResultadoEvaluacionGarantia{
		{Garantia: dominiovec.AuthAssuranceHigh, PoliticaRef: "politica-garantia-v1", HuellaPolitica: "sha256:" + strings.Repeat("a", 64)},
		{Garantia: dominiovec.AuthAssuranceHigh, PoliticaRef: "pga_corta", HuellaPolitica: "sha256:" + strings.Repeat("a", 64)},
		{Garantia: dominiovec.AuthAssuranceHigh, PoliticaRef: "pga_0123456789abcdefghijkl", HuellaPolitica: "SHA256:" + strings.Repeat("A", 64)},
		{Garantia: dominiovec.AuthAssuranceHigh, PoliticaRef: "pga_0123456789abcdefghijkl", HuellaPolitica: strings.Repeat("a", 64)},
	} {
		evaluador.fijarResultado(resultado)
		if _, err := servicio.Resolver(context.Background(), debeCredencial(t, []byte("politica-no-canonica"), canal)); !errors.Is(err, ErrAsercionNoValida) {
			t.Fatalf("politica no canonica aceptada (%#v): %v", resultado, err)
		}
	}

	evaluador.fijarResultado(resultadoGarantia(dominiovec.AuthAssuranceHigh))
	identidad, err := servicio.Resolver(context.Background(), debeCredencial(t, []byte("otra-opaca"), canal))
	if err != nil {
		t.Fatalf("garantia calculada valida: %v", err)
	}
	evaluador.fijarResultado(ResultadoEvaluacionGarantia{
		Garantia: dominiovec.AuthAssuranceHigh, PoliticaRef: "pga_otra23456789abcdefghijkl",
		HuellaPolitica: "sha256:" + strings.Repeat("b", 64),
	})
	if _, _, err := servicio.ProyectarCuentaAutenticada(context.Background(), identidad); !errors.Is(err, ErrSesionNoValida) {
		t.Fatalf("un cambio de politica obliga a reautenticar: %v", err)
	}
}

func TestAreaPersonalMantieneConectoresAlternativosIntercambiables(t *testing.T) {
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	configuracion := configuracionPersonalValida()
	verificador := &verificadorFalso{}
	servicio := debeServicio(t, configuracion, verificador, evaluadorValido(dominiovec.AuthAssuranceSubstantial), nuevoRegistroMemoria(), &relojFijo{ahora: ahora})
	canal := debeCanalTLS(t, servicio, configuracion)
	tipos := []struct {
		factor   FactorAutenticacion
		esperado dominiovec.AuthMethod
	}{
		{FactorAutenticacion{Metodo: MetodoClave, EvidenciaRef: "clave:evidencia", GrupoCriptograficoRef: "grupo:clave"}, dominiovec.AuthMethodClave},
		{FactorAutenticacion{Metodo: MetodoCertificado, CredencialRef: "cert:001", EvidenciaRef: "cert:evidencia", GrupoCriptograficoRef: "grupo:cert"}, dominiovec.AuthMethodCertificate},
		{FactorAutenticacion{Metodo: MetodoDNIe, CredencialRef: "dnie:001", EvidenciaRef: "dnie:evidencia", GrupoCriptograficoRef: "grupo:dnie"}, dominiovec.AuthMethodDNIe},
	}
	for indice, tipo := range tipos {
		emitida := ahora.Add(-time.Minute)
		factor := tipo.factor
		factor.SujetoVinculadoID = "persona-001"
		factor.VerificadoEn = emitida
		asercion := AsercionProxyIdentidad{
			ID: fmt.Sprintf("asercion-personal-%d", indice), Emisor: configuracion.EmisorIdentidad,
			Audiencia: configuracion.Audiencia, Superficie: configuracion.Superficie, SujetoID: "persona-001",
			Cuenta:   CuentaAcceso{ID: "persona.cuenta", SujetoVinculadoID: "persona-001"},
			SesionID: fmt.Sprintf("sesion-personal-%d", indice), CanalVinculadoRef: canal.ReferenciaVinculacion(),
			AutenticacionVerificadaEn: emitida,
			EmitidaEn:                 emitida, NoAntesDe: emitida, ExpiraEn: ahora.Add(time.Minute),
			MetodoPrimario: factor.Metodo, ACRVerificado: "urn:vec:acr:sustancial", Factores: []FactorAutenticacion{factor},
		}
		verificador.fijarAsercion(asercion)
		identidad, err := servicio.Resolver(context.Background(), debeCredencial(t, []byte(fmt.Sprintf("opaca-%d", indice)), canal))
		if err != nil {
			t.Fatalf("resolver con %s: %v", factor.Metodo, err)
		}
		cuenta, _, err := servicio.ProyectarCuentaAutenticada(context.Background(), identidad)
		if err != nil || cuenta.Metodo != tipo.esperado {
			t.Fatalf("proyectar con %s: %#v %v", factor.Metodo, cuenta, err)
		}
	}
}

func TestAdministracionComparaCuentasCanonicalizadas(t *testing.T) {
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	configuracion := configuracionAdministracionValida()
	verificador := &verificadorFalso{}
	registro := nuevoRegistroMemoria()
	servicio := debeServicio(t, configuracion, verificador, evaluadorValido(dominiovec.AuthAssuranceHigh), registro, &relojFijo{ahora: ahora})
	canal := debeCanalTLS(t, servicio, configuracion)
	base := asercionInternaValida(ahora, configuracion, canal)
	base.Cuenta = CuentaAcceso{ID: " ADM-Cuenta-Tecnica ", SujetoVinculadoID: base.SujetoID, CuentaOrdinariaID: " Cuenta-Tecnica ", Privilegiada: true}
	verificador.fijarAsercion(base)
	identidad, err := servicio.Resolver(context.Background(), debeCredencial(t, []byte("opaca"), canal))
	if err != nil {
		t.Fatalf("cuenta administrativa nominativa: %v", err)
	}
	cuenta, auditoria, err := servicio.ProyectarCuentaAutenticada(context.Background(), identidad)
	if err != nil || cuenta.CuentaRef != identidad.confirmacion.CuentaRef ||
		identidad.estado.cuenta.CuentaOrdinariaID != "cuenta-tecnica" || !auditoria.CuentaPrivilegiada() {
		t.Fatalf("cuentas no canonicalizadas: %#v %#v %v", cuenta, auditoria, err)
	}
	registro.inactivar("cuenta-tecnica")
	if _, _, err := servicio.ProyectarCuentaAutenticada(context.Background(), identidad); !errors.Is(err, ErrSesionNoValida) {
		t.Fatalf("la cuenta ordinaria inactiva debe revocar la privilegiada: %v", err)
	}

	invalida := copiarAsercion(base)
	invalida.ID = "asercion-otra"
	invalida.SesionID = "sesion-otra"
	invalida.Cuenta = CuentaAcceso{ID: " ADM ", SujetoVinculadoID: base.SujetoID, CuentaOrdinariaID: "adm", Privilegiada: true}
	verificador.fijarAsercion(invalida)
	if _, err := servicio.Resolver(context.Background(), debeCredencial(t, []byte("otra"), canal)); !errors.Is(err, ErrAsercionNoValida) {
		t.Fatalf("la misma cuenta con espacios/caja debe denegarse: %v", err)
	}
}

func TestRegistroConsumeAsercionAtomicamenteEnCarrera(t *testing.T) {
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	configuracion := configuracionInternaValida()
	verificador := &verificadorFalso{}
	registro := nuevoRegistroMemoria()
	servicio := debeServicio(t, configuracion, verificador, evaluadorValido(dominiovec.AuthAssuranceHigh), registro, &relojFijo{ahora: ahora})
	canal := debeCanalTLS(t, servicio, configuracion)
	verificador.fijarAsercion(asercionInternaValida(ahora, configuracion, canal))
	credencial := debeCredencial(t, []byte("opaca"), canal)

	const intentos = 32
	var exitos atomic.Int64
	var fallosInvalidos atomic.Int64
	var grupo sync.WaitGroup
	grupo.Add(intentos)
	for i := 0; i < intentos; i++ {
		go func() {
			defer grupo.Done()
			if _, err := servicio.Resolver(context.Background(), credencial); err == nil {
				exitos.Add(1)
			} else if errors.Is(err, ErrSesionNoValida) {
				fallosInvalidos.Add(1)
			}
		}()
	}
	grupo.Wait()
	if exitos.Load() != 1 || fallosInvalidos.Load() != intentos-1 {
		t.Fatalf("consumo no atomico: exitos=%d fallos=%d", exitos.Load(), fallosInvalidos.Load())
	}
}

func TestCredencialPrivadaRedactaSerializadores(t *testing.T) {
	configuracion := configuracionInternaValida()
	servicio := debeServicio(t, configuracion, &verificadorFalso{}, evaluadorValido(dominiovec.AuthAssuranceHigh), nuevoRegistroMemoria(), &relojFijo{ahora: time.Now()})
	canal := debeCanalTLS(t, servicio, configuracion)
	secreto := "asercion-super-secreta"
	credencial := debeCredencial(t, []byte(secreto), canal)
	representaciones := []string{
		fmt.Sprintf("%s %v %#v", credencial, credencial, credencial),
		string(debeJSON(t, credencial)),
		string(debeTexto(t, credencial)),
		string(debeBinario(t, credencial)),
	}
	var registro bytes.Buffer
	slog.New(slog.NewTextHandler(&registro, nil)).Info("prueba", "credencial", credencial)
	representaciones = append(representaciones, registro.String())
	var serializacionGob bytes.Buffer
	if err := gob.NewEncoder(&serializacionGob).Encode(credencial); err != nil {
		t.Fatalf("serializar gob redactado: %v", err)
	}
	representaciones = append(representaciones, serializacionGob.String())
	for _, representacion := range representaciones {
		if strings.Contains(representacion, secreto) {
			t.Fatalf("secreto filtrado: %q", representacion)
		}
	}
	var reconstruida CredencialProxy
	if err := gob.NewDecoder(bytes.NewReader(serializacionGob.Bytes())).Decode(&reconstruida); !errors.Is(err, ErrCredencialNoSerializable) {
		t.Fatalf("gob no debe reconstruir una credencial utilizable: %v", err)
	}
}

func TestServicioRequiereTodosLosPuertosYProtegeConfiguracion(t *testing.T) {
	configuracion := configuracionInternaValida()
	verificador := &verificadorFalso{}
	evaluador := evaluadorValido(dominiovec.AuthAssuranceHigh)
	registro := nuevoRegistroMemoria()
	if _, err := NuevoServicioIdentidad(configuracionPublicaValida(), verificador, evaluador, registro, nil); !errors.Is(err, ErrConfiguracionSuperficie) {
		t.Fatalf("anonimo no crea sesiones: %v", err)
	}
	if _, err := NuevoServicioIdentidad(configuracion, nil, evaluador, registro, nil); !errors.Is(err, ErrVerificadorAusente) {
		t.Fatalf("verificador obligatorio: %v", err)
	}
	if _, err := NuevoServicioIdentidad(configuracion, verificador, nil, registro, nil); !errors.Is(err, ErrEvaluadorGarantiaAusente) {
		t.Fatalf("evaluador obligatorio: %v", err)
	}
	if _, err := NuevoServicioIdentidad(configuracion, verificador, evaluador, nil, nil); !errors.Is(err, ErrRegistroSesionesAusente) {
		t.Fatalf("registro obligatorio: %v", err)
	}

	servicio := debeServicio(t, configuracion, verificador, evaluador, registro, &relojFijo{ahora: time.Now()})
	configuracion.MetodosAdmitidos[0] = MetodoSSO
	configuracion.FactoresRequeridos[0] = MetodoSSO
	configuracion.HuellasProxyTLSPermitidas = []string{"sha256:" + strings.Repeat("0", 64)}
	if _, err := servicio.AutenticarCanalTLSMutuo(estadoTLSMutuoReal(t, sanDNSConfigurado(configuracionInternaValida()))); err != nil {
		t.Fatalf("la mutacion externa no debe alterar la confianza del servicio: %v", err)
	}
}

func TestConstructorCredencialLimitaYCopia(t *testing.T) {
	if _, err := NuevaCredencialProxy(nil, CanalProxyAutenticado{}); !errors.Is(err, ErrAsercionAusente) {
		t.Fatalf("ausencia no rechazada: %v", err)
	}
	if _, err := NuevaCredencialProxy(make([]byte, longitudMaximaAsercionProtegida+1), CanalProxyAutenticado{}); !errors.Is(err, ErrAsercionDemasiadoGrande) {
		t.Fatalf("tamano no limitado: %v", err)
	}
	original := []byte("secreto")
	credencial, err := NuevaCredencialProxy(original, CanalProxyAutenticado{})
	if err != nil {
		t.Fatalf("crear: %v", err)
	}
	original[0] = 'X'
	if string(credencial.asercionProtegida) != "secreto" {
		t.Fatal("la entrada no fue copiada")
	}
}

func debeServicio(
	t *testing.T,
	configuracion ConfiguracionSuperficie,
	verificador VerificadorAsercionProtegida,
	evaluador EvaluadorGarantia,
	registro RegistroSesiones,
	reloj Reloj,
) *ServicioIdentidad {
	t.Helper()
	servicio, err := NuevoServicioIdentidad(configuracion, verificador, evaluador, registro, reloj)
	if err != nil {
		t.Fatalf("crear servicio: %v", err)
	}
	return servicio
}

func debeCanalTLS(t *testing.T, servicio *ServicioIdentidad, configuracion ConfiguracionSuperficie) CanalProxyAutenticado {
	t.Helper()
	canal, err := servicio.AutenticarCanalTLSMutuo(estadoTLSMutuoReal(t, sanDNSConfigurado(configuracion)))
	if err != nil {
		t.Fatalf("autenticar canal mTLS: %v", err)
	}
	return canal
}

func debeCredencial(t *testing.T, protegida []byte, canal CanalProxyAutenticado) CredencialProxy {
	t.Helper()
	credencial, err := NuevaCredencialProxy(protegida, canal)
	if err != nil {
		t.Fatalf("crear credencial: %v", err)
	}
	return credencial
}

func evaluadorValido(garantia dominiovec.AuthAssurance) *evaluadorFalso {
	return &evaluadorFalso{resultado: resultadoGarantia(garantia)}
}

func resultadoGarantia(garantia dominiovec.AuthAssurance) ResultadoEvaluacionGarantia {
	return ResultadoEvaluacionGarantia{
		Garantia: garantia, PoliticaRef: "pga_0123456789abcdefghijkl",
		HuellaPolitica: "sha256:" + strings.Repeat("a", 64),
	}
}

func asercionInternaValida(ahora time.Time, c ConfiguracionSuperficie, canal CanalProxyAutenticado) AsercionProxyIdentidad {
	emitida := ahora.Add(-time.Minute)
	return AsercionProxyIdentidad{
		ID: "asercion-001", Emisor: c.EmisorIdentidad, Audiencia: c.Audiencia, Superficie: c.Superficie,
		SujetoID: "persona-001", Cuenta: CuentaAcceso{ID: "Cuenta-Tecnica", SujetoVinculadoID: "persona-001"},
		SesionID: "sesion-001", CanalVinculadoRef: canal.ReferenciaVinculacion(),
		AutenticacionVerificadaEn: emitida, EmitidaEn: emitida,
		NoAntesDe: emitida, ExpiraEn: ahora.Add(time.Minute), MetodoPrimario: MetodoCertificado,
		ACRVerificado: "urn:vec:acr:alto",
		Factores: []FactorAutenticacion{
			{
				Metodo: MetodoKerberos, SujetoVinculadoID: "persona-001", Principal: "cuenta-tecnica@CORPORATIVA.EXAMPLE",
				EvidenciaRef: "krb:ticket:001", GrupoCriptograficoRef: "grupo:contrasena-corporativa", VerificadoEn: emitida,
			},
			{
				Metodo: MetodoCertificado, SujetoVinculadoID: "persona-001", CredencialRef: "cert:sha256:001",
				EvidenciaRef: "cert:validacion:001", GrupoCriptograficoRef: "grupo:tarjeta-001", VerificadoEn: emitida,
			},
		},
	}
}

func copiarAsercion(a AsercionProxyIdentidad) AsercionProxyIdentidad {
	a.Factores = append([]FactorAutenticacion(nil), a.Factores...)
	return a
}

func debeJSON(t *testing.T, valor any) []byte {
	t.Helper()
	contenido, err := json.Marshal(valor)
	if err != nil {
		t.Fatalf("serializar JSON: %v", err)
	}
	return contenido
}

func debeTexto(t *testing.T, valor interface{ MarshalText() ([]byte, error) }) []byte {
	t.Helper()
	contenido, err := valor.MarshalText()
	if err != nil {
		t.Fatalf("serializar texto: %v", err)
	}
	return contenido
}

func debeBinario(t *testing.T, valor interface{ MarshalBinary() ([]byte, error) }) []byte {
	t.Helper()
	contenido, err := valor.MarshalBinary()
	if err != nil {
		t.Fatalf("serializar binario: %v", err)
	}
	return contenido
}

func sanDNSConfigurado(configuracion ConfiguracionSuperficie) string {
	return strings.TrimPrefix(configuracion.IdentidadesSANProxyPermitidas[0], "dns:")
}

// estadoTLSMutuoReal obtiene ConnectionState de un handshake real sobre
// net.Pipe. De este modo la prueba no inventa VerifiedChains manualmente.
func estadoTLSMutuoReal(t *testing.T, sanCliente string) tls.ConnectionState {
	t.Helper()
	ahora := time.Now()
	_, claveCA, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("clave CA: %v", err)
	}
	plantillaCA := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "CA prueba mTLS"},
		NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour), IsCA: true, BasicConstraintsValid: true,
		KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	}
	derCA, err := x509.CreateCertificate(rand.Reader, plantillaCA, plantillaCA, claveCA.Public(), claveCA)
	if err != nil {
		t.Fatalf("crear CA: %v", err)
	}
	certificadoCA, err := x509.ParseCertificate(derCA)
	if err != nil {
		t.Fatalf("parsear CA: %v", err)
	}

	crearCertificado := func(serial int64, nombre string, dns []string, usos []x509.ExtKeyUsage) tls.Certificate {
		t.Helper()
		publica, privada, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("clave: %v", err)
		}
		plantilla := &x509.Certificate{
			SerialNumber: big.NewInt(serial), Subject: pkix.Name{CommonName: nombre}, DNSNames: dns,
			NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour),
			KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: usos,
		}
		der, err := x509.CreateCertificate(rand.Reader, plantilla, certificadoCA, publica, claveCA)
		if err != nil {
			t.Fatalf("crear certificado: %v", err)
		}
		return tls.Certificate{Certificate: [][]byte{der, derCA}, PrivateKey: privada}
	}
	servidorCert := crearCertificado(2, "servidor.test", []string{"servidor.test"}, []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth})
	clienteCert := crearCertificado(3, sanCliente, []string{sanCliente}, []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth})
	raices := x509.NewCertPool()
	raices.AddCert(certificadoCA)
	servidorConfig := &tls.Config{
		Certificates: []tls.Certificate{servidorCert}, ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs: raices, MinVersion: tls.VersionTLS12, MaxVersion: tls.VersionTLS13,
	}
	clienteConfig := &tls.Config{
		Certificates: []tls.Certificate{clienteCert}, RootCAs: raices, ServerName: "servidor.test",
		MinVersion: tls.VersionTLS12, MaxVersion: tls.VersionTLS13,
	}
	parServidor, parCliente := net.Pipe()
	servidor := tls.Server(parServidor, servidorConfig)
	cliente := tls.Client(parCliente, clienteConfig)
	errores := make(chan error, 2)
	go func() { errores <- servidor.Handshake() }()
	go func() { errores <- cliente.Handshake() }()
	for i := 0; i < 2; i++ {
		if err := <-errores; err != nil {
			_ = parServidor.Close()
			_ = parCliente.Close()
			t.Fatalf("handshake mTLS: %v", err)
		}
	}
	estado := servidor.ConnectionState()
	_ = parServidor.Close()
	_ = parCliente.Close()
	return estado
}
