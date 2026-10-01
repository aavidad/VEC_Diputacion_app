# Inventario de Dietas — 1 de octubre de 2026

Base de la primera inspección: `origin/main@460e120c2ec9c2d4953d1e98bdba011403fe17e2`.
Retoma contrastada con `origin/main@0a62a3ea68e58fbf890885a2f59cc80343107e18`.
La fuente funcional es la [ficha D1–D9](ficha_dietas_2026-09-23.md). Este
inventario distingue el código disponible de su autoridad y de su comprobación.
La presencia de una migración en Git no acredita su instalación, ni una prueba
aislada demuestra el recorrido en navegador y PostgreSQL tras reiniciar.

| Requisito | Código en la base examinada | Autoridad y límite | Validación acreditada por este inventario |
| --- | --- | --- | --- |
| D1. Identidad y papeles | La composición de Dietas reutiliza identidad registrada, relación propia de Personal y autorización V3. Hay acciones propias para borradores, rutas y etapas del circuito. | El certificado identifica; cada operación necesita concesión vigente y, en el circuito, competencia de unidad. No hay contraseñas propias de Dietas. | Contratos y pruebas de código presentes; no se comprobó aquí una identidad operativa de cada papel. |
| D2. Comisión de servicio | `internal/modules/dietas/{domain,application,adapters/httpinterno,adapters/postgres}` conserva cabecera, revisiones y documento propio. La API permite crear, listar, consultar, editar, borrar lógicamente y enviar con clave, versión y recibo. | Personal aporta la relación acreditada; Dietas guarda su comisión. Las migraciones `dietas_borradores` 000001, 000004 y 000006 incorporan sus revisiones. | Montaje y contratos de prueba presentes; instalación y recuperación tras reinicio no verificadas en este corte. |
| D3. Tramos y cuantías | `tramos_provisionales.go`, `preparar_comision.go` y `tarifas_provisionales.go` calculan con tarifa y regla versionadas. La migración Dietas 000006 publica `regla_devengo_provisional` y `consultar_regla_devengo_dietas_v1`; Go consume esa consulta. | La regla publicada tiene `liquidable:false`. Cuantías, horarios y aplicación normativa esperan confirmación de RRHH; el resultado es orientativo. | Código, SQL y pruebas focales presentes. No se ensayó aquí su instalación ni se aprobó la regla funcional. |
| D4. Kilometraje | OSRM interno, catálogo de puntos, varias rutas y ajustes motivados llegan al documento v2; el servidor recalcula antes de guardar y conserva la versión del grafo. | La geometría de OSRM no reconoce por sí sola kilómetros liquidables. La tarifa por km sigue provisional. | Adaptadores y pruebas presentes; no se recorrió la cartografía ni la persistencia en este corte. |
| D5. Otros medios y gastos | `otros_gastos.go`, formulario web y Dietas 000009 exigen tipo catalogado, fecha, importe y referencia y SHA-256 del justificante para las líneas nuevas. | El fichero queda en custodia separada; su referencia no concede acceso al contenido. El catálogo de tipos es provisional. | Código y pruebas presentes; no se verificó un justificante en custodia real. |
| D6. Circuito | `circuito_comision.go`, API, migraciones Dietas 000007, 000008, 000010 y 000011 y cliente web contienen envío, bandejas, decisión por etapa, devolución, corrección y reenvío con recibos e historia. | `fuenteCompetenciaCircuitoDietas` entrega «sin fuente» mientras falte el catálogo gobernado de validadores. Sin competencia acreditada no ofrece decisiones. Aprobación, liquidación, fiscalización y pago son hechos distintos. | Montaje y pruebas presentes; ninguna decisión de los cuatro papeles se acredita aquí de extremo a extremo. |
| D7. Asignación | Personal conserva relación y asignación; Dietas consume sus referencias. La consulta y el cambio de grupo están compuestos. Hay contrato y vistas para solicitar rectificación textual. | `catalogoValidadoresCompetentesAsignacionDietas` está vacío: el alta y la corrección completa permanecen cerradas. Los clientes web de rectificación no están inyectados en el portal. | La ruta de consulta existe en código; no se comprobó asignación vigente instalada para una persona. |
| D8. Bandejas y fechas | La API del circuito admite etapa, intervalo civil, límite y cursor; el cliente del circuito está importado e inyectado en la vista interna. | Las bandejas devuelven `sin_fuente` sin competencia. Dietas conserva URL directa, pero el portal no la ofrece en menú ni Inicio. | Montaje estático comprobado; no se acredita una bandeja con registros y papel real. |
| D9. Informe y PDF | Existe un servicio y renderizador PDF gobernado de **borrador** (`dietas.comision.borrador.v1`) con pruebas aisladas. | No equivale al informe/PDF del documento liquidado pedido por D9. No consta en esta inspección su montaje como descarga de Dietas. | Pruebas de componente, sin documento liquidado ni recorrido de descarga acreditado. |

## Continuación sin repetir trabajo

1. Conservar la rama limpia `trabajo/vec-web-dietas-consulta-20260924` en
   `.worktrees/retenido-vec-web-dietas-consulta-20260927`. Su cabeza
   `440dccc47b4fa6d56db5ebe323487660058fad6b` no es ancestro de esta base
   y no tiene PR propio. Sus tres últimos commits tratan la consulta tras un
   resultado incierto, el foco y la ayuda. Comparar cada cambio con la web
   actual antes de portar; no fusionar ni aplicar toda la rama a ciegas.
