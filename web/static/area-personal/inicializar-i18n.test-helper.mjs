import { iniciarI18nAreaPersonal } from "./i18n.js";
import { lectorCatalogos } from "./textos-prueba.test-helper.mjs";

// Estas pruebas renderizan directamente vistas y controles sin pasar por arranque.js.
await iniciarI18nAreaPersonal({ querySelectorAll: () => [], documentElement: {} }, {
  leer: lectorCatalogos(), pantalla: "preferencias",
  ubicacion: { href: "https://vec.example/area-personal/?lang=es" },
});
