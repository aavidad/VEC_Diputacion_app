package postgres

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/json"
	"errors"
	"math/big"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	bolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	vec "vec-diputacion-granada/internal/vec/ports"
)

// Este material solo supera la frontera estructural del adaptador. El doble
// pgx comprueba sus llamadas, no acredita autorización ni instalación B63.
func materialPortalExternoPrueba(t *testing.T, ahora time.Time) vec.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	t.Helper()
	h := strings.Repeat("a", 64)
	r, e := vec.NuevoResumenCapacidadAtestacionAutorizacionV3("decision:prueba", h, h, "contexto:prueba", h, bolsa.AccionConsultarMiBolsa, "consulta:prueba", h, bolsa.AudienciaMiBolsa, ahora, ahora.Add(time.Second))
	if e != nil {
		t.Fatal(e)
	}
	pub := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x42}, ed25519.SeedSize)).Public()
	spki, e := x509.MarshalPKIXPublicKey(pub)
	if e != nil {
		t.Fatal(e)
	}
	m, e := vec.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte{'x'}, 512), r, []byte("decision"), []byte("motivo"), []byte("contexto"), 1, 1, []byte("payload"), []byte("sobre"), []byte("evidencia"), spki)
	if e != nil {
		t.Fatal(e)
	}
	return m
}

type llamadaPortalExternoPrueba struct {
	sql  string
	args []any
}
type txPortalExternoPrueba struct {
	pgx.Tx
	ahora          time.Time
	llamadas       []llamadaPortalExternoPrueba
	confirmaciones int
	reversiones    int
	fallo          error
}
type filaPortalExternoPrueba struct {
	valores []any
	fallo   error
}

func (f filaPortalExternoPrueba) Scan(destinos ...any) error {
	if f.fallo != nil {
		return f.fallo
	}
	if len(destinos) != len(f.valores) {
		return errors.New("columnas de prueba incompatibles")
	}
	for i, v := range f.valores {
		reflect.ValueOf(destinos[i]).Elem().Set(reflect.ValueOf(v))
	}
	return nil
}
func (tx *txPortalExternoPrueba) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	tx.llamadas = append(tx.llamadas, llamadaPortalExternoPrueba{sql, args})
	if strings.Contains(sql, "vec_bolsa_llamamientos.") && tx.fallo != nil {
		return pgconn.CommandTag{}, tx.fallo
	}
	return pgconn.NewCommandTag("SELECT 1"), nil
}
func (tx *txPortalExternoPrueba) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	tx.llamadas = append(tx.llamadas, llamadaPortalExternoPrueba{sql, args})
	if tx.fallo != nil {
		return filaPortalExternoPrueba{fallo: tx.fallo}
	}
	switch {
	case strings.Contains(sql, "consultar_mi_bolsa_"):
		b, e := json.Marshal(map[string]any{"consultada_en": tx.ahora, "participaciones": []any{}})
		return filaPortalExternoPrueba{valores: []any{b}, fallo: e}
	case strings.Contains(sql, "consultar_historial_mi_bolsa_"):
		b, e := json.Marshal(map[string]any{"consultada_en": tx.ahora, "pagina": 1, "tamano": 20, "hay_mas": false, "items": []any{}})
		return filaPortalExternoPrueba{valores: []any{b}, fallo: e}
	case strings.Contains(sql, "leer_portal_candidato_") || strings.Contains(sql, "leer_contacto_candidato_") || strings.Contains(sql, "listar_ofertas_candidato_"):
		return filaPortalExternoPrueba{valores: []any{[]byte("[]")}}
	case strings.Contains(sql, "solicitar_portal_candidato_"):
		return filaPortalExternoPrueba{valores: []any{false, "solicitud:prueba", "recibo:prueba", tx.ahora}}
	case strings.Contains(sql, "responder_llamamiento_portal_"):
		return filaPortalExternoPrueba{valores: []any{false, "respuesta:prueba", "recibo:prueba", tx.ahora, bolsa.ModoRespuestaPortalFirme}}
	case strings.Contains(sql, "manifestar_disposicion_oferta_"):
		return filaPortalExternoPrueba{valores: []any{false, "recibo:prueba", "oferta:prueba", tx.ahora}}
	case strings.Contains(sql, "confirmar_contacto_propio_"):
		return filaPortalExternoPrueba{valores: []any{false, "recibo:prueba", int64(1), tx.ahora}}
	default:
		return filaPortalExternoPrueba{fallo: errors.New("función no esperada")}
	}
}
func (tx *txPortalExternoPrueba) Commit(context.Context) error   { tx.confirmaciones++; return nil }
func (tx *txPortalExternoPrueba) Rollback(context.Context) error { tx.reversiones++; return nil }

