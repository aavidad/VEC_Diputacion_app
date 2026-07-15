# Auditoria adversaria de ejecucion documental V3

Fecha: 15-07-2026.

## Dictamen

Los contratos V3 mejoran de forma material el aislamiento de componentes, el
cercado, la idempotencia, la evidencia y la reconciliacion, pero no son aptos
para produccion administrativa. Permanecen en sombra y no se conectan a HTTP,
CLI, MCP, generadores heredados ni adaptadores productivos.

La revision se realizo sobre los siguientes ficheros introducidos en el commit
`421846e` y conservados sin cambios:

- `internal/vec/ports/ejecucion_componentes_documentales_atestada.go`,
  SHA-256
  `c60ff295e800cc34876d87dca4aa6a82c1c91b35cce587fefac64a48e0cf5433`;
- `internal/vec/ports/ejecucion_componentes_documentales_atestada_test.go`,
  SHA-256
  `bfb7c69fd92ad7d5212dd0f6eee51398f4297d65fc4ad58634a7c0dfec32d348`;
- `internal/vec/ports/ejecuciones_documentales_v3.go`, SHA-256
  `4c9d1c3a696732f4ae04a123764751892982d4cca82c2ac7a3cfc93261dece05`;
- `internal/vec/ports/ejecuciones_documentales_v3_test.go`, SHA-256
  `a3baf868de08b5982ca04fccbaf36d5566d1180e2a6cdeccafacc4940804e78e`.

Pruebas normales, detector de carreras y `go vet` eran correctos. Esto acredita
coherencia del codigo probado, no suficiencia del modelo de confianza.

## Fortalezas confirmadas

- manifiesto comparado por contenido y no por identidad de puntero;
- tres componentes exactos y segregados por rol, artefacto, homologacion,
  dominio, carga, proceso, clave y medicion;
- situacion operativa, revision, plan, efecto, decision y cercado ligados;
- ventanas exclusivas, limites minimos y referencias cruzadas rechazadas;
- copia defensiva de MAC, COSE y sellos;
- el recibo no conserva token, MAC, compromiso completo ni bytes COSE;
- evidencia con HMAC y reconciliacion con sobre firmado;
- estados actuales cerrados y denegacion de valores cero o ambiguos.

Estas propiedades se conservan en V4 salvo que una prueba demuestre que una
sustitucion es mas segura.

## Bloqueos criticos

### Autoridad fabricable mediante la API publica

Constructores publicos convierten datos coherentes, MAC o bytes cualesquiera en
recibos y resultados denominados verificados. No comprueban firma, algoritmo,
clave, audiencia, revocacion o confianza. Cualquier paquete compilado en el
binario puede construir la cadena necesaria sin usar el verificador.

V4 separa estrictamente:

1. prueba criptografica cruda, opaca y sin autoridad;
2. caso de uso neutral que solo depende de
   `ports.ConectorEjecucionDocumentalAtestadaV4`;
3. verificacion COSE, HMAC, socket y transaccion SQL confinados al conector
   `internal/vec/adapters/postgres/confianzadocumental`;
4. confirmacion opaca del puerto, sin capacidad reutilizable ni material
   criptografico;
5. contrato para que el futuro consumidor de negocio solo acepte el resultado
   exacto del conector.

El caso de uso admite la inyeccion del conector, pero la raiz de composicion
productiva y sus superficies HTTP, CLI y MCP siguen sin cablearlo;
`cmd/vec-emisor-capacidad-v4` construye directamente el emisor PostgreSQL. El
nucleo no importa `pgx`, SQL ni transporte. Oracle puede implementar el mismo
puerto sin cambiar `application`, siempre que conserve la revalidacion y la
atomicidad exigidas. La frontera y su conector PostgreSQL han superado la matriz
tecnica; esto no equivale a autorizacion productiva.

Si un HSM o KMS no permite comprobar localmente un MAC, debe devolver una
atestacion firmada verificable con una raiz publica fijada localmente.

### Autorizacion reducida a un DTO

El consumo V3 coteja referencias y huellas aportadas por el llamador, pero no
transporta la capacidad completa ya existente
`EvidenciaUsoDecisionAutorizacion`. V4 debe exigir de forma exacta y positiva:

- concesion, actor y vinculo de autenticacion;
- accion, recurso, modulo, tipo y contexto del recurso;
- finalidad, ambito y correlacion;
- asignacion, version de rol y su control de vigencia;
- politicas y revision del catalogo;
- garantia, campos permitidos y obligaciones satisfechas;
- emision, caducidad y verificacion actual.

Ausencia, denegacion, ambiguedad, comodin, obligacion pendiente, error del PDP,
timeout o atributo incompleto deniegan. `DecisionRef` se consume una vez en la
misma transaccion que el cambio de estado.

## Bloqueos altos

### Entrada, salida y almacenamiento no ligados de extremo a extremo

El compromiso y los bytes se entregan por parametros distintos. Una porcion
puede mutarse, un trabajador puede declarar una huella y escribir otros bytes,
y las referencias de objeto no demuestran existencia o inmutabilidad.

V4 necesita:

- entrada neutral opaca con copia profunda, canonico privado, HMAC y tamano;
- sumidero de cuarentena con limite, contador y SHA-256 observados localmente;
- cierre irreversible antes de comprobar el recibo;
- salida observada inmutable y comparada con las declaraciones firmadas;
- recibo de escritura de almacen ligado a reserva, efecto, conector,
  referencia, version, hash, tamano, cercado, politica e instante;
