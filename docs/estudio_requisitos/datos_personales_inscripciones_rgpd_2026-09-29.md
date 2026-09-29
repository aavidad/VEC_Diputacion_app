# Datos personales de las inscripciones: qué guardar, dónde y con qué garantías

Fecha: 29 de septiembre de 2026.
Estado: estudio sin programar. Pendiente de revisión independiente, del Delegado de
Protección de Datos (DPD) y de RRHH. Nada de lo que aquí se propone autoriza a tratar
datos reales.

Petición de Alberto: en el área personal hacen falta, además del correo, el teléfono,
el móvil, el domicilio, el código postal y el resto de datos que exijan las
inscripciones en bolsas, procesos selectivos y provisión, porque VEC sustituirá a
CONVOCA. Hay que estudiar cómo guardarlos cumpliendo la protección de datos. Durante el
estudio Alberto añadió cuatro condiciones:

- los datos de las personas internas (trabajadores de la Diputación) y los de las
  externas (aspirantes) no se mezclan;
- una persona puede pasar de un lado a otro en cualquier momento, o estar en los dos a
  la vez, y eso debe funcionar sin mezclar;
- el paso a plaza fija (proceso selectivo o provisión) se trata igual que el paso por
  bolsa, con sus diferencias;
- cada decisión debe mostrar sus alternativas, sus riesgos y por qué se elige, y
  distinguir lo que manda la norma de lo que es criterio técnico y de lo que decide el
  DPD.

## Cómo leer este estudio

Cada afirmación relevante lleva una de estas tres marcas:

- «(norma)»: lo exige una ley o un reglamento; se cita el artículo.
- «(criterio técnico)»: es una propuesta de diseño; puede cambiarse sin infringir nada,
  aunque cambiarla tiene consecuencias que se explican.
- «(a confirmar por el DPD)»: es una interpretación jurídica o una decisión que
  corresponde al responsable del tratamiento con el consejo del DPD. Mientras no se
  confirme, VEC la trata como valor provisional y configurable.

Las normas se citan por su nombre corto: RGPD (Reglamento (UE) 2016/679), LOPDGDD (Ley
Orgánica 3/2018), Ley 39/2015 (Procedimiento Administrativo Común), Ley 40/2015
(Régimen Jurídico del Sector Público), TREBEP (Real Decreto Legislativo 5/2015), ENS
(Real Decreto 311/2022). Al final hay una lista con todas las citas.

## Resumen de la propuesta

1. Separar físicamente a las dos poblaciones. Los datos de aspirantes los posee un
   módulo nuevo, Aspirantes; los de trabajadores, Personal. Cada uno con su esquema, sus
   roles de base de datos y sus claves de cifrado. En desarrollo basta con esquemas
   separados en la misma instancia. En producción, la parte externa debería ir en una
   base de datos propia, porque es la que queda expuesta a internet (a decidir por
   Seguridad al categorizar el sistema).
2. Cada población tiene su propio identificador opaco: `asp_` para aspirante y `emp_`
   para trabajador. Ninguno se deriva del otro.
3. La relación entre la persona aspirante y la persona trabajadora se guarda aparte, en
   un registro de correspondencias que solo existe en la zona interna. Solo se crea por
   un acto administrativo (nombramiento, contrato) o por declaración de la propia
   persona cuando un trámite lo necesita, y solo se consulta para una lista cerrada de
   finalidades, con registro de cada consulta.
4. El paso de aspirante a trabajador es un traspaso gobernado: lo dispara la
   resolución o el contrato, viaja por la bandeja de salida (outbox) y lleva solo los
   datos que exige la relación laboral. El expediente de aspirante no se copia.
5. Cada dato se pide en el momento en que hace falta y no antes. Lo que ya tiene una
   Administración se consulta por la Plataforma de Intermediación de Datos en lugar de
   pedírselo a la persona (Ley 39/2015, art. 28.2).
6. Cifrado por campo en la aplicación, con claves fuera del proceso y rotables,
   distintas por población y por categoría. La discapacidad y las adaptaciones van en un
   almacén aparte con su propia clave.
7. Antes de tratar datos reales hacen falta la evaluación de impacto (EIPD), la
   actualización del Registro de Actividades de Tratamiento, la categorización ENS y los
   textos de información aprobados. Hasta entonces se puede construir casi todo con
   datos sintéticos.

## 1. Qué guarda VEC hoy

Revisado en el código de `main` a 29 de septiembre de 2026:

- Bolsa guarda por cada participación un correo y hasta dos teléfonos
  (`internal/modules/bolsa/domain/datos_contacto_participacion.go`). El bloque se cifra
  entero con AES-GCM ligado a la participación y a la versión, y las lecturas
  ordinarias lo reciben enmascarado. Las anotaciones de cada contacto rechazan textos
  que contengan DNI, NIE, correo o teléfono.
- Bolsa identifica a la persona candidata con una referencia `can_` que se calcula con
  una clave secreta a partir del DNI enmascarado (formato `***1234**`) y del nombre
  normalizado, tal como aparecen en las listas de CONVOCA
  (`internal/modules/bolsa/application/constitucion/candidato.go`). Así no guarda el
  documento completo.
- La importación de CONVOCA guarda las filas en una zona de preparación cifrada.
- La proyección pública de Bolsa no incluye DNI, contacto ni expediente.
- El correo obligatorio del alta (`contacto_usuario_vec`) está cifrado con una subclave
  propia.
- Usuarios guarda «Mis preferencias» (5.08a), «Mis correos» (5.08b, cifrados, con
  verificación por código) y la foto (5.08c). Todas sus tablas usan como clave la
  persona VEC (`per_`). La superficie de acceso (`interna_corporativa` o
  `externa_personal`) aparece solo en el contexto de cada transacción y en los filtros
  por fila. Si la misma persona entra por los dos portales, comparte preferencias,
  correos y foto.
- El núcleo de identidad resuelve un contexto de actor con una persona `per_` que puede
  llevar a la vez un vínculo de tipo `candidato` (`can_`) y otro de tipo `empleado`
  (`emp_`). La cuenta se localiza por un alias calculado con HMAC, sin guardar el
  identificador en claro.
- Personal ofrece la ficha propia del empleado (puesto, relación, organización) sin
  datos de contacto.
- Las claves de cifrado de desarrollo se derivan dentro del proceso. No hay todavía
  HSM ni gestor de claves (riesgo R-08 del informe del comité de seguridad).

Conclusión: el cifrado y el enmascarado ya siguen un buen patrón, pero la separación
entre internos y externos es solo lógica, y el contexto de actor une en un mismo objeto
la condición de aspirante y la de trabajador. Los apartados 4 y 5 proponen cómo
corregirlo.

## 2. Inventario de datos que piden las inscripciones

### 2.1 Fuentes revisadas

- Hojas de CONVOCA de un proceso de estabilización (resultado de autobaremación). Solo
  se han mirado los nombres de las columnas: DNI/NIE enmascarado, primer apellido,
  segundo apellido, nombre, turno, grupo de méritos, descripción del grupo, orden,
  descripción del mérito, puntos de autobaremación, puntos del tribunal y motivo del
  tribunal. En los méritos aparecen servicios prestados en la Diputación, en otras
  Administraciones y en el sector público, formación recibida o impartida y
  titulaciones superiores a la exigida.
- Petición de RRHH y pliego de CONVOCA (ficha de adaptación de Bolsa, B3 y B4): correo y
  dos teléfonos, e historial de contactos.
- Portal público de CONVOCA: inscripción con Cl@ve como interesado o como
  representante, y pago de tasas.
- TREBEP, arts. 56, 57 y 59, que fijan los requisitos generales de acceso.

Lo que no figura en esas fuentes se marca como «habitual en las bases», y RRHH debe
confirmar si sus bases lo piden (pregunta 80 de `dudas.md`).

### 2.2 Momentos del recorrido

| Momento | Qué ocurre | Quién trata los datos |
| --- | --- | --- |
| Inscripción | La persona presenta la solicitud y, en su caso, paga la tasa | La persona, RRHH (Selección) |
| Admisión | Se comprueban requisitos y se publican listas de admitidos y excluidos | RRHH |
| Baremación y pruebas | Se valoran méritos y se hacen ejercicios | Tribunal u órgano de selección, RRHH |
| Llamamiento (bolsa) | Se ofrece un puesto y se registra la respuesta | RRHH, centro solicitante en lo imprescindible |
| Nombramiento o contrato | Resolución, toma de posesión o firma del contrato | RRHH, Personal, Intervención |
| Relación laboral | Nómina, jornada, permisos | Personal y demás módulos internos |

