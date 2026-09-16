# Instrucciones de desatasco — 3 de septiembre de 2026

Emitidas por dirección (Alberto) tras revisión externa completa del proyecto.
Este fichero PREVALECE sobre cualquier regla anterior de AGENTS.md, tableros o
documentos de decisión que lo contradiga. Léelo entero antes de tocar nada.

## Diagnóstico que motiva estas instrucciones

El código existente es sano (compila, `go vet` limpio, diseño fiel al Word de
RRHH), pero no hay producto: `NuevasRutas` en
`internal/app/composicion/interna/contrataciontemporal/rutas.go` construye las
5 rutas HTTP del módulo y ningún servidor la llama. Las métricas llevan más de
un mes congeladas (Bolsa 1/14, Contratación 24/46) mientras agosto se gastó en
un runner de pruebas (O3a) con cinco implementaciones rechazadas y 148 commits
`docs` frente a 21 `feat`. El problema no es calidad: es que nunca se ensambla.

## Prioridad única hasta nuevo aviso

Montar el esqueleto andante de Contratación temporal. En orden:

1. Cerrar O2-07: composición interna real. Registrar las 5 rutas ya
   construidas y probadas (alta de solicitud, propuesta/decisión/rectificación
   de cobertura, resultado) en el servidor interno, con identidad de
   desarrollo. No inventar infraestructura nueva: ensamblar la que existe.
2. Conectar el formulario web real de alta (O2-09, ya con GO) a esa API,
   sustituyendo el adaptador DEMO solo en ese recorrido.
3. Demostrar el recorrido navegador → API → autorización → PostgreSQL →
   recibo para: crear solicitud → análisis RRHH → decisión de cobertura →
   asignación a unidad. Datos sintéticos. Reinicio incluido.
4. Entregar a Alberto una guía de 1 página para recorrerlo a mano.

Nada más entra en curso hasta que el punto 3 funcione. Si una pieza falta de
verdad, se construye la versión mínima que permita el recorrido, no la
versión definitiva.

## Moratorias (efectivas ya)

- PROHIBIDO seguir con O3a, el runner endurecido, pidfd, leases y su contrato.
  Para CI basta `go test ./...` con el arnés H0 existente. Se retomará, si
  acaso, cuando el esqueleto ande.
- PROHIBIDO crear nuevos documentos de decisión, enmienda, relevo o contrato
  en `docs/portal_vec/` salvo un único fichero de seguimiento del esqueleto.
- PROHIBIDO añadir subtareas nuevas a O4-05/F0/C2.x. El árbol de tareas se
  congela; solo se cierran tareas existentes que el esqueleto necesite.
- PROHIBIDO reescribir desde cero material con NO-GO por hallazgos P2. Un P2
  se corrige con un parche acotado, no con otra "proyección".

## Listón de revisión por zonas (sustituye al doble GO universal)

- Doble revisión independiente SOLO para: criptografía, identidad/autorización,
  migraciones y funciones SQL, y fronteras de datos personales.
- Todo lo demás (pantallas, proyecciones, composición, adaptadores en
  memoria, docs): una revisión simple o autorrevisión con pruebas verdes.
- Pruebas de mutación y auditoría adversarial: solo en las zonas del primer
  punto, y solo si el cambio las toca.

## Métrica que se reporta a partir de ahora

"Pasos del flujo de RRHH recorribles de extremo a extremo por un humano":
hoy 0 de 8 (solicitud, análisis, bolsa, fiscalización, candidato,
nombramiento, incorporación, seguimiento). Abandonar el 24/46 como titular;
puede mantenerse como anexo. Cada parte de estado dice qué paso nuevo es
recorrible y qué falta para el siguiente.

## Correcciones funcionales pendientes (baratas, hacerlas al pasar)

1. Añadir la «Diligencia» al catálogo de documentos de formalización
   (RRHH pide 6 documentos; hoy hay 5; "Diligencia" no existe en el código).
2. Ofrecer las 5 modalidades en el formulario de análisis
   («Acumulación de tareas» y «Relevo» están en el catálogo de
   `datos-presentacion.js` pero no en las opciones del formulario).
