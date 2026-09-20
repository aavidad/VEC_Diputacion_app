---
name: disenar-sistema-visual-vec
description: Diseña o unifica pantallas web de VEC con el sistema visual corporativo común, permitiendo variar el tema cromático sin cambiar la estructura, semántica ni comportamiento de los componentes.
---

# Diseñar el sistema visual VEC

Mantener una sola gramática visual para todos los módulos. El color puede variar
por tema; la estructura, los estados, la densidad, la navegación y la respuesta
móvil no varían por módulo.

Respetar la arquitectura hexagonal: dominio y casos de uso no conocen HTML, CSS
ni temas. La vista recibe modelos de presentación; los clientes HTTP implementan
puertos del frontend; el montaje inyecta ambos. Un cambio de tema no altera DTO,
caso de uso, autorización, endpoint, recibo ni persistencia.

## Formato común

Componer las pantallas, en este orden cuando sus partes existan:

1. shell corporativo con navegación lateral y cabecera contextual;
2. título, miga y estado operativo real del módulo;
3. navegación del recorrido o pestañas;
4. indicadores con `rejilla-kpi` y `tarjeta-kpi`;
5. área principal de trabajo con `panel`, `cabecera-panel` y `cuerpo-panel`;
6. contexto o resumen lateral solo cuando ayude a decidir;
7. acciones principales al final del bloque que modifican.

Reutilizar primero los componentes de `web/static/portal-empleado/portal-componentes.css`
y los tokens de `portal.css`. No crear un componente paralelo si el común expresa
la misma función. Las variantes de tema solo redefinen tokens `--portal-*`.

## Reglas de decisión

- Mostrar valor, etiqueta y estado como elementos separados; nunca concatenar KPI
  o metadatos en texto corrido.
- Poner instrucciones extensas detrás de Ayuda. Dejar en la tarea solo contexto,
  validación, error, resultado y límites imprescindibles.
- Distinguir conectado, solo lectura, pendiente, vacío, error y denegado. No usar
  color como única señal.
- Una acción sin caso de uso queda deshabilitada y explica la dependencia concreta.
- No afirmar firma, pago, envío, registro, cálculo o persistencia sin respuesta real.
- Mantener foco visible, controles etiquetados, contraste WCAG AA y orden semántico.
- En móvil apilar navegación, KPI y columnas; las tablas usan desplazamiento interno,
  nunca ensanchan la página.

## Temas

Comparar temas sobre el mismo mini-recorrido: navegación, KPI, panel, formulario,
botón, estado, tabla y mapa. Cambiar únicamente tokens de color y, si se justifica,
radio/sombra. No variar contenido ni distribución para favorecer una alternativa.

La galería mantenida en `web/static/presentacion/temas/` es el catálogo de decisión.
Cuando dirección elija un tema, trasladar sus tokens a la raíz común y comprobar al
menos Portal, Dietas, Cronos, Personal y Bolsa en escritorio y móvil. El tema de alto
contraste sigue siendo una capa de accesibilidad independiente de la paleta elegida.

## Comprobación

Verificar contraste de texto y controles, 1440 px y 390 px, foco por teclado,
desbordamiento horizontal, estados carga/vacío/error y que la variante no añade
cookies, almacenamiento web ni llamadas externas.
