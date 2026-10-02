package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/cronos/application"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type proveedorConfirmacionDocPrueba struct {
	t                 *testing.T
	principalCambiado bool
	audienciaCambiada bool
}

func (p proveedorConfirmacionDocPrueba) AutorizarConfirmacionAltaExternaEnlace(_ context.Context, s docports.SolicitudConfirmacionAltaExternaEnlace) (docports.AutorizacionConfirmacionAltaExternaEnlace, error) {
	p.t.Helper()
	if _, err := s.Preimagen(); err != nil {
		p.t.Fatal("solicitud Doc alterada", err)
	}
	r, err := s.RecursoV3()
	if err != nil {
		p.t.Fatal(err)
	}
	audiencia := docports.AudienciaConfirmarAltaExternaEnlace
	if p.audienciaCambiada {
		audiencia = application.AudienciaJustificacion
	}
	principal := s.ActorAltaRef
	if p.principalCambiado {
		principal = "per_ZZZZZZZZZZZZZZZZZZZZZZ"
	}
	return docports.AutorizacionConfirmacionAltaExternaEnlace{
		Material:    exportacionPrueba(p.t, docports.AccionConfirmarAltaExternaEnlace, audiencia, r),
		PrincipalID: principal, PerfilActivoRef: "prf_ZZZZZZZZZZZZZZZZZZZZZZ", CorrelacionRef: "ref:" + strings.Repeat("3", 64),
	}, nil
}

type txAnexoJustificacionPrueba struct {
	*txLecturaPrueba
	argumentos []any
}

func (t *txAnexoJustificacionPrueba) QueryRow(ctx context.Context, q string, args ...any) pgx.Row {
	t.argumentos = make([]any, len(args))
	for i, v := range args {
		if b, ok := v.([]byte); ok {
			t.argumentos[i] = append([]byte(nil), b...)
		} else {
			t.argumentos[i] = v
		}
	}
	return t.txLecturaPrueba.QueryRow(ctx, q, args...)
}

type dbAnexoJustificacionPrueba struct {
	t  *testing.T
	tx *txAnexoJustificacionPrueba
}

func (d dbAnexoJustificacionPrueba) BeginTx(_ context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	if d.tx == nil || o.IsoLevel != pgx.Serializable || o.AccessMode != pgx.ReadWrite {
		d.t.Fatal("transacción anexo no serializable")
	}
	return d.tx, nil
}

func materialYRegistroJustificacion(t *testing.T) (domain.MaterialJustificacion, domain.Justificacion, ports.RegistroDocumentalConfirmado) {
	t.Helper()
	ref := func(c string) string { return "ref:" + strings.Repeat(c, 64) }
	v := domain.VinculoJustificacion{SolicitudRef: "permiso:cronos:solicitud:ensayo001", EmpleadoRef: empleadoPrueba,
		CatalogoVersionRef: "catalogo:permiso:v1", PermisoRef: "permiso:neutral", ExpedienteDocumentalRef: ref("c"),
		Documento: domain.DocumentoJustificacion{ID: ref("d"), Version: 1, SHA256: strings.Repeat("e", 64), CustodioID: "custodia.interna", CustodiaRef: "original:ensayo001"}}
	actor := actorPrueba(t, empleadoPrueba)
	m := domain.MaterialJustificacion{ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, ClaveOperacion: ref("f"),
		Accion: domain.AccionAnexarJustificacion, SolicitudVersion: 3, VersionEsperada: 0,
		PoliticaRef: "politica:justificacion:v1", PoliticaVersion: 1, PoliticaSHA256: strings.Repeat("a", 64), Vinculo: v}
	ahora := time.Date(2026, 10, 2, 8, 0, 0, 0, time.UTC)
	registro := ports.RegistroDocumentalConfirmado{Documento: v.Documento, ModuloID: "cronos", ExpedienteRef: v.ExpedienteDocumentalRef,
		TipoRef: ref("b"), NumeroVEC: "VEC-2026-14", CreadoEnUTC: ahora, PoliticaRef: ref("1"), PoliticaVersion: 1,
		PoliticaSHA256: strings.Repeat("2", 64), ConservacionHastaUTC: ahora.AddDate(5, 0, 0), Proteccion: "conservacion", EstadoPolitica: "aprobada"}
	return m, domain.Justificacion{Vinculo: v, Version: 1, Estado: domain.JustificacionPendiente}, registro
}

