# Guion de demostración para RRHH — septiembre de 2026

Guion para la presentación de VEC a RRHH en la principal de cidonia. Cada afirmación
distingue lo **comprobado en cidonia** (barrido en Chrome tras el despliegue del
23/09/2026, con 0 respuestas HTTP ≥400 y 0 errores JS en las vistas recorridas),
lo **pendiente de ensayo** y lo **que no existe**.
Todos los datos son sintéticos: personas, DNI enmascarados, correos `.test` y
expedientes. Nada de lo que se enseña firma, notifica ni produce efectos administrativos.

## Antes de la reunión (lista de control)

1. **Acceso.** El portal interno de RRHH está servido en
   `https://vec.cidonia.cloud/` con autenticación básica de Caddy configurada fuera
   de Git. Preparar las credenciales por el canal privado para la reunión; Cl@ve,
   certificado y DNIe aún no están habilitados por Sistemas.
2. **Emisión de llamamientos.** El 23/09 se verificaron una emisión B7 con respuesta
   `201` y el correo en Mailpit. No se repitió esta prueba tras los despliegues de la
   tarde: incluir el paso 4 y el buzón de pruebas en el ensayo general.
3. **Ensayo general completo** en cidonia el día anterior, siguiendo este guion de
   principio a fin, incluido un llamamiento que llegue al buzón de pruebas. Revisar
   también el menú, cuya comprobación P4 seguía abierta en el último barrido.
4. Contratación: los expedientes con número legible (`2026/CT-0000NN`) están en
   Solicitud, Asignación y Fiscalización. Los tres que llegan a **Nombramiento**, que son
   los que tienen documentos, tienen número-hash porque los creó una prueba automática;
   usar `2026/CT-4b2ba511…` y decirlo así.

## Orden de la demostración

### 1. Bolsa: la foto de conjunto (petición p. 1, puntos 6 y 7; p. 3)

- **Cuadro de mando** (menú: *Cuadro de mando para dirección*): 12 bolsas vigentes
  importadas de CONVOCA, 390 aspirantes, disponibles, trabajando y no disponibles por
  bolsa, paginado. *Comprobado.*
- **Estadísticas** (menú 7): tarjetas de bolsas, vigentes, sustituidas, personas y
  llamamientos; desglose por situación y por bolsa. Cada cifra abre la lista filtrada.
  *Vista comprobada tras el despliegue; comprobar el acceso desde el menú en P4.*

### 2. Bolsa: una bolsa por dentro (p. 1, puntos 1 y 4; p. 2)

- Pulsar **ADMINISTRATIVO** en el cuadro → **Candidatos** (41 aspirantes).
- Mostrar el **orden calculado** (Nº orden, orden del acta y razón si difieren) y el
  bloque **Criterios de orden**: puntuación descendente, lista rotatoria, reposición en la
  misma posición, rotulado *provisional, pendiente de RRHH (dudas 13–14)*. *Comprobado.*
- Los **siete estados** de la petición (disponible, no disponible, trabajando, pendiente de
  incorporación, renuncia, excluido, disponible desde) como contadores y filtro.
  *Comprobado.*
- **Pausar, reactivar y excluir** desde la ficha (B8): motivo, justificante (referencia y
  huella del documento, que no se sube a VEC) y persona validadora; la exclusión exige una
  segunda persona (regla provisional, duda 6). Recibo e historial de operaciones.
  *Comprobado en cidonia por la API (9/9) y en la ficha.* La última persona de «Encargado»
  ya figura excluida por la prueba.
- **Ficha de la persona**: datos de contacto cifrados (correo y dos teléfonos, B4) y
  histórico de contactos (B3). *Registro de contactos comprobado por la API; ficha
  pendiente de ensayo.*

### 3. Bolsa: nuevo llamamiento (p. 1, punto 2; p. 3, envío por estado)

- Desde la bolsa, **Nuevo llamamiento** → paso 1 (bolsa) → paso 2 (candidatos por el orden
  vigente; solo ocupan turno disponibles y disponibles desde fecha). *Comprobado.*
- **Envío por estado**: «Seleccionar todas las que cumplen el filtro»; si son más de 100,
  se envían las 100 primeras por orden y la pantalla lo dice. *Desplegado; pendiente de
  ensayo completo.*
- Paso 3: referencia, centro, modalidad, fecha, asunto y texto (comunes a todas las
  personas; la personalización por persona no existe). Canal: correo por el **relay de
  pruebas**, no el buzón corporativo. Plazo de respuesta *provisional (dudas 1–3)*.
