package postgres

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"vec-diputacion-granada/internal/modules/bolsa/application/inscripcion"
)

type transaccionInscripcionPrueba struct {
	pgx.Tx
	pasos       []string
	falloCommit error
}

func (t *transaccionInscripcionPrueba) Exec(_ context.Context, _ string, _ ...any) (pgconn.CommandTag, error) {
	t.pasos = append(t.pasos, "configurar")
	return pgconn.NewCommandTag("SELECT 1"), nil
}

func (t *transaccionInscripcionPrueba) Commit(context.Context) error {
	t.pasos = append(t.pasos, "commit")
	return t.falloCommit
}

func (t *transaccionInscripcionPrueba) Rollback(context.Context) error {
	t.pasos = append(t.pasos, "rollback")
	return nil
}

type iniciadorInscripcionPrueba struct {
	tx       *transaccionInscripcionPrueba
	opciones pgx.TxOptions
	inicios  int
}

func (p *iniciadorInscripcionPrueba) BeginTx(_ context.Context, opciones pgx.TxOptions) (pgx.Tx, error) {
	p.inicios++
	p.opciones = opciones
	p.tx.pasos = append(p.tx.pasos, "abrir")
	return p.tx, nil
}

func TestInscripcionPoolsLectoresSegregados(t *testing.T) {
	externo := &iniciadorInscripcionPrueba{tx: &transaccionInscripcionPrueba{}}
	interno := &iniciadorInscripcionPrueba{tx: &transaccionInscripcionPrueba{}}
	lectorExterno := &iniciadorInscripcionPrueba{tx: &transaccionInscripcionPrueba{}}
	lectorEmpleado := &iniciadorInscripcionPrueba{tx: &transaccionInscripcionPrueba{}}
	lectorRRHH := &iniciadorInscripcionPrueba{tx: &transaccionInscripcionPrueba{}}
	r, err := nuevoRepositorioInscripcionesPostgreSQL(externo, interno, lectorExterno, lectorEmpleado, lectorRRHH)
	if err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		canal, accion string
		esperado      iniciadorTransacciones
	}{
		{"externa_personal", inscripcion.AccionListarAbiertas, lectorExterno},
		{"externa_personal", inscripcion.AccionDetallePropia, lectorExterno},
		{"interna_corporativa", inscripcion.AccionListarPropias, lectorEmpleado},
		{"interna_corporativa", inscripcion.AccionDetalleAbierta, lectorEmpleado},
		{"interna_corporativa", inscripcion.AccionListarRRHH, lectorRRHH},
		{"interna_corporativa", inscripcion.AccionDetalleRRHH, lectorRRHH},
		{"interna_corporativa", inscripcion.AccionMotivosRRHH, lectorRRHH},
	}
	for _, caso := range casos {
		obtenido, err := r.seleccionarLectorInscripcion(caso.canal, caso.accion)
		if err != nil || obtenido != caso.esperado {
			t.Fatalf("lector cruzado para %s/%s: %v", caso.canal, caso.accion, err)
		}
	}
	if _, err := r.seleccionarLectorInscripcion("externa_personal", inscripcion.AccionListarRRHH); !errors.Is(err, inscripcion.ErrAccesoDenegado) {
		t.Fatalf("RRHH externo: %v", err)
	}
	if _, err := r.seleccionarLectorInscripcion("interna_corporativa", "accion_desconocida"); !errors.Is(err, inscripcion.ErrAccesoDenegado) {
		t.Fatalf("accion desconocida: %v", err)
	}
	if _, err := nuevoRepositorioInscripcionesPostgreSQL(externo, interno, lectorExterno, lectorEmpleado, lectorEmpleado); !errors.Is(err, inscripcion.ErrNoDisponible) {
		t.Fatalf("pool compartido empleado/RRHH: %v", err)
	}
	if _, err := nuevoRepositorioInscripcionesPostgreSQL(externo, interno, lectorExterno, lectorEmpleado, nil); !errors.Is(err, inscripcion.ErrNoDisponible) {
		t.Fatalf("pool RRHH ausente: %v", err)
	}
	_, err = consultarInscripcion(context.Background(), r,
		inscripcion.Actor{Canal: "interna_corporativa"}, inscripcion.AccionListarRRHH,
		"inscripciones_rrhh_invalida", inscripcion.Filtro{Limite: 1},
		selectorListaInscripcion{Limite: 1}, func(inscripcion.Pagina) error { return nil })
	if !errors.Is(err, inscripcion.ErrAccesoDenegado) || lectorExterno.inicios != 0 ||
		lectorEmpleado.inicios != 0 || lectorRRHH.inicios != 0 {
		t.Fatalf("captura inválida alcanzó pool lector: err=%v inicios=%d/%d/%d",
			err, lectorExterno.inicios, lectorEmpleado.inicios, lectorRRHH.inicios)
	}
}

