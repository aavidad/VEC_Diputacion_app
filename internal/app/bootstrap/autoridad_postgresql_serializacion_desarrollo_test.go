package bootstrap

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// Obliga a los dos publicadores a tomar su snapshot serializable antes de
// soltar el bloqueo del perfil. Uno publica; el otro recibe 40001 al leer y
// recupera únicamente la publicación idéntica desde la misma preimagen.
func TestPublicarInstantaneaConPreimagenCarreraPostgreSQL18(t *testing.T) {
	ctx, gobierno, admin := poolesPerfilDinamicoRRHHPostgreSQLPrueba(t)
	soporte := soporteRRHHPostgreSQLPrueba(t, ctx, gobierno)
	if _, err := asegurarPerfilDinamicoCTDesarrollo(ctx, gobierno, soporte); err != nil {
		t.Fatal(err)
	}
	autoridad := (&autoridadPostgreSQLContratacionTemporalDesarrollo{
		pool: gobierno, soporte: soporte,
	}).autoridadComun()
	perfil := soporte.instantanea.AsignacionPerfil.PerfilActivoRef
	publicada, encontrada, err := leerInstantaneaPublicadaPostgreSQLDesarrollo(ctx, admin, perfil)
	if err != nil || !encontrada {
		t.Fatalf("preimagen publicada: %v", err)
	}
	preimagen := publicada.instantanea
	candidata, err := autoridad.prepararInstantanea(ctx,
		rolDeExpedientePrueba(soporte, "expediente:rrhh:serializacion"), false)
	if err != nil || candidata.AsignacionPerfil.Version != preimagen.AsignacionPerfil.Version+1 {
		t.Fatalf("candidata: %v", err)
	}

	bloqueador, err := gobierno.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer bloqueador.Rollback(context.Background())
	if _, err := bloqueador.Exec(ctx,
		`SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended($1,0))`,
		autoridad.prefijoBloqueo+perfil); err != nil {
		t.Fatal(err)
	}

	var grupo sync.WaitGroup
	resultados := make(chan error, 2)
	for range 2 {
		grupo.Go(func() {
			resultados <- autoridad.publicarInstantaneaDesdePreimagen(ctx, candidata, preimagen)
		})
	}
	limite := time.NewTimer(10 * time.Second)
	defer limite.Stop()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	esperando := false
	for !esperando {
		var numero int
		if err := admin.QueryRow(ctx, `SELECT count(*) FROM pg_catalog.pg_locks
			WHERE locktype='advisory' AND NOT granted`).Scan(&numero); err != nil {
			t.Fatal(err)
		}
		esperando = numero >= 2
		if esperando {
			break
		}
		select {
		case <-ticker.C:
		case <-limite.C:
			t.Fatal("los dos publicadores no llegaron al bloqueo consultivo")
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
	if err := bloqueador.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	grupo.Wait()
	close(resultados)
	for err := range resultados {
		if err != nil {
			t.Fatalf("la publicación idéntica no se recuperó: %v", err)
		}
	}
	historia := historiaPerfilPostgreSQLPrueba(t, ctx, admin, perfil)
	if historia.versiones != 2 || historia.actual != candidata.AsignacionPerfil.Referencia() {
		t.Fatalf("la carrera duplicó o perdió la publicación: %+v", historia)
	}
	if err := autoridad.publicarInstantaneaDesdePreimagen(ctx, candidata, candidata); err != nil {
		t.Fatalf("el replay exacto de la instantánea vigente falló: %v", err)
	}
	if final := historiaPerfilPostgreSQLPrueba(t, ctx, admin, perfil); final != historia {
		t.Fatalf("el replay exacto duplicó historia: antes=%+v después=%+v", historia, final)
	}
	for _, caso := range []struct {
		nombre string
		mutar  func(*dominiovec.InstantaneaAutorizacion)
	}{
		{"origen", func(i *dominiovec.InstantaneaAutorizacion) {
			i.AsignacionPerfil.EmitidaPor = "otro_emisor_prueba"
		}},
		{"ambito", func(i *dominiovec.InstantaneaAutorizacion) {
			i.AsignacionPerfil.Ambitos[0].Valores[0] = "organizacion:ajena"
		}},
		{"rol", func(i *dominiovec.InstantaneaAutorizacion) {
			i.VersionRol.Concesiones[0].Finalidades[0] = "finalidad:ajena"
		}},
		{"control", func(i *dominiovec.InstantaneaAutorizacion) {
			i.ControlVigenciaVersionRol.ActualizadoPor = "otro_control_prueba"
		}},
		{"catalogo", func(i *dominiovec.InstantaneaAutorizacion) {
			i.RevisionCatalogoPoliticas++
		}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			ajena := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(candidata)
			caso.mutar(&ajena)
			if err := ajena.Validar(); err != nil {
				t.Fatalf("preimagen alterada de prueba inválida: %v", err)
			}
			if err := autoridad.publicarInstantaneaDesdePreimagen(ctx, candidata, ajena); !errors.Is(err, errPostgreSQLContratacionTemporalDesarrolloNoDisponible) {
				t.Fatalf("el replay de la misma versión con %s distinto no se rechazó: %v", caso.nombre, err)
			}
		})
	}

	preimagenAjena := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(preimagen)
	preimagenAjena.AsignacionPerfil.EmitidaPor = "otro_emisor_prueba"
	if err := preimagenAjena.Validar(); err != nil {
		t.Fatalf("preimagen ajena de prueba inválida: %v", err)
	}
	if err := autoridad.publicarInstantaneaDesdePreimagen(ctx, candidata, preimagenAjena); !errors.Is(err, errPostgreSQLContratacionTemporalDesarrolloNoDisponible) {
		t.Fatalf("la preimagen distinta no se rechazó: %v", err)
	}
	candidataAjena := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(candidata)
	candidataAjena.AsignacionPerfil.Ambitos = append(candidataAjena.AsignacionPerfil.Ambitos,
		dominiovec.AmbitoPerfil{Clave: "expediente_ajeno_ref", Valores: []string{"expediente:rrhh:ajeno"}})
	if err := candidataAjena.Validar(); err != nil {
		t.Fatalf("candidata ajena de prueba inválida: %v", err)
	}
	if err := autoridad.publicarInstantaneaDesdePreimagen(ctx, candidataAjena, preimagen); !errors.Is(err, errPostgreSQLContratacionTemporalDesarrolloNoDisponible) {
		t.Fatalf("el contenido y ámbito distintos no se rechazaron: %v", err)
	}
	if final := historiaPerfilPostgreSQLPrueba(t, ctx, admin, perfil); final != historia {
		t.Fatalf("un rechazo modificó el historial: antes=%+v después=%+v", historia, final)
	}
}
