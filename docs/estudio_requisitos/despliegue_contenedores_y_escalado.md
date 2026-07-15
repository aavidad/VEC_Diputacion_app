# Arquitectura de despliegue, contenedores y escalado

Estado: **decisión arquitectónica de trabajo pendiente de validación por Sistemas y
Seguridad**.

Fecha de corte: 14 de julio de 2026.

## 1. Decisión ejecutiva

La aplicación se empaquetará desde el primer día en imágenes OCI y se preparará para ser
operada en el futuro por una plataforma Kubernetes. **La primera implantación de Bolsa no
necesita levantar todavía Kubernetes**: se propone un perfil austero de tres máquinas
virtuales con NGINX y contenedores de un solo anfitrión, gestionados declarativamente por
`systemd` y Podman Quadlet o Docker Compose v2, según la tecnología que Sistemas ya sepa
mantener.

Esta reducción aplaza alta disponibilidad y autoescalado, pero no autenticación real,
autorización, segmentación, auditoría remota, cuarentena, antivirus, copias externas ni
restauraciones ensayadas. El perfil completo de dos clústeres sigue siendo la arquitectura
objetivo cuando se incorporen Cronos y otros módulos internos o la carga y los objetivos de
continuidad lo exijan.

No se creará automáticamente un contenedor por cada módulo funcional. Una capacidad se
convierte en unidad de despliegue independiente cuando concurre al menos una de estas
razones:

- necesita una frontera de seguridad o red distinta;
- debe escalar de forma independiente;
- tiene un ciclo de publicación diferente;
- requiere recursos especializados;
- interesa aislar sus fallos o mantenimiento;
- contiene datos con reglas de custodia incompatibles con las demás superficies.

Por ello, Cronos será una unidad interna separada; una pantalla, catálogo o tipo de
certificado nuevo no lo será. El diseño evita tanto el monolito único expuesto a todos los
entornos como una proliferación innecesaria de microservicios.

### 1.1 Perfil de implantación aprobado para dimensionar la fase inicial

```text
Internet / protección corporativa
              |
      VM 1 - NGINX en DMZ
              |
      flujo único autorizado
              |
 VM 2 - aplicación Go y trabajadores
              |
      cuentas y puertos mínimos
              |
 VM 3 - PostgreSQL y documentos

Administración -> bastión/VPN + MFA
Logs          -> SIEM/syslog corporativo externo
Copias        -> repositorio fuera de las tres VM y de su hipervisor
```

- **VM 1, exposición:** NGINX, TLS, límites y proxy; no alberga bases ni documentos.
- **VM 2, servicios:** núcleo modular Go con Bolsa, adaptadores web/API, superficie interna
  de tramitación y trabajadores. Las superficies pública e interna usarán puntos de
  escucha, cuentas y credenciales distintos aunque compartan temporalmente anfitrión.
- **VM 3, datos:** PostgreSQL y almacenamiento documental en volúmenes separados, sin
  acceso desde Internet. El acceso se limitará por origen, servicio, base/esquema y rol.
- Los servicios corporativos existentes de identidad, PKI, WAF, DNS/NTP, correo,
  antivirus, registro, SIEM y copias se reutilizarán antes de instalar equivalentes.
- Cronos, GPS, Nóminas y el resto de módulos internos sensibles quedan fuera de este
  despliegue. Cronos no se añadirá a estas VM: tendrá su enclave interno antes de tratar
  datos reales.

Si solo se pueden reservar dos VM, se unirán NGINX y aplicación en la primera y se
mantendrán PostgreSQL y documentos en la segunda. No se unirá la base de datos con el
servidor expuesto. Esta variante exige copia y logs externos, endurecimiento reforzado y
aceptación expresa de su menor aislamiento, disponibilidad y capacidad de mantenimiento.
Una sola VM se limitará a desarrollo, demostración o piloto sin datos personales reales.

