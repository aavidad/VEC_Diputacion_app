#!/usr/bin/env python3
"""Generate VEC Personal/RPT import JSON from the Diputacion RPT PDF."""

from __future__ import annotations

import argparse
import json
import re
import subprocess
from collections import Counter, defaultdict
from datetime import datetime, timezone
from pathlib import Path


DEFAULT_PDF = Path("/home/alberto/Trabajo/nominas/diputacion_salarios_complementos/rpt_diputacion_granada.pdf")
DEFAULT_OUTPUT = Path("config/rpt_positions_import.json")
DEFAULT_EXPECTED_ROWS = 842
DEFAULT_EXPECTED_DOTATIONS = 1714
SOURCE_URL = "https://www.dipgra.es/diputacion/delegaciones/transparencia-recursos-humanos-y-administracion-electronica/recursos-humanos/"

CENTER_RE = re.compile(r"Centro:\s+([0-9A-Z]+)\s+(.+)$")
ROW_RE = re.compile(r"^\s*(\d{1,4})\s+([A-ZÁÉÍÓÚÜÑ0-9][^\n]*?)\s{2,}(.+)$")
AMOUNT_RE = re.compile(r"^\d{1,3}(?:\.\d{3})*,\d{2}\s*€$")
GROUP_RE = re.compile(r"\b(A1/A2/C1|A1/A2|A2/C1|B/C1|C1/C2|A1|A2|B|C1|C2|AP|3|5)\b")


def clean(value: str) -> str:
    return " ".join(str(value or "").split())


def split_fields(value: str) -> list[str]:
    return [clean(item) for item in re.split(r"\s{2,}", value.strip()) if clean(item)]


def field(fields: list[str], index: int) -> str:
    if index < 0 or index >= len(fields):
        return ""
    return fields[index]


def to_int(value: str) -> int:
    try:
        return int(str(value or "").strip())
    except ValueError:
        return 0


def euro_cents(value: str) -> int:
    normalized = (
        str(value or "")
        .replace("€", "")
        .replace(".", "")
        .replace(",", ".")
        .strip()
    )
    try:
        return int(round(float(normalized) * 100))
    except ValueError:
        return 0


def extract_group(value: str) -> str:
    matches = GROUP_RE.findall(value or "")
    return matches[-1] if matches else ""


def administration_and_provision(fields: list[str]) -> tuple[str, str]:
    administration = field(fields, 2)
    provision = field(fields, 3)
    match = re.match(r"^([A-Z])\s+(2A)$", administration)
    if match:
        return match.group(1), match.group(2)
    return administration, provision


def is_continuation(value: str) -> bool:
    value = clean(value)
    if not value:
        return False
    if "Página " in value or "Centro:" in value or value.startswith("RPT Diputación"):
        return False
    return ROW_RE.match(value) is None


def looks_like_observation(value: str) -> bool:
    upper = value.upper()
    return any(token in upper for token in ("EXTINGUIR", "EXP", "ESPECIAL", "FORM"))


def parse_row(line: str) -> dict | None:
    match = ROW_RE.match(line)
    if not match:
        return None

    fields = split_fields(match.group(3))
    if len(fields) < 4:
        return None

    amount_index = next((idx for idx, item in enumerate(fields) if AMOUNT_RE.match(item)), -1)
    administration, provision = administration_and_provision(fields)
    if amount_index == -1:
        return {
            "official_code": match.group(1),
            "name": clean(match.group(2)),
            "dot": to_int(field(fields, 0)),
            "type": field(fields, 1),
            "administration": administration,
            "provision": provision,
            "geo_dispersion": field(fields, len(fields) - 1),
            "raw": clean(match.group(0)),
        }

    cd_index = amount_index - 2
    if cd_index < 0:
        return None

    group_in_provision = extract_group(provision)
    if group_in_provision:
        provision = clean(provision.replace(group_in_provision, "", 1))

    position = {
        "official_code": match.group(1),
        "name": clean(match.group(2)),
        "dot": to_int(field(fields, 0)),
        "type": field(fields, 1),
        "administration": administration,
        "provision": provision,
        "destination_level": to_int(field(fields, cd_index)),
        "specific_kind": field(fields, amount_index - 1),
        "annual_amount_cents": euro_cents(field(fields, amount_index)),
        "gcp_ct_level": field(fields, amount_index + 1),
        "geo_dispersion": field(fields, amount_index + 2),
        "requirements": clean(" ".join(fields[amount_index + 3 :])),
        "raw": clean(line),
    }

    group_index = -1
    for idx in range(3, cd_index):
        group = extract_group(fields[idx])
        if group:
            position["group"] = group
            group_index = idx
            break
    if group_index >= 0:
        position["area"] = clean(" ".join(fields[group_index + 1 : min(group_index + 2, cd_index)]))
        if group_index + 2 < cd_index:
            position["scale"] = fields[group_index + 2]
        if group_index + 3 < cd_index:
            position["category_code"] = clean(" ".join(fields[group_index + 3 : cd_index]))

    return position


