# Comparativa y composición recomendada de portales públicos de RRHH

Fecha del análisis: 14 de julio de 2026.

Fuentes e integridad de las copias: [README.md](README.md).

Decisión de espacios y persistencia: [decision_espacios_y_persistencia.md](decision_espacios_y_persistencia.md).

## 1. Conclusión

Las referencias respaldan una solución con **persona, expediente, documentos y servicios
comunes**, pero con espacios de acceso realmente separados:

1. portal público y área personal del aspirante;
2. autoservicio del personal empleado;
3. espacio de responsables de unidad;
4. área interna de RRHH;
5. plano de administración funcional, técnica, seguridad y auditoría segregado del
   anterior.

No se propone duplicar bases de datos ni personas. Tampoco basta con una sola aplicación
que oculte menús según el rol. Cada superficie debe tener sesión, audiencia de credencial,
pasarela, proyección de datos, navegación, límites y políticas de autorización propios.

Un empleado que participe en una bolsa entrará como aspirante y tendrá exactamente los
permisos de un aspirante. Un técnico de RRHH que consulte su nómina entrará como empleado
y verá solo sus datos. Los privilegios no se acumulan.

## 2. Cobertura de las referencias

| Fuente | Aspirante / proceso | Méritos compartidos | Bolsa y llamamientos | Autoservicio empleado | Responsable | Área interna RRHH |
| --- | --- | --- | --- | --- | --- | --- |
| AGE IPS 2026 | Alta | Baja | No | No | No | Solo resultados visibles desde fuera |
| GVBORSES | Alta | Media | Media | No | No | Solo resultados visibles desde fuera |
| ULISES | Alta | Media | Media | No | No | No |
| Junta VEPA | Alta, incluido nombramiento | Media | Media | No | No | No |
| Junta CAP 2026 | Proceso interno propio | Media | No | Parcial | No | No |
| SACYL BAPE | Alta | Media | Alta | No | No | Solo resultados visibles desde fuera |
| Osakidetza | Alta | Alta | Alta | No | No | Solo resultados visibles desde fuera |
| Xunta FIDES | Alta | Muy alta | Media | Parcial | No | Solo estados del validador |
| SERMAS empleado | No | No | No | Alta | No | No |
| SERMAS responsable | No | No | No | Parcial | Alta | No |
| CONVOCA Granada/Cádiz/Mérida | Alta | Media | Alta | No | No | Funciones acreditadas, sin manual interno completo |
| Peoplenet AAPP recopilado | No | Parcial | No | Alta | Media | Organización, personal y nómina |

La principal laguna sigue siendo un manual público reciente de tramitación interna,
baremación y gestión de llamamientos. Para esa superficie son prioritarios los pliegos de
CONVOCA de la propia Diputación y los talleres con el personal que realiza el trabajo.

## 3. Patrones que conviene adoptar

### 3.1 Expediente personal y méritos reutilizables

FIDES es la referencia más sólida para un registro único de méritos:

- un mérito se registra una vez con su evidencia;
- distingue experiencia interna obtenida del sistema de personal y experiencia externa
  aportada por la persona;
- consulta titulaciones y formación en otras fuentes cuando es posible;
- si la comprobación automática no es concluyente, deriva a revisión humana;
- permite presentación individual o conjunta;
- distingue pendiente, en trámite, validado, duplicado, no catalogable, documentación
  incompleta y descartado (página física 111 del manual de méritos).

El producto objetivo añadirá `subsanación`, `caducado` y `sustituido`. Cada participación
en un proceso conservará una instantánea de los méritos utilizados. Una rectificación del
registro personal no alterará retroactivamente solicitudes ya presentadas.

Osakidetza confirma el valor de importar de oficio la experiencia interna y de mantener un
currículo reutilizable (instrucciones, páginas 9-11). GVBORSES evita pedir de nuevo
titulaciones e idiomas que ya constan acreditados (páginas 11-12).

### 3.2 Presentación administrativa completa y recuperable

La secuencia común observada y recomendada es:

```text
borrador
→ comprobación de requisitos
→ documentos y méritos
→ cálculo orientativo
→ tasas o exención, cuando proceda
→ resumen exacto
→ firma o consentimiento exigible
→ registro
→ justificante y copia presentada
```

