package postgres

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const esquemaPaqueteCatalogoAccionesV2 = "vec.admin.catalogo-acciones.paquete.v2"

// entradaFuenteCanonicaV2 fija la proyeccion byte a byte de una entrada. Solo
// se excluye la huella que depende de esta misma representacion.
type entradaFuenteCanonicaV2 struct {
	Referencia        string              `json:"referencia"`
	Version           int                 `json:"version"`
	FuenteRef         string              `json:"fuente_ref"`
	FuenteVersion     int                 `json:"fuente_version"`
	Concesion         domain.ConcesionRol `json:"concesion"`
	DimensionesAmbito []string            `json:"dimensiones_ambito"`
	ClaseControl      string              `json:"clase_control"`
	VigenteDesde      time.Time           `json:"vigente_desde"`
	VigenteHasta      time.Time           `json:"vigente_hasta,omitempty"`
}

type fuentePaqueteCatalogoAccionesV2 struct {
	ModuloID      string                                 `json:"modulo_id"`
	Referencia    string                                 `json:"referencia"`
	Version       int                                    `json:"version"`
	HuellaSHA256  string                                 `json:"huella_sha256"`
	EntradasCanon string                                 `json:"entradas_canon"`
	Entradas      []domain.EntradaAccionAdministracionV1 `json:"entradas"`
}

type paqueteCatalogoAccionesV2 struct {
	Esquema    string                                   `json:"esquema"`
	Referencia string                                   `json:"referencia"`
	Version    int                                      `json:"version"`
	Fuentes    []fuentePaqueteCatalogoAccionesV2        `json:"fuentes"`
	Perfiles   []domain.PerfilPublicadoAdministracionV1 `json:"perfiles"`
}

func canonEntradasFuenteV2(entradas []domain.EntradaAccionAdministracionV1) ([]byte, string, error) {
	if len(entradas) == 0 || len(entradas) > 512 {
		return nil, "", ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	proyeccion := make([]entradaFuenteCanonicaV2, len(entradas))
	for i, e := range entradas {
		if err := e.Validar(); err != nil {
			return nil, "", errors.Join(ports.ErrAutoridadAdministracionPerfilesNoDisponible, err)
		}
		proyeccion[i] = entradaFuenteCanonicaV2{
			Referencia: e.Referencia, Version: e.Version, FuenteRef: e.FuenteRef,
			FuenteVersion: e.FuenteVersion, Concesion: e.Concesion,
			DimensionesAmbito: e.DimensionesAmbito, ClaseControl: e.ClaseControl,
			VigenteDesde: e.VigenteDesde, VigenteHasta: e.VigenteHasta,
		}
	}
	canon, err := json.Marshal(proyeccion)
	if err != nil {
		return nil, "", errors.Join(ports.ErrAutoridadAdministracionPerfilesNoDisponible, err)
	}
	h := sha256.Sum256(canon)
	return canon, hex.EncodeToString(h[:]), nil
}

// ValidarPaqueteCatalogoAccionesV2 verifica la autoria de cada fuente y el
// censo exacto. Es una comprobacion de integridad; la admision y la aprobacion
// solo pueden acreditarse mediante la funcion SQL nominal.
func ValidarPaqueteCatalogoAccionesV2(canon []byte, catalogo domain.CatalogoAccionesAdministracionV1,
	ref string, version int, huella string) error {
	denegado := ports.ErrAutoridadAdministracionPerfilesNoDisponible
	if len(canon) == 0 || len(canon) > maximoBytesCatalogoAcciones ||
		ref == "" || version < 1 || len(huella) != 64 {
		return denegado
	}
	if err := catalogo.Validar(); err != nil {
		return errors.Join(denegado, err)
	}
	var paquete paqueteCatalogoAccionesV2
	dec := json.NewDecoder(bytes.NewReader(canon))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&paquete); err != nil {
		return errors.Join(denegado, err)
	}
	if err := dec.Decode(new(any)); !errors.Is(err, io.EOF) {
		if err != nil {
			return errors.Join(denegado, err)
		}
		return denegado
	}
	recodificado, err := json.Marshal(paquete)
	if err != nil {
		return errors.Join(denegado, err)
	}
	if !bytes.Equal(canon, recodificado) || paquete.Esquema != esquemaPaqueteCatalogoAccionesV2 ||
		paquete.Referencia != ref || paquete.Version != version ||
		len(paquete.Fuentes) == 0 || len(paquete.Fuentes) > 512 ||
		len(paquete.Perfiles) == 0 || len(paquete.Perfiles) > 512 {
		return denegado
	}
	suma := sha256.Sum256(canon)
	if hex.EncodeToString(suma[:]) != huella ||
		catalogo.FuenteRef != ref || catalogo.FuenteVersion != version ||
		catalogo.FuenteHuellaSHA256 != huella || len(paquete.Perfiles) != len(catalogo.Perfiles) {
		return denegado
	}
	if !reflect.DeepEqual(paquete.Perfiles, catalogo.Perfiles) {
		return denegado
	}
	indice := 0
	identidades := make(map[struct {
		modulo, referencia string
		version            int
	}]bool, len(paquete.Fuentes))
	for _, fuente := range paquete.Fuentes {
		identidad := struct {
			modulo, referencia string
			version            int
		}{fuente.ModuloID, fuente.Referencia, fuente.Version}
		if fuente.ModuloID == "" || fuente.Referencia == "" || fuente.Version < 1 ||
			len(fuente.HuellaSHA256) != 64 || len(fuente.EntradasCanon) == 0 || identidades[identidad] {
			return denegado
		}
		identidades[identidad] = true
		entradasCanon, h, err := canonEntradasFuenteV2(fuente.Entradas)
		if err != nil {
			return errors.Join(denegado, err)
		}
		if fuente.EntradasCanon != string(entradasCanon) || fuente.HuellaSHA256 != h {
			return denegado
		}
		for _, entrada := range fuente.Entradas {
			if indice >= len(catalogo.Entradas) || !reflect.DeepEqual(entrada, catalogo.Entradas[indice]) ||
				entrada.Concesion.ModuloID != fuente.ModuloID ||
				entrada.FuenteRef != fuente.Referencia || entrada.FuenteVersion != fuente.Version ||
				entrada.FuenteHuellaSHA256 != fuente.HuellaSHA256 {
				return denegado
			}
			indice++
		}
	}
	if indice != len(catalogo.Entradas) {
		return denegado
	}
	return nil
}
