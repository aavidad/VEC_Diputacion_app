# Histórico del plan de firmas

Copia conservada el 20/10/2026 del documento anterior de `origin/main`.
Cada apartado conserva su fecha de corte. Sus ramas, cifras, responsables
e instrucciones de instalación describen aquel momento. No son la cola
vigente ni autorizan actuaciones sobre un servidor. El plan actual está
en [docs/plan_modulos](../../plan_modulos/firmas.md).

---

# Firmas de Contratación temporal

> **Nota del 15/10/2026.** Este plan es del 08/10 y no se ha reescrito. Con
> #1137, fusionada el 15/10, la firma funciona en una copia local: el informe
> con las firmas de Técnico y Jefatura antes de Intervención y la resolución
> con Jefatura, Diputación y Secretaría (cinco firmas reales con GrxFirma), y
> los dos PDF se descargan desde la ficha tras reiniciar aplicación y
> PostgreSQL. Falta el recorrido completo del caso 4.0 y la instalación en
> cidonia. Las tareas vigentes están en el
> [tablero](../../producto/TABLERO_TAREAS.md).

**Estado a 8 de octubre de 2026:** la firma de dos personas no funciona de punta a punta, porque la composición de las dos vías de firma (`componerFirmasR5`) solo se ejecuta en pruebas; faltan la identidad del firmante, el decisor y emisor, la composición en el arranque real y el recorrido.

La firma se hace con GrxFirma por el protocolo `afirma://` desde el navegador; la verificación, con su validador como servicio aparte. Ningún documento de VEC sale hoy firmado. Base: `origin/main@033fda6ed`.

## Lo que falta, en orden

| # | Qué | Cómo se comprueba |
| --- | --- | --- |
| 1 | **Identidad del firmante (4c-5, parte 3a).** La ruta `registro-vec` necesita su propia autoridad de ruta, como la del empleado: certificado del resolvedor de desarrollo, cápsula de un solo uso, sesión sintética con fecha de retirada y contexto `{empleado}`. Configuración privada `identidad/firma-vec.json` fuera de Git, con plantilla sin secretos. Doble revisión de seguridad antes de seguir. | Pruebas con otro certificado, otro perfil, otra persona, cápsula repetida, política vencida y empleado ausente. |
| 2 | **Decisor y emisor (3b).** Decisor sobre la asignación real, emisores de `firma_vec.v2` y de la consulta R5, fuente nominal de la vía VEC y `NuevoEmisorConAmbitos`. | Decisión V3 real emitida y consumida en un clon, sin actor ficticio. |
| 3 | **Composición en el arranque real (4c-8).** Rutas `original`, `preflight`, `registro-vec` y `registro-externo`, con perfil fijo de la vía externa, listas de rutas y de transportes mTLS. | El arranque real compone las dos vías con dependencias reales. |
| 4 | **Recorrido con dos personas**: navegador, dos firmas, mismo PDF verificado, justificante y reinicio, con auditoría de cada descarga. | Capturas y recibos iguales antes y después de reiniciar aplicación y PostgreSQL. |
| 5 | **Verificación de firma en cidonia.** El informe del 08/10 daba `VEC_FIRMA_VERIFICACION_ENABLED=false`; comprobarlo y encenderlo con el validador. | Un PDF firmado se verifica desde la ficha. |
| 6 | **Bootstrap sin escrituras directas en tablas de autorización** ni `SET ROLE` a propietarios; después, retirar la herencia de `vec_ad3_o207_gobierno`. No bloquea el recorrido, pero reduce lo que puede leer ese LOGIN. | Arranque en un clon sin esas escrituras. |

La vía externa y la consulta de RRHH siguen sin unidad (decisión del 06/10, `dudas.md` 148 y `docs/estudio_requisitos/pendientes_v2.md`).

## Esperan a RRHH

- Delegaciones de firma y suplencias: cómo llegan y si una persona con dos cargos puede firmar dos pasos (dudas 122, 128 y 143).
- Qué se firma en Firmadoc y qué con AutoFirma (duda 74). El conector de Firmadoc existe, apagado.

## Ya hecho, no reencargar

- GrxFirma V2, kit de material y recuperación nominal: #558, #579, #588, #656, #657, #702, #703, #705–#708.
- SQL de la cadena, recuperación y plan: #697, #698, #701 (AD177, AD178, AUT41, CT175, CC7, CT176). Gobierno del plan con grupo propio y ámbitos: AD200, AD201, CC9.
- Composición R5 siempre con el plan: #723. Original firmable (4c-6): #746. PDF anterior custodiado (4c-7): #748.
- Selección central del firmante (4c-4): #776, #820, #821.
- Unidad en la decisión de firma (opción B): #780, #823, #824, #825.
- Audiencias de firma V2 y vía externa (4c-5, partes 1 y 2): #826, #827.
