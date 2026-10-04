package postgres

import (
	"bytes"
	"encoding/json"
	"io"
	"maps"
	"reflect"

	"vec-diputacion-granada/internal/vec/ports"
)

// ValidarEmisionUsuariosAdministrables coteja únicamente el formato AUT43.
// El ámbito procede del propio material para cerrar su canon; aquí no se
// acredita actor, asignación, permiso, PDP ni firma. El emisor y AUT43 deben
// ejercer esas autoridades por sus contratos respectivos.
func ValidarEmisionUsuariosAdministrables(e ports.EmisionUsuariosAdministrables) (ports.EmisionUsuariosAdministrables, error) {
	fallo := ports.ErrLecturaUsuariosAdministrablesNoDisponible
	if e.Correlacion.Validar() != nil || len(e.Material) < 2 || len(e.Material) > 4096 {
		return ports.EmisionUsuariosAdministrables{}, fallo
	}
	var p peticion
	var err error
	switch e.Accion {
	case accionListar:
		if e.Audiencia != audienciaListar {
			return ports.EmisionUsuariosAdministrables{}, fallo
		}
		var m materialLista
		if decodificarEmisionCerrada(e.Material, &m) != nil {
			return ports.EmisionUsuariosAdministrables{}, fallo
		}
		p, err = materialListar(ambito{m.OrganizacionRef, m.UnidadRef}, ports.FiltrosUsuariosAdministrables{
			PerfilRef: m.Filtros.PerfilRef, UnidadRef: m.Filtros.UnidadRef, Estado: m.Filtros.Estado, Cursor: m.Cursor})
	case accionConsultar:
		if e.Audiencia != audienciaFicha {
			return ports.EmisionUsuariosAdministrables{}, fallo
		}
		var m materialFicha
		if decodificarEmisionCerrada(e.Material, &m) != nil {
			return ports.EmisionUsuariosAdministrables{}, fallo
		}
		p, err = materialConsultar(ambito{m.OrganizacionRef, m.UnidadRef}, m.PersonaRef)
	default:
		return ports.EmisionUsuariosAdministrables{}, fallo
	}
	if err != nil {
		return ports.EmisionUsuariosAdministrables{}, falloValidacionRedactado(err)
	}
	if !bytes.Equal(e.Material, p.material) || e.Accion != p.accion || e.Audiencia != p.audiencia || !reflect.DeepEqual(e.Recurso, p.recurso) {
		return ports.EmisionUsuariosAdministrables{}, fallo
	}
	huella, err := e.Recurso.HuellaContextoAutorizacionSHA256()
	huellaEsperada, errEsperada := p.recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return ports.EmisionUsuariosAdministrables{}, falloValidacionRedactado(err)
	}
	if errEsperada != nil {
		return ports.EmisionUsuariosAdministrables{}, falloValidacionRedactado(errEsperada)
	}
	if huella != huellaEsperada {
		return ports.EmisionUsuariosAdministrables{}, fallo
	}
	copia := e
	copia.Material = bytes.Clone(e.Material)
	copia.Recurso.Ambitos = maps.Clone(e.Recurso.Ambitos)
	copia.Recurso.Atributos = maps.Clone(e.Recurso.Atributos)
	return copia, nil
}

func decodificarEmisionCerrada(bruto []byte, destino any) error {
	d := json.NewDecoder(bytes.NewReader(bruto))
	d.DisallowUnknownFields()
	if err := d.Decode(destino); err != nil {
		return err
	}
	var sobrante any
	if err := d.Decode(&sobrante); err != io.EOF {
		return ports.ErrLecturaUsuariosAdministrablesNoDisponible
	}
	return nil
}
