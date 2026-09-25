"""Personas inventadas, verosímiles y deterministas para el paquete de ejemplo.

Rangos elegidos para que ningún dato pueda coincidir con uno real utilizable:

- DNI: números del bloque 99 000 000–99 999 999, que la Dirección General de la
  Policía no ha expedido (la numeración vigente no se acerca a ese bloque); la
  letra de control se calcula con el algoritmo oficial (módulo 23).
- NIE: letra Z seguida de 9 000 000–9 999 999, tramo alto que tampoco se ha
  asignado; letra de control con el mismo algoritmo (Z = 2).
- Teléfono: móviles del subrango 79x, sin atribuir en el Plan Nacional de
  Numeración a la fecha de redacción. Antes de un uso fuera de desarrollo debe
  confirmarse con la CNMC; VEC no dispone de pasarela de SMS.
- Correo: dominio `correo.internal`. `.internal` está reservado por ICANN para
  uso privado y nunca se delegará en el DNS público, así que ningún mensaje
  puede llegar a un buzón real; a diferencia de `.test`, `.example` o
  `.invalid`, no parece un dato de prueba en pantalla.
- Domicilio: municipios reales de la provincia de Granada con su código postal
  y nombres de calle corrientes; la combinación con una persona es inventada.

El documento enmascarado sigue la regla de Convoca («***NNNN**», cifras cuarta a
séptima) y se hace único en todo el paquete para que dos personas no se
confundan en pantalla.
"""
from __future__ import annotations

import random
import unicodedata
from dataclasses import dataclass, field
from datetime import date, timedelta

from . import catalogos

LETRAS_DNI = "TRWAGMYFPDXBNJZSQVHLCKE"
DOMINIO_CORREO = "correo.internal"
PROVINCIA = "Granada"
FECHA_REFERENCIA = date(2026, 9, 28)


def letra_control(numero: int) -> str:
    """Letra oficial del DNI/NIE para un número de ocho cifras."""
    return LETRAS_DNI[numero % 23]


def documento_valido(documento: str) -> bool:
    """Comprueba formato y letra de un DNI (8 cifras) o NIE (X/Y/Z + 7 cifras)."""
    if len(documento) != 9 or not documento[-1].isalpha():
        return False
    cuerpo = documento[:-1]
    if cuerpo[0] in "XYZ":
        cuerpo = str("XYZ".index(cuerpo[0])) + cuerpo[1:]
    return cuerpo.isdigit() and letra_control(int(cuerpo)) == documento[-1]


def documento_en_rango_reservado(documento: str) -> bool:
    return (documento[:2] == "99" and documento[:8].isdigit()) or documento[:2] == "Z9"


def enmascarar(documento: str) -> str:
    """Máscara de Convoca: se ven solo las posiciones cuarta a séptima."""
    return "***" + documento[3:7] + "**"


def sin_tildes(texto: str) -> str:
    return "".join(c for c in unicodedata.normalize("NFD", texto) if unicodedata.category(c) != "Mn")


@dataclass(frozen=True)
class Persona:
    referencia: str
    nombre: str
    primer_apellido: str
    segundo_apellido: str
    sexo: str
    documento: str
    fecha_nacimiento: date
    correo: str
    telefono: str
    domicilio: dict = field(hash=False, compare=False)

    @property
    def tipo_documento(self) -> str:
        return "NIE" if self.documento[0] in "XYZ" else "DNI"

    @property
    def documento_enmascarado(self) -> str:
        return enmascarar(self.documento)

    @property
    def nombre_completo(self) -> str:
        return f"{self.nombre} {self.primer_apellido} {self.segundo_apellido}"

    @property
    def tratamiento(self) -> str:
        return "Dña." if self.sexo == "M" else "D."

    def como_dict(self) -> dict:
        return {
            "referencia": self.referencia, "nombre": self.nombre,
            "primer_apellido": self.primer_apellido, "segundo_apellido": self.segundo_apellido,
            "nombre_completo": self.nombre_completo, "sexo": self.sexo,
            "tipo_documento": self.tipo_documento, "documento": self.documento,
            "documento_enmascarado": self.documento_enmascarado,
            "fecha_nacimiento": self.fecha_nacimiento.isoformat(), "correo": self.correo,
            "telefono": self.telefono, "domicilio": self.domicilio,
        }


