# Plan del Canal interno de información — 4 de octubre de 2026

VEC ofrecerá acceso público al Sistema Interno de Información institucional y sus
normas, sin recoger denuncias, identidad o documentos. La primera entrega no exige
login de VEC ni relación de empleo. Recepción, acuses, seguimiento, investigación y
libro-registro permanecen en el Sistema institucional. Una integración mínima solo
se abrirá si la aprueba su Responsable. Este encargo entrega documentación, sin
código, SQL, instalación ni acceso a expedientes del canal.

Base inspeccionada: `origin/main@009472bd760e76cb2951433236711262f10648be`.
Requisitos: [ficha del Canal interno integrada](../estudio_requisitos/ficha_canal_interno_informacion_2026-10-04.md),
CAN1–CAN10. La cola vigente y las autoridades de identidad/autorización K,
núcleo/auditoría L y Documentos se conservan. Personal B no produce un vínculo
obligatorio para acceder al canal ni recibe denuncias.

## Fuentes y decisiones aplicables

La ficha contrastó la [página institucional](https://www.dipgra.es/e-administracion/administracion-electronica/sistema-interno-de-informacion/),
que enlaza el [destino Centinela/Lefebvre](https://centinela.lefebvre.es/lp/canal-denuncias/diputacion-granada),
la [Estrategia provincial de 2024](https://bop.dipgra.es/export/sites/bop/.galleries/Documentos-Anuncios-en-PDF/firmado-1723590017898-final-7e73c306-2.pdf?p=1779215866944)
y el [Reglamento definitivo de 2025](https://bop.dipgra.es/export/sites/bop/.galleries/Documentos-Anuncios-en-PDF/firmado-1751497250139-final-40f10e71-1.pdf).
La página conserva texto sobre una aprobación futura; el reglamento definitivo
publicado es la referencia procedimental de la ficha. Un enlace no acredita
disponibilidad, contrato, encargado, alojamiento, API ni garantías del proveedor.
No se ha enviado una comunicación ni interactuado con su formulario en este encargo.

La [Ley 2/2023](https://www.boe.es/eli/es/l/2023/02/20/2/con) y los
[RGPD](https://www.boe.es/doue/2016/119/L00001-00088.pdf)/
[LOPDGDD](https://www.boe.es/eli/es/lo/2018/12/05/3/con)
tienen el alcance recogido en la ficha. Sus reglas de anonimato, acceso y supresión
no son preferencias desactivables de RRHH. Se conservan separados acuse/respuesta
del art. 9, recepción del art. 32, investigación habilitada y libro-registro del
art. 26. El máximo de diez años del libro no se carga como retención general.
DPD/Responsable deben resolver cómo materializar recepción, investigación y copias
sin retrasar la supresión legal por la formulación provincial del apartado 13.

Aplican [materias reservadas, §8 y bóveda §8.1](../estudio_requisitos/materias_reservadas_economicas_y_relaciones_laborales.md),
[arquitectura](../portal_vec/arquitectura_tecnica.md),
[contrato de módulos](../portal_vec/contrato_modulos_vec.md),
[roles y ámbitos](../portal_vec/matriz_roles_y_ambitos.md) y
[cumplimiento](../portal_vec/cumplimiento_y_seguridad.md).
La duda 136 de [dudas.md](../../dudas.md) pide enlace o integración admitida,
aprobación, contrato, custodios/suplencias y política de eliminación. No se vuelve
a preguntar por la existencia del canal o del reglamento ni se modifica la duda.

## Inventario en la base inspeccionada

Se consultó primero `codebase-memory-mcp`, proyecto
`home-alberto-Trabajo-VEC_Diputacion_app-.worktrees-codexe-original-autorizacion-20261003`.
El índice representa otra instantánea; las rutas y límites siguientes se cotejaron
por `git ls-tree` y lectura en el SHA indicado.

| Pieza | Rutas actuales | Capacidad y límite |
| --- | --- | --- |
| Canal | `docs/estudio_requisitos/ficha_canal_interno_informacion_2026-10-04.md` | Ficha y fuentes integradas; no hay paquete del canal en `internal/modules/`, enlace modular específico montado, bandeja propia ni conector Centinela. |
| Registro/catálogos | `internal/vec/domain/types.go`; `internal/vec/application/service.go`; `internal/vec/ports/ports.go`; `web/static/portal-empleado/portal-catalogo-modulos.js`; `data/catalogos/` | Registro y configuración comunes reutilizables. Un menú interno no satisface el acceso público sin login. Entrada pública y montaje requieren turno de sus custodios; no se crea otro router. |
| Contexto y autorización | `internal/vec/ports/contexto_actor.go`; `internal/vec/adapters/contextoactor/postgres/`; `internal/vec/adapters/administracionperfiles/postgres/` | Autoridades nominales para cambiar configuración y, si se encarga, para gestores. No deben exigir persona/empleado al informante anónimo ni prestar un actor ADMIN material. |
| Auditoría | `internal/vec/ports/{auditoria_intento_nominal,auditoria_frontera_ruta_exacta}.go`; `internal/vec/adapters/postgres/auditoria_intento_nominal.go` | Destinos comunes para operaciones nominales e intentos fallidos/frontera. Falta el consumidor del cambio de configuración. Un intento técnico sin identidad no puede completarse con la persona del login ni con identificadores extra del informante. |
| Documentos/firma | `internal/vec/documentos/application/servicio.go`; `ports/{contratos,firma}.go`; `adapters/validadorautofirma/cliente.go` | Custodia/lecturas y verificación comunes. No acreditan compartimento documental del canal ni autorizan buscar denuncias en el repositorio general. Inicialmente solo se enlazan normas públicas; evidencias reservadas siguen con el custodio institucional. |
| Personal/Usuarios | `internal/modules/personal/ports/relacion_empleado.go`; `internal/modules/usuarios/application/{correos,correo_avisos}.go` | Vínculo y contactos de otras finalidades; no se consumen para identificar/correlacionar informante ni reenviar denuncias. El lector de correo de avisos actual es específico de llamamientos. |

No hay implementación en esta base de Disciplina, Igualdad o PRL como receptores
del canal. Sus fichas no constituyen adaptadores ni designaciones. Tener código
común integrado no demuestra instalación, configuración, publicación ni recorrido
del canal. Un enlace operativo tampoco acredita recepción o protección del proveedor.

## Huecos y propietarios

| Requisitos | Resultado pendiente | Propietario y dependencia |
| --- | --- | --- |
| CAN1–CAN3 | Entrada pública, explicación del ámbito y vías, normas/enlace versionados sin captar datos. | VEC mantiene configuración pública; el Sistema institucional mantiene el trámite y su admisibilidad. |
| CAN4/CAN6 | Designaciones y necesidad de conocer por ente/canal/expediente/actuación. | Responsable y equipo receptor independiente; K provisiona perfiles fijos nominales, sin RRHH genérico. |
| CAN5 | Comunicación/acuse/seguimiento, también anónimos, en el canal institucional. | Sistema institucional; VEC solo consume el estado/acuse mínimo autorizado si existe contrato. |
| CAN7 | Acceso, descarga/exportación minimizada con permiso propio en compartimento reservado. | Custodio y Responsable; Documentos solo por contrato segregado admitido, sin búsqueda general. |
| CAN8/CAN9 | Separar retención, suprimir/anonimizar y tratar copias/restauraciones. | DPD/Responsable/archivo del Sistema, según tratamiento y finalidad; VEC elimina sus mínimos cuando proceda. |
| CAN10 | Cambio de configuración y actuación nominal auditados; anonimato sin inventar persona. | Autoridad común L con consumidor específico; referencia anónima solo si se admite integración anónima. |

La entrada pública no transmite sesión, persona, datos laborales, cabecera de
referencia o parámetros personales al proveedor. Se usa el destino oficial
configurado sin redirección instrumentada, seguimiento individual o vínculo con
login. No hay formulario VEC, copia en RRHH, bot, correo general, ficha Personal o
índice de búsqueda. No se intenta identificar al informante para auditar el acceso
al enlace ni se añade IP, dispositivo o correlación nominal para descubrirlo.

Cambios de configuración y operaciones futuras de gestores tienen actor nominal,
perfil fijo activo provisionado por huella/CAS y autorización positiva exacta:
ente, canal, expediente/actuación, recurso, acción, finalidad, campos y vigencia.
Gestor, Responsable colegiado, investigador y acceso excepcional de RRHH/Jurídico
conservan funciones distintas, designación y ausencia de conflicto. Soporte no se
convierte en gestor; una investigación no concede autoridad para resolver disciplina.
Ningún permiso se publica al recibir la petición ni se infiere del menú.

Para cada operación, lectura y descarga nominal, la auditoría común registra actor,
perfil, acción, recurso opaco, finalidad, instante, `permitido/denegado/error`,
correlación, proceso y canal. Permitidos acompañan lectura/efecto; denegados/errores
usan la autoridad común después del cierre del intento. La frontera sin identidad
conserva solo la traza técnica mínima, sin fabricar persona. Si se admite un envío
anónimo integrado, la ficha exige referencia anónima de envío, sin enriquecimiento
desde Usuarios/Personal ni correlación con otra sesión. El contrato común deberá
soportarla expresamente; no se fuerza un actor nominal sintético para reutilizarlo.

Autorización, estado mínimo, versión, recibo, historia de solo adición y auditoría
del efecto se confirman en la misma transacción; outbox solo para derivación
admitida con consumidor, acuse e idempotencia. Libro-registro legal y auditoría
técnica tienen finalidades distintas. Los logs/eventos no conservan relato,
afectados, contacto o anexos; la traza de expurgo no perpetúa esos datos. Una
referencia seudónima sigue siendo personal si permite reidentificar.

Puertos neutrales y versionados, dominio/aplicación sin SQL/HTTP/proveedor,
adaptadores en composición y referencias opacas. Ninguna consulta/escritura de
tablas ajenas. Disciplina/Igualdad/PRL reciben solo la actuación habilitada por el
responsable competente; Personal, Cronos, Bolsa y Nómina no consumen denuncias.

## Configuración y dudas

Enlace, normas, ente/canal, responsables/suplencias y eventual contrato conservan
fuente/apartado, publicación, versión/huella, vigencia/efectos y órgano aprobador.
La adhesión de un ente requiere su acto; no se deduce de compartir infraestructura.
Anonimato y supresión legales son invariantes, no opciones de RRHH. En el primer
corte VEC conserva solo esta configuración pública y la autoridad nominal del cambio.

La duda 136 habilita decidir el alcance de integración, interfaz y custodios.
CAN1–CAN3 pueden avanzar como acceso informativo; C05 en adelante esperan encargo
expreso y aprobación del Responsable. RAT, bases por tratamiento, información,
encargados, riesgos/EIPD cuando proceda y eliminación se documentan antes de
intercambiar datos reales. No se usa consentimiento genérico como fundamento ni
se conserva todo el contenido por trazabilidad.

## Minitareas y salidas por PR

Dirección confirma SHA actual y archivos exclusivos. Rutas nuevas son previsiones;
los cambios comunes y de entrada pública los producen sus custodios. B1 es
autorización/auditoría nominal; B3 configuración; B5 documentos; B6 intercambio.
La base B2 de vínculo laboral no es requisito del informante. Las horas comunes
de K/L y del Sistema institucional se separan del consumo propio de VEC.

| Corte y base | Salida usable por PR | Archivos propios previstos | Dependencias | Horas |
| --- | --- | --- | --- | ---: |
| C00 · inventario | Revalidar destino/normas en base vigente y qué mantiene cada custodio. | Este plan, en su turno. | Ficha; sin inspección de expedientes ni formulario del proveedor. | 1–2 |
| C01 · B3/CAN3 | Fuente pública versionada de enlace/normas con consumidor y validación de destino. | Nuevos datos `data/catalogos/canal-interno/` y lector de configuración propio. | Fuente institucional; enlace sin campos de persona/empleado/sesión. | 2–3 |
| C02 · B1/CAN10 | Cambio nominal de configuración con permiso exacto/auditoría; lectura pública solo devuelve campos públicos. | Consumidor propio de configuración/auditoría; común por custodio. | C01, K/L y gobierno existente; sin historial nominal de visitas al canal. | 4–6 |
| C03 · CAN1/2 | Acceso público útil al canal y normas, ámbito/vías claros y ayuda desde catálogos. | Vista/cliente/textos propios en nueva superficie pública; montaje por dueño. | C01/C02; sin login/certificado, sin recoger hechos o documentos, revisión usabilidad. | 3–5 |
| C04 · entrega inicial | Chrome comprueba enlace, teclado, público sin login y ausencia de identidad/referrer/parámetros transmitidos. | Pruebas focales/recorrido y ayuda propias. | C03; revisión de frontera/usabilidad; no acredita recepción del proveedor. | 2–4 |
| C05 · decisión, opcional | Contrato mínimo autorizado por Responsable y DPD, límites/datos/retención y sistema de referencia fijados. | Contrato junto al futuro adaptador; actualización de este plan en su turno. | 136 resuelta para integración; proveedor/Sistemas admiten interfaz; sin inferir API. | 3–5 |
| C06 · B1/CAN4/6 | Consumidor nominal segregado de gestores con consulta focal y los tres resultados auditados. | Nuevos `canalinterno/ports/{autorizacion,auditoria}.go`, consumidor propio. | C05, K/L, designaciones/perfiles fijos, conflictos y superficie segregada por custodio. | 6–10 |
| C07 · B6/CAN5 | Puerto/adaptador y consumidor de estado/acuse mínimos desde Sistema institucional. | `ports/canal.go`, `adapters/institucional/` y caso de uso propios. | C05/C06; sin cuerpo, identidad protegida o libro paralelo; anónimo solo conforme contrato. | 4–6 |
| C08 · B6 | Derivación mínima admitida con recibo, reintento y conciliación sin duplicar comunicación institucional. | Entrega/reconciliación propias; SQL reservado si conserva estado mínimo. | C07, acuse del receptor; SQL en borrador, ensayo y dos revisiones antes de instalar. | 5–8 |
| C09 · B5/CAN7 | Consulta/descarga excepcional minimizada desde custodio con permiso propio, sin exportación general. | Consumidor documental reservado y controles propios. | C05/C06; contrato de custodia y necesidad de conocer; entrega de contenido solo donde se habilite. | 4–7 |
| C10 · CAN8/9 | Política operativa implementable para mínimos VEC, copias y eliminación por compartimento. | Consumidor/plan de conservación propio y pruebas focales. | DPD/Responsable y C05; recepción/investigación/libro separados en su sistema, sin plazo general diez años. | 5–8 |
| C11 · CAN9/10 | Ensayo sintético de supresión/restauración y referencia anónima sin identificación ni reintroducción. | Pruebas de integración/custodia y restauración propias. | C08–C10; auditar expurgo mínimo; anónimo permanece sin persona artificial. | 4–7 |
| C12 · CAN6 | Derivación habilitada a órgano concreto con campos mínimos, acuse y rechazo de acceso recíproco general. | `ports/derivacion.go`, consumidor/reconciliación propios. | C06/C08; finalidad legal y receptor admitidos por sus dueños; excluye identidad protegida por defecto. | 4–7 |
| C13 · entrega opcional | Recorrido segregado sintético, negativos/reinicio y manual con efectos realmente admitidos. | Pruebas focales/recorrido y manual propios. | C06–C12 instalados/configurados; revisiones exactas, sin envío real de denuncia. | 5–8 |

Cada PR entrega su consumidor con resultado observable dentro del alcance. C03
no añade un botón detrás del login como sustituto del acceso público. C07/C08
no implementan recepción propia ni trasladan una bandeja RRHH. C09 se elimina si
el acuerdo solo admite metadatos, recalculando la estimación y dejando CAN7 bajo
el Sistema. Si ningún intercambio se autoriza, se entregan C00–C04 y el resto
permanece como horizonte condicionado, sin rutas falsas de indisponibilidad.

SQL propio solo sería necesario para estado mínimo/recibos de una integración
admitida. Se reserva número fuera de Git antes de escribir, con preimagen/orden
causal; permanece en borrador hasta ensayo en clon principal y dos revisiones
exactas. MCP `postgres-clon-local` solo para lectura/EXPLAIN; dirección instala.
No copiar el libro-registro, reaplicar migraciones, ejecutar DOWN sobre historia
o SQL en cidonia desde A. La disponibilidad del conector no acredita garantías
jurídicas ni operativas del proveedor.

## Dependencias, paralelismo y estimación

El corte de enlace C00–C04 suma **12–20 horas técnicas**: **2–3 jornadas de un
equipo** de ocho horas, redondeadas al día completo. Su cadena inventario→fuente→
cambio gobernado→entrada→comprobación limita el solape: con dos equipos se mantiene
la previsión conservadora de **2–3 jornadas**; no se divide por dos ese esfuerzo.

La integración opcional C05–C13 suma **40–66 horas técnicas** adicionales. El
horizonte completo suma **52–86 horas**, **7–11 jornadas de un equipo**. Con dos
equipos se estiman **5–9 jornadas** para el horizonte completo: C00–C04 12–20 h;
C05/C06 9–15 h; C07–C10 9–15 h con adaptador/entrega en un equipo y descarga/
conservación en el otro; C11/C12 4–7 h en paralelo y por archivos propios;
C13 5–8 h. Camino **39–65 h**, redondeado a **5–9 jornadas**. Es la horquilla de asignación
una vez admitido el contrato; si el solape no es posible se vuelve a 7–11.
Un equipo conserva adaptador/acuse/derivación y otro descarga/conservación/pruebas;
el contrato de anonimato y la auditoría común tienen un responsable. El Sistema,
K/L y DPD no se cuentan como equipos adicionales implícitos.

| Trabajo externo, fuera del total técnico | Dedicación orientativa | Condición |
| --- | ---: | --- |
| Responsable del Sistema | 2–4 h iniciales; 4–8 h si integra | Validar enlace/alcance; después aprobar 136, designaciones y contrato mínimo. Espera sin fecha comprometida. |
| DPD/Archivo/Seguridad | 5–10 h si integra | Bases/encargados, separación de conservación, minimización, copias y restauración; sin promesa de cierre formal. |
| Sistemas/proveedor institucional | 6–12 h si integra | Confirmar interfaz admitida, garantías y entorno sintético; ninguna API se presume desde el enlace. |
| K/L, Documentos y receptores autorizados | Estimación por sus propietarios | Capacidad de actor anónimo si procede, perfiles de gestores, auditoría/custodia segregadas y consumidores; fuera de C06/C09/C12. |

Las horas miden esfuerzo del consumo VEC; no incluyen adjudicación/contrato,
auditoría completa del proveedor ni espera de responsables. El alcance y sus
horquillas se revisan al resolver 136.

## Comprobación y entrega

El plan comprueba enlaces locales, inventario, sumas y `git diff --check`. No
necesita Go, SQL, servicios, contenedores o navegador para el cambio documental.
Su PR no acredita recepción, garantías del proveedor, instalación o habilitación
para datos reales. No se prueba el canal enviando una denuncia de ensayo real.

La implementación aplicará `programar-backend-vec`, `persistir-autorizar-vec`,
`programar-interfaz-vec`, `probar-recorridos-vec` según el corte. Antes de pantallas:
`usabilidad-vec`, `aspecto-vec`, `disenar-sistema-visual-vec`,
`impeccable`/`VEC-PRIORIDAD.md`, revisión independiente de usabilidad. Textos con
`humanizer`/`VEC-USO.md`; SQL con `revisar-sql-vec`/`ensayar-sql` y revisiones
exactas; `security-audit` focal y Semgrep local sin enviar código fuera. Go usa
gopls; búsqueda primero por índice. `revisar-cambios-vec`,
`documentar-entregar-vec`, `pr-vec` cierran entrega; dirección integra/despliega.

Chrome del sistema PC/móvil y teclado/foco para la entrada sin login, referrer,
parámetros o identificadores personales. Integración futura: datos sintéticos,
anonimato y segregación por canal/ente, revocación/conflicto, descarga sin permiso,
dependencia caída y auditoría de permitido/denegado/error sin relato ni identidad
protegida. PG real/reinicio/replay donde haya estado propio; ensayo de supresión y
restauración que no reintroduzca contenidos. Las lecturas nominales de gestores
y el uso anónimo del canal tienen contratos diferentes.
