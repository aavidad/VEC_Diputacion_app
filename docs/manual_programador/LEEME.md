# Manual del programador

Referencia de todas las funciones, tipos, constantes y variables exportadas
del portal VEC, organizada por capas. Cada entrada muestra la firma Go y su
comentario de documentacion: para que sirve y como se usa. Los ejemplos de
uso canonicos de cada paquete son sus propios ficheros `*_test.go`.

Este manual se genera con:

```bash
python3 scripts/generar_manual_programador.py
```

No editar a mano los ficheros de este directorio: cualquier correccion debe
hacerse en los comentarios de documentacion del codigo Go (o en el script) y
regenerarse.

## Vision general de la arquitectura

La aplicacion es un shell modular (VEC) que agrega modulos independientes
(Personal/Nominas, Cronos, Dietas, Bolsa, Administracion). Cada capa sigue
arquitectura hexagonal con esta regla de dependencias, siempre hacia dentro:

```
adapters  ->  application  ->  ports  ->  domain
   ^                                        |
   |  (implementan los contratos de ports)  |
   +----------------------------------------+
```

- `domain`: reglas puras, sin HTTP, sin persistencia y sin efectos.
- `ports`: contratos (interfaces y tipos de intercambio). Declaran que se
  necesita, nunca como se implementa.
- `application`: casos de uso probados contra memoria; orquestan dominio y
  puertos y aplican autorizacion por caso de uso.
- `adapters`: implementaciones concretas (HTTP, memoria, PostgreSQL, S3,
  PDF/DOCX, criptografia). Solo aqui hay infraestructura real.
- `cmd/vec-server` y `config` componen todo: es el unico punto de arranque
  soportado.

Convenciones transversales del codigo:

- Fallo cerrado: ante configuracion invalida, dependencia ausente o dato no
  canonico, la operacion se deniega; no hay valores por defecto permisivos.
- Autorizacion de lista positiva sin comodines; las decisiones se versionan
  y dejan recibo auditable.
- Los datos sensibles (HMAC, evidencias, capacidades) no se serializan por
  formatos genericos: los tipos protegidos fallan al formatearse.
- Identificadores y mensajes nuevos en espanol; el codigo heredado de
  candidatos conserva nombres en ingles.
- Puerta de calidad local obligatoria: `go test ./...` (completa:
  `scripts/verificar_calidad.sh`).

Documentos de contexto recomendados antes de tocar codigo:

- [Arquitectura tecnica modular del portal](../portal_vec/arquitectura_tecnica.md)
- [Contrato de modulos VEC](../portal_vec/contrato_modulos_vec.md)
- [Registro de decisiones y mejoras](../portal_vec/registro_decisiones.md)

## Ficheros del manual


- [Arranque, composicion y configuracion](cmd_y_configuracion.md)
- [Nucleo VEC: dominio](vec_dominio.md)
- [Nucleo VEC: puertos](vec_puertos.md)
- [Nucleo VEC: aplicacion y dobles de prueba](vec_aplicacion.md)
- [Nucleo VEC: adaptadores](vec_adaptadores.md)
- [Modulo Bolsa](modulo_bolsa.md)
- [Modulos Personal, Cronos, Dietas y Administracion](modulos_personal_cronos_dietas.md)
- [Nucleo heredado de candidatos (Bolsa)](nucleo_candidate.md)
- [Paquetes compartidos](compartido.md)

## Indice de paquetes

