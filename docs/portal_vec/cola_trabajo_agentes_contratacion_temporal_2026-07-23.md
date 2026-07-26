# Cola dirigida de agentes — contratación temporal

Fecha de corte: 23 de julio de 2026.

Estado: cola preparada por Dirección. Una tarea solo pasa a activa cuando
Dirección comunica expresamente su identificador y el SHA que debe revisar o
usar como base. Ningún agente se autoasigna otra tarea al terminar.

## Reglas comunes e innegociables

Cada agente debe leer completos `AGENTS.md`, `ORQUESTACION_AGENTES.md`,
`ESTADO_PROYECTO.md`, el tablero de contratación temporal y su encargo
específico. Todos los worktrees VEC se crean exclusivamente dentro de
`/home/alberto/Trabajo/VEC_Diputacion_app/.worktrees/`.

Se exige en todas las tareas:

- código, nombres de dominio, documentación, errores propios y pruebas en
  castellano coherente, salvo términos técnicos universalmente adoptados;
- i18n real: ninguna etiqueta o mensaje funcional nuevo queda incrustado como
  texto de presentación;
- arquitectura hexagonal estricta: dominio y aplicación no importan
  adaptadores; `ports` contiene contratos, DTO mínimos y validación local
  neutral, nunca coordinación concreta de varios colaboradores;
- puertos y adaptadores intercambiables para PostgreSQL/Oracle, almacenamiento,
  firma, identidad, comunicaciones y restantes infraestructuras;
- un único caso de uso común para web, aplicación de escritorio, API, CLI y
  MCP; el navegador no es autoridad;
- cero cookies, `localStorage`, `sessionStorage`, credenciales libres,
  cabeceras declarativas de rol o identidad y autorización aportada por el
  cliente;
- denegación por defecto, privilegio mínimo, segregación de funciones,
  minimización, trazabilidad íntegra y errores redactados;
- cero secretos, credenciales, certificados, rutas privadas o datos personales
  reales en código, pruebas, documentación, commits y logs;
- productor distinto de revisor; un `NO-GO` bloquea integración;
- commits pequeños, árbol limpio y pruebas registradas. El agente no integra,
  no publica y no borra ramas.

Cada productor utilizará dos subagentes revisores en solo lectura cuando su
entorno lo permita: uno funcional/concurrencia y otro
seguridad/hexagonalidad. La revisión formal posterior seguirá siendo de un
agente independiente del productor.

## Orden ejecutivo

| Orden | Identificador | Trabajo | Dependencia | Estado |
| ---: | --- | --- | --- | --- |
| 1 | `REV-O2-05-A` | Primera revisión hostil de SQL/replay O2-05 | `cfc935a` | Activa |
| 1 | `REV-O2-05-B` | Segunda revisión independiente O2-05 | `cfc935a` | Activa |
| 1 | `COR-O3-02` | Corregir recibo tras `COMMIT` y hexagonalidad | `bd549ff` | Activa |
| 2 | `REV-O4-02` | Revisión independiente de cobertura | `a0c7ecf` | Preparada |
| 2 | `REV-O2-08A` | Revisión de API interna aislada | `3fb1d5c` | Preparada |
| 2 | `REV-O2-09A` | Revisión web, accesibilidad y contrato neutral | `4d2f169` | Preparada |
| 3 | `ACT-O2-06A` | Alinear diseño del adaptador con el O2-05 integrado | SHA pendiente | Bloqueada por GO O2-05 |
| 4 | `IMP-O2-06` | Implementar adaptador Go y reconciliación | diseño O2-06A revisado | Bloqueada |
| 5 | `IMP-O2-07` | Componer identidad, PDP, VEC, PostgreSQL y servicio | O2-06 | Bloqueada |
| 5 | `INT-O2-08B` | Registrar API real y alinear contrato web/API | O2-07 + GO O2-08A/O2-09A | Bloqueada |
| 6 | `E2E-O2-10` | Probar navegador a recibo durable y aceptación | O2-05…O2-09 | Bloqueada |

Las tareas de la misma orden pueden ejecutarse a la vez si sus write-sets son
disjuntos. Dirección integra de dentro hacia fuera y repite las puertas sobre
el árbol conjunto.

