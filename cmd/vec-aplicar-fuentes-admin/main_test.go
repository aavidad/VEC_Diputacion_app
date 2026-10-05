package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func argsFixture(t *testing.T, p documento) ([]string, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "privado")
	if os.Mkdir(dir, 0700) != nil {
		t.Fatal("directorio")
	}
	write := func(name string, v any) string {
		t.Helper()
		b, e := json.Marshal(v)
		if e != nil {
			t.Fatal(e)
		}
		path := filepath.Join(dir, name)
		if os.WriteFile(path, b, 0600) != nil {
			t.Fatal("archivo")
		}
		return path
	}
	plan := write("plan.json", p)
	conexion := write("conexion.json", conexionPrivada{DSN: "postgres://operador:secreto@database.example.invalid/sintetica?sslmode=verify-full", PermitirSocketDesarrollo: false})
	aprobacion := write("aprobacion.json", aprobacionPrivada{p.HuellaPlanSHA256})
	textos, e := filepath.Abs("../../web/static/textos/es/admin-fuentes-aplicar.json")
	if e != nil {
		t.Fatal(e)
	}
	acuse := filepath.Join(dir, "acuse.json")
	return []string{"--plan", plan, "--conexion", conexion, "--aprobacion", aprobacion, "--acuse", acuse, "--textos", textos, "--timeout", "5s"}, acuse
}
func TestCLIAcuseTrasCommitYCadaInvocacionDestinoNuevo(t *testing.T) {
	p := planFixture(t)
	for _, estado := range []string{"permitido", "denegado", "error"} {
		t.Run(estado, func(t *testing.T) {
			args, path := argsFixture(t, p)
			tx := &txFixture{respuesta: envelopeFixture(t, p, estado, estado == "permitido")}
			llamadas := 0
			abrir := func(context.Context, conexionPrivada, time.Duration) (transaccion, error) {
				llamadas++
				b, e := os.ReadFile(path)
				if e != nil || len(b) != 0 {
					t.Fatal("acuse no reservado vacío antesDB")
				}
				return tx, nil
			}
			var salida, errores bytes.Buffer
			exit := ejecutar(args, &salida, &errores, abrir)
			esperado := 0
			if estado != "permitido" {
				esperado = 1
			}
			if exit != esperado || errores.Len() != 0 || llamadas != 1 {
				t.Fatal("resultado", exit, errores.String())
			}
			b, e := os.ReadFile(path)
			if e != nil || !bytes.Equal(b, append(append([]byte{}, tx.respuesta...), '\n')) {
				t.Fatal("acuse cambia respuesta original")
			}
			info, e := os.Stat(path)
			if e != nil || info.Mode().Perm() != 0600 {
				t.Fatal("acuse público")
			}
			var d diagnostico
			if json.Unmarshal(salida.Bytes(), &d) != nil || !d.Confirmado || !d.AcuseGuardado || d.Estado != estado {
				t.Fatal("estado no confirmado")
			}
			if strings.Contains(salida.String(), "secreto") || strings.Contains(salida.String(), "persona_ref") || strings.Contains(salida.String(), "operador_sintetico") {
				t.Fatal("expone contenido privado")
			}
			if ejecutar(args, &salida, &errores, abrir) == 0 || llamadas != 1 {
				t.Fatal("destino existente enviadoDB")
			}
		})
	}
}
func TestCLICommitInciertoSinAcuseNiReintento(t *testing.T) {
	p := planFixture(t)
	args, path := argsFixture(t, p)
	tx := &txFixture{respuesta: envelopeFixture(t, p, "permitido", false), fallar: "commit"}
	abrir := func(context.Context, conexionPrivada, time.Duration) (transaccion, error) { return tx, nil }
	var salida, errores bytes.Buffer
	if ejecutar(args, &salida, &errores, abrir) != 2 || salida.Len() != 0 {
		t.Fatal("éxito incierto")
	}
	var d diagnostico
	if json.Unmarshal(errores.Bytes(), &d) != nil || d.Estado != "indeterminado" || d.Confirmado || d.AcuseGuardado {
		t.Fatal("no indica incertidumbre")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("guarda recibo no confirmado")
	}
	if strings.Contains(errores.String(), "secreto") {
		t.Fatal("expone errorSQL")
	}
}
func TestCLIFalloConsultaSinRecibo(t *testing.T) {
	p := planFixture(t)
	args, path := argsFixture(t, p)
	tx := &txFixture{fallar: "consulta"}
	abrir := func(context.Context, conexionPrivada, time.Duration) (transaccion, error) { return tx, nil }
	var salida, errores bytes.Buffer
	if ejecutar(args, &salida, &errores, abrir) != 1 || salida.Len() != 0 || strings.Contains(errores.String(), "secreto") {
		t.Fatal("fallo no cerrado")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("conserva falso acuse")
	}
}
func TestCLIRechazaJSONAbiertoAntesDB(t *testing.T) {
	p := planFixture(t)
	args, path := argsFixture(t, p)
	b, e := os.ReadFile(args[3])
	if e != nil {
		t.Fatal(e)
	}
	b = bytes.Replace(b, []byte(`"dsn":`), []byte(`"perfil":"admin","dsn":`), 1)
	if os.WriteFile(args[3], b, 0600) != nil {
		t.Fatal("archivo")
	}
	llamadas := 0
	abrir := func(context.Context, conexionPrivada, time.Duration) (transaccion, error) {
		llamadas++
		return nil, nil
	}
	var salida, errores bytes.Buffer
	if ejecutar(args, &salida, &errores, abrir) != 1 || llamadas != 0 {
		t.Fatal("acepta perfil cliente")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("crea acuse de entrada invalida")
	}
}
