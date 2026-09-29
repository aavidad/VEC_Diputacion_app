# Datos personales de las inscripciones: qué guardar, dónde y con qué garantías

Fecha: 29 de septiembre de 2026 (segunda versión, tras revisión independiente).
Estado: estudio sin programar. Pendiente del Delegado de Protección de Datos (DPD), de
RRHH y de Seguridad. Nada de lo que aquí se propone autoriza a tratar datos reales.

Petición de Alberto: en el área personal hacen falta, además del correo, el teléfono,
el móvil, el domicilio, el código postal y el resto de datos que exijan las
inscripciones en bolsas, procesos selectivos y provisión, porque VEC sustituirá a
CONVOCA. Hay que estudiar cómo guardarlos cumpliendo la protección de datos. Durante el
estudio Alberto añadió cuatro condiciones:

- los datos de las personas internas (trabajadores de la Diputación) y los de las
  externas (aspirantes) no se mezclan;
- una persona puede pasar de un lado a otro en cualquier momento, o estar en los dos a
  la vez, y eso debe funcionar sin mezclar;
- el paso a plaza fija (proceso selectivo o provisión) se trata con el mismo mecanismo
  que el paso por bolsa, con sus diferencias;
- cada decisión debe mostrar sus alternativas, sus riesgos y por qué se elige, y
  distinguir lo que manda la norma de lo que es criterio técnico y de lo que decide el
  DPD.

## Cómo leer este estudio

Cada afirmación relevante lleva una de estas marcas:

- «(norma)»: lo exige una ley o un reglamento; se cita el artículo.
- «(criterio técnico)»: es una propuesta de diseño. Puede cambiarse sin infringir nada,
  aunque cambiarla tiene las consecuencias que se explican.
- «(a confirmar por el DPD)» o «(a confirmar por Seguridad)»: es una interpretación
  jurídica o una decisión que corresponde al responsable del tratamiento. Mientras no se
  confirme, VEC la trata como valor provisional y configurable.

Las normas se citan por su nombre corto: RGPD (Reglamento (UE) 2016/679), LOPDGDD (Ley
Orgánica 3/2018), Ley 39/2015 (Procedimiento Administrativo Común), Ley 40/2015
(Régimen Jurídico del Sector Público), TREBEP (Real Decreto Legislativo 5/2015), ENS
(Real Decreto 311/2022). Al final hay una lista con todas las citas.

## Resumen de la propuesta

1. Cada población tiene su dueño. Los datos de aspirantes los posee un módulo nuevo,
   Aspirantes; los de trabajadores, Personal. Bolsa, Procesos selectivos y Provisión
   guardan referencias y piden los datos por finalidad.
2. Cada población tiene su propio identificador opaco: `asp_` para aspirante y `emp_`
   para trabajador. Ninguno se deriva del otro.
3. La separación empieza por lo que protege de verdad hoy: procesos, credenciales de
   base de datos y material de claves distintos para el portal externo y el interno.
   Después vienen esquemas propios por población y, si la categorización ENS lo pide,
   una base de datos aparte para lo externo.
4. La relación entre la persona aspirante y la persona trabajadora se guarda aparte, en
   un registro de correspondencias que solo existe en la zona interna. Solo se crea por
   un acto administrativo (nombramiento, contrato) o por declaración de la propia
   persona cuando un trámite lo necesita, y solo se consulta para una lista cerrada de
   finalidades, con registro de cada consulta.
5. El paso de aspirante a trabajador es un traspaso gobernado: lo dispara la
   resolución o el contrato, viaja por la bandeja de salida (outbox) y lleva solo los
   datos que exige la relación laboral. El expediente de aspirante no se copia.
6. Cada dato se pide cuando hace falta y no antes. Lo que ya tiene una Administración se
   consulta por la Plataforma de Intermediación de Datos; la consulta se presume
   autorizada y la persona puede oponerse (Ley 39/2015, art. 28.2).
7. Cifrado por campo en la aplicación. La discapacidad, las adaptaciones y la condición
   de víctima de violencia van en un almacén aparte, con acceso más restringido.
8. Nada se suprime de un expediente sin valoración documental. Los plazos se fijan por
   serie documental, con el Archivo.
9. Antes de tratar datos reales hacen falta la evaluación de impacto (EIPD), la
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
  una clave secreta a partir del documento enmascarado (formato `***1234**`) y del
  nombre normalizado, porque CONVOCA solo exporta el documento enmascarado
  (`internal/modules/bolsa/application/constitucion/candidato.go`). La unión con una
  identidad real se produce después, cuando la persona entra con certificado y VEC
  calcula la misma referencia a partir de su documento completo.
- En la importación, `DerivarFilasVinculo`
  (`internal/modules/bolsa/application/constitucion/servicio.go`) calcula la referencia
  de cada fila sin comprobar si dos filas dan la misma. Si dos personas comparten nombre
  y las cuatro cifras visibles del documento, hoy quedarían unidas sin aviso.
- La importación de CONVOCA guarda las filas en una zona de preparación cifrada.
- La proyección pública de Bolsa no incluye DNI, contacto ni expediente.
- El correo obligatorio del alta (`contacto_usuario_vec`) está cifrado con una subclave
  propia.
- Usuarios guarda «Mis preferencias» (5.08a), «Mis correos» (5.08b, cifrados, con
  verificación por código) y la foto (5.08c). Sus tablas usan como clave la persona VEC
  (`per_`). La superficie de acceso (`interna_corporativa` o `externa_personal`)
  aparece en el contexto de cada transacción y en los filtros por fila, y ya hay roles
  de ejecución distintos por superficie (`vec_usuarios_ejecutor_interno` y
  `vec_usuarios_ejecutor_externo`). Si la misma persona entra por los dos portales,
  comparte preferencias, correos y foto.
- El contexto de actor admite vínculos de tipo `candidato` y `empleado`. Ya existe un
  alcance de proyecciones (`internal/vec/domain/contexto_actor_alcance.go`): el vínculo
  de empleado gobernado lo aporta Personal solo cuando la composición lo pide
  expresamente, y se distingue de los vínculos de empleado heredados del núcleo. Lo que
  falta es impedir que el vínculo de candidato aparezca en el portal interno y retirar
  los vínculos de empleado heredados.
- Personal ofrece la ficha propia del empleado (puesto, relación, organización) sin
  datos de contacto.
- Las claves de cifrado de desarrollo se derivan dentro del proceso. No hay todavía
  HSM ni gestor de claves (riesgo R-08 del informe del comité de seguridad).

El cifrado y el enmascarado siguen un buen patrón. La separación entre internos y
externos es lógica y está a medio hacer en el contexto de actor. Los apartados 4 y 5
proponen cómo completarla.

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
| Documento de identidad: tipo (DNI, NIE, pasaporte, documento de otro Estado), país emisor y número | Sí, para identificar y publicar de forma seudonimizada | Inscripción | Certificado o Cl@ve para DNI y NIE; la persona para los demás |
| Nacionalidad | Sí, es requisito (TREBEP art. 56.1.a y art. 57) | Inscripción, como declaración | Declaración; comprobación en admisión |
| Vínculo familiar con un nacional de la UE | Solo si la persona accede como familiar (TREBEP art. 57.2) | Inscripción | La persona, con acreditación del vínculo |
| Fecha de nacimiento | Para comprobar la edad (TREBEP art. 56.1.c) y saber si es menor de 18 | Inscripción | Consulta de identidad por la Plataforma de Intermediación (disponibilidad a confirmar) |
| Sexo | No es requisito. Puede hacer falta para estadística (Ley Orgánica 3/2007, art. 20) | Inscripción, voluntario | La persona (a confirmar por el DPD) |
| Firma o identificación electrónica | Sí (Ley 39/2015, arts. 9 y 11) | Inscripción | Certificado o Cl@ve |

#### Contacto y domicilio

| Dato | ¿Necesario? | Momento | Fuente |
| --- | --- | --- | --- |
| Correo electrónico | En los procesos, sirve para los avisos de puesta a disposición de notificaciones, que la persona aporta de forma voluntaria (Ley 39/2015, arts. 41.6 y 66.1.b). En bolsa es un requisito del llamamiento si así lo fijan el reglamento o las bases | Inscripción | La persona, verificado |
| Teléfono y móvil | En bolsa, requisito para el llamamiento según el reglamento o las bases (a confirmar por RRHH, preguntas 14 y 80). En los procesos, voluntario, como el correo | Inscripción en bolsa | La persona |
| Domicilio y código postal | Solo si la persona no está obligada a relacionarse electrónicamente y elige notificación en papel (Ley 39/2015, arts. 14 y 66.1.b). No hace falta para inscribirse por medios electrónicos | Inscripción, solo en ese caso | La persona |
| Domicilio a efectos laborales y fiscales | Sí, para el contrato, la Seguridad Social y la nómina | Nombramiento o contrato | La persona, ya en Personal |

