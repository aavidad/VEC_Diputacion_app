# Cronos y Dietas: qué falta para instalar — 4 de octubre de 2026

Inventario sobre `origin/main@ab90f99911dba568c62ddedfa094640ee47c9c37`.
Dirección comunicó la instalación de H9 sobre `39ec9858d` el 3 de octubre
a las 22:55: 40 archivos SQL del paquete, sin las cadenas opcionales que
enumera su inventario. No se ha consultado ni modificado cidonia en este corte.
Cronos y Dietas siguen ocultos en el portal por orden de Alberto.

## Lo que se puede aprovechar

| Capacidad | Disponible en main | Falta para el uso conectado |
| --- | --- | --- |
| Cronos: saldo, movimientos y permisos propios | Casos de uso, adaptadores PostgreSQL, rutas y pantallas; informes PDF con ejemplos sintéticos. | Configuración nominal, permisos específicos de exportación y auditoría común. La pregunta 118 delimita campos y tipos exportables. |
| Cronos: correcciones de marcaje | Puerto histórico CRN11 y proveedor de Personal de #397/#421. | La conexión de arranque está conservada fuera de main. Instalar una cadena CRN11 compatible con H9 y probar corrección, consulta y recuperación con identidades sintéticas. |
| Cronos: efectos de permisos en el saldo | Calculador y preparador configurables C3, CLI de #417. | Publicación y adopción durables del catálogo, fuente histórica de colectivo de Personal y lectura conjunta CRN12. |
| Dietas: comisión y circuito | Documento, rutas, recibos, devolución y reenvío; competencia por perfil fijo de #424. | Provisión nominal de los perfiles y unidad, auditoría común de intentos y recorrido completo. Conectar el lector no habilita el alta de asignaciones D7. |
| Dietas: propuestas e informes | Preparación local, comparación de propuestas #383, subtotales #384 y revisión de tarifas #385. | Instantánea económica confirmada para D9, autorización específica de informe/exportación y auditoría común. Una propuesta no es una liquidación. |

Los ejemplos de cálculo siguen siendo configurables y provisionales. Las
preguntas 100–108, 118 y 122 de [dudas.md](../../dudas.md) conservan las decisiones
pendientes. La respuesta informática a la 122 usa perfiles fijos; el perfil
no crea la competencia administrativa ni permite aprobar una solicitud propia.

En Cronos, el perfil central ya autoriza el paso de resolución; la tabla
`permiso_resolutor` añade la restricción de qué empleados puede resolver cada
persona. No exige un acto de Personal y no se sustituye por un lector nuevo
de perfiles. Para usar perfiles fijos, K debe provisionar los contratos y
ámbitos existentes y conservar esa restricción y la separación de funciones.
La frontera de Cronos aún usa su registrador propio de denegaciones: falta
conectarla al puerto común de intentos de L con contexto nominal acreditado.

## Orden de trabajo tras H9

Esta es una cola de preparación, **no una lista para ejecutar**. Los números
pertenecen a módulos distintos; no bastan para establecer el orden causal.

La resolución de permisos C7 tiene una cadena distinta de CRN11:
**AD57 → Cronos9 → AD58 → Cronos10**. En la captura post-AD173 no están los
cuatro clasificadores de AD57 (`cronos_permisos_bandeja`,
`cronos_permiso_resolver`, `cronos_avisos_propio` y `cronos_aviso_archivar`).
El clon post-H9 tampoco devolvió la función `resolver_permiso_v1` al consultar
su catálogo. Por tanto, antes de preparar su instalación se debe comprobar
la existencia y ACL de las fachadas, reconciliar los consumidores históricos
con L y recibir los perfiles y el origen nominal de K. AD181 no cierra C7;
prepara únicamente la lectura histórica de Personal para C5.

1. **CRN11: reanclar el consumidor de autorización antes de Personal26.**
   [AD149](../../deploy/postgresql/autorizacion_atestada_v3/migraciones/000149_consumidor_vinculo_propio_crn11.up.sql)
   exige la definición `5dbdac03…` y el cuerpo `cdc8cb87…` del núcleo previo
   a AD151. H9 instaló la otra cadena. #470 corrigió el anclaje anterior, pero
   no hace compatible AD149 con H9. No ejecutar la lista
   `deploy/principal/lista_sql_codexb_crn11_20261003.txt` sobre la principal
   actual. B conserva Personal26; el reanclaje común requiere acuerdo con L/K,
   reserva si se crea una sucesora, ensayo y dos revisiones sobre la versión exacta.
2. **C3: fuente común antes de adopción y lectura.** CAT5 y AD147 conservados
   preparan el gobierno del catálogo. Su contrato exige publicación común,
   separación del publicador, autorización, auditoría y recibo. Después va
   CRN13, que recibe y adopta la publicación. AD146 aporta el consumo nominal
   de la lectura; CRN12 exige AD146 y CRN13. La adscripción histórica a colectivo
   la aporta Personal. Los anclajes históricos de AD146/147 no son instalables
   por concatenarlos después de H9.