### 2.3 Inventario por categoría

«Necesario» significa que una norma o las bases lo exigen para esa finalidad. Si una
fila dice «no», VEC no debería pedirlo en ese momento (principio de minimización, RGPD
art. 5.1.c, norma).

#### Identificación

| Dato | ¿Necesario? | Momento | Fuente preferente |
| --- | --- | --- | --- |
| Nombre y apellidos | Sí (Ley 39/2015, art. 66.1.a) | Inscripción | Certificado, DNIe o Cl@ve |
| DNI o NIE | Sí, para identificar y publicar de forma seudonimizada | Inscripción | Certificado, DNIe o Cl@ve |
| Pasaporte o documento de otro Estado | Solo para nacionales de otros Estados (TREBEP art. 57) | Inscripción | La persona |
| Nacionalidad | Sí, es requisito (TREBEP art. 56.1.a y art. 57) | Inscripción, como declaración | Declaración; comprobación en admisión |
| Fecha de nacimiento | Solo para comprobar la edad (TREBEP art. 56.1.c) | Admisión | Consulta de identidad por la Plataforma de Intermediación (disponibilidad a confirmar) |
| Sexo | No es requisito. Puede hacer falta para estadística (Ley Orgánica 3/2007, art. 20) | Inscripción, voluntario | La persona (a confirmar por el DPD) |
| Firma o identificación electrónica | Sí (Ley 39/2015, arts. 9 y 11) | Inscripción | Certificado o Cl@ve |

#### Contacto y domicilio

| Dato | ¿Necesario? | Momento | Fuente |
| --- | --- | --- | --- |
| Correo electrónico | Para avisos de puesta a disposición de notificaciones, que la persona aporta de forma voluntaria (Ley 39/2015, arts. 41.6 y 66.1.b). En bolsa es el canal de llamamiento que prevé la petición de RRHH | Inscripción | La persona, verificado |
| Teléfono y móvil | Igual que el correo; en bolsa, para el llamamiento (pendiente, dudas 14 y 45) | Inscripción en bolsa; llamamiento | La persona |
| Domicilio y código postal | Solo si la persona no está obligada a relacionarse electrónicamente y elige notificación en papel (Ley 39/2015, arts. 14 y 66.1.b). No hace falta para inscribirse por medios electrónicos | Inscripción, solo en ese caso | La persona |
| Domicilio a efectos laborales y fiscales | Sí, para el contrato, la Seguridad Social y la nómina | Nombramiento o contrato | La persona, ya en Personal |

Criterio técnico: el domicilio es el dato cuya petición más conviene aplazar. Si las
bases obligan a inscribirse por vía electrónica, basta con el correo y el teléfono
hasta el nombramiento.

#### Académicos y méritos

| Dato | ¿Necesario? | Momento | Fuente preferente |
| --- | --- | --- | --- |
| Titulación exigida | Sí (TREBEP art. 56.1.e) | Declaración en la inscripción; comprobación en admisión o en el hito que fijen las bases (duda 37) | Consulta de titulaciones por la Plataforma de Intermediación; si no está disponible, la persona |
| Servicios prestados en la Diputación | Solo si puntúan | Baremación | Personal (la Diputación ya lo tiene: Ley 39/2015, art. 28.2) |
| Servicios en otras Administraciones o empresas | Solo si puntúan | Baremación | Vida laboral por la Plataforma de Intermediación y certificados de la otra Administración |
| Formación recibida o impartida | Solo si puntúa | Baremación | La persona |
| Titulaciones superiores o idiomas | Solo si puntúan o son requisito | Baremación | Plataforma de Intermediación o la persona |
| Carné de conducir u otras habilitaciones | Solo si las bases lo exigen para la categoría | Admisión | Consulta a la DGT por la Plataforma (disponibilidad a confirmar) o la persona |

#### Situación laboral

| Dato | ¿Necesario? | Momento | Fuente |
| --- | --- | --- | --- |
| Ser personal de la Diputación | Solo en promoción interna o para méritos internos | Inscripción | Registro de correspondencias y Personal (apartado 4) |
| Situación de desempleo | Solo si da derecho a exención o reducción de la tasa según la ordenanza fiscal | Pago de la tasa | Consulta al SEPE por la Plataforma (disponibilidad a confirmar) |
| Declaración de no haber sido separado ni inhabilitado | Sí (TREBEP art. 56.1.d) | Inscripción, como declaración responsable; comprobación antes del nombramiento | La persona |
| Compatibilidad con otro empleo | No en la inscripción. Sí antes de la toma de posesión o del contrato (Ley 53/1984) | Nombramiento o contrato | La persona |

#### Discapacidad y adaptaciones (categoría especial, RGPD art. 9)

| Dato | ¿Necesario? | Momento | Fuente preferente |
| --- | --- | --- | --- |
| Grado de discapacidad igual o superior al 33 % | Solo si la persona opta al cupo de reserva (TREBEP art. 59.1; Real Decreto 2271/2004, art. 3) o pide exención de tasa | Inscripción | Consulta de discapacidad por la Plataforma de Intermediación; si no, certificado |
| Solicitud de adaptación de tiempos o medios | Solo si la persona la pide (TREBEP art. 59.2; Real Decreto 2271/2004, art. 8) | Inscripción | La persona |
| Dictamen técnico sobre la adaptación | Solo si el tribunal lo necesita para resolver la adaptación | Antes de las pruebas | La persona o el órgano competente |
| Diagnóstico o causa de la discapacidad | No. Nunca hace falta | Nunca | No se pide |
| Aptitud médica para el puesto | Solo la conclusión «apto» o «no apto» (Ley 31/1995, art. 22.4) | Antes de la incorporación | Vigilancia de la salud; RRHH recibe solo la conclusión |

#### Cupos y turnos

| Dato | ¿Necesario? | Momento | Nota |
| --- | --- | --- | --- |
| Turno (libre, reserva por discapacidad, promoción interna) | Sí, si la convocatoria tiene varios | Inscripción | El turno de reserva por discapacidad revela un dato de salud: no puede aparecer junto al nombre en listas públicas sin decisión del DPD (pregunta 83) |
| Otros turnos o preferencias (víctimas de violencia de género o de terrorismo) | Solo si las bases o la ordenanza fiscal los prevén | Inscripción | Dato especialmente sensible. RRHH confirma si existen (pregunta 80) |

#### Datos de terceros

| Dato | ¿Necesario? | Momento | Nota |
| --- | --- | --- | --- |
| Representante: nombre, DNI y acreditación de la representación | Solo si actúa un representante (Ley 39/2015, art. 5) | Inscripción | El representante es otro interesado a efectos del RGPD |
| Título de familia numerosa | Solo si da derecho a exención de tasa | Pago de la tasa | Revela datos del resto de la familia. Mejor consultarlo por la Plataforma y guardar solo «tiene derecho a la exención» |
| Hijos y situación familiar para el IRPF (modelo 145) | Solo en la nómina | Nombramiento o contrato | Lo pide Personal, nunca la inscripción |
| Persona de contacto de emergencia | No en ninguna inscripción | Nunca en Aspirantes | Si Personal la quisiera, es otro tratamiento |

#### Datos que solo hacen falta al nombramiento o al contrato

Número de afiliación a la Seguridad Social, cuenta bancaria (IBAN), datos del IRPF,
domicilio fiscal, declaración de compatibilidad, conclusión de aptitud médica y, en los
puestos con contacto habitual con menores, el certificado negativo del Registro Central
de Delincuentes Sexuales (Ley Orgánica 1/1996, art. 13.5). Este último es un dato del
RGPD art. 10 y solo puede tratarse cuando lo exige una norma. Ninguno de estos datos se
pide en la inscripción ni se guarda en Aspirantes (criterio técnico, apoyado en RGPD
art. 5.1.c).

### 2.4 Qué aporta ya la identificación electrónica

El certificado de persona física, el DNIe y Cl@ve entregan nombre, apellidos y número
de DNI o NIE. Por eIDAS, los ciudadanos de otros Estados europeos aportan además la
fecha de nacimiento (a confirmar con Informática según la versión de Cl@ve contratada).
Ninguno aporta domicilio, teléfono ni nacionalidad fiable. Por tanto:

- VEC no debe pedir que se escriban el nombre ni el DNI: los toma de la identificación y
  la persona solo los ve (criterio técnico; reduce errores y suplantaciones).
- El resto se declara o, mejor, se consulta.

## 3. Base jurídica por finalidad

El consentimiento no sirve como base general en un proceso de selección público: la
persona no puede negarse sin perder la opción al puesto, y el RGPD advierte de ese
desequilibrio (considerando 43) (norma). Las bases son la obligación legal y el
ejercicio de poderes públicos, que exigen una norma con rango de ley (RGPD art. 6.3;
LOPDGDD art. 8) (norma).

| Finalidad | Base (RGPD) | Norma que la habilita | Marca |
| --- | --- | --- | --- |
| Gestionar inscripciones, admisión y baremación de procesos selectivos | Art. 6.1.e | TREBEP arts. 55, 56 y 61; Ley 7/1985, art. 91.2; Real Decreto 896/1991 | (norma) |
| Gestionar bolsas y llamamientos | Art. 6.1.e | TREBEP art. 10.2 (interinos) y art. 11 (laborales); reglamento de bolsas de la Diputación | (a confirmar por el DPD: el reglamento de bolsas no tiene rango de ley y la base descansa en el TREBEP) |
| Provisión de puestos | Art. 6.1.e | TREBEP arts. 78 a 84 | (norma) |
| Cupo de discapacidad y adaptaciones | Art. 9.2.b y 9.2.g, con art. 6.1.e | TREBEP art. 59; Real Decreto Legislativo 1/2013; Real Decreto 2271/2004; LOPDGDD art. 9.2 | (a confirmar por el DPD cuál de las dos letras del art. 9.2 aplica) |
| Avisos por correo y teléfono | Art. 6.1.e | Ley 39/2015, art. 41.6; el dato lo aporta la persona voluntariamente | (norma) |
| Publicar listas | Art. 6.1.c y 6.1.e | Ley 39/2015, art. 45; bases; LOPDGDD disposición adicional 7.ª | (norma) |
| Consultar datos a otras Administraciones | Art. 6.1.e | Ley 39/2015, art. 28.2; Ley 40/2015, art. 155 | (norma) |
| Alta por nombramiento de funcionario | Art. 6.1.c y 6.1.e | TREBEP arts. 10 y 62 | (norma) |
| Alta por contrato laboral | Art. 6.1.b y 6.1.c | Estatuto de los Trabajadores; TREBEP art. 11 | (norma) |
| Registro de correspondencias aspirante-trabajador | Art. 6.1.c y 6.1.e | Ley 53/1984 (incompatibilidades); Estatuto de los Trabajadores, art. 15.5 (encadenamiento); TREBEP art. 10 y disposición adicional 17.ª; certificación de servicios | (a confirmar por el DPD) |
| Avisos de oportunidades a trabajadores («búsqueda de talento interno») | Art. 6.1.e, con prueba de compatibilidad del art. 6.4, o consentimiento para recibir el aviso | TREBEP arts. 14.c y 16 (carrera profesional) | (a confirmar por el DPD, pregunta 92) |
| Registro de accesos | Art. 6.1.c | RGPD art. 32; ENS, anexo II | (norma) |
| Foto voluntaria en el portal | Art. 6.1.a | Consentimiento, retirable en cualquier momento | (a confirmar por el DPD, ya planteado en la duda 76) |

### 3.1 No pedir lo que ya tiene la Administración

La Ley 39/2015 reconoce el derecho a no aportar documentos elaborados por cualquier
Administración (art. 28.2) y a no presentar datos que ya estén en su poder (art.
53.1.d). La Administración debe obtenerlos por sus redes o por la Plataforma de
Intermediación de Datos, salvo que la persona se oponga expresamente (art. 28.2)
(norma).

Servicios de la Plataforma que encajan con las inscripciones (disponibilidad y
adhesión de la Diputación a confirmar por Informática, pregunta 81):

| Dato | Servicio |
| --- | --- |
| Identidad | Verificación de datos de identidad |
| Titulación universitaria y no universitaria | Consulta de títulos |
| Discapacidad | Consulta de grado de discapacidad |
| Situación de desempleo | Consulta de inscripción como demandante de empleo |
| Vida laboral | Consulta de vida laboral de la Tesorería General de la Seguridad Social |
| Familia numerosa | Consulta de título de familia numerosa |
| Delitos sexuales | Consulta de inexistencia de antecedentes |

Consecuencias de diseño (criterio técnico):

- La inscripción ofrece «Autorizo la consulta» marcado por defecto y la opción de
  oponerse, con aviso de que entonces debe aportar el documento (Ley 39/2015, art.
  28.2).
- VEC guarda el resultado de la consulta como hecho verificado («titulación X
  verificada el día D, consulta con referencia R»), no el documento. Así se guarda
  menos y el dato es más fiable.
- Si la persona se opone, aporta el documento y VEC lo trata como documento aportado.

La consulta por la Plataforma es una cesión entre Administraciones. La persona debe
saberlo por la información del art. 13 RGPD. Cuando VEC recibe datos que no vienen de la
persona se aplica el art. 14 RGPD, con la excepción del art. 14.5.c si la obtención la
establece expresamente la ley (a confirmar por el DPD).

## 4. Separación entre personas internas y externas

### 4.1 Qué se quiere evitar

- Que un fallo en el portal externo, expuesto a internet, dé acceso a datos de
  trabajadores.
- Que RRHH, al gestionar a un compañero como aspirante, vea sin motivo su expediente
  laboral, o al revés.
- Que la condición de aspirante (por ejemplo, haberse presentado a otra plaza) se
  conozca en el entorno laboral sin necesidad.
- Que un dato se reutilice para una finalidad distinta sin base (RGPD art. 5.1.b,
  norma).

### 4.2 Tres opciones de separación

**Opción A. Separación lógica actual.** Mismas tablas, columna `superficie`, roles por
superficie y filtros por fila.

- Ventaja: ya existe y funciona en Usuarios.
- Riesgos: un error en una política de fila o en una función con privilegios de
  propietario cruza poblaciones. Las copias de seguridad, las réplicas y las
  consultas de mantenimiento ven a las dos poblaciones juntas. Una misma clave cifra a
  todos. Y la clave `per_` ya une a la persona en ambos portales (apartado 1).
- Veredicto: insuficiente para lo que pide Alberto.

**Opción B. Esquemas y roles distintos por población, en la misma instancia.**

- Ventaja: cada población tiene su esquema, sus roles de propietario y de ejecución,
  sin permisos cruzados, y su propia clave de cifrado. Un error en un esquema no da
  acceso al otro. Coste bajo, porque VEC ya separa esquemas y roles por módulo.
- Riesgos: el administrador de la base, las copias y el sistema operativo siguen
  siendo comunes. Si la aplicación externa comparte proceso con la interna, un
  compromiso del proceso alcanza las credenciales de ambas.
- Veredicto: suficiente para desarrollo y para la presentación, con dos condiciones:
  procesos y credenciales separados por superficie, y claves distintas.

**Opción C. Base de datos o instancia separada para la parte externa.**

- Ventaja: la zona expuesta a internet solo alcanza su propia base. Copias, réplicas,
  administración y claves se gobiernan por separado. Es la que mejor encaja con la
  separación por zonas que ya exige el informe del comité (riesgo R-06) y con el ENS.
- Riesgos: más coste de operación (otra instancia, otras copias, otra monitorización).
  RRHH necesita leer datos de aspirantes desde la zona interna, así que la aplicación
  interna tendrá conexión a las dos bases, con roles de lectura por finalidad. El flujo
  permitido va de dentro hacia fuera: la parte externa nunca tiene credenciales de la
  interna.
- Veredicto: recomendable para producción.

**Recomendación** (criterio técnico; la decisión final es de Seguridad al categorizar
el sistema, pregunta 89):

- Ahora: opción B, diseñada para poder pasar a C sin reprogramar. Eso significa ninguna
  clave foránea ni consulta que cruce de un esquema de población al otro, conexiones
  distintas por población, claves de cifrado distintas y procesos distintos para el
  portal externo y el interno.
- Producción: opción C para el almacén de Aspirantes.

### 4.3 Identificadores