El domicilio es el dato cuya petición más conviene aplazar (criterio técnico). Si las
bases obligan a inscribirse por vía electrónica, bastan el correo y el teléfono hasta el
nombramiento.

#### Académicos y méritos

| Dato | ¿Necesario? | Momento | Fuente preferente |
| --- | --- | --- | --- |
| Titulación exigida | Sí (TREBEP art. 56.1.e) | Declaración en la inscripción; comprobación en admisión o en el hito que fijen las bases (duda 37) | Consulta de titulaciones por la Plataforma de Intermediación; si no está disponible, la persona |
| Servicios prestados en la Diputación | Solo si puntúan | Baremación | Personal (la Diputación ya los tiene: Ley 39/2015, art. 28.2) |
| Servicios en otras Administraciones o empresas | Solo si puntúan | Baremación | Vida laboral por la Plataforma de Intermediación y certificados de la otra Administración |
| Formación recibida o impartida | Solo si puntúa | Baremación | La persona |
| Titulaciones superiores o idiomas | Solo si puntúan o son requisito | Baremación | Plataforma de Intermediación o la persona |
| Carné de conducir u otras habilitaciones | Solo si las bases lo exigen para la categoría | Admisión | Consulta a la DGT por la Plataforma (disponibilidad a confirmar) o la persona |

#### Situación laboral

| Dato | ¿Necesario? | Momento | Fuente |
| --- | --- | --- | --- |
| Ser personal de la Diputación | Solo en promoción interna. Ese trámite lo hace la persona como empleada, desde el portal interno (apartado 4.8) | Inscripción | Personal |
| Situación de desempleo | Solo si da derecho a exención o reducción de la tasa según la ordenanza fiscal | Pago de la tasa | Consulta al SEPE por la Plataforma (disponibilidad a confirmar) |
| Declaración de no haber sido separado ni inhabilitado | Sí (TREBEP art. 56.1.d) | Inscripción, como declaración responsable; comprobación antes del nombramiento | La persona |
| Compatibilidad con otro empleo | No en la inscripción. Sí antes de la toma de posesión o del contrato (Ley 53/1984) | Nombramiento o contrato | La persona |

#### Salud, discapacidad y adaptaciones (categoría especial, RGPD art. 9)

| Dato | ¿Necesario? | Momento | Fuente preferente |
| --- | --- | --- | --- |
| Discapacidad igual o superior al 33 % | Solo si la persona opta al cupo de reserva (TREBEP art. 59.1; Real Decreto 2271/2004, art. 3) o pide exención de tasa | Inscripción | Consulta de discapacidad por la Plataforma de Intermediación; si no, certificado |
| Discapacidad intelectual | Solo si la convocatoria reserva plazas a ese cupo (TREBEP art. 59.1 lo fija en el 2 %). Revela el tipo de discapacidad, no solo el grado | Inscripción | Certificado o dictamen, según las bases |
| Solicitud de adaptación de tiempos o medios | Solo si la persona la pide (TREBEP art. 59.2; Real Decreto 2271/2004, art. 8) | Inscripción | La persona |
| Dictamen técnico sobre la adaptación | Solo si el tribunal lo necesita para resolverla | Antes de las pruebas | La persona o el órgano competente |
| Diagnóstico o causa de la discapacidad | No. Nunca hace falta | Nunca | No se pide |
| Capacidad funcional para el puesto | Sí (TREBEP art. 56.1.b). Cuando las bases prevén reconocimiento médico previo, RRHH recibe solo «apto» o «no apto» (Ley 31/1995, art. 22.4) | Antes del nombramiento o la incorporación | Vigilancia de la salud |

#### Cupos, turnos y situaciones protegidas

| Dato | ¿Necesario? | Momento | Nota |
| --- | --- | --- | --- |
| Turno (libre, reserva por discapacidad, reserva por discapacidad intelectual, promoción interna) | Sí, si la convocatoria tiene varios | Inscripción | Los turnos de reserva revelan un dato de salud: no pueden aparecer junto al nombre en listas públicas sin decisión del DPD (pregunta 83) |
| Condición de víctima de violencia de género o sexual | Solo si la persona pide protección de sus datos o si las bases o la ordenanza fiscal prevén algún efecto | Cuando la persona lo pida | Activa una marca de protección (apartado 5.4) |
| Otras exenciones de tasa (víctimas de terrorismo, familia numerosa) | Solo si la ordenanza fiscal las prevé | Pago de la tasa | RRHH confirma cuáles existen (pregunta 80) |

#### Menores de edad

La edad mínima es 16 años (TREBEP art. 56.1.c, norma). Una persona de 16 o 17 años
puede inscribirse en bolsas o procesos de personal laboral, y para firmar el contrato
necesita la autorización de sus padres o tutores (Estatuto de los Trabajadores, art.
7.b, norma). Si la relación es funcionarial, RRHH y el DPD deben confirmar si se exige
algo parecido (pregunta 94). Consecuencias:

- VEC debe saber si la persona es menor, a partir de la fecha de nacimiento.
- Los datos del padre, madre o tutor (nombre, documento, acreditación de la
  representación) son datos de terceros y solo se piden al contratar.
- La información del art. 13 RGPD debe ser comprensible para un menor (RGPD art. 12.1,
  norma).
- Los menores son un grupo vulnerable a efectos de la EIPD (LOPDGDD art. 28.2.e,
  norma).

#### Datos de terceros

| Dato | ¿Necesario? | Momento | Nota |
| --- | --- | --- | --- |
| Representante: nombre, documento y acreditación de la representación | Solo si actúa un representante (Ley 39/2015, art. 5) | Inscripción | El representante es otro interesado a efectos del RGPD |
| Familiar que da derecho al acceso (cónyuge o ascendiente nacional de la UE) | Solo en el caso del TREBEP art. 57.2 | Inscripción | Se guarda la acreditación del vínculo, no más datos del familiar |
| Padre, madre o tutor de un menor | Solo al contratar a un menor | Contrato | Estatuto de los Trabajadores, art. 7.b |
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

El certificado de persona física, el DNIe y Cl@ve entregan nombre, apellidos y número de
DNI o NIE. Las personas que entran con la identificación electrónica de otro Estado de
la UE (eIDAS) aportan nombre, apellidos, fecha de nacimiento y un identificador de su
país, que no es un NIE (a confirmar con Informática según la versión de Cl@ve). Nadie
aporta por esta vía domicilio, teléfono ni nacionalidad fiable. Por tanto:

- VEC no pide escribir el nombre ni el DNI o NIE: los toma de la identificación y la
  persona solo los ve (criterio técnico; reduce errores y suplantaciones).
- El documento se guarda con su tipo, su país y su número, y con historia. Una persona
  que entra con NIE y se nacionaliza pasa a tener DNI. Las dos entradas se unen en la
  misma ficha mediante una fusión gobernada: la pide la persona o RRHH, la aprueba una
  segunda persona, y ninguna de las dos fichas se borra (apartado 6.9).
- El resto se declara o, mejor, se consulta.

## 3. Base jurídica por finalidad

El consentimiento no sirve como base general en un proceso de selección público: la
persona no puede negarse sin perder la opción al puesto, y el RGPD advierte de ese
desequilibrio con las autoridades públicas (considerando 43) (norma). Las bases son la
obligación legal y el ejercicio de poderes públicos, que exigen una norma con rango de
ley (RGPD art. 6.3; LOPDGDD art. 8) (norma).

