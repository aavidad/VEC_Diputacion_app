package bootstrap

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
	aspirantescertificado "vec-diputacion-granada/internal/modules/aspirantes/adapters/certificado"
	aspirantesports "vec-diputacion-granada/internal/modules/aspirantes/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func TestDescriptoresAspirantesUnicosYEnElGobierno(t *testing.T) {
	d := descriptoresMaterialAspirantesDesarrollo()
	if len(d) != 3 {
		t.Fatalf("descriptores: %d", len(d))
	}
	todos := append(append(append(descriptoresMaterialPreferenciasUsuariosDesarrollo(), descriptoresMaterialCorreosUsuariosDesarrollo()...),
		descriptoresMaterialImagenUsuariosDesarrollo()...), d...)
	if _, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo(todos); err != nil {
		t.Fatal("descriptores de Aspirantes repetidos con Usuarios")
	}
	gobierno := map[string]bool{}
	for _, a := range audienciasConsumoGobiernoCTDesarrollo() {
		gobierno[a] = true
	}
	for i, a := range accionesAspirantes {
		audiencia, _ := aspirantesports.Audiencia(a.accion)
		if d[i].Audiencia != audiencia || !gobierno[audiencia] || !strings.Contains(audiencia, ".externa_personal.") {
			t.Fatalf("audiencia %d: %s", i, d[i].Audiencia)
		}
	}
}

// Producción usa exactamente las políticas oficiales; las sintéticas solo
// entran con la doble llave de desarrollo y nunca sustituyen una oficial.
func TestPerfilesCertificadoAspirantes(t *testing.T) {
	sinLlave := config.Config{}
	p, err := perfilesCertificadoAspirantes(sinLlave, configuracionAspirantesDesarrollo{})
	oficiales := aspirantescertificado.PerfilesOficiales()
	if err != nil || len(p) != len(oficiales) {
		t.Fatalf("%v %v", p, err)
	}
	for oid, perfil := range oficiales {
		if p[oid] != perfil {
			t.Fatal("falta una política oficial")
		}
	}
	extra := configuracionAspirantesDesarrollo{PoliticasCertificadoDesarrollo: map[string]aspirantescertificado.Perfil{"1.3.6.1.4.1.99999.1": aspirantescertificado.PerfilFNMT}}
	if _, err := perfilesCertificadoAspirantes(sinLlave, extra); err == nil {
		t.Fatal("política sintética sin doble llave")
	}
	llave := config.Config{ExecutionProfile: config.ExecutionProfileDevelopment, AuthMode: config.AuthModeDevelopment, DevelopmentGuard: config.DevelopmentGuardAcknowledgement}
	p, err = perfilesCertificadoAspirantes(llave, extra)
	if err != nil || len(p) != len(oficiales)+1 {
		t.Fatalf("con doble llave: %v %v", p, err)
	}
	pisar := configuracionAspirantesDesarrollo{PoliticasCertificadoDesarrollo: map[string]aspirantescertificado.Perfil{"2.16.724.1.2.2.2.3": aspirantescertificado.PerfilFNMT}}
	if _, err := perfilesCertificadoAspirantes(llave, pisar); err == nil {
		t.Fatal("una política oficial no se redefine")
	}
}

func TestConfiguracionAspirantesEstricta(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "identidad"), 0o700); err != nil {
		t.Fatal(err)
	}
	escribir := func(contenido string) {
		if err := os.WriteFile(filepath.Join(dir, "identidad", nombreConfiguracionAspirantesExterna), []byte(contenido), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	cfg := config.Config{DevelopmentMaterialDir: dir}
	escribir(`{"version":1,"autoridad":"no_autoritativo","dsn_aspirantes":"postgres://x@h/db","tipos_convocatoria":["bolsa"],"politicas_certificado_desarrollo":{}}`)
	c, err := leerConfiguracionAspirantesDesarrollo(cfg)
	if err != nil || c.TiposConvocatoria[0] != "bolsa" || strings.Contains(c.String(), "postgres") {
		t.Fatalf("%v", err)
	}
	for _, malo := range []string{
		`{"version":1,"autoridad":"no_autoritativo","dsn_aspirantes":"postgres://x@h/db","tipos_convocatoria":["bolsa"],"politicas_certificado_desarrollo":{},"extra":1}`,
		`{"version":2,"autoridad":"no_autoritativo","dsn_aspirantes":"postgres://x@h/db","tipos_convocatoria":["bolsa"],"politicas_certificado_desarrollo":{}}`,
		`{"version":1,"autoridad":"autoritativo","dsn_aspirantes":"postgres://x@h/db","tipos_convocatoria":["bolsa"],"politicas_certificado_desarrollo":{}}`,
		`{"version":1,"autoridad":"no_autoritativo","dsn_aspirantes":"","tipos_convocatoria":["bolsa"],"politicas_certificado_desarrollo":{}}`,
		`{"version":1,"autoridad":"no_autoritativo","dsn_aspirantes":"postgres://x@h/db","tipos_convocatoria":[],"politicas_certificado_desarrollo":{}}`,
	} {
		escribir(malo)
		if _, err := leerConfiguracionAspirantesDesarrollo(cfg); err == nil {
			t.Fatalf("aceptada %s", malo)
		}
	}
}

type registradorFronteraPrueba struct {
	ordenes []vecports.OrdenAuditoriaFronteraRutaExacta
}

func (r *registradorFronteraPrueba) RegistrarAuditoriaFronteraRutaExacta(_ context.Context, o vecports.OrdenAuditoriaFronteraRutaExacta) error {
	r.ordenes = append(r.ordenes, o)
	return nil
}

// La frontera de Aspirantes anota 401 y 403 sin persona y con su superficie.
func TestDenegacionAspirantesSinPersona(t *testing.T) {
	r := &registradorFronteraPrueba{}
	a := &autoridadPreferenciasUsuariosDesarrollo{registrador: r, ruta: "/api/vec/aspirantes/area-personal/mi-ficha",
		superficieAuditoria: vecports.SuperficieAuditoriaFronteraRutaExactaAspirantes}
	if err := a.registrarDenegacion(context.Background(), http.StatusUnauthorized, ""); err != nil {
		t.Fatal(err)
	}
	if err := a.registrarDenegacion(context.Background(), http.StatusForbidden, ""); err != nil {
		t.Fatal(err)
	}
	if err := a.registrarDenegacion(context.Background(), http.StatusForbidden, "per_AAAAAAAAAAAAAAAAAAAAAA"); err == nil {
		t.Fatal("Aspirantes nunca anota persona")
	}
	for _, o := range r.ordenes {
		if o.Superficie != vecports.SuperficieAuditoriaFronteraRutaExactaAspirantes || o.ActorRef != "" {
			t.Fatalf("%+v", o)
		}
	}
	m := registradorFronterasConUsuariosPreferencias{aspirantes: r}
	if err := m.RegistrarAuditoriaFronteraRutaExacta(context.Background(), r.ordenes[0]); err != nil || len(r.ordenes) != 3 {
		t.Fatalf("despachador: %v", err)
	}
}