CPU, RAM, IOPS y disco no se fijarán por intuición. Como referencia de partida se medirá
una VM pequeña para NGINX, una VM de aplicación dimensionada también para el antivirus y
una VM de datos con SSD; la prueba de carga, el tamaño de documentos y el producto
antivirus determinarán los valores finales. Varias VM alojadas en el mismo hipervisor
aportan segregación lógica, pero no alta disponibilidad.

## 2. Módulo lógico, capacidad configurable y unidad de despliegue

Son conceptos distintos:

| Concepto | Ejemplo | ¿Exige imagen nueva? | ¿Puede activarse desde la aplicación? |
| --- | --- | --- | --- |
| Dato o regla configurable | Nuevo tipo de certificado, plazo o categoría | No | Sí, tras el circuito de aprobación |
| Módulo lógico ya instalado | Bolsa o Dietas en un entorno autorizado | No | Sí, con permiso, auditoría y comprobaciones |
| Nueva versión de un módulo | Cambio de lógica o nueva capacidad | Sí | La promoción se inicia desde un flujo gobernado; no se carga código arbitrario |
| Unidad de despliegue | Portal público, Cronos, trabajador antivirus | Sí | Escala o se detiene mediante el orquestador |
| Conector | PostgreSQL, Oracle, antivirus o proveedor de avisos | Sí cuando se incorpora uno nuevo | Se selecciona por configuración una vez instalado y certificado |

Un administrador funcional podrá publicar reglas, formularios, calendarios y catálogos sin
recompilar. Un operador autorizado podrá habilitar o deshabilitar un módulo que ya esté
instalado y aprobado. Añadir código, una imagen o un conector desconocido seguirá siendo un
cambio de software: deberá compilarse, analizarse, firmarse, desplegarse y probarse.

No se permitirá instalar imágenes o complementos desde Internet ni subir código desde el
panel de administración.

## 3. Separación física de las superficies

La arquitectura objetivo para el portal completo mantiene, como mínimo, dos clústeres y
dominios de seguridad sin nodos ni plano de control compartidos:

```text
Internet
   |
   v
WAF/balanceador público
   |
   v
Clúster de la DMZ
   +-- NGINX Gateway público
   +-- portal y API públicos
   +-- Bolsa externa y procesos selectivos
   +-- bot/MCP exclusivamente públicos
   +-- trabajadores asíncronos autorizados
   X  sin ruta hacia Cronos

Red corporativa Mulhacén
   |
   v
WAF/balanceador interno + identidad corporativa
   |
   v
Clúster interno
   +-- NGINX Gateway interno
   +-- portal del empleado y backoffice
   +-- Cronos
   +-- servicios cartográficos internos
   +-- conectores corporativos
   +-- trabajadores y datos internos

Red de gestión separada
   +-- bastiones y administración privilegiada
   +-- registro de imágenes y repositorios aprobados
   +-- secretos, PKI/HSM, observabilidad, SIEM y copias
```

La información pública que nazca en el entorno interno se transferirá a una proyección
específica y minimizada. El portal público nunca consultará directamente las bases de
Cronos, Personal, Nóminas o Dietas.

Esta topología objetivo no obliga a pagar dos clústeres durante la primera Bolsa. La
excepción temporal descrita en el apartado 1.1 solo puede contener Bolsa y su backoffice;
antes de incorporar Cronos o datos internos ajenos al proceso selectivo se construirá la
frontera interna separada.

Los entornos de desarrollo, integración, preproducción y producción también estarán
separados. Los datos reales no se copiarán a entornos no productivos.

## 4. Plataforma recomendada

### 4.1 Plataforma objetivo

Se recomienda una distribución Kubernetes con soporte y endurecimiento mantenido:

- **RKE2** es la propuesta base si se busca una plataforma Kubernetes contenida, instalable
  en servidores locales y compatible con operación sin conexión general a Internet.
- **Red Hat OpenShift** es una alternativa válida cuando la Diputación prefiera una
  plataforma integrada con soporte empresarial, operadores y herramientas de gobierno, y
  disponga de presupuesto y personal especializado.
