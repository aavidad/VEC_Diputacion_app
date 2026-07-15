# Manual 00 para Sistemas: preparación e instalación de la plataforma

Estado: **borrador para revisión conjunta de Sistemas, Seguridad, Arquitectura, DPD y
equipo de desarrollo; no autoriza todavía una instalación productiva**.

Versión del documento: 0.1 — 14 de julio de 2026.

## 1. Finalidad y destinatarios

Este manual indica qué infraestructura necesita el portal transversal de RRHH/VEC, en qué
zonas debe instalarse, en qué orden y cómo demostrar que la plataforma queda preparada para
recibir la aplicación.

Está dirigido a:

- responsables de Sistemas y comunicaciones;
- Seguridad y operación/SOC;
- administradores de sistemas, Kubernetes, bases de datos, almacenamiento y copias;
- responsables de identidad, PKI, redes y puesto de trabajo;
- equipo de desarrollo encargado de entregar los artefactos de la aplicación.

Este es el manual previo al despliegue. Se estructura en tres niveles:

1. **Este documento estable:** arquitectura, productos necesarios, orden, controles y
   pruebas.
2. **Anexo de plataforma elegida:** RKE2, OpenShift o plataforma corporativa, junto con CNI,
   CSI, balanceador, almacenamiento y productos auxiliares.
3. **Libro de ejecución por versión:** direcciones reales, versiones, huellas, valores,
   comandos automatizados, copias de configuración, actualización y reversión.

El tercer nivel solo puede cerrarse cuando Sistemas aporte su inventario y se aprueben los
productos. No se incluirán recetas como «instalar la última versión» ni descargas directas
sin verificar. Cada instalación productiva quedará reproducible y fijada por versión y
huella.

### 1.1 Lectura por perfiles de capacidad

Este manual contiene dos perfiles compatibles:

- **primera implantación austera:** Bolsa sobre tres VM, sin Kubernetes, broker ni alta
  disponibilidad, desarrollada en el apartado 15;
- **plataforma objetivo:** despliegues público e interno separados sobre Kubernetes con
  alta disponibilidad, escalado y servicios de datos redundantes.

No hay que instalar ahora todo el inventario objetivo. Sí hay que respetar desde el primer
día sus fronteras, contratos y formatos para crecer sin reescribir la aplicación. La falta
de presupuesto puede justificar menor disponibilidad y más operación manual; no justifica
suprimir controles de confidencialidad, integridad, autenticidad o trazabilidad.

## 2. Condiciones previas obligatorias

Antes de adquirir, reservar o instalar servidores se aprobarán:

1. alcance del sistema y superficies públicas e internas;
2. categorización ENS y análisis de riesgos inicial;
3. inventario de tratamientos y evaluación de impacto cuando proceda;
4. RTO, RPO, disponibilidad y periodos máximos de mantenimiento por servicio;
5. volumetría y previsión de crecimiento;
6. topología inicial y objetivo de DMZ, Mulhacén, datos y gestión, dominio externo de
   copias y necesidad de segundo emplazamiento según el análisis;
7. modelo de operación, guardias, soporte y responsables;
8. productos corporativos que deben reutilizarse;
9. licencias, soporte y política de ciclo de vida;
10. presupuesto de implantación, operación, auditoría y recuperación.

Cronos y la geolocalización no se pilotarán con datos reales antes de aprobar su EIPD,
finalidades, información al personal, conservación, acceso y canal GPS.

## 3. Arquitectura objetivo que Sistemas debe proporcionar

### 3.1 Dominios separados

El portal completo contará, como mínimo, con los siguientes dominios cuando incorpore
Cronos y módulos internos sensibles:

```text
                         INTERNET
                            |
                 cortafuegos / protección DDoS
                            |
                     WAF y balanceador
                            |
                    NGINX Gateway público
                            |
                CLÚSTER PÚBLICO EN LA DMZ
                +-- portal y API públicos
                +-- Bolsa/procesos selectivos
                +-- bot y MCP públicos limitados
                +-- proyección de datos públicos
                            X
                 SIN RUTA HACIA CRONOS
                            X
                     RED MULHACÉN
                            |
                  WAF/balanceador interno
                            |
                   NGINX Gateway interno
                            |
                     CLÚSTER INTERNO
                +-- portal del empleado
                +-- backoffice de RRHH
                +-- Cronos, Personal, Dietas...
                +-- mapas y conectores internos

                    RED DE GESTIÓN SEGREGADA
                +-- bastión, PAM y MFA
                +-- Git y registro de imágenes
                +-- secretos, PKI/HSM y KMS
                +-- observabilidad, SIEM y copias
```

En esa plataforma objetivo no bastan dos namespaces en un mismo clúster. El clúster público
y el interno no compartirán
nodos, plano de control, pasarela, secretos, cuentas administrativas, bases de datos ni
credenciales de sesión.

La primera implantación de Bolsa aplica la topología reducida del apartado 15. No contendrá
Cronos, GPS, Nóminas ni otros módulos internos de alto impacto; esa exclusión es la medida
que permite aplazar el segundo clúster.

El plano de control de ningún clúster se expondrá a Internet. La administración solo se
realizará desde la red de gestión mediante bastión, MFA y cuentas nominales privilegiadas
de vigencia limitada.

### 3.2 Entornos

Como mínimo:

- desarrollo local sin datos reales;
- integración automatizada;
- preproducción representativa de las fronteras de producción;
- producción pública;
- producción interna;
- entorno aislado de restauración y pruebas de desastre.

Los artefactos serán los mismos entre entornos; cambiarán configuración, secretos y
dimensionamiento. No se recompilará un binario distinto para cada entorno.

### 3.3 Emplazamientos y fallo

Sistemas deberá indicar:

- centro principal y centro de contingencia;
- hipervisores o servidores físicos y sus dominios de fallo;
- almacenamiento, cabinas y caminos redundantes;
- ancho de banda y latencia entre centros;
- autonomía eléctrica y comunicaciones;
- mecanismo de recuperación de DNS, PKI, identidad, registro, secretos y copias.

Una alta disponibilidad dentro de un único CPD no constituye recuperación ante desastre.

## 4. Perfil de plataforma objetivo

