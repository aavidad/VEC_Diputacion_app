# Reconstruir y comprobar el clon H6 local

Corte documental de Codex-M, 1 de octubre de 2026, sobre `c03072f41`.
El clon conserva H1 y las 62 SQL en `awaiting_ad132`, con `ready:false`.
AD132, canario, DB_READY/READY, runtime y SQL posterior de main siguen pendientes.
Los ocho recorridos Chrome y sus manuales finales también siguen pendientes.
No escribir en cidonia, ejecutar DOWN ni reaplicar SQL instalada.

## Entradas que deben quedar fuera de Git

Solicitar a Dirección las rutas absolutas y las huellas aprobadas. Conservar
paquetes, actas, material mTLS, alias y claves en directorios privados externos
al repositorio. No añadir contraseñas, certificados ni datos de identidad a Git.

| Entrada | Huella fija |
| --- | --- |
| Tar H1 sintético | `d1c2e38a85f872e83b6ba9eca6b660b7a5d2ff5d39d4aca612a81ec30a496e5b` |
| TGZ H6 | `b1dbd972a490ee30ce9bb797799a8ea936f0abb7c239aec296aac6ef1153eef5` |
| Fuente SQL H6 | `73e56c106d12fdda0bd16d6fe573503c42c5495f` |
| Imagen local PostgreSQL 18.4 | `sha256:1bf3d6960db467e87a506daef30feb41fecc23b7c5f96b157e873059f2ffb50a` |

El lock, el manifiesto de guiones y el normalizador tienen pines externos propios.
Han cambiado durante el trabajo: cotejar sus bytes y la aprobación vigente antes
de usarlos; no copiar una huella de una nota histórica ni deducir aprobación de
un `sha256sum`. No descargar ni sustituir la imagen fijada silenciosamente.

## H1 y SQL62, una sola vez

Desde la raíz del worktree, definir las variables del ejemplo con las rutas y
SHA entregados por Dirección. `CLON_NUEVO` debe ser una ruta absoluta nueva,
fuera de Git y sin enlaces; su padre debe existir. El guion crea el directorio
privado. No crearlo antes ni reutilizar un estado fallido.

```bash
# Definir antes: H6_TGZ, H6_LOCK, LOCK_SHA, H1_TAR, REPO,
# NORMALIZADOR, NORMALIZADOR_SHA, GUIONES_MANIFIESTO, GUIONES_SHA,
# CLON_NUEVO. Son rutas absolutas o SHA aprobados externos.
export VEC_RECORRIDOS_ESTADO="$CLON_NUEVO"
export VEC_RECORRIDOS_REFERENCIA=73e56c106d12fdda0bd16d6fe573503c42c5495f
entradas=(
  --package-tar "$H6_TGZ" --release-lock "$H6_LOCK"
  --approved-package-sha256 b1dbd972a490ee30ce9bb797799a8ea936f0abb7c239aec296aac6ef1153eef5
  --approved-lock-sha256 "$LOCK_SHA"
  --h1-state-file "$H1_TAR"
  --approved-h1-sha256 d1c2e38a85f872e83b6ba9eca6b660b7a5d2ff5d39d4aca612a81ec30a496e5b
  --git-repo "$REPO" --normalizer-path "$NORMALIZADOR"
  --approved-normalizer-sha256 "$NORMALIZADOR_SHA"
  --guiones-manifest "$GUIONES_MANIFIESTO"
  --approved-guiones-sha256 "$GUIONES_SHA"
  --expected-pg-image-id sha256:1bf3d6960db467e87a506daef30feb41fecc23b7c5f96b157e873059f2ffb50a
)
bash scripts/recorridos/preparar_clon.sh plan "${entradas[@]}"
# Solo tras cotejar el plan y la autorización de esta fase local:
bash scripts/recorridos/preparar_clon.sh preparar-sql "${entradas[@]}"
```

El plan valida las entradas sin instalar. `preparar-sql` restaura H1 y aplica
62 SQL originales, en grupos H3/H4/H6 de **8, 9 y 45**. El resultado esperado
es `awaiting_ad132`, `sql_count:62`, `ready:false`. Conservar el
`acta_sha256` observado en una acta externa y comunicarlo por el canal; no
convertirlo en aprobación de AD132. Para comprobar el mismo estado:

```bash
# ACTA_SQL62_SHA viene del acta externa conservada tras la primera ejecución.
bash scripts/recorridos/preparar_clon.sh verificar-sql "${entradas[@]}" \
  --approved-acta-sha256 "$ACTA_SQL62_SHA"
```

`verificar-sql` compara en lectura la identidad física, postimagen y journal.
Un fallo o marca de operación incierta exige conservar la evidencia y revisar
un estado nuevo; no repetir `preparar-sql` sobre ese directorio.

