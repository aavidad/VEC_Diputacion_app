package postgres

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"vec-diputacion-granada/internal/modules/bolsa/application"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/pruebas"
)

type filaPersonaAceptacionCTPrueba struct {
	datos []byte
	err   error
}

func (f filaPersonaAceptacionCTPrueba) Scan(dest ...any) error {
	if f.err != nil {
		return f.err
	}
	*dest[0].(*[]byte) = append([]byte(nil), f.datos...)
	return nil
}

type txPersonaAceptacionCTPrueba struct {
	pgx.Tx
	fila               pgx.Row
	consulta           string
	args               []any
	commits, rollbacks int
	commitErr          error
	execErr            error
	ajustes            string
	execs              int
}

func (t *txPersonaAceptacionCTPrueba) Exec(_ context.Context, q string, _ ...any) (pgconn.CommandTag, error) {
	t.execs++
	t.ajustes = q
	return pgconn.NewCommandTag("SELECT 1"), t.execErr
}

func (t *txPersonaAceptacionCTPrueba) QueryRow(_ context.Context, q string, args ...any) pgx.Row {
	t.consulta = q
	t.args = append([]any(nil), args...)
	if t.execs != 1 || t.ajustes != ajustesConsultaPersonaAceptacionCT {
		return filaPersonaAceptacionCTPrueba{err: &pgconn.PgError{Code: "22023"}}
	}
	return t.fila
}
func (t *txPersonaAceptacionCTPrueba) Commit(context.Context) error   { t.commits++; return t.commitErr }
func (t *txPersonaAceptacionCTPrueba) Rollback(context.Context) error { t.rollbacks++; return nil }

type poolPersonaAceptacionCTPrueba struct {
	transacciones []*txPersonaAceptacionCTPrueba
	inicios       int
	opciones      pgx.TxOptions
}

func (p *poolPersonaAceptacionCTPrueba) BeginTx(_ context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	p.opciones = o
	i := p.inicios
	p.inicios++
	if i >= len(p.transacciones) {
		return nil, errors.New("inicio inesperado")
	}
	return p.transacciones[i], nil
}

func ordenPersonaAceptacionCTPrueba(t *testing.T) (ports.OrdenConsultaPersonaAceptacionCT, time.Time) {
	t.Helper()
	ahora := time.Date(2026, 9, 30, 11, 0, 0, 1000, time.UTC)
	c, v, e := pruebas.NuevoContextoRegistradoYVinculoV2(ahora.Add(-time.Microsecond), "per_0123456789abcdefghijkl", "prf_0123456789abcdefghijkl", core.AuthMethodCertificate, core.AuthAssuranceHigh)
	if e != nil {
		t.Fatal(e)
	}
	s := ports.SolicitudConsultaPersonaAceptacionCT{Selector: ports.SelectorPersonaAceptacionCT{UnidadRef: "unidad:rrhh", CategoriaRef: "categoria:una", NecesidadRef: "necesidad:una", AceptacionOperacionRef: "aceptacion:una", AceptacionRegistroSHA256: strings.Repeat("a", 64), AperturaOperacionRef: "apertura:una", AperturaRegistroSHA256: strings.Repeat("b", 64), LlamamientoRef: "llamamiento:uno", PropuestaRef: "propuesta:una"}, ActorConfiable: ports.ActorConfiablePersonaAceptacionCT{Resultado: c, Vinculo: v}}
	p, e := application.PrepararConsultaPersonaAceptacionCT(s)
	if e != nil {
		t.Fatal(e)
	}
	h, _ := p.Recurso.HuellaContextoAutorizacionSHA256()
	x, e := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("decision:prueba", strings.Repeat("c", 64), strings.Repeat("d", 64), c.RegistroContextoRef, c.HuellaSHA256, ports.AccionConsultaPersonaAceptacionCT, p.Recurso.Referencia, h, ports.AudienciaConsultaPersonaAceptacionCT, ahora.Add(-time.Microsecond), ahora.Add(time.Second))
	if e != nil {
		t.Fatal(e)
	}
	raiz, _ := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	m, e := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), x, []byte("d"), []byte("m"), c.RepresentacionCanonica, c.Contexto.Instantanea.PersonaVersion, c.Contexto.Instantanea.PerfilVersion, []byte("p"), []byte("s"), []byte("e"), raiz)
	if e != nil {
		t.Fatal(e)
	}
	return ports.OrdenConsultaPersonaAceptacionCT{Solicitud: s, Material: m}, ahora
}

