# Enmienda O3c P6: publicación durable sin reemplazo V2

Fecha: 14 de agosto de 2026.

Tarea: `O3C-P6-CONDUCTOR-FALLO-DURABLE-V2-NOREPLACE`.

Estado: candidato técnico local. Requiere revisión funcional y de seguridad
independientes sobre el SHA exacto. No acredita O3a V3, C21, toolchain, O4,
publicación remota o CI.

## Base y hallazgo que corrige

La base exacta es
`55d6c9420e1adc3c37499b7b236ac0cfc101d6d2`, con padre
`fea52f3ddf796991c93c85cae992ca695db1ae63` y árbol
`b9857dc7d82c0a52bffa575da5957d103558408f`.

La revisión funcional `e943cdbb095322fb360e449df437b61f5330bd89`
dio `GO` acotado al mecanismo. La revisión de seguridad
`c0c6242a22a0e3b8aad074bac5f4472c96b6f6e3` dio
`NO-GO P0=0, P1=1, P2=0`: entre el precheck de inexistencia, la creación del
staging y el `mv`, un actor concurrente podía insertar un enlace simbólico en
el destino. GNU `mv` interpretaba ese enlace a directorio como contenedor,
depositaba allí el staging y devolvía cero aunque la ruta exacta nunca hubiera
sido publicada.

La reproducción independiente quedó fuera de Git en
`/srv/fabrica/revisiones/evidencia-seguridad-o3c-p6-fallo-durable-55d6c94-race-symlink-r1`.
El resultado no se compensa con las puertas verdes del primer candidato.

## Criterio único V2

El publicador solo devuelve éxito si el staging privado que él creó pasa a ser
el directorio exacto solicitado sin sustituir ninguna entrada concurrente. La
operación cumple conjuntamente:

1. el staging y el destino pertenecen al mismo directorio padre;
2. el padre, el origen y el destino aceptado no son enlaces simbólicos;
3. `mv -n -T` solicita no sobrescribir y trata el destino como la entrada
   exacta, no como un directorio contenedor;
4. antes del rename se fija la identidad `(dispositivo,inode)` del staging;
5. después del rename el origen debe haber desaparecido, el destino debe ser
   un directorio real y su identidad debe ser exactamente la fijada;
6. cualquier colisión, desvío, ausencia o identidad distinta devuelve error y
   el `trap` retira el staging privado restante;
7. solo después de esas comprobaciones se desarma la limpieza del staging.

En el entorno de gates, GNU coreutils 9.7 materializa `mv -n -T` mediante
`renameat2(..., RENAME_NOREPLACE)`. Las comprobaciones de identidad no
interpretan el retorno cero de `mv -n` como prueba suficiente, pues ese modo
también puede devolver cero cuando rehúsa una colisión.

## Checksums sin interpolación de ruta

La V2 genera `SHA256SUMS` desde dentro del staging, sobre nombres relativos
ordenados y terminados en NUL. Se elimina la sustitución `sed` que interpolaba
la ruta temporal en una expresión. El manifiesto se verifica antes del rename
y conserva los mismos contenidos mínimos, incluidos `fallo.stdout`,
`fallo.stderr`, `fallo.tsv` y `resumen.txt`.

## Write-set exacto

```text
tools/o3c_p6_conductor/fallo_durable.sh
docs/portal_vec/enmienda_o3c_p6_conductor_fallo_durable_v2_noreplace_2026-08-14.md
```

No cambia `conductor.sh`: ya calcula y liga dinámicamente la huella del helper.
Tampoco cambian G7a/G7b, fuentes Go productivas o test-only, casos, ledgers,
evidencias históricas, workflow, Docker, PostgreSQL, estado transversal o
métricas.

## Huellas y presupuesto

El publicador V2 ocupa 130 líneas y SHA-256
`b8f91102a2e98ce1e2e79ed73bfa9bd48c5d1f8ca002271dc5fc2461512bf174`.
El conductor permanece byte a byte en 182 líneas y SHA-256
`6f777f59d4e157b2c4eabb13f66789cfe11b9c01f646e9512af9af388b715538`.

El publicador sigue bajo el objetivo DEC-051 y no amplía la superficie
productiva: es una herramienta test-only de evidencia del conductor O3c.

## Oráculos y puertas

1. identidad, genealogía, árbol, write-set, modos, líneas y hashes exactos;
2. `bash -n` y ShellCheck sin exclusiones;
3. autoprueba de copia byte a byte, `NO-GO`, checksums y rechazo de una segunda
   publicación;
4. colisión determinista con directorio preexistente: el origen permanece, el
   guardia y la identidad del destino son inmutables y el helper falla;
5. colisión con enlace simbólico: el helper falla, el origen permanece y el
   directorio apuntado queda vacío;
6. mutantes de sustitución, anidamiento, confianza exclusiva en el estado de
   `mv`, identidad y exclusión de raw del manifiesto, todos muertos;
7. integraciones sintéticas ordinaria y BF con el conductor exacto: paquete
   completo en la ruta exacta, salida no cero y `SHA256SUMS` válido;
8. una única corrida canónica O3c normal/race en clon limpio propiedad de
   `orquesta`, Go 1.26.5 y `flock --close`, sin retry ni mayoría;
9. `git diff --check`, Gitleaks, permisos privados y residuos cero;
10. revisión funcional y de seguridad independientes sobre el SHA candidato.

Si la corrida canónica falla, se conserva su primer paquete y no se repite. Si
queda verde, acredita únicamente compatibilidad de este mecanismo; no revoca
`CAP_NORMAL_021`, los dos `NO-GO` de O3a V3 ni el P1 C21 histórico.

## Límites

Esta corrección no interpreta stdout o stderr, no relaja oráculos, plazos,
inventarios, cardinalidades ni aislamiento. No autoriza push, despliegue, CI
remota, producción, credenciales, integración de candidatos rechazados ni
cambios de porcentajes. Una publicación posterior por un actor con autoridad
sobre el directorio padre queda fuera del instante linealizado; durante esta
operación, ninguna entrada concurrente se sustituye ni se acepta como propia.
