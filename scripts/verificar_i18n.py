#!/usr/bin/env python3
"""Puerta estática de catálogos y literales visibles de la interfaz web."""
from __future__ import annotations

import json
import re
import sys
from html.parser import HTMLParser
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
WEB = ROOT / "web/static"
MARCADOR = re.compile(r"\{[a-z_]+\}", re.I)
VISIBLE_JS = re.compile(
    r"(?:\.textContent\s*=|\.innerHTML\s*=|\.setAttribute\(\s*['\"](?:aria-label|title|placeholder)['\"]\s*,)\s*"
    r"(['\"`])([A-Za-zÁÉÍÓÚÜÑáéíóúüñ][^'\"`]*?)\1"
)
TEXTO_PLANTILLA = re.compile(r"<[A-Za-z][^>]*>\s*([A-Za-zÁÉÍÓÚÜÑáéíóúüñ][^<>{}$]+?)\s*<")
ATRIBUTO_PLANTILLA = re.compile(r"(?:aria-label|title|placeholder|alt)=([\"'])([A-Za-zÁÉÍÓÚÜÑáéíóúüñ][^\"'${}]+)\1")
EXCEPCIONES_HTML = {
    # Una inicial decorativa oculta al lector; no transmite información.
    ("portal-empleado/index.html", "P"),
    ("acceso/index.html", "i"),
}
ATRIBUTOS_VISIBLES = {"aria-label", "title", "placeholder", "alt"}
COBERTURA = json.loads((ROOT / "scripts/i18n_cobertura.json").read_text())
# Marcadores de clave admitidos: el genérico `data-i18n*` y el propio de las
# copias de Contratación temporal (`data-ct-copia*`), que resuelve su traductor.
PREFIJOS_MARCA = ("data-i18n", "data-ct-copia")
EXCEPCIONES_JS = {
    # Acrónimos y marcas técnicas invariables entre idiomas.
    "PDF", "SMS", "SHA-256",
}


def claves(objeto: object, prefijo: str = "") -> dict[str, str]:
    if not isinstance(objeto, dict):
        raise ValueError("el catálogo debe ser un objeto")
    salida = {}
    for clave, valor in objeto.items():
        nombre = f"{prefijo}.{clave}" if prefijo else clave
        if isinstance(valor, dict):
            salida.update(claves(valor, nombre))
        elif isinstance(valor, str) and valor:
            salida[nombre] = valor
        else:
            raise ValueError(f"{nombre}: traducción vacía o no textual")
    return salida


def verificar_catalogos() -> list[str]:
    fallos = []
    for es in [ROOT / "locales/es.json", *WEB.rglob("locales/es.json")]:
        en = es.with_name("en.json")
        if not en.exists():
            fallos.append(f"{en.relative_to(ROOT)}: falta catálogo inglés")
            continue
        try:
            original = claves(json.loads(es.read_text()))
            traducido = claves(json.loads(en.read_text()))
        except (ValueError, json.JSONDecodeError) as error:
            fallos.append(f"{es.relative_to(ROOT)}: {error}")
            continue
        for clave in sorted(original.keys() ^ traducido.keys()):
            fallos.append(f"{en.relative_to(ROOT)}: clave asimétrica {clave}")
        for clave in sorted(original.keys() & traducido.keys()):
            if sorted(MARCADOR.findall(original[clave])) != sorted(MARCADOR.findall(traducido[clave])):
                fallos.append(f"{en.relative_to(ROOT)}: marcadores distintos en {clave}")
    return fallos


