package application

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func TestOriginalSubidaLocalAcreditaSoloBytesAdmitidosDeSuTitular(t *testing.T) {
	contenido := []byte("%PDF-1.4\noriginal propio\n%%EOF")
	suma := sha256.Sum256(contenido)
	huella := hex.EncodeToString(suma[:])
	persona := "per_0123456789abcdefghijkl"
	documento := "ref:" + strings.Repeat("a", 64)
	config := ConfiguracionServicioSubidaLocal{ModuloID: "meritos", TipoRecurso: "hecho",
		Finalidad: "custodiar_evidencia_meritos"}
	base := docports.InstantaneaSubidaLocal{
		Resultado: docports.ResultadoSubidaLocal{CargaID: "ref:" + strings.Repeat("b", 64),
			ComprobanteRef: documento, DocumentoVersion: 1, ReciboRef: "ref:" + strings.Repeat("c", 64),
			Estado: docports.EstadoSubidaLocalAdmitida, RegistradoEn: time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)},
		RecursoRef: "ref:" + strings.Repeat("d", 64), PersonaRef: persona,
		HuellaPDFSHA256: huella, MIME: "application/pdf", Tamano: int64(len(contenido)),
		Analisis: &vecports.ResultadoAnalisisContenido{Estado: vecports.EstadoAnalisisContenidoLimpio},
		Admitido: &vecports.ResultadoOperacionObjeto{Objeto: vecports.ObjetoAlmacenado{
			ConectorID: "almacen_ficheros", Zona: vecports.ZonaAlmacenAdmitida,
			MIME: "application/pdf", Tamano: int64(len(contenido)), HuellaSHA256: huella}},
	}
	r, err := acreditarOriginalSubidaLocal(base, persona, documento, 1, contenido, config)
	if err != nil || r.DocumentoID != documento || r.DocumentoVersion != 1 ||
		r.HuellaSHA256Bytes != huella || r.PersonaTitularRef != persona ||
		r.PersonaAportanteRef != persona || r.CustodioID != "almacen_ficheros" ||
		r.EstadoAdmision != docports.EstadoSubidaLocalAdmitida {
		t.Fatalf("original acreditado: %+v, %v", r, err)
	}
	serializado, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var campos map[string]json.RawMessage
	if err := json.Unmarshal(serializado, &campos); err != nil || len(campos) != 12 ||
		campos["huella_sha256_bytes"] == nil || campos["persona_aportante_ref"] == nil ||
		campos["custodio_id"] == nil {
		t.Fatalf("contrato de doce campos: %s, %v", serializado, err)
	}

	casos := []struct {
		nombre  string
		cambiar func(*docports.InstantaneaSubidaLocal, *[]byte, *string, *uint64)
	}{
		{"version distinta", func(_ *docports.InstantaneaSubidaLocal, _ *[]byte, _ *string, v *uint64) { *v = 2 }},
		{"sin admision", func(i *docports.InstantaneaSubidaLocal, _ *[]byte, _ *string, _ *uint64) {
			i.Resultado.Estado = docports.EstadoSubidaLocalAnalizando
		}},
		{"bytes cambiados", func(_ *docports.InstantaneaSubidaLocal, b *[]byte, _ *string, _ *uint64) {
			copia := append([]byte(nil), (*b)...)
			copia[len(copia)-1] ^= 1
			*b = copia
		}},
		{"otro actor", func(_ *docports.InstantaneaSubidaLocal, _ *[]byte, p *string, _ *uint64) {
			*p = "per_abcdefghijkl0123456789"
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			i, b, p, v := base, contenido, persona, uint64(1)
			caso.cambiar(&i, &b, &p, &v)
			if _, err := acreditarOriginalSubidaLocal(i, p, documento, v, b, config); !errors.Is(err, docports.ErrSubidaLocalDenegada) {
				t.Fatalf("original ajeno o alterado aceptado: %v", err)
			}
		})
	}
}