| Paquete | Area | Para que sirve |
| --- | --- | --- |
| [`cmd/bolsa-server`](cmd_y_configuracion.md#paquete-cmdbolsa-server) | Arranque, composicion y configuracion | Centinela retirado: falla cerrado y no arranca ningun servidor. |
| [`cmd/vec-emisor-capacidad-v4`](cmd_y_configuracion.md#paquete-cmdvec-emisor-capacidad-v4) | Arranque, composicion y configuracion | Command vec-emisor-capacidad-v4 ejecuta exclusivamente el verificador COSE y emisor de capacidades V4. |
| [`cmd/vec-server`](cmd_y_configuracion.md#paquete-cmdvec-server) | Arranque, composicion y configuracion | Composicion canonica y arranque del servidor HTTP del portal VEC. |
| [`config`](cmd_y_configuracion.md#paquete-config) | Arranque, composicion y configuracion | Carga y validacion de la configuracion canonica por variables de entorno. |
| [`internal/app/bootstrap`](cmd_y_configuracion.md#paquete-internalappbootstrap) | Arranque, composicion y configuracion | Composicion de la API y montaje de modulos para el arranque. |
| [`internal/app/server`](cmd_y_configuracion.md#paquete-internalappserver) | Arranque, composicion y configuracion | Construccion del servidor HTTP con limites y tiempos canonicos. |
| [`internal/candidate/adapters/auth`](nucleo_candidate.md#paquete-internalcandidateadaptersauth) | Nucleo heredado de candidatos (Bolsa) | Autenticadores del nucleo heredado de Bolsa, incluido el fake local de pruebas. |
| [`internal/candidate/adapters/handler`](nucleo_candidate.md#paquete-internalcandidateadaptershandler) | Nucleo heredado de candidatos (Bolsa) | Handlers HTTP de la API Bolsa heredada. |
| [`internal/candidate/adapters/repository`](nucleo_candidate.md#paquete-internalcandidateadaptersrepository) | Nucleo heredado de candidatos (Bolsa) | Repositorios en memoria y durables del nucleo heredado de Bolsa. |
| [`internal/candidate/application`](nucleo_candidate.md#paquete-internalcandidateapplication) | Nucleo heredado de candidatos (Bolsa) | Casos de uso heredados de candidatos de Bolsa. |
| [`internal/candidate/domain`](nucleo_candidate.md#paquete-internalcandidatedomain) | Nucleo heredado de candidatos (Bolsa) | Tipos y reglas puras del dominio heredado de candidatos. |
| [`internal/candidate/ports`](nucleo_candidate.md#paquete-internalcandidateports) | Nucleo heredado de candidatos (Bolsa) | Contratos hexagonales del nucleo heredado de Bolsa. |
| [`internal/candidate/usecases`](nucleo_candidate.md#paquete-internalcandidateusecases) | Nucleo heredado de candidatos (Bolsa) | Casos de uso del flujo administrativo heredado. |
| [`internal/modules/administracion`](modulos_personal_cronos_dietas.md#paquete-internalmodulesadministracion) | Modulos Personal, Cronos, Dietas y Administracion | Manifiesto del modulo Administracion para el shell VEC. |
| [`internal/modules/bolsa`](modulo_bolsa.md#paquete-internalmodulesbolsa) | Modulo Bolsa | Manifiesto del modulo Bolsa: identidad, permisos y menus para el shell VEC. |
| [`internal/modules/bolsa/adapters/fichero`](modulo_bolsa.md#paquete-internalmodulesbolsaadaptersfichero) | Modulo Bolsa | Package fichero aporta únicamente una fuente local de demostración. |
| [`internal/modules/bolsa/adapters/httppublico`](modulo_bolsa.md#paquete-internalmodulesbolsaadaptershttppublico) | Modulo Bolsa | Package httppublico expone únicamente proyecciones públicas minimizadas. |
| [`internal/modules/bolsa/adapters/memory`](modulo_bolsa.md#paquete-internalmodulesbolsaadaptersmemory) | Modulo Bolsa | Package memory contiene adaptadores efimeros y defensivos del modulo de bolsas. |
| [`internal/modules/bolsa/adapters/postgres`](modulo_bolsa.md#paquete-internalmodulesbolsaadapterspostgres) | Modulo Bolsa | Package postgres implementa la persistencia durable del agregado de baremacion. |
| [`internal/modules/bolsa/application`](modulo_bolsa.md#paquete-internalmodulesbolsaapplication) | Modulo Bolsa | Package application contiene casos de uso del modulo de bolsa. |
| [`internal/modules/bolsa/domain`](modulo_bolsa.md#paquete-internalmodulesbolsadomain) | Modulo Bolsa | Package domain contiene las reglas puras del modulo de bolsas. |
| [`internal/modules/bolsa/internal/transaccion`](modulo_bolsa.md#paquete-internalmodulesbolsainternaltransaccion) | Modulo Bolsa | Package transaccion concentra la derivacion canonica de la evidencia probatoria del modulo Bolsa. |
| [`internal/modules/bolsa/ports`](modulo_bolsa.md#paquete-internalmodulesbolsaports) | Modulo Bolsa | Package ports declara contratos hexagonales del modulo de bolsas. |
| [`internal/modules/cronos`](modulos_personal_cronos_dietas.md#paquete-internalmodulescronos) | Modulos Personal, Cronos, Dietas y Administracion | Manifiesto del modulo Cronos: fichajes, permisos, vacaciones y saldos. |
| [`internal/modules/cronos/adapters/memory`](modulos_personal_cronos_dietas.md#paquete-internalmodulescronosadaptersmemory) | Modulos Personal, Cronos, Dietas y Administracion | Adaptadores en memoria del modulo Cronos. |
| [`internal/modules/cronos/application`](modulos_personal_cronos_dietas.md#paquete-internalmodulescronosapplication) | Modulos Personal, Cronos, Dietas y Administracion | Casos de uso del modulo Cronos. |
| [`internal/modules/cronos/domain`](modulos_personal_cronos_dietas.md#paquete-internalmodulescronosdomain) | Modulos Personal, Cronos, Dietas y Administracion | Reglas puras del dominio Cronos. |
| [`internal/modules/cronos/ports`](modulos_personal_cronos_dietas.md#paquete-internalmodulescronosports) | Modulos Personal, Cronos, Dietas y Administracion | Contratos hexagonales del modulo Cronos. |
| [`internal/modules/dietas`](modulos_personal_cronos_dietas.md#paquete-internalmodulesdietas) | Modulos Personal, Cronos, Dietas y Administracion | Manifiesto del modulo Dietas: comisiones, kilometraje y liquidaciones. |
| [`internal/modules/personal`](modulos_personal_cronos_dietas.md#paquete-internalmodulespersonal) | Modulos Personal, Cronos, Dietas y Administracion | Manifiesto del modulo Personal/Nominas. |
| [`internal/modules/personal/adapters/file`](modulos_personal_cronos_dietas.md#paquete-internalmodulespersonaladaptersfile) | Modulos Personal, Cronos, Dietas y Administracion | Adaptador de catalogo Personal sobre fichero local. |
| [`internal/modules/personal/adapters/memory`](modulos_personal_cronos_dietas.md#paquete-internalmodulespersonaladaptersmemory) | Modulos Personal, Cronos, Dietas y Administracion | Adaptador de catalogo Personal en memoria. |
| [`internal/modules/personal/application`](modulos_personal_cronos_dietas.md#paquete-internalmodulespersonalapplication) | Modulos Personal, Cronos, Dietas y Administracion | Casos de uso del modulo Personal/Nominas. |
| [`internal/modules/personal/domain`](modulos_personal_cronos_dietas.md#paquete-internalmodulespersonaldomain) | Modulos Personal, Cronos, Dietas y Administracion | Reglas puras del dominio Personal: RPT, puestos y categorias. |
| [`internal/modules/personal/ports`](modulos_personal_cronos_dietas.md#paquete-internalmodulespersonalports) | Modulos Personal, Cronos, Dietas y Administracion | Contratos hexagonales del modulo Personal. |
| [`internal/shared/i18n`](compartido.md#paquete-internalsharedi18n) | Paquetes compartidos | Catalogo de internacionalizacion compartido con fallback espanol. |
| [`internal/vec/adapters/almacen`](vec_adaptadores.md#paquete-internalvecadaptersalmacen) | Nucleo VEC: adaptadores | Package almacen contiene adaptadores que componen el puerto documental con conectores de objetos. |
| [`internal/vec/adapters/almacen/s3`](vec_adaptadores.md#paquete-internalvecadaptersalmacens3) | Nucleo VEC: adaptadores | Package s3 implementa un conector de objetos compatible con la API S3. |
| [`internal/vec/adapters/documentos/docx`](vec_adaptadores.md#paquete-internalvecadaptersdocumentosdocx) | Nucleo VEC: adaptadores | Package docx genera documentos Word Open XML sin macros ni recursos externos. |
| [`internal/vec/adapters/documentos/pdf`](vec_adaptadores.md#paquete-internalvecadaptersdocumentospdf) | Nucleo VEC: adaptadores | Package pdf genera la representacion PDF de trabajo mediante un adaptador reemplazable. |
| [`internal/vec/adapters/documentos/seguridad`](vec_adaptadores.md#paquete-internalvecadaptersdocumentosseguridad) | Nucleo VEC: adaptadores | Package seguridad contiene adaptadores criptograficos y de infraestructura local. |
| [`internal/vec/adapters/httpapi`](vec_adaptadores.md#paquete-internalvecadaptershttpapi) | Nucleo VEC: adaptadores | Adaptador HTTP del shell VEC: rutas publicas y privadas. |
| [`internal/vec/adapters/httpseguridad`](vec_adaptadores.md#paquete-internalvecadaptershttpseguridad) | Nucleo VEC: adaptadores | Package httpseguridad define la frontera de seguridad HTTP entre las superficies publica, personal, interna y de administracion. |
| [`internal/vec/adapters/memory`](vec_adaptadores.md#paquete-internalvecadaptersmemory) | Nucleo VEC: adaptadores | Adaptadores en memoria del nucleo VEC para pruebas y arranque local. |
| [`internal/vec/adapters/postgres`](vec_adaptadores.md#paquete-internalvecadapterspostgres) | Nucleo VEC: adaptadores | Package postgres contiene adaptadores duraderos del nucleo para PostgreSQL. |
| [`internal/vec/adapters/postgres/confianzadocumental`](vec_adaptadores.md#paquete-internalvecadapterspostgresconfianzadocumental) | Nucleo VEC: adaptadores | Package confianzadocumental implementa el conector PostgreSQL de ejecucion documental atestada V4. |
| [`internal/vec/adapters/seguridad`](vec_adaptadores.md#paquete-internalvecadaptersseguridad) | Nucleo VEC: adaptadores | Adaptadores criptograficos del nucleo: HMAC, AEAD y atestacion. |
| [`internal/vec/application`](vec_aplicacion.md#paquete-internalvecapplication) | Nucleo VEC: aplicacion y dobles de prueba | Casos de uso del shell VEC: modulos, auditoria, documentos, flujos y cotejo. |
| [`internal/vec/domain`](vec_dominio.md#paquete-internalvecdomain) | Nucleo VEC: dominio | Tipos puros del shell VEC, sin HTTP ni persistencia concreta. |
| [`internal/vec/ports`](vec_puertos.md#paquete-internalvecports) | Nucleo VEC: puertos | Contratos hexagonales del nucleo VEC: autorizacion, auditoria, documental y almacen. |
| [`internal/vec/pruebas`](vec_aplicacion.md#paquete-internalvecpruebas) | Nucleo VEC: aplicacion y dobles de prueba | Package pruebas contiene fabricas exclusivas para dobles automatizados. |
