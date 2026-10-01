package ejecucioncopias

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"

	cs06 "vec-diputacion-granada/internal/modules/administracion/domain/contrastecopias"
	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
)

var ErrEvidencia = errors.New("copias_ejecucion_evidencia_no_comprobable")

func huellaCanonica(v any) string {
	b, _ := json.Marshal(v)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// evidenciaSnapshot traduce contenido observado por CS06 al manifiesto CS01.
// Extensiones y privilegios por defecto quedan incluidos en las raíces: omitir
// una clase no puede producir las mismas huellas que un conjunto completo.
func evidenciaSnapshot(snapshot cs06.Snapshot, archivos []copias.Artefacto, ref, arranque string) (copias.Evidencia, error) {
	if len(cs06.Validar(snapshot)) != 0 || ref == "" || len(archivos) == 0 {
		return copias.Evidencia{}, ErrEvidencia
	}
	objetos := append([]cs06.Objeto(nil), snapshot.Objetos...)
	sort.Slice(objetos, func(i, j int) bool {
		if objetos[i].Clase != objetos[j].Clase {
			return objetos[i].Clase < objetos[j].Clase
		}
		return objetos[i].Clave < objetos[j].Clave
	})
	clase := func(clases ...string) string {
		result := make([]cs06.Objeto, 0)
		for _, o := range objetos {
			for _, c := range clases {
				if o.Clase == c {
					result = append(result, o)
				}
			}
		}
		return huellaCanonica(result)
	}
	type cuenta struct {
		Clave    string
		Cantidad int64
	}
	cuentas := make([]cuenta, 0)
	for _, o := range objetos {
		if o.Clase == "tablas" && o.Clave != "inventario" {
			cuentas = append(cuentas, cuenta{o.Clave, o.Cantidad})
		}
	}
	ficheros := append([]copias.Artefacto(nil), archivos...)
	sort.Slice(ficheros, func(i, j int) bool { return ficheros[i].ID < ficheros[j].ID })
	vistos := map[string]bool{}
	for _, a := range ficheros {
		if vistos[a.ID] || a.ID == "" || a.TamanoBytes < 0 || len(a.SHA256) != 64 {
			return copias.Evidencia{}, ErrEvidencia
		}
		if _, err := hex.DecodeString(a.SHA256); err != nil {
			return copias.Evidencia{}, ErrEvidencia
		}
		vistos[a.ID] = true
	}
	return copias.Evidencia{Ref: ref, ArranqueRef: arranque, RecuentosSHA256: huellaCanonica(cuentas), ContenidoSHA256: clase("tablas"), EsquemaSHA256: clase("esquema", "extensiones"), RolesSHA256: clase("roles"), ACLSHA256: clase("acl", "privilegios_defecto"), SecuenciasSHA256: clase("secuencias"), ObjetosGrandesSHA256: clase("objetos_grandes"), FicherosSHA256: huellaCanonica(ficheros)}, nil
}

// PreimagenDeObservacion identifica el conjunto actual completo: contenido de
// PostgreSQL y ficheros, además de versión/esquema. No basta la release para
// detectar filas cambiadas entre propuesta y sustitución.
func PreimagenDeObservacion(i copias.Inventario, e copias.Evidencia) (string, error) {
	if len(copias.ValidarInventario(i)) != 0 {
		return "", ErrEvidencia
	}
	valores := []string{copias.HuellaInventario(i), e.RecuentosSHA256, e.ContenidoSHA256, e.EsquemaSHA256, e.RolesSHA256, e.ACLSHA256, e.SecuenciasSHA256, e.ObjetosGrandesSHA256, e.FicherosSHA256}
	for _, v := range valores {
		if len(v) != 64 {
			return "", ErrEvidencia
		}
		if _, err := hex.DecodeString(v); err != nil {
			return "", ErrEvidencia
		}
	}
	return huellaCanonica(valores), nil
}
