package bootstrap

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	postgresidentidad "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
)

func seudonimosCuentaNominalDesarrolloPrueba() postgresidentidad.SeudonimosAlta {
	return postgresidentidad.SeudonimosAlta{
		Esquema:          postgresidentidad.EsquemaHMACSHA256V1,
		EspacioIdentidad: espacioIdentidadSesionDesarrollo,
		DominioRef:       dominioIdentidadSesionDesarrollo,
		ClaveID:          "vec.identidad.desarrollo.g1", ClaveVersion: 1,
		CuentaIDHMAC: [32]byte{1}, SujetoIDHMAC: [32]byte{2},
	}
}

// Este doble prueba secuencia, parámetros, rollback y replay del preparador.
// No sustituye la comprobación PostgreSQL del hito ni afirma ejecutar SQL.
type baseCuentaNominalDesarrolloPrueba struct {
	existe, incompatibilidad, guardasInvalidas bool
	fallo                                      string
	inicios, incorporaciones, confirmaciones   int
	transacciones                              []*txCuentaNominalDesarrolloPrueba
	cancelarEnAlias                            context.CancelFunc
}

func (b *baseCuentaNominalDesarrolloPrueba) BeginTx(
	_ context.Context, opciones pgx.TxOptions,
) (pgx.Tx, error) {
	b.inicios++
	if opciones.IsoLevel != pgx.Serializable || opciones.AccessMode != pgx.ReadWrite {
		return nil, errors.New("aislamiento inesperado")
	}
	if b.fallo == "inicio" {
		return nil, errors.New("fallo simulado")
	}
	tx := &txCuentaNominalDesarrolloPrueba{base: b}
	b.transacciones = append(b.transacciones, tx)
	return tx, nil
}

type txCuentaNominalDesarrolloPrueba struct {
	pgx.Tx
	base                          *baseCuentaNominalDesarrolloPrueba
	pasos                         []string
	creada, confirmada, revertida bool
	cuenta                        string
	alias                         []any
}

func (tx *txCuentaNominalDesarrolloPrueba) Exec(
	_ context.Context, sql string, args ...any,
) (pgconn.CommandTag, error) {
	paso := ""
	switch sql {
	case configurarCuentaNominalDesarrolloSQL:
		paso = "configuracion"
	case incorporarCuentaNominalDesarrolloSQL:
		paso = "incorporar"
		tx.creada = true
		if args[0] != tx.cuenta {
			return pgconn.CommandTag{}, errors.New("cuenta inesperada")
		}
	default:
		return pgconn.CommandTag{}, errors.New("SQL fuera del alcance")
	}
	tx.pasos = append(tx.pasos, paso)
	if tx.base.fallo == paso {
		return pgconn.CommandTag{}, errors.New("fallo simulado")
	}
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}

type filaCuentaNominalDesarrolloPrueba func(...any) error

func (f filaCuentaNominalDesarrolloPrueba) Scan(destinos ...any) error { return f(destinos...) }

func (tx *txCuentaNominalDesarrolloPrueba) QueryRow(
	_ context.Context, sql string, args ...any,
) pgx.Row {
	return filaCuentaNominalDesarrolloPrueba(func(destinos ...any) error {
		paso := ""
		switch sql {
		case consultarCuentaNominalDesarrolloSQL:
			paso = "consultar"
			tx.cuenta = args[0].(string)
			*destinos[0].(*bool) = !tx.base.guardasInvalidas
			*destinos[1].(*bool) = tx.base.existe
		case cotejarCuentaNominalDesarrolloSQL:
			paso = "cotejar"
			*destinos[0].(*bool) = !tx.base.incompatibilidad
		case registrarAliasCuentaNominalDesarrolloSQL:
			paso = "alias"
			tx.alias = append([]any(nil), args...)
			*destinos[0].(*string) = tx.cuenta
			if tx.base.fallo == "alias_otra_cuenta" {
				*destinos[0].(*string) = "cta_ajena"
			}
			if tx.base.cancelarEnAlias != nil {
				tx.base.cancelarEnAlias()
			}
		default:
			return errors.New("SQL fuera del alcance")
		}
		tx.pasos = append(tx.pasos, paso)
		if tx.base.fallo == paso {
			return errors.New("fallo simulado")
		}
		return nil
	})
}

func (tx *txCuentaNominalDesarrolloPrueba) Commit(context.Context) error {
	tx.pasos = append(tx.pasos, "commit")
	if tx.base.fallo == "commit" {
		return errors.New("fallo simulado")
	}
	tx.confirmada = true
	tx.base.confirmaciones++
	if tx.creada {
		tx.base.existe = true
		tx.base.incorporaciones++
	}
	return nil
}

func (tx *txCuentaNominalDesarrolloPrueba) Rollback(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if !tx.confirmada {
		tx.revertida = true
	}
	return nil
}