La aplicación se entregará como imágenes OCI y definiciones declarativas. Kubernetes es el
destino porque aporta recuperación, actualizaciones progresivas, políticas y escalado de
réplicas. No es, sin embargo, un requisito de la primera implantación austera.

### 4.1 Alternativas admitidas

| Plataforma | Cuándo elegirla | Responsabilidad operativa |
| --- | --- | --- |
| RKE2 con soporte | Se busca Kubernetes contenido, neutral y apto para instalación local o desconectada | Sistemas integra registro, GitOps, secretos, datos, copias y observabilidad |
| OpenShift con suscripción | Ya existe experiencia/contrato Red Hat y se prefiere una plataforma más integrada | Se opera conforme al ciclo y operadores soportados de Red Hat |
| Kubernetes corporativo equivalente | Ya está implantado y satisface este manual | Debe demostrar separación, soporte, endurecimiento, copia y auditoría |

RKE2 recomienda tres nodos de servidor para un clúster de alta disponibilidad y una
dirección estable delante de ellos: <https://docs.rke2.io/install/ha>. Su perfil CIS ayuda a
endurecer la instalación, pero requiere también preparación del sistema anfitrión:
<https://docs.rke2.io/security/hardening_guide>.

OpenShift será preferible si el soporte y la integración corporativa compensan su mayor
huella y coste. La decisión se documentará en una ADR con coste total, competencias del
equipo, soporte, salida del proveedor y prueba en red restringida.

### 4.2 Base provisional de nodos

Hasta disponer de dimensionamiento, la referencia de alta disponibilidad por clúster
productivo será:

- tres nodos de plano de control, en hipervisores o servidores distintos;
- tres o más nodos de trabajo;
- separación opcional de nodos de sistema, aplicaciones y trabajos intensivos;
- dirección virtual estable delante del API del clúster;
- dirección virtual redundante para la pasarela de aplicaciones;
- al menos un nodo de margen para mantenimiento y ráfagas.

Esta referencia no es una oferta de capacidad. CPU, RAM, IOPS, disco y número de nodos se
fijarán con volumetría y pruebas.

## 5. Inventario de componentes que debe instalar o proporcionar Sistemas

Este inventario describe el destino completo. La selección mínima que se instala en la
primera fase figura en el apartado 15.2; siempre se reutilizará un servicio corporativo
existente que cumpla los requisitos antes de crear otro para el proyecto.

### 5.1 Servicios corporativos previos

| Componente | Función | Requisito principal |
| --- | --- | --- |
| Virtualización o servidores | Alojar los nodos y servicios de datos | Dominios de fallo y capacidad reservada |
| VLAN/VRF y cortafuegos | Separar DMZ, Mulhacén, datos y gestión | Denegación predeterminada en entrada y salida |
| Protección DDoS, WAF y balanceadores | Proteger y distribuir accesos | Pares redundantes por zona |
| DNS e IPAM | Nombres y direcciones | Zonas interna/pública separadas y DNS inverso |
| NTP corporativo | Hora confiable | Redundante y supervisado |
| PKI, OCSP/CRL y HSM/KMS | Certificados, sellos y claves | Custodia y renovación automatizable |
| Active Directory/IdP y MFA | Identidad de personal y operadores | OIDC/SAML/Kerberos según conector aprobado |
| Bastión y PAM | Administración privilegiada | Sesiones nominales, grabadas y temporales |
| EDR y gestión de parches | Proteger anfitriones | Compatible con Kubernetes y sin fuga de datos |
| SIEM/SOC | Correlación y respuesta | Recepción interna, retención y alertas |
| Copia corporativa | Copias inmutables y recuperación | Segundo dominio de fallo y pruebas de restauración |

### 5.2 Plataforma de contenedores

| Componente | Propuesta | Observaciones |
| --- | --- | --- |
| Kubernetes | RKE2, OpenShift o estándar corporativo | Un clúster público y otro interno |
| Motor OCI | El incluido/soportado por la distribución | No es obligatorio Docker Engine |
| CNI | Producto compatible con `NetworkPolicy` | Debe controlar entrada y salida; selección previa |
| CSI | Controlador soportado por la cabina/Ceph | Cifrado, snapshots y expansión probados |
| Gateway API | Canal estable compatible | Contrato moderno de rutas |
| NGINX Gateway Fabric | Proxy inverso NGINX | Una instalación independiente por clúster |
| `metrics-server` | Métricas de recursos | Necesario para HPA básico |
| Adaptador de métricas | Prometheus u otro aprobado | Peticiones, latencia, colas y concurrencia |
| KEDA | Escalado por eventos/calendario | Solo donde exista caso de uso |
| Pod Security Admission | Perfil `restricted` | Excepciones documentadas y temporales |
| Motor de políticas | Kyverno o Gatekeeper | Verificación de reglas y artefactos |
| GitOps | Argo CD o Flux | Ámbito y credenciales independientes por clúster |

No se instalará el proyecto comunitario `ingress-nginx`: quedó retirado en marzo de 2026 y
no recibe correcciones. Kubernetes recomienda Gateway API:
<https://kubernetes.io/blog/2025/11/11/ingress-nginx-retirement/>.

NGINX Gateway Fabric implementa Gateway API y separa su plano de control de los planos de
datos NGINX: <https://docs.nginx.com/nginx-gateway-fabric/>. Su telemetría saliente se
deshabilitará salvo autorización expresa.

### 5.3 Cadena de suministro

| Componente | Propuesta | Requisito |
| --- | --- | --- |
| Git interno | Servicio corporativo | Ramas protegidas, MFA y auditoría |
| Ejecutores de integración | Trabajadores aislados | Sin secretos permanentes ni acceso innecesario a producción |
| Registro OCI | Harbor, Quay o registro corporativo | HTTPS, HA, réplica, cuotas y retención |
| Escáner | Trivy u homologado | Imágenes, dependencias, configuración y SBOM |
| Firma | Cosign o servicio corporativo | Clave en HSM/KMS/Vault y verificación en admisión |
| SBOM | SPDX o CycloneDX | Una por artefacto y versión |
| Espejo de paquetes | Repositorio corporativo | Go, SO, charts e imágenes aprobados |

El registro y el espejo deben existir antes de instalar clústeres sin Internet. No deben
depender únicamente del mismo clúster al que abastecen. Harbor documenta tanto instalación
local como despliegue HA: <https://goharbor.io/docs/main/install-config/>.

