package controlrestauracionpg

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	ordenpg "vec-diputacion-granada/internal/modules/administracion/adapters/ordenescopias/postgres"
	dominio "vec-diputacion-granada/internal/modules/administracion/domain/ordenescopias"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/pruebas"
)

type materializadorFunc func(context.Context, CanalSQL, dominio.Orden, vecdomain.SolicitudAutorizacionLigadaV3, vecdomain.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3) (ordenpg.MaterialV3, error)

func (f materializadorFunc) MaterializarOrdenV3EnTX(ctx context.Context, canal CanalSQL, orden dominio.Orden, solicitud vecdomain.SolicitudAutorizacionLigadaV3, decision vecdomain.DecisionAutorizacionLigadaV3, confirmacion vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3) (ordenpg.MaterialV3, error) {
	return f(ctx, canal, orden, solicitud, decision, confirmacion)
}

type filaFunc func(...any) error

func (f filaFunc) Scan(destinos ...any) error { return f(destinos...) }

type txPrueba struct {
	pgx.Tx              // Unused pgx methods are deliberately unavailable in this unit double.
	t                   *testing.T
	pasos               []string
	consulta            string
	argumentos          int
	bruto               []byte
	fallar              string
	cancelar            context.CancelFunc
	commitConfirmado    bool
	efectoCommitFallido bool
	plazoCierre         time.Duration
}

func (x *txPrueba) BeginTx(ctx context.Context, opciones pgx.TxOptions) (pgx.Tx, error) {
	x.pasos = append(x.pasos, "begin")
	if ctx.Err() != nil || opciones != (pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite}) {
		x.t.Fatal("incorrect transaction mode")
	}
	if x.fallar == "begin" {
		return nil, errors.New("private begin")
	}
	return x, nil
}

func (x *txPrueba) Exec(_ context.Context, consulta string, _ ...any) (pgconn.CommandTag, error) {
	if consulta != "SET LOCAL timezone = 'UTC'" {
		x.t.Fatal("unexpected setup")
	}
	x.pasos = append(x.pasos, "utc")
	if x.fallar == "utc" {
		return pgconn.CommandTag{}, errors.New("private setup")
	}
	return pgconn.NewCommandTag("SET"), nil
}

func (x *txPrueba) QueryRow(_ context.Context, consulta string, argumentos ...any) pgx.Row {
	if consulta == "SELECT 1" {
		x.pasos = append(x.pasos, "fuente")
		return filaFunc(func(destinos ...any) error { *destinos[0].(*int) = 1; return nil })
	}
	x.pasos = append(x.pasos, "consumo")
	x.consulta, x.argumentos = consulta, len(argumentos)
	return filaFunc(func(destinos ...any) error {
		if x.fallar == "consulta" {
			return errors.New("private query")
		}
		*destinos[0].(*[]byte) = append([]byte(nil), x.bruto...)
		if x.fallar == "cancelacion" {
			x.cancelar()
		}
		return nil
	})
}

func (x *txPrueba) Commit(ctx context.Context) error {
	x.pasos = append(x.pasos, "commit")
	if ctx.Err() != nil {
		x.t.Fatal("commit after cancellation")
	}
	if x.fallar == "commit" {
		x.efectoCommitFallido = true // Unknown outcome may include a committed effect.
		return errors.New("private commit with uncertain outcome")
	}
	x.commitConfirmado = true
	if x.cancelar != nil && x.fallar == "cancelacion_tardia" {
		x.cancelar()
	}
	return nil
}

func (x *txPrueba) Rollback(ctx context.Context) error {
	x.pasos = append(x.pasos, "rollback")
	deadline, presente := ctx.Deadline()
	if ctx.Err() != nil || !presente || time.Until(deadline) <= 0 || time.Until(deadline) > x.plazoCierre {
		x.t.Fatal("cleanup inherited cancellation or lacks bound")
	}
	if x.fallar == "rollback" {
		return errors.New("private rollback")
	}
	if x.efectoCommitFallido {
		return pgx.ErrTxClosed
	}
	return nil
}