class GeneradorPersonas:
    """Genera personas sin repetir nombre completo, documento, máscara, correo ni teléfono."""

    def __init__(self, azar: random.Random, prefijo_referencia: str) -> None:
        self._azar = azar
        self._prefijo = prefijo_referencia
        self._nombres: set[tuple[str, str, str]] = set()
        self._mascaras: set[str] = set()
        self._correos: set[str] = set()
        self._telefonos: set[str] = set()
        self.personas: list[Persona] = []

    def _identidad(self) -> tuple[str, str, str, str]:
        while True:
            sexo = "M" if self._azar.random() < 0.55 else "H"
            nombre = self._azar.choice(catalogos.NOMBRES_MUJER if sexo == "M" else catalogos.NOMBRES_HOMBRE)
            primero, segundo = self._azar.sample(catalogos.APELLIDOS, 2)
            if (nombre, primero, segundo) not in self._nombres:
                self._nombres.add((nombre, primero, segundo))
                return sexo, nombre, primero, segundo

    def _documento(self) -> str:
        while True:
            mascara = f"{self._azar.randrange(10000):04d}"
            if mascara not in self._mascaras:
                self._mascaras.add(mascara)
                break
        cifra, final = self._azar.randrange(10), self._azar.randrange(10)
        if self._azar.random() < 0.06:
            numero = int(f"29{cifra}{mascara}{final}")  # Z9… con Z = 2
            return f"Z9{cifra}{mascara}{final}{letra_control(numero)}"
        numero = int(f"99{cifra}{mascara}{final}")
        return f"{numero}{letra_control(numero)}"

    def _correo(self, nombre: str, primero: str, segundo: str) -> str:
        pila = sin_tildes(nombre.split()[0]).lower()
        a1, a2 = (sin_tildes(x).lower().replace(" ", "") for x in (primero, segundo))
        candidatos = [f"{pila}.{a1}", f"{pila}.{a1}.{a2}", f"{pila[0]}{a1}{a2}", f"{pila}{a1}{a2[0]}"]
        candidatos += [f"{pila}.{a1}{n}" for n in range(2, 100)]
        for local in candidatos:
            correo = f"{local}@{DOMINIO_CORREO}"
            if correo not in self._correos:
                self._correos.add(correo)
                return correo
        raise RuntimeError("no se encontró un correo libre")

    def _telefono(self) -> str:
        while True:
            numero = f"79{self._azar.randrange(10)} {self._azar.randrange(1000):03d} {self._azar.randrange(1000):03d}"
            if numero not in self._telefonos:
                self._telefonos.add(numero)
                return numero

    def _domicilio(self) -> dict:
        municipio, codigos = self._azar.choices(
            catalogos.MUNICIPIOS, weights=[12 if m == "Granada" else 1 for m, _ in catalogos.MUNICIPIOS])[0]
        via = self._azar.choice(catalogos.VIAS)
        numero = self._azar.randint(1, 60)
        piso = self._azar.choice(["", "", f"{self._azar.randint(1, 5)}º {self._azar.choice('ABCD')}"])
        return {"via": via, "numero": str(numero), "piso": piso, "codigo_postal": self._azar.choice(codigos),
                "municipio": municipio, "provincia": PROVINCIA}

    def nueva(self, edad_minima: int = 21, edad_maxima: int = 62) -> Persona:
        sexo, nombre, primero, segundo = self._identidad()
        edad_dias = self._azar.randint(edad_minima * 365 + 5, edad_maxima * 365 - 5)
        persona = Persona(
            referencia=f"{self._prefijo}{len(self.personas) + 1:05d}", nombre=nombre,
            primer_apellido=primero, segundo_apellido=segundo, sexo=sexo, documento=self._documento(),
            fecha_nacimiento=FECHA_REFERENCIA - timedelta(days=edad_dias),
            correo=self._correo(nombre, primero, segundo), telefono=self._telefono(), domicilio=self._domicilio(),
        )
        self.personas.append(persona)
        return persona
