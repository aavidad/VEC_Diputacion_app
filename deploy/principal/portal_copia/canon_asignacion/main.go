package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"time"
	domain "vec-diputacion-granada/internal/vec/domain"
)

func main() {
	if len(os.Args) != 6 {
		os.Exit(2)
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	if now.Nanosecond()%10000 == 0 {
		now = now.Add(time.Microsecond)
	}
	a := domain.AsignacionPerfil{
		AsignacionID: os.Args[5], Version: 1, PerfilActivoRef: os.Args[2],
		PrincipalID: os.Args[1], VersionRolRef: os.Args[4],
		Estado:       domain.EstadoAsignacionPerfilActiva,
		Ambitos:      []domain.AmbitoPerfil{{Clave: "candidato_ref", Valores: []string{os.Args[3]}}},
		VigenteDesde: now, VigenteHasta: now.Add(89 * 24 * time.Hour),
		EmitidaPor: "identidad:desarrollo:no-autoritativa", EmitidaEn: now,
	}
	if a.Validar() != nil {
		os.Exit(2)
	}
	doc, e := json.Marshal(a)
	if e != nil {
		os.Exit(2)
	}
	hash, e := a.HuellaSHA256()
	if e != nil {
		os.Exit(2)
	}
	sum := sha256.Sum256(doc)
	if hex.EncodeToString(sum[:]) != hash {
		os.Exit(2)
	}
	if json.NewEncoder(os.Stdout).Encode(map[string]string{"documento": string(doc), "huella_sha256": hash}) != nil {
		os.Exit(2)
	}
}