## `REV-O4-02` — revisión independiente de cobertura

Solo lectura sobre `agent/ct-o4-02-rework2`, SHA `a0c7ecf`.

Debe intentar romper:

1. el suelo temporal monótono en todas las lecturas, incluida regresión entre
   catálogo, fuente, verificación, preconsumo y retorno;
2. expiración exclusiva y retroceso que no alcance el consumidor;
3. `COMMIT` con recibo válido que compita con cancelación o plazo;
4. recibo adulterado, petición distinta, replay y concurrencia;
5. segregación de autoridades y ausencia de coordinación concreta en `ports`;
6. copias defensivas, límites, errores redactados y neutralidad de canal.

Puertas: focal al menos cincuenta repeticiones, `-race`, global, `vet`,
tamaños, `diff-check`, Gitleaks del rango y `merge-tree` contra el SHA de
integración comunicado. Entrega `GO` o `NO-GO` con contraejemplos.

## `REV-O2-08A` — revisión independiente de API interna

Solo lectura sobre `agent/ct-o2-08-api`, SHA `3fb1d5c`.

Debe validar:

- lista positiva y cierre estricto del JSON, Unicode/NFC, duplicados,
  profundidad, tamaño, fechas civiles, importes y referencias;
- ausencia completa de autoridad aportada por cuerpo, query, cookies,
  `Authorization`, cabeceras de rol/identidad o almacenamiento web;
- autoridad obtenida únicamente del contexto confiable y copias defensivas;
- recibo interno validado y proyección pública minimizada;
- precedencia segura de recibo válido tras efecto y resultado indeterminado;
- catálogo de errores i18n, redacción, correlación opaca y cabeceras seguras;
- OpenAPI 3.1 cerrado y validado también por una herramienta semántica
  aprobada, no solo por parseo YAML;
- que el adaptador no registre ruta, cree servicios falsos ni invada dominio,
  aplicación o PostgreSQL.

Debe ejecutar pruebas focales repetidas, `-race`, globales, `vet`, análisis del
OpenAPI, Gitleaks y `merge-tree`. No modifica el candidato.

## `REV-O2-09A` — revisión independiente de interfaz

Solo lectura sobre `agent/ct-o2-09-web`, SHA `4d2f169`.

Debe comprobar con DOM y navegador real cuando esté disponible:

- teclado completo, foco visible y restaurado, lector de pantalla, mensajes
  asociados, contraste, zoom y reducción de movimiento;
- estados vacío, carga, revisión, envío, cancelación indeterminada, error y
  recibo, sin doble efecto;
- i18n sin HTML inyectable y Unicode hostil;
- ningún dato sintético presentado como real, ninguna autoridad del cliente y
  ningún uso de cookies o almacenamiento del navegador;
- comando cerrado, clave idempotente CSPRNG mantenida solo en memoria y
  reutilizada exclusivamente para el mismo contenido;
- desmontaje sin oyentes o promesas residuales;
- CSS heredado del tema principal y sin romper la navegación del portal;
- contrato exacto con O2-08A, documentando cualquier divergencia como
  bloqueo de integración.

Ejecuta la suite JS repetida, análisis estático disponible, prueba de
accesibilidad automatizada, inspección visual en escritorio y Gitleaks. No
conecta un adaptador falso.

## `ACT-O2-06A` — congelar diseño contra O2-05 integrado

Se abre únicamente cuando Dirección comunique el SHA integrado de O2-05. Parte
de los commits `2c800fa` y `4cc4422`, pero no los da por vigentes sin comparar
la firma final.

Write-set:

- documento de diseño O2-06A;
- sus dos actas de revisión;
- un inventario de firma Go↔SQL generado o verificable.

Debe actualizar nombre, orden, tipos, nulabilidad, cánones, hashes, marcador,
barrera y ACL reales de la función final. El verificador falla si el SHA o la
firma cambian. El diseño debe cerrar resultado de `COMMIT`, reconciliación
0/1/>1, replay, reinicio, tres intentos totales solo para `40001`/`40P01`,
plazos, pool/rol y redacción.

