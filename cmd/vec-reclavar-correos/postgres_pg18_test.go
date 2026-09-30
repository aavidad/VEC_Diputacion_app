package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"vec-diputacion-granada/internal/modules/usuarios/adapters/seguridad"
)

// Fixture únicamente en el contenedor desechable propio de esta prueba.
// La variable elige fase; ninguna clave ni conexión procede del entorno.
func TestCLIReclaveadoPG18(t *testing.T) {
	fase := os.Getenv("VEC_RECLAVEADO_PG18_FASE")
	if fase != "preparar" && fase != "ejecutar" {
		t.Skip("prueba PostgreSQL explícita")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	p := planPrueba()
	p.Base = "postgres"
	p.Conexion.Puerto = 55449
	p.Conexion.Usuario = "postgres"
	p.ConexionHuella = huellaConexion(p.Base, p.Conexion)
	dsn := []byte("postgresql://postgres@127.0.0.1:55449/postgres?sslmode=disable")
	cfg, err := configuracionConexion(dsn, p, nil)
	if err != nil {
		t.Fatal(err)
	}
	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	if err := conn.QueryRow(ctx, `SELECT system_identifier::text FROM pg_control_system()`).Scan(&p.Sistema); err != nil {
		t.Fatal(err)
	}
	persona := "per_" + strings.Repeat("q", 22)
	p.Personas = []objetivo{{Persona: persona}}
	vieja, propia := fuentesPrueba()
	defer vieja.borrar()
	defer propia.borrar()
	var anterior, externa [32]byte
	for i := range anterior {
		anterior[i] = byte(i + 1)
		externa[i] = byte(i + 33)
	}
	defer clear(anterior[:])
	defer clear(externa[:])
	if fase == "preparar" {
		if _, err := conn.Exec(ctx, `INSERT INTO vec_usuarios_correos_externo.correos_conjunto VALUES ($1,7,$2,'2026-09-30T00:00:00Z')`, persona, vieja.claves.Igualdad.Ref); err != nil {
			t.Fatal(err)
		}
		adapter, _ := seguridad.NuevoAdaptadorCorreos(vieja, time.Now)
		for i, estado := range []string{"verificado", "retirado", "pendiente"} {
			correo := "correo:" + strings.Repeat(string(rune('a'+i)), 32)
			s, err := adapter.CifrarDireccionCorreo(ctx, persona, correo, 7, []byte([]string{"lucia.morales@example.invalid", "lucia.otra@example.invalid", "lucia.nueva@example.invalid"}[i]))
			if err != nil {
				t.Fatal(err)
			}
			_, err = conn.Exec(ctx, `INSERT INTO vec_usuarios_correos_externo.correos_direccion
 (persona_ref,correo_ref,version_sobre,clave_sobre_ref,clave_igualdad_ref,nonce,cifrado,huella_igualdad,estado,activo,creado_en,verificado_en,retirado_en)
 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,'2026-09-30T00:00:00Z',CASE WHEN $9='verificado' THEN '2026-09-30T00:01:00Z'::timestamptz END,CASE WHEN $9='retirado' THEN '2026-09-30T00:02:00Z'::timestamptz END)`, persona, correo, int64(s.Version), s.ClaveRef, s.ClaveIgualdadRef, s.Nonce, s.Cifrado, s.HuellaIgualdad, estado, i == 0)
			if err != nil {
				t.Fatal(err)
			}
		}
		return
	}
	invocar := func(modo string) informe {
		planJSON, _ := json.Marshal(p)
		o := opciones{modo: modo, planFD: fdPrueba(t, planJSON), dsnFD: fdPrueba(t, dsn), anteriorFD: fdPrueba(t, anterior[:]), externaFD: fdPrueba(t, externa[:]), caFD: -1}
		var salida bytes.Buffer
		if err := ejecutar(ctx, o, &salida); err != nil {
			t.Fatal(modo, err)
		}
		var i informe
		if json.Unmarshal(salida.Bytes(), &i) != nil {
			t.Fatal("salida no JSON")
		}
		return i
	}
	i := invocar("inventario")
	if i.Recuentos["correos_direccion"] != 3 || !digestValido.MatchString(i.Preimagen) {
		t.Fatal("inventario incompleto")
	}
	p.Personas[0].Preimagen = i.Preimagen
	invocar("ensayo")
	var ledger int
	if err := conn.QueryRow(ctx, `SELECT count(*) FROM vec_usuarios_correos_reclaveado.recibo WHERE persona_ref=$1`, persona).Scan(&ledger); err != nil || ledger != 0 {
		t.Fatal("ensayo cambió ledger", err)
	}
	confirmado := invocar("aplicar")
	replay := invocar("aplicar")
	var original, repetido map[string]any
	if json.Unmarshal(confirmado.Recibo, &original) != nil || json.Unmarshal(replay.Recibo, &repetido) != nil || original["recibo_ref"] != repetido["recibo_ref"] || repetido["replay"] != true {
		t.Fatal("recibo repetido divergente")
	}
	if err := conn.QueryRow(ctx, `SELECT count(*) FROM vec_usuarios_correos_reclaveado.recibo WHERE persona_ref=$1`, persona).Scan(&ledger); err != nil || ledger != 1 {
		t.Fatal("recibo duplicado", err)
	}
	var retirada bool
	if err := conn.QueryRow(ctx, `SELECT estado='retirado' AND clave_sobre_ref=$2 FROM vec_usuarios_correos_externo.correos_direccion WHERE persona_ref=$1 AND correo_ref=$3`, persona, propia.claves.CifradoActivo.Ref, "correo:"+strings.Repeat("b", 32)).Scan(&retirada); err != nil || !retirada {
		t.Fatal("retirada no convertida", err)
	}
}

func fdPrueba(t *testing.T, b []byte) int {
	t.Helper()
	ruta := filepath.Join(t.TempDir(), "material")
	if err := os.WriteFile(ruta, b, 0600); err != nil {
		t.Fatal(err)
	}
	fd, err := syscall.Open(ruta, syscall.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
	if err != nil {
		t.Fatal(err)
	}
	return fd
}
