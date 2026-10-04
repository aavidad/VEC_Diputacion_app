package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
	"vec-diputacion-granada/internal/vec/ports"
)

type poolTecnicoPrueba struct {
	eventos        []ports.EventoFronteraAdminTecnica
	txs            []*txTecnicoPrueba
	commitIncierto bool
	cancelado      bool
}

func (p *poolTecnicoPrueba) QueryRow(context.Context, string, ...any) pgx.Row {
	panic("consulta_fuera_transaccion")
}

func (p *poolTecnicoPrueba) BeginTx(ctx context.Context, opciones pgx.TxOptions) (pgx.Tx, error) {
	if opciones.IsoLevel != pgx.Serializable || opciones.AccessMode != pgx.ReadWrite {
		panic("transaccion_no_nominal")
	}
	p.cancelado = ctx.Err() != nil
	t := &txTecnicoPrueba{p: p}
	p.txs = append(p.txs, t)
	return t, nil
}

type txTecnicoPrueba struct {
	pgx.Tx
	p        *poolTecnicoPrueba
	rollback int
	commit   int
}

func (t *txTecnicoPrueba) QueryRow(_ context.Context, _ string, args ...any) pgx.Row {
	var e ports.EventoFronteraAdminTecnica
	if json.Unmarshal(args[0].([]byte), &e) != nil {
		panic("evento_no_json")
	}
	t.p.eventos = append(t.p.eventos, e)
	return filaTecnicaPrueba{e: e}
}
func (t *txTecnicoPrueba) Commit(context.Context) error {
	t.commit++
	if t.p.commitIncierto && len(t.p.txs) == 1 {
		return errors.New("commit_desconocido")
	}
	return nil
}
func (t *txTecnicoPrueba) Rollback(context.Context) error { t.rollback++; return nil }

type filaTecnicaPrueba struct {
	e ports.EventoFronteraAdminTecnica
}

func (r filaTecnicaPrueba) Scan(args ...any) error {
	b, err := json.Marshal(ports.AcuseFronteraAdminTecnica{AuditoriaRef: "aud_v3_fat_" + strings.TrimPrefix(r.e.EventoRef, "evento_"), Secuencia: 1, HuellaSHA256: strings.Repeat("a", 64), CorrelacionRef: r.e.CorrelacionRef, RegistradaEn: time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)})
	if err != nil {
		return err
	}
	*(args[0].(*[]byte)) = b
	return nil
}

type nominalTecnicoPrueba struct {
	llamadas int
	err      error
}

func (n *nominalTecnicoPrueba) RegistrarDenegacionADMIN(context.Context, api.DenegacionADMIN) error {
	n.llamadas++
	return n.err
}
func registradorTecnicoPrueba(p *poolTecnicoPrueba) *RegistradorFronteraTecnica {
	return &RegistradorFronteraTecnica{pool: p, login: "login_admin_ensayo", config: ConfiguracionFronteraTecnica{Proceso: "vec_admin", Canal: "administracion_privilegiada", Plazo: time.Second}, codigos: map[string]string{"autenticacion_requerida": "denegado", "servicio_no_disponible": "error"}}
}

func TestFronteraTecnicaAntesV2ConservaEventoEnCommitIncierto(t *testing.T) {
	p := &poolTecnicoPrueba{commitIncierto: true}
	nominal := &nominalTecnicoPrueba{}
	a, _ := NuevoAuditorFronteraCompuesto(nominal, registradorTecnicoPrueba(p))
	ctx, err := ports.ConCorrelacionIncidenciasPeticion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(ctx)
	cancel()
	if err := a.RegistrarDenegacionADMIN(ctx, api.DenegacionADMIN{Codigo: "autenticacion_requerida", RecursoRef: "SECRET_certificado_ip_query", ActorPersonaRef: "persona_inventada", CorrelacionRef: "cabecera_inventada"}); err != nil {
		t.Fatal(err)
	}
	if p.cancelado || nominal.llamadas != 0 || len(p.eventos) != 2 || p.eventos[0] != p.eventos[1] || p.eventos[0].Resultado != "denegado" || strings.Contains(p.eventos[0].RecursoRef, "SECRET") {
		t.Fatal("evento_identidad_o_contexto_inventados")
	}
	for _, tx := range p.txs {
		if tx.rollback != 1 || tx.commit != 1 {
			t.Fatal("transaccion_sin_cierre")
		}
	}
}
func TestFronteraNominalNuncaDerivaErrorAD169ATecnica(t *testing.T) {
	actor, evidencia := sesionFronteraPrueba(t, time.Now().UTC().Truncate(time.Microsecond))
	p := &poolTecnicoPrueba{}
	nominal := &nominalTecnicoPrueba{err: errors.New("AD169_no_disponible")}
	a, _ := NuevoAuditorFronteraCompuesto(nominal, registradorTecnicoPrueba(p))
	if a.RegistrarDenegacionADMIN(context.Background(), api.DenegacionADMIN{Codigo: "servicio_no_disponible", Actor: actor, Evidencia: evidencia}) == nil || nominal.llamadas != 1 || len(p.eventos) != 0 {
		t.Fatal("fallback_presta_actor")
	}
	evidencia.ResultadoContexto.RepresentacionCanonica = append([]byte(nil), evidencia.ResultadoContexto.RepresentacionCanonica...)
	evidencia.ResultadoContexto.RepresentacionCanonica[0] ^= 1
	if a.RegistrarDenegacionADMIN(context.Background(), api.DenegacionADMIN{Codigo: "servicio_no_disponible", Actor: actor, Evidencia: evidencia}) == nil || len(p.eventos) != 0 {
		t.Fatal("V2invalido_deriva_tecnica")
	}
}
func TestFronteraTecnicaNoEntregaAcuseAnteCommitNoConfirmado(t *testing.T) {
	p := &poolTecnicoPrueba{commitIncierto: true}
	r := registradorTecnicoPrueba(p)
	correlacion := "correlacion_" + strings.Repeat("b", 32)
	recurso, err := ports.RecursoFronteraAdminTecnica(correlacion)
	if err != nil {
		t.Fatal(err)
	}
	e := ports.EventoFronteraAdminTecnica{TipoRegistro: "frontera_admin_tecnica", EventoRef: "evento_" + strings.Repeat("a", 32), OperadorLogin: r.login, Accion: "controlar_frontera_admin_v1", RecursoRef: recurso, Resultado: "error", CodigoRef: "servicio_no_disponible", Proceso: r.config.Proceso, Canal: r.config.Canal, FinalidadRef: "control_frontera_admin", CorrelacionRef: correlacion}
	acuse, err := r.AppendFronteraAdminTecnica(context.Background(), e)
	if !errors.Is(err, ports.ErrFronteraAdminTecnicaCommitIncierto) || acuse != (ports.AcuseFronteraAdminTecnica{}) || len(p.eventos) != 1 || p.txs[0].rollback != 1 {
		t.Fatal("acuse_provisional")
	}
}

func TestFronteraTecnicaRechazaPlazoSuperiorAlNominal(t *testing.T) {
	p := &poolTecnicoPrueba{}
	r, err := nuevoRegistradorFronteraTecnica(context.Background(), p, ConfiguracionFronteraTecnica{Proceso: "vec_admin", Canal: "administracion_privilegiada", Plazo: 2*time.Second + time.Nanosecond})
	if r != nil || !errors.Is(err, ports.ErrFronteraAdminTecnicaNoDisponible) || len(p.txs) != 0 {
		t.Fatal("plazo_no_acotado_antes_db")
	}
}
