# PIDEN RRHH, corte 3: preparación de entrega

**Estado: INCOMPLETO / NO EXPORTAR.** La base provisional es
`trabajo/piden-rrhh-corte3-publicable-20260928`@`8f9196fc6988561a8c46e61d1984fb8551b0ee9e`.
Incluye la corrección de B56 y la recuperación web CT133 en una historia
publicable; la fuente está pendiente de puerta completa exacta. Faltan la
ratificación del hash final por Dirección y las revisiones SQL/operaciones
exactas del paquete.
Esta carpeta no contiene binario ni copia de `web/`; tampoco instala SQL, cambia
servicios o acredita navegador. `validar_plan.sh` y `validar_web.sh` fallan
cerrados por defecto mientras no exista `fuente_final.txt` con el SHA-1 completo
ratificado. `--provisional` solo inspecciona el árbol de partida.

## Inventario SQL y dependencias

[`migraciones.txt`](migraciones.txt) parte de los **27 UP y dos deltas DBA** de
los cortes 1/2, desde la base publicada `7247682cbd1e6e630c86c290e3ddeca281456a94`.
Agrega AD3-100 antes de CT137, AD3-101 antes de Bolsa55 y Bolsa56 después de
Bolsa48: **34 entradas provisionales**. CT137 depende también de CT135;
Bolsa55 de Bolsa46; Bolsa56 de Bolsa16 y Bolsa48. Los
consumidores AD3-90…101 conservan su secuencia. El delta DBA del calculador
precede inmediatamente a Bolsa51 y el del registrador de auditoría a CT136.
`validar_plan.sh` compara el conjunto completo de `UP` y `roles_*_up.sql`
contra el delta Git exacto de la fuente, además del orden. No basta comparar
versiones máximas: cada migración tiene preimagen y postimagen propias.
`preflight_b56.sh` comprueba la preimagen B16/B48, el DBA, PG18 y la ausencia
de la marca B56 inmediatamente antes de ese `UP`, tras aplicar lo anterior;
un B56 ya instalado exige conciliación, nunca reaplicación.

Bolsa56 es **obligatoria en corte 3**: une motivo minimizado y cambio en la
consulta auditada. Aun con ella, RRHH 4.12 sigue **PARCIAL**: falta la causa
catalogada que Dirección ha trasladado al **corte 4**, con Bolsa57 y
AD3-102…105. Esas migraciones no se incluyen en este plan ni bloquean su exportación;
si sus ficheros conviven en el árbol Git, `validar_plan.sh` los excluye del
delta ejecutable de corte 3 y los rechaza si aparecen en la lista. Su orden
sus dependencias, revisión y despliegue corresponden al paquete posterior.
Bolsa49 y Bolsa pública 000003 siguen
en **NO-GO**: ni en el plan ni instaladas en las bases; el preflight detecta
huellas de tabla, función y rol. Si el binario necesitase esas migraciones,
detener esta tanda. No ejecutar `DOWN` ni reaplicar un `UP` con historia.

## Puertas de la fuente final

1. Dirección comunica hash completo final, correcciones documental/UI incluidas,
   B56 incluida y puerta completa verde del mismo árbol: `go test ./...`,
   prueba race relevante, `go vet ./...`, `scripts/verificar_calidad.sh`, Node,
   manifiestos, TLS, dependencia pública, govulncheck y escaneo de secretos.
   Cualquier cambio posterior reabre la puerta y las revisiones afectadas.
2. Incorporar esa fuente a esta rama propia, escribir **solo entonces** el hash
   en `fuente_final.txt` y ajustar `migraciones.txt` si cambia el delta. Ejecutar
   `validar_plan.sh`, `probar_plan.sh` actualizado y `validar_web.sh`. Este último
   exige que producción incluya Mi Bolsa, reincorporaciones y cliente/vista de
   borradores, y verifica el árbol web y el ZIP OSM completo. El ZIP se obtiene
   mediante `scripts/aprovisionar_cartografia_osm.sh` y debe coincidir con
   SHA-256 `0f0d78212832493c424699a42847069ae24b8b3717917780c4d64444aa250165`.
3. Dos revisiones **Sol/high independientes y exactas**: SQL/ACL/orden/preimagen,
   y operaciones/paquete/reversión. Tras corregir hallazgos, repetir ambas sobre
   el nuevo hash. Las revisiones del primer o segundo corte no aprueban el
   tercero. Solo el director principal integra en la rama canónica.

`preparar_paquete.sh` exige `fuente_final.txt` confirmado en Git y las variables
`VEC_PIDEN_C3_PUERTA_SHA` (fuente) y `VEC_PIDEN_C3_GO_SQL_SHA` /
`VEC_PIDEN_C3_GO_OPS_SHA` (ambas iguales al commit exacto de este paquete).
Sin cualquiera de ellas aborta antes de compilar o exportar. Dirección las fija
solo tras recibir la puerta y los dos dictámenes GO; ninguna es un secreto.

