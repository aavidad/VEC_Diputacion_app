package validadorautofirma

import (
	"context"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/documentos/ports"
)

func corpusV2(t *testing.T, nombre string) ([]byte, ports.SolicitudVerificacionFirma) {
	t.Helper()
	leer := func(f string) []byte {
		b, err := os.ReadFile(filepath.Join("testdata", "dictamen-v2", nombre, f))
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	original, firmado := leer("original.pdf"), leer("firmado.pdf")
	return leer("dictamen-esperado.json"), ports.SolicitudVerificacionFirma{
		DocumentoID: "ref:" + strings.Repeat("1", 64), Version: 1, FormatoEsperado: "PAdES",
		HuellaOriginalSHA256: huellaBytesV2(original), ContenidoOriginal: original, ContenidoFirmado: firmado,
	}
}

func clienteCorpusV2(t *testing.T, cuerpo []byte) *Cliente {
	t.Helper()
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v2/verify" || r.Header.Get("Authorization") != "Bearer "+strings.Repeat("t", 40) {
			t.Error("ruta, metodo o credencial distintos del contrato")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		var p map[string]json.RawMessage
		if json.NewDecoder(r.Body).Decode(&p) != nil || string(p["contrato_solicitado"]) != `"autofirmav2.dictamen-verificacion.v2"` || len(p) != 4 {
			t.Error("peticion sin contrato v2 exacto o minimizacion")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(cuerpo)
	}))
	t.Cleanup(s.Close)
	c, err := Nuevo(Configuracion{URL: s.URL, CAPEM: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: s.Certificate().Raw}), Token: []byte(strings.Repeat("t", 40))})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func envolturaV2(dictamen []byte) []byte {
	// Los indicadores deliberadamente opuestos no pueden cambiar el estado.
	var d struct {
		Estado string `json:"estado"`
	}
	_ = json.Unmarshal(dictamen, &d)
	valid := "false"
	if d.Estado != "valida" {
		valid = "true"
	}
	return []byte(`{"ok":false,"valid":` + valid + `,"reason":"texto_ignorado","dictamen":` + string(dictamen) + `}`)
}

func TestCorpusGrxfirmaV2VeinteCasos(t *testing.T) {
	dirs, err := filepath.Glob("testdata/dictamen-v2/[0-9]*")
	if err != nil || len(dirs) != 20 {
		t.Fatalf("corpus incompleto: %d, %v", len(dirs), err)
	}
	for _, dir := range dirs {
		nombre := filepath.Base(dir)
		t.Run(nombre, func(t *testing.T) {
			raw, s := corpusV2(t, nombre)
			var esperado dictamenV2
			if json.Unmarshal(raw, &esperado) != nil {
				t.Fatal("fixture invalida")
			}
			c := clienteCorpusV2(t, envolturaV2(raw))
			r, err := c.VerificarFirmas(context.Background(), s)
			if err != nil {
				t.Fatal(err)
			}
			if nombre == "12_dss" || nombre == "16_dss_valido" || nombre == "18_doctimestamp_valido" || nombre == "19_doctimestamp_imprint_ajeno" || nombre == "20_lta_completo" {
				// Defecto conocido del proveedor: DSS y DocTimeStamp tienen
				// categorias ausentes del esquema. No se amplian catalogos.
				if r.Estado != ports.EstadoVerificacionIndeterminada || r.Motivo != ports.MotivoRespuestaNoInterpretable || len(r.Firmas) != 0 {
					t.Fatalf("DSS fuera de esquema aceptado: %+v", r)
				}
				return
			}
			if string(r.Estado) != esperado.Estado || len(r.Firmas) != len(esperado.Firmas) || r.Motivo.EstadoAsociado() != r.Estado {
				t.Fatalf("estado/revisiones distintos: recibido %+v; esperado %s %s (%d)", r, esperado.Estado, esperado.Motivo, len(esperado.Firmas))
			}
			for i, f := range esperado.Firmas {
				p := r.Firmas[i]
				if p.Orden != f.Orden || p.RevisionLongitud != uint64(f.RevisionLongitud) || p.RevisionHuellaSHA256 != f.RevisionHuellaSHA256 || p.ContenidoFirmadoHuellaSHA256 != f.ContenidoFirmadoHuellaSHA256 || p.CertificadoHuellaSHA256 != f.CertificadoHuellaSHA256 || p.TipoFirma != f.TipoFirma {
					t.Fatal("perdida de revision o certificado")
				}
			}
			if r.HuellaOriginalSHA256 != s.HuellaOriginalSHA256 || r.HuellaFirmadoSHA256 != huellaBytesV2(s.ContenidoFirmado) || r.ComprobadoEn.IsZero() {
				t.Fatal("enlace o fecha perdida")
			}
		})
	}
}