type plazoPortalExternoPrueba struct{}

func (plazoPortalExternoPrueba) VencimientoRespuesta(_ context.Context, desde time.Time) (time.Time, error) {
	return desde.Add(time.Hour), nil
}

// Cada conjunto atraviesa los mismos métodos reales, incluidos los tres
// helpers y la preparación previa. La lista independiente fija el contrato
// B63 y detecta un retorno accidental a las funciones compartidas.
func TestPortalExternoB63OnceFachadasPorConstructor(t *testing.T) {
	base := []string{"consultar_mi_bolsa", "consultar_mi_bolsa_portal", "consultar_historial_mi_bolsa", "solicitar_portal_candidato", "responder_llamamiento_portal", "preparar_respuesta_portal", "leer_portal_candidato", "manifestar_disposicion_oferta", "listar_ofertas_candidato", "confirmar_contacto_propio", "leer_contacto_candidato"}
	for _, externo := range []bool{false, true} {
		nombre := "interno"
		sufijo := "_v1"
		if externo {
			nombre = "externo"
			sufijo = "_externo_v1"
		}
		t.Run(nombre, func(t *testing.T) {
			ahora := time.Date(2026, 9, 30, 10, 0, 0, 0, time.UTC)
			m := materialPortalExternoPrueba(t, ahora)
			tx := &txPortalExternoPrueba{ahora: ahora}
			inicio := &iniciadorLlamamientoPostgreSQLPrueba{tx: tx}
			pool := &pgxpool.Pool{}
			consulta, e := NuevaConsultaMiBolsaPostgreSQL(pool)
			if externo {
				consulta, e = NuevaConsultaMiBolsaExternaPostgreSQL(pool)
			}
			if e != nil {
				t.Fatal(e)
			}
			portal, e := NuevoRegistroPortalCandidatoPostgreSQL(pool)
			if externo {
				portal, e = NuevoRegistroPortalCandidatoExternoPostgreSQL(pool)
			}
			if e != nil {
				t.Fatal(e)
			}
			disposicion, e := NuevoRegistroDisposicionOfertaPostgreSQL(pool)
			if externo {
				disposicion, e = NuevoRegistroDisposicionOfertaExternoPostgreSQL(pool)
			}
			if e != nil {
				t.Fatal(e)
			}
			contacto, e := NuevoRegistroConfirmacionContactoPostgreSQL(pool)
			if externo {
				contacto, e = NuevoRegistroConfirmacionContactoExternoPostgreSQL(pool)
			}
			if e != nil {
				t.Fatal(e)
			}
			consulta.pool, portal.pool, disposicion.portal.pool, contacto.portal.pool = inicio, inicio, inicio, inicio
			ctx := context.Background()
			if _, e = consulta.ConsultarMiBolsa(ctx, bolsa.SolicitudConsultaMiBolsa{CandidatoRef: "can_prueba", ConsultadaEn: ahora, Material: m}); e != nil {
				t.Fatal(e)
			}
			if _, e = consulta.ConsultarMiBolsa(ctx, bolsa.SolicitudConsultaMiBolsa{CandidatoRef: "can_prueba", ConsultadaEn: ahora, Material: m, ResultadosEfectivos: []string{"enviado"}, LeerContacto: true, LeerOfertas: true}); e != nil {
				t.Fatal(e)
			}
			if _, e = consulta.ConsultarHistorialMiBolsa(ctx, bolsa.SolicitudConsultaHistorialMiBolsa{CandidatoRef: "can_prueba", ConsultadaEn: ahora, Material: m, Pagina: 1}); e != nil {
				t.Fatal(e)
			}
			if _, e = portal.SolicitarPortal(ctx, bolsa.SolicitudPortalCandidato{CandidatoRef: "can_prueba", Bolsa: "bolsa:prueba", RegistradaEn: ahora, Material: m}); e != nil {
				t.Fatal(e)
			}
			if _, e = portal.ResponderPortal(ctx, bolsa.RespuestaPortalCandidato{CandidatoRef: "can_prueba", Bolsa: "bolsa:prueba", RespondidaEn: ahora, ResultadosEfectivos: []string{"enviado"}, Material: m}, plazoPortalExternoPrueba{}); e != nil {
				t.Fatal(e)
			}
			if _, e = disposicion.ManifestarDisposicion(ctx, bolsa.DisposicionPortalCandidato{CandidatoRef: "can_prueba", OfertaRef: "oferta:prueba", ReciboRef: "recibo:prueba", ManifestadaEn: ahora, Material: m}); e != nil {
				t.Fatal(e)
			}
			if _, e = contacto.ConfirmarContacto(ctx, bolsa.ConfirmacionContactoPortal{CandidatoRef: "can_prueba", Bolsa: "bolsa:prueba", Version: 1, ReciboRef: "recibo:prueba", ConfirmadaEn: ahora, Material: m}); e != nil {
				t.Fatal(e)
			}
			vistas := map[string]bool{}
			for _, llamada := range tx.llamadas {
				pos := strings.Index(llamada.sql, "vec_bolsa_llamamientos.")
				if pos < 0 {
					continue
				}
				n := strings.SplitN(llamada.sql[pos:], "(", 2)[0]
				vistas[n] = true
				if len(llamada.args) >= 10 {
					esperado := []any{m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), pgtype.Numeric{Int: big.NewInt(1), Valid: true}, pgtype.Numeric{Int: big.NewInt(1), Valid: true}, m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI()}
					if !reflect.DeepEqual(llamada.args[len(llamada.args)-10:], esperado) {
						t.Fatal("el cambio de fachada alteró material o versiones", n)
					}
				}
				if externo != strings.HasSuffix(n, "_externo_v1") {
					t.Fatal("cruzó la frontera SQL", n)
				}
			}
			for _, f := range base {
				if !vistas["vec_bolsa_llamamientos."+f+sufijo] {
					t.Fatal("fachada no recorrida", f)
				}
			}
			if len(vistas) != 11 || inicio.inicios != 7 || tx.confirmaciones != 7 || inicio.opciones.IsoLevel != pgx.Serializable || inicio.opciones.AccessMode != pgx.ReadWrite {
				t.Fatalf("transacciones incompletas: funciones=%d inicios=%d commits=%d", len(vistas), inicio.inicios, tx.confirmaciones)
			}
		})
	}
}

