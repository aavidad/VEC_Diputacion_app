package bootstrap

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPreparacionIncorporacionB2CoincideConPublicadorYComparteAudiencias(t *testing.T) {
	cfg, rutas := generarMaterialDesarrolloPrueba(t)
	ahora := time.Date(2026, 10, 2, 10, 0, 0, 0, time.UTC)
	c, err := NuevaComposicionSeguridadDesarrollo(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	m, err := nuevoMaterialAtestacionContratacionTemporalDesarrollo(c.derivadorIdempotencia, ahora)
	if err != nil {
		t.Fatal(err)
	}
	defer m.borrarCopiasEfimeras()
	publicadas := map[string][]byte{}
	catalogo := catalogoIncorporacionB2Prueba(t)
	if err := publicarMaterialIncorporacionB2ConDesarrollo(m, catalogo, func(x *materialAtestacionContratacionTemporalDesarrollo) error {
		publicadas[x.audienciaConsumo] = append([]byte(nil), x.claveHMAC...)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	claves, err := DerivarClavesIncorporacionB2DesdeMaterialDesarrollo(filepath.Dir(rutas.IdempotencyHMACConfig), ahora)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		for i := range claves {
			claves[i].Borrar()
		}
	}()
	if len(claves) != 23 {
		t.Fatal("faltan operaciones")
	}
	audiencias := map[string][]byte{}
	for _, x := range claves {
		secreto := x.CopiarSecreto()
		defer clear(secreto)
		if x.Version != 0 || x.RevisionGobierno != 0 {
			t.Fatal("la derivación inventó versiones de gobierno")
		}
		if previo, ok := audiencias[x.Audiencia]; ok && !bytes.Equal(previo, secreto) {
			t.Fatal("audiencia compartida con claves diferentes")
		}
		audiencias[x.Audiencia] = secreto
		if esperado, ok := publicadas[x.Audiencia]; ok && !bytes.Equal(esperado, secreto) {
			t.Fatal("clave distinta del publicador real")
		}
	}
	if len(audiencias) != 16 {
		t.Fatalf("audiencias distintas: %d", len(audiencias))
	}
	for _, secreto := range publicadas {
		clear(secreto)
	}
}

// Este ensayo usa una base desechable ya instalada desde main. Publica con
// las mismas funciones de vec-server y comprueba repetición durable; no
// sustituye el montaje HTTP ni concede perfiles de RRHH.
func TestPreparacionIncorporacionB2PublicadorRealPostgreSQL18(t *testing.T) {
	if os.Getenv("VEC_KITB2_TEST_DESECHABLE") != "si" {
		t.Skip("requiere clon PostgreSQL 18 desechable de main")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancelar()
	p, _, err := abrirPoolPostgreSQLContratacionTemporalDesarrollo(ctx, os.Getenv("VEC_KITB2_TEST_GOBIERNO_DSN"), "vec-kit-b2-publicador", rolGobiernoPostgreSQLContratacionTemporalDesarrollo)
	if err != nil {
		t.Fatal("pool nominal de gobierno rechazado")
	}
	defer p.Close()
	dir := os.Getenv("VEC_KITB2_TEST_MATERIAL_IDEMPOTENCIA")
	idem, err := cargarMaterialIdempotenciaDesarrollo(filepath.Dir(dir), filepath.Join(dir, "configuracion.json"))
	if err != nil {
		t.Fatal("material sintético rechazado")
	}
	defer idem.borrar()
	d, err := nuevoDerivadorIdentidadOperacionDesarrollo(&idem)
	if err != nil {
		t.Fatal(err)
	}
	defer d.borrar()
	m, err := nuevoMaterialAtestacionContratacionTemporalDesarrollo(d, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	defer m.borrarCopiasEfimeras()
	if err := publicarGobiernoAtestacionContratacionTemporalDesarrollo(ctx, p, &m); err != nil {
		t.Fatal("publicación base rechazada", err)
	}
	cat, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo(append(append(descriptoresMaterialAutorizacionContratacionTemporalDesarrollo(), descriptoresMaterialPersonalB2Desarrollo()...), descriptoresMaterialIncorporacionB2()...))
	if err != nil {
		t.Fatal(err)
	}
	publicar := func() {
		if _, err := publicarMaterialPersonalB2Desarrollo(ctx, p, m, cat); err != nil {
			t.Fatal("publicación Personal8 rechazada", err)
		}
		if err := publicarMaterialIncorporacionB2Desarrollo(ctx, p, m, cat); err != nil {
			t.Fatal("publicación incorporación9 rechazada", err)
		}
		for _, x := range descriptoresMaterialAutorizacionContratacionTemporalDesarrollo() {
			r, e := derivarMaterialConsumidorV3Desarrollo(m, x)
			if e != nil {
				t.Fatal(e)
			}
			e = publicarGobiernoAtestacionContratacionTemporalDesarrollo(ctx, p, &r)
			borrarBytes(r.claveHMAC)
			if e != nil {
				t.Fatal(e)
			}
		}
	}
	publicar()
	contar := func() string {
		tx, err := p.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(ctx)
		if _, err = tx.Exec(ctx, "SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario"); err != nil {
			t.Fatal(err)
		}
		var cuenta string
		if err := tx.QueryRow(ctx, `SELECT concat_ws('/',(SELECT count(*) FROM vec_autorizacion_atestada_v3.clave_capacidad_version),(SELECT count(*) FROM vec_autorizacion_atestada_v3.puntero_clave_emision))`).Scan(&cuenta); err != nil {
			t.Fatal(err)
		}
		return cuenta
	}
	antes := contar()
	publicar()
	if contar() != antes {
		t.Fatal("repetir publicador duplicó historia")
	}
	claves, err := DerivarClavesIncorporacionB2DesdeMaterialDesarrollo(dir, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		for i := range claves {
			claves[i].Borrar()
		}
	}()
	tx, err := p.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, "SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario"); err != nil {
		t.Fatal(err)
	}
	for _, c := range claves {
		var igual bool
		err := tx.QueryRow(ctx, `SELECT k.huella_secreto_sha256=$2 AND k.huella_gobierno_sha256=$3 AND k.audiencia_consumo=$4
 FROM vec_autorizacion_atestada_v3.puntero_clave_emision p JOIN vec_autorizacion_atestada_v3.clave_capacidad_version k USING(clave_id,version)
 WHERE k.clave_id=$1 ORDER BY p.orden DESC LIMIT 1`, c.ClaveID, c.SHA256, c.HuellaGobierno, c.Audiencia).Scan(&igual)
		if err != nil || !igual {
			t.Fatalf("clave %s: esperado=publicada coincidente observado=ausente o distinta", c.Capacidad)
		}
	}
}

func TestPreparacionIncorporacionB2MaterialAusenteFallaCerrado(t *testing.T) {
	for _, ruta := range []string{"", "relativo/idempotencia", "/no-existe/idempotencia"} {
		if c, err := DerivarClavesIncorporacionB2DesdeMaterialDesarrollo(ruta, time.Now()); err == nil || c != nil {
			t.Fatal("material ausente aceptado")
		}
	}
}
