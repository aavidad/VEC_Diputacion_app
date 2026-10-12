# Arranque de los selectores de firmas y reincorporación CT

El parche identifica el selector, la etapa y el requisito que detienen la composición. Conserva la causa original para `errors.Is/As` y evita imprimir su mensaje libre. La línea estructurada `contratacion temporal: selector no compuesto` precede al fatal; este conserva un componente preciso y el mensaje catalogado `bootstrap: fuente configurada no disponible`.

R5 valida primero la configuración y el preflight del registrador dedicado de intentos. Así, un registrador ausente o no autorizado detiene estas rutas antes de publicar sus motivos y su perfil. Reincorporación consulta los seis objetos requeridos en una sola lectura y señala el primero que falta; distingue la ausencia de objeto de un fallo de la consulta o del permiso sobre CT153.

No se ha confirmado aún cuál es la dependencia que falla en el proceso de cidonia: faltan las líneas previas al fatal. La copia fría del 13/10 contiene las funciones de CT130, CT134, CT153 y AD98; también el preflight y el registro AD169. La función larga de AD98 figura truncada a 63 bytes en el dump, como corresponde a PostgreSQL. Esto no justifica reinstalar esas migraciones.

La consulta de solo lectura sobre el clon PostgreSQL 18 de dirección confirma ambas asignaciones activas, con el acto fijo esperado, un ámbito y dos/tres concesiones, vigentes de 2026 a 2036. CT153 tiene `EXECUTE` para el grupo ejecutor. Existe una configuración interna del registrador. Estos resultados no cotejan el archivo privado ni el LOGIN y proceso que usa el servicio que falló.

## Configuración común

Conservar las conexiones privadas existentes, sus LOGIN separados y la identidad nominal RRHH. Las variables siguientes son las llaves del perfil sintético; no conceden permisos de usuario por sí solas:

```dotenv
VEC_EXECUTION_PROFILE=desarrollo
VEC_AUTH_MODE=desarrollo
VEC_DEVELOPMENT_GUARD=ACEPTO_CREDENCIALES_NO_AUTORITATIVAS_SOLO_DESARROLLO
VEC_CT_FIRMAS_R5_V2_ENABLED=true
VEC_CT_SEGUIMIENTO_CESE_ENABLED=true
VEC_CT_REINCORPORACION_TITULAR_ENABLED=true
VEC_BOLSA_BORRADORES_ENABLED=true
```

Debe existir el material privado indicado por `VEC_DEVELOPMENT_MATERIAL_DIR`. Conservar las conexiones de `VEC_CT_DATABASE_URL`, `VEC_CT_GOBIERNO_DATABASE_URL` y `VEC_CT_REGISTRO_AUTORIZACION_DATABASE_URL`, y las de consulta, registro/revalidación de identidad y contexto actor ya utilizadas por RRHH. No copiar DSN a este documento ni al log.

Para el seguimiento, declarar los tres catálogos en las variables existentes:

- `VEC_CT_REGLAS_SOURCE_PATH`.
- `VEC_CT_CAUSAS_CESE_SOURCE_PATH`.
- `VEC_CT_ANALISIS_RECTIFICACION_MOTIVOS_SOURCE_PATH`.

Las rutas se mantienen cerradas sin una asignación publicada exacta y vigente. Se conserva la provisión administrativa existente. Si el aviso de arranque muestra `pendiente_provision`, dirección debe revisar la asignación de ese entorno antes de aprobar su huella mediante `VEC_CT_PROVISION_PERFILES_RRHH_APROBACION` y `VEC_CT_PROVISION_PERFILES_RRHH_PREIMAGENES`. Nunca se aprueba automáticamente una asignación revocada o un ámbito de otro circuito. Retirar ambas variables después de la provisión CAS.

## Registrador de intentos de R5

Crear fuera de Git `auditoria-intentos.json` directamente bajo el directorio de material privado, con la protección y propietario que exige el lector seguro. Usar el proceso aprobado para ese entorno; `vec-rrhh` es solo un valor sintético de ejemplo:

```json
{
  "esquema": "vec.auditoria.intentos.servidor.v1",
  "dsn_file": "auditoria-intentos.dsn",
  "proceso": "vec-rrhh",
  "canal": "interna_corporativa",
  "limite_segundos": 10
}
```

`auditoria-intentos.dsn` contiene una única conexión privada con TLS válido y sin destinos alternativos. Debe apuntar a la misma base que CT. Su LOGIN es distinto de ejecución, gobierno y registro de autorización y tiene exclusivamente una membresía en `vec_autorizacion_atestada_v3_registrador_intentos`, con `ADMIN FALSE`, `INHERIT TRUE` y `SET FALSE`. El LOGIN necesita `LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS`, sin `rolconfig` ni concesiones de negocio. También requiere `TEMPORARY` en la base para el preflight común de pools.

La provisión DBA debe conservar una fila inmutable en `vec_autorizacion_atestada_v3.configuracion_runtime_intentos` que ligue ese LOGIN al mismo `proceso` y al canal `interna_corporativa`. Una fila existente se coteja; no se actualiza ni se borra para hacer pasar el arranque. El código no crea el LOGIN ni esta fila. `roles_intentos_up.sql`, CA26, IS13 y AD169 son prerrequisitos ya existentes; solo se instalan los que falten en un entorno nuevo.

Comprobación de solo lectura conectada con el LOGIN dedicado, sin `SET ROLE`:

```sql
BEGIN READ ONLY;
SELECT session_user = current_user,
       vec_autorizacion_atestada_v3.preflight_registrador_intentos_v1(
         'vec-rrhh', 'interna_corporativa');
ROLLBACK;
```

Ambos resultados deben ser `true`. La configuración del proceso en el ejemplo debe sustituirse por el valor aprobado antes de usar la consulta.

## Reincorporación

El ejecutor necesita CT130, CT134, CT153 y los consumidores AD92/AD98 con sus dependencias. CT153 debe devolver `organizacion_ref`; su `EXECUTE` está concedido a `vec_contratacion_temporal_ejecutor`. La consulta de presencia no acredita ese permiso: el arranque ejecuta además la fachada CT153 con el LOGIN real.

Se reutiliza el perfil fijo `reincorporacion_titular`, su contexto nominal registrado y su sesión propia derivada de RRHH. La provisión existente comprueba el acto de asignación, la versión de rol, las concesiones y la preimagen; el selector no aporta una identidad ni un permiso.

## Alcance y comprobación pendiente

Este corte modifica el diagnóstico y el orden del preflight de R5, sin añadir SQL ni nuevas autoridades. `VEC_CT_FIRMAS_R5_V2_ENABLED` compone consulta y recuperación nominales; encenderlo no acredita por sí solo las dos vías de registro, la firma de dos personas ni un PDF firmado.

Formato y `git diff --check` comprobados. Se entrega una prueba focal que exige conservar la causa sin imprimir un DSN o una identidad sintéticos. Dirección ejecutará las pruebas, el arranque sobre su clon PostgreSQL 18, Chrome y CI. No se han ejecutado compilación, pruebas Go ni accesos a cidonia en este encargo.