- Una plataforma Kubernetes corporativa ya existente prevalecerá si satisface las
  fronteras, soporte, ENS, red, copias y operación aquí descritos.

La elección definitiva corresponde a Sistemas tras valorar soporte, experiencia,
licencias, infraestructura actual, continuidad y coste total. RKE2 documenta para alta
disponibilidad un número impar de nodos de servidor —tres recomendados— y una dirección
estable delante de ellos: <https://docs.rke2.io/install/ha>.

Para la primera implantación austera podrá emplearse Podman Quadlet o Docker Compose v2 en
un solo anfitrión por zona, con unidades `systemd`, reinicio controlado, comprobaciones de
salud, límites y despliegue reproducible. Docker documenta expresamente el despliegue con
Compose en un único servidor y el uso de configuración adicional de producción:
<https://docs.docker.com/compose/how-tos/production/>. Si Sistemas adopta Podman, Quadlet
permite describir contenedores gestionados por `systemd` sin introducir un clúster:
<https://docs.podman.io/en/stable/markdown/podman-quadlet.1.html>.

No se instalará Kubernetes de un nodo únicamente como puente: puede recrear un proceso,
pero no sobrevivir a la pérdida de la VM ni crear CPU o memoria. RKE2 se introducirá cuando
se pueda operar la topología objetivo o cuando ya exista una plataforma corporativa. Docker
Swarm no se propone como destino principal.

### 4.2 Imágenes y motor de contenedores

Los artefactos serán imágenes **OCI**, fijadas por versión y huella. Kubernetes podrá usar
`containerd` u otro motor compatible; no existe la obligación de instalar Docker Engine en
los nodos.

Reglas mínimas:

- nunca utilizar `latest` en producción;
- ejecutar como usuario no privilegiado;
- sistema de archivos raíz de solo lectura cuando sea viable;
- eliminar capacidades Linux y prohibir elevación de privilegios;
- perfiles `seccomp` y AppArmor/SELinux;
- no montar el socket del motor de contenedores;
- límites y reservas de CPU, memoria y almacenamiento efímero;
- imagen mínima, SBOM, análisis de vulnerabilidades, firma y procedencia verificadas.

## 5. NGINX, Gateway API y balanceo

NGINX terminará o reenviará TLS según la arquitectura aprobada, aplicará límites,
cabeceras, tamaños, tiempos y reglas de enrutamiento y distribuirá tráfico únicamente a
réplicas preparadas.

NGINX no arranca contenedores. El flujo será:

```text
métricas, calendario o profundidad de cola
                  |
                  v
             HPA / KEDA
                  |
                  v
        Kubernetes ajusta réplicas
                  |
                  v
     Service publica extremos preparados
                  |
                  v
 NGINX Gateway envía tráfico solo a esos extremos
```

En la plataforma Kubernetes objetivo se utilizará **Gateway API** con una implementación
mantenida, por ejemplo NGINX Gateway Fabric. Su plano de control observa los recursos
Gateway y crea o mantiene los planos de datos NGINX:
<https://docs.nginx.com/nginx-gateway-fabric/>. La primera fase empleará NGINX autónomo con
configuración versionada; migrar a Gateway API solo cambia el adaptador de despliegue.

No se utilizará el controlador comunitario `ingress-nginx`: su mantenimiento terminó en
marzo de 2026 y ya no recibe correcciones de seguridad. El propio proyecto Kubernetes
recomienda migrar a Gateway API:
<https://kubernetes.io/blog/2025/11/11/ingress-nginx-retirement/>.

Cada zona tendrá su propia pasarela y certificados. El NGINX público no conocerá rutas ni
servicios internos. Se desactivará toda telemetría saliente no autorizada; NGINX Gateway
Fabric documenta expresamente su opción de exclusión:
<https://docs.nginx.com/nginx-gateway-fabric/overview/product-telemetry/>.