func TestV2NoReduceDosFirmasAUnFirmante(t *testing.T) {
	raw, s := corpusV2(t, "02_dos_firmas")
	c := clienteCorpusV2(t, envolturaV2(raw))
	varios, err := c.VerificarFirmas(context.Background(), s)
	if err != nil || varios.Estado != ports.EstadoVerificacionValida || len(varios.Firmas) != 2 || varios.Firmas[0].FirmanteRef == varios.Firmas[1].FirmanteRef {
		t.Fatalf("dos certificados perdidos: %+v %v", varios, err)
	}
	uno, err := c.VerificarMotivado(context.Background(), s)
	if err != nil || uno.Motivo != ports.MotivoFirmanteNoIdentificado || uno.Resultado.Estado != ports.EstadoVerificacionIndeterminada || uno.Resultado.FirmanteRef != "" {
		t.Fatalf("multifirma reducida: %+v %v", uno, err)
	}
}

func TestV2NoAdoptaRevisionOriginalNiOrdenSustituidos(t *testing.T) {
	raw, s := corpusV2(t, "02_dos_firmas")
	mutaciones := map[string]func(map[string]any){
		"mas de veinte firmas": func(d map[string]any) {
			f := d["firmas"].([]any)[0]
			lista := make([]any, 21)
			for i := range lista {
				lista[i] = f
			}
			d["firmas"] = lista
		},
		"firmas nulas":    func(d map[string]any) { d["firmas"] = nil },
		"v1 silenciosa":   func(d map[string]any) { d["contrato"] = contratoDictamenV1 },
		"orden invertido": func(d map[string]any) { fs := d["firmas"].([]any); fs[0], fs[1] = fs[1], fs[0] },
		"orden repetido":  func(d map[string]any) { d["firmas"].([]any)[1].(map[string]any)["orden"] = float64(1) },
		"revision equivocada": func(d map[string]any) {
			d["firmas"].([]any)[0].(map[string]any)["revisionHuellaSHA256"] = strings.Repeat("a", 64)
		},
		"contenido firmado equivocado": func(d map[string]any) {
			d["firmas"].([]any)[1].(map[string]any)["contenidoFirmadoHuellaSHA256"] = strings.Repeat("a", 64)
		},
		"original equivocado": func(d map[string]any) { d["huellaOriginalSHA256"] = strings.Repeat("a", 64) },
		"original omitido":    func(d map[string]any) { delete(d, "huellaOriginalSHA256") },
		"byterange desbordado": func(d map[string]any) {
			d["firmas"].([]any)[1].(map[string]any)["byteRange"] = []any{0, 1, 9223372036854775807, 1}
		},
		"revision truncada":    func(d map[string]any) { d["firmas"].([]any)[1].(map[string]any)["revisionLongitud"] = float64(1) },
		"campo personal ajeno": func(d map[string]any) { d["firmas"].([]any)[0].(map[string]any)["campo"] = "persona ajena" },
		"clave plegada":        func(d map[string]any) { d["ESTADO"] = d["estado"]; delete(d, "estado") },
		"cubre omitido": func(d map[string]any) {
			delete(d["firmas"].([]any)[0].(map[string]any), "cubreDocumentoCompletoHastaAqui")
		},
		"incertidumbre ocultada": func(d map[string]any) { d["cambiosPosteriores"].(map[string]any)["estado"] = "no_comprobados" },
		"revocado ocultado": func(d map[string]any) {
			d["firmas"].([]any)[1].(map[string]any)["revocacion"].(map[string]any)["estado"] = "revocado"
		},
	}
	for nombre, mutar := range mutaciones {
		t.Run(nombre, func(t *testing.T) {
			var d map[string]any
			if json.Unmarshal(raw, &d) != nil {
				t.Fatal("fixture invalida")
			}
			mutar(d)
			b, err := json.Marshal(d)
			if err != nil {
				t.Fatal(err)
			}
			r, err := clienteCorpusV2(t, envolturaV2(b)).VerificarFirmas(context.Background(), s)
			if err != nil || r.Estado != ports.EstadoVerificacionIndeterminada || r.Motivo != ports.MotivoRespuestaNoInterpretable || len(r.Firmas) != 0 {
				t.Fatalf("evidencia sustituida adoptada: %+v %v", r, err)
			}
		})
	}
}