3. Registrar en una nota breve la decisión sobre la ambigüedad del Word
   (8 fases con Incorporación/Seguimiento frente a "PASO 7 Integración
   Ginpix") para validarla con RRHH. No cambiar el flujo sin esa validación.

## Defectos conocidos que no deben perderse

- CT-LITE-O7-05C (escritura GINPIX, remoto): P1 carrera replay/cancelación
  tras `Link`; P2 carrera Lstat/OpenRoot con symlink. Corregir solo esos dos.
- O2-06 (confirmación): replay tras reinicio genera candidatura duplicada.
- CT-LITE-O6-02 (selección/llamamiento, remoto): tres archivos sin commit
  preservados en su worktree. NO borrarlos, NO reconstruirlos a ciegas.

## Reglas que siguen vigentes

Castellano coherente, i18n, arquitectura hexagonal, denegación por defecto,
cero cookies y cero almacenamiento web, puertos intercambiables, trazabilidad,
sin datos reales hasta autorización expresa, y sin atribución de IA en los
commits (autoría de aavidad únicamente). No tocar el Word de RRHH. No borrar
worktrees ajenos. Higiene: eliminar `.ssl-key.log` local (está en .gitignore,
no se sube, pero no debe existir).

## Criterio general

Ante la duda entre "más evidencia" y "más producto ensamblado", elige
producto. La evidencia se le añade a lo que funciona; lo que no se puede
recorrer no existe para RRHH.


## Recordatorio de especificaciones — instrucción del operador, 12 de septiembre de 2026

Las especificaciones originales siguen siendo obligatorias durante toda la
implementación. Hexagonalidad, i18n, roles y permisos, auditoría, seguridad,
protección de datos y restricciones de almacenamiento no son ampliaciones
opcionales ni requisitos que puedan omitirse para cerrar Contratación.

El director local y el remoto deben recordarlas en cada asignación de tarea,
cambio de alcance y revisión previa a integrar. En sesiones prolongadas,
repasar las instrucciones de los agentes activos al menos cada 30 minutos.
Cada encargo debe indicar: responsabilidad y archivos, fuentes concretas de
los requisitos aplicables, restricciones que el cambio debe conservar y
comprobación necesaria para aceptarlo. Los subagentes deben transmitir esas
mismas condiciones si delegan. No basta enumerar principios sin explicar
cómo afectan a la tarea concreta.

Antes de confirmar una entrega, comprobar tanto su funcionamiento como su
conformidad con las especificaciones. Una prueba verde no sustituye esa
comprobación. Registrar discrepancias y corregirlas en el corte; no declarar
terminada una funcionalidad que las incumpla. El sistema completo de roles y
la gestión completa de auditoría permanecen pendientes de producto, sin
rebajar los controles necesarios en cada operación ya implementada.


## Consenso y orden de correcciones — 15 de septiembre de 2026

Resultado del estudio transversal entre el agente programador y Claude (revisor),
cerrado bilateralmente en `comunicacion.md` (consenso final y tabla de 18 puntos)
con el informe de evidencias en https://claude.ai/artifact/Dcua14w8CjJ4RhmMEAvq9T.
Alberto ordena documentarlo para que los agentes sigan trabajando conforme a él.
Prevalece sobre las secciones anteriores de este fichero donde las precise; no
sustituye ESPECIFICACIONES_AGENTES.md.

### Diagnóstico aceptado por ambas partes

El mecanismo ha crecido más que el producto: 312.000 líneas de Go de producción,
493 funciones SQL, 167 políticas RLS y 71 roles para 52 expedientes sintéticos;
ninguna composición de producción; el cuadro de RRHH responde 502 cuando hay más
resultados que el límite de página y no deja log; y lo visible se desvía del Word
(raíl de fases, coste, comprobaciones de bolsa, observaciones, número visible,
plantillas, correo, adjudicación). La calidad de base es buena; el problema es de
proporción y de ensamblaje.

### Reglas vigentes desde hoy

1. Un caso de uso coordinador por operación: autorización vigente, carga, mutación
   de dominio, persistencia y fila de auditoría en una transacción, recibo. `ctx`
   solo a E/S y trabajo largo cancelable; nada de comprobaciones entre constructores.
2. Ningún adaptador nuevo sin consumidor real (servidor, CLI o trabajador) y un
   recorrido que lo use en el mismo corte. Los pendientes se preservan en rama,
   identificados, fuera del árbol activo.
3. Todo 5xx deja una línea de log en frontera con `correlacion_ref`, ruta,
   operación y código de causa. Nunca payloads ni credenciales. La respuesta al
   cliente sigue siendo mínima.
4. `ports/` solo interfaces y DTO. Validación en dominio; proyecciones como modelos
   de lectura en `application`; serialización en adaptadores.
5. Sin autohuellas en configuración. Huellas solo para versiones de reglas o
   documentos que participan en evidencia, calculadas por el sistema. Clones solo
   en fronteras mutables compartidas.
6. Pruebas por comportamiento y riesgo (positivo, denegación, entrada inválida y,
   si aplica, idempotencia y paginación). Adversariales y de mutación solo en
   cripto, identidad, SQL y datos personales. Tests por encima de producción
   disparan revisión, no prohibición.
7. Los topes de líneas son objetivos, no condiciones; mover código de paquete no
   cuenta. Toda retirada comprueba antes consumidores e invariantes. Ninguna
   eliminación por número de líneas, nombres o antigüedad.
8. Producto primero: no se abre mecanismo nuevo mientras haya un punto del orden
   de abajo sin cerrar.

### Orden de correcciones y criterio de cierre

1. **Cuadro de RRHH.** Localizar la causa con log en frontera (hipótesis fuerte:
   representación SHA del cursor en `adapters/postgres/consulta_rrhh_postgresql_salida.go:83-108`
   o `ports/proyecciones_rrhh_resultados.go:244`; contraste prioritario con CT89,
   `3e189e10`) y aplicar la corrección pertinente; el puente de 64 continuidades se
   evalúa aparte. *Cierre:* con más de 100 expedientes la bandeja abre y pagina;
   toda página con `hay_mas = true` devuelve 200 con su siguiente.
2. **Log de 5xx** (regla 3). *Cierre:* ningún 5xx sin línea correlacionable.
3. **Lo que RRHH ve.** Raíl de 8 fases con los 5 estados del Word derivados de hitos
   reales (tabla de mapeo, no motor de reglas); número visible estable; `observaciones`
   de extremo a extremo; nombres de catálogo en cuadro, detalle y documentos; coste
   desde fuente o ausencia explícita, nunca 40.000 € fijos. *Cierre:* recorrido en
   navegador sobre un expediente del servidor.
4. **Composición común** y retirada de la tercera vía `*_desarrollo.go`, empezando
   por un ensamblaje CT compartido; en el mismo movimiento `ports/` a interfaces y
   errores con causa. *Cierre:* `VEC_EXECUTION_PROFILE=produccion` arranca con
   adaptadores reales inyectados y falla solo por dependencias ausentes.
5. **Matriz identidad técnica → privilegios → consumidor vivo** para los 71 roles y
   las 11 conexiones (10 `VEC_CT_*`, 1 Bolsa; «gobierno» pasa a instalación). Pools
   por (módulo desplegable × exterior/interior) con excepciones justificadas en la
   matriz. Huellas y clones según regla 5. *Cierre:* ninguna fila sin consumidor.
6. **Cliente web y documentación.** Transporte común con límites de bytes y tiempo y
   un esquema por contrato; fuera el contador de fragmentos. Raíz con README, ESTADO,
   especificaciones, estas instrucciones, dudas y una guía de recorrido de una página;
   histórico a `docs/historico/` sin duplicar; inventario de worktrees con dueño y
   estado; el Word de RRHH en `.gitignore`.
7. **Adaptadores.** SMTP compuesto contra buzón de prueba rotulado como tal; ficha
   GINPIX compuesta; `ginpixapi` y `seguimientoejercicio` compuestos o preservados
   fuera del árbol.
8. **Bloques B1–B6** (Bolsa, público y candidato, Personal/GINPIX, administración,
   roles y auditoría, despliegue) según los criterios de cierre de `comunicacion.md`.
   B4 recupera, corrige y revisa `dddf2699` antes de integrarlo (tiene `innerHTML`
   sin escapar). B6 exige artefacto inmutable con binario y web dentro de la imagen.

### Lo que no se programa hasta tener respuesta

- De RRHH (`dudas.md`, preguntas 13–14): parámetros de las reglas temporales
  (+5, +9 meses, tres años) y regla de adjudicación automática con sus plazos.
- De Alberto: resuelto el 16 de septiembre; ver «Decisiones de alcance» abajo.

### Parte de estado

Cada corte dice qué punto del orden cierra, con su criterio de cierre comprobado,
y qué punto sigue. La métrica «pasos del flujo de RRHH recorribles» se mantiene.

### Decisiones de alcance de Alberto — 16 de septiembre de 2026

Tomadas sobre el estudio de Bolsa, Cronos y Dietas
(https://claude.ai/artifact/5xmJ7snNSiQFpV9L18e5Ng; anexo en `comunicacion.md`).

1. **Bolsa sustituye a Convoca.** El proceso selectivo construido (convocatorias,
   solicitudes, autobaremación, baremación, cálculo de experiencia, reglas de
   baremo) está en alcance. Condición: un **método de importación desde Convoca**,
   al menos para la carga inicial de las bolsas existentes. Fuente de requisitos
   para la sustitución: los pliegos SE15/2020 (prescripciones técnicas y cláusulas
   administrativas) y los manuales de Convoca en `docs/convoca_dipgra/`, más
   `Peticion.pdf` para la gestión de bolsas. Requisitos numerados (B1–B14 gestión
   de bolsas, S1–S7 proceso selectivo), estado en VEC y cambio necesario:
   `docs/estudio_requisitos/ficha_adaptacion_bolsa_convoca_2026-09-16.md`.
   Cuando Bolsa se retome, el orden es el de esa ficha, empezando por B1. Orden que se deriva: importador
   invocable (CLI o servidor) → participaciones y situaciones con fecha de
   disponibilidad → llamamientos desde el orden real de la bolsa → cuadro de
   control → portal del candidato → correo por estado; el proceso selectivo se
   conecta después, cuando haya una convocatoria nueva que tramitar. No se
   programa nada de Bolsa hasta cerrar el orden de correcciones de Contratación.
2. **Cronos se reescribe.** Sustituye a la aplicación actual de control horario,
   en PHP 5 y obsoleta. La aplicación actual es la fuente de requisitos: antes de
   programar hay que inventariar sus pantallas, reglas, datos y fuentes de fichaje,
   y citar la normativa de jornada aplicable. El dominio existente se traduce al
   castellano o se rehace; no se amplía la pantalla de demostración.
3. **Dietas se crea de nuevo**, por la misma razón que Cronos, con tres puntos de
   vista obligatorios: el empleado que solicita y justifica, el jefe que autoriza
   y valida, y RRHH que liquida y controla. La aplicación actual y la normativa de
   indemnizaciones (RD 462/2002 y cuantías provinciales vigentes) son la fuente de
   requisitos. La cartografía interna (OSRM y teselas propias) se conserva.

4. **Identificación del candidato en el portal: DNIe o certificado digital.**
   Nunca DNI y clave. Cierra la duda que el equipo había resuelto por su cuenta en
   `docs/estudio_requisitos/peticion_rrhh_transcripcion_y_lectura.md:59` y no se
   pregunta a RRHH.

**Aparcados (Alberto, 16 de septiembre):** Bolsa, Cronos y Dietas quedan fuera
del trabajo en curso hasta cerrar el orden de correcciones de Contratación. El
estudio se conserva (anexo de `comunicacion.md` y
https://claude.ai/artifact/5xmJ7snNSiQFpV9L18e5Ng) y las decisiones anteriores
siguen vigentes para cuando se retomen. Cronos y Dietas necesitarán además su
ficha de requisitos aprobada. Ninguno de los tres se reporta como avance.
