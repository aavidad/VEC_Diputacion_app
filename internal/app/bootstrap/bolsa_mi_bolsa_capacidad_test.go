package bootstrap

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
	bolsapersonal "vec-diputacion-granada/internal/modules/bolsa/adapters/httppersonal"
	httpinterno "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// El soporte de sesión del candidato debe admitir todas las rutas de Mi
// Bolsa. Antes solo admitía las dos consultas y las acciones propias
// (responder al llamamiento, solicitudes, disposición y contacto) acababan
// en 500 sin llegar a la base.
func TestMiBolsaSoporteCandidatoAdmiteAccionesPropiasDelPortal(t *testing.T) {
	sello := &selloConsultasContratacionTemporalDesarrollo{}
	huella := strings.Repeat("a", 64)
	candidato := dominiovec.Principal{
		ID: "per_candidato_sintetico_1234567890123456", Roles: []string{"candidato_bolsa"},
		AuthMethod: dominiovec.AuthMethodCertificate, AuthAssurance: dominiovec.AuthAssuranceHigh,
		Attributes: map[string]string{
			"autoridad": AutoridadNoAutoritativa, "perfil_ejecucion": config.ExecutionProfileDevelopment,
			"certificate_sha256": huella,
		},
	}
	soporte := &soporteAltaContratacionTemporalDesarrollo{
		sello: sello, principalID: candidato.ID, certificadoSHA256: huella, candidatoBolsa: true,
	}
	capacidadEn := func(ruta string, principal dominiovec.Principal, s *selloConsultasContratacionTemporalDesarrollo) context.Context {
		return context.WithValue(context.Background(), claveCapacidadConsultasContratacionTemporalDesarrollo{},
			capacidadConsultaContratacionTemporalDesarrollo{sello: s, ruta: ruta, metodo: http.MethodPost, principal: principal})
	}
	for _, ruta := range []string{
		bolsapersonal.RutaMiBolsa, bolsapersonal.RutaMiBolsaHistorial,
		bolsapersonal.RutaMiBolsaRespuestas, bolsapersonal.RutaMiBolsaSolicitudes,
		bolsapersonal.RutaMiBolsaSolicitudesDocumentales, bolsapersonal.RutaMiBolsaDisposiciones,
		bolsapersonal.RutaMiBolsaContacto,
	} {
		if _, ok := soporte.capacidadValida(capacidadEn(ruta, candidato, sello)); !ok {
			t.Errorf("%s: el candidato no alcanza su propia ruta de Mi Bolsa", ruta)
		}
	}
	// Sigue cerrado fuera de Mi Bolsa, con otro sello o con otro rol.
	if _, ok := soporte.capacidadValida(capacidadEn(httpinterno.RutaAltaSolicitudes, candidato, sello)); ok {
		t.Error("el candidato alcanza una ruta de RRHH")
	}
	if _, ok := soporte.capacidadValida(capacidadEn(bolsapersonal.RutaMiBolsaRespuestas, candidato, &selloConsultasContratacionTemporalDesarrollo{})); ok {
		t.Error("se admite una capacidad con sello ajeno")
	}
	otroRol := candidato
	otroRol.Roles = []string{rolTecnicoRRHHContratacionTemporalDesarrollo}
	if _, ok := soporte.capacidadValida(capacidadEn(bolsapersonal.RutaMiBolsaRespuestas, otroRol, sello)); ok {
		t.Error("se admite un principal que no es candidato")
	}
	otraHuella := candidato
	otraHuella.Attributes = map[string]string{
		"autoridad": AutoridadNoAutoritativa, "perfil_ejecucion": config.ExecutionProfileDevelopment,
		"certificate_sha256": strings.Repeat("b", 64),
	}
	if _, ok := soporte.capacidadValida(capacidadEn(bolsapersonal.RutaMiBolsaRespuestas, otraHuella, sello)); ok {
		t.Error("se admite otro certificado")
	}
}
