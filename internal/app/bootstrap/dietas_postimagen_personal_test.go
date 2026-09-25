package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// La huella de cada función es la de su cuerpo en la migración canónica: si
// alguien edita la migración sin actualizar el preflight (o al revés), falla.
func TestPostimagenPersonalDietasHuellasCoincidenConMigraciones(t *testing.T) {
	directorio := filepath.Join("..", "..", "..", "deploy", "postgresql", "personal", "migraciones")
	for _, f := range postimagenPersonalDietas {
		if f.soloFirma {
			if f.huella != "" || f.fichero != "" {
				t.Fatalf("%s: prerrequisito de orden no debe fijar huella", f.firma)
			}
			continue
		}
		contenido, err := os.ReadFile(filepath.Join(directorio, f.fichero))
		if err != nil {
			t.Fatal(err)
		}
		patron := regexp.MustCompile(`(?s)CREATE (?:OR REPLACE )?FUNCTION vec_personal\.` + regexp.QuoteMeta(f.nombreCorto) + `\(.*?\bAS (\$[A-Za-z_]*\$)(.*?)(\$[A-Za-z_]*\$)`)
		coincidencias := patron.FindAllSubmatch(contenido, -1)
		if len(coincidencias) != 1 || string(coincidencias[0][1]) != string(coincidencias[0][3]) {
			t.Fatalf("%s: definicion no localizada de forma univoca en %s", f.nombreCorto, f.fichero)
		}
		suma := sha256.Sum256(coincidencias[0][2])
		if got := hex.EncodeToString(suma[:]); got != f.huella {
			t.Fatalf("%s: huella %s, migracion %s", f.nombreCorto, f.huella, got)
		}
		if !strings.HasPrefix(f.firma, "vec_personal."+f.nombreCorto+"(") || !strings.HasPrefix(f.fichero, f.migracion+"_") {
			t.Fatalf("%s: firma o migracion incoherentes", f.nombreCorto)
		}
	}
	// Ninguna otra migración redefine ni reconcede las funciones fijadas.
	entradas, err := os.ReadDir(directorio)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entradas {
		if !strings.HasSuffix(e.Name(), ".up.sql") {
			continue
		}
		contenido, err := os.ReadFile(filepath.Join(directorio, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range postimagenPersonalDietas {
			if f.soloFirma || e.Name() == f.fichero {
				continue
			}
			if regexp.MustCompile(`FUNCTION vec_personal\.` + regexp.QuoteMeta(f.nombreCorto) + `\(`).Match(contenido) {
				t.Fatalf("%s redefine o reconcede %s: actualizar el preflight", e.Name(), f.nombreCorto)
			}
		}
	}
}

type filaPostimagenFalsa struct {
	fallos []string
	err    error
}

func (f filaPostimagenFalsa) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	*(destinos[0].(*[]string)) = f.fallos
	return nil
}

type consultorPostimagenFalso struct {
	fila filaPostimagenFalsa
	args []any
}

func (c *consultorPostimagenFalso) QueryRow(_ context.Context, _ string, args ...any) pgx.Row {
	c.args = args
	return c.fila
}

func TestAcreditarPostimagenPersonalDietasFallaCerrado(t *testing.T) {
	ctx := context.Background()
	if err := acreditarPostimagenPersonalDietas(ctx, nil); !errors.Is(err, errPostimagenPersonalDietasNoAcreditada) {
		t.Fatalf("pool nulo: %v", err)
	}
	var poolNulo *pgxpool.Pool
	if err := acreditarPostimagenPersonalDietas(ctx, poolNulo); !errors.Is(err, errPostimagenPersonalDietasNoAcreditada) {
		t.Fatalf("pool tipado nulo: %v", err)
	}
	cancelado, cancelar := context.WithCancel(ctx)
	cancelar()
	if err := acreditarPostimagenPersonalDietas(cancelado, &consultorPostimagenFalso{}); !errors.Is(err, errPostimagenPersonalDietasNoAcreditada) {
		t.Fatalf("contexto cancelado: %v", err)
	}
	if err := acreditarPostimagenPersonalDietas(ctx, &consultorPostimagenFalso{fila: filaPostimagenFalsa{err: errors.New("secreto postgresql://x:y@h")}}); !errors.Is(err, errPostimagenPersonalDietasNoAcreditada) || strings.Contains(err.Error(), "postgresql://") {
		t.Fatalf("error de catalogo: %v", err)
	}
	muchos := make([]string, 20)
	for i := range muchos {
		muchos[i] = "000012 vec_personal.x: huella distinta"
	}
	err := acreditarPostimagenPersonalDietas(ctx, &consultorPostimagenFalso{fila: filaPostimagenFalsa{fallos: muchos}})
	if !errors.Is(err, errPostimagenPersonalDietasNoAcreditada) || !strings.Contains(err.Error(), "huella distinta") || !strings.Contains(err.Error(), "y 8 mas") {
		t.Fatalf("diagnostico acotado ausente: %v", err)
	}
	consultor := &consultorPostimagenFalso{fila: filaPostimagenFalsa{fallos: []string{}}}
	if err := acreditarPostimagenPersonalDietas(ctx, consultor); err != nil {
		t.Fatalf("postimagen completa rechazada: %v", err)
	}
	if len(consultor.args) != 5 || !strings.Contains(consultor.args[0].(string), `"huella":"4ec7cbfa`) {
		t.Fatalf("especificacion no enviada: %#v", consultor.args)
	}
}

// Requiere la base desechable que prepara
// deploy/postgresql/personal/pruebas_sql/probar_postimagen_dietas_pg18.sh:
// un LOGIN de vec_dietas_ejecutor y una conexión DBA para las mutaciones.
func TestAcreditarPostimagenPersonalDietasPGReal(t *testing.T) {
	dsnLogin, dsnAdmin := os.Getenv("VEC_DIETAS_POSTIMAGEN_PG_URL"), os.Getenv("VEC_DIETAS_POSTIMAGEN_ADMIN_URL")
	if dsnLogin == "" || dsnAdmin == "" {
		t.Skip("PostgreSQL 18 desechable de postimagen no configurado")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	login, err := pgxpool.New(ctx, dsnLogin)
	if err != nil {
		t.Fatal(err)
	}
	defer login.Close()
	admin, err := pgx.Connect(ctx, dsnAdmin)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close(ctx)
	if err := acreditarPostimagenPersonalDietas(ctx, login); err != nil {
		t.Fatalf("postimagen canonica rechazada: %v", err)
	}
	const consulta = "vec_personal.consultar_asignacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)"
	var definicion string
	if err := admin.QueryRow(ctx, "SELECT pg_get_functiondef($1::regprocedure)", consulta).Scan(&definicion); err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		nombre, mutar, restaurar, esperado string
	}{
		{"EXECUTE D7 al ejecutor Dietas", "GRANT EXECUTE ON FUNCTION " + consulta + " TO vec_dietas_ejecutor", "REVOKE EXECUTE ON FUNCTION " + consulta + " FROM vec_dietas_ejecutor", "EXECUTE ajeno"},
		{"EXECUTE a PUBLIC", "GRANT EXECUTE ON FUNCTION " + consulta + " TO PUBLIC", "REVOKE EXECUTE ON FUNCTION " + consulta + " FROM PUBLIC", "EXECUTE a PUBLIC"},
		{"sin SECURITY DEFINER", "ALTER FUNCTION " + consulta + " SECURITY INVOKER", "ALTER FUNCTION " + consulta + " SECURITY DEFINER", "SECURITY DEFINER"},
		{"propietario ajeno", "ALTER FUNCTION " + consulta + " OWNER TO postgres", "ALTER FUNCTION " + consulta + " OWNER TO vec_personal_propietario", "propietario distinto"},
		{"cuerpo alterado", strings.Replace(definicion, "SELECT * FROM", "SELECT  * FROM", 1), definicion, "huella distinta"},
		{"lectura directa de tabla", "GRANT SELECT ON vec_personal.asignacion_dietas TO vec_personal_d7_ejecutor", "REVOKE SELECT ON vec_personal.asignacion_dietas FROM vec_personal_d7_ejecutor", "privilegio directo"},
		{"RLS no forzada", "ALTER TABLE vec_personal.recibo_asignacion_dietas NO FORCE ROW LEVEL SECURITY", "ALTER TABLE vec_personal.recibo_asignacion_dietas FORCE ROW LEVEL SECURITY", "tabla vec_personal.recibo_asignacion_dietas"},
		{"grupo D7 asciende", "GRANT pg_read_all_data TO vec_personal_d7_ejecutor", "REVOKE pg_read_all_data FROM vec_personal_d7_ejecutor", "grupo vec_personal_d7_ejecutor"},
		{"000013 ausente", "ALTER FUNCTION vec_personal.registrar_auditoria_frontera_asignacion_dietas_v1(text,text,text,text,text,text,text,smallint) RENAME TO registrar_auditoria_frontera_asignacion_dietas_v0", "ALTER FUNCTION vec_personal.registrar_auditoria_frontera_asignacion_dietas_v0(text,text,text,text,text,text,text,smallint) RENAME TO registrar_auditoria_frontera_asignacion_dietas_v1", "000013 vec_personal.registrar_auditoria_frontera_asignacion_dietas_v1(text,text,text,text,text,text,text,smallint): ausente"},
	}
	for _, caso := range casos {
		if _, err := admin.Exec(ctx, caso.mutar); err != nil {
			t.Fatalf("%s: mutar: %v", caso.nombre, err)
		}
		err := acreditarPostimagenPersonalDietas(ctx, login)
		if _, errRestaurar := admin.Exec(ctx, caso.restaurar); errRestaurar != nil {
			t.Fatalf("%s: restaurar: %v", caso.nombre, errRestaurar)
		}
		if !errors.Is(err, errPostimagenPersonalDietasNoAcreditada) || !strings.Contains(err.Error(), caso.esperado) {
			t.Fatalf("%s: se esperaba %q, obtenido %v", caso.nombre, caso.esperado, err)
		}
		if err := acreditarPostimagenPersonalDietas(ctx, login); err != nil {
			t.Fatalf("%s: restauracion no acreditada: %v", caso.nombre, err)
		}
	}
}
