# Sistema de diseño, plantillas y temas visuales

Estado: requisito transversal incorporado al estudio.

Fecha: 14 de julio de 2026.

## 1. Objetivo

Toda la familia de portales y módulos de RRHH tendrá una apariencia coherente y gobernada
desde un único sistema de diseño. Cambiar la identidad visual general deberá consistir en
activar una nueva versión de tema aprobada, sin editar cada pantalla ni recompilar el
dominio de los módulos.

El resultado debe ser conceptualmente parecido al sistema de plantillas de Joomla, pero
con controles adicionales de compatibilidad, accesibilidad, seguridad y trazabilidad.

Este requisito no contradice la separación entre portal externo, portal del empleado,
espacio del responsable y área interna de RRHH. Las superficies de seguridad serán
distintas, pero consumirán la misma fuente visual versionada.

## 2. Principio de herencia

La resolución visual seguirá esta cadena:

```text
sistema de diseño base
→ tema institucional
→ variante de la superficie
→ acento opcional del módulo
→ preferencias de accesibilidad de la persona
```

Ejemplos:

- Bolsa hereda el tema institucional completo y puede aplicar un color de acento propio.
- Dietas puede utilizar otro acento sin cambiar botones, tablas, tipografía o formularios.
- Cronos puede tener una identidad reconocible conservando la misma navegación y los
  mismos estados de interacción.
- El modo de alto contraste o la preferencia de movimiento reducido prevalecerán sobre
  cualquier personalización del módulo.

La configuración por defecto será heredar todo. Un módulo que no declare variante alguna
se verá automáticamente como el resto del portal.

### 2.1 Preferencias personales de apariencia

Cada persona podrá adaptar la presentación sin alterar el contenido, los permisos ni el
significado jurídico de las acciones. Como mínimo se ofrecerán variantes aprobadas para:

- seguir la preferencia del sistema o dispositivo;
- tema institucional claro y oscuro;
- alto contraste y compatibilidad con los colores forzados del sistema;
- escala de texto, interlineado, separación entre caracteres y anchura de lectura;
- tipografía de alta legibilidad entre las autorizadas;
- densidad de la interfaz, sin ocultar datos o acciones obligatorias;
- realce adicional de enlaces y foco de teclado;
- reducción o eliminación de movimiento no esencial.

La persona no podrá introducir CSS, fuentes, imágenes, scripts o colores arbitrarios. Solo
elegirá combinaciones validadas del sistema de diseño. Ninguna preferencia podrá reducir el
contraste, el tamaño de interacción, la visibilidad del foco o la semántica por debajo del
nivel obligatorio.

Antes de autenticarse se aplicarán inicialmente las preferencias que exponga el navegador o
el sistema operativo. La elección explícita de la persona tendrá prioridad y podrá guardarse
localmente. Al autenticarse podrá sincronizarla entre dispositivos mediante su perfil, sin
que una combinación local sobrescriba silenciosamente una elección ya guardada.

La plataforma almacenará ajustes de interfaz, no diagnósticos como ceguera, baja visión o
dislexia. Estas preferencias no formarán parte del expediente de personal ni podrán ser
consultadas por RRHH para tomar decisiones sobre una persona. Solo los servicios técnicos
estrictamente necesarios podrán tratar los ajustes para aplicarlos y resolver incidencias.

El cambio será inmediato, reversible y conservará la pantalla, los filtros, el foco y los
borradores abiertos. Siempre habrá una acción accesible para restablecer la apariencia. Si
una actualización vuelve incompatible una preferencia, se aplicará una alternativa segura y
se informará a la persona sin bloquear su acceso.

## 3. Elementos centralizados

### 3.1 Variables de diseño

No se usarán colores, tamaños, tipografías, sombras o espacios arbitrarios en las pantallas.
Se definirán variables semánticas centrales para:

- identidad institucional y color de acento;
- fondos, superficies, bordes y separadores;
- texto principal, secundario e invertido;
- foco, selección, interacción y elementos deshabilitados;
- información, éxito, advertencia, error y estados administrativos;
- tipografía, tamaños, pesos y altura de línea;
- escala de separación, densidad, radios y elevaciones;
- iconos, tamaños y grosor de trazo;
- duración y curva de las transiciones;
- anchuras, puntos de adaptación y capas visuales.

En web podrán materializarse mediante propiedades personalizadas CSS; también se
exportarán en un formato neutral, por ejemplo JSON, para una futura aplicación de
escritorio u otros clientes.

Los módulos consumirán nombres semánticos como `color_acento`, `texto_principal` o
`estado_error`, no códigos hexadecimales.

### 3.2 Biblioteca común de componentes

La plataforma proporcionará componentes accesibles y tematizables para:

- cabecera, navegación, migas y pie;
- selector y aviso del perfil de seguridad activo;
- botones, enlaces, menús, pestañas y cuadros de diálogo;
- formularios, campos, ayudas, errores y asistentes por pasos;
- tablas densas, filtros, paginación y selección masiva;
- tarjetas, paneles laterales y vistas de detalle;
- etiquetas de estado, alertas y notificaciones;
- líneas temporales, documentos, firmas y justificantes;
- calendarios, contadores y plazos;
- gráficos con alternativa tabular;
- carga, vacío, error, bloqueo, conflicto y operación completada.

Un módulo compondrá estos elementos; no creará su propia versión de un botón, formulario o
tabla salvo que exista una necesidad funcional nueva aprobada para incorporarla a la
biblioteca común.

### 3.3 Marcos de portal

Cada superficie tendrá un marco propio porque su navegación y densidad son diferentes:

- marco público y del aspirante;
- marco del empleado;
- marco del responsable;
- marco de RRHH;
- marco de administración.

Todos compartirán identidad visual, variables y componentes. Separar los marcos evita que
un cambio visual convierta accidentalmente una superficie pública en una extensión del
área interna.

## 4. Qué puede personalizar un módulo

Un módulo podrá declarar mediante configuración aprobada:

- color de acento y su color de contraste;
- icono dentro de la familia institucional;
- ilustración o cabecera autorizada;
- nombre corto y descripción;
- densidad preferida entre variantes previstas;
- variante de determinados componentes ya existentes.

No podrá redefinir por sí mismo:

- tipografía general;
- navegación principal;
- foco de teclado;
- significado de éxito, advertencia o error;
- componentes de formulario;
- tamaños mínimos de interacción;
- reglas de accesibilidad;
- estilos de acciones jurídicas como firmar, registrar o resolver;
- CSS global, JavaScript visual o recursos remotos arbitrarios.

El acento del módulo servirá para orientación y reconocimiento, no para transmitir por sí
solo un estado. Por ejemplo, una bolsa azul y un Cronos violeta seguirán utilizando el
mismo patrón global de error, acompañado siempre de texto e icono.

Para evitar una interfaz multicolor y fragmentada, el acento se limitará normalmente a
navegación activa, encabezados, iconos, indicadores y acciones principales. Fondos,
formularios y tablas mantendrán una base común.

## 5. CSS y estilos propios

El objetivo no debe interpretarse como la prohibición absoluta de cualquier regla local.
Una pantalla especializada puede necesitar estructura propia, por ejemplo la cuadrícula de
un baremo o un calendario. En tal caso:

- el estilo será encapsulado y no afectará a otros módulos;
- solo definirá disposición o estructura específica;
- utilizará exclusivamente variables y componentes centrales;
- no incluirá colores, fuentes, sombras o estados visuales propios;
- tendrá pruebas en todos los temas compatibles.

La mayor parte de las pantallas no debería necesitar CSS visual específico del módulo.

## 6. Paquetes de tema

Cada tema será un paquete declarativo y versionado con, al menos:

```text
identificador
nombre
versión
tema del que hereda
variables visuales
recursos institucionales
superficies compatibles
variantes clara, oscura y alto contraste
informe de contraste
versión mínima del sistema de diseño
huella e información de aprobación
```