func TestErrorJustificacionDistinguePoliticaYCAS(t *testing.T) {
	for codigo, esperado := range map[string]error{
		"PC001": domain.ErrJustificacionInvalida,
		"PC002": domain.ErrJustificacionConflicto,
		"PC011": ports.ErrDependenciaNoDisponible,
		"PC015": ports.ErrPoliticaJustificacionNoVigente,
	} {
		if err := errorJustificacion(context.Background(), &pgconn.PgError{Code: codigo}); !errors.Is(err, esperado) {
			t.Fatalf("SQLSTATE %s: %v", codigo, err)
		}
	}
}

func TestRepositorioJustificacionAnexoTransportaConfirmacionYMaterialSeparados(t *testing.T) {
	m, j, registro := materialYRegistroJustificacion(t)
	recurso, err := application.RecursoJustificacion(m)
	if err != nil {
		t.Fatal(err)
	}
	v3 := exportacionPrueba(t, domain.AccionAnexarJustificacion, application.AudienciaJustificacion, recurso)
	h, _ := m.Huella()
	recibo := ports.ReciboJustificacion{Justificacion: j, Registro: &registro, HuellaMaterial: h,
		ReciboRef: "recibo:cronos:0b9f3c2e-1d4a-4c6b-9e8f-0a1b2c3d4e5f", FechaUTC: time.Date(2026, 10, 2, 8, 1, 0, 0, time.UTC)}
	salida, _ := json.Marshal(recibo)
	salida = []byte(strings.ReplaceAll(string(salida), `Z"`, `+00:00"`))
	tx := &txAnexoJustificacionPrueba{txLecturaPrueba: &txLecturaPrueba{respuestas: [][]byte{salida}}}
	r := &RepositorioJustificacion{db: dbAnexoJustificacionPrueba{t: t, tx: tx}, docConfirmador: proveedorConfirmacionDocPrueba{t: t}}
	got, err := r.ConfirmarJustificacion(context.Background(), m, j, &registro, v3)
	if err != nil || tx.commits != 1 || len(tx.consultas) != 1 || tx.consultas[0] != consultaAnexarJustificacion || got.ReciboRef != recibo.ReciboRef ||
		got.FechaUTC.Location() != time.UTC || got.Registro == nil || got.Registro.CreadoEnUTC.Location() != time.UTC || got.Registro.ConservacionHastaUTC.Location() != time.UTC {
		t.Fatal("anexo sin confirmacion propia", err, got, tx.consultas)
	}
	var enviado ports.RegistroDocumentalConfirmado
	if json.Unmarshal([]byte(tx.materiales[0]), &enviado) != nil || !mismoRegistroSQL(enviado, registro) {
		t.Fatal("no se transportó la confirmación exacta")
	}
	if len(tx.argumentos) != 24 || tx.argumentos[13] != string(mustCanonicoJustificacion(t, m)) {
		t.Fatal("anexo no transportó ambas V3 y material exacto", len(tx.argumentos))
	}
	var preimagen struct {
		ActorRef             string `json:"actor_ref"`
		SolicitudRef         string `json:"solicitud_ref"`
		MaterialEnlaceSHA256 string `json:"material_enlace_sha256"`
	}
	if json.Unmarshal(tx.argumentos[1].([]byte), &preimagen) != nil || preimagen.ActorRef != m.ActorRef || preimagen.SolicitudRef != m.Vinculo.SolicitudRef || preimagen.MaterialEnlaceSHA256 != h {
		t.Fatal("preimagen Doc no liga material Cronos")
	}
	var auth struct {
		PrincipalID     string `json:"principal_id"`
		PerfilActivoRef string `json:"perfil_activo_ref"`
	}
	if json.Unmarshal([]byte(tx.argumentos[2].(string)), &auth) != nil || auth.PrincipalID != m.ActorRef || auth.PerfilActivoRef == m.PerfilRef {
		t.Fatal("perfil Doc derivado indebidamente de Cronos")
	}
	ajena := exportacionPrueba(t, domain.AccionRevisarJustificacion, application.AudienciaJustificacion, recurso)
	fallo := &RepositorioJustificacion{db: dbLecturaPrueba{t: t}}
	if _, err := fallo.ConfirmarJustificacion(context.Background(), m, j, &registro, ajena); !errors.Is(err, ports.ErrJustificacionNoDisponible) {
		t.Fatal("acepta V3 de otra acción", err)
	}
	for _, caso := range []string{"principal ajeno", "audiencia ajena"} {
		t.Run(caso, func(t *testing.T) {
			fallo.docConfirmador = proveedorConfirmacionDocPrueba{t: t, principalCambiado: caso == "principal ajeno", audienciaCambiada: caso == "audiencia ajena"}
			if _, err := fallo.ConfirmarJustificacion(context.Background(), m, j, &registro, v3); !errors.Is(err, ports.ErrJustificacionNoDisponible) {
				t.Fatal("acepta material Doc ajeno", err)
			}
		})
	}
}

