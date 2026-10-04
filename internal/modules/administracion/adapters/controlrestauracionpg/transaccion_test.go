package controlrestauracionpg

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestControlDevuelveReciboSoloTrasCommitYMaterializaEnLaMismaTX(t *testing.T) {
	for _, revisar := range []bool{false, true} {
		r, tx, orden, c := escenarioTX(t, revisar)
		var resultado Resultado
		var err error
		if revisar {
			resultado, err = r.RevisarCAS(context.Background(), orden, 1, c.Solicitud, c.Decision, c.Confirmacion)
		} else {
			resultado, err = r.Proponer(context.Background(), orden, c.Solicitud, c.Decision, c.Confirmacion)
		}
		if err != nil || resultado.Recibo == "" || !tx.commitConfirmado || !reflect.DeepEqual(tx.pasos, []string{"begin", "utc", "materializar", "fuente", "consumo", "commit"}) {
			t.Fatal("receipt without confirmed commit", err, tx.pasos)
		}
		wantQuery, wantArgs := registrarPropuestaSQL, 11
		if revisar {
			wantQuery, wantArgs = registrarRevisionSQL, 12
		}
		if tx.consulta != wantQuery || tx.argumentos != wantArgs {
			t.Fatal("SQL facade changed")
		}
	}
}

func TestFalloDeMaterialConsultaOReciboInconsistenteNuncaConfirma(t *testing.T) {
	for _, caso := range []string{"begin", "utc", "material", "material_invalido", "decision_distinta", "consulta", "recibo_orden", "recibo_huella", "recibo_persona", "recibo_version", "recibo_json"} {
		t.Run(caso, func(t *testing.T) {
			r, tx, orden, c := escenarioTX(t, false)
			tx.fallar = caso
			var respuesta Resultado
			if json.Unmarshal(tx.bruto, &respuesta) != nil {
				t.Fatal("fixture")
			}
			switch caso {
			case "recibo_orden":
				respuesta.Orden = "orden:distinta"
			case "recibo_huella":
				respuesta.PlanSHA256 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
			case "recibo_persona":
				respuesta.PersonaRef = "per_1123456789abcdefghijkl"
			case "recibo_version":
				respuesta.Version = 2
			}
			tx.bruto, _ = json.Marshal(respuesta)
			if caso == "recibo_json" {
				tx.bruto = []byte("invalid SQL response")
			}
			result, err := r.Proponer(context.Background(), orden, c.Solicitud, c.Decision, c.Confirmacion)
			if err != ErrRegistro || result != (Resultado{}) || tx.commitConfirmado {
				t.Fatal("false receipt/commit or private error", err)
			}
			wantRollback := caso != "begin"
			if (tx.pasos[len(tx.pasos)-1] == "rollback") != wantRollback {
				t.Fatal("missing cleanup", tx.pasos)
			}
			for _, paso := range tx.pasos {
				if paso == "commit" {
					t.Fatal("validated after commit")
				}
			}
		})
	}
}

func TestCommitFallidoNoDevuelveReciboNiPrometeRollback(t *testing.T) {
	r, tx, orden, c := escenarioTX(t, false)
	tx.fallar = "commit"
	result, err := r.Proponer(context.Background(), orden, c.Solicitud, c.Decision, c.Confirmacion)
	if err != ErrRegistro || result != (Resultado{}) || !tx.efectoCommitFallido || tx.commitConfirmado || !reflect.DeepEqual(tx.pasos, []string{"begin", "utc", "materializar", "fuente", "consumo", "commit", "rollback"}) {
		t.Fatal("uncertain commit turned into success or retry", err, tx.pasos)
	}
}

func TestCancelacionLimpiaConContextoIndependienteYCommitConfirmadoPermanece(t *testing.T) {
	for _, tarde := range []bool{false, true} {
		r, tx, orden, c := escenarioTX(t, false)
		ctx, cancelar := context.WithCancel(context.Background())
		defer cancelar()
		tx.cancelar = cancelar
		tx.fallar = "cancelacion"
		if tarde {
			tx.fallar = "cancelacion_tardia"
		}
		result, err := r.Proponer(ctx, orden, c.Solicitud, c.Decision, c.Confirmacion)
		if tarde {
			if err != nil || result.Recibo == "" || !tx.commitConfirmado || ctx.Err() == nil {
				t.Fatal("confirmed commit lost by late cancellation", err)
			}
		} else if err != ErrRegistro || result != (Resultado{}) || tx.pasos[len(tx.pasos)-1] != "rollback" || tx.commitConfirmado {
			t.Fatal("cancelled transaction not cleaned", err, tx.pasos)
		}
	}
}

func TestFalloDeCierreEsNominalYNuncaDevuelveRecibo(t *testing.T) {
	r, tx, orden, c := escenarioTX(t, false)
	tx.fallar, tx.bruto = "rollback", []byte("bad response")
	result, err := r.Proponer(context.Background(), orden, c.Solicitud, c.Decision, c.Confirmacion)
	if err != ErrCierre || result != (Resultado{}) {
		t.Fatal("private cleanup error or false receipt", err)
	}
}

func TestConstructorExigeProveedorNuevoYPlazoExplicito(t *testing.T) {
	pool := new(pgxpool.Pool) // Never used: no connection or network in this test.
	var typedNil materializadorFunc
	for _, plazo := range []time.Duration{0, -time.Second, 31 * time.Second} {
		if _, err := Nuevo(pool, &materializadorPrueba{}, plazo); err != ErrRegistro {
			t.Fatal("invalid deadline admitted", err)
		}
	}
	if _, err := Nuevo(pool, typedNil, time.Second); err != ErrRegistro {
		t.Fatal("typed nil provider", err)
	}
	if _, err := Nuevo(nil, &materializadorPrueba{}, time.Second); err != ErrRegistro {
		t.Fatal("nil pool", err)
	}
	if _, err := Nuevo(pool, &materializadorPrueba{}, time.Second); err != nil {
		t.Fatal(err)
	}
}
