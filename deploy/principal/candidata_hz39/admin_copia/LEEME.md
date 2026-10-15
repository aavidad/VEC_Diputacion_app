# Administración en la copia de ensayo

El kit privado crea dos administradores ficticios, Irene Robles Vega y Daniel
Ortega Soria, con sus certificados y perfiles. Cada equipo lo aplica en su
propio contenedor PostgreSQL 18, restaurado desde el respaldo de la candidata.
Las claves, conexiones, planes y recibos se guardan fuera de Git.

```sh
export VEC_ENSAYO_ADMIN_KIT="$HOME/.local/state/vec-ensayo-candidata/admin-copia/kit"
```

## Preparar la copia

Antes del alta, conservar una preimagen de Administración y comprobar que el
contenedor, la base, el puerto y el material pertenecen a la copia del equipo.
Aplicar las ACL de base y de tipos del respaldo y mantener un sellador de
auditoría activo. El guion conserva los rechazos y las preimágenes; no cambia
las comprobaciones de identidad o permisos.

Preparar una carpeta nueva mediante el generador del kit:

```sh
python3 "$VEC_ENSAYO_ADMIN_KIT/preparar_copia.py" \
  --instancia "$VEC_ENSAYO_ADMIN_INSTANCIA" \
  --contenedor "$VEC_ENSAYO_PG_CONTENEDOR" \
  --puerto-pg "$VEC_ENSAYO_PG_PUERTO" \
  --backup-admin "$VEC_ENSAYO_ADMIN_PREIMAGEN" \
  --material "$VEC_ENSAYO_MATERIAL" \
  --ca-pg "$VEC_ENSAYO_PG_CA" \
  --catalogo-entrada-unidad "$VEC_ENSAYO_ADMIN_ENTRADA_UNIDAD" \
  --salida "$VEC_ENSAYO_ADMIN_RUN"
```

El generador escribe archivos privados, fija las huellas de los binarios y
conserva las fuentes del kit. El ejecutor las coteja antes de cada etapa.
El alta se hace con los publicadores existentes; preparar
archivos no acredita que se haya publicado nada.

## Dar de alta los dos administradores

El ejecutor usa `config.json` de la carpeta recién preparada:

```sh
python3 "$VEC_ENSAYO_ADMIN_KIT/arranque_principal_h10.py" \
  --config "$VEC_ENSAYO_ADMIN_RUN/config.json" --etapa comprobar
```

Ejecutar las etapas en este orden, conservando cada acuse:

1. `preparar_fuentes`, `configurar_fuentes`, `aplicar_fuentes`.
2. `preparar_unidad`, `configurar_unidad`, `aplicar_unidad`.
3. `preparar_bootstrap`, `configurar_bootstrap`, `aplicar_bootstrap`.
4. `verificar`.

Las etapas `configurar_*` preparan la provisión de los LOGIN y la aprobación
de la huella del plan. Aplicar también el archivo privado de contraseña
correspondiente, por la conexión DBA de la copia, antes de `aplicar_*`.
Si el resultado de COMMIT es incierto, usar `recuperar_fuentes`,
`recuperar_unidad` o `recuperar_bootstrap` con el mismo material. No repetir
la preparación ni inventar otro recibo.

El resultado verificado debe conservar una fuente, una unidad, dos perfiles
de Aplicación y uno de Sistemas. La organización y la unidad salen de los
recibos de esta copia. No usar las referencias de otro equipo.

## Arranque y selección de perfil

El kit monta `vec-admin` en una escucha HTTPS de loopback propia, con TLS de
cliente, lista de revocación, conexiones segregadas y confianza vigente.
La renovación de confianza usa `vec-gobierno-usuarios-admin`:
`preparar`, provisión de la aprobación, `aplicar` y `verificar`.
Su ventana diaria se conserva; una confianza caducada requiere ese acto de
renovación. El procedimiento está en
[`cmd/vec-gobierno-usuarios-admin/README.md`](../../../../cmd/vec-gobierno-usuarios-admin/README.md).

Cada administrador consulta `GET /api/admin/seleccion-perfil/v1/propios` y
selecciona su perfil de Aplicación con
`POST /api/admin/seleccion-perfil/v1/seleccion`, usando la revisión observada.
La consulta posterior debe recuperar el mismo perfil y la revisión siguiente.
Conservar el recibo y su referencia de auditoría.