| Finalidad | Base (RGPD) | Norma que la habilita | Marca |
| --- | --- | --- | --- |
| Gestionar inscripciones, admisión y baremación de procesos selectivos | Art. 6.1.e | TREBEP arts. 55, 56 y 61; Ley 7/1985, art. 91.2; Real Decreto 896/1991 | (norma) |
| Gestionar bolsas y llamamientos | Art. 6.1.e | TREBEP art. 10.2 (interinos) y art. 11 (laborales); reglamento de bolsas de la Diputación | (a confirmar por el DPD: el reglamento de bolsas no tiene rango de ley y la base descansa en el TREBEP) |
| Provisión de puestos | Art. 6.1.e | TREBEP arts. 78 a 84 | (norma) |
| Cupos de discapacidad y adaptaciones | Art. 9.2.b o 9.2.g, con art. 6.1.e | TREBEP art. 59; Real Decreto Legislativo 1/2013; Real Decreto 2271/2004; LOPDGDD art. 9.2 | (a confirmar por el DPD cuál de las dos letras del art. 9.2 aplica) |
| Protección de víctimas de violencia de género o sexual | Art. 6.1.c y art. 9.2.g si hay datos de salud | TREBEP art. 82; LOPDGDD disposición adicional 7.ª, apartado 2 | (a confirmar por el DPD) |
| Avisos por correo y teléfono en procesos | Art. 6.1.e | Ley 39/2015, art. 41.6; el dato lo aporta la persona voluntariamente | (norma) |
| Llamamientos de bolsa por teléfono y correo | Art. 6.1.e | Reglamento de bolsas y bases | (a confirmar por el DPD) |
| Publicar listas | Art. 6.1.c y 6.1.e | Ley 39/2015, art. 45; bases; LOPDGDD disposición adicional 7.ª | (norma) |
| Consultar datos a otras Administraciones | Art. 6.1.e | Ley 39/2015, art. 28.2; Ley 40/2015, art. 155 | (norma) |
| Alta por nombramiento de funcionario | Art. 6.1.c y 6.1.e | TREBEP arts. 10 y 62 | (norma) |
| Alta por contrato laboral | Art. 6.1.b y 6.1.c | Estatuto de los Trabajadores; TREBEP art. 11 | (norma) |
| Registro de correspondencias aspirante-trabajador | Art. 6.1.c y 6.1.e | Ley 53/1984 (incompatibilidades); Estatuto de los Trabajadores, art. 15.5 (encadenamiento); TREBEP art. 10 y disposición adicional 17.ª; certificación de servicios | (a confirmar por el DPD) |
| Avisos de oportunidades a trabajadores («búsqueda de talento interno») | Art. 6.1.e, con prueba de compatibilidad del art. 6.4 y posibilidad de darse de baja | TREBEP arts. 14.c y 16 (carrera profesional) | (a confirmar por el DPD, pregunta 92) |
| Registro de accesos | Art. 6.1.c | RGPD art. 32; ENS, anexo II | (norma) |
| Foto voluntaria en el portal | Art. 6.1.a | Consentimiento, retirable en cualquier momento. Aquí sí cabe, porque negarse no tiene consecuencias | (a confirmar por el DPD, ya planteado en la duda 76) |

### 3.1 No pedir lo que ya tiene la Administración

La Ley 39/2015 reconoce el derecho a no aportar documentos elaborados por cualquier
Administración (art. 28.2) y a no presentar datos que ya estén en su poder (art.
53.1.d). En la redacción que le dio la LOPDGDD (disposición final 12.ª), el art. 28.2
presume que la consulta está autorizada salvo que conste la oposición expresa de la
persona o una ley especial exija su consentimiento. La Administración debe obtener los
datos por sus redes o por la Plataforma de Intermediación de Datos (norma).

Una excepción importante: los datos tributarios solo se ceden a otra Administración con
autorización previa de la persona (Ley 58/2003, General Tributaria, art. 95.1.k)
(norma). Si alguna exención o requisito depende de la renta, esa consulta sí necesita una
autorización expresa.

Servicios de la Plataforma que encajan con las inscripciones (disponibilidad y adhesión
de la Diputación a confirmar por Informática, pregunta 81):

| Dato | Servicio |
| --- | --- |
| Identidad | Verificación de datos de identidad |
| Titulación universitaria y no universitaria | Consulta de títulos |
| Discapacidad | Consulta de grado de discapacidad |
| Situación de desempleo | Consulta de inscripción como demandante de empleo |
| Vida laboral | Consulta de vida laboral de la Tesorería General de la Seguridad Social |
| Familia numerosa | Consulta de título de familia numerosa |
| Delitos sexuales | Consulta de inexistencia de antecedentes |

Consecuencias de diseño (criterio técnico sobre la norma anterior):

- La inscripción informa de que VEC consultará esos datos y ofrece una casilla «Me opongo
  a la consulta de…» por cada servicio, desmarcada. Si la persona la marca, se le pide el
  documento correspondiente.
- Para datos tributarios, la casilla es la contraria: «Autorizo la consulta», también
  desmarcada, porque ahí la ley exige autorización.
- VEC guarda el resultado de la consulta como hecho verificado («titulación verificada
  el día D, consulta con referencia R»), no el documento. Así se guarda menos y el dato
  es más fiable.

La consulta por la Plataforma es una transmisión entre Administraciones. La persona debe
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

### 4.2 Opciones de separación

Opción A, la actual: mismas tablas, columna de superficie, roles por superficie y
filtros por fila.

- A favor: ya existe y funciona en Usuarios.
- En contra: un error en una política de fila o en una función con privilegios de
  propietario cruza poblaciones. Copias, réplicas y consultas de mantenimiento ven a las
  dos juntas. La clave `per_` une a la persona en ambos portales.

Opción B: esquemas propios por población en la misma instancia, con roles sin permisos
cruzados.

- A favor: un error en un esquema no da acceso al otro. Coste bajo, porque VEC ya separa
  esquemas y roles por módulo.
- En contra: el administrador de la base, las copias y el sistema operativo siguen
  siendo comunes.

Opción C: base de datos o instancia aparte para la parte externa.

- A favor: la zona expuesta a internet solo alcanza su propia base. Copias, réplicas y
  administración se gobiernan por separado. Encaja con la separación por zonas del
  informe del comité (riesgo R-06).
- En contra: más coste de operación. RRHH necesita leer datos de aspirantes desde la zona
  interna, así que la aplicación interna tendrá conexión a las dos bases, con roles de
  lectura por finalidad. El flujo va solo de dentro hacia fuera: la parte externa nunca
  tiene credenciales de la interna.

Cualquiera de las tres opciones es inútil si el mismo proceso atiende a los dos portales
con las mismas credenciales y el mismo material de claves: quien comprometa ese proceso
lo tiene todo. Lo mismo pasa con «una clave por población» mientras no haya gestor de
claves, porque hoy todas las claves se derivan dentro del proceso.

Recomendación, por orden (criterio técnico):

1. Ya: procesos distintos para el portal externo y el interno, cada uno con sus propias
   credenciales de base de datos y su propio material de claves. Los roles de ejecución
   interno y externo de Usuarios ya existen y marcan el camino.
2. Después: esquemas propios por población en los almacenes con datos personales (opción
   B), sin claves foráneas ni consultas que crucen de un esquema de población al otro,
   para que pasar a C sea un cambio de despliegue.
3. La opción C la decide la categorización ENS (a confirmar por Seguridad, pregunta 89).

### 4.3 Identificadores

| Opción | Descripción | Riesgo |
| --- | --- | --- |
| Un único identificador de persona (`per_`) para todo | Lo que hay hoy en Usuarios | Cualquier tabla puede cruzarse con cualquier otra |
| Identificadores derivados del documento con la misma clave en los dos lados | Cada almacén calcula su referencia con HMAC del DNI | Quien tenga la clave o ambas bases puede unirlas |
| Identificadores aleatorios distintos por almacén y enlace aparte | `asp_` en Aspirantes, `emp_` en Personal; la relación vive en un registro propio | Hay que mantener ese registro y gobernar su uso. Es lo que se recomienda |

Recomendación (criterio técnico):

- Aspirantes genera `asp_` al azar y guarda el documento cifrado, con un índice ciego
  (HMAC con clave propia) para buscar por documento y detectar inscripciones duplicadas.
- Personal genera o mantiene `emp_` con su propio índice ciego y su propia clave.
- Contexto de actor: el portal interno nunca recibe el vínculo de candidato; el externo
  no pide la proyección de empleado. Los vínculos de empleado heredados del núcleo se
  retiran a favor de la proyección gobernada de Personal, que ya existe.
- Usuarios: para preferencias y foto basta con añadir la superficie a la clave. Un
  esquema por población solo compensa en los correos, que son datos de contacto.
- La referencia `can_` de Bolsa se sustituye por `asp_` con una migración cuidadosa
  (apartado 4.10).

### 4.4 Registro de correspondencias

Hace falta saber, en ciertos casos y solo en ellos, que una aspirante es o fue
trabajadora:

- para darle de alta en Personal cuando se la nombra o contrata;
- para certificar los servicios prestados en la Diputación cuando puntúan, sin pedirle
  un certificado que la Diputación ya tiene (Ley 39/2015, art. 28.2);
- para controlar el encadenamiento de contratos (Estatuto de los Trabajadores, art.
  15.5) y la duración de las interinidades (TREBEP art. 10 y disposición adicional
  17.ª);
- para las incompatibilidades (Ley 53/1984).

La promoción interna no está en la lista: la hace la persona como empleada, desde el
portal interno (apartado 4.8).

Alternativas:

1. Sin enlace. Los traspasos y los méritos internos serían imposibles o exigirían pedir a
   la persona certificados que la Diputación ya tiene.
2. Enlace automático al iniciar sesión con el mismo certificado en los dos portales.
   Cualquier trabajador que se inscriba quedaría unido sin una finalidad concreta.
3. Enlace gobernado en un registro aparte. Es la opción recomendada.

Diseño del registro (criterio técnico; finalidades a confirmar por el DPD, pregunta 78):

- Vive en la zona interna y lo posee Personal, porque la mayoría de los enlaces nacen de
  un acto de personal. El portal externo no tiene acceso.
