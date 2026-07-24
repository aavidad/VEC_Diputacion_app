package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

func TestRevalidadorAutenticacionActorProyectaSoloReferenciasOpacas(t *testing.T) {
	solicitud := solicitudRevalidacionActorValida()
	fila := filaRevalidacionActorValida()
	zona := time.FixedZone("zona-no-utc", 2*60*60)
	fila[15] = fila[15].(time.Time).In(zona).Add(987 * time.Nanosecond)
	fila[16] = fila[16].(time.Time).In(zona).Add(987 * time.Nanosecond)
	fila[17] = fila[17].(time.Time).In(zona).Add(987 * time.Nanosecond)
	fila[18] = fila[18].(time.Time).In(zona).Add(987 * time.Nanosecond)
	tx := &transaccionDoble{filas: [][]any{fila}}
	pool := &iniciadorDoble{transacciones: []*transaccionDoble{tx}}
	revalidador, err := nuevoRevalidadorAutenticacionActorPostgreSQL(pool)
	if err != nil {
		t.Fatal("crear revalidador de prueba")
	}

	resultado, err := revalidador.RevalidarAutenticacionActorV1(
		context.Background(),
		solicitud,
	)
	if err != nil || resultado.Validar() != nil {
		t.Fatal("la proyeccion durable valida fue rechazada")
	}
	if resultado.AutenticacionRef != solicitud.AutenticacionRef ||
		resultado.SesionRef != solicitud.SesionRef || pool.llamadas != 1 ||
		tx.commits != 1 || len(tx.argumentos) != 1 ||
		len(tx.argumentos[0]) != 2 ||
		tx.argumentos[0][0] != solicitud.AutenticacionRef ||
		tx.argumentos[0][1] != solicitud.SesionRef ||
		!strings.Contains(tx.consultas[0], "revalidar_autenticacion_actor_v1") {
		t.Fatal("el adaptador amplio o altero el contrato SQL de dos referencias")
	}
	comprobarSerializable(t, pool.opciones)
	for _, instante := range []time.Time{
		resultado.AutenticacionVerificadaEn,
		resultado.SesionEmitidaEn,
		resultado.SesionValidaHasta,
		resultado.SesionRevalidadaEn,
	} {
		if instante.Location() != time.UTC || instante.Nanosecond()%1_000 != 0 {
			t.Fatal("un timestamptz no se normalizo a UTC y microsegundos")
		}
	}
}

func TestRevalidadorAutenticacionActorFallaAntesDeSQLConEntradasInvalidas(t *testing.T) {
	pool := &iniciadorDoble{}
	revalidador, err := nuevoRevalidadorAutenticacionActorPostgreSQL(pool)
	if err != nil {
		t.Fatal("crear revalidador de prueba")
	}
	casos := []domain.SolicitudRevalidacionAutenticacionActorV1{
		{},
		{AutenticacionRef: "aut_inyectada' OR true--", SesionRef: referencia("ses_", "s")},
		{AutenticacionRef: referencia("aut_", "a"), SesionRef: "ses_demo"},
	}
	for _, caso := range casos {
		_, err = revalidador.RevalidarAutenticacionActorV1(context.Background(), caso)
		if !errors.Is(err, domain.ErrAutenticacionRevalidadaInvalida) {
			t.Fatal("una solicitud no canonica no fallo cerrada")
		}
	}
	if pool.llamadas != 0 {
		t.Fatal("una solicitud no canonica alcanzo PostgreSQL")
	}

	var revalidadorNulo *RevalidadorAutenticacionActorPostgreSQL
	_, err = revalidadorNulo.RevalidarAutenticacionActorV1(
		context.Background(),
		solicitudRevalidacionActorValida(),
	)
	if !errors.Is(err, domain.ErrAutenticacionRevalidadaInvalida) {
		t.Fatal("un receptor tipado nulo fue aceptado")
	}
}

func TestConstructorRevalidadorAutenticacionActorRechazaTypedNil(t *testing.T) {
	var pool *iniciadorDoble
	revalidador, err := nuevoRevalidadorAutenticacionActorPostgreSQL(pool)
	if revalidador != nil ||
		!errors.Is(err, ErrRevalidadorAutenticacionActorNoDisponible) {
		t.Fatal("el constructor acepto un iniciador tipado nulo")
	}
}

