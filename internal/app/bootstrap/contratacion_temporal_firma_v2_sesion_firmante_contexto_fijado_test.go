package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

type sesionContextoFijadoPrueba struct {
	mu       sync.Mutex
	actual   ports.EvidenciaSesionFirmanteV2
	llamadas int
}

func (s *sesionContextoFijadoPrueba) RevalidarSesionFirmanteV2(context.Context) (ports.EvidenciaSesionFirmanteV2, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.llamadas++
	e := s.actual
	clon, err := e.Resultado.Clonar()
	e.Resultado = clon
	return e, err
}

type instantaneaContextoFijadoPrueba struct {
	mu     sync.Mutex
	actual core.InstantaneaAutorizacion
}

func (s *instantaneaContextoFijadoPrueba) ObtenerInstantaneaAutorizacion(context.Context, string, string) (core.InstantaneaAutorizacion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.actual, nil
}

type escenarioCapsulaContextoFijado struct {
	autoridad *autoridadSesionFirmanteV2
	capsula   *capsulaSesionFirmanteV2
	peticion  *http.Request
	sesion    *sesionContextoFijadoPrueba
	perfiles  *instantaneaContextoFijadoPrueba
	primera   ports.EvidenciaSesionFirmanteV2
}

func nuevaCapsulaContextoFijadoPrueba(t *testing.T, desfase time.Duration) escenarioCapsulaContextoFijado {
	t.Helper()
	soporte, _, _ := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	semilla := resultadoConVinculoEmpleadoF1(t, soporte.contexto.Resultado)
	semilla.Contexto.Instantanea.Vinculos[0].VinculoRef = referenciaAltaContratacionTemporalDesarrollo("pep_", "f1-empleado-proyectado")
	manifiesto, err := core.RehidratarManifiestoProcedenciaContextoActorV1(semilla.ManifiestoProcedenciaCanonico)
	if err != nil {
		t.Fatal(err)
	}
	manifiesto.Vinculos[0].VinculoRef = semilla.Contexto.Instantanea.Vinculos[0].VinculoRef
	semilla.ManifiestoProcedenciaCanonico, err = manifiesto.RepresentacionCanonicaV1()
	if err != nil {
		t.Fatal(err)
	}
	semilla.ManifiestoProcedenciaHuellaSHA256, err = core.HuellaSHA256ManifiestoProcedenciaContextoActorV1(semilla.ManifiestoProcedenciaCanonico)
	if err != nil {
		t.Fatal(err)
	}
	actor, err := core.NuevoContextoActor(core.CuentaAutenticadaContextoActor{
		CuentaRef: semilla.Contexto.Instantanea.CuentaRef,
		Metodo:    core.AuthMethodCertificate, Garantia: core.AuthAssuranceSubstantial,
	}, semilla.Contexto.Instantanea, semilla.ResueltoEnAutoritativo)
	if err != nil {
		t.Fatal(err)
	}
	semilla.Contexto = actor
	semilla.RepresentacionCanonica, err = actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	semilla.HuellaSHA256, err = actor.HuellaSHA256VinculadaV2()
	if err != nil || semilla.Validar() != nil {
		t.Fatalf("contexto empleado sustancial inválido: %v", err)
	}
	datos, err := soporte.contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	autenticacion := datos.Autenticacion()
	autenticacion.GarantiaObservada = core.AuthAssuranceSubstantial
	reloj := &relojSesionConsultaPrueba{ahora: semilla.ResueltoEnAutoritativo.Add(time.Second + desfase)}
	resolutor := &resolutorSesionConsultaPrueba{base: semilla, reloj: reloj}
	solicitud := core.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: datos.AutenticacionRef, SesionRef: datos.SesionRef}
	cuenta := core.CuentaAutenticadaContextoActor{CuentaRef: datos.CuentaRef,
		Metodo: core.AuthMethodCertificate, Garantia: core.AuthAssuranceSubstantial}
	resolver := func() (core.VinculoAutenticacionActorV2, core.ResultadoContextoActorRegistradoV2) {
		v, r, err := core.CrearVinculoAutenticacionActorV2ConResultado(context.Background(),
			revalidadorNominalFijoPrueba{actual: autenticacion}, solicitud, resolutor,
			core.SolicitudContextoActor{Cuenta: cuenta, PerfilActivoRef: datos.PerfilActivoRef}, reloj)
		if err != nil {
			t.Fatal(err)
		}
		return v, r
	}
	v1, r1 := resolver()
	reloj.ahora = reloj.Ahora().Add(time.Second)
	v2, r2 := resolver()
	if bytes.Equal(r1.RepresentacionCanonica, r2.RepresentacionCanonica) ||
		!r2.Contexto.AlcanceProyecciones().IncluyeEmpleado() {
		t.Fatal("la fuente no generó una segunda observación F1 del empleado")
	}
	hoja := &x509.Certificate{Raw: []byte("certificado-sintetico-firma-fijada"),
		NotBefore: reloj.Ahora().Add(-time.Hour), NotAfter: reloj.Ahora().Add(time.Hour)}
	raw := httptest.NewRequest(http.MethodPost, httpinterno.RutaRegistroFirmaVec, nil)
	raw.TLS = &tls.ConnectionState{HandshakeComplete: true, Version: tls.VersionTLS13,
		PeerCertificates: []*x509.Certificate{hoja}, VerifiedChains: [][]*x509.Certificate{{hoja, {Raw: []byte("ca-sintetica")}}}}
	certificado := sha256.Sum256(hoja.Raw)
	huella := hex.EncodeToString(certificado[:])
	hasta := hoja.NotAfter.UTC().Truncate(time.Microsecond)
	primera := ports.EvidenciaSesionFirmanteV2{Vinculo: v1, Resultado: r1, CertificadoCanalSHA256: huella, CertificadoValidoHasta: hasta}
	sesion := &sesionContextoFijadoPrueba{actual: ports.EvidenciaSesionFirmanteV2{
		Vinculo: v2, Resultado: r2, CertificadoCanalSHA256: huella, CertificadoValidoHasta: hasta}}
	perfiles := &instantaneaContextoFijadoPrueba{actual: soporte.instantanea}
	fuente := &fuenteSinteticaSesionFirmanteV2{}
	autoridad := &autoridadSesionFirmanteV2{fuente: fuente, reloj: reloj,
		preacreditar: func(*http.Request) error { return nil }, garantia: core.AuthAssuranceSubstantial}
	nominal := &fuenteNominalSesionFirmaCT{autoridad: autoridad, fuente: fuente, asignaciones: perfiles, reloj: reloj}
	p := &peticionNominalFirmaCT{fuente: nominal, sesion: sesion, peticion: raw,
		contextos: make(map[string]contextoNominalFirmaCT, 1)}
	p.solicitud = ports.SolicitudSesionFirmanteV2{CertificadoCanalSHA256: huella,
		CertificadoTLSValidoHasta: hasta, PersonaEsperadaRef: datos.PrincipalID,
		CuentaEsperadaRef: datos.CuentaRef, PerfilEsperadoRef: datos.PerfilActivoRef,
		RolEsperadoID: soporte.instantanea.VersionRol.RolID}
	if _, err := p.conservarContextoNominalFirmaCT(primera, soporte.instantanea, reloj.Ahora()); err != nil {
		t.Fatal(err)
	}
	peticion := raw.WithContext(context.WithValue(raw.Context(), clavePeticionNominalFirmaCT{}, p))
	capsula := &capsulaSesionFirmanteV2{autoridad: autoridad, peticion: peticion, sesion: sesion,
		personaCA25: datos.PrincipalID, huellaAUT56: huella, validoHasta: hasta,
		seleccion: ports.SeleccionFirmanteV2{PersonaRef: datos.PrincipalID, CuentaRef: datos.CuentaRef,
			PerfilActivoRef: datos.PerfilActivoRef, RolID: soporte.instantanea.VersionRol.RolID}}
	capsula.consumida.Store(true)
	return escenarioCapsulaContextoFijado{autoridad, capsula, peticion, sesion, perfiles, primera}
}