AGE IPS documenta además estados intermedios, modificación dentro de plazo y
subsanación con nuevo justificante (páginas 38-48). GVBORSES conserva borradores,
presentaciones vigentes y versiones archivadas (páginas 21-23). Ese patrón debe reforzarse:

- autoguardado en servidor con fecha y versión;
- operaciones idempotentes;
- conciliación tras pago o firma externos;
- recuperación tras caída del navegador o de una integración;
- una presentación nunca se sobrescribe ni desaparece;
- el justificante permanece siempre en el expediente.

### 3.3 Autobaremación calculada y explicable

SACYL ofrece desglose por área, tipo y subtipo y diferencia la puntuación declarada de la
validada (páginas 16-18 y 20-21). GVBORSES, en cambio, pide que la persona calcule y
escriba los puntos (páginas 15-16); este patrón no se adoptará.

El sistema objetivo calculará:

- regla y versión aplicadas;
- unidades acreditadas;
- puntos brutos;
- topes;
- incompatibilidades y solapamientos;
- puntuación declarada, calculada, validada y definitiva;
- explicación de cualquier diferencia introducida por RRHH.

### 3.4 Cortes, versiones y posición propia

SACYL conserva cada registro de solicitud, los cortes en los que participó, la validación de
méritos y la posición por zona y modalidad (páginas 19-24). Este patrón encaja con la
petición de RRHH y con la reconstrucción probatoria.

La vista privada mostrará posición, situación, última actuación y siguiente acción de la
persona. La vista pública solo publicará el detalle individual que resulte necesario,
proporcionado y jurídicamente aprobado. Se evitará publicar listados íntegros cuando una
consulta de posición propia resuelva la necesidad.

### 3.5 Preferencias y disponibilidad configurables

SACYL, ULISES y Osakidetza separan territorios, centros, jornada, vacantes, nombramientos
cortos o largos y estados voluntarios. Junta VEPA y CAP permiten ordenar plazas por
preferencia.

La solución debe soportar, por bolsa y versión de reglamento:

- categoría y subcategoría;
- territorio, centro o unidad;
- duración y jornada;
- tipo de nombramiento o contrato;
- disponibilidad, suspensión y reactivación;
- causas justificadas y periodos;
- prioridades ordenadas;
- fecha efectiva y fecha de solicitud del cambio;
- aprobación cuando corresponda.

Los catálogos, límites y compatibilidades procederán de configuración funcional versionada,
no de constantes en el código.

### 3.6 Llamamientos explicables

La petición de RRHH exige llamamientos automáticos, detección de saltos y periodos de
indisponibilidad. Los manuales de Mérida, SACYL y Osakidetza aportan preferencias,
posición, ofertas y cambios de situación.

El sistema propondrá la siguiente persona mediante una ejecución reproducible de la regla.
Antes de emitir la oferta mostrará:

- bolsa, versión y puesto;
- personas consideradas y exclusiones aplicadas;
- orden, desempate y posición;
- disponibilidad y preferencias vigentes;
- último llamamiento y consecuencias;
- excepciones o conflictos detectados.

La selección ordinaria podrá automatizarse, pero los saltos, sanciones, exclusiones,
renuncias justificadas y excepciones requerirán competencia humana, motivo y evidencia.

### 3.7 Historial visible para la persona

El manual actual de Osakidetza incluye un registro de acciones visible para el usuario
(página 9). Se adoptará como línea temporal comprensible:

- qué ocurrió;
- cuándo;
- sobre qué solicitud, mérito, oferta o documento;
- resultado;
- justificante disponible;
- siguiente acción.

No mostrará información de seguridad que facilite ataques. La línea temporal ciudadana, la
auditoría probatoria y la telemetría técnica serán registros relacionados pero diferentes.

### 3.8 Autoservicio del empleado

El manual SERMAS separa información personal, datos económicos, puesto e historial,
trienios, situaciones administrativas, vacaciones, permisos, ausencias y turnos (páginas
3-15). El portal objetivo añadirá carpeta documental, certificados, dietas, formación,
carrera y movilidad, reutilizando la misma persona y expediente.

