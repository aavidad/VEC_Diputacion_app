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

`bash scripts/recorridos/preparar_clon.sh estado` muestra la fase anotada en
el diario H6 sin requerir el marcador del clon antiguo. Su salida
`v2_sin_revalidacion_viva` y el número de SQL **declaradas** son informativos:
para confirmar las instalaciones se necesita `verificar-sql` con su acta fijada.
`estado` nunca declara `ready:true` por leer el diario.

La ejecución local del guion ya obtuvo acta `sql62-fase.json` SHA
`b995f8feb72d62eb6f56282d0a3e0fdae8167103aeeb66cb9c4689774f762a40`.
La verificación devolvió `verified:true` antes y después de reiniciar el mismo
PostgreSQL, conservando esquema, roles, ACL y las 62 rutas/bytes. El acta externa
del ensayo tiene SHA `c48544c55c29c699f48ed5f4b6aa8954b12f18b01a06cf7b17e1f7a047a95475`.
La segunda copia del ensayo se retiró; estas huellas no corresponden a una nueva
reconstrucción. La recuperación acredita SQL local, sin aplicación ni navegador.
`preparar` completo y `reiniciar` continúan bloqueados en el orquestador.

El clon local conservado con sufijo `-b` tiene diario SQL62 y preimagen
comprobada, pero se creó antes de emitir el recibo `sql62-fase.json` de este
guion. No debe construirse ese recibo a posteriori ni copiarse el del ensayo
retirado: identifica otra copia física. Para usar el contrato P/A/L local hace
falta una copia viva nueva con `preparar-sql` y pines de lock/guiones aprobados
**para ese fin**. La copia `-b` se conserva solo como evidencia y no se
reaplican sus SQL.

Dirección aprobó después una copia desechable `-c` con lock
`b85359675b20367d204342352db66ff76806e026a82d96c26fa63eae875fc9d1`
y manifiesto de guiones `b6b580f6530c0f35863310dc8bdc0fef8a12b1c3cdf5832b546eda980f06f7ff`.
Su recibo SQL62 SHA
`1707793ca68bed3e21e23f3459a2fc04e90f5576e7192ca040b5c169ffac5077`
se verificó antes y después de reiniciar el mismo PostgreSQL. Esa copia se
retiró según la aprobación: contenedor y PGDATA de `/dev/shm` ya no existen;
el recibo y el acta de retiro permanecen como evidencia privada. `-c` no
acredita AD132 ni sirve como clon vivo para P/A/L o Chrome. `-b` sigue intacta.

El paquete H6 canónico conserva una lista propia de 45 rutas, SHA
`18f977413a431fb4e112577ac989e8c80cc1f678002494902e6394adebf8b0b2`.
Un cotejo de solo lectura encontró las 45 rutas y huellas, en el mismo orden,
en las posiciones 18–62 del diario local y en el commit fuente fijado. Las
primeras 17 entradas son H3/H4. Esta comparación de bytes no sustituye el
recibo físico del kit D ni la validación de historia y permisos.

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
- [Canario por archivo](clon_h6_archive_controller.py): la acción explícita
  `canario-archivo` ya tiene controlador de intento único y publicación
  conjunta del plan/recibo. Su autoridad de material definitivo está ausente:
  deniega antes de leer estado, bloquear o llamar a Docker. Las pruebas con
  dobles no acreditan ejecución real ni habilitan `preparar`/`reiniciar`.
- [Plan postmain](clon_postmain_plan.py): lee commit, árbol, cuatro listas causales y
  SHA de **RPT6+B6+AD136+B2**. Es un plan de lectura pendiente de aprobación y
  recibos previos; no instala SQL. No incorporar SQL descubierta en main por
  inferencia.
  **U17 queda diferida**.

La acción `bash scripts/recorridos/preparar_clon.sh plan-postmain --help`
enumera los pines externos requeridos. Para inventariar un corte hay que pasar
el commit completo de `origin/main` y cada SHA aprobado; sin ellos deniega
antes de consultar el estado del clon. Esta acción solo lee objetos Git y no
crea un recibo de instalación.

`origin/main@77ea7a762` añadió AD136 después de RPT6 y B6. El plan v3 la
inventariaba como operación 13, con DOWN y sonda no ejecutables. Su instalación
exige la postimagen B y las definiciones históricas exactas de
AD113/Documentos9. `origin/main@b79b51e21` añadió después las 13 UP de B2:
el plan v4 las inventaría en posiciones 14–26, junto a 22 acompañantes no
ejecutables. La sonda AD136 no reemplaza la postimagen final B2.
`origin/main@960795f30` añadió un fixture E3 que crea roles y sustituye
fachadas de autorización con dobles; v5 lo inventaría como
`excluded_lab_fixture` solo si coinciden la ruta y el SHA fijo revisado
`bfaaccaefeea37cca46e4cdef12a03bbad2a8b17bb882234869b885a56f3ece6`.
Queda fuera de operaciones y acompañantes, con prohibición explícita de
ejecución en el clon causal. Cualquier cambio en sus bytes u otra SQL nueva
requiere otra revisión; el plan sigue pendiente de aprobación y no ejecutable.

AUT26 y su recibo de extensión preAD132, LOGIN nominal, sesiones CAS vinculadas
y provisión siguen pendientes. AUT26 no se suma al journal de 62 ni modifica
el paquete histórico. Canary/preview, aprobación externa y ejecución AD132
necesitan sus autoridades y recibos exactos; después se podrá revisar READY,
la transición postmain y runtime. Este README no aporta un comando AD132 ni
una aprobación para esas fases.

[Validador puro P/A/L](clon_fuente_autorizacion_login.py): comprueba bytes de
recibos candidatos, pines externos previos y los cambios nominales de roles,
sin leer archivos ni PostgreSQL. Su puerta operativa está ausente. Los guiones
P/A/L actuales de D esperan un volumen anónimo, mientras este clon local
conserva PostgreSQL en un bind de `/dev/shm`; no se usarán sus recibos remotos
como prueba de este clon. D debe entregar una variante física local revisada
y M deberá atestarla antes de conectar este validador al CAS.

Las huellas y evidencias proceden de las notas de Codex-M del canal compartido
`CANAL_CLAUDE_CODEX.md`, conservado fuera del árbol de esta candidata. Consultar
las notas vigentes antes de reconstruir; los manuales finales esperan el
recorrido completo de una persona.
