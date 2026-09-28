package bootstrap

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

type filaPreflightB55Prueba struct {
	autorizada bool
	err        error
}

func (f filaPreflightB55Prueba) Scan(destino ...any) error {
	if f.err != nil {
		return f.err
	}
	*destino[0].(*bool) = f.autorizada
	return nil
}

type consultaPreflightB55Prueba struct {
	fila     filaPreflightB55Prueba
	consulta string
}

func (c *consultaPreflightB55Prueba) QueryRow(_ context.Context, consulta string, _ ...any) pgx.Row {
	c.consulta = consulta
	return c.fila
}

func TestPreflightB55ExigeFuncionLecturaYLoginNominal(t *testing.T) {
	for _, caso := range []struct {
		nombre      string
		fila        filaPreflightB55Prueba
		debeAceptar bool
	}{
		{"permitido", filaPreflightB55Prueba{autorizada: true}, true},
		{"revocado_o_rol_ajeno", filaPreflightB55Prueba{}, false},
		{"sql_ausente", filaPreflightB55Prueba{err: errors.New("funcion ausente")}, false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			consulta := &consultaPreflightB55Prueba{fila: caso.fila}
			err := comprobarLecturaReincorporacionTitularB55Desarrollo(t.Context(), consulta)
			if caso.debeAceptar && err != nil || !caso.debeAceptar && !errors.Is(err, puertosbolsa.ErrReincorporacionTitularNoDisponible) {
				t.Fatalf("preflight B55: %v", err)
			}
			for _, contrato := range []string{
				"listar_reincorporaciones_titular_ct_v2", "consumir_consulta_reincorporacion_titular_bolsa_v3_atestada",
				"vec_bolsa_llamamientos_ejecutor", "session_user = current_user", "has_function_privilege(session_user,funciones.nueva,'EXECUTE')",
				"NOT EXISTS (SELECT 1 FROM membresias_efectivas",
				"NOT COALESCE(pg_catalog.has_function_privilege(session_user,funciones.anterior,'EXECUTE')",
				"JOIN pg_catalog.pg_namespace AS n ON n.oid = p.pronamespace",
				"pg_catalog.has_function_privilege('vec_bolsa_llamamientos_propietario',fachada.oid,'EXECUTE')",
				"NOT pg_catalog.has_function_privilege(session_user,fachada.oid,'EXECUTE')",
				"c.relowner = 'vec_bolsa_llamamientos_propietario'::pg_catalog.regrole",
				"t.tgname = 'reincorporacion_titular_lectura_inmutable'",
				"t.tgqual IS NULL AND t.tgattr::text = ''",
				"has_table_privilege(session_user,tabla_lectura.oid,",
				"has_table_privilege('vec_bolsa_llamamientos_ejecutor',tabla_lectura.oid,",
				"has_any_column_privilege(session_user,tabla_lectura.oid,",
				"has_any_column_privilege('vec_bolsa_llamamientos_ejecutor',tabla_lectura.oid,",
			} {
				if !strings.Contains(consulta.consulta, contrato) {
					t.Fatalf("preflight B55 omite %s", contrato)
				}
			}
		})
	}
	if err := comprobarLecturaReincorporacionTitularB55Desarrollo(t.Context(), nil); !errors.Is(err, puertosbolsa.ErrReincorporacionTitularNoDisponible) {
		t.Fatalf("consulta B55 sin pool aceptada: %v", err)
	}
}