| Opción | Descripción | Riesgo |
| --- | --- | --- |
| Un único identificador de persona (`per_`) para todo | Lo que hay hoy en Usuarios y en el contexto de actor | Cualquier tabla de cualquier módulo puede cruzarse con cualquier otra: une a las dos poblaciones |
| Identificadores derivados del DNI con la misma clave en los dos lados | Cada almacén calcula su referencia con HMAC del DNI | Quien tenga la clave o ambas bases puede unirlas. La separación es aparente |
| Identificadores aleatorios distintos por almacén y enlace aparte | `asp_` en Aspirantes, `emp_` en Personal, generados al azar; la relación vive en un registro propio | Hay que mantener ese registro y gobernar su uso. Es lo que se recomienda |

Recomendación (criterio técnico):

- Aspirantes genera `asp_` al azar y guarda el DNI cifrado, con un índice ciego (HMAC
  con clave propia de Aspirantes) para buscar por DNI y detectar inscripciones
  duplicadas.
- Personal genera o mantiene `emp_` y guarda el DNI con su propio índice ciego y su
  propia clave.
- La autenticación puede seguir siendo común, pero la resolución del contexto de actor
  debe proyectar solo el vínculo de la población del portal: en el portal externo, la
  persona es `asp_` y nada más; en el interno, `emp_` y nada más. Además, los alias de
  cuenta deberían calcularse con claves distintas por superficie, para que la base de
  identidad tampoco sea un punto de unión (criterio técnico; en el portal interno la
  entrada acabará siendo Kerberos, que ya no usa el DNI).
- La referencia `can_` de Bolsa, calculada a partir del DNI enmascarado y del nombre,
  debería sustituirse por `asp_`. Hoy dos personas con el mismo nombre y las mismas
  cuatro cifras centrales del DNI tendrían la misma referencia. Además, si la clave se
  filtrara, la referencia se podría calcular a partir de las listas públicas, que
  muestran esos mismos datos.

### 4.4 Registro de correspondencias

Hace falta saber, en ciertos casos y solo en ellos, que una aspirante es o fue
trabajadora:

- para darle de alta en Personal cuando se la nombra o contrata;
- para comprobar la condición de personal en promoción interna;
- para certificar los servicios prestados en la Diputación cuando puntúan, sin pedirle
  un certificado que la Diputación ya tiene (Ley 39/2015, art. 28.2);
- para controlar el encadenamiento de contratos (Estatuto de los Trabajadores, art.
  15.5) y la duración de las interinidades (TREBEP art. 10 y disposición adicional
  17.ª);
- para las incompatibilidades (Ley 53/1984).

Alternativas:

1. Sin enlace. Cada lado ignora al otro. No sirve: los traspasos y los méritos
   internos serían imposibles o exigirían pedir a la persona certificados que la
   Diputación ya tiene.
2. Enlace automático al iniciar sesión con el mismo certificado en los dos portales.
   Mezcla por defecto: cualquier trabajador que se inscriba quedaría unido sin
   finalidad concreta.
3. Enlace gobernado en un registro aparte. Es la opción recomendada.

Diseño del registro (criterio técnico; finalidades a confirmar por el DPD, pregunta
78):

- Vive en la zona interna y lo posee Personal, porque la mayoría de los enlaces nacen
  de un acto de personal. El portal externo no tiene acceso.
- Cada fila guarda `asp_`, `emp_`, origen (nombramiento, contrato o declaración de la
  persona), referencia del acto, base jurídica, fecha, actor y estado. Es de solo
  adición: una anulación es una fila nueva.
- Se crea solo de dos maneras. Por traspaso, cuando lo dispara un acto de nombramiento
  o contrato (apartado 4.5). O por declaración de la persona en un trámite que lo
  requiera, como la promoción interna o la solicitud de que se tengan en cuenta sus
  servicios en la Diputación: el portal externo envía el DNI acreditado por el
  certificado en un sobre cifrado para el registro, que calcula el índice ciego con la
  clave de Personal y localiza `emp_`. El portal externo nunca llega a conocer `emp_`.
- Se consulta solo para una lista cerrada de finalidades, con permiso propio y registro
  de cada consulta. Nada de «¿es esta persona un trabajador?» como consulta libre.
- La respuesta es la mínima: «es personal de la Diputación en la categoría exigida: sí
  o no», o «servicios certificables: N días en la categoría X», nunca el expediente.

### 4.5 Traspasos gobernados

Hay cuatro recorridos. Los cuatro siguen el mismo mecanismo; cambian el origen, el acto
que los dispara y lo que pasa al final.

Mecanismo común (criterio técnico, alineado con la arquitectura de VEC: referencias
opacas, eventos y bandeja de salida):

1. El módulo que dicta el acto (Contratación temporal o Procesos selectivos) confirma
   el nombramiento o el contrato y, en la misma transacción, escribe el evento en su
   bandeja de salida con la referencia del acto, `asp_` y la base jurídica.
2. Un proceso de traspaso de la zona interna recibe el evento, pide a Aspirantes por un
   puerto autorizado, con la finalidad «alta por nombramiento», solo los campos del
   catálogo de traspaso: identidad y contacto. Aspirantes los entrega en un sobre
   cifrado para Personal y anota la entrega en su registro de accesos.
3. Personal busca en el registro de correspondencias si hay un `emp_` anterior. Si lo
   hay (la persona ya trabajó aquí), reabre o añade una relación a ese mismo empleado.
   Si no, crea uno nuevo. En ambos casos escribe el enlace con el acto, la base y la
   lista y huella de los campos recibidos.
4. Todo es idempotente por la referencia del acto: repetir el evento no crea otro alta.
5. Los datos propios de la relación laboral (IBAN, afiliación, IRPF, aptitud médica,
   compatibilidad) no pasan por Aspirantes. La persona los aporta en un trámite de
   incorporación. Como quizá aún no tiene cuenta corporativa, ese trámite puede
   presentarse desde el portal externo, pero los datos viajan cifrados para Personal y
   se guardan solo allí.
6. El expediente de aspirante (inscripción, méritos, notas, listas) no se copia. Sigue
   en Aspirantes y en el módulo del proceso, con sus propios plazos.

| Recorrido | Acto que dispara | Relación | Qué queda en Aspirantes | Al terminar |
| --- | --- | --- | --- | --- |
| Bolsa → interinidad o contrato temporal | Resolución de nombramiento interino o contrato temporal (Contratación temporal) | Temporal, con fin previsto o por causa | La participación en la bolsa sigue viva, en estado «trabajando»; el resto del expediente, igual | Evento de fin de relación: Bolsa aplica su regla de reposición. La persona sigue siendo aspirante |
| Proceso selectivo → funcionario de carrera | Nombramiento publicado y toma de posesión (TREBEP art. 62.1) | Sin fin previsto | El expediente del proceso, con su plazo de conservación. La ficha de aspirante sigue disponible por si se presenta a otro proceso | Si un día pierde la condición (TREBEP art. 63), Personal cierra la relación. No se genera ningún alta en Aspirantes |
| Proceso selectivo → laboral fijo | Firma del contrato indefinido | Sin fin previsto | Igual que el anterior | Igual |
| Provisión de puestos | Resolución del concurso o de la libre designación | Cambia el puesto, no la relación | Los participantes de la Diputación ya son trabajadores: la solicitud vive en el módulo de provisión con referencia `emp_`. Los de otras Administraciones entran como aspirantes y, si obtienen el puesto, siguen el traspaso | Según el caso |

La diferencia práctica entre bolsa y plaza fija está en el final. En la bolsa se espera
que la relación termine y la persona vuelva a la lista. En la plaza fija, no: el enlace
se mantiene, pero la ficha de aspirante puede quedarse sin uso, y entonces se aplica su
plazo de conservación (apartado 5.8).

Caso inverso, fin de interinidad o de contrato temporal: Personal registra el cese
(TREBEP art. 10.3) y emite el evento de fin de relación. Contratación temporal lo
traslada a Bolsa, que aplica la reposición (duda 64). Nada viaja de Personal a
Aspirantes. Si la persona cambió su teléfono mientras trabajaba, lo cambió en Personal;
para que llegue a Aspirantes debe cambiarlo ella misma en el portal externo. VEC puede
ofrecerle en ese momento un botón «usar también en mis inscripciones», que es una
acción suya y queda registrada (criterio técnico).

### 4.6 Qué pasa con cada dato al terminar la relación