### 5.4 Secretos e identidad de cargas

Se reutilizará el gestor corporativo o se proporcionará uno de alta disponibilidad, por
ejemplo Vault, integrado con PKI/HSM. Kubernetes Secret no será el repositorio maestro.

Debe permitir:

- credenciales de base de datos rotables;
- certificados de servicio y mTLS;
- separación por clúster, entorno, módulo y cuenta de servicio;
- entrega sin incluir secretos en Git, imagen o línea de comandos;
- revocación y rotación de emergencia;
- auditoría de lecturas y cambios;
- recuperación documentada del propio gestor.

### 5.5 Datos

| Componente | Función | Implantación pendiente de decidir |
| --- | --- | --- |
| PostgreSQL HA | Persistencia inicial | Servicio corporativo/VM dedicadas u operador soportado |
| Gestor de conexiones | Proteger PostgreSQL | PgBouncer o equivalente |
| Objeto compatible S3 | Cuarentena, documentos y evidencias | Ceph RGW, producto S3 local u opción corporativa soportada |
| Broker de mensajes | Trabajos, reintentos y eventos | RabbitMQ, NATS, Kafka u otro tras medir necesidades |
| Almacén de auditoría | Evidencia resistente a alteración | Separado de logs técnicos, copia WORM/inmutable |
| Futuro Oracle | Adaptador alternativo | No se instala hasta aprobar conector y migración |

PostgreSQL se desplegará con réplica/failover según RTO, copia continua de WAL,
recuperación a un punto en el tiempo, cifrado, supervisión y restauración periódica. Se
elegirá una versión soportada y su última revisión menor; el calendario oficial está en
<https://www.postgresql.org/support/versioning/>.

La base de datos no se expondrá a Internet ni a los nodos públicos de forma general. Cada
servicio utilizará cuenta, esquema y permisos mínimos; Cronos tendrá almacén y credenciales
propios.

El almacenamiento S3 será un puerto de la aplicación, no una dependencia de un fabricante.
La carga directa mediante URL temporal terminará primero en cuarentena. Versionado,
retención, cifrado, replicación, borrado y bloqueo se configurarán según la política
documental.

El broker solo se instalará cuando estén definidos durabilidad, orden, reintentos,
idempotencia, volumen y recuperación. Introducir una cola sin operación y conciliación no
mejora la fiabilidad.

### 5.6 Observabilidad y seguridad operacional

Propuesta mínima:

- OpenTelemetry Collector para recepción y exportación interna;
- Prometheus para métricas;
- Alertmanager para alertas;
- Grafana para cuadros de mando;
- Loki, OpenSearch o servicio corporativo para logs;
- envío de eventos de seguridad al SIEM;
- protección de ejecución/nodo aprobada por Seguridad;
- monitorización externa de disponibilidad que no dependa del clúster observado.

Métricas, trazas y logs no incluirán documentos, DNI completos, coordenadas GPS, motivos
médicos ni contenido de solicitudes. Ningún agente enviará telemetría a Internet sin
evaluación y autorización.

### 5.7 Copias y recuperación

Se protegerán de forma independiente:

- configuración e inventario de los clústeres;
- etcd y certificados del plano de control;
- PostgreSQL y WAL;
- objetos y sus metadatos;
- broker y configuraciones necesarias para conciliar;
- Git, registro y artefactos firmados;
- secretos y procedimiento de recuperación de claves;
- auditoría y SIEM;
- servicios cartográficos y versiones de datos;
- documentación de red, DNS, PKI y automatización.

Las copias estarán cifradas, serán inmutables cuando proceda, usarán claves separadas y se
replicarán fuera del dominio de fallo principal. Una copia que no se ha restaurado no se
considera validada.

### 5.8 Servicios exclusivos de Cronos

Sistemas deberá proporcionar dentro de Mulhacén:

- pasarela y API internas sin salida general;
- PostgreSQL/objeto y secretos exclusivos;
- teselas, biblioteca cartográfica, geocodificador y OSRM internos;
- ingestión GPS desde APN privada o mecanismo aprobado;
- SIEM y copias internas;
- puestos gestionados y DLP para perfiles sensibles.

El `deploy/osrm-granada` actual solo es una base de laboratorio. Antes de producción se
fijará la imagen por huella —no `latest`—, se añadirá alta disponibilidad, actualización
controlada, teselas/geocodificación, red interna y pruebas de ausencia de salida.

## 6. Información que Sistemas debe entregar al proyecto

### 6.1 Inventario técnico

- sistemas operativos Linux homologados y soporte;
- virtualización, hardware, CPU, RAM y almacenamiento disponibles;
- cabinas, CSI, clases de almacenamiento, snapshots e IOPS;
- productos de LB, WAF, cortafuegos, DDoS y proxy de salida;
- DNS, NTP, PKI, HSM/KMS, OCSP y certificados;
- AD, IdP, MFA, PAM, bastiones y VPN/ZTNA corporativa;
- Git, CI, registro, escáner y repositorios actuales;
- SIEM, SOC, logs, EDR y herramientas de vulnerabilidades;
- PostgreSQL, Oracle, copias, segundo CPD y procedimientos existentes;
- correo, Telegram, firma, registro, archivo, antivirus y restantes conectores;
- capacidad y soporte del equipo que operará la plataforma.

### 6.2 Volumetría

- personas externas, empleados, gestores y administradores totales;
- concurrencia normal y de último día de plazo;
- peticiones por segundo, latencia y disponibilidad objetivo;
- convocatorias simultáneas y solicitudes por convocatoria;
- número, tamaño máximo, tamaño medio y tipos de documentos;
- crecimiento anual y conservación por serie;
- firmas, verificaciones, notificaciones, PDF y audios por hora;
- trabajos antivirus simultáneos y tiempos admitidos;
- vehículos, frecuencia GPS y consultas cartográficas;
- ventanas de mantenimiento y lotes nocturnos;
- RTO y RPO diferenciados por portal, Bolsa, Cronos y datos.

Sin esta información no se fijarán CPU, RAM, IOPS, almacenamiento, réplicas máximas, cuotas,
retención ni objetivos de arranque.

## 7. Matriz conceptual de red