La ejecución local del guion ya obtuvo acta `sql62-fase.json` SHA
`b995f8feb72d62eb6f56282d0a3e0fdae8167103aeeb66cb9c4689774f762a40`.
La verificación devolvió `verified:true` antes y después de reiniciar el mismo
PostgreSQL, conservando esquema, roles, ACL y las 62 rutas/bytes. El acta externa
del ensayo tiene SHA `c48544c55c29c699f48ed5f4b6aa8954b12f18b01a06cf7b17e1f7a047a95475`.
La segunda copia del ensayo se retiró; estas huellas no corresponden a una nueva
reconstrucción. La recuperación acredita SQL local, sin aplicación ni navegador.
`preparar` completo y `reiniciar` continúan bloqueados en el orquestador.

## Material externo y alias offline

Son fases separadas de PostgreSQL. Las rutas deben corresponder a las ubicaciones
privadas aprobadas por Dirección: el exportador fija también su identidad.
La acreditación de procedencia no concede permisos.

```bash
# Definir FUENTE, ACUSE, MATERIAL, BINARIO y ALIAS_SALIDA con rutas aprobadas.
bash scripts/recorridos/preparar_clon.sh preparar-material-externo \
  --fuente "$FUENTE" --acuse "$ACUSE" --directorio "$MATERIAL"
bash scripts/recorridos/preparar_clon.sh exportar-alias \
  --binario "$BINARIO" --fuente "$FUENTE" --acuse "$ACUSE" \
  --material "$MATERIAL" --salida "$ALIAS_SALIDA"
# Usar el SHA del recibo conservado en el canal, sin recalcularlo desde la salida.
bash scripts/recorridos/preparar_clon.sh verificar-alias \
  --binario "$BINARIO" --fuente "$FUENTE" --acuse "$ACUSE" \
  --material "$MATERIAL" --salida "$ALIAS_SALIDA" \
  --replay-receipt-sha256 ed20faf707078f414e7a8722b025488e5000c44c33d23f7bd824b303be272342
```

Fuente acreditada por Claude a las 01:53 CEST: SHA
`f7fcd35db52d1776e7f0ea34dabbe5c86466c24b352b0007690470f32a4df226`;
acuse privado: `a5bb87b1cc25cbf9bfd21d6a6eaa9d0e22c3675c7c2b33b443125a1bfe4ce30d`.
El material mTLS sintético se generó una vez, con recibo
`d2f7c6b5e7253f1c300f4c34428e05e4c8f7099526ecaa322c285d3768b653a4`.
Custodia y runtime están separados; conservar las claves CA/cliente/P12 en custodia.

El alias offline ya exportado conserva 487 bytes, SHA
`46bee08751b6d034994d23fc2c6fee4eb5dd5e099169e0336c758c4ea87c6870`,
y el recibo usado arriba. Su binario histórico procede de `main@460e120c2`,
SHA `84f6ba90d77b00ab33b739b9778f8515a0317e8568941eabe568555870246136`.
Los replay verificados conservaron bytes y fechas sin otro exportador ni
regeneración de claves. No se han provisionado identidades/permisos en SQL.
Una interrupción se conserva para revisión; no regenerar ni reexportar para
ocultar el fallo. El binario de arranque final necesitará su propio pin.

## Trabajo pendiente y guiones responsables

- [Orquestador H6](clon_h6_orquestador.py) y [entrada Bash](preparar_clon.sh):
  H1/SQL62; la composición completa conserva bloqueos explícitos.
- [Material externo](clon_material_externo_offline.py) y [alias](clon_alias_export.py):
  productores offline con recibos separados, sin aprobación AD132 ni READY.
- [Plan postmain](clon_postmain_plan.py): lectura de commit/tree, lista causal y
  SHA de seis SQL RPT. La ampliación **RPT6+B6** está en una candidata separada;
  ambas listas siguen siendo un plan de lectura, pendiente de revisión/aprobación
  y recibos previos, sin instalación. No incorporar SQL descubierta en main por
  inferencia. **U17 queda diferida**.

AUT26 y su recibo de extensión preAD132, LOGIN nominal, sesiones CAS vinculadas
y provisión siguen pendientes. AUT26 no se suma al journal de 62 ni modifica
el paquete histórico. Canary/preview, aprobación externa y ejecución AD132
necesitan sus autoridades y recibos exactos; después se podrá revisar READY,
la transición postmain y runtime. Este README no aporta un comando AD132 ni
una aprobación para esas fases.

Las huellas y evidencias proceden de las notas de Codex-M del canal compartido
`CANAL_CLAUDE_CODEX.md`, conservado fuera del árbol de esta candidata. Consultar
las notas vigentes antes de reconstruir; los manuales finales esperan el
recorrido completo de una persona.