func TestInscripcionProyeccionAbiertaListaYDetalle(t *testing.T) {
	base := inscripcion.BolsaAbierta{
		ConvocatoriaRef: "cv1_abc_v1", Titulo: "Convocatoria de prueba",
		NumeroCategorias: 2, CatalogoVersion: 1,
		PlazoInicio:       time.Date(2026, 10, 9, 8, 0, 0, 0, time.UTC),
		PlazoFin:          time.Date(2026, 10, 19, 8, 0, 0, 0, time.UTC),
		RequisitosResumen: "Requisitos publicados", PuedeIniciar: true,
	}
	if err := validarBolsaAbiertaInscripcion(base, false); err != nil {
		t.Fatalf("la lista mínima válida fue rechazada: %v", err)
	}
	base.Categorias = []inscripcion.Categoria{
		{CategoriaRef: "categoria:a", Categoria: "Categoría A"},
		{CategoriaRef: "categoria:b", Categoria: "Categoría B"},
	}
	if err := validarBolsaAbiertaInscripcion(base, false); !errors.Is(err, inscripcion.ErrNoDisponible) {
		t.Fatalf("la lista incluyó categorías: %v", err)
	}
	if err := validarBolsaAbiertaInscripcion(base, true); err != nil {
		t.Fatalf("el detalle completo fue rechazado: %v", err)
	}
	base.Categorias[1] = base.Categorias[0]
	if err := validarBolsaAbiertaInscripcion(base, true); !errors.Is(err, inscripcion.ErrNoDisponible) {
		t.Fatalf("el detalle duplicó una categoría: %v", err)
	}
	base.Categorias = base.Categorias[:1]
	if err := validarBolsaAbiertaInscripcion(base, true); !errors.Is(err, inscripcion.ErrNoDisponible) {
		t.Fatalf("el detalle omitió una categoría: %v", err)
	}
	base.NumeroCategorias = 129
	if err := validarBolsaAbiertaInscripcion(base, false); !errors.Is(err, inscripcion.ErrNoDisponible) {
		t.Fatalf("el recuento excedió el contrato: %v", err)
	}
}

func TestInscripcionEntregaSoloTrasAuditoriaYCommit(t *testing.T) {
	tx := &transaccionInscripcionPrueba{}
	p := &iniciadorInscripcionPrueba{tx: tx}
	resultado, err := transaccionInscripcion(context.Background(), p, func(pgx.Tx) ([]byte, error) {
		tx.pasos = append(tx.pasos, "leer", "auditar")
		return []byte(`{"ok":true}`), nil
	}, func(contenido []byte) error {
		tx.pasos = append(tx.pasos, "validar")
		return decodificarInscripcionEstricta(contenido, new(struct {
			OK bool `json:"ok"`
		}))
	})
	if err != nil || string(resultado) != `{"ok":true}` {
		t.Fatalf("resultado sin commit: %v", err)
	}
	if p.opciones.IsoLevel != pgx.Serializable || p.opciones.AccessMode != pgx.ReadWrite {
		t.Fatalf("opciones de lectura auditada: %+v", p.opciones)
	}
	esperados := []string{"abrir", "configurar", "leer", "auditar", "validar", "commit", "rollback"}
	if !reflect.DeepEqual(tx.pasos, esperados) {
		t.Fatalf("orden transaccional: %v", tx.pasos)
	}
}