func resultadoPersonaAceptacionCTPrueba(o ports.OrdenConsultaPersonaAceptacionCT, ahora time.Time) ports.ResultadoConsultaPersonaAceptacionCT {
	s := o.Solicitud.Selector
	x := o.Material.ResumenCapacidad()
	return ports.ResultadoConsultaPersonaAceptacionCT{Estado: "acreditado", Aceptacion: &ports.AceptacionPersonaCT{OperacionRef: s.AceptacionOperacionRef, ReciboRef: "recibo:uno", RegistroSHA256: s.AceptacionRegistroSHA256, AperturaOperacionRef: s.AperturaOperacionRef, AperturaRegistroSHA256: s.AperturaRegistroSHA256, LlamamientoRef: s.LlamamientoRef}, Persona: &ports.PersonaAceptadaCT{Ref: "per_" + strings.Repeat("p", 24), Version: 1}, Vinculo: &ports.VinculoPersonaAceptadaCT{Ref: "vinculo:uno", Version: 1, ProcedenciaRef: "procedencia:una", ProcedenciaVersion: 1, ProcedenciaSHA256: strings.Repeat("d", 64), Poblacion: "externa", VigenteHasta: ahora.Add(time.Hour)}, Evidencia: ports.EvidenciaConsultaPersonaAceptacionCT{DecisionRef: x.DecisionRef(), ConsumoHuellaSHA256: strings.Repeat("e", 64), AuditoriaRef: "auditoria:una", ConsultadaEn: ahora}}
}

func TestPersonaAceptacionCTPostgreSQLConfirmaLosTresEstadosConEvidencia(t *testing.T) {
	for _, estado := range []string{"acreditado", "pendiente", "no_encontrada"} {
		t.Run(estado, func(t *testing.T) {
			o, ahora := ordenPersonaAceptacionCTPrueba(t)
			v := resultadoPersonaAceptacionCTPrueba(o, ahora)
			v.Estado = estado
			if estado != "acreditado" {
				v.Persona = nil
				v.Vinculo = nil
			}
			if estado == "no_encontrada" {
				v.Aceptacion = nil
			}
			b, _ := json.Marshal(v)
			tx := &txPersonaAceptacionCTPrueba{fila: filaPersonaAceptacionCTPrueba{datos: b}}
			pool := &poolPersonaAceptacionCTPrueba{transacciones: []*txPersonaAceptacionCTPrueba{tx}}
			r := &RepositorioConsultaPersonaAceptacionCTPostgreSQL{pool: pool, ahora: func() time.Time { return ahora }}
			salida, e := r.ConsultarPersonaAceptacionCT(context.Background(), o)
			if e != nil || salida.Estado != estado || tx.commits != 1 || tx.rollbacks != 1 || pool.opciones.IsoLevel != pgx.Serializable || pool.opciones.AccessMode != pgx.ReadWrite {
				t.Fatalf("transacción estado %s: %v", estado, e)
			}
			p, _ := application.PrepararConsultaPersonaAceptacionCT(o.Solicitud)
			if tx.execs != 1 || tx.ajustes != ajustesConsultaPersonaAceptacionCT || tx.consulta != funcionConsultaPersonaAceptacionCT || len(tx.args) != 11 || tx.args[0] != string(p.MaterialCanonico) || !bytes.Equal(tx.args[1].([]byte), o.Material.CapacidadCanonica()) || !bytes.Equal(tx.args[4].([]byte), o.Material.ContextoActorCanonico()) || !bytes.Equal(tx.args[10].([]byte), o.Material.RaizPublicaSPKI()) {
				t.Fatal("fachada o material V3 sustituidos")
			}
		})
	}
}

func TestPersonaAceptacionCTPostgreSQLFalloDeAjustesLocalesNoConsultaNiConfirma(t *testing.T) {
	o, ahora := ordenPersonaAceptacionCTPrueba(t)
	tx := &txPersonaAceptacionCTPrueba{execErr: errors.New("detalle privado"), fila: filaPersonaAceptacionCTPrueba{}}
	pool := &poolPersonaAceptacionCTPrueba{transacciones: []*txPersonaAceptacionCTPrueba{tx}}
	r := &RepositorioConsultaPersonaAceptacionCTPostgreSQL{pool: pool, ahora: func() time.Time { return ahora }}
	v, err := r.ConsultarPersonaAceptacionCT(context.Background(), o)
	if !errors.Is(err, ports.ErrConsultaPersonaAceptacionCTNoDisponible) || v.Estado != "" || tx.execs != 1 || tx.consulta != "" || tx.commits != 0 || tx.rollbacks != 1 || pool.inicios != 1 {
		t.Fatal("consultó o confirmó sin límites locales")
	}
}