func TestFirmaVecRevalidacionesConservanElMismoContextoAtestado(t *testing.T) {
	e := nuevaCapsulaContextoFijadoPrueba(t, 0)
	var primero, segundo core.ResultadoContextoActorRegistradoV2
	for i := 0; i < 2; i++ {
		v, r, err := e.autoridad.revalidarConContexto(e.peticion.Context(), e.peticion, e.capsula)
		if err != nil || v.ValidarPara(r) != nil {
			t.Fatalf("emisión %d: %v", i+1, err)
		}
		if i == 0 {
			primero = r
		} else {
			segundo = r
		}
	}
	if !bytes.Equal(primero.RepresentacionCanonica, segundo.RepresentacionCanonica) ||
		!bytes.Equal(primero.RepresentacionCanonica, e.primera.Resultado.RepresentacionCanonica) ||
		e.sesion.llamadas != 4 {
		t.Fatal("las dos emisiones no reutilizaron el recibo F1 tras revalidar cada uso")
	}
	primero.RepresentacionCanonica[0] ^= 1
	if !bytes.Equal(segundo.RepresentacionCanonica, e.primera.Resultado.RepresentacionCanonica) {
		t.Fatal("la primera exportación alteró el recibo fijado")
	}
	const concurrentes = 8
	var espera sync.WaitGroup
	resultados := make(chan bool, concurrentes)
	for i := 0; i < concurrentes; i++ {
		espera.Add(1)
		go func() {
			defer espera.Done()
			v, r, err := e.autoridad.revalidarConContexto(e.peticion.Context(), e.peticion, e.capsula)
			resultados <- err == nil && v.ValidarPara(r) == nil &&
				bytes.Equal(r.RepresentacionCanonica, segundo.RepresentacionCanonica)
		}()
	}
	espera.Wait()
	close(resultados)
	for correcto := range resultados {
		if !correcto {
			t.Fatal("una emisión concurrente cambió el contexto de la cápsula")
		}
	}
	if e.sesion.llamadas != 4+2*concurrentes {
		t.Fatal("la concurrencia omitió revalidaciones de sesión")
	}
	// La instantánea se consulta otra vez en cada uso; un control retirado
	// cierra la cápsula antes de transportar el recibo inicial.
	e.perfiles.mu.Lock()
	e.perfiles.actual.ControlVigenciaVersionRol.Revision++
	e.perfiles.actual.ControlVigenciaVersionRol.Estado = core.EstadoControlVigenciaVersionRolRetirada
	e.perfiles.actual.ControlVigenciaVersionRol.ActoRef = "acto:retirada"
	e.perfiles.actual.ControlVigenciaVersionRol.MotivoCodigo = "retirada"
	e.perfiles.mu.Unlock()
	if _, _, err := e.autoridad.revalidarConContexto(e.peticion.Context(), e.peticion, e.capsula); err == nil || !e.capsula.fallida {
		t.Fatal("cambio de control de rol admitido")
	}
}

