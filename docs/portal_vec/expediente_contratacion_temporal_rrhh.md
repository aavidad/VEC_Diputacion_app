# Expediente de contratación temporal solicitado por RRHH

Estado: especificación de implementación inicial.

Fuente funcional: documento «Pantalla de procedimiento de gestión de
contratación y gestión de bolsas», recibido el 23 de julio de 2026.

## Decisión de arquitectura

El procedimiento no sustituye el módulo Bolsa. Se incorpora el módulo
`contrataciontemporal`, que coordina capacidades de otros módulos mediante
referencias opacas, puertos y eventos:

```text
solicitud del centro
→ análisis de RRHH y RC
→ decisión de vía de cobertura
→ asignación a unidad
→ informe jurídico
→ fiscalización
→ llamamiento
→ formalización y firma
→ incorporación
→ GINPIX
→ seguimiento y cierre
```

Bolsa conserva la autoridad sobre convocatorias, integrantes, posiciones,
disponibilidad, reglas y llamamientos. Personal conserva la autoridad sobre
relaciones jurídicas, ocupaciones e incorporaciones. El expediente no copia
sus agregados ni escribe directamente en sus tablas.

## Fases y pantallas

Las ocho fases superiores de las capturas son hitos administrativos. Las
diecisiete pantallas son tareas o vistas dentro de esos hitos. La definición de
flujo será un dato gobernado, versionado e inmutable tras publicarse.

Cada expediente inmoviliza:

- referencia, versión y huella de la definición de flujo;
- fase y estado operativo actuales;
- unidad y responsable de cada tarea;
- instante de entrada y salida;
- actuaciones, documentos, recibos y decisiones;
- referencias a agregados de otros módulos.

Los estados operativos técnicos son pendiente, en curso, esperando a otro
departamento, completado, incidencia y cancelado. Las fases, modalidades,
causas, vías de cobertura, unidades y documentos exigibles proceden de
catálogos gobernados, no de listas cerradas compiladas.

## Primer corte vertical

El primer corte implementa:

1. alta de la solicitud del centro;
2. análisis de RRHH y validación de la RC;
3. comprobaciones y decisión de la vía de cobertura;
4. asignación a una unidad gestora;
5. cronología de solo adición;
6. proyección para cuadro de mando y detalle.

Este corte termina únicamente cuando exista recorrido navegador → API interna
→ aplicación → PostgreSQL → recibo, con identidad y autorización efectivas.
Un adaptador en memoria o una pantalla con datos sintéticos no lo cierra.

## Campos mínimos

### Solicitud

- número visible y referencia opaca del expediente;
- centro solicitante y persona de contacto por referencia;
- categoría y grupo/subgrupo;
- motivo catalogado y detalle;
- fechas previstas de inicio y fin;
- declaración de existencia de RC;
- número, fecha, importe y documento de RC cuando se aporten;
- documentos complementarios y observaciones.

### Análisis de RRHH

- modalidad catalogada;
- categoría y grupo/subgrupo validados;
- causa y observaciones;
- fechas y jornada;
- validación de RC con fuente y recibo;
- coste previsto cuando exista fuente autorizada;
- actor, unidad, instante y recibo de la decisión.

### Vía de cobertura

- comprobaciones automáticas con resultado, fuente e instante;
- bolsa vigente, agotamiento, candidaturas disponibles, SAE y nueva
  convocatoria;
- vía elegida y motivo;
- referencias de Bolsa o procedimiento cuando correspondan.

### Asignación

- unidad gestora;
- responsable;
- fecha de entrada en bandeja;
- recibo de asignación y notificación.

## Intercambio entre módulos

Los cambios cruzados usan outbox/inbox e idempotencia. No se realizan
transacciones distribuidas ni lecturas directas de tablas ajenas.

```text
Contratación temporal --solicita propuesta--> Bolsa
Bolsa --publica propuesta/aceptación--> Contratación temporal
Contratación temporal --solicita alta--> Personal
Personal --confirma relación/ocupación--> Contratación temporal
Contratación temporal --prepara entrega--> conector GINPIX
```

La indisponibilidad de un conector deja la tarea en «esperando a otro
departamento/sistema» y conserva un reintento recuperable; nunca se interpreta
como éxito.

## Seguridad

- denegación predeterminada y concesiones exactas por expediente, fase, unidad,
  finalidad y acción;
- una sola identidad y un solo perfil activo por sesión;
- segregación entre solicitante, RRHH, unidad, jefatura, Intervención,
  formalización, firmantes y operación de GINPIX;
- datos personales minimizados en bandejas y cuadros de mando;
- registro de lectura, descarga, exportación y transición;
- documentos por referencia opaca en almacén cifrado;
- ninguna cabecera, campo de formulario o dato del navegador concede autoridad;
- cada efecto revalida autorización y versión dentro de la transacción durable.

El alta inicial aplica además idempotencia semántica. Una clave de petición
queda ligada mediante HMAC a organización, actor, perfil, flujo versionado y
contenido normalizado. Repetir exactamente la operación devuelve el mismo
recibo; reutilizar la clave con otros datos falla sin crear un segundo
expediente. El secreto HMAC se obtiene del gestor de secretos y no se almacena
con el material en claro.

La reserva previa solo adjudica referencias estables. No concede permisos ni
confirma el expediente. La persistencia debe consumir una autorización de
efecto vigente y escribir reserva confirmada, expediente, actuación inicial,
auditoría y outbox en la misma transacción.

## Contrato visual mínimo

La interfaz definitiva conserva el tema común y reproduce como mínimo:

- lateral azul marino con módulos y áreas de trabajo;
- barra superior con contexto, avisos, ayuda y perfil;
- indicadores compactos;
- tabla filtrable de expedientes;
- línea persistente de progreso;
- formularios por fase;
- paneles de documentos, comunicaciones, historial y auditoría;
- gris para pendiente, azul para en curso, verde para completado, rojo para
  incidencia y amarillo para espera externa.

Color, icono y texto comunicarán juntos el estado. Todas las acciones
administrativas separarán guardar borrador, validar, enviar, firmar, registrar
y confirmar.

## Conectores previstos

- fuente presupuestaria y validación de RC;
- cálculo de coste de personal;
- Bolsa y SAE;
- generación documental;
- portafirmas y validación;
- notificación y registro;
- almacén documental y antivirus;
- Personal/RPT;
- GINPIX por API o exportación estructurada;
- calendario hábil;
- auditoría y sellado de tiempo.

GINPIX se implementará como puerto. La salida automática y la ficha de carga
manual usarán el mismo modelo canónico y una versión explícita del mapeo.

## Condición de terminado

Una fase exige conjuntamente dominio, aplicación, puerto, adaptador durable,
API, autorización, auditoría, interfaz conectada, pruebas unitarias,
PostgreSQL, concurrencia, recuperación, seguridad, accesibilidad,
documentación y validación funcional de RRHH.