La selección del tema podrá configurarse por entorno, superficie, módulo y persona, siempre
dentro de las variantes aprobadas. La aplicación resolverá la herencia en tiempo de
ejecución o despliegue sin modificar el núcleo.

Los temas se distribuirán con la aplicación o desde infraestructura propia controlada. No se
dependerá de hojas de estilo, fuentes o bibliotecas cargadas desde terceros en Internet.

## 7. Administración visual desde la aplicación

La administración funcional permitirá:

- crear una variante a partir de un tema aprobado;
- modificar únicamente variables autorizadas;
- previsualizar todas las superficies y componentes;
- comprobar contraste, estados y puntos de adaptación;
- comparar la versión vigente con la propuesta;
- someterla a revisión y aprobación;
- programar su entrada en vigor;
- publicar y revertir;
- conocer qué versión estaba activa en cada momento.

No permitirá introducir CSS, HTML o JavaScript arbitrarios. Un cambio estructural de
plantilla o un componente nuevo seguirá el proceso de desarrollo, revisión de seguridad y
pruebas.

## 8. Accesibilidad obligatoria

Todos los temas deberán superar los mismos criterios:

- contraste mínimo aplicable a texto, iconos, foco y controles;
- foco visible y navegación completa por teclado;
- información no basada únicamente en color;
- etiquetas persistentes y errores asociados a los campos;
- ampliación de texto sin pérdida de contenido;
- objetivos de interacción de tamaño suficiente;
- lector de pantalla y orden semántico;
- movimiento reducido;
- variantes clara, oscura y alto contraste verificadas por separado;
- diseño adaptable sin desplazamiento horizontal indebido.

Un tema que no supere estas pruebas no podrá publicarse, aunque respete los colores de
marca.

## 9. Seguridad del sistema de temas

- Política de seguridad de contenido estricta y sin estilos o scripts remotos no
  autorizados.
- Recursos autoalojados, inventariados y versionados por huella.
- Validación de formato, tamaño y contenido de imágenes y fuentes.
- Temas aprobados, firmados o integrados en artefactos confiables.
- Sin expresiones, URLs, HTML o código ejecutable introducidos desde el editor de temas.
- Permisos separados para editar, aprobar y publicar.
- Auditoría de cada cambio y reversión.
- Caché invalidada por versión, sin mezclar recursos de dos temas durante una operación.

## 10. Pruebas y gobierno

El sistema de diseño tendrá una versión y responsables definidos. Cada entrega incluirá:

- catálogo interactivo de componentes y estados;
- ejemplos para formularios y pantallas administrativas densas;
- pruebas automáticas de accesibilidad;
- comprobación manual con teclado y lector de pantalla;
- pruebas visuales de regresión por tema, módulo y tamaño de pantalla;
- comprobación de contraste;
- pruebas de impresión y documentos cuando proceda;
- reglas automáticas que impidan colores o fuentes fijados en módulos;
- matriz de compatibilidad entre sistema de diseño, temas y módulos;
- guía para crear componentes y variantes.

La versión de un módulo declarará la versión mínima y máxima compatible del sistema de
diseño. Un cambio incompatible del tema o de un componente exigirá migración y no se
publicará de manera silenciosa.

## 11. Documentos generados

Contratos, resoluciones, certificados y justificantes compartirán identidad institucional,
pero sus plantillas serán independientes del tema web y estarán también versionadas.

Cambiar el color del portal no modificará un documento ya firmado ni regenerará su
apariencia. Cada documento conservará la versión exacta de plantilla y recursos con los que
fue producido.

## 12. Estructura conceptual recomendada

```text
interfaz/
├── sistema_diseno/
│   ├── variables/
│   ├── componentes/
│   ├── patrones/
│   └── accesibilidad/
├── temas/
│   ├── base/
│   ├── institucional/
│   ├── alto_contraste/
│   └── acentos_modulos/
├── marcos/
│   ├── publico/
│   ├── empleado/
│   ├── responsable/
│   ├── rrhh/
│   └── administracion/
└── catalogo_componentes/
```