El WAF puede situarse delante de NGINX o integrarse en el producto escogido. La decisión
entre WAF corporativo, NGINX Plus/F5 WAF u otra solución soportada debe tomarla Sistemas y
Seguridad; NGINX Open Source por sí solo no sustituye a un WAF.

## 6. Cómo se escala

### 6.0 Primera fase: recuperación automática, capacidad medida y preescalado manual

En el perfil austero no habrá HPA, KEDA ni creación automática de VM. `systemd` y el motor
de contenedores recuperarán un proceso terminado; las comprobaciones de salud alertarán al
operador. Una tabla duradera de trabajos en PostgreSQL absorberá los picos asíncronos.

Antes de un cierre conocido se aplicará un procedimiento versionado: comprobar margen de
CPU, memoria, conexiones y disco; aumentar la concurrencia del trabajador o arrancar una
segunda réplica Go; ejecutar una prueba breve; y revertir después del vaciado. Una segunda
réplica en la misma VM aumenta capacidad o facilita una actualización, pero no proporciona
alta disponibilidad frente al fallo del anfitrión.

La aplicación nunca recibirá el socket del motor de contenedores ni privilegios para crear
recursos. El arranque corresponde a definiciones aprobadas de Operación. El autoescalado
descrito en los apartados siguientes es el destino posterior, no un requisito de compra de
la fase inicial.

### 6.1 Servicios web permanentes

El escalado horizontal de Kubernetes —HPA— aumentará o reducirá réplicas utilizando CPU,
memoria y, preferiblemente, métricas de servicio como:

- peticiones por segundo;
- concurrencia;
- latencia percentil 95/99;
- conexiones activas;
- saturación de los grupos de conexión;
- tasa de errores.

El bucle HPA tiene un intervalo predeterminado de 15 segundos y admite métricas
personalizadas y múltiples:
<https://kubernetes.io/docs/concepts/workloads/autoscaling/horizontal-pod-autoscale/>.

Portal, API, identidad, Cronos y las pasarelas críticas mantendrán un mínimo de dos réplicas
en producción, distribuidas entre nodos. No se reducirán a cero.

### 6.2 Trabajos asíncronos

KEDA podrá escalar trabajadores por eventos o profundidad de cola. Es adecuado para:

- antivirus y desarme de documentos;
- generación de PDF y audio;
- firma o validación por lotes;
- importaciones y exportaciones;
- correo, Telegram y otros avisos;
- cálculo masivo de baremos o listados;
- procesamiento cartográfico que sea paralelizable.

Cuando no exista trabajo, los trabajadores no críticos podrán quedar a cero; al llegar un
evento KEDA activa el despliegue y alimenta a HPA:
<https://keda.sh/docs/2.20/concepts/scaling-deployments/>.

### 6.3 Preescalado conocido

Los cierres de plazo son previsibles. La plataforma deberá permitir elevar de antemano el
mínimo de réplicas mediante calendario, además de reaccionar a métricas. KEDA admite
ventanas horarias de escalado:
<https://keda.sh/docs/2.20/scalers/cron/>.

Ejemplo: aumentar capacidad varias horas antes del último día de una convocatoria y no
reducirla hasta que hayan terminado las cargas pendientes.

### 6.4 Qué significa «arrancar en segundos»

Crear una réplica puede tardar segundos si:

- ya existe un nodo con CPU y memoria libres;
- la imagen está precargada en ese nodo o en un registro local rápido;
- el binario Go inicia con rapidez;
- las comprobaciones de arranque y preparación están bien diseñadas;
- no hay migraciones ni dependencias lentas en el camino de arranque.

Crear una máquina virtual, añadir un nodo físico, descargar imágenes o restaurar una base de
datos normalmente tarda bastante más. Por ello, para absorber picos en segundos se
mantendrá capacidad caliente y se precargarán imágenes. El objetivo exacto —por ejemplo,
réplica preparada en menos de 30 segundos— deberá validarse mediante pruebas en la
infraestructura real; no se prometerá antes de medirlo.