3. **C8: enlace documental antes de justificar.** La cadena conservada
   Documentos12/AD148 precede al consumidor CRN14; este también exige AD146.
   La confirmación de Documentos debe vincular referencia, versión y huella
   del original. No se sustituye por un JSON ni por una validación local.
4. **Dietas D9: conservar el importe aprobado antes de emitir el informe.**
   El borrador Dietas12 describe una única transacción para instantánea
   económica, transición, autorización, competencia, auditoría y evento.
   Falta una migración instalable y su contrato nominal. El documento liquidado
   deberá leer esa instantánea, sin recalcular con la tarifa actual.
5. **Montaje y recorrido.** Solo después del ensayo compatible se preparan
   configuración privada, provisión por huella y CAS y conexiones nominales.
   Dirección instala. Se prueban permiso permitido, denegación, error auditado,
   recibo y recuperación tras reiniciar aplicación y PostgreSQL. El montaje
   no cambia la orden de mantener ocultos ambos módulos.

## Trabajo conservado que no se debe rehacer

Estas ramas locales tienen trabajo adicional a main y se mantienen intactas.
Los SHA identifican lo inspeccionado, sin atribuirles un ensayo sobre H9.

| Pieza | Rama / SHA | Estado |
| --- | --- | --- |
| Conexión CRN11 | `trabajo/codexe-crn11-conexion-20261002@3f6faad7f` | Composición conservada fuera de main; requiere cadena SQL compatible y revisión del montaje final. |
| Lectura conjunta CRN12 | `trabajo/codexe-crn12-final-20261002@727357207` | Borrador ejecutable y metadatos; requiere AD146/CRN13. |
| Adopción C3/CRN13 | `trabajo/codexe-crn13-final-20261002@1e0a020e4` | Candidata y ensayos estructurales anteriores; falta cadena común causal. |
| Justificación CRN14 | `trabajo/codexe-crn14-final-20261002@4a6bf683b` | Borrador; exige Documentos12/AD148 y AD146. |
| Consumidores AD146 | `trabajo/codexe-cronos-ad146-20261002@8a7ee0f9a` | Preimagen pendiente; no instalar. |
| Gobierno C3 | `trabajo/codexe-catalogos-c3-durable-20261002@65e91fcd5` | CAT5/AD147 conservados; no activados. |
| Enlace documental | `trabajo/codexe-documentos-enlace-cronos-20261002@f498cb16d` | Documentos12/AD148 conservados; no activados. |
| Liquidación D9 | `trabajo/codexe-dietas-d9-20261002@5b8c95f80` | Preparación de dominio y aplicación; Dietas12 sigue en Markdown. |

## Clon y límites de esta entrega

La receta post-H9 comunicada por Dirección reconstruye la copia fría H7 con
las nueve SQL de H7, las tres de H8, AD155, Bolsa77 e importación Convoca5,
y luego los 40 archivos de `h9kit/sql.list`, en ese orden. Esos pasos son
solo para reconstruir un clon: **no reaplicar en la principal ni ejecutar DOWN**.
El manifiesto de H9 se identifica por SHA256 `49271452…` en el canal.

En este corte se abrió un clon de la copia fría post-H9 publicada por K,
SHA256 `2c094d9d1d1de49f2fa2f4c4fce9b4020c7d0324a91772946134b54d29e562da`.
Una consulta de metadatos con `search_path=pg_catalog,pg_temp` midió la
definición del núcleo como
`00fdab71ff0477cbe3fb1dcabcde7377da857d20578b9be478339d031ac034ce`
y su cuerpo como
`1c4a33b316fe58454c76c207db1722504d69a2de21e5c4fb32c73a4cffc20fe4`.
AD149 y la función de Personal26 están ausentes. La incompatibilidad del
anclaje se confirma por resultado; no se intentó instalar ninguna migración.

El PostgreSQL de ensayo será uno por equipo, versión 18, memoria máxima
2 GB, datos en disco y `--rm`. Se conserva la copia fría antes de cada
cadena nueva; las comprobaciones se centran en esquema, roles, ACL, datos,
auditoría y resultado. Se retiran contenedor y datos al acabar.

Este inventario coteja fuentes, dependencias y metadatos del clon; no acredita un ensayo de migración nuevo,
una instalación pendiente ni un recorrido nominal. Las fuentes funcionales
son las fichas [Cronos C1–C12](ficha_cronos_2026-09-23.md) y
[Dietas D1–D9](ficha_dietas_2026-09-23.md), el
[inventario de Dietas](inventario_dietas_2026-10-01.md) y su
[preparación de liquidación](dietas_liquidacion_2026-10-01.md).