func TestPortalExternoB63NoReintentaFachadaInternaSiExternaDenegada(t *testing.T) {
	ahora := time.Date(2026, 9, 30, 10, 0, 0, 0, time.UTC)
	m := materialPortalExternoPrueba(t, ahora)
	tx := &txPortalExternoPrueba{ahora: ahora, fallo: &pgconn.PgError{Code: "42501"}}
	inicio := &iniciadorLlamamientoPostgreSQLPrueba{tx: tx}
	r, e := NuevoRegistroPortalCandidatoExternoPostgreSQL(&pgxpool.Pool{})
	if e != nil {
		t.Fatal(e)
	}
	r.pool = inicio
	if _, e = r.SolicitarPortal(context.Background(), bolsa.SolicitudPortalCandidato{CandidatoRef: "can_prueba", Bolsa: "bolsa:prueba", RegistradaEn: ahora, Material: m}); !errors.Is(e, bolsa.ErrPortalCandidatoInvalido) {
		t.Fatal(e)
	}
	if len(tx.llamadas) != 2 || !strings.Contains(tx.llamadas[1].sql, "solicitar_portal_candidato_externo_v1(") || tx.confirmaciones != 0 || tx.reversiones != 1 {
		t.Fatal("denegación cambió de función o confirmó")
	}
}

