package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"vec-diputacion-granada/internal/app/bootstrap"
)

func TestCLIProvisionUsuariosRechazaEntradaAntesDeLeerDSN(t *testing.T) {
	for _, args := range [][]string{nil, {"-plan", "relativo.json"}, {"-plan", "/inexistente-plan-sintetico.json"}, {"-desconocido"}} {
		var out bytes.Buffer
		codigo := ejecutar(args, &out, func(string) string { t.Fatal("se leyó DSN para entrada inválida"); return "" })
		if codigo != 2 || !strings.Contains(out.String(), "entrada_invalida") {
			t.Fatalf("salida incorrecta: %d %s", codigo, out.String())
		}
	}
}

func TestCLIUsuariosMotivosRechazaFaseFuenteOAprobacionSinLeerDSN(t *testing.T) {
	// Las claves de las entradas son parte del contrato cerrado de Usuarios.
	var entradas []bootstrap.EntradaMotivosUsuariosExterno
	for _, e := range []struct {
		tipo     string
		acciones []string
	}{
		{"consulta", []string{"vec.preferencias.consultar", "vec.imagen.consultar", "vec.correos.consultar"}},
		{"actualizacion", []string{"vec.preferencias.actualizar", "vec.imagen.actualizar", "vec.correos.anadir", "vec.correos.reenviar", "vec.correos.verificar", "vec.correos.activar", "vec.correos.retirar"}},
	} {
		h := sha256.Sum256([]byte("vec.ct.alta.desarrollo.v1\x00motivos_usuarios_propios_desarrollo\x00" + e.tipo))
		entradas = append(entradas, bootstrap.EntradaMotivosUsuariosExterno{Tipo: e.tipo, Clave: "motivo_" + hex.EncodeToString(h[:16]), ModuloID: "usuarios", Acciones: e.acciones, Etiquetas: map[string]string{"xx": "etiqueta de prueba"}})
	}
	s := bootstrap.SolicitudMotivosUsuariosExterno{AprobacionRef: "aprobacion:prueba", SecuenciaEsperada: 37, Catalogo: bootstrap.CatalogoMotivosUsuariosExterno{CatalogoID: "motivos_usuarios_propios_desarrollo", Version: 1, PublicadoEn: time.Date(2026, 9, 30, 1, 0, 0, 0, time.UTC), Entradas: entradas}}
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "plan.json")
	if err := os.WriteFile(p, b, 0o600); err != nil {
		t.Fatal(err)
	}
	var preview bytes.Buffer
	if codigo := ejecutar([]string{"-plan", p, "-fase", "motivos"}, &preview, func(string) string { t.Fatal("preview leyó DSN"); return "" }); codigo != 0 || !strings.Contains(preview.String(), "preparado") {
		t.Fatalf("preview puro falló: %d %s", codigo, preview.String())
	}
	for _, fase := range []string{"motivos", "autorizacion", "contexto", "fase_ajena"} {
		var out bytes.Buffer
		codigo := ejecutar([]string{"-plan", p, "-fase", fase, "-publicar"}, &out, func(string) string { t.Fatal("se leyó una conexión para fuente rechazada"); return "" })
		if codigo != 2 {
			t.Fatalf("fase %s aceptó fuente incompatible: %d", fase, codigo)
		}
		if fase == "motivos" && !strings.Contains(out.String(), "aprobacion_divergente") {
			t.Fatal("no comprobó aprobación antes de la conexión")
		}
	}
	var cambiado bytes.Buffer
	if codigo := ejecutar([]string{"-plan", p, "-fase", "motivos", "-publicar", "-huella-aprobada", strings.Repeat("a", 64), "-aprobacion-ref", s.AprobacionRef}, &cambiado, func(string) string { t.Fatal("huella incorrecta leyó DSN"); return "" }); codigo != 2 || !strings.Contains(cambiado.String(), "aprobacion_divergente") {
		t.Fatal("aprobación de otra huella admitida")
	}
}

type salidaFallida struct{}

func (salidaFallida) Write([]byte) (int, error) { return 0, errors.New("salida interrumpida") }

func TestCLIProvisionUsuariosPropagaFalloDeSalida(t *testing.T) {
	if codigo := fallo(salidaFallida{}, "entrada_invalida", 2); codigo != 1 {
		t.Fatalf("fallo de salida oculto: %d", codigo)
	}
}

func TestCLIProvisionUsuariosTLSVerificaServidorSinFallbackPlano(t *testing.T) {
	for _, modo := range []string{"disable", "allow", "prefer", "require", "verify-ca", "verify-full"} {
		cfg, err := pgx.ParseConfig("host=localhost user=sintetico dbname=sintetica sslmode=" + modo)
		if err != nil {
			t.Fatal(err)
		}
		if tlsVerificado(cfg) != (modo == "verify-full") {
			t.Fatalf("modo TLS %s admitido incorrectamente", modo)
		}
	}
}
