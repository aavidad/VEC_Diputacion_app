package plannominal

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
	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/pruebas"
)

type relojGobiernoPrueba struct{ t time.Time }

func (r relojGobiernoPrueba) Ahora() time.Time { return r.t }

type filaGobiernoPrueba struct {
	bruto string
	err   error
}

func (f filaGobiernoPrueba) Scan(d ...any) error {
	if f.err != nil {
		return f.err
	}
	*(d[0].(*[]byte)) = []byte(f.bruto)
	return nil
}

type txGobiernoPrueba struct {
	pgx.Tx
	fila               filaGobiernoPrueba
	args               []any
	sql                string
	commits, rollbacks int
}

func (t *txGobiernoPrueba) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	t.sql = sql
	// Copia: el adaptador borra sus buffers al terminar.
	t.args = make([]any, len(args))
	for i, a := range args {
		if b, ok := a.([]byte); ok {
			a = bytes.Clone(b)
		}
		t.args[i] = a
	}
	return t.fila
}
func (t *txGobiernoPrueba) Commit(context.Context) error   { t.commits++; return nil }
func (t *txGobiernoPrueba) Rollback(context.Context) error { t.rollbacks++; return nil }

type poolGobiernoPrueba struct {
	tx      *txGobiernoPrueba
	inicios int
}

func (p *poolGobiernoPrueba) BeginTx(_ context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	p.inicios++
	if o.IsoLevel != pgx.Serializable || o.AccessMode != pgx.ReadWrite {
		return nil, errors.New("aislamiento alterado")
	}
	return p.tx, nil
}

type emisorGobiernoPrueba struct {
	t        *testing.T
	ambito   AmbitoGobiernoPlanFirma
	alterar  string
	err      error
	llamadas int
}

func (e *emisorGobiernoPrueba) EmitirGobiernoPlanFirma(_ context.Context, _ vd.ContextoActor, _ vd.EvidenciaSesionAdministracionPerfiles,
	_ vd.InstantaneaAutorizacion, material []byte, _ string) (EmisionGobiernoPlanFirma, error) {
	e.llamadas++
	if e.err != nil {
		return EmisionGobiernoPlanFirma{}, e.err
	}
	accion, recurso, err := RecursoGobiernoPlanFirma(material, e.ambito)
	if err != nil {
		e.t.Fatal(err)
	}
	audiencia := AudienciaGobiernoPlanFirma
	switch e.alterar {
	case "audiencia":
		audiencia = "vec.admin.usuarios.consultar.v1"
	case "recurso":
		recurso.Atributos["revision"] = "9"
	}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		e.t.Fatal(err)
	}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	resumen, err := vp.NuevoResumenCapacidadAtestacionAutorizacionV3("dec_prueba", strings.Repeat("a", 64), strings.Repeat("b", 64),
		"ctx_prueba", strings.Repeat("c", 64), accion, recurso.Referencia, huella, audiencia, ahora, ahora.Add(3*time.Second))
	if err != nil {
		e.t.Fatal(err)
	}
	raiz, _ := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	m, err := vp.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), resumen, []byte("decision"),
		[]byte("motivo"), []byte("contexto"), 1, 1, []byte("payload"), []byte("sobre"), []byte("evidencia"), raiz)
	if err != nil {
		e.t.Fatal(err)
	}
	return EmisionGobiernoPlanFirma{Accion: accion, Recurso: recurso, Ambito: e.ambito, Material: m}, nil
}

type registradorGobiernoPrueba struct{ llamadas int }

func (r *registradorGobiernoPrueba) AppendIntentoAuditoria(context.Context, vp.OrdenIntentoAuditoria) (vp.AcuseIntentoAuditoria, error) {
	r.llamadas++
	return vp.AcuseIntentoAuditoria{}, errors.New("sin registro en la prueba")
}

