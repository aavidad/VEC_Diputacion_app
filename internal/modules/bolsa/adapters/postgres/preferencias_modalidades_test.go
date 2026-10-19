package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	bolsaapp "vec-diputacion-granada/internal/modules/bolsa/application"
	bolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type transaccionModalidadesPrueba struct {
	pgx.Tx
	consulta string
	args     []any
	fila     pgx.Row
	commits  int
}

func (t *transaccionModalidadesPrueba) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (t *transaccionModalidadesPrueba) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	t.consulta, t.args = sql, args
	return t.fila
}
func (t *transaccionModalidadesPrueba) Commit(context.Context) error   { t.commits++; return nil }
func (t *transaccionModalidadesPrueba) Rollback(context.Context) error { return pgx.ErrTxClosed }

type poolModalidadesPrueba struct {
	tx       *transaccionModalidadesPrueba
	opciones pgx.TxOptions
}

func (p *poolModalidadesPrueba) BeginTx(_ context.Context, opciones pgx.TxOptions) (pgx.Tx, error) {
	p.opciones = opciones
	return p.tx, nil
}

type filaModalidadesPrueba struct {
	modalidades []string
	fecha       time.Time
}

func (f filaModalidadesPrueba) Scan(dest ...any) error {
	*dest[0].(*bool) = false
	*dest[1].(*string) = "recibo:preferencias-modalidades:" + strings.Repeat("a", 64)
	*dest[2].(*int64) = 1
	*dest[3].(*[]string) = slices.Clone(f.modalidades)
	*dest[4].(*time.Time) = f.fecha
	return nil
}

func TestEscrituraModalidadesUsaFuncionNominalYConservaNilOVacio(t *testing.T) {
	for _, seleccion := range [][]string{nil, {}} {
		t.Run(map[bool]string{true: "todas", false: "ninguna"}[seleccion == nil], func(t *testing.T) {
			fecha := time.Now().UTC().Truncate(time.Microsecond)
			tx := &transaccionModalidadesPrueba{fila: filaModalidadesPrueba{seleccion, fecha}}
			pool := &poolModalidadesPrueba{tx: tx}
			a := &EscrituraPreferenciasModalidadesPostgreSQL{pool: pool}
			s := bolsa.GuardarPreferenciasModalidades{SolicitudPreferenciasBolsas: bolsa.SolicitudPreferenciasBolsas{
				CandidatoRef: "can_" + strings.Repeat("a", 43), ConsultadaEn: fecha},
				Modalidades: seleccion, VersionEsperada: 0, ClaveIdempotencia: "clave-modalidades-prueba"}
			recibo, err := a.guardarIntento(t.Context(), s, []byte(`[]`))
			if err != nil || tx.commits != 1 || pool.opciones.IsoLevel != pgx.Serializable ||
				pool.opciones.AccessMode != pgx.ReadWrite || !strings.Contains(tx.consulta, funcionGuardarPreferenciasModalidades) ||
				(recibo.Modalidades == nil) != (seleccion == nil) ||
				(tx.args[1].([]string) == nil) != (seleccion == nil) {
				t.Fatalf("contrato de transacción o selección roto: %+v %v", recibo, err)
			}
		})
	}
}

func TestEscrituraModalidadesRechazaReciboConOrdenDistinto(t *testing.T) {
	fecha := time.Now().UTC().Truncate(time.Microsecond)
	tx := &transaccionModalidadesPrueba{fila: filaModalidadesPrueba{[]string{"vacante", "sustitucion"}, fecha}}
	a := &EscrituraPreferenciasModalidadesPostgreSQL{pool: &poolModalidadesPrueba{tx: tx}}
	s := bolsa.GuardarPreferenciasModalidades{SolicitudPreferenciasBolsas: bolsa.SolicitudPreferenciasBolsas{
		CandidatoRef: "can_" + strings.Repeat("a", 43), ConsultadaEn: fecha},
		Modalidades: []string{"programa", "vacante"}, ClaveIdempotencia: "clave-modalidades-prueba"}
	_, err := a.guardarIntento(t.Context(), s, []byte(`[]`))
	if !errors.Is(err, bolsa.ErrPreferenciasNoDisponibles) || tx.commits != 0 {
		t.Fatalf("recibo divergente confirmado: %v", err)
	}
}