func TestPersonaAceptacionCTPostgreSQLRechazaIncoherenciaAntesCommit(t *testing.T) {
	for _, caso := range []string{"desconocido", "cruce", "parcial", "dosJSON", "exceso", "commit"} {
		t.Run(caso, func(t *testing.T) {
			o, ahora := ordenPersonaAceptacionCTPrueba(t)
			v := resultadoPersonaAceptacionCTPrueba(o, ahora)
			if caso == "cruce" {
				v.Aceptacion.RegistroSHA256 = strings.Repeat("f", 64)
			}
			if caso == "parcial" {
				v.Persona = nil
			}
			b, _ := json.Marshal(v)
			if caso == "desconocido" {
				b = append(b[:len(b)-1], []byte(`,"dni":"dato_inesperado"}`)...)
			}
			if caso == "dosJSON" {
				b = append(b, []byte(` {}`)...)
			}
			if caso == "exceso" {
				b = bytes.Repeat([]byte("x"), 17<<10)
			}
			tx := &txPersonaAceptacionCTPrueba{fila: filaPersonaAceptacionCTPrueba{datos: b}}
			if caso == "commit" {
				tx.commitErr = errors.New("detalle privado")
			}
			pool := &poolPersonaAceptacionCTPrueba{transacciones: []*txPersonaAceptacionCTPrueba{tx}}
			r := &RepositorioConsultaPersonaAceptacionCTPostgreSQL{pool: pool, ahora: func() time.Time { return ahora }}
			v, e := r.ConsultarPersonaAceptacionCT(context.Background(), o)
			if !errors.Is(e, ports.ErrConsultaPersonaAceptacionCTNoDisponible) || v.Estado != "" || tx.rollbacks != 1 || pool.inicios != 1 || (caso != "commit" && tx.commits != 0) {
				t.Fatalf("resultado parcial/commit indebido: %v", e)
			}
		})
	}
}

func TestPersonaAceptacionCTPostgreSQLSerializacionRealYDeclarada(t *testing.T) {
	for _, caso := range []struct {
		nombre, rutina string
		commit         bool
		intentos       int
		exito          bool
	}{{"real", "ExecUpdate", false, 2, true}, {"commit", "PreCommit_CheckForSerializationFailure", true, 2, true}, {"declarada", "exec_stmt_raise", false, 1, false}, {"agotada", "ExecUpdate", false, 3, false}} {
		t.Run(caso.nombre, func(t *testing.T) {
			o, ahora := ordenPersonaAceptacionCTPrueba(t)
			b, _ := json.Marshal(resultadoPersonaAceptacionCTPrueba(o, ahora))
			fallo := &pgconn.PgError{Code: "40001", Routine: caso.rutina}
			pool := &poolPersonaAceptacionCTPrueba{}
			for i := 0; i < caso.intentos; i++ {
				tx := &txPersonaAceptacionCTPrueba{fila: filaPersonaAceptacionCTPrueba{datos: b}}
				if i == 0 || !caso.exito {
					if caso.commit {
						tx.commitErr = fallo
					} else {
						tx.fila = filaPersonaAceptacionCTPrueba{err: fallo}
					}
				}
				pool.transacciones = append(pool.transacciones, tx)
			}
			r := &RepositorioConsultaPersonaAceptacionCTPostgreSQL{pool: pool, ahora: func() time.Time { return ahora }}
			v, e := r.ConsultarPersonaAceptacionCT(context.Background(), o)
			if pool.inicios != caso.intentos || (caso.exito && (e != nil || v.Estado != "acreditado")) || (!caso.exito && (!errors.Is(e, ports.ErrConsultaPersonaAceptacionCTNoDisponible) || v.Estado != "")) {
				t.Fatalf("reintentos: %d error: %v", pool.inicios, e)
			}
		})
	}
}

func TestPersonaAceptacionCTPostgreSQLDenegadoOCaducadoNoExpone(t *testing.T) {
	o, ahora := ordenPersonaAceptacionCTPrueba(t)
	tx := &txPersonaAceptacionCTPrueba{fila: filaPersonaAceptacionCTPrueba{err: &pgconn.PgError{Code: "42501", Message: "detalle privado"}}}
	pool := &poolPersonaAceptacionCTPrueba{transacciones: []*txPersonaAceptacionCTPrueba{tx}}
	r := &RepositorioConsultaPersonaAceptacionCTPostgreSQL{pool: pool, ahora: func() time.Time { return ahora }}
	if v, e := r.ConsultarPersonaAceptacionCT(context.Background(), o); !errors.Is(e, ports.ErrConsultaPersonaAceptacionCTDenegada) || v.Estado != "" || tx.commits != 0 || pool.inicios != 1 {
		t.Fatal("denegación reinterpretada")
	}
	pool.inicios = 0
	r.ahora = func() time.Time { return ahora.Add(time.Hour) }
	if _, e := r.ConsultarPersonaAceptacionCT(context.Background(), o); !errors.Is(e, ports.ErrConsultaPersonaAceptacionCTDenegada) || pool.inicios != 0 {
		t.Fatal("capacidad caducada llegó a SQL")
	}
	if _, e := NuevoRepositorioConsultaPersonaAceptacionCTPostgreSQL(nil, time.Now); e == nil {
		t.Fatal("pool nulo")
	}
}
