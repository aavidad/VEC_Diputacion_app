package simulacionbaremo

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

	ce "vec-diputacion-granada/internal/modules/bolsa/domain/calculoexperiencia"
	rb "vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
)

func fixture(t *testing.T, n string) ([]byte, string) {
	t.Helper()
	contenido, err := os.ReadFile(filepath.Join("testdata", n+".json"))
	if err != nil {
		t.Fatal(err)
	}
	huella, err := os.ReadFile(filepath.Join("testdata", n+".sha256"))
	if err != nil {
		t.Fatal(err)
	}
	return contenido, strings.TrimSpace(string(huella))
}
func solicitudPrueba(t *testing.T, reglas, entrada string) Solicitud {
	t.Helper()
	r, hr := fixture(t, reglas)
	e, he := fixture(t, entrada)
	return Solicitud{r, hr, e, he}
}
func huellaPrueba(v []byte) string { s := sha256.Sum256(v); return hex.EncodeToString(s[:]) }
func TestConvocatoriasConfiguranPuntuacionYConservanReproduccion(t *testing.T) {
	var anteriores []byte
	var convocatoriaAnterior string
	for _, caso := range []struct {
		reglas  string
		version uint64
		total   int64
	}{{"reglas_a", 1, 101667}, {"reglas_b", 3, 250000}} {
		t.Run(caso.reglas, func(t *testing.T) {
			solicitud := solicitudPrueba(t, caso.reglas, "entrada")
			servicio := Servicio{}
			primera, err := servicio.Simular(solicitud)
			if err != nil {
				t.Fatal(err)
			}
			segunda, err := servicio.Simular(solicitud)
			if err != nil {
				t.Fatal(err)
			}
			total, presente := primera.Resultado().Total()
			if !presente || total.Micropuntos() != caso.total {
				t.Fatalf("total=%d presente=%t", total.Micropuntos(), presente)
			}
			if primera.Resultado().Estado() != ce.ResultadoExperienciaCompletado {
				t.Fatal("no completado")
			}
			if primera.Resultado().Vinculos().Conjunto().Version() != caso.version {
				t.Fatal("version no conservada")
			}
			if !bytes.Equal(primera.RepresentacionCanonica(), segunda.RepresentacionCanonica()) {
				t.Fatal("replay diferente")
			}
			var envoltorio struct {
				Esquema         string          `json:"esquema"`
				Alcance         string          `json:"alcance"`
				ConvocatoriaRef string          `json:"convocatoria_ref"`
				Huella          string          `json:"huella_resultado_sha256"`
				Resultado       json.RawMessage `json:"resultado"`
			}
			if err := json.Unmarshal(primera.RepresentacionCanonica(), &envoltorio); err != nil {
				t.Fatal(err)
			}
			if envoltorio.Esquema != "vec.bolsa.simulacion_experiencia.v1" || envoltorio.Alcance != "simulacion" {
				t.Fatal("alcance equivoco")
			}
			if envoltorio.Huella != huellaPrueba(envoltorio.Resultado) || envoltorio.Huella != primera.HuellaResultadoSHA256() {
				t.Fatal("huella resultado incorrecta")
			}
			conjunto, err := rb.RestaurarConjuntoReglasBaremo(solicitud.ConjuntoCanonico)
			if err != nil {
				t.Fatal(err)
			}
			if envoltorio.ConvocatoriaRef != conjunto.Identidad().ConvocatoriaRef() {
				t.Fatal("convocatoria incorrecta")
			}
			if _, err := ce.RestaurarResultadoExperienciaV1ConHuellaSHA256(envoltorio.Resultado, envoltorio.Huella); err != nil {
				t.Fatal(err)
			}
			if bytes.Equal(anteriores, envoltorio.Resultado) || convocatoriaAnterior == envoltorio.ConvocatoriaRef {
				t.Fatal("convocatorias confundidas")
			}
			anteriores = append([]byte(nil), envoltorio.Resultado...)
			convocatoriaAnterior = envoltorio.ConvocatoriaRef
			copia := primera.RepresentacionCanonica()
			copia[0] = 'x'
			if primera.RepresentacionCanonica()[0] != '{' {
				t.Fatal("salida mutable")
			}
		})
	}
}
func TestBloqueoNoEquivaleACero(t *testing.T) {
	simulacion, err := (Servicio{}).Simular(solicitudPrueba(t, "reglas_a", "entrada_bloqueada"))
	if err != nil {
		t.Fatal(err)
	}
	if simulacion.Resultado().Estado() != ce.ResultadoExperienciaBloqueado || len(simulacion.Resultado().Bloqueos()) == 0 {
		t.Fatal("bloqueo perdido")
	}
	if _, presente := simulacion.Resultado().Total(); presente {
		t.Fatal("bloqueo con total")
	}
	if bytes.Contains(simulacion.RepresentacionCanonica(), []byte(`"total"`)) {
		t.Fatal("total serializado en bloqueo")
	}
}
func TestHuellasYBytesInvalidosNuncaDevuelvenResultado(t *testing.T) {
	for _, caso := range []struct {
		nombre, fase string
		alterar      func(*Solicitud)
	}{
		{"huella_reglas", "conjunto", func(s *Solicitud) { s.HuellaConjuntoSHA256 = strings.Repeat("0", 64) }},
		{"huella_entrada", "entrada", func(s *Solicitud) { s.HuellaEntradaSHA256 = strings.Repeat("0", 64) }},
		{"reglas_no_canonicas", "conjunto", func(s *Solicitud) {
			s.ConjuntoCanonico = append(s.ConjuntoCanonico, '\n')
			s.HuellaConjuntoSHA256 = huellaPrueba(s.ConjuntoCanonico)
		}},
		{"entrada_no_canonica", "entrada", func(s *Solicitud) {
			s.EntradaCanonica = append(s.EntradaCanonica, '\n')
			s.HuellaEntradaSHA256 = huellaPrueba(s.EntradaCanonica)
		}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			s := solicitudPrueba(t, "reglas_a", "entrada")
			caso.alterar(&s)
			r, err := (Servicio{}).Simular(s)
			var fallo *Error
			if !errors.As(err, &fallo) || fallo.Fase != caso.fase {
				t.Fatalf("error=%v", err)
			}
			if len(r.RepresentacionCanonica()) != 0 || r.HuellaResultadoSHA256() != "" || r.Resultado().Estado() != "" {
				t.Fatal("fallo tecnico con resultado")
			}
		})
	}
}
