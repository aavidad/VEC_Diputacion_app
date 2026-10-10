package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"vec-diputacion-granada/internal/modules/bolsa/application/inscripcion"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type filaAmbitoInscripcionPrueba struct {
	documento []byte
	err       error
}

func (f filaAmbitoInscripcionPrueba) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	if len(destinos) != 1 {
		return errors.New("scan inesperado")
	}
	puntero, ok := destinos[0].(*[]byte)
	if !ok {
		return errors.New("destino inesperado")
	}
	*puntero = append([]byte(nil), f.documento...)
	return nil
}

type txAmbitoInscripcionPrueba struct {
	pgx.Tx
	filas                 []filaAmbitoInscripcionPrueba
	consultas             []string
	argumentos            [][]any
	confirmada, revertida bool
}

func (tx *txAmbitoInscripcionPrueba) QueryRow(_ context.Context, consulta string, argumentos ...any) pgx.Row {
	tx.consultas = append(tx.consultas, consulta)
	tx.argumentos = append(tx.argumentos, append([]any(nil), argumentos...))
	if len(tx.filas) == 0 {
		return filaAmbitoInscripcionPrueba{err: errors.New("consulta inesperada")}
	}
	fila := tx.filas[0]
	tx.filas = tx.filas[1:]
	return fila
}
func (tx *txAmbitoInscripcionPrueba) Commit(context.Context) error   { tx.confirmada = true; return nil }
func (tx *txAmbitoInscripcionPrueba) Rollback(context.Context) error { tx.revertida = true; return nil }

type poolAmbitoInscripcionPrueba struct {
	tx       *txAmbitoInscripcionPrueba
	llamadas int
	opciones pgx.TxOptions
}

func (p *poolAmbitoInscripcionPrueba) BeginTx(_ context.Context, opciones pgx.TxOptions) (pgx.Tx, error) {
	p.llamadas++
	p.opciones = opciones
	return p.tx, nil
}

func fuenteAmbitoInscripcionPrueba(t *testing.T, pool *poolAmbitoInscripcionPrueba, s contextoSeguridadComunDesarrollo, ahora time.Time) *fuenteAmbitoRRHHInscripcionPostgreSQL {
	t.Helper()
	f, err := nuevaFuenteAmbitoRRHHInscripcionPostgreSQL(pool, relojFijoAltaContratacionTemporalDesarrollo{ahora: ahora},
		[]identidadConsultaRRHHDesarrollo{{perfilRef: s.Resultado.Contexto.PerfilActivoRef,
			identidad: identidadCertificadoDesarrollo{principal: vecdomain.Principal{Attributes: map[string]string{
				"certificate_sha256": "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"}}}}})
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestFuenteAmbitoRRHHInscripcionPreviaSegregaSesionYTransaccion(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	s := contextoInscripcionCanalPrueba(t, ahora, false)
	a := acreditacionSesionInscripcionPrueba(t, s, "interna_corporativa", ahora)
	ref := "solicitud_inscripcion_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	conjunto := ConjuntoGestionRRHHInscripcionBolsa{ConjuntoRef: "conjunto_prueba", UnidadRef: "unidad_prueba",
		AmbitoRef: "ambito_prueba", FuenteRef: "inscripcion.gestion.rrhh.conjunto", FuenteVersion: 2,
		FuenteHuellaSHA256: strings.Repeat("a", 64)}
	historico := AmbitoSolicitudRRHHInscripcionBolsa{SolicitudRef: ref, UnidadRef: conjunto.UnidadRef,
		AmbitoRef: conjunto.AmbitoRef, FuenteRef: "fuente_historica", FuenteVersion: 1,
		FuenteHuellaSHA256: strings.Repeat("b", 64)}
	conjuntoJSON, _ := json.Marshal(conjunto)
	historicoJSON, _ := json.Marshal(historico)
	tx := &txAmbitoInscripcionPrueba{filas: []filaAmbitoInscripcionPrueba{{documento: conjuntoJSON}, {documento: historicoJSON}}}
	pool := &poolAmbitoInscripcionPrueba{tx: tx}
	f := fuenteAmbitoInscripcionPrueba(t, pool, s, ahora)
	mal := a
	mal.ValidaHasta = ahora
	if _, err := f.ResolverAmbitoLecturaRRHH(context.Background(), s, mal, inscripcion.AccionDetalleRRHH, ref, inscripcion.Filtro{}); !errors.Is(err, inscripcion.ErrAccesoDenegado) || pool.llamadas != 0 {
		t.Fatalf("sesión inválida consultó SQL: %v llamadas=%d", err, pool.llamadas)
	}
	resultado, err := f.ResolverAmbitoLecturaRRHH(context.Background(), s, a, inscripcion.AccionDetalleRRHH, ref, inscripcion.Filtro{})
	if err != nil || resultado.AmbitoSolicitud == nil || resultado.AmbitoSolicitud.FuenteVersion != 1 ||
		resultado.ConjuntoGestion.FuenteVersion != 2 || len(tx.consultas) != 2 || !tx.confirmada ||
		pool.opciones.IsoLevel != pgx.Serializable || pool.opciones.AccessMode != pgx.ReadWrite {
		t.Fatalf("lectura PRE-PDP: %+v tx=%+v err=%v", resultado, tx, err)
	}
}