El autoescalado de nodos solo se habilitará si el hipervisor o nube privada dispone de una
API soportada y segura. No sustituye a la reserva de capacidad para los plazos críticos.

### 6.5 Los dos bucles automáticos

El sistema automático completo necesita dos controles encadenados:

```text
1. Escalado de aplicación

Prometheus/KEDA -> HPA -> más réplicas de Bolsa -> Service -> NGINX
                              |
                              +-- si caben en los nodos: arranque inmediato
                              +-- si no caben: quedan pendientes

2. Escalado de infraestructura

réplicas pendientes -> autoescalador de nodos -> API del hipervisor/nube privada
                     -> nueva VM/nodo -> réplica preparada -> Service -> NGINX
```

El primer bucle forma parte de Kubernetes. El segundo exige que la infraestructura
corporativa pueda crear o encender máquinas desde una plantilla mediante una API soportada.
La integración concreta dependerá de que la Diputación utilice VMware, OpenStack,
OpenShift, Harvester, otro hipervisor o servidores físicos.

En servidores físicos sin una capa de aprovisionamiento no se puede fabricar CPU o memoria
automáticamente. Se pueden encender nodos ya instalados mediante su gestión remota, pero
para obtener respuesta rápida es preferible mantener parte de ellos activos y disponibles.

El proveedor de nodos será otro adaptador de infraestructura: el contrato de capacidad no
quedará mezclado con Bolsa ni con el núcleo de RRHH.

Kubernetes documenta dos familias mantenidas: Cluster Autoscaler, que aumenta o reduce
grupos de nodos previamente configurados, y Karpenter, que aprovisiona según restricciones
cuando existe integración con el proveedor. Ambas necesitan hablar con la API concreta que
crea las máquinas virtuales; no son universales ni añaden hardware físico por sí solas:
<https://kubernetes.io/docs/concepts/cluster-administration/node-autoscaling/>. Para un CPD
local se elegirá la que soporte oficialmente el hipervisor y la distribución adoptados.

### 6.6 Perfil de escalado obligatorio por unidad

Cada unidad de despliegue declarará junto a su manifiesto:

- si es permanente, programable o apta para reducirse a cero;
- réplicas mínimas, ordinarias, máximas y de contingencia;
- CPU, memoria y almacenamiento solicitados y máximos;
- métricas y umbrales que justifican aumentar o reducir;
- tiempo máximo de arranque y de preparación;
- plazo de drenaje y trabajos que no pueden interrumpirse;
- distribución entre nodos y presupuesto de interrupción;
- conexiones máximas a base de datos y otros servicios compartidos;
- imagen que debe precargarse;
- dependencia de cola y reglas de idempotencia;
- propietario funcional y operador responsable.

Estos valores se publicarán como configuración de operación versionada, no como constantes
dispersas. La aplicación podrá mostrar estado, capacidad y recomendaciones, pero solo el rol
de Operación podrá cambiar límites; elevar máximos por encima de la capacidad aprobada
requerirá autorización y quedará auditado.

### 6.7 Ejemplo de funcionamiento para Bolsa

Valores meramente iniciales para una prueba, no para producción:

1. Bolsa mantiene dos réplicas preparadas.
2. Prometheus mide peticiones, concurrencia, duración, errores, CPU y saturación del grupo
   de conexiones.
3. HPA calcula réplicas usando las métricas aprobadas y puede crecer hasta el máximo de
   ensayo.
4. Kubernetes coloca cada réplica en un nodo distinto cuando sea posible.
5. La réplica carga configuración, comprueba PostgreSQL/S3/identidad y solo entonces supera
   su comprobación de preparación.
6. El `Service` añade su dirección y NGINX empieza a enviarle peticiones sin editar su
   configuración a mano.
7. El último día de plazo, KEDA o el programador eleva el mínimo antes del pico previsto.
8. Al terminar, una ventana de estabilización evita apagar y encender réplicas de forma
   oscilante; se drenan conexiones antes de retirarlas.