- Cada fila guarda `asp_`, `emp_`, origen (nombramiento, contrato, declaración de la
  persona), referencia del acto, base jurídica, fecha, actor y estado. Es de solo
  adición: una anulación es una fila nueva.
- Se crea de dos maneras. Por traspaso, cuando lo dispara un acto de nombramiento o
  contrato (apartado 4.5). O por declaración de la persona en un trámite que lo
  necesita, como pedir que se tengan en cuenta sus servicios en la Diputación: el portal
  externo envía el documento acreditado por el certificado en un sobre cifrado para el
  registro, que calcula el índice ciego con la clave de Personal y localiza `emp_`. El
  portal externo nunca conoce `emp_`.
- Se consulta solo para la lista cerrada de finalidades, con permiso propio y registro
  de cada consulta.
- La respuesta es la mínima, por ejemplo «servicios certificables: N días en la
  categoría X». Nunca el expediente.

### 4.5 Traspasos gobernados

Mecanismo común (criterio técnico, alineado con la arquitectura de VEC):

1. El módulo que dicta el acto (Contratación temporal o Procesos selectivos) confirma el
   nombramiento o el contrato y, en la misma transacción, escribe el evento en su bandeja
   de salida con la referencia del acto, `asp_` y la base jurídica.
2. Un proceso de traspaso de la zona interna recibe el evento y pide a Aspirantes, por un
   puerto autorizado y con la finalidad «alta por nombramiento», solo los campos del
   catálogo de traspaso: identidad (con el historial de documentos) y contacto.
   Aspirantes los entrega en un sobre cifrado para Personal y anota la entrega.
3. Personal busca en el registro de correspondencias un `emp_` anterior. Si lo hay (la
   persona ya trabajó aquí), añade una relación a ese mismo empleado. Si no, crea uno. En
   ambos casos escribe el enlace con el acto, la base y la lista y huella de los campos
   recibidos.
4. Todo es idempotente por la referencia del acto: repetir el evento no crea otro alta.
5. Los datos propios de la relación laboral (IBAN, afiliación, IRPF, aptitud médica,
   compatibilidad, autorización del tutor si es menor) no pasan por Aspirantes. La
   persona los aporta en un trámite de incorporación. Como quizá aún no tiene cuenta
   corporativa, ese trámite puede presentarse desde el portal externo, pero los datos
   viajan cifrados para Personal y se guardan solo allí.
6. El expediente de aspirante (inscripción, méritos, notas, listas) no se copia.
7. Si la persona tiene activa una marca de protección como víctima de violencia, la
   marca viaja con el traspaso, porque Personal debe aplicar la misma reserva.

| Recorrido | Acto que dispara | Relación | Qué queda en Aspirantes | Al terminar |
| --- | --- | --- | --- | --- |
| Bolsa → interinidad o contrato temporal | Resolución de nombramiento interino o contrato temporal (Contratación temporal) | Temporal | La participación en la bolsa sigue viva, en estado «trabajando» | Evento de fin de relación: Bolsa aplica su regla de reposición. La persona sigue siendo aspirante |
| Proceso selectivo → funcionario de carrera | Nombramiento publicado y toma de posesión (TREBEP art. 62.1) | Sin fin previsto | El expediente del proceso, con su plazo de conservación. La ficha sigue disponible por si se presenta a otro proceso en turno libre | Si pierde la condición (TREBEP art. 63), Personal cierra la relación. No se genera nada en Aspirantes |
| Proceso selectivo → laboral fijo | Firma del contrato indefinido | Sin fin previsto | Igual que el anterior | Igual |
| Provisión de puestos | Resolución del concurso o de la libre designación | Cambia el puesto | Los participantes de la Diputación son empleados: la solicitud vive en la zona interna con `emp_`. Los de otras Administraciones entran como aspirantes y, si obtienen el puesto, siguen el traspaso | Según el caso |

En la bolsa se espera que la relación termine y la persona vuelva a la lista. En la plaza
fija, no: el enlace se mantiene y la ficha de aspirante puede quedarse sin uso.

Fin de interinidad o de contrato temporal: Personal registra el cese (TREBEP art. 10.3)
y emite el evento de fin de relación. Contratación temporal lo traslada a Bolsa, que
aplica la reposición (duda 64). Nada viaja de Personal a Aspirantes. Como el teléfono es
lo que usa la bolsa para el siguiente llamamiento, VEC pide a la persona, al cesar, que
confirme o actualice el teléfono y el correo de su participación en el portal externo.
Si los cambió en Personal mientras trabajaba, puede usar un botón «usar también en mis
inscripciones», que es una acción suya y queda registrada (criterio técnico).

### 4.6 Qué pasa con cada dato al terminar la relación

| Dato | Dónde | Al terminar la relación |
| --- | --- | --- |
| Identidad y relaciones de servicio | Personal | Se conservan: sirven para certificados de servicios, trienios y jubilación. Plazo según la serie documental (pregunta 84) |
| IBAN, datos del IRPF, afiliación | Personal | Dejan de usarse tras la última nómina. Su conservación la fija la serie documental de nóminas; como referencia, las obligaciones fiscales y de Seguridad Social prescriben a los cuatro años (Ley General Tributaria, art. 66; Ley General de la Seguridad Social, art. 24) (a confirmar por el DPD y el Archivo) |
| Contacto laboral | Personal | Deja de usarse al cerrar la relación, salvo para enviar documentos finales durante un plazo corto (a confirmar por el DPD) |
| Enlace aspirante-trabajador | Registro de correspondencias | Se conserva mientras exista alguno de los dos lados, porque se necesita si la persona vuelve |
| Participación en bolsa | Aspirantes y Bolsa | Sigue viva: la persona vuelve a la lista |

### 4.7 Persona que es a la vez trabajadora y aspirante

Es un caso habitual: un trabajador que se inscribe en otra bolsa o en el turno libre de
una oposición.

- En el portal externo es `asp_`, con su ficha y sus propios datos de contacto. En el
  interno es `emp_`. No comparten preferencias, correos ni foto (criterio técnico,
  pedido por Alberto).
- RRHH de Selección lo ve como aspirante y no ve su expediente laboral.
- Su jefatura no sabe que se ha presentado, salvo por las listas públicas que exige el
  procedimiento.
- Si pertenece a RRHH y gestiona ese mismo proceso, debe abstenerse (Ley 40/2015, arts.
  23 y 24) (norma). VEC puede mantener una lista de exclusiones por proceso: quien
  figura como aspirante no puede abrir ese proceso desde el portal interno (criterio
  técnico, pregunta 90).

### 4.8 Promoción interna y convocatorias mixtas

Los empleados públicos están obligados a relacionarse por medios electrónicos en los
trámites que hacen por su condición de empleado (Ley 39/2015, art. 14.2.e, norma). La
promoción interna es uno de ellos, y sus requisitos (antigüedad, subgrupo, titulación)
están en Personal. Por eso (criterio técnico):

- La inscripción en promoción interna se hace desde el portal interno, con `emp_`, y los
  requisitos se comprueban con datos de Personal. No pasa por Aspirantes ni por el
  registro de correspondencias.
- En una convocatoria mixta (turno libre y promoción interna), el módulo de Procesos
  selectivos guarda dos tipos de inscripción: con `asp_` las del turno libre y con `emp_`
  las de promoción interna. Las listas y la baremación las tratan juntas con los datos
  que cada almacén entrega para esa finalidad.
- Si la persona que promociona ya está en la nueva plaza, Personal registra el cambio de
  relación. No hay traspaso.

### 4.9 Búsqueda de talento interno

Idea futura: avisar a trabajadores (por ejemplo, de subgrupo C2 con titulación de C1) de
convocatorias a las que podrían optar.

- Se hace dentro de Personal y con datos de Personal: titulación acreditada, categoría,
  relación. No consulta Aspirantes.
- No ordena ni puntúa a nadie. Solo avisa a la persona, así que no hay decisión
  automatizada con efectos (RGPD art. 22, norma).
- Base: art. 6.1.e con la prueba de compatibilidad del art. 6.4, porque la titulación se
  recogió para gestionar la relación y ahora se usa para informar de la carrera
  profesional (TREBEP arts. 14.c y 16). El consentimiento no es buena base con
  trabajadores, por el mismo desequilibrio del considerando 43. La persona puede darse
  de baja de los avisos en cualquier momento (criterio técnico; a confirmar por el DPD,
  pregunta 92).

### 4.10 Impacto en lo ya construido