func escenarioTX(t *testing.T, revisar bool) (*Registro, *txPrueba, dominio.Orden, pruebas.ConcesionV3Prueba) {
	t.Helper()
	// Reuse the archived synthetic order and common sealed test factory only.
	bruto, err := os.ReadFile("../../../../../cmd/vec-copias-orden/testdata/orden.sintetica.json")
	if err != nil {
		t.Fatal(err)
	}
	var ejemplo struct {
		Orden dominio.Datos `json:"orden"`
	}
	if json.Unmarshal(bruto, &ejemplo) != nil {
		t.Fatal("fixture")
	}
	d := ejemplo.Orden
	d.ProponentePersona, d.AprobadorPersona = "per_0123456789abcdefghijkl", "per_1123456789abcdefghijkl"
	orden, err := dominio.Nueva(d)
	if err != nil {
		t.Fatal(err)
	}
	persona, accion, finalidad, version := d.ProponentePersona, accionProponer, finalidadProponer, uint64(1)
	if revisar {
		persona, accion, finalidad, version = d.AprobadorPersona, accionRevisar, finalidadRevisar, 2
	}
	concesion, err := pruebas.NuevaConcesionV3Prueba(pruebas.DatosConcesionV3Prueba{Instante: time.Now().UTC().Truncate(time.Microsecond), PersonaRef: persona, PerfilRef: "prf_0123456789abcdefghijkl", Accion: accion, Finalidad: finalidad, DecisionRef: "decision:control:prueba", Campos: []string{"orden", "recibo"}, Recurso: vecdomain.RecursoAutorizable{Referencia: d.Orden, ModuloID: modulo, Tipo: tipoRecurso, Ambitos: map[string]string{"destino": d.Destino}}})
	if err != nil {
		t.Fatal(err)
	}
	planSHA, _ := orden.PlanSHA256()
	respuesta := Resultado{Orden: d.Orden, PlanSHA256: planSHA, PersonaRef: persona, Version: version, Recibo: "recibo:control:prueba", RegistradaEn: time.Now().UTC().Truncate(time.Microsecond)}
	x := &txPrueba{t: t, plazoCierre: time.Second}
	x.bruto, _ = json.Marshal(respuesta)
	material := materializadorFunc(func(ctx context.Context, canal CanalSQL, _ dominio.Orden, _ vecdomain.SolicitudAutorizacionLigadaV3, decision vecdomain.DecisionAutorizacionLigadaV3, _ vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3) (ordenpg.MaterialV3, error) {
		x.pasos = append(x.pasos, "materializar")
		for _, metodo := range []string{"Begin", "BeginTx", "Commit", "Rollback", "Conn"} {
			if _, disponible := reflect.TypeOf(canal).MethodByName(metodo); disponible {
				t.Fatal("transaction owner leaked to provider")
			}
		}
		if x.fallar == "material" {
			return ordenpg.MaterialV3{}, errors.New("private material")
		}
		var uno int
		if canal.QueryRow(ctx, "SELECT 1").Scan(&uno) != nil || uno != 1 {
			t.Fatal("provider did not use shared transaction")
		}
		canon, err := vecdomain.RepresentacionCanonicaDecisionAutorizacionV3(decision)
		if err != nil {
			t.Fatal(err)
		}
		m := ordenpg.MaterialV3{Capacidad: []byte("fixture"), Decision: canon, Motivo: []byte("fixture"), Contexto: []byte("fixture"), Payload: []byte("fixture"), Sobre: []byte("fixture"), Evidencia: []byte("fixture"), Raiz: []byte("fixture"), PersonaVersion: 1, PerfilVersion: 1}
		if x.fallar == "material_invalido" {
			m.PersonaVersion = 0
		}
		if x.fallar == "decision_distinta" {
			m.Decision = []byte("otra decision sintetica")
		}
		return m, nil
	})
	return &Registro{pool: x, material: material, plazoCierre: x.plazoCierre}, x, orden, concesion
}