func TestMaterialAtestacionContratacionTemporalDesarrolloEsEstableYSeparado(
	t *testing.T,
) {
	cfg, _ := generarMaterialDesarrolloPrueba(t)
	composicion, err := NuevaComposicionSeguridadDesarrollo(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	ahora := time.Date(2026, 9, 4, 0, 45, 0, 0, time.UTC)
	primero, err := nuevoMaterialAtestacionContratacionTemporalDesarrollo(
		composicion.derivadorIdempotencia, ahora,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer primero.borrarCopiasEfimeras()
	segundo, err := nuevoMaterialAtestacionContratacionTemporalDesarrollo(
		composicion.derivadorIdempotencia, ahora.Add(time.Hour),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer segundo.borrarCopiasEfimeras()
	if primero.claveID != segundo.claveID ||
		primero.claveHMACID != segundo.claveHMACID ||
		primero.spkiHuella != segundo.spkiHuella ||
		primero.claveHMACSecreto != segundo.claveHMACSecreto ||
		primero.configuracionHuella != segundo.configuracionHuella {
		t.Fatal("el material cambio al reconstruir la composicion")
	}
	if bytes.Equal(primero.claveHMAC, primero.privada[:32]) {
		t.Fatal("firma y capacidad reutilizan el mismo material")
	}
	if primero.configuracionRef != "confianza:atestacion:ct:desarrollo:2026-09-04" ||
		primero.configuracionOrden != 20260904 {
		t.Fatalf("gobierno diario inesperado: %q/%d",
			primero.configuracionRef, primero.configuracionOrden)
	}
}

func TestAutoridadSinteticaContratacionTemporalDesarrolloEsEstableYNoColisiona(
	t *testing.T,
) {
	_, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	primero, err := nuevoContextoAltaContratacionTemporalDesarrollo(principal, ahora)
	if err != nil {
		t.Fatal(err)
	}
	segundo, err := nuevoContextoAltaContratacionTemporalDesarrollo(
		principal, ahora.Add(time.Hour),
	)
	if err != nil {
		t.Fatal(err)
	}
	if primero.Resultado.RegistroContextoRef != segundo.Resultado.RegistroContextoRef ||
		!bytes.Equal(
			primero.Resultado.RepresentacionCanonica,
			segundo.Resultado.RepresentacionCanonica,
		) ||
		!bytes.Equal(
			primero.Resultado.ManifiestoProcedenciaCanonico,
			segundo.Resultado.ManifiestoProcedenciaCanonico,
		) {
		t.Fatal("el contexto sintetico cambio entre dos arranques")
	}
	altaPrimera, err := nuevaInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(
		"per_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"prf_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ahora,
	)
	if err != nil {
		t.Fatal(err)
	}
	altaRepetida, err := nuevaInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(
		"per_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"prf_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ahora.Add(time.Hour),
	)
	if err != nil {
		t.Fatal(err)
	}
	otraIdentidad, err := nuevaInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(
		"per_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		"prf_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		ahora,
	)
	if err != nil {
		t.Fatal(err)
	}
	if altaPrimera.AsignacionPerfil.Referencia() !=
		altaRepetida.AsignacionPerfil.Referencia() {
		t.Fatal("la asignacion cambio entre dos arranques")
	}
	if altaPrimera.AsignacionPerfil.Referencia() ==
		otraIdentidad.AsignacionPerfil.Referencia() {
		t.Fatal("dos identidades comparten referencia de asignacion")
	}
	otroPerfil, err := nuevaInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(
		"per_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"prf_cccccccccccccccccccccccccccccccc",
		ahora,
	)
	if err != nil {
		t.Fatal(err)
	}
	if altaPrimera.AsignacionPerfil.Referencia() ==
		otroPerfil.AsignacionPerfil.Referencia() {
		t.Fatal("dos certificados de una identidad comparten asignacion")
	}
}

func TestPublicacionAutorizacionContratacionTemporalDesarrolloRechazaAsignacionAjena(
	t *testing.T,
) {
	casos := []struct {
		nombre string
		mutar  func(*soporteAltaContratacionTemporalDesarrollo)
	}{
		{
			nombre: "perfil",
			mutar: func(soporte *soporteAltaContratacionTemporalDesarrollo) {
				soporte.instantanea.AsignacionPerfil.PerfilActivoRef = "perfil:ajeno"
			},
		},
		{
			nombre: "principal",
			mutar: func(soporte *soporteAltaContratacionTemporalDesarrollo) {
				soporte.instantanea.AsignacionPerfil.PrincipalID = "principal:ajeno"
			},
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			soporte, _, _ := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
			caso.mutar(soporte)
			err := publicarAutorizacionPostgreSQLContratacionTemporalDesarrollo(
				context.Background(), nil, soporte,
			)
			if !errors.Is(err, errPostgreSQLContratacionTemporalDesarrolloNoDisponible) {
				t.Fatalf("asignacion ajena aceptada: %v", err)
			}
		})
	}
}