func mustCanonicoJustificacion(t *testing.T, m domain.MaterialJustificacion) []byte {
	t.Helper()
	b, err := m.Canonico()
	if err != nil {
		t.Fatal(err)
	}
	return b
}

type proveedorLecturaJustificacionPrueba struct {
	t *testing.T
}

func (p *proveedorLecturaJustificacionPrueba) ProveerMaterialConsultaJustificacion(_ context.Context, m domain.MaterialConsultaJustificacion) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	r, err := application.RecursoConsultaJustificacion(m)
	if err != nil {
		p.t.Fatal(err)
	}
	return exportacionPrueba(p.t, application.AccionConsultaJustificacion, application.AudienciaConsultaJustificacion, r), nil
}

func (p *proveedorLecturaJustificacionPrueba) ProveerMaterialReciboJustificacion(_ context.Context, m domain.MaterialReciboJustificacion) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	r, err := application.RecursoReciboJustificacion(m)
	if err != nil {
		p.t.Fatal(err)
	}
	return exportacionPrueba(p.t, application.AccionReciboJustificacion, application.AudienciaReciboJustificacion, r), nil
}

type proveedorEscrituraJustificacionPrueba struct{}

func (proveedorEscrituraJustificacionPrueba) ProveerMaterialJustificacion(context.Context, domain.MaterialJustificacion) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrJustificacionNoDisponible
}

