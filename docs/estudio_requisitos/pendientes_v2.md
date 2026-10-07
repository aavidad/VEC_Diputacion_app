# Pendientes V2 del baremador

El código preparado incluye el cálculo exacto de experiencia y las familias básicas de formación, titulaciones y otros méritos. Usa reglas e instantáneas identificadas por sus huellas. El editor local compara borradores con los mismos servicios Go; Provisión añade una simulación propia de grado, trabajo desarrollado por nivel, antigüedad, permanencia, cursos y titulaciones, con tablas y topes editables. Son simulaciones con datos sintéticos, no una capacidad institucional instalada. La valoración oficial aún requiere lectura atestada de la versión gobernada, autorización, confirmación durable y auditoría; esa cadena es una dependencia del producto, no una mejora opcional de esta lista.

Las ampliaciones siguientes se harán por convocatoria y contra sus bases aprobadas:

- Ampliar las familias básicas con incompatibilidades, equivalencias y tratamiento de evidencias previstos en las bases. La familia V1 excluye el título marcado como requisito y bloquea duplicidades; no cubre bases que permitan otro tratamiento.
- Incorporar las políticas de jornada por horas, solapes favorables, mínimos de sección y tope global que el cálculo de experiencia V1 rechaza expresamente.
- Conectar el editor local al portal interno con su autoridad real, casos de conformidad aprobados y el circuito de revisión y publicación. Reutilizar el puerto de gobierno y su control de versión; la activación sigue deshabilitada.
- Completar la valoración administrativa y la adjudicación global de Provisión con preferencias y desempates publicados. Su simulador conserva reglas propias; no reutiliza resultados de Bolsa.
- Ampliar la conversión por tramos a meses civiles o años sólo cuando las bases definan cómo acumular periodos y restos. Provisión V1 la rechaza; permite esos métodos por periodo con descarte explícito. No convertir meses en treinta días ni unir servicios implícitamente.
- Reunir experiencia y los demás méritos en una valoración completa por convocatoria. Los modos actuales conservan contratos separados y muestran sus desgloses y bloqueos, sin puntuar definitivamente.

Los coeficientes y fechas de los ejemplos sintéticos no aprueban bases institucionales. No se importan datos del prototipo local. Cada nueva familia necesita ejemplos aprobados para su convocatoria.

## Contraste que guía el corte

