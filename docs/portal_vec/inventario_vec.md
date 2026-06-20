# Inventario funcional VEC/VEPA para portal empleado

## Alcance

Este inventario traduce patrones publicos de VEC SAS, bolsa de empleo y portal
de empleo publico a un portal empleado/candidato. No define integraciones
reales ni persistencia productiva: sirve como mapa funcional para evolucionar el
prototipo de bolsa con frontera hexagonal.

Terminologia usada:

- VEC SAS: ventanilla electronica del Servicio Andaluz de Salud para relacion
  con bolsa/contratacion profesional, meritos, solicitudes y listados.
- VEPA/portal empleo publico: se trata aqui como portal general de empleo
  publico o ventanilla del aspirante/empleado. El paquete no materializa una
  definicion verificable de VEPA, asi que no se ata a un proveedor concreto.
- Portal empleado: superficie comun para candidato externo, empleado temporal y
  personal interno que consulta/gestiona procesos selectivos o bolsas.

## Fuentes publicas de referencia

- Servicio Andaluz de Salud, area profesional/ofertas de empleo y bolsa unica:
  https://www.sspa.juntadeandalucia.es/servicioandaluzdesalud/profesionales/ofertas-de-empleo/bolsa-unica
- Acceso VEC SAS usado publicamente como ventanilla de profesionales:
  https://ws027.juntadeandalucia.es/profesionales/eat/services/acceso?entrada=VEC
- Portal general de la Junta de Andalucia para empleo publico y funcion
  publica:
  https://www.juntadeandalucia.es/organismos/justiciaadministracionlocalyfuncionpublica/areas/empleo-publico.html
- BOJA, publicacion oficial de convocatorias, acuerdos y resoluciones:
  https://www.juntadeandalucia.es/boja.html
- Sede electronica de la Junta de Andalucia, tramites y notificaciones:
  https://www.juntadeandalucia.es/servicios/sede.html

Nota de evidencia: el entorno local no resolvio DNS externo con `curl`; las
fuentes quedan como enlaces publicos de trabajo y deben revisarse en navegacion
humana antes de convertir este inventario en especificacion contractual.

## Distincion funcional VEC SAS vs VEPA/portal empleo publico

| Eje | VEC SAS | VEPA/portal empleo publico | Portal empleado objetivo |
| --- | --- | --- | --- |
| Dominio | Sanidad publica, bolsa unica, contratacion temporal y procesos SAS. | Empleo publico transversal: oposiciones, bolsas, concursos, llamamientos. | Bolsa local con candidato, meritos, baremo, expediente y listados. |
| Usuario principal | Profesional sanitario/no sanitario inscrito en bolsa. | Aspirante, empleado publico, interino o personal interno. | Candidato y gestor administrativo. |
| Operacion fuerte | Alta/actualizacion de solicitud y meritos, baremacion, listados, cortes y ofertas. | Consulta de convocatorias, inscripcion, seguimiento, notificaciones y expedientes. | Flujo completo de solicitud, meritos, baremo, alegacion y decision. |
| Riesgo | Datos sensibles, trazabilidad de meritos, criterios de corte y contratacion. | Fragmentacion por administracion, diversidad normativa y plazos. | Mantener reglas parametrizables por convocatoria sin mezclar dominio con HTTP. |
| Encaje hexagonal | Dominio de bolsa y baremo; adaptadores para identidad, notificacion y documentos. | Puertos para identidad, registro, notificacion, expediente y fuentes oficiales. | Nucleo neutral + puertos pequenos + adaptadores opt-in. |

## Inventario funcional

### 1. Acceso y cuenta

Funciones:

- Acceso con identidad fuerte: Cl@ve, certificado, usuario corporativo o
  credencial interna.
- Perfil de cuenta con datos personales, contacto, consentimiento y preferencias
  de notificacion.
- Delegacion/representacion si la sede lo permite.
- Control de rol: candidato, empleado, gestor, revisor y administrador.
- Registro de ultimo acceso, sesion activa y avisos de seguridad.

MVP:

- Puerto `Authenticator` ya existente o equivalente.
- Roles minimos `candidate` y `staff`.
- Datos de contacto editables y validados.

Fases posteriores:

- Integracion real Cl@ve/certificado/LDAP.
- Doble factor para gestores.
- Representacion y apoderamiento.

### 2. Inicio y bandeja

Funciones:

- Resumen de estado: solicitudes abiertas, meritos pendientes, avisos, plazos y
  ultimas resoluciones.
- Acciones principales: inscribirse, aportar merito, subsanar, alegar, aceptar o
  rechazar oferta.