| Pieza | Situación | Cambio propuesto | Cuándo |
| --- | --- | --- | --- |
| Procesos y credenciales | Portal externo e interno comparten composición | Procesos, credenciales y material de claves distintos por portal | Fase 1, primero |
| Usuarios, preferencias (5.08a) y foto (5.08c) | Clave `per_` común a los dos portales | Añadir la superficie a la clave | Fase 1 |
| Usuarios, correos (5.08b) | Clave `per_` común | Esquema por población con roles y claves propios | Fase 1 |
| Contexto de actor | Alcance de proyecciones ya existe; puede llevar vínculos heredados | Prohibir el vínculo de candidato en el portal interno y retirar los vínculos de empleado heredados. Toca el núcleo de identidad: revisión de seguridad y SQL | Fase 1 |
| Bolsa, contacto por participación | Correo y dos teléfonos por participación | Leer el contacto de la ficha de Aspirantes por puerto autorizado. Queda un contacto propio de la participación solo si las bases lo permiten (pregunta 80) | Fase 1 |
| Bolsa, referencia `can_` | Derivada de documento enmascarado y nombre | Sustituir por `asp_`. Cada fila importada recibe su `asp_` provisional; la unión con la identidad real se hace cuando la persona entra por primera vez con certificado, como hoy. Si dos filas coinciden en documento enmascarado y nombre, la importación las marca y RRHH las resuelve a mano; nunca se unen solas | Fase 1 |
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
| Un dueño por población: Aspirantes (externos) y Personal (internos) | La separación cae de forma natural | Un módulo más. Hay que mover el contacto de Usuarios y de Bolsa |

Recomendación (criterio técnico): módulo nuevo Aspirantes para la población externa y
Personal para la interna. Usuarios se queda con lo que es del portal (preferencias y
foto). Bolsa, Procesos selectivos y Provisión guardan referencias y piden a cada dueño
lo que necesitan para cada finalidad. Así se cumple la regla de AGENTS.md de que ningún
módulo lee tablas de otro, y la especificación E02: «Bolsa consulta el expediente por un
puerto autorizado y no duplica titulaciones, méritos ni datos personales». La identidad
sigue siendo la común; Aspirantes guarda el expediente de la persona aspirante y no se
convierte en otra autoridad de identidad.

Contenido de Aspirantes:

- Ficha: identidad tomada de la identificación, historial de documentos, nacionalidad,
  fecha de nacimiento, contacto, domicilio solo si hay notificación en papel.
- Méritos y titulaciones declarados, y hechos verificados (resultado de consultas).
- Documentos aportados, guardados por la capacidad documental común con referencia
  opaca.
- En un almacén aparte, los datos protegidos (apartado 5.3).

### 5.2 Cifrado

| Alternativa | Qué protege | Qué no protege |
| --- | --- | --- |
| Cifrado del disco o de la base | Robo del disco o de las copias en frío | A quien pueda consultar la base: administradores, una inyección SQL, un volcado |
| Cifrado dentro de PostgreSQL (`pgcrypto`) | Algo más que lo anterior | La clave pasa por la base y puede acabar en registros y en la memoria del servidor |
| Cifrado por campo en la aplicación con claves en un gestor externo | La base solo ve texto cifrado. Un volcado o una consulta indebida no revelan nada | Impide buscar por contenido: hacen falta índices ciegos para la igualdad y descifrar en la aplicación |

Recomendación (criterio técnico, apoyado en RGPD art. 32.1.a, que cita el cifrado como
medida, norma):

- Cifrado por campo en la aplicación, como ya hacen Bolsa y Usuarios, con AES-GCM y
  datos asociados que liguen el cifrado a la persona, al campo y a la versión.
- Claves en un gestor de claves o HSM fuera del proceso (pendiente, riesgo R-08). Cada
  sobre guarda la referencia de la clave con que se cifró, para rotar sin parar el
  servicio.
- Claves distintas por población, por categoría y por función (cifrar e índice ciego).
  Esta separación solo protege de verdad cuando las claves viven fuera del proceso y cada
  portal tiene su propio material; hasta entonces, lo que protege es separar procesos y
  credenciales (apartado 4.2).
- Índice ciego con HMAC para buscar por documento y detectar duplicados. Para buscar por
  apellidos, índice ciego del apellido normalizado completo, solo por igualdad.
- Además, cifrado del disco y de las copias.

El borrado criptográfico (destruir la clave de una persona para que sus datos queden
ilegibles incluso en las copias) es útil para datos que no forman parte de un expediente,
como contactos adicionales o la foto. Para datos del expediente solo puede usarse cuando
la eliminación esté aprobada por la valoración documental (apartado 5.8).

### 5.3 Datos protegidos: salud y víctimas de violencia

- Discapacidad, adaptaciones, cualquier dato de salud y la condición de víctima de
  violencia de género o sexual van en un esquema propio de Aspirantes (y de Personal para
  los trabajadores), con clave propia y un rol de lectura limitado a finalidades
  concretas: admisión por cupo, exención de tasa, adaptación de pruebas y aplicación de
  la marca de protección.
- Se guarda lo mínimo que exija el cupo: «discapacidad igual o superior al 33 %,
  verificada el día D» para el cupo general; para el cupo de discapacidad intelectual,
  además, que la discapacidad es intelectual, porque ese cupo lo exige. Nunca el
  diagnóstico.
- El tribunal ve la adaptación concedida («30 minutos más», «sala accesible»), no el
  grado ni el dictamen.
- Si el proceso crea una bolsa con cupo de reserva, Bolsa necesita saber a qué cupo
  pertenece cada participación mientras la bolsa esté vigente. Lo recibe como referencia
  al cupo, sin el dato de salud.
- No se reutiliza en otro proceso sin que la persona lo pida en ese proceso. VEC puede
  ofrecer «usar la verificación que ya consta», que es una acción suya (criterio
  técnico).
- Los datos del art. 10 RGPD (delitos sexuales) no pasan por Aspirantes. Van a Personal
  al incorporarse, y allí solo se guarda «certificado negativo verificado el día D».

### 5.4 Vistas por finalidad y rol

Cada vista es una función con permiso propio que devuelve solo los campos de su
finalidad y anota la lectura. La tabla es una propuesta (a confirmar por RRHH y el DPD,
preguntas 80 y 90).

| Quién | Qué ve | Qué no ve |
| --- | --- | --- |
| La propia persona (portal externo) | Toda su ficha, sus inscripciones y documentos, sus ejercicios con las anotaciones del corrector y quién ha consultado sus datos (unidad y finalidad) | Datos de otros, salvo lo que le corresponda como interesada |
| Otra persona interesada en el mismo proceso | Lo que la Ley 39/2015 le permite ver del expediente (art. 53.1.a), con los datos de terceros reducidos a lo necesario | Contacto, domicilio, datos de salud y cualquier otro dato de los demás que no sea necesario para defender su derecho |
| RRHH, Selección | Ficha y méritos de los aspirantes de sus procesos; contacto para llamamientos | Datos protegidos, salvo para admitir por cupo; expediente laboral |
| Tribunal u órgano de selección (TREBEP art. 60) | En pruebas anónimas, el ejercicio con código, sin nombre. En méritos, los méritos y documentos del proceso. La adaptación concedida | Contacto, domicilio, grado de discapacidad, otras inscripciones |
| Centro solicitante (bolsa) | Nombre de la persona asignada y fecha de incorporación | Méritos, contacto, posición en la lista. Si hay marca de protección, lo que la marca excluya |
| Intervención | La identidad de la persona nombrada y el acto que fiscaliza | Méritos, contacto, datos protegidos |
| Personal (tras el traspaso) | Los campos del catálogo de traspaso | El resto del expediente de aspirante |
| Soporte técnico | Referencias y metadatos (duda 36) | Ningún dato en claro |
| Publicación | Listas con nombre y documento parcial (apartado 5.9) | Todo lo demás |

La vista de llamamientos de RRHH muestra el teléfono y el correo directamente. La propia
vista ya acredita la finalidad y deja constancia de la lectura, así que no se añade un
clic más para «desvelar» el dato (criterio técnico; un paso extra en cada llamada solo
entorpece).

Marca de protección para víctimas de violencia de género o sexual. El TREBEP obliga a
proteger su intimidad y, en especial, sus datos personales y los de sus descendientes
(art. 82, norma), y la LOPDGDD prevé procedimientos seguros de publicación y
notificación para ellas (disposición adicional 7.ª, apartado 2, norma). La persona la
activa acreditando su situación por cualquiera de los medios que admite la Ley Orgánica
1/2004 (art. 23). Efectos (criterio técnico, a confirmar por el DPD, pregunta 93):

- El domicilio no lo ve nadie más que quien tenga que notificar en papel.
- En listas publicadas y en adjudicaciones de centro, la persona aparece por un código o
  por el documento parcial sin nombre, según decida el DPD.
- El centro de destino no ve su historial en bolsa ni otros datos.
- Las vistas de RRHH muestran un aviso de protección y cada lectura se revisa.

### 5.5 Registro de accesos

- Toda lectura en claro de un dato personal deja constancia de actor, instante,
  finalidad, persona afectada (por referencia), campos leídos y resultado. Bolsa ya lo
  hace (registro de accesos T13) y es el patrón a seguir (criterio técnico; ENS, anexo II,
  registro de actividad, norma).