func TestPortalExternoB63DescriptorVacioOMezcladoNoAbreTransaccion(t *testing.T) {
	ahora := time.Date(2026, 9, 30, 10, 0, 0, 0, time.UTC)
	m := materialPortalExternoPrueba(t, ahora)
	mezclado := funcionesPortalBolsaExternas()
	mezclado.prepararRespuesta = funcionPrepararRespuestaV1
	for _, f := range []funcionesPortalBolsa{{}, mezclado} {
		inicio := &iniciadorLlamamientoPostgreSQLPrueba{}
		consulta := &ConsultaMiBolsaPostgreSQL{pool: inicio, funciones: f}
		portal := &RegistroPortalCandidatoPostgreSQL{pool: inicio, funciones: f}
		if _, e := consulta.ConsultarMiBolsa(context.Background(), bolsa.SolicitudConsultaMiBolsa{CandidatoRef: "can_prueba", ConsultadaEn: ahora, Material: m}); !errors.Is(e, bolsa.ErrMaterialMiBolsaNoDisponible) {
			t.Fatal(e)
		}
		if _, e := consulta.ConsultarHistorialMiBolsa(context.Background(), bolsa.SolicitudConsultaHistorialMiBolsa{CandidatoRef: "can_prueba", ConsultadaEn: ahora, Material: m, Pagina: 1}); !errors.Is(e, bolsa.ErrHistorialMiBolsaNoDisponible) {
			t.Fatal(e)
		}
		if _, e := portal.SolicitarPortal(context.Background(), bolsa.SolicitudPortalCandidato{CandidatoRef: "can_prueba", Bolsa: "bolsa:prueba", RegistradaEn: ahora, Material: m}); !errors.Is(e, bolsa.ErrPortalCandidatoNoDisponible) {
			t.Fatal(e)
		}
		if inicio.inicios != 0 {
			t.Fatal("descriptor incompleto abrió PostgreSQL")
		}
	}
	for _, constructor := range []func(*pgxpool.Pool) error{
		func(p *pgxpool.Pool) error { _, e := NuevaConsultaMiBolsaExternaPostgreSQL(p); return e },
		func(p *pgxpool.Pool) error { _, e := NuevoRegistroPortalCandidatoExternoPostgreSQL(p); return e },
		func(p *pgxpool.Pool) error { _, e := NuevoRegistroDisposicionOfertaExternoPostgreSQL(p); return e },
		func(p *pgxpool.Pool) error { _, e := NuevoRegistroConfirmacionContactoExternoPostgreSQL(p); return e },
	} {
		if constructor(nil) == nil {
			t.Fatal("constructor exterior aceptó pool nulo")
		}
	}
}

func TestPortalExternoB63VersionCompatibleConCodecNumericPGX(t *testing.T) {
	mapa := pgtype.NewMap()
	for _, v := range []struct {
		valor    uint64
		esperada string
	}{{1, "1"}, {9007199254740991, "9007199254740991"}} {
		b, e := mapa.Encode(pgtype.NumericOID, pgtype.BinaryFormatCode, versionMaterialPortalBolsa(v.valor), nil)
		if e != nil {
			t.Fatal(e)
		}
		var recuperada pgtype.Numeric
		if e = mapa.Scan(pgtype.NumericOID, pgtype.BinaryFormatCode, b, &recuperada); e != nil || !recuperada.Valid || recuperada.Exp != 0 || recuperada.Int.String() != v.esperada {
			t.Fatal("codec numeric perdió la versión", e)
		}
	}
}
