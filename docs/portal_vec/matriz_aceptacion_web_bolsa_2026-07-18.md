# Matriz de aceptación de la web de Bolsa

**Fecha de corte:** 18 de julio de 2026

**Objetivo:** presentar a RRHH una interfaz funcional completa sin confundir
una simulación con un acto administrativo y sin crear una segunda aplicación
que haya que desechar después.

## Regla de continuidad

La interfaz, sus contratos y sus renderizadores son la base definitiva. Cada
capacidad se conecta mediante un adaptador seleccionado únicamente en la raíz
de composición:

```text
pantalla definitiva -> contrato/caso de uso -> adaptador seleccionado
                                            |- presentación: memoria volátil
                                            `- producción: conector autorizado
```

La migración se hará capacidad a capacidad. Cambiar un adaptador no autoriza a
duplicar la pantalla, alterar el dominio ni introducir una ruta alternativa.
Si una dependencia productiva no está disponible, la operación queda
deshabilitada y explica el bloqueo. Producción nunca cae automáticamente al
adaptador de presentación.

## Superficies que se aceptarán

| Superficie | Usuario | Contenido mínimo de la presentación | Separación exigida |
| --- | --- | --- | --- |
| Consulta pública | Cualquier persona | Convocatorias, categorías, plazos, requisitos, documentación y ayuda | Sin identidad, cookies ni datos privados |
| Área personal | Aspirante o integrante de Bolsa | Perfil, méritos, documentos, solicitud, autobaremación, expediente, disponibilidad, llamamientos, subsanaciones, alegaciones, mensajes, certificados y ayuda | Solo información propia; el adaptador real falla cerrado sin identidad y autorización |
| Gestión interna | Personal autorizado de RRHH, jefatura, administración o auditoría | Convocatorias, admisión, reglas, revisión, listas, llamamientos, contratos, documentos, comunicaciones, importación, auditoría, roles y configuración | Listener, identidad, sesión y permisos internos separados; denegación por defecto |

## Sustitución incremental de adaptadores

| Capacidad visible | Presentación | Adaptador productivo previsto | Condición para sustituirlo |
| --- | --- | --- | --- |
| Identidad y sesión | Perfil sintético inequívoco | Certificado/Cl@ve para aspirante; certificado y Kerberos corporativo para personal interno | Integración de Sistemas, sesión protegida y autorización por finalidad y recurso |
| Lecturas y escrituras de expedientes | Copia en memoria, eliminada al recargar | PostgreSQL inicial; Oracle u otro conector futuro | Transacción, control de versión, idempotencia, cifrado, auditoría y recuperación probados |
| Documentos | Referencias y resultados DEMO | Almacén de objetos compatible S3, cuarentena y metadatos en PostgreSQL | Cifrado, antivirus, integridad, retención, autorización y descarga trazada |
| Firma y sello de tiempo | Recibo visible sin firma real | Autofirma/portafirmas y TSA cualificada mediante conectores | Verificación de firma, política, sello, evidencia y revocación probadas |
| Registro | Asiento DEMO | Registro electrónico/interoperabilidad administrativa | Recibo verificable, fecha oficial, idempotencia y reconciliación |
| Pago | Resultado DEMO sin movimiento económico | Pasarela autorizada | Orden, retorno firmado, conciliación, devolución y no duplicidad |
| Correo, Telegram y avisos | Mensaje DEMO no enviado | Conectores de comunicación configurables | Consentimiento/base jurídica, minimización, reintentos, acuse y auditoría |
| Generación de documentos | Descarga marcada DEMO | Adaptadores de PDF, PDF/A, ODT, DOCX, CSV, JSON, XML, TXT u otros | Plantilla versionada, huella, accesibilidad, firma cuando proceda y conservación |
| Importación Convoca | Lote sintético no autoritativo | Importador gobernado | Parser endurecido, procedencia, huella, validación, deduplicación y revisión humana |

## Garantías del artefacto de presentación

- Perfil y binario exclusivos; no se selecciona por omisión.
- Dos guardas literales y listener limitado a una dirección local explícita.
- Datos sintéticos con referencias `DEMO-` y correos reservados `.test`, sin
  documentos de identidad formalmente válidos.
- Sin cookies, almacenamiento de navegador, conectores externos ni volumen
  durable.
- Toda mutación es volátil, exige confirmación y muestra actor, instante,
  objetivo, resultado, referencia `DEMO-` y ausencia de efectos reales.
- Al recargar se restaura el escenario inicial.
- El artefacto productivo excluye físicamente el lanzador, los datos y los
  adaptadores de presentación.

## Criterios de aceptación funcional y visual

Una pantalla no se considera terminada porque exista HTML. Debe cumplir todos
estos puntos:

1. Se alcanza desde el menú y conserva la navegación lateral y la identidad
   institucional.
2. Sus acciones funcionan en presentación o aparecen deshabilitadas con una
   causa concreta; no hay botones muertos.
3. No muestra datos de otra persona en las superficies pública o privada.
4. La interfaz se maneja con teclado, conserva foco visible y regiones de
   estado, y no depende solo del color.
5. Se revisa a 1440, 1024 y 390 píxeles, además de ampliación al 200 %.
6. No introduce CSS de módulo que rompa el sistema de temas compartido.
7. Supera pruebas de contrato, privacidad, separación DEMO/producción y el
   recorrido automático correspondiente.

## Evidencia pendiente para la presentación

Esta sección se actualizará con resultados medidos, capturas y limitaciones
antes de entregar a RRHH. Una marca `DEMO` acredita que el recorrido puede
evaluarse visualmente; nunca acredita integración, validez administrativa ni
aptitud para producción.
