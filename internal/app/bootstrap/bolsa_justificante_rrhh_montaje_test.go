package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

func TestJustificanteRenunciaRRHHSinConfiguracionDejaRutaCerrada(t *testing.T) {
	ruta, cerrar := nuevaRutaJustificanteRenunciaRRHH(config.Config{}, nil, nil, nil, nil)
	if ruta == nil || ruta.Ruta != bolsahttp.RutaJustificanteRenunciaRRHH || ruta.Manejador == nil || cerrar == nil {
		t.Fatal("la falta de configuración impidió montar la ruta cerrada")
	}
	w := httptest.NewRecorder()
	ruta.Manejador.ServeHTTP(w, httptest.NewRequest(http.MethodGet,
		ruta.Ruta+"?bolsa=bolsa_1&participacion=part_1", nil))
	if w.Code != http.StatusServiceUnavailable || w.Header().Get("Cache-Control") == "" {
		t.Fatalf("sin configuración: estado=%d cabeceras=%v", w.Code, w.Header())
	}
	cerrar()
}

func TestConfiguracionJustificanteRenunciaRRHHNoExponeDSN(t *testing.T) {
	c := configuracionJustificanteRenunciaRRHH{DSNDocumentos: "postgres://usuario:clave-ficticia@127.0.0.1/vec"}
	b, err := json.Marshal(c)
	if err != nil || strings.Contains(string(b), "clave-ficticia") ||
		strings.Contains(fmt.Sprintf("%+v %#v", c, c), "clave-ficticia") {
		t.Fatal("la configuración expuso el DSN")
	}
}

func TestConfiguracionJustificanteRRHHSoloAceptaDependenciasTecnicas(t *testing.T) {
	directorio := t.TempDir()
	ruta := filepath.Join(directorio, rutaConfiguracionJustificanteRenunciaRRHH)
	if err := os.MkdirAll(filepath.Dir(ruta), 0o700); err != nil {
		t.Fatal(err)
	}
	base := `{"version":1,"superficie":"interna_corporativa","dsn_documentos":"postgres://ejemplo"`
	for _, caso := range []struct {
		nombre, contenido string
		aceptar           bool
	}{
		{"tecnica", base + `}`, true},
		{"persona_fijada", base + `,"persona_ref":"per_fijada"}`, false},
		{"perfil_fijado", base + `,"perfil_ref":"prf_fijado"}`, false},
		{"certificado_fijado", base + `,"certificado_sha256":"fijado"}`, false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			if err := os.WriteFile(ruta, []byte(caso.contenido), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := leerConfiguracionJustificanteRenunciaRRHH(config.Config{DevelopmentMaterialDir: directorio})
			if (err == nil) != caso.aceptar {
				t.Fatalf("configuracion aceptada=%t", err == nil)
			}
		})
	}
}

type consultorJustificanteRRHHPermisoPrueba struct {
	actorRefs []string
	denegar   bool
}

func (c *consultorJustificanteRRHHPermisoPrueba) AbrirJustificanteRenunciaRRHH(_ context.Context,
	i bolsahttp.IdentidadJustificanteRenunciaRRHH, _, _ string) (io.ReadCloser, error) {
	c.actorRefs = append(c.actorRefs, i.Actor.Contexto.PersonaRef)
	if c.denegar {
		return nil, docports.ErrSubidaLocalDenegada
	}
	return io.NopCloser(strings.NewReader("%PDF-1.4\nrecibo\n%%EOF")), nil
}