9. Si no hay sitio en los nodos, se solicita capacidad al proveedor de infraestructura o se
   alerta para activar la reserva manual prevista.

Los umbrales —por ejemplo CPU objetivo, peticiones por réplica, latencia admisible y máximo
de réplicas— se obtendrán con una prueba de carga que incluya base de datos, documentos,
firma y antivirus. Copiar cifras genéricas podría trasladar el cuello de botella a
PostgreSQL o a una integración.

## 7. Matriz objetivo de unidades de despliegue

| Unidad | Zona | Estado mínimo | Escalado propuesto | Observación |
| --- | --- | --- | --- | --- |
| NGINX Gateway público | DMZ | 2 réplicas | HPA por conexiones/peticiones | Solo rutas públicas |
| Portal/API pública y Bolsa | DMZ | 2 réplicas | HPA + preescalado | Sin acceso directo a datos internos |
| Bot/MCP público | DMZ | 2 réplicas o límite aprobado | HPA con cuotas | Solo corpus y datos publicados |
| NGINX Gateway interno | Mulhacén | 2 réplicas | HPA controlado | Sin exposición a Internet |
| Portal interno | Mulhacén | 2 réplicas | HPA | Identidad corporativa |
| Cronos | Mulhacén restringida | 2 réplicas | HPA; nunca cero | Artefacto y datos exclusivos |
| Ingesta GPS | DMZ telemática restringida | 2 réplicas | HPA por mensajes | APN privada, mTLS y antirrepetición |
| Trabajadores documentales | Zona que corresponda al dato | 0/1 o más | KEDA por cola | Cuarentena antes de abrir |
| Servicio de teselas/OSRM/geocodificación | Mulhacén | 2 réplicas o HA equivalente | Réplicas/caché | Sin respaldo externo automático |
| PostgreSQL | Zona de datos | HA administrada | Capacidad planificada | No se clona como un servicio web |
| Almacenamiento de objetos | Zona de datos | HA administrada | Según producto | API compatible S3 como puerto |
| Cola de mensajes | Por zona | HA si es crítica | Particiones/consumidores | Producto pendiente de elección |

Los mínimos de esta tabla corresponden a la plataforma objetivo, no a la fase austera. Son
una base de diseño, no un dimensionamiento. Volumetría, RTO, RPO y pruebas de carga
determinarán las cifras definitivas.

## 8. Datos y componentes con estado

El escalado de aplicaciones sin estado no resuelve la capacidad de la base de datos. En la
fase austera se admite un único PostgreSQL, con roles mínimos, copia base, archivado WAL,
recuperación a un punto en el tiempo y restauraciones fuera de la VM. Se acepta así un RTO
de recuperación manual, no alta disponibilidad. PostgreSQL documenta su copia continua y
PITR en <https://www.postgresql.org/docs/current/continuous-archiving.html>.

La plataforma objetivo añadirá alta disponibilidad, réplicas cuando aporten valor y un
gestor de conexiones si las medidas lo justifican. La versión mayor se elegirá dentro de
las soportadas y se mantendrá siempre en la revisión menor vigente; el proyecto PostgreSQL
publica su política y calendario en
<https://www.postgresql.org/support/versioning/>.

La posible adopción futura de Oracle se resolverá con un adaptador y un plan de migración,
no ejecutando ambas bases como si fueran intercambiables en caliente.

Los documentos se abstraerán mediante un puerto de almacenamiento. Si ya existe un S3
corporativo se usará desde el comienzo. En caso contrario, la fase austera puede utilizar
un volumen documental dedicado de la VM de datos, con áreas separadas de cuarentena y
objetos limpios, claves opacas, montaje `noexec,nodev,nosuid`, huella y copia externa. El
navegador usará siempre el mismo contrato de sesión de carga: inicialmente la API transmitirá
el cuerpo a disco sin cargarlo completo en memoria y, al incorporar S3, devolverá una URL
prefirmada sin cambiar el dominio ni la interfaz de usuario.

