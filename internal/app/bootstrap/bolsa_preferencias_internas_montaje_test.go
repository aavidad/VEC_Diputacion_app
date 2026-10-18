package bootstrap

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httppersonal"
	core "vec-diputacion-granada/internal/vec/domain"
)

func TestPreferenciasBolsasInternasConfiguracionOpcional(t *testing.T) {
	t.Setenv(envBolsaPreferenciasInternas, "")
	ruta, cerrar := nuevaRutaPreferenciasBolsasInternas(config.Config{}, nil, nil, nil)
	if ruta != nil || cerrar == nil {
		t.Fatal("la capacidad apagada alteró las rutas")
	}
	cerrar()
	t.Setenv(envBolsaPreferenciasInternas, "true")
	ruta, cerrar = nuevaRutaPreferenciasBolsasInternas(config.Config{}, nil, nil, nil)
	if ruta == nil || ruta.Ruta != bolsahttp.RutaPreferenciasBolsas || ruta.Manejador == nil || cerrar == nil {
		t.Fatal("la capacidad incompleta afectó al resto del arranque")
	}
	for _, metodo := range []string{http.MethodGet, http.MethodPut} {
		w := httptest.NewRecorder()
		ruta.Manejador.ServeHTTP(w, httptest.NewRequest(metodo, ruta.Ruta, nil))
		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("%s incompleto devolvió %d", metodo, w.Code)
		}
	}
	cerrar()
}

func TestCuentaMiBolsaInternaProcedeDeMaterialCandidato(t *testing.T) {
	const huella = "certificado-ficticio"
	principal := core.Principal{ID: "antonio", Attributes: map[string]string{"certificate_sha256": huella}}
	candidato := &identidadCandidatoBolsaDesarrollo{cuentaRef: "cuenta-externa", perfilRef: "perfil-externo"}
	interna := &autoridadPreferenciasUsuariosDesarrollo{}
	propia := cuentaUsuariosPreferenciasDesarrollo{cuentaRutasDietasDesarrollo: cuentaRutasDietasDesarrollo{
		CertificadoSHA256: huella, Sujeto: "antonio", CuentaRef: "cuenta-interna", PerfilRef: "perfil-interno"}}
	cuenta, ok := cuentaSesionMiBolsa(interna, candidato, principal, core.SuperficieAutenticacionInternaCorporativaV1, propia)
	if !ok || cuenta.CuentaRef != "cuenta-interna" || cuenta.PerfilRef != "perfil-interno" {
		t.Fatal("Mi Bolsa interna no usó la cuenta corporativa")
	}
	cuenta, ok = cuentaSesionMiBolsa(interna, candidato, principal, core.SuperficieAutenticacionExternaPersonalV1, propia)
	if !ok || cuenta.CuentaRef != "cuenta-externa" || cuenta.PerfilRef != "perfil-externo" {
		t.Fatal("Mi Bolsa externa cambió de cuenta")
	}
	principal.ID = "otra-persona"
	if _, ok := cuentaSesionMiBolsa(interna, candidato, principal, core.SuperficieAutenticacionInternaCorporativaV1, propia); ok {
		t.Fatal("un sujeto ajeno obtuvo la cuenta interna")
	}
}

func TestConfiguracionBolsaPreferenciasInternasNoExponeDSN(t *testing.T) {
	c := configuracionBolsaPreferenciasInternas{DSNBolsa: "postgres://usuario:clave-ficticia@127.0.0.1/vec"}
	b, err := json.Marshal(c)
	if err != nil || strings.Contains(string(b), "clave-ficticia") ||
		strings.Contains(fmt.Sprintf("%+v %#v", c, c), "clave-ficticia") {
		t.Fatal("la configuración expuso el DSN")
	}
}