func TestV2RechazaDuplicadosTamanosYProfundidad(t *testing.T) {
	raw, _ := corpusV2(t, "01_una_firma")
	b := string(envolturaV2(raw))
	for _, cuerpo := range []string{
		strings.Replace(b, `"estado": "valida"`, `"estado": "valida", "ESTADO":"valida"`, 1),
		strings.Replace(b, `"firmas": [`, `"firmas": null, "firmas": [`, 1),
		b + `{}`, b + `0`, `{"DICTAMEN":` + string(raw) + `}`,
		`{"ignored":` + strings.Repeat("[", maximaProfundidad+1) + strings.Repeat("]", maximaProfundidad+1) + `}`,
		`{"ignored":"` + strings.Repeat("x", maximaRespuesta) + `"}`,
	} {
		if _, err := decodificarRespuestaV2([]byte(cuerpo)); err == nil {
			t.Fatal("JSON ambiguo o excesivo aceptado")
		}
	}
}

func TestV2FormatoEsperadoNoAdoptaPositivoAjeno(t *testing.T) {
	raw, s := corpusV2(t, "01_una_firma")
	s.FormatoEsperado = "CAdES"
	r, err := clienteCorpusV2(t, envolturaV2(raw)).VerificarMotivado(context.Background(), s)
	if err != nil || r.Motivo != ports.MotivoRespuestaNoInterpretable || r.Resultado.Formato != "" || r.Resultado.FirmanteRef != "" {
		t.Fatalf("formato ajeno adoptado: %+v %v", r, err)
	}
}

func TestV2NoTransportaDatosDescriptivosDelCertificado(t *testing.T) {
	raw, s := corpusV2(t, "02_dos_firmas")
	r, err := clienteCorpusV2(t, envolturaV2(raw)).VerificarFirmas(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	for _, fragmento := range []string{"CN=", "asunto", "emisor", "serie", "campo", "firmante-a", "firmante-b"} {
		if strings.Contains(string(b), fragmento) {
			t.Fatalf("dato descriptivo propagado: %s", fragmento)
		}
	}
}

func TestV2ConservaErrorDeParseoHastaLaFrontera(t *testing.T) {
	var fecha *time.ParseError
	if err := cumpleEsquemaV2("fecha-no-admisible", &esquemaV2{Tipo: "string", Formato: "date-time"}, 0); !errors.As(err, &fecha) {
		t.Fatalf("causa de fecha no conservada: %T", err)
	}
	var numero *strconv.NumError
	if err := cumpleEsquemaV2(json.Number("1.5"), &esquemaV2{Tipo: "integer"}, 0); !errors.As(err, &numero) {
		t.Fatalf("causa de entero no conservada: %T", err)
	}
	raw, s := corpusV2(t, "01_una_firma")
	var d map[string]any
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatal(err)
	}
	d["comprobadoEn"] = "fecha-no-admisible"
	raw, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = decodificarRespuestaV2(envolturaV2(raw)); !errors.As(err, &fecha) {
		t.Fatalf("causa perdida en el decodificador: %T", err)
	}
	r, err := clienteCorpusV2(t, envolturaV2(raw)).VerificarFirmas(context.Background(), s)
	if err != nil || r.Estado != ports.EstadoVerificacionIndeterminada || r.Motivo != ports.MotivoRespuestaNoInterpretable || len(r.Firmas) != 0 {
		t.Fatalf("causa sin traduccion nominal: estado=%s motivo=%s error=%v", r.Estado, r.Motivo, err)
	}
}
