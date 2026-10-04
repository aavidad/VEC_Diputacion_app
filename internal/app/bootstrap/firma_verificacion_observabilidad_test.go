package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/vec/adapters/observabilidad"
	"vec-diputacion-granada/internal/vec/documentos/adapters/validadorautofirma/servidorprueba"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func TestVerificadorCompuestoEmiteResultadoMinimoConCorrelacionInterna(t *testing.T) {
	for _, caso := range []struct {
		nombre    string
		escenario servidorprueba.Escenario
		resultado string
	}{
		{"respuesta_5xx", servidorprueba.ErrorInterno, "no_disponible"},
		{"dictamen_ligado", servidorprueba.ValidaSinSello, "correcto"},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			servidor := servidorprueba.Nuevo(strings.Repeat("t", 40), false)
			defer servidor.Close()
			servidor.Escenario(caso.escenario)
			dir := t.TempDir()
			guardar := func(nombre string, b []byte) string {
				p := filepath.Join(dir, nombre)
				if err := os.WriteFile(p, b, 0600); err != nil {
					t.Fatal(err)
				}
				return p
			}
			cfg := config.Config{FirmaVerificacionEnabled: "true", FirmaVerificacionURL: servidor.URL,
				FirmaVerificacionCAFile:    guardar("ca.pem", pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: servidor.Certificate().Raw})),
				FirmaVerificacionTokenFile: guardar("token", []byte(strings.Repeat("t", 40))), FirmaVerificacionTimeout: "5s",
				DocumentosEnabled: "true", ExecutionProfile: config.ExecutionProfileDevelopment, AuthMode: config.AuthModeDevelopment,
				DevelopmentGuard: config.DevelopmentGuardAcknowledgement}
			var lineas bytes.Buffer
			var emisor *observabilidad.EmisorJSONLines
			verificador, err := nuevoVerificadorFirmaDocumentos(cfg, func() vecports.EmisorResultadosTecnicosConContexto {
				if emisor == nil {
					return nil
				}
				return emisor
			})
			if err != nil || verificador == nil {
				t.Fatalf("constructor: %v", err)
			}
			emisor, err = observabilidad.NuevoEmisorJSONLines(observabilidad.OpcionesEmisor{Destino: &lineas, Entorno: "pruebas"})
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := verificador.(docports.VerificadorFirmasDocumento); !ok {
				t.Fatal("el puerto multifirma quedó oculto")
			}
			ctx, err := vecports.ConCorrelacionIncidenciasPeticion(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			original := []byte("original sintetico")
			sha := sha256.Sum256(original)
			peticion := docports.SolicitudVerificacionFirma{DocumentoID: "ref:" + strings.Repeat("1", 64), Version: 1,
				HuellaOriginalSHA256: hex.EncodeToString(sha[:]), ContenidoOriginal: original,
				ContenidoFirmado: append(bytes.Clone(original), []byte("revision sintetica firmada")...)}
			if _, err := verificador.VerificarMotivado(ctx, peticion); err != nil {
				t.Fatal(err)
			}
			cierre, cancelar := context.WithTimeout(context.Background(), time.Second)
			defer cancelar()
			if err := emisor.Cerrar(cierre); err != nil {
				t.Fatal(err)
			}
			partes := bytes.Split(bytes.TrimSpace(lineas.Bytes()), []byte{'\n'})
			if len(partes) != 1 {
				t.Fatalf("resultados=%d", len(partes))
			}
			var salida map[string]any
			if json.Unmarshal(partes[0], &salida) != nil || salida["resultado"] != caso.resultado || salida["componente"] != "grxfirma" ||
				salida["etapa"] != "peticion" || salida["correlacion"] == "" || salida["correlacion_ref"] == "" || len(salida) != 10 {
				t.Fatal("resultado técnico o correlación incorrectos")
			}
			for _, privado := range []string{"original sintetico", peticion.DocumentoID, servidor.URL, "actor_ref", "expediente_ref"} {
				if bytes.Contains(partes[0], []byte(privado)) {
					t.Fatal("resultado técnico expone datos")
				}
			}
		})
	}
}
