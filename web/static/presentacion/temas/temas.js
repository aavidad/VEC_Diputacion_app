export const TEMAS_VEC = Object.freeze([
  { id: "01", nombre: "Diputación Azul", caracter: "Institucional, sereno y familiar", rail: "#062d56", rail2: "#073b6c", primary: "#075fca", soft: "#e9f2ff", accent: "#8ab82d", bg: "#f5f8fc", surface: "#ffffff", text: "#10213b", muted: "#586985", border: "#dce5ef", success: "#17823b", map: "#dcecff" },
  { id: "02", nombre: "Sierra Verde", caracter: "Natural, público y cercano", rail: "#173f36", rail2: "#215447", primary: "#236b55", soft: "#e6f4ee", accent: "#b4cb3c", bg: "#f4f8f5", surface: "#ffffff", text: "#162720", muted: "#566b62", border: "#d4e2db", success: "#17653c", map: "#dcefe5" },
  { id: "03", nombre: "Vega Turquesa", caracter: "Luminoso, digital y amable", rail: "#083f4c", rail2: "#075767", primary: "#087889", soft: "#e2f5f6", accent: "#d49b19", bg: "#f2f8f9", surface: "#ffffff", text: "#102a31", muted: "#516b72", border: "#cfe2e5", success: "#24703f", map: "#d9f0f2" },
  { id: "04", nombre: "Alhambra Granate", caracter: "Solemne, cultural y cálido", rail: "#4a1622", rail2: "#642130", primary: "#8b2941", soft: "#f8e9ed", accent: "#c79a3b", bg: "#faf6f5", surface: "#ffffff", text: "#2d171b", muted: "#765c62", border: "#ead7db", success: "#386b3b", map: "#f1e2dc" },
  { id: "05", nombre: "Genil Cobalto", caracter: "Preciso, administrativo y actual", rail: "#142f57", rail2: "#1a4173", primary: "#235da8", soft: "#e8f0fb", accent: "#e18b2d", bg: "#f4f7fb", surface: "#ffffff", text: "#14233b", muted: "#596a83", border: "#d7e0ed", success: "#24713f", map: "#dce8f8" },
  { id: "06", nombre: "Albaicín Índigo", caracter: "Elegante, distintivo y sobrio", rail: "#292454", rail2: "#38316e", primary: "#5546a5", soft: "#efedfb", accent: "#b88d23", bg: "#f7f6fb", surface: "#ffffff", text: "#211d38", muted: "#666078", border: "#dfdceb", success: "#297044", map: "#e7e4f6" },
  { id: "07", nombre: "Poniente Terracota", caracter: "Humano, cálido y resolutivo", rail: "#4e2d24", rail2: "#674033", primary: "#9a4d32", soft: "#faece6", accent: "#b49a25", bg: "#faf6f3", surface: "#ffffff", text: "#312019", muted: "#75625b", border: "#eadbd4", success: "#3d713e", map: "#f0e4da" },
  { id: "08", nombre: "Costa Mediterránea", caracter: "Fresco, claro y optimista", rail: "#073d52", rail2: "#07556c", primary: "#087b8c", soft: "#e4f5f6", accent: "#d86548", bg: "#f2f9fa", surface: "#ffffff", text: "#112a34", muted: "#536d76", border: "#cfe3e6", success: "#27703f", map: "#d8eff3" },
  { id: "09", nombre: "Institucional Grafito", caracter: "Neutral, denso y profesional", rail: "#242b35", rail2: "#333c48", primary: "#3d5f88", soft: "#e9eef4", accent: "#a8c33f", bg: "#f4f5f7", surface: "#ffffff", text: "#1c232c", muted: "#5c6673", border: "#d9dde3", success: "#326b42", map: "#e2e7ec" },
  { id: "10", nombre: "Olivar", caracter: "Territorial, estable y orgánico", rail: "#343d20", rail2: "#4a542d", primary: "#66742f", soft: "#f0f3e5", accent: "#b67d1f", bg: "#f7f7f2", surface: "#ffffff", text: "#272b1c", muted: "#696d5a", border: "#e0e2d3", success: "#3d6a35", map: "#e8ecd9" },
  { id: "11", nombre: "Sierra Nevada", caracter: "Limpio, técnico y espacioso", rail: "#26384d", rail2: "#354d68", primary: "#4a6f9c", soft: "#ebf1f7", accent: "#7457b7", bg: "#f5f8fb", surface: "#ffffff", text: "#1a2939", muted: "#5e6d7d", border: "#d8e1ea", success: "#2f7048", map: "#e4edf5" },
  { id: "12", nombre: "Zaidín Ciruela", caracter: "Contemporáneo, urbano y singular", rail: "#402743", rail2: "#57365a", primary: "#76507b", soft: "#f3ebf4", accent: "#c5832e", bg: "#faf6fa", surface: "#ffffff", text: "#2d1e2f", muted: "#706074", border: "#e6d9e7", success: "#347044", map: "#eee3ef" },
  { id: "13", nombre: "Archivo Sepia", caracter: "Clásico, documental y contenido", rail: "#42372c", rail2: "#594b3d", primary: "#755b3e", soft: "#f4eee6", accent: "#9c6d1d", bg: "#f8f5f0", surface: "#fffefa", text: "#302820", muted: "#6f655b", border: "#e4ddd3", success: "#436b3c", map: "#eee7dc" },
  { id: "14", nombre: "Noche Atlántica", caracter: "Oscuro, concentrado y premium", rail: "#091724", rail2: "#10283a", primary: "#205c8d", soft: "#183a54", accent: "#b8d84a", bg: "#0f1d29", surface: "#172938", text: "#f2f7fb", muted: "#b9c8d3", border: "#365064", success: "#72ca8b", map: "#203b4d" },
  { id: "15", nombre: "Cívico Alto Contraste", caracter: "Máxima claridad y foco visible", rail: "#000000", rail2: "#202020", primary: "#004fc4", soft: "#dceaff", accent: "#f1c400", bg: "#ffffff", surface: "#ffffff", text: "#000000", muted: "#333333", border: "#565656", success: "#086b2a", map: "#dceaff" },
].map((tema) => Object.freeze(tema)));