La plataforma objetivo usará almacenamiento de objetos compatible con S3, con versionado,
cifrado, retención, bloqueo cuando proceda y copias. Ningún documento estará disponible
hasta superar cuarentena, detección de tipo, límites, antivirus y promoción. Si falla el
análisis, permanece en cuarentena.

No se instalará inicialmente RabbitMQ, NATS o Kafka. Una tabla de trabajos y una bandeja de
salida transaccional de PostgreSQL implementarán concesiones temporales, reintentos,
idempotencia, deduplicación, estado terminal y conciliación. El puerto de cola permitirá
sustituirlas más adelante por un broker sin modificar los casos de uso. `LISTEN/NOTIFY`
podrá despertar trabajadores, pero no será la fuente duradera. PostgreSQL contempla
`SKIP LOCKED` para consumidores múltiples sobre tablas tipo cola:
<https://www.postgresql.org/docs/current/sql-select.html>.

## 9. Activar y desactivar módulos sin perder información

Cada unidad desplegable tendrá manifiesto, versión contractual, dependencias, zona
permitida, migraciones, salud y compatibilidad. Su ciclo será:

```text
instalado -> validado -> habilitado -> calentando -> preparado -> en servicio
                                                          |
                                                          v
deshabilitado <- sin rutas <- drenando <- retirada solicitada
```

Al deshabilitar:

1. se impiden nuevas operaciones;
2. se drenan peticiones y trabajos pendientes;
3. se confirma la bandeja de salida y la auditoría;
4. se retiran las rutas de la pasarela;
5. se reducen las réplicas hasta el mínimo permitido;
6. los datos históricos permanecen conservados y consultables según autorización.

Deshabilitar un contenedor nunca borra tablas, documentos, auditoría ni expedientes. Una
migración destructiva no formará parte de este flujo.

La interfaz administrativa podrá solicitar estas operaciones, pero las de impacto alto
requerirán doble control y se ejecutarán mediante el mecanismo declarativo de despliegue.

## 10. Resiliencia y despliegue sin corte

Todo servicio deberá declarar:

- comprobación de arranque;
- comprobación de preparación para recibir tráfico;
- comprobación de vida sin depender innecesariamente de servicios externos;
- cierre ordenado y plazo de drenaje;
- presupuesto de interrupción;
- distribución entre nodos y dominios de fallo;
- estrategia de actualización progresiva, azul/verde o canario;
- procedimiento de reversión compatible con datos y migraciones.

Las sesiones no dependerán de memoria local de una réplica. Las operaciones serán
idempotentes y las tareas usarán colas con reintento, límite, cuarentena de mensajes y
conciliación.

## 11. Seguridad de plataforma

Como mínimo:

- redes con denegación predeterminada en entrada y salida;
- namespaces, cuentas de servicio y credenciales distintas por componente;
- Pod Security `restricted` y políticas de admisión;
- secretos desde gestor corporativo, no desde Git ni variables visibles;
- cifrado de tráfico y de datos en reposo según el análisis de riesgos;
- registro interno de imágenes, firma, SBOM y análisis antes de promoción;
- GitOps o mecanismo declarativo con revisión y trazabilidad;
- logs, métricas y trazas exclusivamente internos y minimizados;
- alertas de escalado anómalo, intentos de salida y cambios de rutas;
- administración desde bastión con MFA, cuentas nominales y elevación temporal;
- copias inmutables, segundo emplazamiento y restauraciones ensayadas.

RKE2 ofrece un perfil CIS que aplica endurecimiento adicional, pero no sustituye la
categorización ENS ni los controles de sistema, red, personal y operación:
<https://docs.rke2.io/security/hardening_guide>.

## 12. Requisitos verificables

Los requisitos DES-001 a DES-014 describen la plataforma objetivo o los servicios que
hayan sido declarados de alta disponibilidad. DES-015 a DES-021 gobiernan la excepción
austera. Cronos activa obligatoriamente las fronteras objetivo y nunca puede ampararse en
la excepción.

