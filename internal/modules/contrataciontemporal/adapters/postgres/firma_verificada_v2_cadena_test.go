package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestConsultaFirmaV2RechazaRangoConNull(t *testing.T) {
	m, w := fixtureConsultaFirmaV2()
	canon, err := m.Canonico()
	if err != nil {
		t.Fatal(err)
	}
	contenido, err := json.Marshal(w)
	if err != nil {
		t.Fatal(err)
	}
	invalido := bytes.ReplaceAll(contenido, []byte(`"ByteRange":[0,`), []byte(`"ByteRange":[null,`))
	if bytes.Equal(invalido, contenido) {
		t.Fatal("fixture no contiene ByteRange esperado")
	}
	tx := &txFirmaV2Prueba{t: t, canonico: string(canon), contenido: invalido}
	r := &RegistroFirmasVerificadasPostgreSQL{pool: &poolFirmaV2Prueba{tx: tx}}
	if l, err := r.ConsultarFirmasAutorizadasV2(context.Background(), m, capacidadConsultaFirmaV2Prueba(t, m)); !errors.Is(err, ports.ErrResultadoFirmaDocumentoInvalido) || len(l.Firmas) != 0 || tx.commits != 0 || tx.rollbacks != 1 {
		t.Fatal("null admitido como cero explícito")
	}
}

func TestConsultaFirmaV2LigaSegundoPasoAlPDFAnterior(t *testing.T) {
	casos := []struct {
		nombre  string
		alterar func(*respuestaFirmasR5SQL172)
	}{
		{"cadena_exacta", nil},
		{"firma_ajena", func(w *respuestaFirmasR5SQL172) { ref := "firma:ajena"; w.RevisionesPDF[1].FirmaAnteriorRef = &ref }},
		{"recibo_ajeno", func(w *respuestaFirmasR5SQL172) { ref := "recibo:ajeno"; w.RevisionesPDF[1].ReciboAnteriorRef = &ref }},
		{"entrada_ref", func(w *respuestaFirmasR5SQL172) { w.RevisionesPDF[1].EntradaDocumentoRef = "custodia:ajena" }},
		{"entrada_version", func(w *respuestaFirmasR5SQL172) { w.RevisionesPDF[1].EntradaDocumentoVersion++ }},
		{"entrada_huella", func(w *respuestaFirmasR5SQL172) { w.RevisionesPDF[1].EntradaDocumentoHuella = strings.Repeat("f", 64) }},
		{"entrada_longitud", func(w *respuestaFirmasR5SQL172) { w.RevisionesPDF[1].EntradaDocumentoLongitud-- }},
		{"revision_ausente", func(w *respuestaFirmasR5SQL172) { w.RevisionesPDF = w.RevisionesPDF[1:] }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			m, w := fixtureConsultaFirmaV2()
			m.PasoOrden = 2
			custodia, version := "custodia:primer-pdf", uint64(1)
			w.Firmas[0].DocumentoCustodia, w.Firmas[0].VersionCustodia = &custodia, &version
			w.RevisionesPDF[0].firmaExternaSQL170 = w.Firmas[0]
			base := w.Firmas[0]
			base.FirmaRef, base.ReciboRef, base.Secuencia, base.PasoOrden = "firma:segundo-paso", "recibo:segundo-paso", 2, 2
			firmado := strings.Repeat("e", 64)
			base.FirmadoHuella = &firmado
			v := w.RevisionesPDF[0]
			v.firmaExternaSQL170 = base
			v.FirmaAnteriorRef, v.ReciboAnteriorRef = &w.Firmas[0].FirmaRef, &w.Firmas[0].ReciboRef
			v.OrdenFirmaPDF, v.EntradaDocumentoRef, v.EntradaDocumentoLongitud = 2, custodia, 200
			v.EntradaDocumentoHuella = *w.Firmas[0].FirmadoHuella
			v.ByteRange, v.RevisionLongitud, v.RevisionHuellaSHA256 = []uint64{0, 220, 280, 20}, 300, firmado
			v.EvidenciaFirmasCanonica = `[{"orden":1},{"orden":2}]`
			h := sha256.Sum256([]byte(v.EvidenciaFirmasCanonica))
			v.EvidenciaFirmasHuellaSHA256 = hex.EncodeToString(h[:])
			w.Firmas = append(w.Firmas, base)
			w.RevisionesPDF = append(w.RevisionesPDF, v)
			if caso.alterar != nil {
				caso.alterar(&w)
			}
			canon, err := m.Canonico()
			if err != nil {
				t.Fatal(err)
			}
			contenido, err := json.Marshal(w)
			if err != nil {
				t.Fatal(err)
			}
			tx := &txFirmaV2Prueba{t: t, canonico: string(canon), contenido: contenido}
			r := &RegistroFirmasVerificadasPostgreSQL{pool: &poolFirmaV2Prueba{tx: tx}}
			l, err := r.ConsultarFirmasAutorizadasV2(context.Background(), m, capacidadConsultaFirmaV2Prueba(t, m))
			if caso.alterar == nil {
				if err != nil || len(l.RevisionesPDF) != 2 || tx.commits != 1 || tx.rollbacks != 0 {
					t.Fatalf("cadena exacta rechazada: %v", err)
				}
			} else if !errors.Is(err, ports.ErrResultadoFirmaDocumentoInvalido) || len(l.Firmas) != 0 || tx.commits != 0 || tx.rollbacks != 1 {
				t.Fatalf("cadena ajena confirmada: %v, commit=%d", err, tx.commits)
			}
		})
	}
}

func TestConsultaFirmaV2ConservaCabezaHistoricaReplay(t *testing.T) {
	m, w := fixtureConsultaFirmaV2()
	// SQL devuelve la cabeza previa guardada junto al acto, no la cabeza nueva.
	w.HistoriaRevision = w.Firmas[0].HistoriaRevision
	w.HistoriaHuella = *w.Firmas[0].HistoriaHuella
	canon, err := m.Canonico()
	if err != nil {
		t.Fatal(err)
	}
	contenido, err := json.Marshal(w)
	if err != nil {
		t.Fatal(err)
	}
	tx := &txFirmaV2Prueba{t: t, canonico: string(canon), contenido: contenido}
	r := &RegistroFirmasVerificadasPostgreSQL{pool: &poolFirmaV2Prueba{tx: tx}}
	l, err := r.ConsultarFirmasAutorizadasV2(context.Background(), m, capacidadConsultaFirmaV2Prueba(t, m))
	if err != nil || l.HistoriaRevision != 0 || l.HistoriaHuella != w.HistoriaHuella || tx.commits != 1 {
		t.Fatalf("recuperación cambió historia: %v", err)
	}
}