func TestPrepararCuentaNominalDesarrolloAltaYReplaySinAutenticacion(t *testing.T) {
	soporte, _, _ := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	base := &baseCuentaNominalDesarrolloPrueba{}
	seudonimos := seudonimosCuentaNominalDesarrolloPrueba()
	if err := prepararCuentaNominalConsultasDesarrolloConTransaccion(
		context.Background(), base, soporte, seudonimos,
	); err != nil {
		t.Fatal(err)
	}
	inicial := base.transacciones[0]
	if !reflect.DeepEqual(inicial.pasos, []string{"configuracion", "consultar", "incorporar", "cotejar", "alias", "commit"}) {
		t.Fatalf("secuencia inicial: %v", inicial.pasos)
	}
	datos, err := soporte.contexto.Vinculo.Datos()
	if err != nil || inicial.cuenta != datos.CuentaRef || inicial.alias[1] != datos.CuentaRef {
		t.Fatal("no conservó la cuenta del soporte")
	}
	if !reflect.DeepEqual(inicial.alias[6], seudonimos.CuentaIDHMAC[:]) ||
		!reflect.DeepEqual(inicial.alias[7], seudonimos.SujetoIDHMAC[:]) {
		t.Fatal("no utilizó los digests nominales suministrados")
	}
	// Repetición de arranque: cambiar estos campos NO genera otro acto/alias.
	seudonimos.AsercionIDHMAC = [32]byte{31}
	seudonimos.SesionIDHMAC = [32]byte{32}
	if err := prepararCuentaNominalConsultasDesarrolloConTransaccion(
		context.Background(), base, soporte, seudonimos,
	); err != nil {
		t.Fatal(err)
	}
	replay := base.transacciones[1]
	if !reflect.DeepEqual(replay.pasos, []string{"configuracion", "consultar", "cotejar", "alias", "commit"}) ||
		!reflect.DeepEqual(inicial.alias, replay.alias) || base.incorporaciones != 1 ||
		base.confirmaciones != 2 {
		t.Fatal("replay no idempotente o dependiente de aserción/sesión")
	}
}

func TestPrepararCuentaNominalDesarrolloDeniegaAntesDeTransaccion(t *testing.T) {
	casos := []struct {
		nombre  string
		cambiar func(*soporteAltaContratacionTemporalDesarrollo, *postgresidentidad.SeudonimosAlta)
	}{
		{"sin_sello", func(s *soporteAltaContratacionTemporalDesarrollo, _ *postgresidentidad.SeudonimosAlta) { s.sello = nil }},
		{"certificado_distinto", func(s *soporteAltaContratacionTemporalDesarrollo, _ *postgresidentidad.SeudonimosAlta) {
			s.certificadoSHA256 = strings.Repeat("e", 64)
		}},
		{"certificado_invalido", func(s *soporteAltaContratacionTemporalDesarrollo, _ *postgresidentidad.SeudonimosAlta) {
			s.certificadoSHA256 = "no-certificado"
		}},
		{"principal_distinto", func(s *soporteAltaContratacionTemporalDesarrollo, _ *postgresidentidad.SeudonimosAlta) {
			s.principalID = "principal_ajeno"
		}},
		{"contexto_alterado", func(s *soporteAltaContratacionTemporalDesarrollo, _ *postgresidentidad.SeudonimosAlta) {
			s.contexto.Resultado.Contexto.Instantanea.CuentaRef = "cta_ajena"
		}},
		{"namespace_externo", func(_ *soporteAltaContratacionTemporalDesarrollo, p *postgresidentidad.SeudonimosAlta) {
			p.EspacioIdentidad = "https://corporativo.invalid/identidad"
		}},
		{"dominio_externo", func(_ *soporteAltaContratacionTemporalDesarrollo, p *postgresidentidad.SeudonimosAlta) {
			p.DominioRef = "idh_ajeno"
		}},
		{"esquema_externo", func(_ *soporteAltaContratacionTemporalDesarrollo, p *postgresidentidad.SeudonimosAlta) {
			p.Esquema = "otro"
		}},
		{"clave_inconsistente", func(_ *soporteAltaContratacionTemporalDesarrollo, p *postgresidentidad.SeudonimosAlta) {
			p.ClaveVersion = 2
		}},
		{"version_fuera_rango", func(_ *soporteAltaContratacionTemporalDesarrollo, p *postgresidentidad.SeudonimosAlta) {
			p.ClaveVersion = ^uint64(0)
		}},
		{"cuenta_sin_digest", func(_ *soporteAltaContratacionTemporalDesarrollo, p *postgresidentidad.SeudonimosAlta) {
			p.CuentaIDHMAC = [32]byte{}
		}},
		{"sujeto_sin_digest", func(_ *soporteAltaContratacionTemporalDesarrollo, p *postgresidentidad.SeudonimosAlta) {
			p.SujetoIDHMAC = [32]byte{}
		}},
		{"digests_iguales", func(_ *soporteAltaContratacionTemporalDesarrollo, p *postgresidentidad.SeudonimosAlta) {
			p.SujetoIDHMAC = p.CuentaIDHMAC
		}},
		{"cuenta_privilegiada", func(_ *soporteAltaContratacionTemporalDesarrollo, p *postgresidentidad.SeudonimosAlta) {
			p.CuentaOrdinariaIDHMAC = [32]byte{3}
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			soporte, _, _ := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
			base := &baseCuentaNominalDesarrolloPrueba{}
			seudonimos := seudonimosCuentaNominalDesarrolloPrueba()
			caso.cambiar(soporte, &seudonimos)
			err := prepararCuentaNominalConsultasDesarrolloConTransaccion(context.Background(), base, soporte, seudonimos)
			if !errors.Is(err, ports.ErrConsultaRRHHNoDisponible) || base.inicios != 0 {
				t.Fatalf("entrada incompatible alcanzó PostgreSQL: %v", err)
			}
		})
	}
}