La matriz definitiva la elaborará Sistemas tras elegir CNI, CSI y productos. La tabla
incluye los flujos de la plataforma objetivo; en la fase inicial solo se habilitarán los
correspondientes a NGINX, aplicación, PostgreSQL y servicios corporativos aprobados. Como
base:

| Origen → destino | Puerto habitual | Regla |
| --- | ---: | --- |
| Internet → WAF/balanceador público | TCP 443 | Única entrada pública; 80 solo para redirección aprobada |
| WAF/balanceador → NGINX público | TCP 443 | Solo IP de los balanceadores |
| Mulhacén → NGINX interno | TCP 443 | Solo redes y VPN/ZTNA autorizadas |
| Red administrativa → bastión | TCP 443/22 | MFA/PAM y equipos gestionados |
| Bastión → API Kubernetes | TCP 6443 | Nunca desde Internet ni redes de usuario |
| Nodos RKE2 → servidores | TCP 6443 y 9345 | Solo redes de nodos |
| Servidor → servidor etcd | TCP 2379/2380 | Exclusivo del plano de control |
| Control autorizado → kubelet | TCP 10250 | Solo componentes y bastión necesarios |
| Nodos → CNI | Según CNI elegido | Overlay/BGP nunca expuesto al exterior |
| Nodos → registro/Git/secretos | TCP 443 o puerto aprobado | Solo destinos internos enumerados |
| Aplicación → PostgreSQL | TCP 5432 | Por identidad y segmento; sin acceso general |
| Aplicación → S3 interno | TCP 443 | Bucket y credenciales por finalidad |
| Componentes → DNS/NTP | 53 TCP/UDP y 123 UDP | Solo servidores corporativos |
| Agentes → observabilidad | 4317/4318 o 6514 | OTLP o syslog con TLS |
| APN GPS → pasarela telemática | Según diseño | mTLS, lista de dispositivos y antirrepetición |

El rango NodePort no se abrirá completo. Las excepciones se limitarán al balanceador y
puertos concretos. IPv4, IPv6, DNS, HTTP/HTTPS, QUIC/UDP e ICMP de salida se controlarán;
bloquear solo TCP/443 no constituye «sin salida».

## 8. Orden de preparación e instalación

Las fases de este apartado describen la construcción de la plataforma Kubernetes objetivo;
no deben confundirse con la primera implantación austera. El orden mínimo de esta última
está en el apartado 15.7.

### Fase 0 — aprobar diseño y versiones

1. completar los datos de los apartados anteriores;
2. elegir plataforma, CNI, CSI, WAF/LB, registro, secretos, datos y copias;
3. aprobar licencias y soporte;
4. crear una matriz de compatibilidad con versión exacta y fin de soporte;
5. obtener imágenes, paquetes, charts y firmas desde fuentes oficiales;
6. analizar y reflejar todo en repositorios internos;
7. aprobar el plan de reversión y la ventana de instalación.

Salida: ADR firmada, lista de materiales de plataforma y libro de ejecución 0.1.

### Fase 1 — redes y servicios básicos

1. crear VLAN/VRF y CIDR de nodos, pods y servicios sin solapes;
2. reservar VIP, DNS directo/inverso e IPAM;
3. configurar cortafuegos con denegación predeterminada;
4. preparar balanceadores/WAF público e interno;
5. validar NTP, PKI, OCSP/CRL, bastión, PAM y MFA;
6. preparar rutas al segundo emplazamiento y a copias;
7. probar explícitamente la ausencia de ruta DMZ → Cronos.

Salida: matriz de flujos aprobada y pruebas de red adjuntas.

### Fase 2 — plano de gestión y cadena de suministro

1. instalar o reservar Git, registro OCI y espejo de paquetes;
2. habilitar HTTPS, identidad corporativa, MFA y roles;
3. instalar escáner, SBOM y firma;
4. instalar o integrar gestor de secretos/KMS/HSM;
5. preparar repositorio de copias y monitorización externa;
6. cargar por digest todos los artefactos necesarios para instalar sin Internet;
7. ensayar recuperación del registro y de secretos.

Salida: clústeres capaces de instalarse sin contactar repositorios públicos.

### Fase 3 — anfitriones

1. actualizar firmware y verificar cadena de arranque;
2. instalar SO soportado desde plantilla controlada;
3. aplicar guía CCN-STIC/CIS y configuración de la distribución Kubernetes;
4. cifrar discos cuando proceda y configurar claves;
5. habilitar SELinux/AppArmor, `auditd`, EDR y cortafuegos de host;
6. configurar NTP, DNS, certificados y repositorios internos;
7. reservar recursos del sistema y almacenamiento rápido para etcd;
8. eliminar cuentas compartidas y acceso directo no administrado;
9. registrar nodos en inventario/CMDB y copia.

Salida: informe de endurecimiento sin hallazgos críticos.

### Fase 4 — laboratorio y preproducción

1. instalar primero una topología representativa no productiva;
2. comprobar instalación desconectada;
3. probar alta y baja de nodos, actualización y reversión;
4. comprobar políticas de red, Pod Security y auditoría;
5. simular pérdida de nodo, certificado, almacenamiento y registro;
6. corregir y versionar la automatización.

No se pasará a producción si el procedimiento solo funciona manualmente en un nodo.

### Fase 5 — clústeres productivos

Para cada clúster, sin reutilizar credenciales entre ellos:

1. colocar la dirección estable delante del plano de control;
2. instalar tres servidores y comprobar quorum/etcd;
3. unir nodos de trabajo;
4. instalar/configurar CNI y políticas de red;
5. instalar CSI y probar volumen, snapshot, expansión y recuperación;
6. habilitar cifrado de secretos, auditoría del API y RBAC con IdP;
7. aplicar Pod Security `restricted` y cuotas base;
8. hacer copia inicial de configuración y etcd;
9. simular la pérdida de un servidor y un trabajador.

Salida: clústeres separados, saludables y recuperables.

### Fase 6 — pasarela, políticas y escalado

1. instalar recursos estables de Gateway API compatibles;
2. instalar NGINX Gateway Fabric desde el registro interno;
3. deshabilitar telemetría saliente no autorizada;
4. conectar certificados y WAF/LB de cada zona;
5. instalar `metrics-server` y adaptador de métricas;
6. instalar KEDA donde se haya aprobado;
7. instalar Kyverno o Gatekeeper y políticas en modo auditoría;
8. corregir incumplimientos y pasar las políticas críticas a bloqueo;
9. probar preparación, drenaje y distribución entre nodos.

