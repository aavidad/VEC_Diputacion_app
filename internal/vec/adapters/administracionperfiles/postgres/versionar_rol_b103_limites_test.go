package postgres

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"vec-diputacion-granada/internal/vec/ports"
)

type poolCatalogoVersionB103 struct {
	*poolFalso
	consultas []string
	rol       []byte
	ref       string
}

func (p *poolCatalogoVersionB103) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	p.consultas = append(p.consultas, sql)
	switch sql {
	case acreditarVersionarRolBolsaB103SQL:
		return filaFalsa{dato: true}
	case catalogoSQL:
		if len(args) == 1 && args[0] == p.ref {
			return filaFalsa{dato: p.rol}
		}
	}
	return filaFalsa{err: ports.ErrAutoridadAdministracionPerfilesNoDisponible}
}

func TestVersionarRolBolsaB103ResuelveCatalogoConMismoPoolAcreditado(t *testing.T) {
	ahora := time.Date(2026, 10, 12, 10, 0, 0, 0, time.UTC)
	ref := "rol:administracion_perfiles:v3"
	rol := rolAUT24Prueba(ref, ahora)
	rol["categoria_admin"] = "aplicacion"
	b, err := json.Marshal(rol)
	if err != nil {
		t.Fatal(err)
	}
	pool := &poolCatalogoVersionB103{poolFalso: &poolFalso{}, rol: b, ref: ref}
	a, err := nuevaAutoridadVersionarRolBolsaB103(context.Background(), pool,
		fuenteCatalogoGobiernoReferenciaPrueba{}, emisorVersionarRolBolsaPrueba{t: t, ahora: ahora}, relojFijo(ahora))
	if err != nil {
		t.Fatal(err)
	}
	r, err := a.ResolverRolAdministrable(context.Background(), ref)
	if err != nil || r.VersionRef != ref || len(pool.consultas) != 2 ||
		pool.consultas[0] != acreditarVersionarRolBolsaB103SQL || pool.consultas[1] != catalogoSQL {
		t.Fatalf("consulta ADMIN B103: %+v %v; consultas=%d", r, err, len(pool.consultas))
	}
	if _, err := a.ResolverRolAdministrable(context.Background(), "rol:otro:v1"); err == nil ||
		ports.ClaseFalloVersionBolsa(err) != "rol_administrable_version" || len(pool.consultas) != 2 {
		t.Fatal("rol fuera de aplicación llegó a SQL")
	}
	rol["categoria_admin"] = "sistemas"
	pool.rol, err = json.Marshal(rol)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.ResolverRolAdministrable(context.Background(), ref); err == nil ||
		ports.ClaseFalloVersionBolsa(err) != "rol_administrable_incoherente" {
		t.Fatal("categoría ADMIN ajena aceptada")
	}
	if strings.Contains(acreditarVersionarRolBolsaB103SQL, "proponer_version_rol_bolsa_v1(") ||
		strings.Contains(acreditarVersionarRolBolsaB103SQL, "cerrar_version_rol_bolsa_v1(") {
		t.Fatal("preflight B103 depende de fachadas B1")
	}
}