La estructura física definitiva dependerá de la tecnología web elegida, pero esta
separación será parte del contrato arquitectónico.

## 13. Requisitos verificables iniciales

- **VIS-001.** Ningún módulo fijará colores, tipografías o sombras fuera del sistema de
  variables aprobado.
- **VIS-002.** Cambiar el tema institucional actualizará todas las superficies compatibles
  sin modificar el código de dominio de los módulos.
- **VIS-003.** Un módulo heredará el tema institucional si no declara una variante.
- **VIS-004.** La variante de módulo solo podrá modificar las variables expresamente
  autorizadas.
- **VIS-005.** Las preferencias de accesibilidad prevalecerán sobre el tema y el acento del
  módulo.
- **VIS-006.** Todos los componentes comunes tendrán estados normal, foco, interacción,
  deshabilitado, carga y error.
- **VIS-007.** Cada tema superará pruebas de contraste, teclado, lector de pantalla,
  ampliación y diseño adaptable.
- **VIS-008.** Los temas no contendrán código ejecutable ni dependencias remotas no
  aprobadas.
- **VIS-009.** Editar, aprobar y publicar un tema serán permisos segregados y auditados.
- **VIS-010.** Las pruebas visuales de regresión cubrirán tema institucional, alto contraste
  y acentos de los módulos.
- **VIS-011.** Los estados jurídicos y operativos mantendrán el mismo significado visual en
  todos los módulos.
- **VIS-012.** Las plantillas documentales conservarán su propia versión y no cambiarán
  retroactivamente al activar un tema web.
- **VIS-013.** Todas las superficies, incluso antes de la autenticación, ofrecerán un panel
  de apariencia utilizable con teclado y lector de pantalla.
- **VIS-014.** La persona solo podrá seleccionar temas y perfiles de lectura aprobados; no
  podrá introducir código, recursos o valores visuales arbitrarios.
- **VIS-015.** Escala de texto, tipografía autorizada, interlineado, espaciado, anchura de
  lectura, densidad, contraste, foco y movimiento reducido serán ajustes independientes.
- **VIS-016.** Ninguna personalización podrá rebajar los mínimos obligatorios de contraste,
  foco, interacción, semántica o legibilidad.
- **VIS-017.** Se respetarán inicialmente las preferencias del dispositivo; una elección
  explícita y compatible de la persona tendrá prioridad.
- **VIS-018.** Las preferencias anónimas podrán conservarse localmente y las autenticadas se
  sincronizarán entre dispositivos con un procedimiento de combinación explícito.
- **VIS-019.** Se almacenarán preferencias funcionales de interfaz, nunca diagnósticos de
  discapacidad; no serán datos disponibles para decisiones de RRHH.
- **VIS-020.** Cambiar de apariencia no perderá campos, filtros, foco, desplazamiento ni
  borradores abiertos.
- **VIS-021.** Alto contraste, colores forzados y demás necesidades de accesibilidad
  prevalecerán sobre los acentos de Bolsa, Dietas, Cronos o cualquier otro módulo.
- **VIS-022.** Existirá una acción accesible para restablecer la apariencia y una
  recuperación segura ante configuraciones incompatibles.
- **VIS-023.** Todas las combinaciones admitidas se probarán con ampliación, redistribución,
  teclado, lectores de pantalla, colores forzados y movimiento reducido.

## 14. Decisiones pendientes

- Manual oficial de identidad gráfica de la Diputación y usos permitidos del escudo o
  logotipo.
- Tipografía institucional y posibilidad de autoalojarla.
- Paleta base y colores de acento permitidos.
- Tecnología de componentes y catálogo interactivo.
- Grado de personalización permitido a administradores funcionales.
- Si los acentos se asignarán por módulo, por superficie o por ambos.

Hasta resolverlas se define la arquitectura de temas, pero no una paleta definitiva.
