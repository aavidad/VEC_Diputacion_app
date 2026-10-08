# Preparar el catálogo base de reglas CT190

Este comando lee el paquete JSON indicado por el operador y genera dos ficheros: el catálogo canónico y un manifiesto que liga su huella con la del fichero fuente. No consulta ni modifica PostgreSQL. No se ejecuta al arrancar VEC.

```bash
go run ./cmd/vec-preparar-catalogo-reglas-ct \
  -fuente "$FUENTE_CATALOGO_CT" \
  -salida "$CATALOGO_CANONICO_CT" \
  -manifiesto "$MANIFIESTO_CATALOGO_CT" \
  -aprobacion-ref "$APROBACION_REF_CT"
```

Las tres rutas deben ser absolutas. Cree antes un directorio privado fuera de Git y asigne nombres nuevos a las salidas; el comando no sobrescribe ficheros ni escribe en `/tmp`. Las salidas tienen permisos `0600`. El catálogo canónico contiene exactamente los bytes de `json.Marshal(catalogo.ClonarCanonico())`, sin salto de línea final. La huella SHA256 del manifiesto corresponde a esos bytes; la huella de fuente corresponde a los bytes originales, incluidos espacios y saltos de línea. El manifiesto también conserva identificador, versión, referencia y revisión de la fuente.

El paquete debe tener el sobre `version_esquema`, `fuente` y `catalogo` del catálogo configurable vigente. La fuente de instalación debe ser un catálogo publicado, con `fuente.demostracion=false` y una referencia distinta de la marca de ejemplo. El comando valida cada entrada con el lector tipado de reglas, incluidas las entradas futuras. La referencia de aprobación suministrada debe coincidir exactamente con `catalogo.aprobacion_ref`. Los valores y plazos proceden del fichero proporcionado. El cotejo de la referencia no acredita por sí solo la aprobación: esta corresponde al circuito administrativo.

Para un ensayo sintético aislado se puede añadir `-permitir-ejemplo`. El manifiesto saldrá con `ejemplo=true`. El publicador CT190 solo acepta ese caso en desarrollo con autorización explícita; no sirve como catálogo real de la principal.

Tras instalar CT190 y CT191, compruebe la última secuencia de activación. Mientras no haya base activa, las altas y los cambios de fase continúan como legado, con el cálculo de plazos anterior. El publicador recibe los tres ficheros y esa secuencia. La conexión se configura con las variables habituales de libpq y una cuenta técnica que pueda asumir `vec_contratacion_temporal_propietario`.

```bash
VEC_CT190_ENTORNO=produccion \
VEC_CT190_APROBACION_REF="$APROBACION_REF_CT" \
deploy/principal/publicar_catalogo_ct190.sh \
  "$FUENTE_CATALOGO_CT" "$CATALOGO_CANONICO_CT" \
  "$MANIFIESTO_CATALOGO_CT" "$SECUENCIA_ESPERADA"
```

Para ensayar el paquete de ejemplo en una base sintética, use `VEC_CT190_ENTORNO=desarrollo VEC_CT190_PERMITIR_EJEMPLO=1` y prepare antes los artefactos con `-permitir-ejemplo`. Esa vía no representa una aprobación de RRHH. La referencia de aprobación de una publicación real debe coincidir con la del catálogo y proceder del circuito administrativo.

El script comprueba las huellas y publica la versión y su activación en una transacción serializable. Si la secuencia cambió, falla sin conservar una publicación parcial. Conserve los tres ficheros hasta cotejar la fila publicada, la activación y sus huellas. Si cambia la fuente, prepare nuevos artefactos; una versión publicada es inmutable. La cuenta técnica que publica queda en `publicada_por` y `activada_por`; esos campos no atribuyen una decisión a RRHH.

## Habilitar la edición en «Reglas vigentes»

Instale una vez CT190 y CT191, en el orden de `deploy/principal/lista_sql_codexy_ct_plazos_minimo_20261008.txt`, antes de arrancar el nuevo binario. La primera publicación no necesita detener las altas: los tramos que se abran antes quedan como legado sin captura. La edición permanece deshabilitada hasta publicar una base activa y aprobar la provisión. Prepare la base a partir del mismo fichero que usa `VEC_CT_REGLAS_SOURCE_PATH`; la primera activación, con secuencia esperada `0`, conserva esa base para los tramos anteriores. Exige que todavía no haya ajustes CT148. Esa referencia de transición no atribuye reglas históricas a esos tramos. Los tramos abiertos con una base activa conservan su propia captura. Si después se desactiva la base, los nuevos tramos quedan como `legado_sin_instantanea` y usan el catálogo actual, preparado una sola vez por consulta. La referencia de transición sólo se aplica a los tramos cuya fecha de entrada es anterior o igual a la primera activación; no se atribuye a los abiertos después de desactivarla.

Copie `data/catalogos/contratacion_temporal/motivos_ajuste_v1.json` al directorio de configuración que ya se monta en cidonia, como `/vec-incorporacion/motivos_ajuste_ct_v1.json`. Configure:

```text
VEC_CT_REGLAS_AJUSTES_ENABLED=true
VEC_CT_REGLAS_AJUSTES_MOTIVOS_PATH=/vec-incorporacion/motivos_ajuste_ct_v1.json
```

Conserve la fuente CT ya configurada. El arranque comprueba que su identificador, versión, huella y aprobación coinciden con la base activa de CT191; una diferencia impide habilitar la edición.

La edición reutiliza el perfil fijo `entrega-peticion-rrhh-lector`. Administración debe aprobar la ampliación de su asignación vigente mediante las variables existentes `VEC_CT_PROVISION_PERFILES_RRHH_APROBACION` y `VEC_CT_PROVISION_PERFILES_RRHH_PREIMAGENES`. La primera contiene la referencia del acto de aprobación; la segunda, la huella SHA256 exacta de la asignación actual, o varias huellas separadas por comas. No use una huella de otro entorno. El aviso de provisión pendiente del arranque indica el perfil y su preimagen; también se puede obtener con la consulta de solo lectura siguiente, usando la conexión privada de gobierno:

```sql
BEGIN READ ONLY;
SET LOCAL ROLE vec_autorizacion_propietario;
SELECT v.perfil_activo_ref, a.huella_sha256
FROM vec_autorizacion.asignacion_perfil_actual v
JOIN vec_autorizacion.asignacion_perfil a
  ON a.perfil_activo_ref = v.perfil_activo_ref
 AND a.asignacion_ref = v.asignacion_ref
JOIN vec_autorizacion.version_rol r
  ON r.version_rol_ref = a.version_rol_ref
WHERE v.acto_ref = 'acto:ct:perfil-fijo:asignacion:v1'
  AND r.documento->>'rol_id' = 'entrega-peticion-rrhh-lector';
ROLLBACK;
```

Después de la provisión aprobada, retire ambas variables de aprobación. El arranque y las peticiones consumen la asignación publicada; nunca conceden acceso por el contenido del POST. Sin activar la edición, se conservan la lectura de reglas y los permisos anteriores.