- La persona puede consultar qué unidades han accedido a sus datos, cuándo y para qué.
  El Tribunal de Justicia de la UE ha dicho que el derecho de acceso comprende las fechas
  y las finalidades de las consultas, pero no necesariamente el nombre de cada empleado
  que consultó (sentencia de 22 de junio de 2023, asunto C-579/21) (a confirmar por el
  DPD, pregunta 91).
- El registro de accesos contiene referencias, no datos en claro, y tiene su propio plazo
  de conservación (pregunta 84).

### 5.6 Historia de solo adición, actos y rectificaciones

- Cada cambio de la ficha es una versión nueva, con motivo, actor y fecha. No se
  sobrescribe nada (criterio técnico ya asentado en VEC).
- Los actos dictados (listas publicadas, baremaciones, resoluciones) conservan su
  contenido tal como se dictaron: nombre, documento enmascarado, puntuaciones y lo demás
  que figure en ellos. Además guardan la versión y la huella de los datos de la ficha que
  usaron. Una rectificación posterior de la ficha no cambia un acto ya dictado.
- Si una rectificación afecta a una baremación, VEC avisa a RRHH de qué procesos usaron la
  versión anterior y RRHH decide por la vía del procedimiento (apartado 6.3).

### 5.7 Derechos de las personas

| Derecho | Artículo RGPD | Cómo en VEC | Límites |
| --- | --- | --- | --- |
| Información | 13 y 14 | Primera capa en el formulario, segunda en la ayuda (LOPDGDD art. 11) | Textos aprobados por el DPD (pregunta 87) |
| Acceso | 15 | «Mis datos» en el portal y copia descargable (art. 15.3). Incluye sus ejercicios y las anotaciones del corrector, que el Tribunal de Justicia de la UE considera datos personales del aspirante (sentencia de 20 de diciembre de 2017, asunto C-434/16, Nowak) | Datos de terceros; el acceso no sirve para corregir respuestas de un examen |
| Acceso de otros interesados al expediente | Ley 39/2015, art. 53.1.a; Ley 19/2013, art. 15 para quien no es interesado | Vista del expediente para interesados con los datos de los demás reducidos a lo necesario | Hasta dónde se ven ejercicios y méritos de los competidores lo decide el DPD (pregunta 91) |
| Rectificación | 16 | La persona corrige contacto y datos declarados. La identidad se corrige en origen | Lo ya baremado sigue su vía (apartado 6.3) |
| Supresión | 17 | Datos no ligados a un expediente: al momento | No mientras el dato sea necesario para una obligación legal o una misión pública (art. 17.3.b) ni mientras motive un acto que pueda recurrirse (art. 17.3.e). Siempre sujeta a valoración documental (apartado 5.8) |
| Limitación | 18 | Estado «limitado» mientras se resuelve una rectificación discutida | El dato se conserva pero no se usa |
| Portabilidad | 20 | No se aplica a tratamientos basados en el art. 6.1.e (art. 20.3). VEC ofrece de todos modos la copia estructurada del art. 15.3 | (a confirmar por el DPD) |
| Oposición | 21 | Cabe frente al art. 6.1.e. En un proceso selectivo equivale en la práctica a desistir (Ley 39/2015, art. 94) | (a confirmar por el DPD) |
| No ser objeto de decisiones automatizadas | 22 | La aplicación propone y RRHH o el tribunal confirman. Ninguna exclusión, orden o adjudicación es solo automática | Si algún día se usara inteligencia artificial, los sistemas de selección son de alto riesgo (Reglamento (UE) 2024/1689, art. 6.2 y anexo III, punto 4) |

### 5.8 Conservación, bloqueo y eliminación

Los documentos y datos de un expediente administrativo son patrimonio documental. Su
eliminación exige una valoración documental previa y la resolución que la autorice, con
el dictamen de la Comisión Andaluza Calificadora de Documentos Administrativos (Ley
7/2011 de Documentos, Archivos y Patrimonio Documental de Andalucía) (norma). Por eso:

- Los plazos se fijan por serie documental (expedientes de selección, bolsas,
  expedientes de personal, nóminas), no dato a dato. Los fija el Archivo con la tabla de
  valoración, y el DPD comprueba que sean coherentes con el RGPD art. 5.1.e (pregunta
  84).
- Ningún dato que forme parte de un expediente se suprime ni se borra
  criptográficamente sin esa valoración, aunque lo pida la persona. Tampoco mientras
  motive un acto recurrible (RGPD art. 17.3.e).
- Cuando un dato deja de ser necesario para su finalidad pero debe conservarse, se
  bloquea: queda reservado y solo puede ponerse a disposición de jueces, tribunales,
  Ministerio Fiscal y Administraciones competentes durante el plazo de prescripción de
  las responsabilidades (LOPDGDD art. 32, norma). En VEC, un dato bloqueado sale de todas
  las vistas y solo lo lee una función especial con doble control.
- Al final del plazo, el dato se elimina si la tabla lo dispone, o pasa al Archivo si hay
  conservación permanente (RGPD art. 89; LOPDGDD art. 26).

Referencias para proponer la valoración (todas provisionales y configurables):

| Serie o dato | Uso activo | Después | Nota |
| --- | --- | --- | --- |
| Expediente de un proceso selectivo | Hasta la firmeza: resolución final, dos meses para el recurso contencioso-administrativo (Ley 29/1998, art. 46) y lo que duren los recursos | Bloqueo al menos cuatro años más (recurso extraordinario de revisión, Ley 39/2015, art. 125.2) y después lo que diga la tabla | Suele conservarse de forma permanente lo esencial (bases, listas, resoluciones) |
| Participación en bolsa, incluido el cupo al que pertenece | Mientras la bolsa esté vigente | Como el anterior, desde la derogación de la bolsa o la baja | El cupo se necesita mientras la bolsa exista |
| Contacto de aviso | Mientras la persona lo mantenga | Se retira el valor; la huella de la dirección usada en cada aviso queda en el expediente como prueba | La prueba sigue el plazo del expediente |
| Datos protegidos fuera de un expediente (verificaciones no usadas) | Mientras la persona los quiera disponibles | Supresión, con borrado criptográfico | Lo que se usó en un proceso sigue el plazo de ese expediente |
| Registro de accesos | Dos años (propuesta) | Supresión | Plazo a fijar con Seguridad |
| Ficha de aspirante sin inscripciones ni bolsas | Dos años sin actividad (propuesta), con aviso previo | Supresión de lo que no forme parte de ningún expediente | |

Hasta que el Archivo y el DPD aprueben los plazos, VEC los guarda como anotación y no
bloquea ni elimina nada de forma irreversible, igual que en la duda 61.

### 5.9 Seudonimización en listados y publicaciones

- Pantallas internas: los listados muestran el nombre completo solo a quien tiene
  finalidad y el documento siempre parcial.
- Publicación de actos: nombre y apellidos más cuatro cifras aleatorias del documento
  (LOPDGDD disposición adicional 7.ª, apartado 1, norma). La orientación de la AEPD de
  2019 fija las cifras: para el DNI `12345678X` se publica `***4567**`, que es el formato
  que ya usa Bolsa; para el NIE `X1234567L`, `****4567*`; y para pasaportes y otros
  documentos, cuatro cifras en posiciones equivalentes (a confirmar por el DPD).
- Si la publicación sirve de notificación (Ley 39/2015, art. 44), solo el número de
  documento, sin nombre (LOPDGDD disposición adicional 7.ª, apartado 1, segundo párrafo,
  norma).
- No se publica junto al nombre ningún dato que revele salud: ni el turno de reserva por
  discapacidad ni la causa de exclusión si es de salud (a confirmar por el DPD, pregunta
  83).
- Las personas con marca de protección se publican según el apartado 5.4.
- Las publicaciones en la web tienen fecha de retirada y no se ofrecen a buscadores
  (criterio técnico, pregunta 83).
- Hacia fuera del sistema, los eventos y la auditoría solo llevan referencias opacas
  (criterio técnico ya vigente).

### 5.10 Datos de prueba

- Todo desarrollo usa datos sintéticos con nombres verosímiles, sin tomar nada de
  CONVOCA hasta que haya visto bueno (regla ya vigente en VEC).
- Los DNI y NIE sintéticos tienen formato y letra válidos. Como cualquier número válido
  puede corresponder a una persona real, nunca se combinan con datos reales ni salen del
  entorno de desarrollo. Cada conjunto de prueba se marca como sintético.
- Los conjuntos de prueba incluyen los casos difíciles: menores, personas extranjeras
  con cambio de NIE a DNI, marcas de protección, coincidencias de documento enmascarado
  y nombre, personas que son a la vez aspirantes y trabajadoras.