Salida: una aplicación de prueba recibe tráfico solo dentro de su zona y escala sin saltarse
las políticas.

### Fase 7 — datos y servicios comunes

1. desplegar PostgreSQL HA y gestor de conexiones;
2. configurar cifrado, cuentas mínimas, WAL, PITR y alertas;
3. desplegar objeto S3 con versionado, retención y cuarentena;
4. desplegar broker solo si sus contratos están aprobados;
5. integrar secretos dinámicos y certificados de cargas;
6. instalar servicios cartográficos internos y su canal de actualización;
7. integrar antivirus y restantes conectores por sus puertos;
8. restaurar una copia completa en entorno aislado.

Salida: servicios de estado medidos, respaldados y sin acceso general entre zonas.

### Fase 8 — observabilidad, GitOps y operación

1. instalar recolectores, métricas, alertas, logs y trazas;
2. enviar seguridad al SIEM y probar alertas;
3. instalar GitOps con ámbito separado por clúster;
4. proteger repositorios y requerir revisión/doble control;
5. crear cuadros de capacidad, certificados, copias, colas y seguridad;
6. configurar guardias, escalado, incidentes y mantenimiento;
7. probar que la observabilidad no transmite fuera ni contiene datos prohibidos.

Salida: cualquier cambio productivo deja commit, aprobación, artefacto, despliegue y
resultado correlacionados.

### Fase 9 — recepción de la aplicación

El equipo de desarrollo entregará:

- imágenes por digest, firmadas y con SBOM;
- manifiestos/plantillas declarativos;
- requisitos de CPU/memoria y puertos;
- comprobaciones de arranque, preparación y vida;
- migraciones y compatibilidad hacia atrás;
- matriz de secretos y configuración;
- HPA/KEDA con mínimos, máximos y métricas;
- políticas de red esperadas;
- manual de actualización, drenaje y reversión;
- pruebas de contrato, carga, seguridad y restauración;
- inventario de licencias y vulnerabilidades aceptadas.

Sistemas no reconstruirá imágenes ni modificará código en producción.

## 9. Configuración de escalado

Este apartado se aplicará cuando se active la plataforma objetivo. En la primera
implantación se utilizarán recuperación de procesos, alertas, capacidad medida y
preescalado manual conforme al apartado 15.5.

