package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestConexionSoloClonLocalSinOpcionesAjenas(t *testing.T) {
	valida := "postgresql://operador@127.0.0.1:55441/sintetica?sslmode=disable"
	p := planPrueba()
	c, err := configuracionConexion([]byte(valida), p, nil)
	if err != nil || c.Host != "127.0.0.1" || c.Database != "sintetica" || len(c.Fallbacks) != 0 {
		t.Fatal("clon rechazado")
	}
	for _, s := range []string{
		"postgresql://operador@cidonia:5432/sintetica?sslmode=disable",
		"postgresql://operador@127.0.0.1:55441/otra?sslmode=disable",
		valida + "&host=cidonia", valida + "&sslkey=privada", valida + "&sslmode=disable",
		"host=127.0.0.1 dbname=sintetica", "postgresql://operador@localhost:55441/sintetica?sslmode=disable",
	} {
		if _, err := configuracionConexion([]byte(s), p, nil); err == nil {
			t.Fatal("destino u opciones ajenas aceptados")
		}
	}
}

func TestPlanExigePreimagenPorPersonaYUnaSolaOcurrencia(t *testing.T) {
	b, _ := json.Marshal(planPrueba())
	p := string(b)
	if _, err := cargarPlan([]byte(p), "inventario"); err != nil {
		t.Fatal(err)
	}
	if _, err := cargarPlan([]byte(p), "aplicar"); err == nil {
		t.Fatal("CAS ausente aceptado")
	}
	p = strings.Replace(p, `"preimagen_sha256":""`, `"preimagen_sha256":"`+strings.Repeat("a", 64)+`"`, 1)
	if _, err := cargarPlan([]byte(p), "ensayo"); err != nil {
		t.Fatal(err)
	}
	if _, err := cargarPlan([]byte(p+`{}`), "ensayo"); err == nil {
		t.Fatal("contenido sobrante aceptado")
	}
	duplicadoCampo := strings.Replace(p, `"base":"sintetica"`, `"base":"sintetica","base":"otra"`, 1)
	if _, err := cargarPlan([]byte(duplicadoCampo), "ensayo"); err == nil {
		t.Fatal("campo repetido aceptado")
	}
	duplicado := strings.Replace(p, `}]}`, `},{"persona_ref":"per_`+strings.Repeat("a", 22)+`","preimagen_sha256":"`+strings.Repeat("a", 64)+`"}]}`, 1)
	if _, err := cargarPlan([]byte(duplicado), "ensayo"); err == nil {
		t.Fatal("persona repetida aceptada")
	}
}

func TestValidacionJSONPropagaEntradaInvalidaSinDetalles(t *testing.T) {
	for _, entrada := range []string{
		"", `{"campo":`, `{"campo"`, `{"campo":"material-sintetico"`,
		`{"campo":1,"campo":2}`, `{"objeto":{"campo":1,"campo":2}}`,
		`[1}`, `{} {}`, strings.Repeat("[", 33) + "0" + strings.Repeat("]", 33),
	} {
		if err := validarJSONSinDuplicados([]byte(entrada)); err != errEntrada {
			t.Fatal("entrada inválida sin rechazo nominal")
		}
		if _, err := cargarPlan([]byte(entrada), "inventario"); err != errEntrada {
			t.Fatal("rechazo no propagado al plan")
		}
	}
	for _, entrada := range []string{
		`{"objeto":{"campo":[1,true,null]}}`,
		strings.Repeat("[", 32) + "0" + strings.Repeat("]", 32),
	} {
		if err := validarJSONSinDuplicados([]byte(entrada)); err != nil {
			t.Fatal("JSON válido rechazado", err)
		}
	}
}

func planPrueba() plan {
	p := plan{Esquema: "usuarios.correos.reclaveado.plan.v1", Base: "sintetica", Sistema: "123456789", Lote: "lote:reclaveado:01", Aprobacion: "aprobacion:local:01", Personas: []objetivo{{"per_" + strings.Repeat("a", 22), ""}}, Conexion: destinoConexion{Host: "127.0.0.1", Puerto: 55441, Usuario: "operador", SSLMode: "disable", ClonLocal: true}}
	p.ConexionHuella = huellaConexion(p.Base, p.Conexion)
	return p
}

func TestEntornoPGNoInfluye(t *testing.T) {
	t.Setenv("PGSERVICEFILE", "inexistente")
	if _, err := configuracionConexion([]byte("postgresql://operador@127.0.0.1:55441/sintetica?sslmode=disable"), planPrueba(), nil); err == nil {
		t.Fatal("entorno PG aceptado")
	}
}

func TestDescriptorRechazaPermisosAmpliosYTamano(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "entrada")
	if err := os.WriteFile(ruta, []byte("secreto sintético"), 0600); err != nil {
		t.Fatal(err)
	}
	// leerDescriptor asume la propiedad del descriptor y lo cierra. Abrirlo
	// sin os.File evita un segundo cierre de su finalizador tras reutilizarlo.
	abrir := func() int {
		fd, err := syscall.Open(ruta, syscall.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
		if err != nil {
			t.Fatal(err)
		}
		return fd
	}
	if _, err := leerDescriptor(abrir(), 32); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(ruta, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := leerDescriptor(abrir(), 32); err == nil {
		t.Fatal("lectura pública aceptada")
	}
	if _, err := leerDescriptor(0, 32); err == nil {
		t.Fatal("stdin aceptado")
	}
	if err := os.Chmod(ruta, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := leerDescriptor(abrir(), 4); err == nil {
		t.Fatal("entrada sin límite aceptada")
	}
}