No implementa el adaptador. Entrega nuevo GO independiente PostgreSQL/pgx y
seguridad/hexagonal.

## `IMP-O2-06` — adaptador Go definitivo

Se crea desde el SHA que contenga O2-05 y O2-06A revisados.

Write-set previsto:

- `internal/modules/contrataciontemporal/adapters/postgres/**`;
- pruebas focales de aplicación estrictamente necesarias;
- documentación técnica del adaptador.

Debe:

1. mapear el puerto común a la firma SQL congelada sin serialización libre;
2. usar pool/rol exclusivos, transacción nueva y `SET LOCAL`;
3. reintentar solamente `40001` y `40P01`, con tres intentos totales;
4. distinguir éxito, rollback concluyente y `COMMIT` indeterminado;
5. reconciliar en conexión nueva, `READ COMMITTED READ ONLY`, tras la misma
   barrera y con dos sentencias para renovar instantánea;
6. aceptar solo una fila íntegra y validar el recibo completo;
7. no repetir el efecto después de un `COMMIT` incierto;
8. sobrevivir reinicio sin memoria, txid, WAL/LSN ni reloj cliente;
9. redactar errores y borrar material sensible efímero;
10. demostrar PostgreSQL real, concurrencia, respuesta perdida, corrupción,
    replay, rotación/revocación y ACL.

## `IMP-O2-07` — composición real

Se abre después del GO de O2-06. No añade lógica de dominio.

Debe construir la composición interna con identidad de garantía alta,
sesión/perfil/organización confiables, PDP, emisor VEC segregado,
PostgreSQL/KMS/HSM mediante adaptadores y el servicio de aplicación común. El
arranque falla cerrado si falta cualquier dependencia productiva. No existe
fallback DEMO, credencial embebida ni lectura de autoridad desde HTTP.

Debe retirar de la composición productiva la preparación durable separada y
registrar únicamente la transacción atómica. Incluye pruebas de arranque,
perfiles, capacidades, revocación y canales equivalentes.

## `INT-O2-08B` — conexión API/web sin cambiar el núcleo

Integra únicamente candidatos O2-08A y O2-09A con GO, sobre O2-07.

Debe resolver las brechas documentadas: catálogos autoritativos, relaciones
centro–contacto y categoría–grupo, referencias documentales gobernadas,
importe interoperable, publicación del manifiesto/activos, router del portal y
traducción neutral de resultado indeterminado. El cliente HTTP es un adaptador
de presentación; la vista conserva el ejecutor neutral para escritorio u otros
canales.

No se permite una ruta que devuelva éxito simulado.

## `E2E-O2-10` — prueba de la primera vertical

Debe acreditar, en entorno efímero y reproducible:

```text
navegador
→ frontera HTTP interna
→ contexto confiable
→ caso de uso común
→ autorización y capacidad VEC
→ PostgreSQL
→ recibo durable
→ proyección minimizada
→ pantalla
```

Incluye éxito, denegación, dato inválido, replay, doble envío, concurrencia,
cancelación antes y después de `COMMIT`, respuesta perdida, reconciliación,
reinicio, corrupción, accesibilidad y ausencia de cookies/almacenamiento. La
evidencia E2E no equivale todavía a identidad corporativa, HSM/KMS o TSA
productivos; esas dependencias externas quedan nombradas como bloqueos de
producción, nunca simuladas como verdes.

La tarea termina con informe reproducible y guion de aceptación para RRHH.
Solo Dirección, después de pruebas conjuntas y acta, puede marcar O2-10 y la
primera vertical como cerrados.

## Integración y limpieza

Dirección aplica cada serie de commits sobre
`feature/contratacion-temporal`, ejecuta pruebas conjuntas y publica esa rama.
Después verifica que:

1. el SHA remoto coincide;
2. el árbol de integración está limpio;
3. el commit candidato es ancestro o su contenido está íntegramente
   incorporado;
4. no existe trabajo útil sin integrar en el worktree;
5. ningún proceso está usando el worktree.

Solo entonces elimina el worktree y la rama ya fusionada, sin `--force`. Los
`NO-GO`, candidatos pendientes, ramas sucias y evidencias no integradas no se
limpian.
