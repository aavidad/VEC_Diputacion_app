# Brechas para un producto profesional, seguro y trazable

Fecha de revisión: 14 de julio de 2026.

## 1. Conclusión ejecutiva

La visión funcional y la arquitectura hexagonal son una base válida, pero aún no existe un
producto apto para datos reales. La principal brecha no consiste en añadir más pantallas,
sino en convertir el conjunto en un sistema con gobierno, evidencia, seguridad y operación
continua demostrables.

No debe afirmarse que la aplicación cumple el ENS en categoría ALTA antes de que los
responsables competentes categoricen formalmente el sistema, se implanten las medidas,
se evalúen y se supere la auditoría correspondiente.

## 2. Bloqueantes del ejecutable actual

El código presente es un prototipo útil para explorar dominio y presentación. Antes de
introducir datos personales reales deben resolverse, como mínimo:

1. **Autenticación y autorización de demostración.** Existen identidades y permisos
   permisivos y rutas sin control efectivo uniforme. Producción exige denegación por
   defecto, proveedor real, autorización por objeto y eliminación completa de identidades
   de demostración.
2. **Persistencia no productiva.** Memoria y ficheros JSON deben quedar limitados a pruebas
   o demostraciones. PostgreSQL será el primer adaptador productivo, con migraciones,
   transacciones, copias y restauraciones probadas.
3. **Persona no unificada.** Bolsa, Personal y las vistas actuales mantienen
   representaciones diferentes. Se necesita una persona canónica y flujos auditados de
   deduplicación, vinculación y rectificación.
4. **Auditoría no probatoria.** La cadena actual basada únicamente en SHA-256 y memoria no
   impide la reconstrucción por quien controle el almacén. El cambio de negocio, su evento
   de auditoría y la bandeja de salida deben confirmarse en una misma transacción.
5. **Módulos y permisos estáticos.** Rutas, acciones y composición están codificadas. Hay que
   separar capacidad de software de configuración funcional y decidir el contrato de
   incorporación de módulos sin modificar el núcleo.
6. **Datos de demostración en el navegador.** No se conservarán documentos o datos
   personales en almacenamiento local como fuente de verdad. Desarrollo, demostración y
   producción tendrán artefactos y configuraciones inequívocos.
7. **Circuito documental simulado.** Falta implantar subida directa temporal, cuarentena,
   validación, antivirus o desarme, huella, custodia, firma/sello y promoción al almacén
   definitivo.
8. **Endurecimiento de red y aplicación.** Faltan controles productivos de TLS, cabeceras,
   CSRF, límites, sesiones, secretos, red, contenedores, correlación, disponibilidad y
   observabilidad.

## 3. Gobierno ENS y privacidad