Las peticiones usan el certificado de esa persona. Los POST llevan el origen
HTTPS configurado y `Sec-Fetch-Site: same-origin`, `Sec-Fetch-Mode: cors`,
`Sec-Fetch-Dest: empty`. La selección no se sustituye por una cabecera de
perfil ni por almacenamiento del navegador.

## Asignar y retirar perfiles

Activar el lote con `VEC_ADMIN_LOTE_CONFIG_FILE`, su LOGIN exclusivo y las
capacidades del conjunto de gobierno correspondiente. Consultar la
preparación de la persona y unidad:

```text
GET /api/admin/perfiles/v1/personas/{persona_ref}/preparacion-lote?unidad_ref=…
POST /api/admin/perfiles/v1/lotes-ordinarios
```

El cuerpo del lote usa las referencias, versiones y huellas del GET.
Para otorgar, el perfil debe estar publicado, habilitado y registrado por
AUT49 para la organización de estos administradores. El lote añade la unidad.
Una lista de altas vacía no acredita una asignación. Un rol registrado para
otra organización necesita otra versión gobernada; la referencia anterior
se conserva. Una asignación retirada no se revive: se otorga una nueva.

El plan técnico de registro de perfiles y su aprobación están descritos en
[`PERFILES_ASIGNABLES_AUT49.md`](../../../postgresql/autorizacion/PERFILES_ASIGNABLES_AUT49.md).
Los perfiles de firma CT con ámbitos auxiliares se cargan como datos ficticios
en la copia, por la decisión de dirección del 14 de octubre.

## Catálogo de acciones y versiones de rol

El catálogo de acciones se admite por la CLI protegida existente. Preparar
el paquete completo, conservar los descriptores anteriores y aprobar la
huella de sus bytes. Configurar su LOGIN exclusivo y la aprobación antes de
aplicar:

```sh
vec-catalogo-acciones-admitir --fase aplicar \
  --config "$VEC_ENSAYO_ACCIONES_CONFIG" --plan "$VEC_ENSAYO_ACCIONES_PLAN" \
  --sha "$VEC_ENSAYO_ACCIONES_SHA" --idioma es
vec-catalogo-acciones-admitir --fase consultar \
  --config "$VEC_ENSAYO_ACCIONES_CONFIG" --ref "$VEC_ENSAYO_ACCIONES_REF" \
  --version "$VEC_ENSAYO_ACCIONES_VERSION" \
  --catalogo-sha "$VEC_ENSAYO_ACCIONES_CATALOGO_SHA" --idioma es
```

La consulta recupera la instantánea publicada. Ante un COMMIT incierto se
consulta primero; cualquier reintento conserva el plan, su huella y la clave
de operación. El formato, el censo obligatorio y la provisión están en
[`vec-catalogo-acciones-admitir`](../../../../cmd/vec-catalogo-acciones-admitir/README.md).

Las propuestas y los cierres de roles usan dos administradores distintos y
descriptores ya admitidos. El montaje de roles nuevos requiere
`VEC_ADMIN_GOBIERNO_ROLES_CONFIG_FILE`; las versiones específicas de Bolsa
usan sus propios montajes y consumidores. El patrón probado de B103, con
preimagen, catálogo, selección de entradas, propuesta y cierre, está en
[`habilitacion_b103_respuestas_20261012.md`](../../habilitacion_b103_respuestas_20261012.md).
No enviar concesiones libres como autoridad ni trasladar un cuerpo B103 a
otro módulo.

Los motivos de cada módulo se cargan como datos ficticios en su copia. No hay
un publicador común de motivos en este kit. El motivo que vaya a cidonia
necesita la migración revisada de su módulo y su inclusión en la candidata.

## Evidencia disponible

En la copia H se confirmaron fuentes, unidad y bootstrap con sus recibos,
la renovación de gobierno y la selección de los dos perfiles por HTTPS.
Después del reinicio vigilado, ambas selecciones se recuperaron y el lote
devolvió una preparación auditada. No se ha confirmado un lote, una versión
de rol ni un catálogo nuevo de acciones en este ensayo del kit. Esas
operaciones deben probarse con el material y los perfiles del módulo que
las necesite.