def parse_text(text: str) -> list[dict]:
    positions: list[dict] = []
    delegation = ""
    center_code = ""
    center_name = ""

    for page_index, page_text in enumerate(text.split("\f"), start=1):
        lines = page_text.splitlines()
        index = 0
        while index < len(lines):
            line = lines[index].rstrip(" ")
            if line.startswith("RPT Diputación de Granada"):
                delegation = clean(line.replace("RPT Diputación de Granada", "", 1))
                index += 1
                continue
            center_match = CENTER_RE.search(line)
            if center_match:
                center_code = center_match.group(1)
                center_name = clean(center_match.group(2))
                index += 1
                continue

            position = parse_row(line)
            if not position:
                index += 1
                continue

            position["delegation"] = delegation
            position["center_code"] = center_code
            position["center_name"] = center_name
            position["page"] = page_index

            if index + 1 < len(lines):
                next_line = clean(lines[index + 1])
                if is_continuation(next_line):
                    position["category_code"] = clean(f"{position.get('category_code', '')} {next_line}")
                    position["raw"] = clean(f"{position.get('raw', '')} {next_line}")
                    index += 1

            if index + 1 < len(lines):
                next_line = clean(lines[index + 1])
                if is_continuation(next_line) and looks_like_observation(next_line):
                    position["observations"] = clean(f"{position.get('observations', '')} {next_line}")
                    position["raw"] = clean(f"{position.get('raw', '')} {next_line}")
                    index += 1

            positions.append(position)
            index += 1

    return positions


def provision_label(code: str) -> str:
    return {
        "C": "Concurso",
        "L": "Libre designacion",
        "I": "Codigo RPT I",
        "2A": "Codigo RPT 2A",
    }.get(clean(code), clean(code) or "Segun RPT")


def build_import(positions: list[dict], source_pdf: Path) -> dict:
    official_counts = Counter(item["official_code"] for item in positions)
    seen: dict[str, int] = defaultdict(int)
    imported = []

    for item in positions:
        official_code = item["official_code"]
        seen[official_code] += 1
        if official_counts[official_code] > 1:
            center = clean(item.get("center_code")) or "SIN"
            code = f"{official_code}-{center}-{seen[official_code]:03d}"
        else:
            code = official_code

        amount = item.get("annual_amount_cents", 0)
        specific_kind = clean(item.get("specific_kind"))
        specific_complement = specific_kind
        if amount:
            specific_complement = clean(f"{specific_kind} {amount / 100:.2f} EUR")

        observations = [
            clean(item.get("observations")),
            f"Codigo RPT oficial: {official_code}",
            f"Centro: {clean(item.get('center_code'))} {clean(item.get('center_name'))}",
            f"Pagina PDF: {item.get('page', 0)}",
        ]
        if official_counts[official_code] > 1:
            observations.append(f"Clave VEC desduplicada: {code}")

        imported.append(
            {
                "code": code,
                "name": clean(item.get("name")),
                "dot": int(item.get("dot") or 0),
                "type": clean(item.get("type")),
                "administration": clean(item.get("administration")),
                "provision": clean(item.get("provision")),
                "group": clean(item.get("group")),
                "area": clean(item.get("area")),
                "scale": clean(item.get("scale")),
                "category_code": clean(item.get("category_code")),
                "category_slug": "",
                "delegation": clean(item.get("delegation")),
                "center_code": clean(item.get("center_code")),
                "center_name": clean(item.get("center_name")),
                "destination_level": int(item.get("destination_level") or 0),
                "specific_kind": specific_kind,
                "annual_amount_cents": int(amount or 0),
                "gcp_ct_level": clean(item.get("gcp_ct_level")),
                "specific_complement": specific_complement,
                "geo_dispersion": clean(item.get("geo_dispersion")),
                "telework": "Segun RPT",
                "coverage": provision_label(item.get("provision", "")),
                "state": "Vigente RPT 2026",
                "source": f"{source_pdf} | {SOURCE_URL}",
                "requirements": clean(item.get("requirements")),
                "observations": clean(" | ".join(part for part in observations if clean(part))),
                "page": int(item.get("page") or 0),
                "raw": clean(item.get("raw")),
            }
        )

    return {
        "source": str(source_pdf),
        "version": f"rpt-diputacion-granada-2026-05-07-generated-{datetime.now(timezone.utc).date().isoformat()}",
        "replace": True,
        "positions": imported,
    }


def pdftotext(pdf_path: Path) -> str:
    return subprocess.check_output(["pdftotext", "-layout", str(pdf_path), "-"], text=True)


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--pdf", type=Path, default=DEFAULT_PDF)
    parser.add_argument("--out", type=Path, default=DEFAULT_OUTPUT)
    parser.add_argument("--expected-rows", type=int, default=DEFAULT_EXPECTED_ROWS)
    parser.add_argument("--expected-dotations", type=int, default=DEFAULT_EXPECTED_DOTATIONS)
    args = parser.parse_args()

    positions = parse_text(pdftotext(args.pdf))
    dotations = sum(int(item.get("dot") or 0) for item in positions)
    if len(positions) != args.expected_rows:
        raise SystemExit(f"RPT rows = {len(positions)}, expected {args.expected_rows}")
    if dotations != args.expected_dotations:
        raise SystemExit(f"RPT dotations = {dotations}, expected {args.expected_dotations}")

    payload = build_import(positions, args.pdf)
    args.out.parent.mkdir(parents=True, exist_ok=True)
    args.out.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(f"wrote {args.out} with {len(payload['positions'])} positions and {dotations} dotations")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