func TestFirmaVecNoComparteContextoEntrePeticiones(t *testing.T) {
	primera := nuevaCapsulaContextoFijadoPrueba(t, 0)
	segunda := nuevaCapsulaContextoFijadoPrueba(t, 10*time.Second)
	_, r1, err1 := primera.autoridad.revalidarConContexto(primera.peticion.Context(), primera.peticion, primera.capsula)
	_, r2, err2 := segunda.autoridad.revalidarConContexto(segunda.peticion.Context(), segunda.peticion, segunda.capsula)
	if err1 != nil || err2 != nil ||
		bytes.Equal(r1.RepresentacionCanonica, r2.RepresentacionCanonica) ||
		!bytes.Equal(r1.RepresentacionCanonica, primera.primera.Resultado.RepresentacionCanonica) ||
		!bytes.Equal(r2.RepresentacionCanonica, segunda.primera.Resultado.RepresentacionCanonica) {
		t.Fatalf("dos peticiones compartieron o alteraron el recibo F1: %v, %v", err1, err2)
	}
}

func TestFirmaVecCapsulaDivergenteDeniegaSinRecibo(t *testing.T) {
	for nombre, cambiar := range map[string]func(*escenarioCapsulaContextoFijado){
		"otra_sesion": func(e *escenarioCapsulaContextoFijado) {
			e.capsula.sesion = &sesionContextoFijadoPrueba{actual: e.sesion.actual}
		},
		"otro_rol": func(e *escenarioCapsulaContextoFijado) { e.capsula.seleccion.RolID = "rol_ajeno" },
		"otro_certificado": func(e *escenarioCapsulaContextoFijado) {
			prefijo := "0"
			if e.sesion.actual.CertificadoCanalSHA256[0] == '0' {
				prefijo = "1"
			}
			e.sesion.actual.CertificadoCanalSHA256 = prefijo + e.sesion.actual.CertificadoCanalSHA256[1:]
		},
		"otro_actor":     func(e *escenarioCapsulaContextoFijado) { e.capsula.personaCA25 = "per_ajena" },
		"sin_proyeccion": func(e *escenarioCapsulaContextoFijado) { e.sesion.actual.Resultado.Contexto.Instantanea.Vinculos = nil },
	} {
		t.Run(nombre, func(t *testing.T) {
			e := nuevaCapsulaContextoFijadoPrueba(t, 0)
			cambiar(&e)
			if _, _, err := e.autoridad.revalidarConContexto(e.peticion.Context(), e.peticion, e.capsula); err == nil || !e.capsula.fallida {
				t.Fatal("una deriva de AUT56 obtuvo el contexto fijado")
			}
		})
	}
}