- Los datos protegidos de prueba se generan en proporciones realistas, sin diagnósticos.
- Riesgo detectado al preparar este estudio: en equipos de desarrollo hay exportaciones
  reales de CONVOCA (hojas de cálculo con nombres y documentos enmascarados, y
  certificados de servicios en PDF). Para este estudio solo se han leído los nombres de
  las columnas. Esos ficheros son datos reales y deberían tratarse con las medidas que
  correspondan o retirarse de esos equipos (a confirmar por el DPD).

## 6. Casos difíciles

### 6.1 Trabajadora que vuelve a la bolsa al terminar su interinidad

Personal cierra la relación y emite el evento. Bolsa la repone según su regla. VEC le pide
que confirme el teléfono y el correo de su participación. Su ficha de aspirante no se
tocó mientras trabajaba. Sus datos de nómina dejan de usarse y siguen el plazo de su
serie (apartado 4.6).

### 6.2 Petición de supresión con un proceso abierto

- Los datos necesarios para el proceso no se suprimen mientras siga abierto ni mientras
  motiven actos recurribles (RGPD art. 17.3.b y 17.3.e, norma). Si la persona quiere
  salir, puede desistir o renunciar (Ley 39/2015, art. 94). Desde ese momento sus datos
  dejan de usarse para el proceso y siguen el plazo del expediente.
- Lo que no forma parte de ningún expediente (preferencias, foto, contactos adicionales)
  se suprime en el momento.
- VEC responde en el plazo de un mes (RGPD art. 12.3, norma) indicando qué se suprime,
  qué se conserva, hasta cuándo y por qué.

### 6.3 Rectificación de un dato que ya se usó para baremar

- La baremación se hace con los méritos acreditados a la fecha que fijan las bases. Si el
  dato era correcto en esa fecha y después cambió, la baremación no cambia: se añade la
  versión nueva a la ficha.
- Si el dato era erróneo y afecta a la puntuación, el cauce es el del procedimiento:
  subsanación (Ley 39/2015, art. 68), reclamación contra las listas provisionales o
  recurso. La rectificación del RGPD no reabre por sí sola un acto administrativo (a
  confirmar por el DPD).
- VEC registra la versión nueva, avisa a RRHH de los procesos que usaron la anterior y no
  recalcula nada solo. Los actos dictados conservan su contenido (apartado 5.6).

### 6.4 Antigua trabajadora que vuelve años después como aspirante

- Se inscribe en el portal externo como cualquier aspirante. No se crea ningún enlace
  solo por inscribirse.
- Si quiere que puntúen sus servicios en la Diputación, lo declara. El registro de
  correspondencias localiza su `emp_` antiguo y Personal emite un certificado de servicios
  para ese proceso. Ella no tiene que aportarlo (Ley 39/2015, art. 28.2).
- El certificado se basa en las relaciones de servicio, que se conservan.
- Si la nombran de nuevo, el traspaso reutiliza el mismo `emp_`.

### 6.5 Procesos de estabilización

En una estabilización, gran parte de los aspirantes son o fueron personal de la
Diputación y reclaman servicios prestados aquí. Pedir certificados uno a uno sería
desproporcionado. Propuesta (criterio técnico, a confirmar por RRHH):

- En la inscripción, la persona indica si quiere que se tengan en cuenta sus servicios en
  la Diputación.
- Al cerrar el plazo, el registro de correspondencias localiza en bloque los `emp_` de
  quienes lo pidieron y Personal emite una certificación masiva de servicios para el
  proceso, con una fila por aspirante y referencia al acto.
- La certificación entra en el expediente del proceso como documento y el tribunal la usa
  como cualquier otro mérito acreditado.

### 6.6 Error en un traspaso

Ejemplo: se enlaza a una aspirante con el trabajador equivocado.

- No se borra nada. Se añade una anulación del enlace, con motivo, y la hace una persona
  distinta de la que lo creó (doble control).
- Personal recibe un evento de corrección: anula el alta errónea con una versión nueva y
  crea la correcta.
- Todos los accesos hechos con el enlace erróneo quedan en el registro. Si alguien vio
  datos que no debía, es una brecha de seguridad. La Diputación, como responsable, valora
  si debe notificarla a la AEPD en 72 horas y, en su caso, a las personas afectadas
  (RGPD arts. 33 y 34), y la documenta en todo caso, se notifique o no (art. 33.5)
  (norma).
- Prevención: el enlace por traspaso usa el índice ciego del documento, no el nombre, y
  exige que coincidan identidad y acto.

### 6.7 RRHH ante aspirantes que son compañeros

- Solo ven a los aspirantes quienes tienen asignado ese proceso, y cada consulta queda
  anotada.
- Quien sea aspirante en un proceso no puede abrirlo desde el portal interno.
- Quien tenga relación personal con un aspirante debe abstenerse (Ley 40/2015, art. 23,
  norma). VEC permite registrar la abstención y retira el acceso a ese proceso.

### 6.8 Tribunales

- Sus miembros pueden ser de otras Administraciones. Actúan como órgano de selección
  (TREBEP art. 60), no como encargados del tratamiento (a confirmar por el DPD).
- Acceso temporal, limitado al proceso y a la fase, con identificación reforzada y
  compromiso de confidencialidad.
- En pruebas anónimas, VEC separa el código del ejercicio de la identidad hasta que se
  abren las plicas en acto formal.
- Las anotaciones del corrector son datos de la persona examinada y ella puede verlas
  (apartado 5.7).

### 6.9 Cambio de documento y fusión de fichas

Caso típico: una persona se inscribe con NIE, se nacionaliza y vuelve a entrar con DNI.
VEC la ve como dos fichas, porque los índices ciegos no coinciden.

- La persona, desde su ficha nueva, declara su documento anterior, o RRHH detecta la
  coincidencia por nombre y fecha de nacimiento.
- Una persona de RRHH propone la fusión y otra la aprueba. Se registra con motivo y
  justificante (por ejemplo, la consulta de identidad).
- La fusión añade el documento antiguo al historial de la ficha que se mantiene y marca
  la otra como fusionada, con referencia a la primera. No se borra ninguna de las dos, y
  los actos dictados con la ficha antigua siguen apuntando a ella.
- Si la fusión fue un error, se deshace con otra fila, igual que en el apartado 6.6.

### 6.10 Publicación de listas

Ver los apartados 5.4 y 5.9. Las dudas principales son cómo publicar los turnos de
reserva sin revelar un dato de salud y cómo publicar a las personas con marca de
protección (preguntas 83 y 93).

## 7. Requisitos formales

### 7.1 Evaluación de impacto (EIPD)

Es obligatoria (norma): el RGPD la exige cuando es probable un alto riesgo (art. 35.1) y,
en particular, ante el tratamiento a gran escala de categorías especiales (art. 35.3.b).
Aquí se dan varios de los criterios de las directrices del Grupo de Trabajo del Artículo
29 (WP248 rev.01, asumidas por el Comité Europeo de Protección de Datos) y de la lista de
la AEPD del art. 35.4:

- evaluación y puntuación de personas (baremación);
- categorías especiales (discapacidad) y datos de víctimas de violencia;
- gran escala (miles de aspirantes por convocatoria);
- combinación de fuentes (Plataforma de Intermediación, CONVOCA, Personal);
- personas en situación de desequilibrio frente a la Administración, y menores;
- efecto en el acceso a un empleo.

La LOPDGDD añade como factores de mayor riesgo el tratamiento no incidental de
categorías especiales, la evaluación de aspectos personales y los grupos vulnerables,
en particular menores y personas con discapacidad (art. 28.2) (norma).

La hace el responsable del tratamiento con el consejo del DPD (RGPD art. 35.2). Si el
riesgo residual sigue siendo alto, hay consulta previa a la AEPD (art. 36). Propuesta:
una EIPD para «Selección y bolsas» y otra para «Gestión de personal», que coinciden con
las dos poblaciones (a confirmar por el DPD, pregunta 85).

### 7.2 Registro de Actividades de Tratamiento

La Diputación publica su inventario de actividades (RGPD art. 30; LOPDGDD art. 31.2,
norma). Hay que revisar si las actividades existentes, como selección de personal, bolsas
de trabajo y gestión de personal, cubren lo que hará VEC: nuevas categorías de datos,
consultas a la Plataforma, registro de correspondencias, marca de protección, avisos de
oportunidades y plazos (pregunta 86).

### 7.3 Categorización ENS

- El ENS incluye las medidas de seguridad para datos personales en el sector público
  (LOPDGDD disposición adicional 1.ª, norma).
- La categoría la fija el responsable de la información según el impacto en
  confidencialidad, integridad, trazabilidad, autenticidad y disponibilidad (ENS, arts.
  13 y 40 y anexo I, norma).
- Propuesta técnica: categoría MEDIA como mínimo. La confidencialidad del almacén de
  datos protegidos podría valorarse como ALTA según la guía CCN-STIC-803 (a confirmar por
  Seguridad, pregunta 89).
- De esa categorización depende la opción C del apartado 4.2 y la custodia de claves.