func TestInscripcionNoEntregaProyeccionNiConfirmaSinValidar(t *testing.T) {
	tx := &transaccionInscripcionPrueba{}
	p := &iniciadorInscripcionPrueba{tx: tx}
	resultado, err := transaccionInscripcion(context.Background(), p, func(pgx.Tx) ([]byte, error) {
		return []byte(`{"ok":true,"dato_inesperado":"personal"}`), nil
	}, func(contenido []byte) error {
		return decodificarInscripcionEstricta(contenido, new(struct {
			OK bool `json:"ok"`
		}))
	})
	if resultado != nil || !errors.Is(err, inscripcion.ErrNoDisponible) {
		t.Fatalf("se entregó fila inválida: resultado=%v err=%v", resultado, err)
	}
	for _, paso := range tx.pasos {
		if paso == "commit" {
			t.Fatal("se confirmó una proyección inválida")
		}
	}
}

func TestInscripcionErrorPostgreSQLConservaCausaNominalSinTextoSQL(t *testing.T) {
	casos := []struct {
		codigo string
		causa  error
	}{
		{"42501", inscripcion.ErrAccesoDenegado},
		{"B9601", inscripcion.ErrCatalogoCambiado},
		{"B9602", inscripcion.ErrPlazoCerrado},
		{"B9603", inscripcion.ErrClaveConflicto},
		{"B9604", inscripcion.ErrSolicitudExistente},
		{"B9605", inscripcion.ErrDeclaracionInvalida},
		{"B9606", inscripcion.ErrRequisitoInvalido},
		{"B9607", inscripcion.ErrNoDisponible},
		{"B9701", inscripcion.ErrVinculoIdentidadPendiente},
		{"B9702", inscripcion.ErrActaNoDisponible},
		{"B9703", inscripcion.ErrConflicto},
	}
	for _, caso := range casos {
		t.Run(caso.codigo, func(t *testing.T) {
			err := errorInscripcionPostgreSQL(context.Background(), &pgconn.PgError{
				Code: caso.codigo, Message: "dato personal de prueba", Detail: "detalle privado",
			}, "ejecutar")
			if !errors.Is(err, caso.causa) || err.Error() == "dato personal de prueba" {
				t.Fatalf("error nominal o privacidad inválida: %v", err)
			}
			var diagnostico interface{ DiagnosticoInscripcion() (string, string) }
			if !errors.As(err, &diagnostico) {
				t.Fatal("falta diagnóstico seguro para 5xx")
			}
			etapa, codigo := diagnostico.DiagnosticoInscripcion()
			if etapa != "ejecutar" || codigo != caso.codigo ||
				strings.Contains(err.Error(), "detalle privado") || strings.Contains(err.Error(), "dato personal") {
				t.Fatalf("diagnóstico expuso datos o perdió SQLSTATE: %v / %s %s", err, etapa, codigo)
			}
		})
	}
	malformado := errorInscripcionPostgreSQL(context.Background(), &pgconn.PgError{
		Code: "B96\n!", Detail: "detalle privado",
	}, "etapa-controlada-por-error")
	var diagnostico interface{ DiagnosticoInscripcion() (string, string) }
	if !errors.As(malformado, &diagnostico) {
		t.Fatal("falta diagnóstico de código malformado")
	}
	etapa, codigo := diagnostico.DiagnosticoInscripcion()
	if etapa != "desconocida" || codigo != "" || strings.Contains(malformado.Error(), "detalle privado") {
		t.Fatalf("diagnóstico malformado filtró datos: %s %q %v", etapa, codigo, malformado)
	}
}
