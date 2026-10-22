# Estado de VEC al 20 de octubre de 2026

VEC está en la fase 1: Bolsa y Contratación temporal, con las bases comunes que
necesitan. Se trabaja con datos ficticios en copias locales creadas con
[`deploy/principal/copia_local.py`](deploy/principal/copia_local.py).
Cidonia está congelada; la siguiente instalación irá a un servidor nuevo.

Este corte contrasta `origin/main@3c643ce34`, el
[tablero](docs/producto/TABLERO_TAREAS.md), el
[caso de aceptación](docs/producto/FLUJO_APP.md#40-caso-de-aceptación-de-la-presentación)
y los ensayos citados abajo. El tablero conserva el estado de cada tarea.
Integrar código, activarlo en una copia y acreditar el recorrido completo son
resultados distintos. No hay un acta que cierre todavía la fase 1.

## Qué está integrado

| Área | Qué permite el código integrado | Comprobación y límite |
| --- | --- | --- |
| Copia local | Crear, arrancar, comprobar, reiniciar y borrar una instancia propia; preparar módulos, Administración, firma y Personal | [Receta de copia local](deploy/principal/LEEME_copia_local.md). Algunas etapas requieren material privado autorizado; disponer del guion no acredita por sí solo el recorrido. |
| Administración y Organización | Preparar Administración y la consulta de puestos históricos que necesita RRHH | La receta declara los actos de gobierno necesarios y los datos ficticios de preparación. Los datos sembrados no prueban una aprobación administrativa. |
| Bolsa | Inscripción y admisión, candidaturas y llamamientos; variante de aceptación y adjudicación de dos plazas | [Ensayo de dos plazas](deploy/principal/aceptacion_dos_plazas_f1_55_20261018.md): conserva orden y recibos tras reinicio. La continuación hasta formalización sigue pendiente. |
| Área del aspirante | Entrada externa propia a Mi Bolsa | [Ensayo del 20/10](docs/estudio_requisitos/cuenta_externa_f1_46_20261020.md): consulta antes y después del reinicio con una cuenta sin participaciones. No acredita sus respuestas ni una inscripción completa. |
| Firma de Contratación | Preparación del informe, firmas de técnica y Jefatura y paso a fiscalización; consulta de firmas anteriores | [Firma en copia](docs/estudio_requisitos/firma_copia_f1_32_20261019.md) y [consulta histórica](deploy/principal/firma_historica_ficha_f1_36c_20261016.md). Las firmas de desarrollo no tienen eficacia administrativa. |
| Personal y cese | Preparación del alta, incorporación, confirmación manual de GINPIX y cese; cierre de ocupación y recepción del cese por Bolsa | [Preparación de Personal](deploy/principal/LEEME_copia_local.md#12-personal-incorporación-alta-ginpix-manual-y-cese-f1-119). El [ensayo de cese sin vínculo](docs/estudio_requisitos/recorrido_f1_30_cese_sin_vinculo_20261020.md) no sustituye la vuelta a la bolsa del caso único. |
| Rendimiento | Medición HTTP de las seis pantallas principales | [Acta del 20/10](docs/producto/acta_rendimiento_lecturas_f1_34_20261016.md): el objetivo sigue abierto por errores y latencias con peticiones simultáneas. |

Las PR abiertas no se cuentan como integradas en este corte. En particular,
las mejoras en preparación de firma, publicación de perfiles, respuesta al
llamamiento y consulta de comunicaciones deben comprobarse tras su fusión.

## Qué debe demostrar el recorrido completo

La [guía del recorrido](GUIA_RECORRIDO_ALBERTO.md) explica cómo seguirlo en
pantalla. La tabla siguiente resume la aceptación pendiente; el orden exacto
y los resultados están en el apartado 4.0 del flujo.

| Tramo | Quién actúa | Qué debe quedar acreditado en la misma copia |
| --- | --- | --- |
| Petición y ratificación | Centro y ratificador, personas distintas | Petición con puesto de la RPT, ratificación y recibos |
| Alta y preparación | Técnica de RRHH | Número MOAD, análisis con crédito, cobertura, unidad e informe |
| Informe y fiscalización | Técnica, Jefatura e Intervención | Informe con sus dos firmas verificadas; fiscalización, incluido reparo y subsanación |
| Llamamiento y respuesta | RRHH y aspirante | Llamamiento desde la bolsa constituida, comunicación, aceptación y resolución de esa aceptación |
| Nombramiento y documentos | RRHH y cargos de firma | Propuesta, resolución y sus tres firmas en el orden del catálogo; PDF verificado y borradores PDF y Word |
| Incorporación y Personal | Centro y RRHH | Acreditación del centro, alta y ocupación en Personal, descarga y confirmación manual de GINPIX |
| Cese y cierre | RRHH y Bolsa | Cese, vuelta a la bolsa con la disponibilidad y el orden debidos, expediente cerrado |
| Recuperación | Cada perfil autorizado | Tras reiniciar aplicación y PostgreSQL: mismos recibos, fechas y versiones; ningún efecto duplicado |

La inscripción, los méritos y las alegaciones del aspirante se muestran aparte.
Las variantes de vacante y de varias plazas conservan su aceptación propia;
no sustituyen el caso singular ni se dan por terminadas con el llamamiento.

Los cuatro documentos que el caso permite descargar sin firma deben seguir
identificados como pendientes de firma. Una confirmación manual en VEC no
acredita una transmisión a GINPIX ni una entrega del correo corporativo.

## Hito y criterio de cierre

El hito es el **recorrido completo de un expediente del caso
4.0 con reinicio, antes del 6 de noviembre de 2026**. Si no se llega, se
recalcula la fecha de fin de la fase 1, estimada para la segunda quincena de
noviembre. El hito no
autoriza datos reales ni sustituye estas condiciones del
[plan](docs/producto/PLAN_IMPLEMENTACION.md#5-fases-y-criterios-de-cierre):

1. Caso completo del apartado 4.0 en Chrome, con sus variantes, perfiles
   separados, reparo y subsanación, documentos y recuperación tras reiniciar.
2. Funciones necesarias activadas, incluidos firma y auditoría de RRHH,
   con sus dependencias comprobadas.
3. Botones y enlaces útiles: sin rutas ausentes ni fallos de disponibilidad
   durante el recorrido. Cualquier impedimento debe decir qué falta.
4. Lecturas principales por debajo de 300 ms en el percentil 95; sesión,
   módulos y preferencias por debajo de 100 ms, medidos en la copia local.

La revisión de dirección, la integración y la aceptación funcional siguen
siendo necesarias. El número de tareas fusionadas no sustituye estas pruebas.

## Riesgos y decisiones pendientes

- El recorrido único y la vuelta a Bolsa tras el cese aún necesitan un acta
  conjunta. Los ensayos parciales no bastan para cerrar la fase.
- La firma requiere plan, cargos, certificados y verificador coherentes en la
  copia. La preparación reproducible desde Git continúa en trabajo.
- La medición bajo concurrencia detecta fallos y tiempos superiores al objetivo;
  deben corregirse y comprobarse sin retirar autorización ni auditoría.
- Las reglas que RRHH no ha confirmado se mantienen como valores provisionales
  configurables. Las preguntas abiertas están en [dudas.md](dudas.md); las
  integraciones técnicas, en la [petición a Informática](docs/estudio_requisitos/peticion_informatica_integraciones_2026-09-23.md).
- Correo corporativo, GINPIX y portafirmas no se presumen conectados por un
  recibo local. El uso con datos reales exige las condiciones de la fase de
  producción.

## Antecedentes

Los estados de septiembre y las instalaciones anteriores son antecedentes.
Se conservan en la historia de Git de este fichero y en los documentos
fechados de [histórico](docs/historico/). Las referencias a subidas y ramas antiguas se conservan como antecedentes,
sin convertirlas en instrucciones de trabajo actuales.