### 7.4 Información a las personas

- Información en capas (LOPDGDD art. 11, norma): la primera junto al botón de enviar la
  solicitud, con responsable, finalidad, derechos y dónde ampliar; la segunda en la ayuda
  del botón «?», con todo lo que exige el art. 13 RGPD: base, destinatarios (tribunal,
  publicaciones, Plataforma), plazos, derecho a reclamar ante la AEPD y contacto del DPD.
- Un texto por finalidad (inscripción, bolsa, traspaso, avisos de talento, marca de
  protección), en catálogo i18n y versionado, con redacción comprensible también para
  menores de 16 y 17 años (RGPD art. 12.1). Cada inscripción guarda la versión del texto
  que se mostró.
- Los textos los aprueba el DPD (pregunta 87).

### 7.5 Encargados del tratamiento

Posibles encargados (RGPD art. 28, norma; lista a confirmar, pregunta 88):

- proveedor del alojamiento de VEC en producción, si no es infraestructura propia de la
  Diputación;
- proveedor del correo saliente, si el SMTP corporativo lo presta un tercero;
- proveedor de SMS o de llamadas, si se usa para llamamientos;
- el actual proveedor de CONVOCA, durante la migración;
- la Plataforma de Intermediación y Cl@ve, cuyo papel fija el convenio de adhesión (a
  confirmar por el DPD).

Cada uno necesita contrato o convenio con las cláusulas del art. 28.3 RGPD.

## 8. Plan de implantación por fases

Alineado con la arquitectura de VEC: hexagonal, un dueño por dato, referencias opacas,
bandeja de salida y catálogos versionados. Ningún corte de este plan desplaza la
prioridad de Contratación temporal: cuando una tarea de Contratación necesite algo de
aquí, se abre la minitarea dependiente y se vuelve al camino crítico (AGENTS.md).

### Fase 0. Ahora, sin programar

- Este estudio, su revisión y las preguntas 77 a 94 de `dudas.md`.
- Decisiones de dirección sobre el módulo Aspirantes, la separación de procesos y el
  registro de correspondencias.

### Fase 1. Con datos sintéticos (se puede construir ya)

Por orden de dependencia:

1. Procesos, credenciales de base de datos y material de claves distintos para el portal
   externo y el interno.
2. Catálogo de campos por finalidad y momento: qué pide cada tipo de convocatoria, con
   obligatoriedad y momento. Versionado y modificable, con los valores provisionales del
   apartado 2 como paquete de ejemplo retirable.
3. Usuarios: superficie en la clave de preferencias y foto; esquema por población en
   correos.
4. Contexto de actor: sin vínculo de candidato en el portal interno y retirada de los
   vínculos de empleado heredados. Necesita revisión de seguridad y SQL.
5. Módulo Aspirantes: ficha cifrada por campo, documento tipado con historial, índices
   ciegos, historia, vistas por finalidad, registro de accesos y «Perfil y contacto» del
   área personal (teléfono, móvil, domicilio y código postal solo si el catálogo los
   pide).
6. Almacén de datos protegidos con clave propia y marca de protección, sin activar en la
   presentación.
7. «Mis datos»: copia descargable (art. 15.3) y lista de accesos a mis datos.
8. Bolsa: pasar a `asp_`, con detección de coincidencias en la importación y unión con la
   identidad al primer acceso con certificado; leer el contacto de Aspirantes.
9. Registro de correspondencias y traspaso por bandeja de salida desde Contratación
   temporal a Personal.
10. Conservación como anotación, sin bloqueo ni eliminación irreversibles.
11. Generador de datos sintéticos con los casos difíciles del apartado 5.10.

Todo con las puertas habituales: pruebas focales, PostgreSQL efímero con roles y ACL
reales, revisión SQL independiente y revisión de seguridad antes de fusionar.

### Fase 2. Tras la EIPD y el visto bueno del DPD

- Activar los datos protegidos y la marca de protección.
- Integrar la Plataforma de Intermediación.
- Gestor de claves o HSM real, con rotación probada.
- Base de datos aparte para Aspirantes si la categorización ENS lo exige.
- Textos de información aprobados.
- Reglas de publicación aprobadas.
- Plazos por serie documental aprobados por el Archivo, con bloqueo real.
- Actualización del Registro de Actividades.

### Fase 3. Datos reales

- Carga de las bolsas de CONVOCA en Aspirantes (dudas 16 y 45).
- Contratos con encargados firmados.
- Formación de RRHH y de tribunales en las vistas por finalidad.
- Revisión periódica del registro de accesos.

## 9. Riesgos y límites de este estudio

- Es un estudio técnico. Lo marcado «(a confirmar por el DPD)» no es asesoramiento
  jurídico cerrado.
- Las bases concretas de la Diputación no están en el repositorio. El inventario del
  apartado 2 debe contrastarse con ellas (pregunta 80).
- La disponibilidad de cada servicio de la Plataforma de Intermediación depende de la
  adhesión de la Diputación.
- Cambiar el contexto de actor y las claves de Usuarios toca piezas ya desplegadas. Hay
  que hacerlo en minitareas pequeñas, con ensayo sobre el clon de la principal.
- Mientras no exista un gestor de claves real, el cifrado protege frente a un volcado de
  la base, pero no frente a un compromiso del proceso de aplicación. Por eso separar
  procesos y credenciales va primero.

## 10. Preguntas añadidas a `dudas.md`

Preguntas 77 a 94: separación entre poblaciones (77), registro de correspondencias (78),
campos del traspaso (79), datos por momento (80), Plataforma de Intermediación y
oposición a la consulta (81), discapacidad y adaptaciones (82), publicación de listas
(83), conservación por serie documental (84), EIPD (85), Registro de Actividades (86),
textos de información (87), encargados (88), ENS y claves (89), RRHH y tribunales ante
compañeros (90), acceso a registros, exámenes y expediente (91), búsqueda de talento
interno (92), víctimas de violencia (93) y menores y personas extranjeras (94).

## Anexo. Normas citadas

- Reglamento (UE) 2016/679 (RGPD): considerando 43; arts. 4.5, 5, 6, 9, 10, 12 a 22, 25,
  28, 30, 32 a 36 y 89.
- Ley Orgánica 3/2018 (LOPDGDD): arts. 8, 9, 11, 26, 28.2, 31 y 32; disposiciones
  adicionales 1.ª y 7.ª; disposición final 12.ª (nueva redacción del art. 28.2 de la Ley
  39/2015).
- Ley 39/2015: arts. 5, 9, 11, 14, 28, 41, 44, 45, 53, 66, 68, 94 y 125.
- Ley 40/2015: arts. 23, 24 y 155.
- Ley 19/2013, de transparencia: art. 15.
- Real Decreto Legislativo 5/2015 (TREBEP): arts. 10, 11, 14, 16, 55 a 63, 78 a 84;
  disposición adicional 17.ª.
- Ley 7/1985, de Bases del Régimen Local: art. 91.2.
- Real Decreto 896/1991, sobre selección de funcionarios de Administración Local.
- Real Decreto 2271/2004, sobre acceso al empleo público de personas con discapacidad:
  arts. 3 y 8.
- Real Decreto Legislativo 1/2013, Ley General de derechos de las personas con
  discapacidad.
- Ley Orgánica 1/2004, de Medidas de Protección Integral contra la Violencia de Género:
  art. 23.
- Ley Orgánica 1/1996, de Protección Jurídica del Menor: art. 13.5.
- Ley Orgánica 3/2007, de igualdad efectiva de mujeres y hombres: art. 20.
- Ley 31/1995, de Prevención de Riesgos Laborales: art. 22.4.
- Ley 53/1984, de incompatibilidades del personal al servicio de las Administraciones
  Públicas.
- Estatuto de los Trabajadores: arts. 7.b y 15.5.
- Ley 58/2003, General Tributaria: arts. 66 y 95.1.k. Real Decreto Legislativo 8/2015,
  Ley General de la Seguridad Social: art. 24.
- Ley 29/1998, de la Jurisdicción Contencioso-administrativa: art. 46.
- Ley 7/2011, de Documentos, Archivos y Patrimonio Documental de Andalucía.
- Real Decreto 311/2022 (ENS): arts. 13 y 40; anexos I y II.
- Reglamento (UE) 2024/1689 de inteligencia artificial: art. 6.2 y anexo III, punto 4.
- Sentencias del Tribunal de Justicia de la UE de 20 de diciembre de 2017 (asunto
  C-434/16, Nowak) y de 22 de junio de 2023 (asunto C-579/21).
- Grupo de Trabajo del Artículo 29, directrices WP248 rev.01 sobre EIPD, asumidas por el
  Comité Europeo de Protección de Datos; lista de la AEPD de tratamientos que requieren
  EIPD (art. 35.4 RGPD); orientación de la AEPD sobre la disposición adicional 7.ª de la
  LOPDGDD.