| Dato | Dónde | Al terminar la relación |
| --- | --- | --- |
| Identidad y relaciones de servicio | Personal | Se conservan: son la base de certificados de servicios, trienios y jubilación. Plazo según la tabla de valoración del Archivo (pregunta 84) |
| IBAN, datos del IRPF, afiliación | Personal | Se bloquean al cerrar la última nómina y se suprimen al prescribir las obligaciones fiscales y de Seguridad Social: cuatro años (Ley General Tributaria, art. 66; Ley General de la Seguridad Social, art. 24) (a confirmar por el DPD) |
| Contacto laboral | Personal | Se suprime al cerrar la relación, salvo el necesario para enviar documentos finales (finiquito, certificados) durante un plazo corto (a confirmar por el DPD) |
| Enlace aspirante-trabajador | Registro de correspondencias | Se conserva mientras exista alguno de los dos lados, porque se necesita si la persona vuelve |
| Participación en bolsa | Aspirantes y Bolsa | Sigue viva: la persona vuelve a la lista |

### 4.7 Persona que es a la vez trabajadora y aspirante

Es un caso normal: un trabajador que se inscribe en otra bolsa o se presenta a una
oposición.

- En el portal externo es `asp_`, con su ficha de aspirante y sus propios datos de
  contacto. En el interno es `emp_`. No comparten preferencias, correos ni foto
  (criterio técnico, pedido por Alberto).
- RRHH de Selección lo ve como aspirante y no ve su expediente laboral. Si una
  convocatoria exige ser personal de la Diputación (promoción interna), la comprobación
  pasa por el registro de correspondencias y devuelve solo «cumple» o «no cumple».
- La jefatura del trabajador no sabe que se ha presentado, salvo por las listas
  públicas que exige el procedimiento.
- Si el trabajador pertenece a RRHH y gestiona ese mismo proceso, debe abstenerse
  (Ley 40/2015, arts. 23 y 24) (norma). VEC puede ayudar con una lista de exclusiones por
  proceso: quien figura como aspirante en un proceso no puede abrir ese proceso desde el
  portal interno (criterio técnico, pregunta 90).

### 4.8 Búsqueda de talento interno

Idea futura: avisar a trabajadores (por ejemplo, de subgrupo C2 con titulación de C1)
de convocatorias a las que podrían optar.

- Se hace dentro de Personal y con datos de Personal: titulación acreditada, categoría,
  relación. No consulta Aspirantes.
- No ordena ni puntúa a nadie. Solo avisa a la persona. Así no hay decisión
  automatizada con efectos (RGPD art. 22, norma).
- Base: art. 6.1.e con la prueba de compatibilidad del art. 6.4, porque la titulación
  se recogió para gestionar la relación y ahora se usa para informar de la carrera. O
  consentimiento para recibir los avisos, que la persona puede retirar. Criterio
  técnico: ofrecerlo como preferencia que la persona activa. Así el control es suyo,
  sea cual sea la base que fije el DPD (pregunta 92).

### 4.9 Impacto en lo ya construido

| Pieza | Situación | Cambio propuesto | Cuándo |
| --- | --- | --- | --- |
| Usuarios 5.08a, 5.08b y 5.08c | Clave `per_` común a los dos portales | Añadir la población a la clave o, mejor, un esquema por población con roles y claves propios. Migración con copia duplicada por población de lo ya existente (datos sintéticos) | Fase 1 |
| Contexto de actor | Un objeto con vínculos `can_` y `emp_` a la vez | Proyectar solo el vínculo de la población del portal. Afecta al núcleo de identidad: necesita revisión de seguridad y SQL | Fase 1, con cuidado |
| Bolsa, contacto por participación | Correo y dos teléfonos por participación | Pasa a leer el contacto de la ficha de Aspirantes por puerto autorizado. Queda un contacto propio de la participación solo si las bases permiten uno distinto por bolsa (pregunta 80) | Fase 1 |
| Bolsa, referencia `can_` | Derivada de DNI enmascarado y nombre | Sustituir por `asp_` con tabla de equivalencia durante la migración | Fase 1 |
| Importación de CONVOCA | Zona de preparación cifrada | Descarga en Aspirantes, no en Bolsa (duda 45 sobre la base para usar esos contactos) | Fase 3 |
| Contratación temporal | Nombramiento e incorporación | Emitir el evento de traspaso por la bandeja de salida con referencia del acto y base jurídica | Fase 1 con datos sintéticos |
| Personal | Ficha propia, sin contacto | Recibe traspasos, crea `emp_`, posee el registro de correspondencias y el contacto interno | Fase 1 |
| `contacto_usuario_vec` | Correo obligatorio del alta | Se reparte por población: el externo va a Aspirantes y el interno a Personal o al directorio corporativo (duda 34) | Fase 1 |

## 5. Diseño del almacenamiento

### 5.1 Qué módulo posee los datos

| Alternativa | A favor | En contra |
| --- | --- | --- |
| Usuarios para todos | Ya guarda correos y preferencias | Es un módulo de cuenta y portal. Mezclaría poblaciones y convertiría a Usuarios en un almacén de expedientes |
| Personal para todos | Ya es la autoridad de la persona empleada | Meter a los aspirantes en Personal es justo la mezcla que Alberto quiere evitar |
| Un módulo nuevo por población: Aspirantes (externos) y Personal (internos) | Cada población tiene un dueño; la separación física cae de forma natural | Un módulo más. Hay que mover el contacto de Usuarios y de Bolsa |

Recomendación (criterio técnico): módulo nuevo **Aspirantes** para la población
externa y **Personal** para la interna. Usuarios se queda con lo que es del portal
(preferencias y foto), separado también por población. Bolsa, Procesos selectivos y
Provisión no guardan datos personales: guardan `asp_` y piden a Aspirantes lo que
necesitan para cada finalidad. Esto cumple la regla de AGENTS.md de que ningún módulo
lee tablas de otro, y la especificación E02 («Bolsa consulta el expediente por un
puerto autorizado y no duplica titulaciones, méritos ni datos personales»). Aspirantes
no es una segunda autoridad de identidad: la identidad sigue siendo la común, y
Aspirantes guarda el expediente de la persona aspirante.

Contenido de Aspirantes:

- Ficha: identidad (tomada de la identificación), nacionalidad, fecha de nacimiento si
  hace falta, contacto, domicilio solo si hay notificación en papel.
- Méritos y titulaciones declarados, y hechos verificados (resultado de consultas).
- Documentos aportados, guardados por la capacidad documental común con referencia
  opaca.
- En un almacén aparte, las categorías especiales (apartado 5.3).

### 5.2 Cifrado

| Alternativa | Qué protege | Qué no protege |
| --- | --- | --- |
| Cifrado del disco o de la base (TDE) | Robo del disco o de las copias en frío | A quien pueda consultar la base: administradores, una inyección SQL, un volcado |
| Cifrado dentro de PostgreSQL (`pgcrypto`) | Algo más que lo anterior | La clave pasa por la base y puede acabar en registros y en la memoria del servidor |
| Cifrado por campo en la aplicación con claves en un gestor externo (sobre: clave de datos envuelta por clave maestra) | La base solo ve texto cifrado. Un volcado o una consulta indebida no revelan nada | Impide buscar por contenido. Obliga a índices ciegos para la igualdad y a descifrar en la aplicación |

Recomendación (criterio técnico, apoyada en RGPD art. 32.1.a, que cita el cifrado como
medida, norma):

- Cifrado por campo en la aplicación, como ya hacen Bolsa y Usuarios, con AES-GCM y
  datos asociados que liguen el cifrado a la persona, al campo y a la versión. Así un
  valor no puede moverse a otra fila sin que falle el descifrado.
- Claves en un gestor de claves o HSM fuera del proceso (pendiente, riesgo R-08). Cada
  sobre guarda la referencia de la clave con que se cifró, para rotar sin parar el
  servicio: primero se cifra lo nuevo con la clave nueva y después se recifra lo
  antiguo por lotes.
- Claves distintas por población (externa e interna), por categoría (ordinaria y
  especial) y por función (cifrar e índice ciego). El portal externo solo puede usar
  las claves externas.
- Índice ciego con HMAC para buscar por DNI y detectar duplicados. Para buscar por
  apellidos, índice ciego del apellido normalizado completo, solo por igualdad. Los
  listados se descifran en la aplicación con permiso y finalidad.
- Además, cifrado del disco y de las copias.