2. Cerrar con Personal la fuente versionada de validadores competentes, su
   vigencia y la separación de solicitante, administrativo, responsable, RRHH
   e Intervención. Después abrir únicamente las operaciones D7 y D6 que esa
   fuente permita y comprobarlas con autorización V3, PostgreSQL y navegador.
3. Conectar los clientes existentes de solicitud de rectificación D7c cuando
   su ruta y autorización estén compuestas. Mantener la confirmación cerrada
   hasta acreditar catálogo y competencia. El PDF de documento liquidado D9
   exige antes una liquidación válida y una plantilla gobernada propia.

RRHH debe resolver las cuantías, tarifa por km, tramos horarios y alcance de la
liquidación; también quién revisa y autoriza en cada centro, qué paso recibe un
reenvío y cómo se relaciona la liquidación con nómina. Estas decisiones y los
permisos fijos o dependencias que resulten se elevan por coordinación. Este
inventario no las da por aprobadas ni modifica `dudas.md` o los manuales.

## Contraste para los siguientes cortes

El contraste de arquitectura, de solo lectura, acordó reutilizar el motor de
tramos existente. Un ensayo de propuestas puede comprobar fechas, porcentajes
e importes sin crear una comisión ni publicar tarifas. Sus huellas deben
identificar el ensayo y no sustituir la huella del catálogo PostgreSQL.

Las fuentes públicas consultadas fueron:

- [RD 462/2002, artículos 10 y 12](https://www.boe.es/buscar/act.php?id=BOE-A-2002-10337):
  distingue condiciones temporales, límites y justificación de gastos.
- [Resolución de 2 de diciembre de 2005](https://www.boe.es/buscar/doc.php?id=BOE-A-2005-19988):
  referencia de las cuantías nacionales sembradas para el ensayo.
- [Orden HFP/793/2023](https://www.boe.es/buscar/doc.php?id=BOE-A-2023-16462):
  referencia del kilometraje; la tarifa de motocicleta no acredita su integración en el recorrido.
- [Presupuestos de Diputación de Granada](https://www.dipgra.es/diputacion/delegaciones/economia-y-atencion-al-alcalde/servicio-gestion-presupuestaria-y-contable/presupuesto-general-00001/presupuestos-por-ejercicio-y-organismo/):
  las Bases de ejecución de 2026, artículo 30, contienen reglas propias y excepciones.
- [Normas de dietas UGR 2026](https://gerencia.ugr.es/sites/webugr/gerencia/public/ficheros/Presupuestos%20UGR/2026/Normars%20liquidaci%C3%B3n%20y%20tramitaci%C3%B3n%20dietas%202026.pdf):
  comparación del circuito de autorización y justificación, sin trasladar sus plazos a Diputación.

El motor actual redondea los porcentajes hacia arriba al céntimo y el
kilometraje a la mitad hacia arriba. RRHH debe confirmar este criterio y el
tratamiento del alojamiento antes de habilitar importes liquidables. La
simulación conserva el carácter provisional y rechaza variantes no soportadas.

## Dependencias para aprobación y documento liquidado

El consenso de revisión exige una fuente de Base que acredite, por acto vigente,
la persona, unidad y etapa competentes, junto con versión y huella de catálogo.
Su consulta debe tener una fachada de revalidación dentro de la transacción de
Dietas. La asignación de un administrativo o responsable en D7 no sustituye esa
competencia; tampoco cubre RRHH o Intervención. Autorización debe publicar los
cuatro perfiles nominales antes del montaje, por huella y comparación de estado.

La decisión de liquidación actual copia el documento orientativo y produce un
recibo de transición. Antes de emitir D9 como documento liquidado falta una
instantánea económica propia con conceptos admitidos y rechazados, importes en
céntimos, justificación, tarifa, acto y recibo de aprobación. El PDF de borrador
actual conserva su finalidad.

El corte SQL se reserva cuando Base publique el contrato y dirección fije su
lugar en la cola del núcleo V3. No se preparan funciones ejecutables con firmas
de autoridad supuestas, ni se modifican las migraciones Dietas 000001–000011.

## Retoma de la tarde

La base publicada incluye la preparación local de liquidación y su informe
HTML, los informes sintéticos por persona, unidad y periodo, el catálogo de
tarifas de ejemplo y los guiones de navegador. Esas piezas ya están entregadas.

Esta entrega amplía la propuesta local para revisar otros medios y gastos
justificados D5. Antes, la preparación rechazaba esas líneas. La corrección
independiente de informes conserva los filtros confirmados al recargar; hasta
que se integre, la recarga aplica campos editados sin confirmar el filtro.

D6 y D9 efectivos siguen pendientes de Personal y Autorización. Personal
000014 acredita asignaciones de administrativo y responsable; falta la
competencia para decidir por etapa, con acto, versión, huella y vigencia,
revalidada dentro de la transacción de Dietas. AUT27 conserva su parada por
contrato pendiente. Se mantienen cerradas las decisiones y la exportación.

D7c tiene servicios, manejador y clientes de solicitud de rectificación, pero
falta su composición en el servidor y la inyección web. Ese cableado requiere
verificar sus concesiones y ACL; no se abre en este corte. La rama histórica
`440dccc47` permanece conservada hasta comparar sus tres mejoras con la vista
actual. Los manuales esperan la confirmación de dirección sobre la petición de
RRHH y el recorrido instalado.
