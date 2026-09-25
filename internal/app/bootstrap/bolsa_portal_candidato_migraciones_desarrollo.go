package bootstrap

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrPortalCandidatoMigracionesNoDisponibles detiene el arranque cuando
// VEC_BOLSA_PORTAL_CANDIDATO_ENABLED está encendido y falta alguna migración
// de la que dependen las acciones propias del candidato. Los errores
// concretos la envuelven y nombran la migración ausente.
var ErrPortalCandidatoMigracionesNoDisponibles = errors.New("bootstrap: portal del candidato de Bolsa sin sus migraciones")

var (
	ErrPortalCandidatoFaltaAD384       = fmt.Errorf("%w: falta AD3-84 (consumidores de pausa, reactivación, respuesta y disposición)", ErrPortalCandidatoMigracionesNoDisponibles)
	ErrPortalCandidatoFaltaAD386       = fmt.Errorf("%w: falta AD3-86 (consumidor de la confirmación del contacto propio)", ErrPortalCandidatoMigracionesNoDisponibles)
	ErrPortalCandidatoFaltaBolsa29     = fmt.Errorf("%w: falta Bolsa 000029 (disposición a ofertas publicadas)", ErrPortalCandidatoMigracionesNoDisponibles)
	ErrPortalCandidatoFaltaBolsa30     = fmt.Errorf("%w: falta Bolsa 000030 (solicitudes y respuestas del portal)", ErrPortalCandidatoMigracionesNoDisponibles)
	ErrPortalCandidatoFaltaBolsa40     = fmt.Errorf("%w: falta Bolsa 000040 (confirmación del contacto propio)", ErrPortalCandidatoMigracionesNoDisponibles)
	errPortalCandidatoComprobacionRota = fmt.Errorf("%w: no se pudo comprobar el catálogo", ErrPortalCandidatoMigracionesNoDisponibles)
)

// consultaMigracionesPortalCandidato se ejecuta con el LOGIN ejecutor de
// Bolsa: exige que las fachadas públicas de Bolsa 000029, 000030 y 000040
// existan y que pueda ejecutarlas; de AD3-84 y AD3-86 comprueba en el
// catálogo (legible por todos) sus fachadas de consumo, que ese LOGIN no
// ejecuta directamente. Una función ausente cuenta como no instalada.
const consultaMigracionesPortalCandidato = `SELECT
 (SELECT count(*)=1 FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
   WHERE n.nspname='vec_autorizacion_atestada_v3' AND p.proname='registrar_y_consumir_portal_candidato_bolsa_v3_atestada'),
 (SELECT count(*)=1 FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
   WHERE n.nspname='vec_autorizacion_atestada_v3' AND p.proname='registrar_y_consumir_contacto_propio_bolsa_v3_atestada'),
 (SELECT coalesce(bool_and(pg_catalog.to_regprocedure(f) IS NOT NULL AND pg_catalog.has_function_privilege(pg_catalog.to_regprocedure(f),'EXECUTE')),false) FROM pg_catalog.unnest(ARRAY[
  'vec_bolsa_llamamientos.manifestar_disposicion_oferta_v1(text,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_bolsa_llamamientos.listar_ofertas_candidato_v1(text,timestamptz)']) f),
 (SELECT coalesce(bool_and(pg_catalog.to_regprocedure(f) IS NOT NULL AND pg_catalog.has_function_privilege(pg_catalog.to_regprocedure(f),'EXECUTE')),false) FROM pg_catalog.unnest(ARRAY[
  'vec_bolsa_llamamientos.solicitar_portal_candidato_v1(text,text,text,text,text,timestamptz,timestamptz,text[],text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_bolsa_llamamientos.responder_llamamiento_portal_v1(text,text,text,text,text,text,text,text,text,timestamptz,timestamptz,text[],text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_bolsa_llamamientos.leer_portal_candidato_v1(text,timestamptz,text[])']) f),
 (SELECT coalesce(bool_and(pg_catalog.to_regprocedure(f) IS NOT NULL AND pg_catalog.has_function_privilege(pg_catalog.to_regprocedure(f),'EXECUTE')),false) FROM pg_catalog.unnest(ARRAY[
  'vec_bolsa_llamamientos.confirmar_contacto_propio_v1(text,text,bigint,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_bolsa_llamamientos.leer_contacto_candidato_v1(text,timestamptz)']) f)`

type estadoMigracionesPortalCandidato struct {
	ad384, ad386, bolsa29, bolsa30, bolsa40 bool
}

// diagnostico devuelve el error de la primera migración ausente en el orden
// de instalación: AD3-84, Bolsa 000029, Bolsa 000030, AD3-86, Bolsa 000040.
func (e estadoMigracionesPortalCandidato) diagnostico() error {
	switch {
	case !e.ad384:
		return ErrPortalCandidatoFaltaAD384
	case !e.bolsa29:
		return ErrPortalCandidatoFaltaBolsa29
	case !e.bolsa30:
		return ErrPortalCandidatoFaltaBolsa30
	case !e.ad386:
		return ErrPortalCandidatoFaltaAD386
	case !e.bolsa40:
		return ErrPortalCandidatoFaltaBolsa40
	}
	return nil
}

// comprobarMigracionesPortalCandidatoDesarrollo se llama al arrancar, solo
// con el portal del candidato encendido, antes de publicar sus claves.
func comprobarMigracionesPortalCandidatoDesarrollo(ctx context.Context, bolsa *pgxpool.Pool) error {
	if bolsa == nil {
		return errPortalCandidatoComprobacionRota
	}
	var e estadoMigracionesPortalCandidato
	if err := bolsa.QueryRow(ctx, consultaMigracionesPortalCandidato).Scan(&e.ad384, &e.ad386, &e.bolsa29, &e.bolsa30, &e.bolsa40); err != nil {
		return errPortalCandidatoComprobacionRota
	}
	return e.diagnostico()
}