func escenarioGobiernoPostgres(t *testing.T, respuesta filaGobiernoPrueba) (*AutoridadGobiernoPlanFirmaPostgreSQL, *poolGobiernoPrueba, *emisorGobiernoPrueba, *registradorGobiernoPrueba, SolicitudGobiernoPlanFirma) {
	t.Helper()
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	resultado, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(ahora, "per_"+strings.Repeat("a", 22), "prf_"+strings.Repeat("b", 22),
		vd.AuthMethodCertificate, vd.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	pool := &poolGobiernoPrueba{tx: &txGobiernoPrueba{fila: respuesta}}
	emisor := &emisorGobiernoPrueba{t: t, ambito: ambitoPrueba}
	registrador := &registradorGobiernoPrueba{}
	motivo := vd.ReferenciaEntradaCatalogo{CatalogoID: "motivos_admin", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("d", 64), EntradaClave: "motivo_gobierno"}
	a, err := NuevaAutoridadGobiernoPlanFirmaPostgreSQL(pool, emisor, registrador, ConfiguracionAuditoriaGobiernoPlanFirma{
		Proceso: "vec-admin", MotivoDenegado: motivo, MotivoError: motivo, Plazo: time.Second}, relojGobiernoPrueba{ahora})
	if err != nil {
		t.Fatal(err)
	}
	s := SolicitudGobiernoPlanFirma{Actor: resultado.Contexto,
		Evidencia: vd.EvidenciaSesionAdministracionPerfiles{ResultadoContexto: resultado, Vinculo: vinculo},
		Instantanea: vd.InstantaneaAutorizacion{AsignacionPerfil: vd.AsignacionPerfil{Ambitos: []vd.AmbitoPerfil{
			{Clave: "organizacion_ref", Valores: []string{ambitoPrueba.OrganizacionRef}}, {Clave: "unidad_ref", Valores: []string{ambitoPrueba.UnidadRef}}}}},
		Material:       materialGobiernoPrueba(t, "publicar", catalogoPublicadoPrueba(t)),
		CorrelacionRef: "correlacion_" + strings.Repeat("e", 32)}
	return a, pool, emisor, registrador, s
}

func respuestaGobiernoPrueba(t *testing.T, s SolicitudGobiernoPlanFirma, nuevo bool) filaGobiernoPrueba {
	t.Helper()
	_, recurso, err := RecursoGobiernoPlanFirma(s.Material, ambitoPrueba)
	if err != nil {
		t.Fatal(err)
	}
	huella, _ := recurso.HuellaContextoAutorizacionSHA256()
	b, _ := json.Marshal(map[string]any{
		"recibo": map[string]any{"recibo_ref": "recibo:8c2a1f04-3b7e-4c1d-9a6b-0e5f7d2c4b1a", "estado": "publicado", "revision": 2,
			"huella_comun": strings.Repeat("f", 64), "publicacion_sha256": strings.Repeat("f", 64), "actor_ref": "per_x",
			"confirmado_en": "2026-10-05T18:00:00Z", "auditoria_ref": "aud-original", "outbox_recibo_ref": "recibo:8c2a1f04-3b7e-4c1d-9a6b-0e5f7d2c4b1a"},
		"consumo": map[string]any{"decision_ref": "dec_prueba", "efecto_ref": recurso.Referencia, "huella_efecto_sha256": huella,
			"consumo_huella_sha256": strings.Repeat("1", 64), "auditoria_ref": "aud-consumo", "consumida_en": "2026-10-05T18:00:01Z", "consumo_nuevo": nuevo}})
	return filaGobiernoPrueba{bruto: string(b)}
}

// Un gobierno coherente consume la decisión con AD201 en una transacción
// serializable y confirma: los ámbitos salen de la asignación, no de la petición.
func TestGobiernoPlanFirmaConfirmaConAD201(t *testing.T) {
	a, pool, emisor, registrador, s := escenarioGobiernoPostgres(t, filaGobiernoPrueba{})
	pool.tx.fila = respuestaGobiernoPrueba(t, s, true)
	recibo, err := a.GobernarPlanFirma(t.Context(), s)
	if err != nil {
		t.Fatalf("gobierno coherente rechazado: %v", err)
	}
	tx := pool.tx
	if emisor.llamadas != 1 || pool.inicios != 1 || tx.commits != 1 || registrador.llamadas != 0 || tx.sql != registrarGobiernoPlanFirmaSQL201 ||
		!bytes.Equal(tx.args[0].([]byte), s.Material) || tx.args[1] != ambitoPrueba.OrganizacionRef || tx.args[2] != ambitoPrueba.UnidadRef ||
		recibo.Accion != "vec.catalogos.publicar" || recibo.CatalogoRef != "ct.plan.firma.sintetico:1" || recibo.ConsumoAuditoriaRef != "aud-consumo" ||
		recibo.AuditoriaRef != "aud-original" || recibo.Estado != "publicado" {
		t.Fatalf("efecto o recibo distintos: %+v", recibo)
	}
}

// Ninguna incoherencia confirma: emisión ajena, consumo repetido, SQL
// rechazado o asignación sin ámbitos únicos. Sin sesión ADMIN válida para
// auditar, la respuesta es indisponibilidad y nunca un permiso o un 403.
func TestGobiernoPlanFirmaNoConfirmaIncoherencias(t *testing.T) {
	for _, caso := range []string{"audiencia", "recurso", "consumo_repetido", "sql_denegado", "sql_conflicto", "dos_unidades", "emisor_denegado"} {
		t.Run(caso, func(t *testing.T) {
			a, pool, emisor, registrador, s := escenarioGobiernoPostgres(t, filaGobiernoPrueba{})
			pool.tx.fila = respuestaGobiernoPrueba(t, s, true)
			switch caso {
			case "audiencia", "recurso":
				emisor.alterar = caso
			case "consumo_repetido":
				pool.tx.fila = respuestaGobiernoPrueba(t, s, false)
			case "sql_denegado":
				pool.tx.fila = filaGobiernoPrueba{err: &pgconn.PgError{Code: "42501"}}
			case "sql_conflicto":
				pool.tx.fila = filaGobiernoPrueba{err: &pgconn.PgError{Code: "40001"}}
			case "dos_unidades":
				s.Instantanea.AsignacionPerfil.Ambitos[1].Valores = append(s.Instantanea.AsignacionPerfil.Ambitos[1].Valores, "unidad:otra")
			case "emisor_denegado":
				emisor.err = vd.ErrAutorizacionDenegada
			}
			_, err := a.GobernarPlanFirma(t.Context(), s)
			if !errors.Is(err, ErrGobiernoPlanFirmaNoDisponible) || pool.tx.commits != 0 {
				t.Fatalf("%s confirmado o sin cerrar: %v", caso, err)
			}
			if caso == "dos_unidades" && (emisor.llamadas != 0 || pool.inicios != 0) {
				t.Fatal("ámbito ambiguo alcanzó el PDP o la base")
			}
			if (caso == "audiencia" || caso == "recurso" || caso == "emisor_denegado") && pool.inicios != 0 {
				t.Fatal("emisión ajena alcanzó la base")
			}
			_ = registrador
		})
	}
}

func TestErrorGobiernoSQLClasifica(t *testing.T) {
	for codigo, esperado := range map[string]error{"42501": vd.ErrAutorizacionDenegada, "40001": ErrGobiernoPlanFirmaConflicto,
		"23505": ErrGobiernoPlanFirmaConflicto, "22023": vd.ErrActoAdministracionPerfilesInvalido, "XX000": ErrGobiernoPlanFirmaNoDisponible} {
		if err := errorGobiernoSQL(t.Context(), &pgconn.PgError{Code: codigo}); !errors.Is(err, esperado) {
			t.Fatalf("%s → %v", codigo, err)
		}
	}
}

func TestAmbitoGobiernoDeAsignacion(t *testing.T) {
	a := vd.AsignacionPerfil{Ambitos: []vd.AmbitoPerfil{{Clave: "unidad_ref", Valores: []string{ambitoPrueba.UnidadRef}},
		{Clave: "organizacion_ref", Valores: []string{ambitoPrueba.OrganizacionRef}}}}
	if g, err := AmbitoGobiernoPlanFirmaDeAsignacion(a); err != nil || g != ambitoPrueba {
		t.Fatalf("ámbitos de la asignación no leídos: %v", err)
	}
	for nombre, mutar := range map[string]func(*vd.AsignacionPerfil){
		"una dimensión": func(x *vd.AsignacionPerfil) { x.Ambitos = x.Ambitos[:1] },
		"otra clave":    func(x *vd.AsignacionPerfil) { x.Ambitos[0].Clave = "centro_ref" },
		"dos valores":   func(x *vd.AsignacionPerfil) { x.Ambitos[0].Valores = []string{"a", "b"} },
		"org mal":       func(x *vd.AsignacionPerfil) { x.Ambitos[1].Valores = []string{"org_x"} },
	} {
		c := vd.AsignacionPerfil{Ambitos: []vd.AmbitoPerfil{{Clave: a.Ambitos[0].Clave, Valores: append([]string(nil), a.Ambitos[0].Valores...)},
			{Clave: a.Ambitos[1].Clave, Valores: append([]string(nil), a.Ambitos[1].Valores...)}}}
		mutar(&c)
		if _, err := AmbitoGobiernoPlanFirmaDeAsignacion(c); err == nil {
			t.Fatalf("%s aceptado", nombre)
		}
	}
}