func TestRepositorioJustificacionRecuperaConLecturaNueva(t *testing.T) {
	m, j, registro := materialYRegistroJustificacion(t)
	actor := actorPrueba(t, empleadoPrueba)
	orden, err := ports.NuevaOrdenJustificacion(actor, proveedorEscrituraJustificacionPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	h, _ := m.Huella()
	recibo := ports.ReciboJustificacion{Justificacion: j, Registro: &registro, HuellaMaterial: h,
		ReciboRef: "recibo:cronos:0b9f3c2e-1d4a-4c6b-9e8f-0a1b2c3d4e5f", FechaUTC: time.Date(2026, 10, 2, 8, 1, 0, 0, time.UTC)}
	salida, _ := json.Marshal(struct {
		Encontrado bool                      `json:"encontrado"`
		Recibo     ports.ReciboJustificacion `json:"recibo"`
	}{true, recibo})
	salida = []byte(strings.ReplaceAll(string(salida), `Z"`, `+00:00"`))
	tx := &txLecturaPrueba{respuestas: [][]byte{salida}}
	r := &RepositorioJustificacion{db: dbLecturaPrueba{t: t, tx: tx}, lecturas: &proveedorLecturaJustificacionPrueba{t}}
	got, ok, err := r.RecuperarJustificacion(context.Background(), orden, m)
	if err != nil || !ok || !got.Replay || tx.commits != 1 || tx.consultas[0] != consultaReciboJustificacion || !got.FechaUTC.Equal(recibo.FechaUTC) ||
		got.FechaUTC.Location() != time.UTC || got.Registro == nil || got.Registro.CreadoEnUTC.Location() != time.UTC || got.Registro.ConservacionHastaUTC.Location() != time.UTC {
		t.Fatal("recuperación sin recibo histórico", err, got)
	}
	var lectura domain.MaterialReciboJustificacion
	if json.Unmarshal([]byte(tx.materiales[0]), &lectura) != nil || lectura.HuellaMaterial != h || lectura.ClaveOperacion != m.ClaveOperacion {
		t.Fatal("material de lectura no liga clave y huella")
	}
	tx = &txLecturaPrueba{respuestas: [][]byte{[]byte(`{"encontrado":false}`)}}
	r.db = dbLecturaPrueba{t: t, tx: tx}
	if _, ok, err := r.RecuperarJustificacion(context.Background(), orden, m); err != nil || ok || tx.commits != 1 {
		t.Fatal("ausencia confundida con recibo", err, ok)
	}
}

type filaEmpleadoJustificacion struct{ ref string }

func (f filaEmpleadoJustificacion) Scan(destino ...any) error {
	*(destino[0].(*string)) = f.ref
	return nil
}

type dbFuenteJustificacionPrueba struct {
	dbLecturaPrueba
	empleado string
}

func (d dbFuenteJustificacionPrueba) QueryRow(_ context.Context, q string, args ...any) pgx.Row {
	if q != consultaResolverEmpleadoJustificacion || len(args) != 1 {
		d.t.Fatal("resolver técnico alterado")
	}
	return filaEmpleadoJustificacion{ref: d.empleado}
}

func TestFuenteJustificacionConsultaPoliticaYVinculoNominal(t *testing.T) {
	m, j, registro := materialYRegistroJustificacion(t)
	_ = registro
	actor := actorPrueba(t, empleadoPrueba)
	orden, err := ports.NuevaOrdenJustificacion(actor, proveedorEscrituraJustificacionPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	p := domain.PoliticaJustificacion{Referencia: m.PoliticaRef, Version: m.PoliticaVersion, SHA256: m.PoliticaSHA256,
		CatalogoVersionRef: m.Vinculo.CatalogoVersionRef, PermisoRef: m.Vinculo.PermisoRef, TipoDocumentalRef: "ref:" + strings.Repeat("b", 64),
		CustodioID: m.Vinculo.Documento.CustodioID, MotivosRef: []string{"motivo:documentacion:conforme"}}
	s := domain.SolicitudJustificable{SolicitudRef: m.Vinculo.SolicitudRef, EmpleadoRef: m.Vinculo.EmpleadoRef,
		CatalogoVersionRef: m.Vinculo.CatalogoVersionRef, PermisoRef: m.Vinculo.PermisoRef,
		ExpedienteDocumentalRef: m.Vinculo.ExpedienteDocumentalRef, Version: m.SolicitudVersion,
		Estado: domain.EstadoPermisoConcedido, JustificanteExigido: true}
	salida, _ := json.Marshal(ports.PreparacionJustificacion{Solicitud: s, Politica: p, PoliticaVigente: true, Actual: &j})
	tx := &txLecturaPrueba{respuestas: [][]byte{salida}}
	f := &FuenteJustificacion{db: dbFuenteJustificacionPrueba{dbLecturaPrueba: dbLecturaPrueba{t: t, tx: tx}, empleado: empleadoPrueba}, lecturas: &proveedorLecturaJustificacionPrueba{t}}
	got, err := f.PrepararJustificacion(context.Background(), orden, s.SolicitudRef)
	if err != nil || got.Solicitud != s || got.Actual == nil || tx.commits != 1 || tx.consultas[0] != consultaPrepararJustificacion {
		t.Fatal("fuente sin autoridad o política", err, got)
	}
}
