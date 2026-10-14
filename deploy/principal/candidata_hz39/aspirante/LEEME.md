# Material y activación del aspirante en HZ39

Este guion prepara las pantallas de inscripción y cartera del aspirante en la
copia local de la candidata. El caso que se debe comprobar está en
`docs/producto/FLUJO_APP.md`, apartado 4.0. Los scripts que modifican la base
aceptan solo el contenedor de esta copia. Sus publicaciones de datos sintéticos
no son los actos de Administración que requerirá cidonia.

## Material privado

Con el respaldo de la misma copia y un destino privado fuera del repositorio:

```sh
umask 077
python3 deploy/principal/candidata_hz39/aspirante/preparar_material.py \
  --respaldo "$RESPALDO" --destino "$ENSAYO" --aplicar
python3 deploy/principal/candidata_hz39/aspirante/preparar_material.py \
  --respaldo "$RESPALDO" --destino "$ENSAYO"
```

El preparador copia el material interno a `material/`, la configuración de
incorporación a `incorporacion/` y el portal externo a `externo/`. Conserva el
certificado, la persona y la candidatura de Antonio; verifica que su cuenta y
su perfil externos sean distintos de los internos. Coteja el certificado y el
sujeto de Carmen con el perfil técnico fijo de Alta, y prepara
`identidad/consultas-rrhh.json` y `identidad/inscripcion-bolsa-rrhh.json` con
sus seis conexiones nominales. Las conexiones del proceso web usan
`localhost:5432`, TLS `verify-full` y `/vec-pg-ca.crt`. Si un destino ya existe,
el preparador lo comprueba y no reemplaza material divergente.

Montar `material/` en `/vec-material`, `externo/` en
`/vec-material-externo` y `incorporacion_web/` en `/vec-incorporacion`, esta
última solo lectura. Montar aparte la CA pública de PostgreSQL en
`/vec-pg-ca.crt`. `incorporacion/` conserva actos y conexiones de gobierno en
el host, fuera de la web. El proceso combinado puede utilizar
`/vec-incorporacion/arrancar_combinado.sh`: carga las 17 conexiones de base,
las conexiones externas nominales y el lector de inscripciones de RRHH desde
ficheros privados. Dirección autorizó mantener en este corte la conexión
heredada de gobierno de CT; su segregación queda pendiente. El script no
imprime conexiones y deja
`VEC_PORTAL_PROCESO` sin valor,
enciende `VEC_BOLSA_INSCRIPCIONES_ENABLED` y usa el material externo separado.
Si se instala el fichero aprobado de cartera, exige también sus siete
conexiones nominales antes de arrancar esa función.

Las fuentes de acciones se copian, con modo `0600`, a
`incorporacion/actos/inscripcion_bolsa_actos_v1.json` y
`incorporacion/actos/meritos_cartera_propia_v1.json`. Son entradas para los
circuitos existentes; no contienen una propuesta, aprobación ni asignación.

## Orden de activación en la copia O

1. Comprobar la historia SQL de la candidata y de la cartera. Aplicar solo las
   migraciones ausentes, en el orden de `deploy/principal/candidata_hz39/lista_sql.txt`
   y de las listas de Méritos de
   `deploy/principal/habilitacion_meritos_cartera_20261013.md`. Nunca repetir
   una migración instalada. Estos scripts no instalan migraciones.
2. Cotejar que la identidad ID7 y el snapshot CTX15 externos de Antonio ya
   existen con la cuenta, el perfil, la persona y la candidatura del material.
   En esta copia existían. Si faltan, usar las fases y preimágenes del CLI
   `vec-provisionar-candidato-externo` antes de seguir. Publicar la asignación
   externa y añadir las cinco concesiones de inscripción con los scripts de
   esta carpeta; ambos comprueban la cabeza actual y conservan la candidatura.
3. Provisionar los siete LOGIN runtime de Méritos y preparar las dos cuentas
   nuevas: una para Usuarios externo y otra para Méritos. Los snapshots
   reutilizan la persona y el certificado acreditados de Antonio. El
   exportador calcula sus alias desde una copia temporal privada del material
   externo; ID7 se publica por el CLI nominal y CTX15 por su función de
   gobierno. Se comprueban el plan, la preimagen y los recibos.
4. Publicar los roles de Usuarios y Méritos y asignarlos mediante la fachada
   AUT81. Los motivos V2 de retirada y declaración se generan desde catálogos
   sintéticos con huellas calculadas a partir de su contenido. Son solo datos
   de ensayo. El rol de Méritos tiene consultar, retirar y declarar con ámbito
   de una persona.