class TextosHTML(HTMLParser):
    def __init__(self, ruta: Path):
        super().__init__(convert_charrefs=True)
        self.ruta = ruta
        self.pila: list[tuple[str, dict[str, str | None]]] = []
        self.fallos: list[str] = []

    def handle_starttag(self, tag: str, attrs: list[tuple[str, str | None]]) -> None:
        atributos = dict(attrs)
        for nombre in ATRIBUTOS_VISIBLES | ({"content"} if tag == "meta" and atributos.get("name") == "description" else set()):
            valor = (atributos.get(nombre) or "").strip()
            if not any(c.isalpha() for c in valor) or atributos.get("aria-hidden") == "true":
                continue
            marcado = any(f"{prefijo}-{nombre}" in atributos for prefijo in (*PREFIJOS_MARCA, "data-i18n-portal"))
            marcado = marcado or f"{nombre}:" in (atributos.get("data-i18n-atributo") or "")
            if not marcado:
                self.fallos.append(f"{self.ruta.relative_to(ROOT)}: atributo {nombre} sin clave: {valor[:90]}")
        if tag not in {"br", "hr", "img", "input", "link", "meta", "source", "area", "embed", "wbr"}:
            self.pila.append((tag, atributos))

    def handle_startendtag(self, tag: str, attrs: list[tuple[str, str | None]]) -> None:
        pass

    def handle_endtag(self, tag: str) -> None:
        for indice in range(len(self.pila) - 1, -1, -1):
            if self.pila[indice][0] == tag:
                del self.pila[indice:]
                break

    def handle_data(self, data: str) -> None:
        texto = data.strip()
        if not texto or not any(letra.isalpha() for letra in texto) or not self.pila:
            return
        tag, attrs = self.pila[-1]
        if tag in {"script", "style", "svg"} or attrs.get("aria-hidden") == "true":
            return
        # Sin JavaScript no hay traductor posible: el aviso de <noscript>
        # queda en el idioma base del documento.
        if any(etiqueta == "noscript" for etiqueta, _ in self.pila):
            return
        if any(nombre.startswith(PREFIJOS_MARCA) for nombre in attrs):
            return
        relativa = str(self.ruta.relative_to(WEB))
        if (relativa, texto) in EXCEPCIONES_HTML:
            return
        self.fallos.append(f"{self.ruta.relative_to(ROOT)}: texto HTML sin clave: {texto[:90]}")


def cubierto(ruta: Path, prefijos: list[str], excluidos: list[str] = ()) -> bool:
    relativa = ruta.relative_to(WEB).as_posix()
    return any(relativa.startswith(p) for p in prefijos) and not any(relativa.startswith(p) for p in excluidos)


def verificar_web() -> list[str]:
    fallos = []
    prefijos, excluidos = COBERTURA["literales"], COBERTURA["literales_excluidos"]
    for ruta in WEB.rglob("*.html"):
        if "vendor" in ruta.parts or not cubierto(ruta, prefijos, excluidos):
            continue
        analizador = TextosHTML(ruta)
        analizador.feed(ruta.read_text(errors="replace"))
        fallos.extend(analizador.fallos)
    for ruta in WEB.rglob("*.js"):
        if "vendor" in ruta.parts or ruta.name.endswith(".test.js") or not cubierto(ruta, prefijos, excluidos):
            continue
        for numero, linea in enumerate(ruta.read_text(errors="replace").splitlines(), 1):
            for coinc in VISIBLE_JS.finditer(linea):
                texto = coinc.group(2).strip()
                if texto in EXCEPCIONES_JS or not any(c.isalpha() for c in texto):
                    continue
                fallos.append(f"{ruta.relative_to(ROOT)}:{numero}: literal visible sin clave: {texto[:90]}")
            # Los catálogos i18n pueden contener HTML traducido. En el resto,
            # texto estático entre etiquetas dentro de plantillas es interfaz.
            if "i18n" not in ruta.name:
                for coinc in TEXTO_PLANTILLA.finditer(linea):
                    texto = coinc.group(1).strip()
                    if texto not in EXCEPCIONES_JS:
                        fallos.append(f"{ruta.relative_to(ROOT)}:{numero}: HTML de plantilla sin clave: {texto[:90]}")
                for coinc in ATRIBUTO_PLANTILLA.finditer(linea):
                    fallos.append(f"{ruta.relative_to(ROOT)}:{numero}: atributo de plantilla sin clave: {coinc.group(2)[:90]}")
    return fallos


def main() -> int:
    fallos = verificar_catalogos() + verificar_web()
    if fallos:
        for fallo in fallos[:100]:
            print(fallo, file=sys.stderr)
        print(f"i18n: {len(fallos)} incumplimientos; se muestran hasta 100", file=sys.stderr)
        return 1
    print("i18n: catálogos simétricos y sin literales visibles detectados")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