- promocion atomica solo despues de tres resultados positivos.

### Reconciliacion reproducible y mutaciones sin CAS explicito

La consulta V3 no contiene reto CSPRNG consumible, version esperada ni ventana.
Una respuesta antigua `no_aplicado` podria reutilizarse despues de aparecer el
efecto.

Toda consulta incorpora `ConsultaRef`, reto, emision, expiracion, estado y
version esperados. El COSE compromete esos campos. La aplicacion exige CAS
`estado=indeterminada AND version=N AND cercado=S`; desconocido o conflictivo
solo registra evidencia. `no_aplicado` abandona unicamente con prueba fresca y
exacta.

Inicio, indeterminacion, confirmacion y promocion deben incorporar tambien
estado/version esperados. El inicio devuelve un ticket privado de un solo uso,
obtenido por CAS inmediatamente antes del efecto.

### Resultados negativos e indeterminados ausentes

Cada fase debe poder firmar:

- resultado positivo;
- no conforme, con politica, revision y codigos de reglas;
- indeterminado, con incidente y causa tecnica controlada.

Todos conservan identidad, medicion, plan, cercado, salida, tiempos y prueba
cruda. Solo tres positivos independientes confirman. Se incorporan los estados
`salida_no_conforme_cuarentenada`, `validacion_indeterminada` y
`conflicto_reconciliacion`; compensacion pendiente/compensada se habilita si el
almacen externo lo requiere.

### Cronologia no demostrable

La evidencia inicial permitia afirmar confirmacion antes de verificar el
sello. El orden obligatorio es:

```text
maximo(recibos) <= generado <= firmado <= verificado <= confirmado durable
```

El registro asigna el instante confirmado y la version resultante dentro de la
transaccion. Un instante propuesto por el llamador no se denomina confirmado.

### Recuperacion tras reinicio

El registro debe conservar cifrados el contexto recuperable de cercado y las
referencias inmutables a pruebas crudas. Tras reinicio se vuelven a comprobar
criptograficamente; nunca se restaura autoridad desde una huella publica o un
booleano persistido.

## Endurecimientos medios

- referencias generadas exclusivamente por servidor con esquema cerrado y
  entropia suficiente; las heuristicas de PII son defensa secundaria;
- compromiso, token y sobres siempre redactados y no serializables;
- UTC, precision de microsegundo y anos interoperables antes de calcular
  huellas persistentes;
- el metodo de segregacion no sustituye el agregado que prueba pertenencia de
  los tres recibos al mismo plan, efecto y salida.

Los dos endurecimientos de redaccion y tiempo se aplicaron inmediatamente tras
la auditoria y disponen de pruebas propias.

## Trabajo de adaptadores y producto

- PostgreSQL: unicidad, versionado, CAS, outbox y auditoria en una transaccion;
- almacen: cuarentena cifrada, escritura condicional, versionado, retencion y
  promocion cercada;
- COSE: algoritmos permitidos, cabeceras protegidas, clave, audiencia,
  confianza, revocacion y atestacion;
- KMS/HSM: material distinto por funcion, no solo alias diferentes;
- CSPRNG y registro anti-replay de retos, operaciones y recibos;
- reloj confiable y politica explicita de desfase;
- persistencia cifrada y reverificacion de pruebas tras reinicio;
- concurrencia real contra PostgreSQL y almacen, ademas de pruebas unitarias.
- privilegios PostgreSQL: revocar `USAGE` sobre `TYPES` y cerrar mediante guarda
  DDL los tipos fila implicitos futuros, comprobando privilegios efectivos;
- desmontaje: `roles_down.sql` falla antes de actuar ante miembros inesperados;
- criptografia SQL: inventario de `pgcrypto`, retirada de ejecucion general y
  concesion minima mediante envoltorio cerrado;
- recuperacion: la retirada exige siempre opcion destructiva, usa `RESTRICT` y
  rechaza dependencias externas u objetos futuros no soportados.

Aunque se cierren esos controles tecnicos, V4 permanece **NO-GO productivo**
sin credenciales emisora/ejecutora segregadas, ACL de directorio y socket,
custodia HSM/KMS o gestor homologado, copia/restauracion y procedimientos de
operacion aprobados.

## Puerta para sustituir V3

V4 solo puede proponerse para integracion cuando:

1. un paquete externo no pueda fabricar autoridad mediante API publica;
2. las pruebas usen criptografia real y cubran firma, algoritmo, clave,
   audiencia, revocacion, caducidad y manipulación;
3. mutar entrada, escribir salida distinta o escribir tras cierre falle;
4. autorizacion ausente o incompleta deniegue;
5. reto, cercado, recibo o `no_aplicado` repetidos no cambien estado;
6. la matriz positiva, negativa, indeterminada y conflictiva sea completa;
7. se recupere una ejecucion indeterminada despues de reinicio sin repetir el
   efecto;
8. pruebas normales, de carrera, analisis estatico y auditoria independiente
   sean verdes;
9. el nucleo dependa solo del puerto y las pruebas de arquitectura impidan que
   `application` importe el adaptador PostgreSQL;
10. la matriz SQL demuestre ACL de tipos actuales/futuros, membresias cerradas,
    superficie minima de `pgcrypto` y desmontaje destructivo denegado por
    defecto.

Los controles arquitectonicos y SQL nuevos de los puntos 8 a 10 han superado su
matriz limpia. Los restantes siguen siendo contrato de aceptacion del producto y
no quedan aprobados por ese resultado tecnico.
