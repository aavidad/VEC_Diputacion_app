package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestConsultaFirmaV2ConservaEvidenciaTrasCommit(t *testing.T) {
	m, w := fixtureConsultaFirmaV2()
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
	if err != nil || len(l.RevisionesPDF) != 1 || len(l.Firmas) != 1 || tx.commits != 1 || tx.rollbacks != 0 || tx.consultas != 1 {
		t.Fatalf("lectura sin confirmar: %v, commits=%d, rollbacks=%d", err, tx.commits, tx.rollbacks)
	}
	v := l.RevisionesPDF[0]
	if !bytes.Equal(v.EvidenciaFirmasCanonica, []byte(w.RevisionesPDF[0].EvidenciaFirmasCanonica)) ||
		v.FirmaRef != w.Firmas[0].FirmaRef || v.ReciboRef != w.Firmas[0].ReciboRef ||
		v.FirmanteRef != w.RevisionesPDF[0].FirmanteRef || v.ByteRange != [4]uint64(w.RevisionesPDF[0].ByteRange) ||
		l.HistoriaRevision != *w.HistoriaRevision || l.HistoriaHuella != w.HistoriaHuella {
		t.Fatal("la lectura modificó la revisión o su historia")
	}
}

func TestConsultaFirmaV2RevierteRespuestaIncoherente(t *testing.T) {
	casos := []struct {
		nombre  string
		alterar func(*respuestaFirmasR5SQL172)
	}{
		{"recibo ajeno", func(w *respuestaFirmasR5SQL172) { w.RevisionesPDF[0].ReciboRef = "recibo:ajeno" }},
		{"evidencia alterada", func(w *respuestaFirmasR5SQL172) { w.RevisionesPDF[0].EvidenciaFirmasCanonica = `[{"orden":2}]` }},
		{"rango sin cubrir", func(w *respuestaFirmasR5SQL172) { w.RevisionesPDF[0].ByteRange[3]-- }},
		{"rango quinto elemento", func(w *respuestaFirmasR5SQL172) {
			w.RevisionesPDF[0].ByteRange = append(w.RevisionesPDF[0].ByteRange, 0)
		}},
		{"entrada ajena", func(w *respuestaFirmasR5SQL172) { w.RevisionesPDF[0].EntradaDocumentoRef = "documento:ajeno" }},
		{"firma repetida", func(w *respuestaFirmasR5SQL172) { w.RevisionesPDF = append(w.RevisionesPDF, w.RevisionesPDF[0]) }},
		{"version futura", func(w *respuestaFirmasR5SQL172) { w.Firmas[0].ExpedienteVersion++ }},
		{"sin proyeccion", func(w *respuestaFirmasR5SQL172) { w.RevisionesPDF = nil }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			m, w := fixtureConsultaFirmaV2()
			caso.alterar(&w)
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
			if !errors.Is(err, ports.ErrResultadoFirmaDocumentoInvalido) || len(l.Firmas) != 0 || tx.commits != 0 || tx.rollbacks != 1 {
				t.Fatalf("respuesta incoherente salió de la TX: %v, commits=%d, rollbacks=%d", err, tx.commits, tx.rollbacks)
			}
		})
	}
}

func TestConsultaFirmaV2NoEntregaLecturaTrasFallo(t *testing.T) {
	for _, commit := range []bool{false, true} {
		t.Run(map[bool]string{false: "denegado_sql", true: "commit_incierto"}[commit], func(t *testing.T) {
			m, w := fixtureConsultaFirmaV2()
			canon, err := m.Canonico()
			if err != nil {
				t.Fatal(err)
			}
			contenido, err := json.Marshal(w)
			if err != nil {
				t.Fatal(err)
			}
			tx := &txFirmaV2Prueba{t: t, canonico: string(canon), contenido: contenido}
			if commit {
				tx.falloCommit = errors.New("commit incierto")
			} else {
				tx.falloConsulta = &pgconn.PgError{Code: "42501"}
			}
			r := &RegistroFirmasVerificadasPostgreSQL{pool: &poolFirmaV2Prueba{tx: tx}}
			l, err := r.ConsultarFirmasAutorizadasV2(context.Background(), m, capacidadConsultaFirmaV2Prueba(t, m))
			if err == nil || len(l.Firmas) != 0 || tx.rollbacks != 1 || tx.consultas != 1 {
				t.Fatalf("fallo entregó lectura: %v, commits=%d, rollbacks=%d", err, tx.commits, tx.rollbacks)
			}
		})
	}
}

func TestConsultaFirmaV2RechazaCapacidadAntesDeBegin(t *testing.T) {
	m, _ := fixtureConsultaFirmaV2()
	tx := &txFirmaV2Prueba{t: t}
	p := &poolFirmaV2Prueba{tx: tx}
	r := &RegistroFirmasVerificadasPostgreSQL{pool: p}
	if _, err := r.ConsultarFirmasAutorizadasV2(context.Background(), m, ports.CapacidadConsultaFirmasR5V2{}); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || p.inicios != 0 {
		t.Fatal("capacidad ausente abrió transacción")
	}
	mAjeno := m
	mAjeno.Via = ports.ViaFirmaExternaPortafirmas
	if _, err := r.ConsultarFirmasAutorizadasV2(context.Background(), mAjeno, capacidadConsultaFirmaV2Prueba(t, m)); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || p.inicios != 0 {
		t.Fatal("capacidad de otra vía abrió transacción")
	}
}
