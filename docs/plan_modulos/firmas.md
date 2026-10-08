# Firmas de Contratación temporal

**Estado a 8 de octubre de 2026:** la firma de dos personas no funciona de punta a punta, porque la composición de las dos vías de firma (`componerFirmasR5`) solo se ejecuta en pruebas; faltan la identidad del firmante, el decisor y emisor, la composición en el arranque real y el recorrido.

La firma se hace con GrxFirma por el protocolo `afirma://` desde el navegador; la verificación, con su validador como servicio aparte. Ningún documento de VEC sale hoy firmado. Base: `origin/main@033fda6ed`.

## Lo que falta, en orden

| # | Qué | Quién | Cómo se comprueba |
| --- | --- | --- | --- |
| 1 | **Identidad del firmante (4c-5, parte 3a).** La ruta `registro-vec` necesita su propia autoridad de ruta, como la del empleado: certificado del resolvedor de desarrollo, cápsula de un solo uso, sesión sintética con fecha de retirada y contexto `{empleado}`. Configuración privada `identidad/firma-vec.json` fuera de Git, con plantilla sin secretos. Doble revisión de seguridad antes de seguir. | Codex-V. Ramas sin PR: `codexv-origen-firma-vec-20261007`, `codexv-admision-garantia-firma-dev-20261007`, `codexv-politica-firma-dev-20261008`, `codexv-firmas-limite-20261007`. | Pruebas con otro certificado, otro perfil, otra persona, cápsula repetida, política vencida y empleado ausente. |
| 2 | **Decisor y emisor (3b).** Decisor sobre la asignación real, emisores de `firma_vec.v2` y de la consulta R5, fuente nominal de la vía VEC y `NuevoEmisorConAmbitos`. | Codex-V. Rama `codexv-firma-emisor-preparada-20261007`. | Decisión V3 real emitida y consumida en un clon, sin actor ficticio. |
| 3 | **Composición en el arranque real (4c-8).** Rutas `original`, `preflight`, `registro-vec` y `registro-externo`, con perfil fijo de la vía externa, listas de rutas y de transportes mTLS. | Codex-V, después de 1 y 2. | El arranque real compone las dos vías con dependencias reales. |
| 4 | **Recorrido con dos personas**: navegador, dos firmas, mismo PDF verificado, justificante y reinicio, con auditoría de cada descarga. | Claude, después de 3. | Capturas y recibos iguales antes y después de reiniciar aplicación y PostgreSQL. |
| 5 | **Verificación de firma en cidonia.** El informe del 08/10 daba `VEC_FIRMA_VERIFICACION_ENABLED=false`; comprobarlo y encenderlo con el validador. | Claude, en el despliegue. | Un PDF firmado se verifica desde la ficha. |
| 6 | **Bootstrap sin escrituras directas en tablas de autorización** ni `SET ROLE` a propietarios; después, retirar la herencia de `vec_ad3_o207_gobierno`. No bloquea el recorrido, pero reduce lo que puede leer ese LOGIN. | Equipo de administración (K). | Arranque en un clon sin esas escrituras. |

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