func TestJustificanteRRHHResuelveDosTecnicasYDeniegaSinPermiso(t *testing.T) {
	base, _, principalBase := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	principalLector := clonarPrincipalDesarrollo(principalBase)
	principalLector.ID = "tecnica_rrhh_segunda"
	principalLector.Attributes["certificate_sha256"] = strings.Repeat("e", 64)
	contextoLector, err := nuevoContextoAltaContratacionTemporalDesarrollo(principalLector,
		time.Now().UTC().Truncate(time.Microsecond))
	if err != nil {
		t.Fatal(err)
	}
	lector := &soporteAltaContratacionTemporalDesarrollo{sello: base.sello,
		principalID: principalLector.ID, certificadoSHA256: principalLector.Attributes["certificate_sha256"],
		lectorConsultasRRHH: true, tecnicoConsultaRRHH: true, contexto: contextoLector,
		contextoEsperadoRegistrado: contextoLector.Resultado, reloj: relojContratacionTemporalDesarrollo{},
		sesionOperativa: proveedorSesionOperativaCTPrueba{contexto: contextoLector}}
	perfiles := []string{base.contexto.Resultado.Contexto.PerfilActivoRef,
		lector.contexto.Resultado.Contexto.PerfilActivoRef}
	if perfiles[0] == perfiles[1] {
		t.Fatal("las tecnicas comparten un perfil")
	}
	// Ambas técnicas tienen también Preferencias, cada una con un perfil
	// personal distinto. Ninguna referencia de ese catálogo entra en la
	// sesión técnica que se entrega al justificante.
	perfilesUsuarios := map[string]string{
		principalBase.ID:   referenciaAltaContratacionTemporalDesarrollo("prf_", "preferencias-primera"),
		principalLector.ID: referenciaAltaContratacionTemporalDesarrollo("prf_", "preferencias-segunda"),
	}
	if perfilesUsuarios[principalBase.ID] == perfilesUsuarios[principalLector.ID] ||
		perfilesUsuarios[principalBase.ID] == perfiles[0] || perfilesUsuarios[principalLector.ID] == perfiles[1] {
		t.Fatal("la prueba no separa perfiles personales y técnicos")
	}
	fronteras, err := nuevoCatalogoFronterasComunDesarrollo([]descriptorFronteraComunDesarrollo{{
		Clave: "bolsa-justificante-rrhh-prueba", Metodo: http.MethodGet,
		Ruta:               bolsahttp.RutaJustificanteRenunciaRRHH,
		Superficie:         superficieInternaSeguridadComunDesarrollo,
		PerfilesActivosRef: perfiles, ClavePolitica: "politica-documental-prueba",
		ClaveCapacidad: "capacidad-documental-prueba",
	}})
	if err != nil {
		t.Fatal(err)
	}
	autoridad := &autoridadConsultasContratacionTemporalDesarrollo{sello: base.sello,
		soporteDocumentosBolsa: base,
		soportesLectoresDocumentosBolsa: map[string]*soporteAltaContratacionTemporalDesarrollo{
			principalLector.ID: lector}, fronterasSeguridadComun: fronteras}
	consulta := &consultorJustificanteRRHHPermisoPrueba{}
	h, err := bolsahttp.NuevoHandlerJustificanteRenunciaRRHH(
		proveedorIdentidadJustificanteRenunciaRRHH{sesion: sesionTecnicaJustificanteRenunciaRRHH{autoridad}}, consulta)
	if err != nil {
		t.Fatal(err)
	}
	peticion := func(soporte *soporteAltaContratacionTemporalDesarrollo, principal core.Principal) *http.Request {
		r := httptest.NewRequest(http.MethodGet,
			bolsahttp.RutaJustificanteRenunciaRRHH+"?bolsa=bolsa_1&participacion=part_1", nil)
		ctx := contextoRutaCoberturaDesarrolloPrueba(soporte, principal, bolsahttp.RutaJustificanteRenunciaRRHH)
		capacidad := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
		capacidad.metodo = http.MethodGet
		ctx = context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidad)
		ctx = context.WithValue(ctx, claveFronteraSeguridadComunDesarrollo{}, fronteraSeguridadComunDesarrollo{
			metodo: http.MethodGet, ruta: bolsahttp.RutaJustificanteRenunciaRRHH,
			superficie: superficieInternaSeguridadComunDesarrollo, catalogo: fronteras})
		return r.WithContext(ctx)
	}
	peticiones := []*http.Request{peticion(base, principalBase), peticion(lector, principalLector)}
	contextos := []core.ResultadoContextoActorRegistradoV2{
		base.contexto.Resultado, lector.contexto.Resultado}
	for indice, r := range peticiones {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusOK || len(consulta.actorRefs) != indice+1 ||
			consulta.actorRefs[indice] != contextos[indice].Contexto.PersonaRef {
			t.Fatalf("tecnica %d: estado=%d actor=%v", indice, w.Code, consulta.actorRefs)
		}
	}
	if consulta.actorRefs[0] == consulta.actorRefs[1] {
		t.Fatal("las tecnicas compartieron identidad")
	}
	consulta.denegar = true
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticiones[1])
	if w.Code != http.StatusForbidden || len(consulta.actorRefs) != 3 ||
		consulta.actorRefs[2] != contextos[1].Contexto.PersonaRef {
		t.Fatalf("permiso retirado: estado=%d actor=%v", w.Code, consulta.actorRefs)
	}
}