El diseño detallado, incluidos los dos bucles de escalado y el ejemplo de Bolsa, se mantiene
en [Arquitectura de despliegue, contenedores y escalado](despliegue_contenedores_y_escalado.md#65-los-dos-bucles-automáticos).

Sistemas debe indicar antes de habilitar el segundo bucle:

- producto de virtualización o nube privada y su API;
- proveedor/controlador de máquinas compatible con la distribución Kubernetes;
- plantilla endurecida, firmada y actualizada de nodo;
- red, IPAM, DNS, certificados y unión automática al clúster;
- cuotas y límite económico/físico;
- tiempo medido para crear, preparar, vaciar y destruir un nodo;
- comportamiento cuando el proveedor no responde;
- nodos que nunca se apagarán y reserva mínima caliente;
- permisos de la cuenta de máquina y auditoría de cada alta/baja.

El autoaprovisionador no recibirá permisos generales sobre el hipervisor. Solo podrá operar
en el conjunto, plantilla, red y cuota destinados a cada clúster.

La plataforma Kubernetes distingue el escalado de cargas del escalado de nodos. Cluster
Autoscaler trabaja con grupos de nodos preconfigurados; Karpenter necesita una integración
específica del proveedor. La elección se hará solo después de comprobar compatibilidad con
el hipervisor corporativo:
<https://kubernetes.io/docs/concepts/cluster-administration/node-autoscaling/>.

### Servicios que nunca se reducen a cero

- NGINX Gateway;
- portal y API públicos durante su horario de servicio;
- identidad y autorización;
- portal interno y Cronos;
- componentes de auditoría críticos;
- servicios cuya activación tardía incumpla el SLO.

Mantendrán al menos dos réplicas si han sido declarados de alta disponibilidad.

### Servicios aptos para KEDA y escala a cero

- antivirus y desarme;
- generación de PDF/audio;
- avisos y notificaciones;
- importaciones/exportaciones;
- trabajos masivos de baremación;
- otros consumidores idempotentes de cola.

HPA puede ajustar réplicas aproximadamente cada 15 segundos según métricas:
<https://kubernetes.io/docs/concepts/workloads/autoscaling/horizontal-pod-autoscale/>.
KEDA puede activar trabajadores desde cero por eventos:
<https://keda.sh/docs/2.20/concepts/scaling-deployments/>.

Para conseguir respuesta en segundos se mantendrá capacidad caliente y se precargarán
imágenes. El escalado de nodos o máquinas virtuales no se considerará instantáneo. Antes de
fechas conocidas se elevará el mínimo por calendario y se realizará una prueba de carga.

## 10. Reglas de instalación segura

- Automatizar mediante infraestructura como código y gestión de configuración.
- No ejecutar `curl | sh` en producción ni confiar en etiquetas mutables.
- Verificar firma, procedencia, SBOM y huella antes de entrar al registro interno.
- Desplegar siempre por digest.
- No incluir contraseñas, tokens o claves en comandos, historial, Git o tickets.
- No conceder `cluster-admin` a procesos de entrega ni cuentas cotidianas.
- No conectar contenedores a la vez a redes públicas e internas.
- No montar socket de contenedores, `/`, `/proc` o directorios del host.
- No usar imágenes privilegiadas salvo excepción técnica aprobada y compensada.
- No habilitar respaldo automático a servicios de Internet.
- No abrir todos los NodePort ni todos los puertos entre nodos sin justificación.
- No instalar paneles Kubernetes anónimos ni publicar APIs administrativas.
- No introducir datos reales hasta superar la puerta de seguridad y privacidad.

## 11. Pruebas de aceptación de plataforma

### 11.1 Separación y seguridad

- Desde Internet solo responde el HTTPS público autorizado.
- Desde Internet y desde la DMZ no se alcanza Cronos, su DNS, API, base, mapas, GPS o
  copias.
- API Kubernetes, etcd, registro, secretos y bases no están expuestos públicamente.
- Desde cada carga se prueban salidas IPv4, IPv6, DNS, DoH, HTTP/HTTPS, QUIC e ICMP; solo
  pasan las excepciones aprobadas.
- La aplicación pública no posee secreto, ruta ni audiencia válida para el clúster interno.
- Se rechazan imágenes sin firma, por etiqueta mutable, privilegiadas o fuera del registro.
- Las cuentas administrativas requieren MFA/PAM y sus acciones llegan al SIEM.
- Ninguna telemetría sale a servicios de fabricante sin autorización.

### 11.2 Alta disponibilidad y capacidad

Las siguientes pruebas corresponden a los servicios declarados de alta disponibilidad y a
la plataforma objetivo:

- Se pierde de forma controlada un nodo de trabajo sin corte no admitido.
- Se pierde un nodo de control manteniendo quorum y operación.
- NGINX retira inmediatamente una réplica no preparada.
- HPA y KEDA responden a carga y cola dentro del objetivo medido.
- Se ejecuta el pico de último día con margen acordado.
- Se prueba la saturación de PostgreSQL, S3, broker, antivirus y firma, no solo HTTP.
- Se verifica actualización progresiva y reversión.

En la primera implantación austera se sustituirán por:

- reinicio automático de cada proceso y reconstrucción completa de cada VM desde su
  definición;
- prueba de saturación y margen antes de abrir una convocatoria;
- arranque manual o programado de capacidad adicional dentro de la VM, cuando exista;
- medición del corte durante despliegue, fallo y restauración;
- restauración completa dentro del RTO/RPO provisionalmente aceptado.

No se afirmará que existe alta disponibilidad por ejecutar dos contenedores en el mismo
anfitrión.

### 11.3 Datos y recuperación

En la plataforma objetivo:

- Failover de PostgreSQL sin pérdida superior al RPO.
- Recuperación PITR en instancia aislada.
- Restauración de objetos y metadatos.
- Recuperación de etcd/configuración y secretos.
- Copias inmutables inaccesibles desde una cuenta comprometida del clúster.
- Recuperación completa desde el segundo emplazamiento dentro del RTO.
- Conciliación posterior demuestra que no hay expedientes o mensajes ambiguos.

En la primera implantación se marcarán como riesgos diferidos el failover, etcd y el
segundo emplazamiento activo. Seguirán siendo obligatorias la recuperación PITR aislada,
la restauración de documentos y metadatos, la reconstrucción de configuración y secretos,
la copia externa a las credenciales de origen y la conciliación completa dentro del
RTO/RPO provisional aprobado.

### 11.4 Operación y entrega

- CMDB con propietarios, versiones y fin de soporte.
- cuadros, alertas, guardias y escalado definidos;
- renovación de certificados ensayada;
- runbooks de incidente, brecha, ransomware y desastre;
- matriz final de IP, DNS, puertos, reglas, cuentas y dependencias;
- documentación `as built` firmada por responsables;
- formación y simulacro de operación antes de la apertura pública.

## 12. Puerta de autorización para desplegar la aplicación

Sistemas declarará la plataforma preparada solo cuando existan evidencias de:

- decisiones y riesgos aprobados;
- topología y flujos construidos conforme al diseño;
- versiones y soportes vigentes;
- endurecimiento y análisis sin críticos;
- cadena de suministro firmada;
- secretos y certificados operativos;
- copias y restauraciones superadas;
- separación pública/interna demostrada;
- carga y escalado medidos;
- monitorización, SIEM y guardias activos;
- manuales de instalación, actualización, reversión e incidente;
- aceptación conjunta de Sistemas, Seguridad y propietarios del servicio.

## 13. Anexos que habrá que completar

| Anexo | Responsable principal | Momento |
| --- | --- | --- |
| A. Inventario y volumetría | Sistemas + RRHH | Antes de dimensionar |
| B. Topología, CIDR, DNS y flujos | Redes + Seguridad | Antes de instalar |
| C. ADR RKE2/OpenShift | Arquitectura + Sistemas | Antes de comprar/licenciar |
| D. Matriz de versiones y compatibilidad | Sistemas + Desarrollo | En cada línea base |
| E. Libro automatizado de instalación | Sistemas | Tras elegir productos |
| F. Configuración NGINX/Gateway/WAF | Redes + Sistemas | Antes de publicar |
| G. PostgreSQL/S3/broker | DBA + Almacenamiento | Antes de datos reales |
| H. Copia y recuperación | Continuidad + Sistemas | Antes de producción |
| I. Observabilidad y SIEM | Operación + Seguridad | Antes de producción |
| J. Cronos, APN y cero salida | Redes + DPD + Seguridad | Antes del piloto Cronos |
| K. Actualización y reversión | Sistemas + Desarrollo | Antes de cada entrega |
| L. Informe de aceptación `as built` | Todos los responsables | Antes de apertura |

## 14. Relación con la fase de prototipo

El documento histórico `docs/portal_vec/desarrollo_vec_orquesta.md` prohibía introducir
Kubernetes y otros servicios en la primera vertical mínima para no sobredimensionar el
prototipo. Esa restricción sigue siendo razonable para desarrollar y probar la lógica local.

Este manual describe el **destino de producción profesional**, no obliga a levantar toda la
plataforma durante la primera iteración de código. La instalación se iniciará por fases
cuando la arquitectura, el alcance y la capacidad hayan sido aprobados.

## 15. Perfil operativo de la primera implantación austera

### 15.1 Alcance y topología

La primera implantación con datos reales incluirá únicamente el núcleo transversal
necesario y el módulo Bolsa. La base recomendada son tres máquinas virtuales:

```text
                         INTERNET
                            |
             cortafuegos/WAF corporativo, si existe
                            |
                 VM 1 - EXPOSICIÓN / DMZ
                 NGINX, TLS, límites y proxy
                 sin documentos ni base de datos
                            |
                  único flujo autorizado
                            |
                 VM 2 - SERVICIOS PRIVADOS
              API/portal público + backoffice RRHH
                trabajadores + conector antivirus
                            |
                  cuentas y puertos mínimos
                            |
                    VM 3 - DATOS
            PostgreSQL + cuarentena + objetos limpios
              volúmenes separados y sin Internet

  gestión -> bastión/VPN y MFA     logs -> SIEM/syslog corporativo
  copias  -> destino externo a VM, hipervisor y credenciales de origen
```

Las interfaces pública e interna de la VM 2 serán procesos o puntos de escucha distintos,
con cuentas de sistema, credenciales de base de datos, rutas y reglas de cortafuegos
distintas.
Compartirán temporalmente anfitrión porque solo existe Bolsa; se documentará este dominio
de fallo. La aplicación pública no tendrá permisos de administración de convocatorias,
baremación, usuarios, auditoría ni exportaciones.

Las tres VM pueden comenzar en el mismo CPD e incluso, si no existe alternativa, en el mismo
hipervisor. Esto aporta segregación lógica, no continuidad ante la pérdida del hipervisor.
La copia externa y el procedimiento de reconstrucción son por ello obligatorios.

Como orden de magnitud para preparar el laboratorio y ejecutar las primeras pruebas, no
como dimensionamiento productivo cerrado:

| VM | Punto de partida para medir | Factor que más puede alterarlo |
| --- | --- | --- |
| VM 1, NGINX | 1–2 vCPU y 2 GiB RAM | TLS, WAF, conexiones y tamaño de carga |
| VM 2, servicios | 4 vCPU y 8 GiB RAM; 12 GiB si aloja antivirus local | concurrencia, firma, PDF y firmas del antivirus |
| VM 3, datos | 4 vCPU, 8–16 GiB RAM y SSD | expedientes, IOPS, WAL y volumen documental |

El antivirus local puede ser el mayor consumidor de memoria; se empezará con un trabajador
y concurrencia de análisis igual a uno. Si existe un servicio corporativo de análisis se
preferirá su conector y se volverá a medir la VM 2. ClamAV advierte de la memoria necesaria
y de los picos de recarga de firmas en su documentación de contenedores:
<https://docs.clamav.net/manual/Installing/Docker.html>.

### 15.2 Qué hay que instalar y qué no

| Capacidad | Primera implantación | Evolución preparada |
| --- | --- | --- |
| Proxy | Un NGINX endurecido en VM 1 | Dos o más pasarelas detrás de LB/WAF |
| Aplicación | Uno o varios procesos OCI en VM 2 | `Deployment` Kubernetes con las mismas imágenes |
| Supervisión de procesos | `systemd` + Podman Quadlet o Docker Compose v2 | Controladores Kubernetes, HPA y KEDA |
| Persistencia | Un PostgreSQL en VM 3 | PostgreSQL HA o servicio corporativo |
| Documentos | S3 corporativo existente o volumen dedicado en VM 3 | S3 redundante, versionado y bloqueo |
| Trabajos | Tabla duradera y bandeja de salida PostgreSQL | Broker detrás del mismo puerto de cola |
| Antivirus | Servicio corporativo/ICAP; si no existe, un trabajador local | Pool o servicio corporativo escalable |
| Logs | `journald`/syslog estructurado enviado al servicio corporativo | Plataforma completa de observabilidad |
| Métricas | salud, capacidad, cola y alertas esenciales | Prometheus, paneles y métricas personalizadas |
| Secretos | gestor corporativo o ficheros cifrados y montados en solo lectura | gestor central/PKI con rotación automática |
| Copias | repositorio corporativo externo y PITR | réplica, segundo emplazamiento y recuperación automatizada |
| Registro OCI | registro corporativo o repositorio sencillo interno | registro HA con políticas y promoción |

No se instalarán en esta fase RKE2/OpenShift, etcd, HPA, KEDA, un broker, Ceph, una pila
completa Prometheus/Grafana/Loki, Vault, Harbor o GitOps salvo que la Diputación ya los
opere como servicio corporativo. Evitar una instalación nueva no significa prescindir de
su función: se aplicará la alternativa mínima de la tabla.

La decisión entre Podman Quadlet y Docker Compose v2 será operativa:

- si Sistemas estandariza Podman, Quadlet integrará los contenedores con `systemd`;
- si ya opera Docker, Compose v2 usará un fichero adicional de producción, políticas de
  reinicio, salud, límites y redes separadas;
- en ambos casos las imágenes se fijarán por versión y digest y la aplicación no accederá
  al socket del motor.

Docker documenta Compose sobre un único servidor para producción:
<https://docs.docker.com/compose/how-tos/production/>. Quadlet proporciona unidades
declarativas para `systemd`: <https://docs.podman.io/en/stable/markdown/podman-quadlet.1.html>.

### 15.3 Datos, documentos y trabajos con pocos componentes

PostgreSQL será inicialmente un primario único. Desde el primer día tendrá:

- roles distintos para migración, aplicación pública, backoffice, trabajador, auditoría y
  copia; la aplicación no será propietaria de la base;
- consistencia y durabilidad activas, sumas de verificación cuando lo soporte la línea base
  y alertas de disco, WAL, bloqueos, conexiones y consultas lentas;
- copia base, archivado continuo de WAL, recuperación a un punto en el tiempo y restauración
  ensayada fuera de la VM;
- migraciones versionadas y ejecutadas como una tarea separada;
- límites del pool de conexiones medidos; PgBouncer solo se añadirá si hace falta.

Los documentos no se guardarán como binarios dentro de PostgreSQL. El dominio conservará
una clave opaca, adaptador de almacenamiento, versión, tamaño, huella, tipos declarado y
detectado, estado, resultado antivirus y conservación; nunca una ruta absoluta del
servidor.

El contrato de carga será estable:

```text
creada -> recibiendo -> cuarentena -> analizando
                                      +-- infectado/rechazado
                                      +-- error/reintento
                                      +-- limpio -> promoviendo -> disponible
```

Solo `disponible` puede descargarse, firmarse o incorporarse al expediente. En el adaptador
inicial la API transmitirá el cuerpo a un fichero temporal sin cargarlo entero en memoria,
calculará su huella y hará promoción atómica cuando sea posible. Al adoptar S3, la creación
de sesión devolverá una URL prefirmada; no cambiarán la pantalla, la entidad ni el caso de
uso. La cuarentena y los objetos limpios usarán áreas y permisos distintos, montaje
`noexec,nodev,nosuid`, nombres opacos y conciliación de huérfanos.

No se necesita un broker para la primera Bolsa. La misma transacción que confirme una
operación insertará su auditoría, el trabajo requerido y el evento de bandeja de salida.
Los trabajadores reclamarán trabajos en lotes breves con concesión temporal y
`FOR UPDATE SKIP LOCKED`, ejecutarán fuera de la transacción y confirmarán de forma
idempotente. Habrá reintento con espera, deduplicación, máximo de intentos, estado terminal,
antigüedad de cola y conciliador. PostgreSQL documenta el uso de `SKIP LOCKED` para evitar
contención entre consumidores de tablas tipo cola:
<https://www.postgresql.org/docs/current/sql-select.html>.

Cuando el volumen o el reparto de eventos lo justifique, un adaptador publicará la misma
bandeja de salida en RabbitMQ, NATS, Kafka u otro producto aprobado. No se reescribirá el
dominio.

### 15.4 Controles que no se aplazan

Antes de tratar datos reales deberán estar implantados y probados, como mínimo:

- categorización y análisis de riesgos iniciales, inventario de activos y tratamientos,
  responsables, conservación, participación del DPD y valoración de EIPD;
- identidad nominal, MFA para personal privilegiado y administración, autorización de
  servidor por rol, ámbito y objeto y denegación predeterminada;
- TLS, cortafuegos por lista permitida, base privada y administración solo por bastión o
  VPN corporativa;
- sistemas soportados, endurecimiento, parcheo, análisis de vulnerabilidades, protección de
  host y pruebas de seguridad web y de API;
- subida limitada y transmitida en flujo, cuarentena, detección real de tipo, antivirus y
  política de fallo cerrado;
- auditoría de negocio separada de logs técnicos y exportada a un destino controlado por
  credenciales diferentes;
- secretos fuera de Git, imágenes, variables visibles y líneas de comandos, con permisos,
  rotación y recuperación documentados;
- imágenes inmutables, analizadas, no privilegiadas, sin `latest`, con usuario no root,
  capacidades eliminadas y límites de recursos;
- logs remotos minimizados, alertas, sincronización horaria y procedimiento de incidentes y
  brechas;
- copias cifradas de base, WAL, documentos, configuración, auditoría y material de
  recuperación, fuera del hipervisor, y una restauración completa satisfactoria.

La categoría ENS se obtiene del impacto y del análisis de riesgos, no del presupuesto. Este
perfil no constituye por sí mismo una declaración o certificación de conformidad.

### 15.5 Riesgos aceptados y condiciones de evolución

| Riesgo temporal | Medida compensatoria | Condición que obliga a evolucionar |
| --- | --- | --- |
| Una instancia por función | reinicio, alerta y reconstrucción automatizada | RTO exigido menor que el tiempo probado de recuperación |
| PostgreSQL sin failover | WAL, PITR y restauración ensayada | RPO/RTO o carga no satisfechos |
| Documentos sin réplica activa | copia externa, huellas y conciliación | pérdida potencial o recuperación fuera del objetivo |
| Un anfitrión de aplicación | límites, endurecimiento y proceso de reserva | incorporación de Cronos/otro módulo sensible o aislamiento insuficiente |
| Escalado manual | prueba de carga y preescalado antes de plazos | margen menor que el aprobado o picos impredecibles |
| Mantenimiento con corte | ventana y reversión medidas | disponibilidad exigida incompatible con el corte |
| Un único CPD/hipervisor | copia externa y reconstrucción | continuidad exige conmutación automática |
| PostgreSQL como cola | métricas, limpieza e idempotencia | edad, volumen, reparto o contención superan umbrales |

Cada riesgo tendrá propietario, impacto, medida, RTO/RPO, evidencia, fecha de revisión y
aceptación de los responsables de Servicio e Información asesorados por Seguridad y DPD.
Aceptar riesgo no permite incumplir una obligación normativa.

### 15.6 Variante excepcional de dos VM

Si no caben tres VM:

```text
VM 1 DMZ: NGINX + aplicación Go + trabajador restringido
VM 2 datos: PostgreSQL + cuarentena + objetos limpios
destinos externos: identidad, logs y copias
```

La base y los documentos nunca se unirán al servidor expuesto. La variante no entrará en
producción si no existen destino externo de logs, copia externa, restauración probada,
segmentación, límites y riesgo residual aceptado. Se documentará que proxy, aplicación y
trabajador comparten dominio de fallo y compiten por CPU y memoria.

### 15.7 Orden mínimo de preparación

1. Aprobar alcance Bolsa, categorización inicial, riesgos, RTO/RPO y servicios corporativos
   reutilizables.
2. Crear segmentos, reglas y tres VM endurecidas; verificar que solo 443 queda publicado y
   que datos no tiene ruta desde Internet.
3. Integrar identidad, PKI, hora, registro OCI, secretos, logs/SIEM y destino de copias.
4. Instalar PostgreSQL y almacenamiento documental; configurar cuentas, WAL, copia y
   restaurar en un entorno aislado.
5. Instalar NGINX, aplicación, trabajador y antivirus desde imágenes fijadas por digest.
6. Ejecutar pruebas funcionales, permisos, carga, ficheros hostiles, caída a mitad de
   operación, auditoría, copia, restauración y reversión.
7. Registrar las limitaciones aceptadas y autorizar la apertura mediante acta conjunta.

### 15.8 Condiciones de portabilidad hacia la plataforma objetivo

Desde la primera compilación serán obligatorios:

- imágenes OCI inmutables y la misma imagen entre entornos;
- núcleo Go sin estado local, sesiones externas o autocontenidas de forma segura y cierre
  ordenado;
- configuración, secretos y migraciones separados de la imagen;
- endpoints de arranque, vida y preparación;
- puertos para persistencia, documentos, antivirus, cola, identidad y avisos;
- operaciones, trabajadores y consumidores idempotentes;
- documentos referidos por clave y adaptador de almacenamiento, no por rutas locales;
- logs estructurados por salida estándar y auditoría transaccional;
- manifiestos de recursos, puertos, dependencias y política de red junto a cada entrega.

La evolución levantará RKE2 o la plataforma corporativa en paralelo, desplegará las mismas
imágenes, mantendrá inicialmente PostgreSQL y documentos fuera del clúster, probará la
carga y cambiará la ruta de NGINX. Después añadirá nodos, HPA/KEDA, datos HA y el segundo
dominio interno. Habrá trabajo de infraestructura y migración, pero no una reescritura de
Bolsa ni del núcleo.