func TestDecodificarLecturaModalidadesDistingueNullYVacio(t *testing.T) {
	fecha := time.Now().UTC().Truncate(time.Microsecond).Format("2006-01-02T15:04:05.000000Z")
	base := map[string]any{"auditoria_ref": "aud_v3_lp_" + strings.Repeat("a", 32),
		"auditoria_huella_sha256": strings.Repeat("b", 64), "correlacion_ref": "correlacion:propia", "auditada_en": fecha}
	for _, tc := range []struct {
		version   int
		guardadas any
		recibo    any
		fecha     any
	}{
		{0, nil, nil, nil},
		{1, nil, "recibo:preferencias-modalidades:" + strings.Repeat("a", 64), fecha},
		{1, []string{}, "recibo:preferencias-modalidades:" + strings.Repeat("a", 64), fecha},
	} {
		base["estado"] = map[string]any{"version": tc.version, "modalidades_guardadas": tc.guardadas,
			"modalidades_disponibles": []any{}, "recibo_ref": tc.recibo, "registrado_en": tc.fecha}
		contenido, _ := json.Marshal(base)
		estado, err := decodificarLecturaModalidades(contenido, "correlacion:propia")
		if err != nil || (estado.ModalidadesGuardadas == nil) != (tc.guardadas == nil) {
			t.Fatalf("proyección nula/vacía alterada: %+v %v", estado, err)
		}
	}
	base["estado"].(map[string]any)["modalidades_disponibles"] = []string{"dato no autorizado"}
	contenido, _ := json.Marshal(base)
	if _, err := decodificarLecturaModalidades(contenido, "correlacion:propia"); !errors.Is(err, bolsa.ErrPreferenciasNoDisponibles) {
		t.Fatalf("la base aceptó modalidades disponibles no auditadas: %v", err)
	}
}

func TestSnapshotModalidadesSQLTransportaFuenteExacta(t *testing.T) {
	contenido, err := snapshotModalidadesSQL([]bolsa.ModalidadPreferencia{{Clave: "vacante",
		Fuente: dominiovec.ReferenciaEntradaCatalogo{CatalogoID: "vec.contratacion_temporal.reglas",
			CatalogoVersion: 3, CatalogoHuellaSHA256: strings.Repeat("b", 64), EntradaClave: "c12.modalidad.vacante"}}})
	var opciones []map[string]any
	if err != nil || json.Unmarshal(contenido, &opciones) != nil || len(opciones) != 1 ||
		len(opciones[0]) != 5 || opciones[0]["catalogo_version"] != float64(3) {
		t.Fatalf("snapshot de fuente incompleto: %v", err)
	}
}

func TestVectorModalidadesGoCompartidoConB138(t *testing.T) {
	candidato := "can_" + strings.Repeat("a", 43)
	opciones := make([]bolsa.ModalidadPreferencia, 0, 3)
	for _, clave := range []string{"vacante", "programa", "sustitucion"} {
		opciones = append(opciones, bolsa.ModalidadPreferencia{Clave: clave,
			Fuente: dominiovec.ReferenciaEntradaCatalogo{CatalogoID: "vec.contratacion_temporal.reglas",
				CatalogoVersion: 3, CatalogoHuellaSHA256: strings.Repeat("b", 64),
				EntradaClave: "c12.modalidad." + clave}})
	}
	huella, err := bolsaapp.HuellaSolicitudPreferenciasModalidades(candidato, 7,
		[]string{"vacante", "sustitucion"}, opciones)
	if err != nil || huella != "2b894c9c699723d958f3c0ca679e632ef478651966d99e2d64656bf29e26c76a" {
		t.Fatalf("huella de solicitud distinta del vector B138: %s %v", huella, err)
	}
	recurso := dominiovec.RecursoAutorizable{Referencia: "mi-bolsa:" + candidato,
		ModuloID: "bolsa", Tipo: "participaciones_candidato",
		Ambitos:   map[string]string{"candidato_ref": candidato},
		Atributos: map[string]string{"material_sha256": huella, "propiedad": "candidato", "tipo_preferencia": "modalidades"}}
	contexto, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil || contexto != "1717c482a88d6cafbbd27f81b7a2dd238429f5b6c709e2fec048a2498fc52648" {
		t.Fatalf("contexto de autorización distinto del vector B138: %s %v", contexto, err)
	}
}
