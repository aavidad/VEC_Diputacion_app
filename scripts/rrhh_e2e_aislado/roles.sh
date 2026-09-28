#!/usr/bin/env bash
# Puerta de LOGINs para el PG18 aislado. Se carga con `source`.
# Ningún DSN de aplicación puede usar postgres ni una cuenta con más de una
# membresía nominal. La precondición de gobierno siguiente impide aprovisionar
# parcialmente el resto de las cuentas y deja la base intacta.

rrhh_e2e_exportar_dsns() {
  [[ -n ${VEC_E2E_PG_CONTAINER:-} && -n ${VEC_E2E_PG_ADMIN_DSN:-} &&
     ${VEC_E2E_PG_PORT:-} =~ ^[0-9]{1,5}$ ]] || {
    printf '%s\n' 'RRHH E2E roles: falta PG18 aislado y DSN administrativo efímero' >&2
    return 1
  }

  # La raíz abre VEC_CT_GOBIERNO_DATABASE_URL con un solo LOGIN y usa ese
  # mismo pool para SET LOCAL ROLE sobre cuatro propietarios/proyectores:
  #   contratacion_temporal_postgresql_gobierno_desarrollo.go:614
  #   contratacion_temporal_autoridad_postgresql_desarrollo.go:131,413
  #   contratacion_temporal_autoridad_motivos_postgresql_desarrollo.go:116
  # El grupo de arranque vec_autorizacion_atestada_v3_migrador sólo tiene
  # SET sobre vec_autorizacion_atestada_v3_propietario. No hay una membresía
  # nominal única existente que cubra también las otras tres autoridades.
  # Dar al LOGIN varios grupos o encadenar propietarios aquí cambiaría ACL y
  # rompería la separación solicitada; este fixture carece de esa autoridad.
  printf '%s\n' \
    'RRHH E2E roles: VEC_CT_GOBIERNO_DATABASE_URL requiere SET LOCAL ROLE para vec_autorizacion_atestada_v3_propietario, vec_contexto_actor_v1_propietario, vec_autorizacion_propietario y vec_autorizacion_motivos_proyector; no existe grupo nominal único acreditado para una sola membresía por LOGIN. No se han creado cuentas ni exportado DSN.' >&2
  return 1
}