5. Publicar la confianza con el LOGIN administrativo nominal en un proceso
   separado. Conservar los consumidores actuales y añadir
   `usuarios_preferencias`; la cartera necesita además `meritos_cartera`.
   Si Correos e Imagen siguen encendidos, añadir también sus consumidores.
   Seguir `deploy/principal/confianza_externa_hz38.md`. El DSN publicador no
   entra en el montaje web.
6. Activar los dos ficheros de identidad externos y crear
   `incorporacion_web/` por lista positiva. El material web solo lee las
   conexiones runtime y los catálogos. El plan privado de montaje queda en
   `mountplan-aspirante.json`.

Los comandos de los pasos 2–4 y 6 se ejecutan desde el repositorio con
`$ENSAYO`, `$RESPALDO`, `$PG_COPIA`, `$CA_PG` y `$BINARIO_CLI` fijados a
rutas privadas de la copia. El exportador usa el binario ya compilado en
`$ENSAYO/scratch/vec-server`:

```sh
python3 deploy/principal/candidata_hz39/aspirante/asignar_candidato_copia.py --ensayo "$ENSAYO" --pg "$PG_COPIA" --ca "$CA_PG" --aplicar
python3 deploy/principal/candidata_hz39/aspirante/activar_inscripcion_copia.py --ensayo "$ENSAYO" --pg "$PG_COPIA" --aplicar
python3 deploy/principal/candidata_hz39/aspirante/provisionar_login_meritos_copia.py --ensayo "$ENSAYO" --pg "$PG_COPIA" --aplicar
python3 deploy/principal/candidata_hz39/aspirante/preparar_identidades_meritos_copia.py --ensayo "$ENSAYO" --pg "$PG_COPIA" --aplicar
python3 deploy/principal/candidata_hz39/aspirante/exportar_alias_meritos_copia.py --ensayo "$ENSAYO" --aplicar
python3 deploy/principal/candidata_hz39/aspirante/publicar_identidades_meritos_copia.py --ensayo "$ENSAYO" --pg "$PG_COPIA" --ca "$CA_PG" --binario "$BINARIO_CLI" --aplicar
python3 deploy/principal/candidata_hz39/aspirante/publicar_contextos_meritos_copia.py --ensayo "$ENSAYO" --pg "$PG_COPIA" --ca "$CA_PG" --aplicar
python3 deploy/principal/candidata_hz39/aspirante/publicar_roles_meritos_copia.py --ensayo "$ENSAYO" --pg "$PG_COPIA" --aplicar
python3 deploy/principal/candidata_hz39/aspirante/publicar_asignaciones_meritos_copia.py --ensayo "$ENSAYO" --pg "$PG_COPIA" --ca "$CA_PG" --aplicar
python3 deploy/principal/candidata_hz39/aspirante/publicar_motivos_meritos_copia.py --ensayo "$ENSAYO" --pg "$PG_COPIA" --aplicar
python3 deploy/principal/candidata_hz39/aspirante/activar_material_meritos_copia.py --ensayo "$ENSAYO" --aplicar
python3 deploy/principal/candidata_hz39/aspirante/preparar_incorporacion_web.py --respaldo "$RESPALDO" --ensayo "$ENSAYO" --aplicar
```

Cada comando admite una repetición sin `--aplicar` para comprobar el estado,
salvo que indique una precondición pendiente. El script de asignación conserva
la operación original cuando reconcilia una publicación anterior de la copia.
El material nominal y las conexiones quedan fuera de Git.

En cidonia, las concesiones, los perfiles y sus asignaciones se tramitan por
Administración, con dos ADMIN cuando el gobierno del rol lo exige. Las fuentes
y el orden están en
`deploy/principal/inscripcion_recorrido_20261013/gobierno_inscripcion.md` y
`deploy/principal/habilitacion_meritos_cartera_20261013.md`. Los actos
sintéticos de este clon no se exportan como permisos de cidonia.

Para el paso de alegaciones en esta copia, publicar el motivo V2 y activar las
concesiones propias del candidato después de la inscripción. El preparador toma el
catálogo y el ejemplo de `deploy/principal/alegaciones_bolsa/` y comprueba sus
huellas; conserva el inventario externo y el material existente. Antonio
conserva el mismo perfil y avanza primero a Preferencias y luego a
Alegaciones. Las nuevas instantáneas de candidato se cotejan con sus huellas
y la asignación vigente antes del recorrido.

```sh
python3 deploy/principal/candidata_hz39/aspirante/preparar_alegaciones_copia.py --ensayo "$ENSAYO" --pg "$PG_COPIA" --aplicar
python3 deploy/principal/candidata_hz39/aspirante/activar_permisos_alegaciones_copia.py --ensayo "$ENSAYO" --pg "$PG_COPIA" --aplicar
```