- **DES-001:** los despliegues público e interno no comparten clúster, nodos, plano de
  control, pasarela ni credenciales.
- **DES-002:** el artefacto público no contiene rutas, interfaz ni manifiesto de Cronos.
- **DES-003:** NGINX solo dirige tráfico a réplicas preparadas y nunca crea contenedores.
- **DES-004:** HPA escala servicios web por métricas aprobadas y respeta mínimos.
- **DES-005:** KEDA puede activar trabajadores desde cero por eventos sin perder mensajes.
- **DES-006:** una convocatoria puede preescalarse por calendario antes de un cierre.
- **DES-007:** una réplica nueva alcanza el objetivo de preparación medido en prueba de
  carga con nodos calientes e imágenes precargadas.
- **DES-008:** deshabilitar un módulo drena trabajo, retira rutas y conserva datos.
- **DES-009:** ninguna imagen productiva usa etiqueta mutable ni carece de firma y SBOM.
- **DES-010:** las conexiones de salida están prohibidas salvo lista aprobada por zona.
- **DES-011:** existe reversión probada para aplicación, configuración y migraciones.
- **DES-012:** una caída de nodo no interrumpe los servicios declarados de alta
  disponibilidad.
- **DES-013:** base de datos, objetos, cola y copias se prueban como límites de capacidad,
  no solo los procesos web.
- **DES-014:** el autoescalado y los cambios manuales generan evidencia de actor, motivo,
  métricas, valor anterior y posterior.
- **DES-015:** la primera fase puede desplegarse en tres VM sin Kubernetes usando las mismas
  imágenes OCI y contratos que la plataforma objetivo.
- **DES-016:** una variante de dos VM no se autoriza sin riesgos, RTO/RPO, copias externas,
  restauración y logs externos aprobados.
- **DES-017:** una sola VM no tratará datos personales reales.
- **DES-018:** la fase inicial no contiene Cronos, GPS, Nóminas ni módulos internos ajenos a
  Bolsa.
- **DES-019:** los trabajos iniciales son duraderos e idempotentes aunque PostgreSQL actúe
  temporalmente como cola.
- **DES-020:** migrar de volumen documental a S3 no modifica entidades ni casos de uso y
  conserva huellas, estados y trazabilidad.
- **DES-021:** ningún proceso de aplicación accede al socket del motor de contenedores.

## 13. Decisiones que debe cerrar Sistemas

Para la primera implantación:

1. disponibilidad de tres VM o aceptación excepcional de dos;
2. Podman Quadlet o Docker Compose v2 y sistema operativo homologado;
3. servicios corporativos reutilizables y productos mínimos que falten;
4. PostgreSQL único, volumen o S3 corporativo, antivirus y copia externa;
5. RTO/RPO provisional, capacidad medida y riesgos residuales aceptados.

Para la plataforma objetivo:

1. RKE2, OpenShift o plataforma Kubernetes corporativa equivalente;
2. servidores, hipervisor, sedes, dominios de fallo y capacidad caliente;
3. balanceador L4/WAF exterior a cada clúster y edición de NGINX;
4. CNI, CSI, almacenamiento de bloques y objetos;
5. PostgreSQL gestionado fuera del clúster u operador dentro de una zona de datos;
6. cola de mensajes y criterios para introducirla;
7. registro de imágenes, firma, SBOM y escáner aprobados;
8. gestor de secretos, PKI, HSM/KMS e identidad de cargas;
9. observabilidad, SIEM, EDR y protección de ejecución;
10. GitOps, repositorio interno y circuito de promoción sin salida a Internet;
11. RTO, RPO, segundo emplazamiento y capacidad de recuperación;
12. objetivos medibles de arranque, latencia, concurrencia y carga máxima.

Estas decisiones y el orden para implantarlas se desarrollan en el
[Manual de preparación de plataforma para Sistemas](manual_sistemas_preparacion_plataforma.md).