Los validadores locales actuales se ejecutan sin base con:

```bash
bash deploy/principal/piden_rrhh_corte3_20260928/validar_plan.sh --provisional
bash deploy/principal/piden_rrhh_corte3_20260928/probar_plan.sh
bash deploy/principal/piden_rrhh_corte3_20260928/validar_web.sh --provisional
```

## Ensayo antes de exportar

Inventariar la principal y Bolsa pública con lectura DBA: nombre/versión PG18,
roles y membresías, ACL explícitas y predeterminadas, `SECURITY DEFINER`, tipos
de fila, postimagen de funciones, versiones instaladas, recibos y contadores
testigo. Guardar la evidencia privada fuera de Git. Ejecutar
`preflight_no_go.sh --destino`; una huella B49/Pública3 o una conexión no
inventariable detiene la tanda. Los preflights de roles prueban los dos deltas
DBA con sustitución del `COMMIT` final por `ROLLBACK`; esto no equivale a
instalarlos. Los LOGIN nominales y sus secretos proceden del material privado.

Tomar `pg_dump -Fc` de la principal y de Bolsa pública, `pg_dumpall
--globals-only` de cada clúster si son distintos, y huellas SHA-256. Custodiar
globales como secreto porque pueden contener verificadores. En un clúster
PostgreSQL 18 **aislado, desechable y distinto del principal**, restaurar primero
globales y después bases nuevas desde `template0` con propietario, codificación,
locale, ACL y extensiones cotejadas. Usar `pg_restore --exit-on-error
--single-transaction`, sin omitir propietarios ni ACL. Comprobar equivalencia de
recibos, tipos de fila y contadores antes de cada `UP`. No crear una base nueva
en el clúster de la principal. Emplear `PGSERVICEFILE`/`PGPASSFILE` privados
`0600`; nunca contraseñas o DSN en argumentos, Git ni logs.

En clon limpio, ejecutar los preflights `--clon`, aplicar **una sola vez** los
`UP` pendientes en orden con `psql -X -v ON_ERROR_STOP=1`, y confirmar
postimagen/ACL tras cada commit. Ensayar cada LOGIN real, denegaciones,
concurrencia, recibos e historia. Arrancar la aplicación aislada contra el
clon, repetir tras reiniciar aplicación y PostgreSQL, y comparar referencias,
versiones, fechas y huellas sin duplicados. Si falla, descartar el clon y
restaurar otro; ningún `DOWN` sobre historia. No hay ejecutor de SQL a destino
en esta carpeta.

## Artefacto y ventana de cambio, pendientes

Tras GO SQL/ops y clon verde, ejecutar `preparar_paquete.sh` desde el
**hash final publicado**. Construye con
Go **1.26.6 linux/amd64**, `CGO_ENABLED=0 GOOS=linux GOARCH=amd64`,
`-buildvcs=false -trimpath -ldflags='-s -w'`; usar caché y carpeta temporal bajo
`/dev/shm`. Exigir `file` ELF amd64 estático y `ldd` sin dependencias. Copiar
solo `web/produccion.manifest` al artefacto, con `rsync --files-from`; incluir
el propio manifiesto, ZIP OSM verificado, SQL/roles inventariados y referencia
del commit. Crear `SHA256SUMS` de todos los ficheros y comprobarlo sobre la
copia privada final. No incluir configuración, secretos, datos o WIP ajeno.
Si se fusiona a `main` con otro hash, regenerar y revalidar; la equivalencia
de una rama previa no acredita el artefacto de `main`.

En la ventana autorizada, Dirección conserva release anterior, huellas y
copias privadas; detiene solo la aplicación, aplica exclusivamente lo pendiente
tras preflight limpio y activa binario, `web/` y manifiestos juntos. Reinicia
y exige la línea **literal** `vec server listening` posterior al arranque;
un contenedor activo sin ella es fallo. Ejecuta sondas mTLS/HTTP y Chrome real
a **1440 y 390 px** para RRHH, plantillas documentales de tres ámbitos,
reincorporación y Mi Bolsa, con actor nominal/ajeno, errores JS, carga de
activos, overflow, cookies y almacenamiento. Repite recibos e historia tras
reinicio de aplicación y PostgreSQL. No presentar borradores como firma,
aviso como entrega ni política sintética como regla aprobada.

Si falla la activación, volver al binario, `web/` y manifiestos anteriores con
sus huellas y repetir la sonda literal y recibos testigo. Conservar los commits
SQL y su historia: **sin `DOWN` y sin restaurar un dump sobre la principal con
escrituras posteriores**. Si el binario anterior no admite el esquema nuevo,
mantener la aplicación detenida y resolver mediante migración correctiva
revisada. Este documento prepara el procedimiento; no acredita exportación,
instalación, publicación ni producción.