Opción a valorar: una clave de datos por persona en el almacén de categorías
especiales. Si hay que suprimir, se destruye esa clave y el dato queda ilegible también
en las copias de seguridad («borrado criptográfico»). Cuesta más operaciones con el
gestor de claves. Se recomienda al menos para las categorías especiales (criterio
técnico, a validar por Seguridad).

### 5.3 Categorías especiales

- Discapacidad, adaptaciones y cualquier dato de salud van en un esquema propio de
  Aspirantes, con clave propia y un rol de lectura que solo tienen las finalidades
  «admisión por cupo», «exención de tasa» y «adaptación de pruebas».
- Se guarda lo mínimo: «grado igual o superior al 33 %, verificado el día D», no el
  grado exacto salvo que las bases lo usen, y nunca el diagnóstico.
- El tribunal ve la adaptación concedida («30 minutos más», «sala accesible»), no el
  grado ni el dictamen.
- No se reutiliza en otro proceso sin que la persona lo pida en ese proceso. VEC puede
  ofrecer «usar la verificación que ya consta», que es una acción suya (criterio
  técnico).
- Los datos del art. 10 RGPD (delitos sexuales) no pasan por Aspirantes. Van a Personal
  al incorporarse y solo se guarda «certificado negativo verificado el día D».

### 5.4 Vistas por finalidad y rol

Cada vista es una función con permiso propio, que devuelve solo los campos de su
finalidad y anota la lectura. La tabla es una propuesta (a confirmar por RRHH y el DPD,
preguntas 80 y 90).

| Quién | Qué ve | Qué no ve |
| --- | --- | --- |
| La propia persona (portal externo) | Toda su ficha, sus inscripciones, sus documentos y quién ha consultado sus datos (unidad y finalidad) | Datos de otros |
| RRHH, Selección | Ficha y méritos de los aspirantes de sus procesos; contacto para llamamientos | Categorías especiales, salvo para admitir por cupo; expediente laboral |
| Tribunal u órgano de selección (TREBEP art. 60) | En pruebas anónimas, el ejercicio con código, sin nombre. En méritos, los méritos y documentos del proceso. La adaptación concedida | Contacto, domicilio, grado de discapacidad, otras inscripciones |
| Centro solicitante (bolsa) | Nombre de la persona asignada y fecha de incorporación | Méritos, contacto, posición en la lista |
| Intervención | Lo que fiscaliza del expediente de contratación: la identidad de la persona nombrada y el acto | Méritos, contacto, categorías especiales |
| Personal (tras el traspaso) | Los campos del catálogo de traspaso | El resto del expediente de aspirante |
| Soporte técnico | Referencias y metadatos (duda 36) | Ningún dato en claro |
| Publicación | Listas con nombre y DNI parcial (apartado 5.9) | Todo lo demás |

### 5.5 Registro de accesos

- Toda lectura en claro de un dato personal deja constancia de actor, instante,
  finalidad, persona afectada (por referencia), campos leídos y resultado. Bolsa ya lo
  hace (registro de accesos T13) y es el patrón a seguir (criterio técnico; ENS, anexo
  II, registro de actividad, norma).
- La persona puede consultar qué unidades han accedido a sus datos, cuándo y para qué.
  El Tribunal de Justicia de la UE ha dicho que el derecho de acceso comprende fechas y
  finalidades de las consultas, pero no necesariamente el nombre de cada empleado que
  consultó (sentencia de 22 de junio de 2023, asunto C-579/21) (a confirmar por el DPD,
  pregunta 91).
- El registro de accesos contiene referencias, no datos en claro, y tiene su propio
  plazo de conservación (pregunta 84).

### 5.6 Historia de solo adición y rectificaciones

- Cada cambio de la ficha es una versión nueva, con motivo, actor y fecha. No se
  sobrescribe nada (criterio técnico ya asentado en VEC).
- Todo acto que usa un dato (admisión, baremación, llamamiento) guarda la versión y la
  huella del dato que usó. Así se sabe siempre con qué dato se decidió.
- Una rectificación posterior no cambia actos ya dictados. Si afecta a una baremación,
  VEC avisa a RRHH de qué procesos usaron la versión anterior y RRHH decide por la vía
  del procedimiento (apartado 6.3).

### 5.7 Derechos de las personas

| Derecho | Artículo RGPD | Cómo en VEC | Límites |
| --- | --- | --- | --- |
| Información | 13 y 14 | Primera capa en el formulario, segunda en la ayuda (LOPDGDD art. 11) | Textos aprobados por el DPD (pregunta 87) |
| Acceso | 15 | «Mis datos» en el portal y copia descargable (art. 15.3) | Datos de terceros y del tribunal, según el procedimiento |
| Rectificación | 16 | La persona corrige contacto y datos declarados. La identidad se corrige en origen (certificado, Registro Civil) | Lo ya baremado sigue su vía (apartado 6.3) |
| Supresión | 17 | Datos no ligados a un proceso: al momento | Durante un proceso o sus plazos de recurso, no (art. 17.3.b y 17.3.e) (apartado 6.2) |
| Limitación | 18 | Estado «limitado» mientras se resuelve una rectificación discutida | El dato se conserva pero no se usa |
| Portabilidad | 20 | No se aplica a tratamientos basados en el art. 6.1.e (art. 20.3). VEC ofrece de todos modos la copia estructurada del art. 15.3 | (a confirmar por el DPD) |
| Oposición | 21 | Cabe frente al art. 6.1.e. En un proceso selectivo equivale en la práctica a desistir (Ley 39/2015, art. 94) | (a confirmar por el DPD) |
| No ser objeto de decisiones automatizadas | 22 | La aplicación propone y RRHH o el tribunal confirman. Ninguna exclusión, orden o adjudicación es solo automática | Norma. Además, si algún día se usara inteligencia artificial, los sistemas de selección son de alto riesgo (Reglamento (UE) 2024/1689, art. 6.2 y anexo III, punto 4) |

### 5.8 Conservación y bloqueo

Tres fases por finalidad (criterio técnico sobre RGPD art. 5.1.e y LOPDGDD art. 32,
norma):

1. Uso activo: mientras dura la finalidad.
2. Bloqueo: el dato se reserva y solo puede ponerse a disposición de jueces, tribunales,
   Ministerio Fiscal y Administraciones competentes para atender responsabilidades,
   durante el plazo de prescripción (LOPDGDD art. 32.2, norma). En VEC, un dato
   bloqueado sale de todas las vistas y solo lo lee una función especial con doble
   control.
3. Fin: supresión, o transferencia al Archivo si la tabla de valoración dispone
   conservación permanente (Ley 7/2011 de Documentos, Archivos y Patrimonio Documental
   de Andalucía; RGPD art. 89 y LOPDGDD art. 26 para fines de archivo) (norma).

Plazos propuestos, todos provisionales y configurables por catálogo (a confirmar por el
DPD y el Archivo, pregunta 84; relacionada con las dudas 60 y 61):

| Dato | Uso activo | Bloqueo | Después |
| --- | --- | --- | --- |
| Inscripción no seleccionada | Hasta que el proceso sea firme: resolución final, dos meses para el recurso contencioso-administrativo (Ley 29/1998, art. 46) y el tiempo que duren los recursos | Hasta cuatro años desde la firmeza (plazo del recurso extraordinario de revisión, Ley 39/2015, art. 125.2) | Según la tabla de valoración: normalmente se eliminan las solicitudes y se conservan las listas y resoluciones |
| Participación en bolsa | Mientras la bolsa esté vigente y la persona no renuncie | Igual que el anterior, desde la derogación o la baja | Igual |
| Contacto de aviso | Mientras la persona lo mantenga | No se bloquea: se suprime el valor y se conserva la huella de la dirección usada en cada aviso como prueba | La prueba sigue el plazo del expediente |
| Discapacidad y adaptaciones | Solo el proceso en que se pidió | Mientras dure el plazo de recurso | Supresión, con borrado criptográfico |
| Registro de accesos | Dos años (propuesta) | No aplica | Supresión |
| Ficha de aspirante sin inscripciones vivas | Dos años sin actividad (propuesta), con aviso previo a la persona | No aplica si no hay procesos | Supresión |

Hasta que el DPD y el Archivo aprueben los plazos, VEC los guarda como anotación y no
bloquea ni borra nada de forma irreversible, igual que en la duda 61.

### 5.9 Seudonimización en listados y publicaciones