El [Real Decreto 311/2022](https://www.boe.es/eli/es/rd/2022/05/03/311/con)
exige delimitar, categorizar, analizar riesgos, asignar responsabilidades, aplicar medidas,
vigilar y mejorar el sistema. Deben producirse y aprobarse:

- alcance del sistema y sus dependencias;
- inventario de información, servicios, activos y responsables;
- valoración de disponibilidad, autenticidad, integridad, confidencialidad y trazabilidad;
- categorización formal;
- análisis y tratamiento de riesgos;
- declaración de aplicabilidad;
- política y procedimientos de seguridad;
- responsables de información, servicio, seguridad y sistema correctamente segregados;
- auditorías ordinarias y extraordinarias, certificación cuando corresponda y publicación
  de la conformidad.

Para protección de datos, el [RGPD](https://eur-lex.europa.eu/eli/reg/2016/679/oj?locale=es),
la [LOPDGDD consolidada](https://www.boe.es/eli/es/lo/2018/12/05/3/con) y las
[guías de la AEPD para administraciones públicas](https://www.aepd.es/areas-de-actuacion/administraciones-publicas/guias-informes-y-documentos)
obligan a cerrar antes de producción:

- registro de actividades por finalidades concretas;
- base jurídica de cada tratamiento, sin utilizar consentimiento donde no corresponda;
- información por capas y gestión de derechos;
- minimización por pantalla, rol, exportación y publicación;
- conservación, bloqueo, archivo y eliminación por serie documental;
- evaluación de impacto y participación del DPD desde el diseño;
- contratos de encargado y control de subencargados y transferencias;
- procedimiento de brechas, incluida documentación y notificación en el plazo aplicable;
- tratamiento reforzado y compartimentado de salud, discapacidad, embargos, datos
  disciplinarios y demás categorías de especial riesgo.

## 4. Expediente, registro y documento válidos

Guardar un PDF en almacenamiento de objetos no constituye por sí solo un expediente
administrativo. El [ENI](https://www.boe.es/eli/es/rd/2010/01/08/4/con) y sus normas
técnicas requieren diseñar:

- integración con el registro electrónico competente;
- recibo con asiento, fecha y hora, documentos y huellas;
- documento electrónico con contenido, firma y metadatos;
- índice electrónico firmado del expediente;
- copias auténticas, digitalización y CSV;
- política de gestión documental y cuadro de clasificación;
- apertura, cierre, reapertura motivada, transferencia y expurgo;
- preservación de firmas y formatos;
- exportación interoperable y verificable;
- integración con archivo, notificación y sistemas corporativos cuando corresponda.

Cada solicitud, subsanación, alegación, llamamiento, aceptación, renuncia, sanción,
nombramiento y cese debe quedar como actuación versionada, no como modificación silenciosa
de una fila.

## 5. Identidad, autorización y privilegios

Se necesita una plataforma de identidad y acceso completa:

- identificador interno de persona;
- vinculación verificada de certificado, Cl@ve, Active Directory y futuras identidades;
- alta, cambio y baja de permisos ligada a puesto, unidad y vigencia;
- autenticación reforzada para personal y operaciones sensibles;
- recertificación periódica de accesos;
- cuentas privilegiadas separadas, elevación temporal y acceso de emergencia;
- autorización por acción, recurso, campo, relación, finalidad, ámbito y nivel de garantía;
- separación entre tramitación, validación, resolución, firma, publicación, seguridad y
  auditoría;
- autoexclusión y conflicto de interés;
- representación con actor real, representado, poder, alcance y vigencia;
- una sesión y un único perfil activo, sin suma de privilegios.

El DNI no será clave técnica ni autenticador. Ocultar un botón en el navegador nunca se
considerará control de acceso.

## 6. Tres trazabilidades relacionadas pero distintas

1. **Expediente administrativo:** actuaciones, documentos, firmas, registros,
   notificaciones y resoluciones.
2. **Auditoría de negocio y seguridad:** quién accedió o actuó, con qué perfil, ámbito,
   autorización, finalidad, regla y resultado.
3. **Telemetría técnica:** métricas, trazas, errores y rendimiento con datos personales
   minimizados.

Para cada operación relevante se conservarán actor, perfil, representación, autenticación,
acción, objeto, versión, fecha y hora confiable, resultado, antes/después necesario, motivo,
regla, huellas, correlación y autorización. Se recomienda registro anexable e inmutable,
sellado periódico con claves protegidas, copia en otro dominio de seguridad y exportación
con manifiesto y cadena de custodia.

## 7. Desarrollo seguro y cadena de suministro

Las buenas prácticas deben convertirse en controles automáticos de entrega:

- modelo de amenazas por módulo e integración;
- entornos separados y prohibición de datos reales en desarrollo;
- revisión por otra persona y protección de ramas;
- pruebas unitarias, integración, contrato, autorización, concurrencia, migración y extremo
  a extremo;
- análisis estático, dinámico, dependencias, contenedores, infraestructura y secretos;
- pruebas de penetración y revisión independiente de componentes críticos;
- artefactos reproducibles, firmados e identificados por huella;
- inventario y lista de materiales de software por versión;
- política de vulnerabilidades y fin de vida;
- despliegue y retorno seguros;
- canal de comunicación responsable de vulnerabilidades;
- manuales, decisiones arquitectónicas, contratos API/eventos y diccionario de datos
  mantenidos junto con el código.

## 8. Operación, vigilancia y continuidad

La máxima trazabilidad exige capacidad operativa real:

- registros estructurados sin secretos ni exceso de datos personales;
- métricas, trazas y correlación de extremo a extremo;
- SIEM independiente y servicio SOC con cobertura acordada;
- alertas de consultas masivas, accesos fuera de ámbito, cambios de reglas, exportaciones y
  elevaciones de privilegio;
- gestión de vulnerabilidades, EDR, protección web y control de salida;
- respuesta a incidentes y preservación forense;
- coordinación con CCN-CERT y DPD;
- análisis de impacto de negocio;
- RTO y RPO aprobados por servicio;
- redundancia y recuperación en un segundo emplazamiento o modalidad equivalente;
- copias cifradas, inmutables y desconectadas;
- restauraciones y simulacros de pérdida de CPD y ransomware;
- protección ante sobrecarga y pruebas del último día de plazo;
- funcionamiento degradado que no perjudique al ciudadano por la caída de antivirus,
  firma, pago u otra integración.

## 9. Accesibilidad y atención

El [Real Decreto 1112/2018](https://www.boe.es/eli/es/rd/2018/09/07/1112/con)
y la norma armonizada aplicable deben traducirse en pruebas, no solo en una declaración:

- teclado, lector de pantalla, foco, contraste y ampliación;
- errores vinculados a campos y resumen de errores;
- texto e icono además del color;
- lenguaje claro y recorridos en lectura fácil;
- documentos y justificantes accesibles;
- móvil y redes lentas;
- pruebas con personas usuarias, incluidas personas con discapacidad;
- canal de accesibilidad y mecanismo de reclamación;
- asistencia humana sin suplantación silenciosa.

## 10. Integraciones y portabilidad

Cada dependencia será un puerto con capacidades, versión, salud, reintentos y auditoría:

- PostgreSQL y futuro Oracle;
- almacenamiento de objetos;
- antivirus, análisis y desarme de contenido;
- Cl@ve, certificados y Active Directory;
- AutoFirmaV2, validación, sello institucional y tiempo;
- registro, notificación y archivo;
- correo, Telegram, SMS u otros avisos;
- GINPIX u otra fuente de personal;
- nómina, contabilidad, control horario y firma;
- reloj oficial, calendarios y servicios de intermediación.

La sustitución de un conector no debe modificar el dominio. Sin embargo, incorporar y
certificar un conector nuevo sí es desarrollo de software, y trasladar datos entre motores o
proveedores requiere un plan de migración y conciliación.

## 11. Configuración funcional gobernada

Catálogos, formularios, procedimientos, estados, transiciones, baremos, calendarios,
plantillas, tipos de certificados, roles y políticas no quedarán fijados en el código cuando
sean variabilidad ordinaria de negocio.

Toda configuración relevante tendrá:

- borrador;
- validación y simulación;
- casos de prueba;
- revisión y doble aprobación según riesgo;
- fecha de efecto;
- publicación;
- historial y comparación;
- reversión;
- huella;
- relación con los expedientes que la utilizaron.

La configuración no sustituirá al código cuando aparezca una capacidad de negocio nueva o
una garantía que no exista. En ese caso se añadirá un módulo o una versión contractual.

## 12. API, CLI, escritorio, MCP e inteligencia artificial

Todos los canales usarán los mismos casos de uso y autorización. No habrá accesos directos
a base de datos o almacenamiento.

- API versionada y documentada.
- CLI administrativa solo desde puesto gestionado y con identidad individual.
- futuro escritorio como adaptador, no como lógica duplicada.
- MCP público separado, de solo lectura y limitado a una proyección publicada.
- MCP privado futuro con herramientas cerradas por caso de uso, confirmación y auditoría.
- defensa ante inyección de instrucciones y documentos hostiles.
- prohibición de enviar datos personales a modelos externos sin base, contrato y evaluación.
- bot identificado como tal, con respuesta, fuente oficial, versión y fecha.
- ninguna IA decidirá admisión, puntuación, exclusión o selección sin un análisis jurídico y
  técnico específico. El motor de baremo será determinista, versionado y explicable.

El [Reglamento europeo de IA](https://eur-lex.europa.eu/eli/reg/2024/1689/oj/spa)
será aplicable con carácter general desde el 2 de agosto de 2026; el bot debe diseñarse ya
con transparencia, control y registro.

## 13. Decisiones pendientes de la Diputación

1. Categoría ENS y valoración de cada dimensión.
2. Alcance exacto del sistema y de cada superficie de acceso.
3. Fuentes maestras: persona, puesto, RPT, contrato, nómina, jornada y documentos.
4. Integraciones disponibles de sede, registro, archivo, notificación, Cl@ve y firma.
5. RTO, RPO y segundo emplazamiento.
6. SOC propio, corporativo o contratado y cobertura horaria.
7. HSM, gestor de claves, sello institucional y sellado de tiempo.
8. Políticas de publicación y conservación aprobadas por RRHH, Archivo, Secretaría,
   asesoría jurídica y DPD.
9. Información que puede enviarse por correo, Telegram, SMS u otros canales.
10. Ámbito inicial exacto de API, CLI y MCP.
11. Presupuesto y responsables de certificación, auditoría independiente y operación.
12. Hoja de ruta de PostgreSQL y condiciones para desarrollar y certificar Oracle.

## 14. Puerta de producción

No deben utilizarse datos reales hasta disponer de:

- autenticación y autorización productivas;
- categorización, análisis de riesgos y declaración de aplicabilidad;
- registro de tratamientos y evaluación de impacto aprobados;
- modelo de amenazas;
- persistencia, migraciones, copias y restauración probadas;
- circuito documental, firma, registro y expediente verificables;
- auditoría inmutable y correlacionada;
- pruebas de seguridad, carga, accesibilidad y continuidad superadas;
- proveedores y contratos evaluados;
- procedimientos de incidentes y brechas ensayados;
- manuales operativos y formación;
- ausencia de incumplimientos críticos en auditoría independiente;
- aceptación conjunta de RRHH, Sistemas, Seguridad, Secretaría/Asesoría, Archivo y DPD.

## 15. Prioridad recomendada

### P0 — plataforma transversal

Persona, identidad, roles y ámbitos, expediente, documentos, almacenamiento, análisis,
firma, registro, auditoría, calendario, notificaciones, configuración, PostgreSQL,
observabilidad, accesibilidad, copias y continuidad mínima.

### P1 — Bolsa jurídicamente completa

Convocatoria versionada, perfil y méritos, solicitud, firma y registro, autobaremo,
admisión, validación, subsanación, listado provisional, alegaciones, listado definitivo,
constitución, primer llamamiento, aceptación o rechazo e incorporación.

### P2 — Portal del empleado y evolución

Autoservicio, responsable, certificados, Personal y después Cronos, Nóminas, Dietas,
formación, carrera, movilidad y restantes módulos, siempre reutilizando los servicios
transversales.
