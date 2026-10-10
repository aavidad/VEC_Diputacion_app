package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// rolAUT24Prueba reproduce las seis claves que devuelve hoy
// resolver_rol_administrable_v1 (AUT24), sin categoria_admin.
func rolAUT24Prueba(ref string, ahora time.Time) map[string]any {
	return map[string]any{"version_ref": ref, "clase": "administrador",
		"huella_sha256": strings.Repeat("a", 64), "vigente_desde": ahora.Add(-time.Hour),
		"vigente_hasta": ahora.Add(time.Hour), "unidad_requerida": false}
}

func TestVersionarRolBolsaClaseResolverRolAdministrable(t *testing.T) {
	ahora := time.Date(2026, 10, 10, 9, 0, 0, 0, time.UTC)
	ref := "rol:administracion_perfiles:v3"
	conCategoria := func(c string) map[string]any {
		x := rolAUT24Prueba(ref, ahora)
		x["categoria_admin"] = c
		return x
	}
	for _, caso := range []struct {
		nombre, ref, clase string
		fila               filaFalsa
	}{
		{"version_fuera_de_aplicacion", "rol:otro:v1", "rol_administrable_version", filaFalsa{}},
		{"consulta", ref, "rol_administrable_consulta", filaFalsa{err: errors.New("caida")}},
		{"aut24_sin_categoria", ref, "rol_administrable_respuesta", filaFalsa{dato: rolAUT24Prueba(ref, ahora)}},
		{"categoria_sistemas", ref, "rol_administrable_incoherente", filaFalsa{dato: conCategoria("sistemas")}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			if m, ok := caso.fila.dato.(map[string]any); ok {
				b, err := json.Marshal(m)
				if err != nil {
					t.Fatal(err)
				}
				caso.fila.dato = b
			}
			a := &AutoridadVersionarRolBolsa{pool: &poolFalso{fila: caso.fila},
				catalogo: fuenteCatalogoGobiernoReferenciaPrueba{},
				emisor:   emisorVersionarRolBolsaPrueba{t: t, ahora: ahora}, reloj: relojFijo(ahora)}
			_, err := a.ResolverRolAdministrable(context.Background(), caso.ref)
			if err == nil || ports.ClaseFalloVersionBolsa(err) != caso.clase {
				t.Fatalf("clase=%q err=%v", ports.ClaseFalloVersionBolsa(err), err)
			}
			if strings.Contains(err.Error(), "caida") {
				t.Fatal("la causa de la consulta llegó al texto del error")
			}
		})
	}
	b, err := json.Marshal(conCategoria("aplicacion"))
	if err != nil {
		t.Fatal(err)
	}
	a := &AutoridadVersionarRolBolsa{pool: &poolFalso{fila: filaFalsa{dato: b}},
		catalogo: fuenteCatalogoGobiernoReferenciaPrueba{},
		emisor:   emisorVersionarRolBolsaPrueba{t: t, ahora: ahora}, reloj: relojFijo(ahora)}
	if _, err := a.ResolverRolAdministrable(context.Background(), ref); err != nil {
		t.Fatalf("rol con categoría aplicacion rechazado: %v", err)
	}
}

func TestVersionarRolBolsaClaseEjecutarSQL(t *testing.T) {
	ahora := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	base, _, _, _ := contratoV2Prueba(t)
	_, asignacion := escenarioRecursoGobiernoRol(t)
	base.InstantaneaAutorizacion.AsignacionPerfil.Ambitos = asignacion.Ambitos
	s := domain.SolicitudCierreVersionarRolBolsa{
		OperacionRef:          "cierre_admin:" + strings.Repeat("4", 32),
		PropuestaRef:          "propuesta_admin:" + strings.Repeat("1", 32),
		PropuestaHuellaSHA256: strings.Repeat("a", 64),
		Aprobador:             base.Actor, Evidencia: base.Evidencia,
		InstantaneaAutorizacion: base.InstantaneaAutorizacion,
		Decision:                domain.DecisionAprobarPropuestaPerfil,
		Motivo: domain.ReferenciaEntradaCatalogo{CatalogoID: "motivos_administracion",
			CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("c", 64),
			EntradaClave: "motivo_" + strings.Repeat("a", 32)},
		CorrelacionRef: "correlacion_" + strings.Repeat("5", 32),
	}
	e, err := materialCierreVersionarRolBolsa(s)
	if err != nil {
		t.Fatal(err)
	}
	intento := func(estado, codigo, auditoria string) []byte {
		b, err := json.Marshal(map[string]any{"estado": estado, "codigo": codigo,
			"auditoria_intento": map[string]any{"auditoria_ref": auditoria}})
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	aud := "aud_v3_" + strings.Repeat("a", 32)
	for _, caso := range []struct {
		nombre, clase string
		fila          filaFalsa
		invalida      bool
	}{
		{"intento_error", "sql_intento_error", filaFalsa{dato: intento("error", "version_rol_bolsa_error", aud)}, false},
		{"intento_sin_auditoria", "sql_intento_incoherente", filaFalsa{dato: intento("error", "version_rol_bolsa_error", "")}, false},
		{"respuesta_invalida", "sql_respuesta_invalida", filaFalsa{dato: []byte(`{"estado":"permitido"}`)}, true},
		{"consulta", "sql_consulta", filaFalsa{err: errors.New("caida")}, false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			tx := &txGobiernoReferenciaPrueba{fila: caso.fila}
			a := &AutoridadVersionarRolBolsa{pool: &poolGobiernoReferenciaPrueba{tx: tx},
				catalogo: fuenteCatalogoGobiernoReferenciaPrueba{},
				emisor:   emisorVersionarRolBolsaPrueba{t: t, ahora: ahora}, reloj: relojFijo(ahora)}
			err := a.ejecutar(context.Background(), s.Aprobador, s.Evidencia, s.InstantaneaAutorizacion,
				e, cerrarVersionarRolBolsaSQL, func([]byte) error {
					if !caso.invalida {
						t.Fatal("validó una respuesta que no era de éxito")
					}
					return ports.ErrAutoridadAdministracionPerfilesNoDisponible
				})
			if !errors.Is(err, ports.ErrAutoridadAdministracionPerfilesNoDisponible) ||
				ports.ClaseFalloVersionBolsa(err) != caso.clase || errors.Is(err, domain.ErrAutorizacionDenegada) {
				t.Fatalf("clase=%q err=%v", ports.ClaseFalloVersionBolsa(err), err)
			}
		})
	}
}