- Paso 4: confirmar y emitir; recibo y paso al histórico. *Emisión `201` y correo en
  Mailpit verificados el 23/09 antes de los despliegues de la tarde; pendiente de
  repetir en el ensayo general.*

### 4. Contratación temporal (el procedimiento que remitió RRHH)

- **Bandeja de expedientes**: 142 expedientes sintéticos con centro, categoría, modalidad,
  estado, fase y plazo; filtros por número, estado y fase. *Comprobado tras el despliegue.*
- **Primeras fases**: abrir un expediente legible (`2026/CT-0000NN`, en Solicitud o
  Fiscalización) y enseñar su recorrido. *Bandeja y detalle comprobados tras el despliegue.*
- **Documentos**: filtrar la fase *Nombramiento*, abrir `2026/CT-4b2ba511…` →
  *Abrir expediente completo*. Ofrece seis documentos preparatorios, **cada uno en PDF y
  en Word**: informe, resolución, diligencia, toma de posesión, notificación y
  comunicación al centro. *Comprobado: el informe se descarga (PDF de una página) con
  los datos del expediente y el rótulo «borrador preparatorio, no firmado ni
  validado».* Sin firma ni valor administrativo.

### 5. Cierre: lo que falta y de quién depende

Decirlo sin rodeos al final (ver tabla).
El módulo Dietas permanece desactivado y queda fuera de esta demostración.

## Petición de RRHH frente a lo que hay

| Petición | Qué se enseña | Estado | Falta y de quién |
| --- | --- | --- | --- |
| 1. Bolsas y candidatos | Cuadro, lista ordenada, ficha, contactos cifrados, pausas y exclusiones con justificante | Hecho | — |
| 2. Llamamientos según reglamento | Asistente de 4 pasos por el orden vigente; emisión y correo de prueba verificados antes del último despliegue | Parcial | Reglamento de Granada: RRHH (dudas 13–14); repetir emisión en el ensayo general |
| 3. Contratos, ceses, reincorporaciones | Tramitación en Contratación temporal | Parcial | Ceses y reincorporaciones completos: Desarrollo |
| 4. Motor de reglas configurable | Política de orden versionada y rotulada | Parcial | Parámetros aprobados: RRHH |
| 5. Portal del candidato con acceso seguro | «Mi bolsa» no se muestra al candidato en cidonia | Pendiente | Completar P3 y acceso con DNIe/certificado (ya decidido): desarrollo propio |
| 6. Cuadro de mando | Cuadro B12 | Hecho | — |
| 7. Estadísticas | Pestaña de estadísticas | Hecho | — |
| 8. Documentos Word/PDF | Seis documentos preparatorios en PDF y Word desde el expediente | Parcial | Plantillas oficiales: RRHH; firma en portafirmas: integración |
| 9. Correo, SMS, mensajería | Correo por relay de pruebas | Parcial | Correo corporativo: Sistemas; SMS/WhatsApp: no iniciado |
| 10. Auditoría y trazabilidad | Recibos e históricos de cada operación | Hecho en lo conectado | IP/equipo según política de seguridad |
| Histórico (p. 2) | Estados, contactos, llamamientos | Parcial | Contratos y sanciones: Desarrollo |
| Reglas de 5 y 9 meses (p. 2) | Reposición provisional | Pendiente | Regla exacta: RRHH (duda 13) |
| Zona pública (p. 2) | Consulta pública B10 en instancia separada | Hecho, no publicado | Publicación y minimización: RRHH, Secretaría y DPD |
| Envío masivo por estado (p. 3) | Selección por filtro, hasta 100 | Hecho, sin ensayo | Personalización por persona: Desarrollo |
| Avisos de salto de orden y 3 años (p. 3) | Bloque «Avisos» en el cuadro de mando | Hecho | Tres años sale a cero hasta que haya histórico; art. 15.5 ET: RRHH (duda 13) |
| Plantillas y coste por categoría (p. 4) | — | Pendiente | Tabla de retribuciones y plantillas: RRHH; GINPIX: integración |

## Frases de cierre

- «Lo que se ve conectado usa autorización, persistencia y recibo reales; los datos son
  sintéticos.»
- «El correo sale por un relay de pruebas; no acredita entrega en el buzón corporativo.»
- «El orden y los plazos están rotulados como provisionales hasta que nos confirméis el
  reglamento.»
- «Identificarse con certificado no es firmar: la firma irá por el portafirmas.»