- Pantallas internas: los listados muestran el nombre completo solo a quien tiene
  finalidad y el DNI siempre parcial. La vista completa de un campo se pide con un clic
  que deja constancia.
- Publicación de actos: nombre y apellidos más cuatro cifras aleatorias del DNI (LOPDGDD
  disposición adicional 7.ª, apartado 1, norma). La orientación de la AEPD de 2019 fija
  las posiciones cuarta a séptima, `***1234**`, que es el formato que ya usa Bolsa (a
  confirmar por el DPD).
- Si la publicación sirve de notificación (Ley 39/2015, art. 44), solo el número de
  documento, sin nombre (LOPDGDD disposición adicional 7.ª, apartado 1, segundo párrafo,
  norma).
- No se publica junto al nombre ningún dato que revele salud: ni el turno de reserva por
  discapacidad ni la causa de exclusión si es de salud (a confirmar por el DPD, pregunta
  83).
- Las publicaciones en la web tienen fecha de retirada y no se ofrecen a buscadores
  (criterio técnico, pregunta 83).
- Hacia fuera del sistema, los eventos y la auditoría solo llevan referencias opacas
  (criterio técnico ya vigente).

### 5.10 Datos de prueba

- Todo desarrollo usa datos sintéticos con nombres verosímiles, sin tomar nada de
  CONVOCA hasta que haya visto bueno (regla ya vigente en VEC).
- Los DNI sintéticos tienen formato y letra válidos. Como cualquier número válido puede
  corresponder a una persona real, nunca se combinan con datos reales ni se publican
  fuera del entorno de desarrollo. Cada conjunto de prueba se marca como sintético.
- Las categorías especiales de prueba se generan en proporciones realistas, pero sin
  diagnósticos.
- Riesgo detectado al preparar este estudio: en equipos de desarrollo hay exportaciones
  reales de CONVOCA (hojas de cálculo con nombres y DNI enmascarados, y certificados de
  servicios en PDF). Para este estudio solo se han leído los nombres de las columnas.
  Esos ficheros son datos reales y deberían tratarse con las medidas que correspondan
  o retirarse de esos equipos (a confirmar por el DPD).

## 6. Casos difíciles

### 6.1 Trabajadora que vuelve a la bolsa al terminar su interinidad

Personal cierra la relación y emite el evento. Bolsa la repone según su regla. Su ficha
de aspirante no se tocó mientras trabajaba. Si su contacto cambió, ella decide si lo
traslada (apartado 4.5). Sus datos de nómina entran en bloqueo (apartado 4.6).

### 6.2 Petición de supresión con un proceso abierto

- Los datos necesarios para el proceso no se suprimen mientras siga abierto ni durante
  el plazo de recursos (RGPD art. 17.3.b y 17.3.e, norma). Si la persona quiere salir,
  puede desistir o renunciar (Ley 39/2015, art. 94). Desde ese momento sus datos pasan a
  bloqueo, no a uso activo.
- Lo que no está ligado a ningún proceso (preferencias, foto, contactos adicionales) se
  suprime en el momento.
- VEC responde en el plazo de un mes (RGPD art. 12.3, norma) indicando qué se suprime,
  qué se bloquea, hasta cuándo y por qué.

### 6.3 Rectificación de un dato que ya se usó para baremar

- La baremación se hace con los méritos acreditados a la fecha que fijan las bases. Si
  el dato era correcto en esa fecha y después cambió, no hay nada que rectificar en la
  baremación: se añade la versión nueva a la ficha.
- Si el dato era erróneo y afecta a la puntuación, el cauce es el del procedimiento:
  subsanación (Ley 39/2015, art. 68), reclamación contra las listas provisionales o
  recurso. La rectificación del RGPD no reabre por sí sola un acto administrativo (a
  confirmar por el DPD).
- VEC registra la versión nueva, avisa a RRHH de los procesos que usaron la anterior y
  no recalcula nada solo.

### 6.4 Antigua trabajadora que vuelve años después como aspirante

- Se inscribe en el portal externo como cualquier aspirante. Si no hay enlace, no se
  crea ninguno solo por inscribirse.
- Si quiere que puntúen sus servicios en la Diputación, lo declara. El registro de
  correspondencias localiza su `emp_` antiguo y Personal emite un certificado de
  servicios para ese proceso. Ella no tiene que aportarlo (Ley 39/2015, art. 28.2).
- Si parte de sus datos laborales antiguos ya están bloqueados, el certificado se basa
  en las relaciones de servicio, que se conservan, no en datos bloqueados.
- Si la nombran de nuevo, el traspaso reutiliza el mismo `emp_` (apartado 4.5).

### 6.5 Error en un traspaso

Ejemplo: se enlaza una aspirante con el trabajador equivocado.

- No se borra nada. Se añade una anulación del enlace, con motivo, y la hace una
  persona distinta de la que lo creó (doble control).
- Personal recibe un evento de corrección: anula el alta errónea con una versión nueva y
  crea la correcta.
- Todos los accesos hechos con el enlace erróneo quedan en el registro. Si alguien vio
  datos que no debía, es una brecha de seguridad y RRHH debe valorar su notificación a la
  AEPD en 72 horas y, en su caso, a las personas afectadas (RGPD arts. 33 y 34, norma).
- Prevención: el enlace por traspaso usa el índice ciego del DNI, no el nombre, y exige
  que coincidan identidad y acto.

### 6.6 RRHH ante aspirantes que son compañeros

- Solo ven a los aspirantes quienes tienen asignado ese proceso, y cada consulta queda
  anotada.
- Quien sea aspirante en un proceso no puede abrirlo desde el portal interno.
- Quien tenga relación personal con un aspirante debe abstenerse (Ley 40/2015, art. 23,
  norma). VEC permite registrar la abstención y retira el acceso a ese proceso.

### 6.7 Tribunales

- Sus miembros pueden ser de otras Administraciones. Actúan como órgano de selección
  (TREBEP art. 60), no como encargados del tratamiento (a confirmar por el DPD).
- Acceso temporal, limitado al proceso y a la fase, con identificación reforzada y
  compromiso de confidencialidad.
- En pruebas anónimas, VEC separa el código del ejercicio de la identidad hasta que se
  abren las plicas en acto formal.

### 6.8 Publicación de listas

Ver el apartado 5.9. La duda principal es cómo publicar el turno de discapacidad sin
revelar un dato de salud (pregunta 83).

## 7. Requisitos formales

### 7.1 Evaluación de impacto (EIPD)

Es obligatoria (norma): el RGPD la exige cuando es probable un alto riesgo (art. 35.1) y,
en particular, ante el tratamiento a gran escala de categorías especiales (art.
35.3.b). Aquí se dan varios de los criterios que usan las directrices del Comité
Europeo de Protección de Datos (WP248 rev.01) y la lista de la AEPD del art. 35.4:

- evaluación y puntuación de personas (baremación);
- categorías especiales (discapacidad);
- gran escala (miles de aspirantes por convocatoria);
- combinación de fuentes (Plataforma de Intermediación, CONVOCA, Personal);
- personas en situación de desequilibrio frente a la Administración;
- efecto en el acceso a un empleo.

La LOPDGDD añade como factores de mayor riesgo el tratamiento no incidental de
categorías especiales, la evaluación de aspectos personales y los grupos vulnerables,
en particular personas con discapacidad (art. 28.2) (norma).

Pasos: la hace el responsable del tratamiento, con el consejo del DPD (RGPD art. 35.2).
Si el riesgo residual sigue siendo alto, hay consulta previa a la AEPD (art. 36).
Propuesta: una EIPD para «Selección y bolsas» y otra para «Gestión de personal», que
coinciden con las dos poblaciones (a confirmar por el DPD, pregunta 85).

### 7.2 Registro de Actividades de Tratamiento

La Diputación publica su inventario de actividades (RGPD art. 30; LOPDGDD art. 31.2,
norma). Hay que revisar si las actividades existentes, como selección de personal,
bolsas de trabajo y gestión de personal, cubren lo que hará VEC: nuevas categorías de
datos, consultas a la Plataforma, registro de correspondencias, avisos de oportunidades
y plazos (pregunta 86).

### 7.3 Categorización ENS

- El ENS se aplica y además incluye las medidas para datos personales en el sector
  público (LOPDGDD disposición adicional 1.ª, norma).