function aplicarTokens(elemento, tema) {
  for (const clave of ["rail", "rail2", "primary", "soft", "accent", "bg", "surface", "text", "muted", "border", "success", "map"]) {
    elemento.style.setProperty(`--tema-${clave}`, tema[clave]);
  }
}

export function montarGaleriaTemas(documento = globalThis.document) {
  const galeria = documento?.querySelector?.("#galeria-temas");
  const plantilla = documento?.querySelector?.("#plantilla-tema");
  const salida = documento?.querySelector?.("#tema-elegido");
  if (!galeria || !plantilla?.content || !salida) return Object.freeze({ desmontar() {} });
  const escuchas = [];
  TEMAS_VEC.forEach((tema) => {
    const fragmento = plantilla.content.cloneNode(true);
    const tarjeta = fragmento.querySelector(".tema");
    tarjeta.dataset.tema = tema.id;
    aplicarTokens(tarjeta, tema);
    fragmento.querySelector(".tema-numero").textContent = tema.id;
    fragmento.querySelector("h2").textContent = tema.nombre;
    fragmento.querySelector(".tema-caracter").textContent = tema.caracter;
    const boton = fragmento.querySelector(".elegir-tema");
    boton.textContent = `Elegir ${tema.id} · ${tema.nombre}`;
    const alElegir = () => {
      galeria.querySelectorAll(".tema").forEach((otra) => otra.removeAttribute("data-seleccionado"));
      tarjeta.dataset.seleccionado = "true";
      salida.textContent = `Seleccionado: ${tema.id} · ${tema.nombre}`;
    };
    boton.addEventListener("click", alElegir);
    escuchas.push([boton, alElegir]);
    galeria.append(fragmento);
  });
  return Object.freeze({
    desmontar() { escuchas.forEach(([boton, escucha]) => boton.removeEventListener("click", escucha)); },
  });
}

if (typeof document !== "undefined") montarGaleriaTemas(document);