La [Bolsa Única de Andalucía](https://www.juntadeandalucia.es/boja/2019/240/18.html) fija coeficientes y topes en su convocatoria. La [Ventanilla Electrónica del SAS](https://www.sspa.juntadeandalucia.es/servicioandaluzdesalud/profesionales/ventanilla-electronica-de-profesionales/como-cumplimento-una-solicitud-de-autobaremo) separa cálculo, revisión y presentación. [SAP SuccessFactors](https://help.sap.com/docs/successfactors-performance-and-goals/implementing-and-managing-performance-management/rating-calculation) y [Odoo](https://www.odoo.com/documentation/17.0/applications/hr/appraisals.html) muestran criterios y escalas configurables por proceso. Se adopta la separación entre regla, cálculo y decisión; ninguno de esos productos aporta una fórmula aplicable por defecto a VEC.

## Original de la primera firma local de Contratación

El verificador acredita que la firma cubre el original aportado por el cliente.
Falta cotejarlo en backend con el borrador exacto de VEC por expediente, versión,
tipo documental y versión de plantilla antes de verificar y custodiar. Resolver
o conservar ese original por su autoridad; regenerar desde el catálogo vigente
no basta. La firma local mantiene `firma_eficaz=false` y no sustituye al circuito
corporativo. Los certificados distintos de una misma persona necesitan una
relación nominal aprobada; el canal de desarrollo exige el mismo certificado
para autenticarse y firmar.

## Cofirma PAdES de un mismo PDF

El diseño aprobado conserva el visto bueno de Dirección o Jefatura y la firma
de la Diputada como dos PDF distintos sobre un mismo original, que debe quedar
custodiado antes de firmar. Cada PDF tendrá un firmante verificado, su recibo
y su referencia de custodia. No se presentará como un PDF con dos firmas.

Si más adelante se necesita una cofirma incremental en un único PDF, GrxFirma
y el puerto de verificación deberán identificar el certificado de cada firma
con su revisión y `ByteRange`, comprobar que las firmas anteriores siguen
válidas y rechazar cambios no permitidos. El dictamen actual solo acredita un
firmante y rechaza la identidad ambigua en un PDF con varios. Este cambio queda
para V2; no condiciona el recorrido de dos evidencias separadas aprobado por
dirección el 2 de octubre de 2026.

## Verificación integrada del fin por causa

El kit local de recorridos mTLS está fijado a un plan SQL anterior. Para probar la rama de fin por causa en un clon completo hay que ampliar su lista de migraciones a CT165, CT166, CT167 y Bolsa B74, y proyectar los catálogos de reglas CT v3 y plantillas v2 con sus huellas. La adaptación debe conservar las comprobaciones de fuente, perfil, versión y preimagen; no se acepta saltarlas con variables sueltas. Después se acredita un POST real desde el navegador hasta recibo, replay y reinicio, antes del despliegue en cidonia.

Una reserva de alta que todavía no se ha confirmado carece de expediente histórico del que recuperar la política de fin. Si RRHH cambia c12 durante esa ventana, hará falta conservar la instantánea junto a la reserva para permitir su reintento sin reinterpretar la regla. Las operaciones ya confirmadas recuperan su política original mediante CT167.

## Selección: omitidas que reclaman en plazo

Hoy una solicitud omitida en la provisional entra con una nueva revisión de esa provisional, que abre su propio plazo de subsanación. Cuando la persona reclamó la omisión dentro del plazo original y cumple los requisitos, la práctica habitual es incluirla directamente en la definitiva como reclamación estimada. Habría que permitir ese camino en la definitiva, con su antecedente S4, sin pasar por otra provisional.

## Administración: anular un alta de perfil programada antes de que empiece

El lote ordinario (AUT44) asigna perfiles con inicio inmediato o programado y los retira cuando están vigentes. Un alta programada todavía no vigente no se puede retirar: la revocación de CA20/CA35 exige un contexto vigente. Para anularla antes de que empiece hará falta una operación propia en Contexto Actor que cierre el vínculo pendiente sin pasar por el bloqueo de contexto vigente, con su recibo y auditoría. Mientras tanto, la persona administradora tiene que esperar al inicio y retirarlo entonces.

## Administración: procedencias de actos fuera de otras fuentes

CA35 registra la procedencia de cada lote también en `procedencia_acto_admin_v1` y sólo la admite para vínculos del lote. Las funciones de denominación (CA32) y titularidad (CA33) aún aceptan cualquier procedencia maestra existente; en su próxima reconstrucción deben excluir las de esa tabla.

## Firmas: unidad de RRHH en la consulta R5 y en la vía externa

Desde CT186, la consulta y la recuperación R5 V2 llevan la unidad de la asignación de quien consulta y la ligan a la unidad del paso del plan publicado (opción A, aprobada el 06/10). Eso cubre al firmante y su consulta previa. Una persona de RRHH con unidad que consulte las firmas de un expediente fuera de un paso queda denegada. Si RRHH lo necesita, se hará en su propio corte con la unidad del expediente (opción B), comprobada con el predicado de CT174 (`leer_revalidar_relacion_unidad_expediente_ct_v1`).

La vía externa (RRHH registra una firma hecha en el portafirmas) sigue sólo con organización: AD206, AD209 y AD210 deniegan una unidad en esa vía, así que un operador de RRHH con unidad no puede registrar (decisión de dirección del 06/10, opción C). Las dos cosas dependen de cómo asigna RRHH unidades a su personal; está preguntado en `dudas.md` (148).

La recuperación R5 V2 con unidad se comprueba contra el plan publicado en ese momento, no contra el que había cuando se firmó. Si el plan se republica sin ese paso o se retira, quien tiene unidad deja de poder recuperar el recibo de una firma anterior (falla cerrado). Si hace falta recuperar recibos históricos así, la recuperación debe ligarse a la versión del plan que consta en la firma original.