- La categoría la fija el responsable de la información según el impacto en
  confidencialidad, integridad, trazabilidad, autenticidad y disponibilidad (ENS, arts.
  13 y 40 y anexo I, norma).
- Propuesta técnica: categoría MEDIA como mínimo. La confidencialidad del almacén de
  categorías especiales podría valorarse como ALTA según la guía CCN-STIC-803 (a
  confirmar por Seguridad, pregunta 89).
- La categorización decide también la opción C del apartado 4.2 y la custodia de claves.

### 7.4 Información a las personas

- Información en capas (LOPDGDD art. 11, norma): la primera, junto al botón de enviar la
  solicitud, con responsable, finalidad, derechos y dónde ampliar; la segunda, en la
  ayuda del botón «?», con todo lo que exige el art. 13 RGPD: base, destinatarios
  (tribunal, publicaciones, Plataforma), plazos, derecho a reclamar ante la AEPD y
  contacto del DPD.
- Un texto por finalidad (inscripción, bolsa, traspaso, avisos de talento), en catálogo
  i18n, versionado. Cada inscripción guarda la versión del texto que se mostró.
- Los textos los aprueba el DPD (pregunta 87).

### 7.5 Encargados del tratamiento

Posibles encargados (RGPD art. 28, norma; lista a confirmar, pregunta 88):

- proveedor del alojamiento de VEC en producción, si no es infraestructura propia de la
  Diputación;
- proveedor del correo saliente, si el SMTP corporativo lo presta un tercero;
- proveedor de SMS o de llamadas, si se usa para llamamientos;
- el actual proveedor de CONVOCA, durante la migración;
- la Plataforma de Intermediación y Cl@ve: la AGE actúa como intermediaria y el
  reparto de papeles lo fija su convenio de adhesión (a confirmar por el DPD).

Cada uno necesita contrato o convenio con las cláusulas del art. 28.3 RGPD.

## 8. Plan de implantación por fases

Alineado con la arquitectura de VEC: hexagonal, un dueño por dato, referencias opacas,
bandeja de salida y catálogos versionados. Ningún corte de este plan desplaza la
prioridad de Contratación temporal: cuando una tarea de Contratación necesite algo de
aquí, se abre la minitarea dependiente y se vuelve al camino crítico (AGENTS.md).

### Fase 0. Ahora, sin programar

- Este estudio, su revisión independiente y las preguntas 77 a 92 de `dudas.md`.
- Decisiones de dirección sobre el módulo Aspirantes, la opción B y el registro de
  correspondencias.

### Fase 1. Con datos sintéticos (se puede construir ya)

Por orden de dependencia:

1. Catálogo de campos por finalidad y momento: qué pide cada tipo de convocatoria, con
   obligatoriedad y momento. Versionado y modificable, con los valores provisionales
   del apartado 2 como paquete de ejemplo retirable.
2. Separar Usuarios por población (5.08a, 5.08b y 5.08c): esquemas, roles y claves.
3. Proyección del contexto de actor por portal (solo `asp_` fuera y solo `emp_`
   dentro). Necesita revisión de seguridad y SQL.
4. Módulo Aspirantes: ficha cifrada por campo, índices ciegos, historia, vistas por
   finalidad, registro de accesos y «Perfil y contacto» del área personal (teléfono,
   móvil, domicilio y código postal solo si el catálogo los pide).
5. Almacén de categorías especiales con clave propia, sin activar en la presentación.
6. «Mis datos»: copia descargable (art. 15.3) y lista de accesos a mis datos.
7. Bolsa: pasar a `asp_` y leer el contacto de Aspirantes.
8. Registro de correspondencias y traspaso por bandeja de salida desde Contratación
   temporal a Personal, con datos sintéticos.
9. Conservación como anotación, sin bloqueo irreversible.
10. Generador de datos sintéticos para todo lo anterior.

Todo con las puertas habituales: pruebas focales, PostgreSQL efímero con roles y ACL
reales, revisión SQL independiente y revisión de seguridad antes de fusionar.

### Fase 2. Tras la EIPD y el visto bueno del DPD

- Activar las categorías especiales.
- Integrar la Plataforma de Intermediación.
- Gestor de claves o HSM real y rotación probada.
- Base de datos separada para Aspirantes en producción (opción C).
- Textos de información aprobados.
- Reglas de publicación aprobadas.
- Plazos de conservación con bloqueo real.
- Actualización del Registro de Actividades.

### Fase 3. Datos reales

- Carga de las bolsas de CONVOCA en Aspirantes (dudas 16 y 45).
- Contratos con encargados firmados.
- Formación de RRHH y de tribunales en las vistas por finalidad.
- Revisión periódica del registro de accesos.

## 9. Riesgos y límites de este estudio

- Es un estudio técnico. Las marcas «(a confirmar por el DPD)» no son asesoramiento
  jurídico cerrado.
- Las bases concretas de la Diputación no están en el repositorio. El inventario del
  apartado 2 debe contrastarse con ellas (pregunta 80).
- La disponibilidad de cada servicio de la Plataforma de Intermediación depende de la
  adhesión de la Diputación.
- Cambiar el contexto de actor y las claves de Usuarios toca piezas ya desplegadas.
  Hay que hacerlo en minitareas pequeñas, con ensayo sobre el clon de la principal.
- Mientras no exista un gestor de claves real, todo el cifrado descrito protege frente
  a un volcado de la base, pero no frente a un compromiso del servidor de aplicación.

## 10. Preguntas añadidas a `dudas.md`

Se han añadido las preguntas 77 a 92: separación y base de datos externa (77),
registro de correspondencias (78), campos del traspaso (79), datos por momento (80),
Plataforma de Intermediación (81), discapacidad y adaptaciones (82), publicación de
listas (83), plazos de conservación (84), EIPD (85), Registro de Actividades (86),
textos de información (87), encargados (88), ENS y claves (89), RRHH y tribunales ante
compañeros (90), registro de accesos (91) y búsqueda de talento interno (92).

## Anexo. Normas citadas

- Reglamento (UE) 2016/679 (RGPD): considerando 43; arts. 4.5, 5, 6, 9, 10, 12 a 22,
  25, 28, 30, 32 a 36 y 89.
- Ley Orgánica 3/2018 (LOPDGDD): arts. 8, 9, 11, 26, 28.2, 31, 32; disposiciones
  adicionales 1.ª y 7.ª.
- Ley 39/2015: arts. 5, 9, 11, 14, 28, 41, 44, 45, 53, 66, 68, 94 y 125.
- Ley 40/2015: arts. 23, 24 y 155.
- Real Decreto Legislativo 5/2015 (TREBEP): arts. 10, 11, 14, 16, 55 a 63, 78 a 84;
  disposición adicional 17.ª.
- Ley 7/1985, de Bases del Régimen Local: art. 91.2.
- Real Decreto 896/1991, sobre selección de funcionarios de Administración Local.
- Real Decreto 2271/2004, sobre acceso al empleo público de personas con discapacidad:
  arts. 3 y 8.
- Real Decreto Legislativo 1/2013, Ley General de derechos de las personas con
  discapacidad.
- Ley Orgánica 1/1996, de Protección Jurídica del Menor: art. 13.5.
- Ley Orgánica 3/2007, de igualdad efectiva de mujeres y hombres: art. 20.
- Ley 31/1995, de Prevención de Riesgos Laborales: art. 22.4.
- Ley 53/1984, de incompatibilidades del personal al servicio de las Administraciones
  Públicas.
- Estatuto de los Trabajadores: art. 15.5.
- Ley 58/2003, General Tributaria: art. 66. Real Decreto Legislativo 8/2015, Ley General
  de la Seguridad Social: art. 24.
- Ley 29/1998, de la Jurisdicción Contencioso-administrativa: art. 46.
- Ley 7/2011, de Documentos, Archivos y Patrimonio Documental de Andalucía.
- Real Decreto 311/2022 (ENS): arts. 13 y 40; anexos I y II.
- Reglamento (UE) 2024/1689 de inteligencia artificial: art. 6.2 y anexo III, punto 4.
- Sentencia del Tribunal de Justicia de la UE de 22 de junio de 2023, asunto C-579/21.
- Comité Europeo de Protección de Datos, directrices WP248 rev.01 sobre EIPD; lista de
  la AEPD de tratamientos que requieren EIPD (art. 35.4 RGPD); orientación de la AEPD
  sobre la disposición adicional 7.ª de la LOPDGDD.