- Filtros por convocatoria, categoria, provincia/centro y estado.
- Mensajes de sistema con gravedad: informativo, requerimiento, vencimiento y
  bloqueo.

MVP:

- Panel de candidato con expediente, baremo calculado y estado de procedimiento.
- Panel de gestor con demo de convocatoria/listado.

Fases posteriores:

- Bandeja multiconvocatoria.
- Calendario de plazos.
- Tareas asignadas a unidades administrativas.

### 3. Convocatorias, bolsas y categorias

Funciones:

- Catalogo de convocatorias y bolsas publicadas.
- Ficha de categoria con requisitos, titulacion, ambito, turno, cupo y reglas de
  baremo.
- Periodos de alta, actualizacion de meritos, alegaciones y resolucion.
- Versionado de bases y anexos aplicables.
- Consulta de estado: abierta, en baremacion, provisional, alegaciones,
  definitiva, activa o cerrada.

MVP:

- Entidad convocatoria simple con categorias y estado.
- Reglas de baremo cargadas por composicion, no hardcodeadas en handler.

Fases posteriores:

- Importacion de BOJA/anexos.
- Parametrizacion por categoria y convocatoria.
- Historico de versiones y firma de bases.

### 4. Inscripcion y solicitud

Funciones:

- Alta de solicitud por convocatoria/categoria.
- Declaracion responsable y aceptacion de bases.
- Seleccion de ambitos, disponibilidad, jornada, turno y preferencias.
- Validacion de requisitos minimos antes de presentar.
- Estados: borrador, presentada, en revision, subsanacion, validada, rechazada y
  desistida.

MVP:

- Flujo `Borrador -> Presentado -> Validado/Rechazado/Subsanacion`.
- Guardado de solicitud y candidato.

Fases posteriores:

- Desistimiento, renuncia temporal y suspension.
- Restricciones por incompatibilidad o falta de requisito.
- Presentacion por registro electronico.

### 5. Meritos y evidencias

Funciones:

- Alta de experiencia, formacion, titulacion, oposicion, idiomas, docencia,
  investigacion y otros meritos.
- Adjuntos justificativos con tipo documental, fecha, emisor y vigencia.
- Reutilizacion de meritos entre convocatorias con copia controlada.
- Estados por merito: borrador, presentado, validado, rechazado, subsanable.
- Motivo de rechazo/subsanacion visible para candidato.

MVP:

- Tipos ya presentes: experiencia, formacion y otros.
- Calculo determinista con topes por seccion.
- Estado de merito independiente del estado de solicitud.

Fases posteriores:

- OCR/validacion documental asistida.
- Importacion desde vida laboral, titulos o servicios prestados.
- Caducidad y verificacion externa de documentos.

### 6. Baremo, listados y transparencia

Funciones:

- Calculo de puntuacion por seccion y total.
- Desglose explicable: merito, regla, unidad, puntos brutos, tope y puntos
  computados.
- Listado provisional y definitivo.
- Ordenacion por puntuacion y desempates.
- Consulta individual de posicion, corte y motivo de exclusion.

MVP:

- Baremo con reglas deterministas y desempate basico.
- Endpoint/vista de expediente con puntuacion.
- Listado demo provisional/definitivo.

Fases posteriores:

- Simulador de impacto de nuevos meritos.
- Publicacion anonimizada/pseudonimizada.
- Historico de cortes, llamamientos y movimientos.

### 7. Alegaciones, subsanaciones y recursos

Funciones:

- Apertura de plazo de alegaciones contra listado provisional.
- Requerimiento de subsanacion con motivo, plazo y documentos esperados.
- Respuesta del candidato con texto, adjuntos y meritos afectados.
- Resolucion administrativa de alegacion/subsanacion.
- Trazabilidad de cada cambio sobre solicitud y baremo.

MVP:

- Estado `Subsanacion` y retorno a `Presentado`.
- Motivo textual de requerimiento/rechazo.

Fases posteriores:

- Expediente de alegacion separado.
- Plantillas de resolucion.
- Registro electronico y acuse de recibo.

### 8. Ofertas, llamamientos y disponibilidad

Funciones:

- Perfil de disponibilidad por zona, centro, categoria, jornada y duracion.
- Llamamiento/oferta con plazo de respuesta.
- Aceptacion, rechazo justificado, no respuesta y penalizacion si aplica.
- Historico de llamamientos y situacion en bolsa.
- Reincorporacion tras suspension o indisponibilidad.

MVP:

- No incluir salvo que la convocatoria demo lo necesite.

Fases posteriores:

- Motor de llamamiento con reglas por bolsa.
- Integracion con RRHH/contratacion.
- Notificacion multicanal y prueba de entrega.

### 9. Notificaciones y comunicaciones

Funciones:

- Avisos en bandeja, correo/SMS opcional y notificacion electronica formal.
- Preferencias de canal no vinculantes para actos obligatorios.
- Plantillas i18n con variables controladas.
- Acuses de lectura y vencimientos.

MVP:

- Mensajes internos en expediente.
- Catalogo i18n centralizado para textos de estado/error.

Fases posteriores:

- Integracion con Direccion Electronica Habilitada o sistema autonomico.
- Cola de comunicaciones y reintentos.
- Auditoria de entrega.

### 10. Gestion interna

Funciones:

- Busqueda de candidatos y solicitudes.
- Revision de meritos por lotes.
- Cambios de estado con motivo obligatorio.
- Publicacion de listados con control de version.
- Exportacion de expediente administrativo.
- Cuadro de mando: pendientes, plazos, errores y productividad.

MVP:

- Demo administrativa con convocatoria y listados.
- Repositorio en memoria para pruebas.

Fases posteriores:

- Workflow por unidad/rol.
- Firma de resoluciones.
- Exportacion interoperable y evidencias selladas.

### 11. Auditoria, seguridad y proteccion de datos

Funciones:

- Registro de quien hizo que, cuando y por que.
- Trazabilidad de cambios de datos personales, meritos, estados y baremo.
- Minimizacion de datos en listados publicos.
- Separacion entre datos internos y datos publicados.
- Retencion, bloqueo y borrado segun normativa aplicable.

MVP:

- Eventos de dominio para transiciones relevantes.
- Tests de transiciones invalidas.

Fases posteriores:

- Auditoria inmutable.
- Politicas de retencion.
- Pseudonimizacion configurable.

### 12. Accesibilidad, i18n y experiencia

Funciones:

- Textos en catalogo, sin literales dispersos en handlers.
- Estados y errores comprensibles.
- Accesibilidad WCAG: foco visible, teclado, contraste y etiquetas.
- Formularios largos por pasos con guardado parcial.
- Descarga de justificantes en formato accesible.

MVP:

- Mantener catalogo `internal/shared/i18n`.
- Etiquetas y errores consistentes en espanol.

Fases posteriores:

- Multidioma si aplica.
- Lectura facil para resoluciones principales.
- Pruebas de accesibilidad automatizadas.

## Prioridad propuesta

| Prioridad | Bloque | Motivo |
| --- | --- | --- |
| MVP 1 | Acceso/cuenta, candidato, solicitud, meritos, baremo y expediente | Permite flujo ciudadano verificable. |
| MVP 2 | Convocatoria demo, listado provisional/definitivo y gestor basico | Permite flujo administrativo y prueba de reglas. |
| MVP 3 | Subsanacion/alegacion basica y mensajes internos | Cubre ciclo minimo tras listado provisional. |
| Fase 2 | Publicacion anonima, disponibilidad y ofertas | Acerca el portal a bolsa operativa. |
| Fase 3 | Identidad real, registro, notificacion formal y auditoria externa | Requisito productivo, alto acoplamiento institucional. |
| Fase 4 | Importaciones externas, OCR, firma, interoperabilidad RRHH | Optimiza gestion, no debe bloquear MVP. |

## Frontera tecnica recomendada

- Nucleo: candidato, solicitud, merito, convocatoria, regla de baremo, listado,
  alegacion y oferta como entidades/reglas sin HTTP, DB, rutas locales ni
  proveedor.
- Puertos pequenos: identidad, repositorio, documentos, notificacion, registro,
  reloj/auditoria y publicacion de listados.
- Adaptadores opt-in: HTTP, memoria, identidad fake, notificacion fake,
  conectores externos futuros.
- Composicion canonica: configuracion en `config`/`cmd`, con defaults claros y
  nombres unicos.
- i18n: textos de usuario y errores visibles en catalogo centralizado.

## Riesgos y decisiones pendientes

- VEPA no queda definido por contexto materializado; antes de desarrollo se debe
  confirmar si es producto concreto, acronimo interno o nombre generico.
- Las fuentes publicas deben verificarse en entorno con DNS antes de congelar
  URLs en documentacion contractual.
- No abrir integraciones reales de identidad, registro, notificacion, firma o
  RRHH sin write-set especifico y credenciales/entornos autorizados.
- No mezclar reglas de baremo SAS con reglas de una diputacion concreta:
  parametrizar por convocatoria.
- Mantener listados publicos minimizados: la posicion puede ser visible sin
  exponer datos personales innecesarios.

