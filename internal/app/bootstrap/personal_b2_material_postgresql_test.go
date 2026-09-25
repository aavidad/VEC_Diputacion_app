package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Primer paso del ensayo PostgreSQL 18.4 de cmd/vec-preparar-material-interno
// (probar_postgresql_pg18.sh): publica el gobierno exactamente como vec-server
// al arrancar con B2 activo, desde un material de idempotencia sintético y con
// un LOGIN de gobierno como el de vec-server. No usa fixtures de DBA: pool con
// la comprobación de identidad real, raíz, configuración y clave base CT con
// publicarGobiernoAtestacionContratacionTemporalDesarrollo y las ocho claves
// B2 con publicarMaterialPersonalB2Desarrollo. El segundo paso ejecuta la
// herramienta contra este gobierno y el mismo material.
func TestPublicaGobiernoB2ComoVecServerParaEnsayoPostgreSQL18(t *testing.T) {
	if os.Getenv("VEC_T4_PG_DESECHABLE") != "si" {
		t.Skip("requiere PostgreSQL 18.4 desechable (cmd/vec-preparar-material-interno/probar_postgresql_pg18.sh)")
	}
	directorio := os.Getenv("VEC_T4_MATERIAL_IDEMPOTENCIA")
	ctx, cancelar := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancelar()
	gobierno, _, err := abrirPoolPostgreSQLContratacionTemporalDesarrollo(ctx, os.Getenv("VEC_T4_PG_GOBIERNO_DSN"),
		"vec-ct-desarrollo-gobierno", rolGobiernoPostgreSQLContratacionTemporalDesarrollo)
	if err != nil {
		t.Fatalf("pool de gobierno de vec-server rechazado: %v", err)
	}
	defer gobierno.Close()
	raiz := filepath.Dir(directorio)
	idempotencia, err := cargarMaterialIdempotenciaDesarrollo(raiz, filepath.Join(raiz, "idempotencia", "configuracion.json"))
	if err != nil {
		t.Fatal("material de idempotencia sintético rechazado")
	}
	derivador, err := nuevoDerivadorIdentidadOperacionDesarrollo(&idempotencia)
	if err != nil {
		t.Fatal(err)
	}
	defer derivador.borrar()
	material, err := nuevoMaterialAtestacionContratacionTemporalDesarrollo(derivador, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	defer material.borrarCopiasEfimeras()
	if err := publicarGobiernoAtestacionContratacionTemporalDesarrollo(ctx, gobierno, &material); err != nil {
		t.Fatalf("publicación CT: %v", err)
	}
	catalogo, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo(append(descriptoresMaterialAutorizacionContratacionTemporalDesarrollo(), descriptoresMaterialPersonalB2Desarrollo()...))
	if err != nil {
		t.Fatal(err)
	}
	publicadas, err := publicarMaterialPersonalB2Desarrollo(ctx, gobierno, material, catalogo)
	if err != nil {
		t.Fatalf("publicación B2: %v", err)
	}
	// Idempotente, como un reinicio de vec-server.
	otra, err := publicarMaterialPersonalB2Desarrollo(ctx, gobierno, material, catalogo)
	if err != nil || otra != publicadas {
		t.Fatalf("republicación B2 no idempotente: %v", err)
	}
	claves, err := DerivarClavesPersonalB2V3DesdeMaterialDesarrollo(directorio, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	// El secreto guardado se compara en el servidor con el derivado, sin
	// sacarlo de PostgreSQL: misma transacción y rol que el publicador.
	tx, err := gobierno.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	if _, err := tx.Exec(ctx, `SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario`); err != nil {
		t.Fatal(err)
	}
	for i := range claves {
		secreto := claves[i].CopiarSecreto()
		var igual bool
		err := tx.QueryRow(ctx, `SELECT secreto_hmac = $2 FROM vec_autorizacion_atestada_v3.clave_capacidad_version
		  WHERE clave_id = $1 AND version = $3`, publicadas[i].ClaveID, secreto, int64(publicadas[i].Version)).Scan(&igual)
		borrarBytes(secreto)
		if err != nil || !igual || claves[i].ClaveID != publicadas[i].ClaveID || claves[i].SHA256 != publicadas[i].SHA256 ||
			claves[i].HuellaGobierno != publicadas[i].HuellaGobierno {
			t.Fatalf("clave B2 %s divergente de la publicada byte a byte", claves[i].Capacidad)
		}
		claves[i].Borrar()
	}
}