Los datos tendrán procedencia y estado:

- dato oficial de solo lectura;
- propuesta de rectificación;
- pendiente de comprobación;
- validado;
- rechazado con motivo;
- sustituido o histórico.

Cambiar una cuenta bancaria, un dato de identidad o un dato oficial no será una edición
ordinaria: exigirá un caso de uso específico, autenticación reforzada, evidencia y avisos.

### 3.9 Espacio del responsable

El manual del responsable SERMAS aporta una separación útil:

- cola de tareas;
- filtros por unidad;
- delegación por responsabilidad, unidad y fechas;
- aprobación de permisos;
- calendario y cobertura de equipo (páginas 4-10).

Se adoptará con mayor minimización. El responsable verá lo necesario para decidir y cubrir
el servicio, no el expediente completo, la nómina, los méritos de bolsa ni la causa clínica de
una ausencia. Las relaciones jerárquicas y delegaciones tendrán inicio, fin, ámbito,
aprobador y revocación.

### 3.10 Ayuda, accesibilidad y soporte

AGE separa dudas funcionales de incidencias técnicas y ofrece una versión en lectura fácil.
FIDES distingue ayuda documental, ayuda del formulario, discrepancias y problemas
técnicos. Se adoptarán:

- ayuda contextual versionada;
- preguntas frecuentes vinculadas a la convocatoria;
- lectura fácil para recorridos principales;
- diagnóstico de identidad, firma, pago y fichero;
- ticket enlazado al expediente sin enviar DNI o documentos por correo ordinario;
- estado público del servicio;
- bot limitado a conocimiento publicado, con fuente y fecha.

## 4. Espacios resultantes

### 4.1 Portal público y área del aspirante

Permitirá únicamente:

- consultar convocatorias, bases, requisitos, plazos y anuncios publicados;
- identificarse o actuar mediante representación acreditada;
- mantener datos propios, documentos y méritos;
- presentar solicitudes, pagar tasas, firmar y obtener justificantes;
- consultar sus subsanaciones, alegaciones, recursos, posición y llamamientos;
- gestionar sus preferencias y suscripciones;
- consultar su historial y abrir solicitudes de ayuda.

No mostrará módulos internos, datos de otras personas, búsquedas por DNI ni herramientas de
RRHH. La identidad corporativa de un empleado no ampliará este ámbito.

### 4.2 Portal del personal empleado

Será autoservicio propio para:

- expediente personal e históricos;
- puesto, destino, relación, antigüedad y trienios;
- certificados y solicitudes;
- nóminas y datos fiscales;
- Cronos, calendario, vacaciones y permisos;
- dietas y desplazamientos;
- documentos, titulaciones y formación;
- carrera, movilidad y procesos internos;
- comunicaciones corporativas.

Un dato verificado podrá ofrecerse al mismo titular como mérito reutilizable, sin abrirle
acceso al sistema interno que lo origina.

### 4.3 Espacio del responsable

Se orientará a tareas:

- autorizaciones pendientes;
- calendario y cobertura;
- sustituciones y delegaciones;
- alertas de vencimiento;
- datos mínimos de la unidad;
- informes agregados.

Cada acceso derivará de una relación organizativa o delegación vigente y del tipo de tarea.

### 4.4 Área interna de RRHH

La primera pantalla será una bandeja operativa con expedientes asignados, bloqueados y
próximos a vencer. Incluirá:

- filtros guardados y búsqueda autorizada;
- tabla densa con panel lateral de detalle;
- documento y mérito lado a lado;
- propuesta, revisión, resolución, firma y publicación separadas;
- motivos normalizados con explicación complementaria;
- abstención, recusación y conflicto de interés;
- imposibilidad de tramitar el propio expediente;
- operaciones masivas con previsualización, doble confirmación y recibo;
- auditoría visible de regla, versión, dato previo, dato resultante y autor;
- configuración funcional en un espacio separado de la tramitación diaria.

Este diseño es una recomendación profesional y deberá validarse con RRHH, porque los
manuales públicos recopilados no documentan el área interna completa.

## 5. Interacciones seguras entre espacios y módulos