Cada comando permite después una repetición sin `--aplicar` para comprobar
que el estado sigue siendo el mismo. El segundo script actúa solo sobre el
candidato. No se exportan sus publicaciones sintéticas a cidonia: allí se
tramitan por Administración.

La consulta y decisión RRHH de Carmen en esta copia siguen pendientes de una
provisión compatible con el perfil fijo de Alta. Ese perfil tiene una
plantilla cerrada de Contratación temporal: ampliar directamente su rol
impide que el proceso lo consuma. La ampliación exploratoria se retiró del
guion; no se atribuye al circuito fijo un acto que no ha publicado. El
ensayo debe restaurar la copia previa a ese intento para recuperar la lectura
CT de Carmen antes de repetir el guion candidato.

Quedan estos actos de Administración antes de declarar completo el caso:

1. Tramitar en el rol externo de candidato las dos acciones de Preferencias
   (`bolsa.preferencias_bolsas.consultar_propias`,
   `bolsa.preferencias_bolsas.ordenar_propias`) y las dos de Alegaciones
   (`bolsa.alegacion.propias.consultar`,
   `bolsa.inscripcion.alegacion.presentar`) con las fuentes publicadas y dos
   ADMIN distintos. La copia usa datos sintéticos, que no son esos actos.
2. Tramitar `bolsa.inscripcion.alegacion.rrhh.consultar` en el rol RRHH B103.
   Para Carmen, resolver además el perfil técnico fijo de Alta por una vía
   que conserve su plantilla CT; no se amplía a mano ese perfil.
3. Publicar la fuente `meritos_cartera_propia_v2.json` y tramitar en la misma
   cuenta y perfil M4 de Antonio las dos acciones nuevas de comprobante:
   `meritos.cartera.comprobante.subir_propio` y
   `meritos.cartera.comprobante.descargar_propio`. El catálogo V2 solo se
   consume después de instalar AD270, AUT95, D20 y RUM7 y ensayar el peer.

La subida local de comprobantes se prepara solo tras instalar AD270 y DOC20
en el ensayo completo. Reutiliza la cuenta y el perfil de Méritos, con dos
LOGIN documentales exclusivos, la custodia privada local y el analizador
real. El archivo privado de montaje se escribe en
`mountplan-documentos-aspirante.json`; dirección monta el directorio de
custodia con escritura y solo la CA pública, el certificado y la clave del
cliente analizador en lectura. La clave de la CA nunca entra en la web. La
configuración de la subida queda en el material externo y exige un reinicio
vigilado, porque la web la lee al arrancar.
Sus tres claves binarias privadas (`correlacion.bin`, `seudonimo.bin` y
`solicitud.bin`) son distintas, de 32 bytes y modo `0600`. La retención mínima
se toma del material de Documentos de esta copia: cero días. Antes de montar
el material se comprueba con el clasificador del portal externo; la extensión
`.bin` conserva su pertenencia externa sin abrir excepciones para claves.

```sh
python3 deploy/principal/candidata_hz39/aspirante/preparar_subida_documental_copia.py --ensayo "$ENSAYO" --pg "$PG_COPIA" --aplicar
```

## Comprobación de la candidata

Con los actos anteriores acreditados, arrancar el binario de la candidata con
el script privado, comprobar `/livez` y `/readyz` y recorrer e1–e5 en Chrome.
Antonio debe recuperar la misma solicitud y los mismos recibos al repetir su
operación; RRHH debe ver esa solicitud con Carmen y dejar auditada la consulta
de méritos. Reiniciar aplicación y PostgreSQL y repetir las lecturas sin crear
otra solicitud ni otro mérito. Ensayar también el arranque con inscripción y
cartera sin configurar: el fallo debe quedar limitado a esas funciones.
Conservar el resultado de ambas pruebas junto a los actos, la configuración y
la huella del binario que se instalará.

El acta de prueba publicada para Antonio se lee de
`$ENSAYO/incorporacion/actos/acta-antonio-actual.json`, con sus referencias
actuales de participación, categoría y bolsa. El acta de la guía histórica de
inscripción no coincide con la cabeza de esta copia; el recorrido usa la
relación vigente y no inventa un acta ni una admisión.

## Relevo congelado del ensayo
Mi Bolsa sigue sin lectura funcional; el diagnóstico privado de sesión queda a cargo de dirección.
El perfil fijo CT de Carmen quedó no consumible en la copia tras una ampliación sintética; restaurar su preestado sin atribuirle actos no publicados.
La decisión vigente exige otro perfil de Bolsa para Carmen, conservando intacto el de CT; el guion de Alegaciones solo actúa sobre el candidato.
La dependencia documental se está consolidando en AD273; comprobar la lista SQL final antes de activar LOGIN, configuración o subida.