func TestFuenteAmbitoRRHHInscripcionAuditadaConservaNoEncontrada(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	s := contextoInscripcionCanalPrueba(t, ahora, false)
	a := acreditacionSesionInscripcionPrueba(t, s, "interna_corporativa", ahora)
	ref := "solicitud_inscripcion_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	v, err := s.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	captura := inscripcion.CapturaLectura{PersonaRef: a.PersonaRef, PerfilRef: a.PerfilRef, CuentaRef: a.CuentaRef,
		SesionRef: a.SesionRef, AutenticacionRef: a.AutenticacionRef, CertificadoHuellaSHA256: a.CertificadoHuellaSHA256,
		Canal: a.Canal, Accion: inscripcion.AccionDetalleRRHH, RecursoRef: ref, Finalidad: "consulta_inscripcion_rrhh",
		CorrelacionRef: "correlacion_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", RevisionPermisos: 1,
		HuellaInstantaneaSHA256: "0000000000000001000000000000000000000000000000000000000000000000",
		Campos:                  camposLecturaInscripcionBolsa(inscripcion.AccionDetalleRRHH, true), EmitidaEn: ahora, ValidaHasta: ahora.Add(time.Minute),
		ConjuntoGestion: &inscripcion.AmbitoGestionInscripcion{ConjuntoRef: "conjunto_prueba", UnidadRef: "unidad_prueba",
			AmbitoRef: "ambito_prueba", FuenteRef: "inscripcion.gestion.rrhh.conjunto", FuenteVersion: 2, FuenteSHA256: strings.Repeat("a", 64)},
		AmbitoSolicitud: &inscripcion.AmbitoGestionInscripcion{UnidadRef: "unidad_prueba", AmbitoRef: "ambito_prueba",
			FuenteRef: "fuente_historica", FuenteVersion: 1, FuenteSHA256: strings.Repeat("b", 64)}}
	if v.Superficie != vecdomain.SuperficieAutenticacionInternaCorporativaV1 {
		t.Fatal("fixture externa")
	}
	noEncontrada := []byte(`{"resultado":"no_encontrada","ambito":null,"auditoria_ref":"auditoria_prueba"}`)
	tx := &txAmbitoInscripcionPrueba{filas: []filaAmbitoInscripcionPrueba{{documento: noEncontrada}}}
	pool := &poolAmbitoInscripcionPrueba{tx: tx}
	f := fuenteAmbitoInscripcionPrueba(t, pool, s, ahora)
	if _, err := f.ResolverAmbitoRRHH(context.Background(), s, a, ref, captura); !errors.Is(err, inscripcion.ErrAccesoDenegado) ||
		pool.llamadas != 1 || !tx.confirmada || len(tx.consultas) != 1 || tx.consultas[0] != consultaAmbitoRRHHInscripcionAuditada {
		t.Fatalf("no_encontrada no auditada: llamadas=%d commit=%v err=%v", pool.llamadas, tx.confirmada, err)
	}
	if len(tx.argumentos[0]) != 4 {
		t.Fatalf("aridad SQL = %d", len(tx.argumentos[0]))
	}
	bytesCaptura, ok := tx.argumentos[0][3].([]byte)
	if !ok {
		t.Fatal("captura SQL ausente")
	}
	var enviada struct {
		Accion     string                                `json:"accion"`
		RecursoRef string                                `json:"recurso_ref"`
		Idioma     string                                `json:"idioma"`
		Conjunto   *inscripcion.AmbitoGestionInscripcion `json:"conjunto_gestion"`
		Historico  *inscripcion.AmbitoGestionInscripcion `json:"ambito_solicitud"`
	}
	if json.Unmarshal(bytesCaptura, &enviada) != nil || enviada.Accion != inscripcion.AccionDetalleRRHH || enviada.RecursoRef != ref ||
		enviada.Idioma != "es" || enviada.Conjunto == nil || enviada.Historico == nil ||
		enviada.Conjunto.FuenteVersion != 2 || enviada.Historico.FuenteVersion != 1 {
		t.Fatalf("captura SQL cambió ámbito/idioma: %+v", enviada)
	}
	historico := AmbitoRecursoRRHHInscripcionBolsa{SolicitudRef: ref, UnidadRef: "unidad_prueba", AmbitoRef: "ambito_prueba",
		FuenteRef: "fuente_historica", FuenteVersion: 1, FuenteHuellaSHA256: strings.Repeat("b", 64)}
	historicoJSON, _ := json.Marshal(historico)
	respuesta, _ := json.Marshal(struct {
		Resultado    string          `json:"resultado"`
		Ambito       json.RawMessage `json:"ambito"`
		AuditoriaRef string          `json:"auditoria_ref"`
	}{Resultado: "obtenida", Ambito: historicoJSON, AuditoriaRef: "auditoria_prueba"})
	txBuena := &txAmbitoInscripcionPrueba{filas: []filaAmbitoInscripcionPrueba{{documento: respuesta}}}
	fBuena := fuenteAmbitoInscripcionPrueba(t, &poolAmbitoInscripcionPrueba{tx: txBuena}, s, ahora)
	resultado, err := fBuena.ResolverAmbitoRRHH(context.Background(), s, a, ref, captura)
	if err != nil || resultado.AuditoriaRef != "auditoria_prueba" || resultado.FuenteVersion != 1 || !txBuena.confirmada {
		t.Fatalf("lectura auditada perdió historia: %+v err=%v", resultado, err)
	}
	historico.FuenteHuellaSHA256 = strings.Repeat("c", 64)
	historicoJSON, _ = json.Marshal(historico)
	respuesta, _ = json.Marshal(struct {
		Resultado    string          `json:"resultado"`
		Ambito       json.RawMessage `json:"ambito"`
		AuditoriaRef string          `json:"auditoria_ref"`
	}{Resultado: "obtenida", Ambito: historicoJSON, AuditoriaRef: "auditoria_prueba"})
	txMala := &txAmbitoInscripcionPrueba{filas: []filaAmbitoInscripcionPrueba{{documento: respuesta}}}
	fMala := fuenteAmbitoInscripcionPrueba(t, &poolAmbitoInscripcionPrueba{tx: txMala}, s, ahora)
	if _, err := fMala.ResolverAmbitoRRHH(context.Background(), s, a, ref, captura); !errors.Is(err, inscripcion.ErrAccesoDenegado) || txMala.confirmada {
		t.Fatalf("historia divergente confirmada: %v commit=%v", err, txMala.confirmada)
	}
}