### Título ya validado

```text
Personal valida un título del empleado
→ el registro canónico conserva procedencia y evidencia
→ el titular lo selecciona como aspirante
→ la solicitud de bolsa guarda una instantánea
```

No se abre el expediente de Personal desde Bolsa y la validación histórica no se altera.

### Incorporación desde una bolsa

```text
La persona acepta el llamamiento
→ Bolsa emite un evento de incorporación
→ Personal abre y valida el alta o nombramiento
→ después se activan, según proceda, identidad corporativa, Nóminas y Cronos
```

Bolsa no escribirá directamente en las tablas de Nóminas, Active Directory o control
horario.

### Permiso del empleado

```text
Empleado solicita permiso
→ responsable ve la tarea y la cobertura necesaria
→ aprueba o deniega dentro de su ámbito
→ RRHH interviene solo si lo exige la regla aplicable
```

## 6. Patrones que se descartan

- Un único portal que se limita a mostrar u ocultar módulos según una lista de roles.
- Sumar permisos de aspirante, empleado, responsable y RRHH en una misma sesión.
- Diferenciar contextos solo mediante color.
- Autenticación con DNI y contraseña propia de la aplicación.
- Iconos sin texto o estados expresados solo con color.
- Grandes formularios sin progreso ni resumen de errores.
- Guardado exclusivamente manual.
- Confundir guardar, enviar, firmar y registrar.
- Sobrescribir o eliminar una presentación registrada.
- Hacer depender la validez de que el usuario regrese de la pasarela de pago o firma.
- Obligar a conservar en el equipo la única copia del justificante.
- Separar Bolsa y currículo en productos inconexos para el usuario.
- Pedir nuevamente datos o documentos ya verificados sin causa.
- Autobaremo introducido manualmente por el candidato.
- Listados públicos completos cuando basta la posición propia.
- Exportaciones ofimáticas sin permiso, finalidad, marca y auditoría.
- Soporte mediante correo ordinario que solicite DNI o documentación personal.
- Acceso del responsable al expediente completo de sus subordinados.
- Modificación directa de Personal o Nóminas desde el módulo Bolsa.

## 7. Relación con las fuentes de la Diputación

La petición de RRHH y `convoca_dipgra` siguen siendo la base funcional local:

- gestión de bolsas y candidatos;
- llamamientos según reglamento;
- contratos, ceses y reincorporaciones;
- reglas configurables;
- portal privado y consulta pública;
- control interno y estadísticas;
- plantillas;
- comunicaciones;
- auditoría.

Las referencias externas añaden profundidad a esos requisitos: méritos compartidos,
versiones registradas, cortes, posición propia, preferencias, ofertas, subsanación,
justificantes, autoservicio de empleado y delegaciones de responsable.

No acreditan, por sí solas, el procedimiento interno exacto de la Diputación. Antes de
cerrar las especificaciones será necesario levantar con RRHH:

- actores y competencias reales;
- reglamentos y excepciones vigentes;
- fuentes maestras de persona, puesto, contrato y nómina;
- circuito de firma, registro, archivo y notificación;
- modelos documentales;
- conservación y publicación;
- controles de conflicto de interés;
- picos de carga y continuidad;
- integraciones disponibles de GINPIX, Active Directory y sede electrónica.

## 8. Documentación existente que debe rebaselinarse

Los documentos actuales de `docs/portal_vec` hablan en varios puntos de un único portal o
una única envolvente que cambia por rol. Antes de convertirlos en especificación contractual
deben revisarse, en especial:

1. `contrato_modulos_vec.md`;
2. `arquitectura_tecnica.md`;
3. `ux_portal_empleado.md`;
4. `desarrollo_vec_orquesta.md`;
5. `plan_implantacion_orquesta.md`;
6. `inventario_vec.md`;
7. `estudio_pantallas_profesionales.md`;
8. `cumplimiento_y_seguridad.md`.

La revisión conservará el núcleo y los servicios comunes, pero dividirá entradas,
proyecciones, sesiones y permisos por superficie. PostgreSQL será la persistencia inicial y
Oracle tendrá un adaptador, migraciones y pruebas de contrato independientes cuando se
incorpore.