func TestRevalidadorAutenticacionActorSaneaFilasAdversariales(t *testing.T) {
	solicitud := solicitudRevalidacionActorValida()
	casos := []struct {
		nombre string
		fila   []any
	}{
		{"revision cero", cambiarCampoRevalidacionActor(5, "0")},
		{"revision fuera de uint64", cambiarCampoRevalidacionActor(5, "18446744073709551616")},
		{"eco de autenticacion distinto", cambiarCampoRevalidacionActor(0, referencia("aut_", "z"))},
		{"eco de sesion distinto", cambiarCampoRevalidacionActor(3, referencia("ses_", "z"))},
		{"metodo demo", cambiarCampoRevalidacionActor(11, string(domain.AuthMethodDemo))},
		{"superficie inventada", cambiarCampoRevalidacionActor(10, "superficie_inventada")},
		{"cuenta confundida", cambiarCampoRevalidacionActor(8, referencia("cta_", "z"))},
		{"cronologia alterada", cambiarCampoRevalidacionActor(18, instanteRevalidacionActor().Add(-time.Minute))},
		{"fila truncada", filaRevalidacionActorValida()[:18]},
		{"fila ampliada", append(filaRevalidacionActorValida(), "campo-no-contratado")},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			tx := &transaccionDoble{filas: [][]any{caso.fila}}
			pool := &iniciadorDoble{transacciones: []*transaccionDoble{tx}}
			revalidador, err := nuevoRevalidadorAutenticacionActorPostgreSQL(pool)
			if err != nil {
				t.Fatal("crear revalidador de prueba")
			}
			resultado, err := revalidador.RevalidarAutenticacionActorV1(
				context.Background(), solicitud,
			)
			if resultado != (domain.AutenticacionRevalidadaV1{}) ||
				!errors.Is(err, domain.ErrAutenticacionRevalidadaInvalida) ||
				strings.Contains(err.Error(), "columna") || tx.commits != 0 {
				t.Fatal("una fila adversarial no fallo cerrada y saneada")
			}
		})
	}
}

func TestRevalidadorAutenticacionActorSaneaFallosTransaccionales(t *testing.T) {
	solicitud := solicitudRevalidacionActorValida()
	casos := []struct {
		nombre string
		pool   *iniciadorDoble
	}{
		{"begin", &iniciadorDoble{}},
		{"preparacion", &iniciadorDoble{transacciones: []*transaccionDoble{{errExec: errors.New("secreto interno exec")}}}},
		{"consulta", &iniciadorDoble{transacciones: []*transaccionDoble{{}}}},
		{"commit", &iniciadorDoble{transacciones: []*transaccionDoble{{filas: [][]any{filaRevalidacionActorValida()}, errCommit: errors.New("secreto interno commit")}}}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			revalidador, err := nuevoRevalidadorAutenticacionActorPostgreSQL(caso.pool)
			if err != nil {
				t.Fatal("crear revalidador de prueba")
			}
			_, err = revalidador.RevalidarAutenticacionActorV1(
				context.Background(), solicitud,
			)
			if !errors.Is(err, domain.ErrAutenticacionRevalidadaInvalida) ||
				strings.Contains(err.Error(), "secreto") ||
				strings.Contains(err.Error(), "detalle") {
				t.Fatal("un error de infraestructura no fue saneado")
			}
		})
	}
}

func TestRevalidadorAutenticacionActorConservaCancelacion(t *testing.T) {
	pool := &iniciadorDoble{}
	revalidador, err := nuevoRevalidadorAutenticacionActorPostgreSQL(pool)
	if err != nil {
		t.Fatal("crear revalidador de prueba")
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	_, err = revalidador.RevalidarAutenticacionActorV1(
		ctx, solicitudRevalidacionActorValida(),
	)
	if !errors.Is(err, context.Canceled) || pool.llamadas != 0 {
		t.Fatal("la cancelacion no se conservo antes de abrir transaccion")
	}
}

func solicitudRevalidacionActorValida() domain.SolicitudRevalidacionAutenticacionActorV1 {
	return domain.SolicitudRevalidacionAutenticacionActorV1{
		AutenticacionRef: referencia("aut_", "a"),
		SesionRef:        referencia("ses_", "s"),
	}
}

func instanteRevalidacionActor() time.Time {
	return time.Date(2026, time.July, 21, 12, 0, 1, 0, time.UTC)
}

func filaRevalidacionActorValida() []any {
	revalidada := instanteRevalidacionActor()
	emitida := revalidada.Add(-time.Second)
	return []any{
		referencia("aut_", "a"), strings.Repeat("a", 64),
		referencia("ase_", "e"), referencia("ses_", "s"),
		referencia("cse_", "c"), "1", strings.Repeat("c", 64),
		referencia("cta_", "t"), referencia("cta_", "t"), false,
		string(domain.SuperficieAutenticacionInternaCorporativaV1),
		string(domain.AuthMethodKerberos), string(domain.AuthAssuranceHigh),
		referencia("pga_", "p"), strings.Repeat("b", 64),
		emitida.Add(-time.Second), emitida, revalidada.Add(4 * time.Minute), revalidada,
	}
}

func cambiarCampoRevalidacionActor(indice int, valor any) []any {
	fila := filaRevalidacionActorValida()
	fila[indice] = valor
	return fila
}