func TestPrepararCuentaNominalDesarrolloRollbackYActividadAntesDeReplay(t *testing.T) {
	for _, fallo := range []string{"inicio", "configuracion", "consultar", "incorporar", "cotejar", "alias", "alias_otra_cuenta", "commit", "estado_incompatible", "guardas_invalidas"} {
		t.Run(fallo, func(t *testing.T) {
			soporte, _, _ := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
			base := &baseCuentaNominalDesarrolloPrueba{fallo: fallo}
			if fallo == "estado_incompatible" {
				base.existe, base.incompatibilidad = true, true
			}
			base.guardasInvalidas = fallo == "guardas_invalidas"
			err := prepararCuentaNominalConsultasDesarrolloConTransaccion(
				context.Background(), base, soporte, seudonimosCuentaNominalDesarrolloPrueba(),
			)
			if !errors.Is(err, ports.ErrConsultaRRHHNoDisponible) || base.confirmaciones != 0 || base.incorporaciones != 0 {
				t.Fatalf("fallo no atómico: %v", err)
			}
			if len(base.transacciones) > 0 {
				tx := base.transacciones[0]
				if !tx.revertida || tx.confirmada {
					t.Fatal("no revirtió la transacción")
				}
				if (base.incompatibilidad || base.guardasInvalidas) && tx.alias != nil {
					t.Fatal("recuperó alias sin cotejar actividad/guardas")
				}
			}
		})
	}
}

func TestPrepararCuentaNominalDesarrolloCancelacionYNulos(t *testing.T) {
	soporte, _, _ := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	seudonimos := seudonimosCuentaNominalDesarrolloPrueba()
	if err := prepararCuentaNominalConsultasDesarrollo(context.Background(), nil, soporte, seudonimos); err == nil {
		t.Fatal("aceptó pool nulo")
	}
	for _, ctx := range []context.Context{nil, context.Background()} {
		base := &baseCuentaNominalDesarrolloPrueba{}
		if err := prepararCuentaNominalConsultasDesarrolloConTransaccion(ctx, base, nil, seudonimos); err == nil || base.inicios != 0 {
			t.Fatal("aceptó soporte/contexto nulo")
		}
	}
	for _, antes := range []bool{true, false} {
		ctx, cancelar := context.WithCancel(context.Background())
		defer cancelar()
		base := &baseCuentaNominalDesarrolloPrueba{}
		if antes {
			cancelar()
		} else {
			base.cancelarEnAlias = cancelar
		}
		err := prepararCuentaNominalConsultasDesarrolloConTransaccion(ctx, base, soporte, seudonimos)
		if !errors.Is(err, context.Canceled) || base.confirmaciones != 0 || base.incorporaciones != 0 {
			t.Fatalf("cancelación no conservada: %v", err)
		}
		if !antes && !base.transacciones[0].revertida {
			t.Fatal("cancelación sin rollback independiente")
		}
	}
}

func TestPrepararCuentaNominalDesarrolloSQLAcotado(t *testing.T) {
	// Guardas de forma, no ejecución SQL: una cuenta existente nunca se
	// reactiva/repara y el estado se bloquea incluso al recuperar un alias.
	for _, guarda := range []string{
		"NOT cuenta.cuenta_privilegiada", "cuenta.cuenta_ordinaria_ref IS NULL",
		"actual.revision = 1", "estado.revision = 1", "estado.estado = 'activa'",
		"cuenta.acto_ref = $2::text", "estado.acto_ref = $3::text", "actual.acto_ref = $4::text",
		"historia.revision <> 1", "FOR UPDATE OF actual",
	} {
		if !strings.Contains(cotejarCuentaNominalDesarrolloSQL, guarda) {
			t.Fatalf("falta guarda %q", guarda)
		}
	}
	sql := strings.ToLower(configurarCuentaNominalDesarrolloSQL + consultarCuentaNominalDesarrolloSQL +
		incorporarCuentaNominalDesarrolloSQL + cotejarCuentaNominalDesarrolloSQL + registrarAliasCuentaNominalDesarrolloSQL)
	for _, prohibido := range []string{"delete ", "update vec_", "on conflict", "consumo_asercion", "registrar_sesion", "sesion_autenticacion", "control_sesion", "vec_contexto_actor"} {
		if strings.Contains(sql, prohibido) {
			t.Fatalf("SQL fuera del alcance: %s", prohibido)
		}
	}
	if strings.Count(sql, "insert into ") != 3 {
		t.Fatal("alta distinta de cuenta/estado/puntero")
	}
}
