package bootstrap

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const rutaPublicacionesPropuestaDesarrollo = "web/static/portal-empleado/modulos/contratacion-temporal/formalizacion-desarrollo.json"

// El navegador y la composición leen el mismo activo público. PostgreSQL
// coteja además cada publicación con el contenido inmutable instalado: este
// fichero no concede permisos, aprueba un modelo oficial ni habilita firma.
type publicacionesPropuestaDesarrollo map[string]ports.SnapshotGobernadoFormalizacion

func cargarPublicacionesPropuestaDesarrollo() (publicacionesPropuestaDesarrollo, error) {
	for _, ruta := range []string{rutaPublicacionesPropuestaDesarrollo, "../../../" + rutaPublicacionesPropuestaDesarrollo} {
		f, err := os.Open(ruta)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		p, err := leerPublicacionesPropuestaDesarrollo(f)
		_ = f.Close()
		return p, err
	}
	return nil, ports.ErrResultadoPropuestaFormalizacionNoConfiable
}

func leerPublicacionesPropuestaDesarrollo(r io.Reader) (publicacionesPropuestaDesarrollo, error) {
	var entrada struct {
		Esquema       string `json:"esquema"`
		Publicaciones map[string]struct {
			Referencia string `json:"referencia"`
			Version    uint64 `json:"version"`
			Contenido  string `json:"contenido"`
		} `json:"publicaciones"`
	}
	d := json.NewDecoder(io.LimitReader(r, 16385))
	d.DisallowUnknownFields()
	if d.Decode(&entrada) != nil || d.Decode(new(any)) != io.EOF ||
		entrada.Esquema != "vec.ct.propuesta.publicaciones-desarrollo.v1" || len(entrada.Publicaciones) != 4 {
		return nil, ports.ErrResultadoPropuestaFormalizacionNoConfiable
	}
	refs := map[string]string{
		"tipo_formalizacion": "tipo:ct:propuesta-desarrollo:20260906",
		"plantilla":          "plantilla:ct:propuesta-desarrollo:20260906",
		"politica_firma":     "politica:ct:firma-pendiente:20260906",
		"plan_firma":         "plan:ct:firma-pendiente:20260906",
	}
	resultado := make(publicacionesPropuestaDesarrollo, 4)
	for k, ref := range refs {
		p, ok := entrada.Publicaciones[k]
		if !ok || p.Referencia != ref || p.Version != 1 || len(p.Contenido) == 0 || len(p.Contenido) > 2048 {
			return nil, ports.ErrResultadoPropuestaFormalizacionNoConfiable
		}
		h := sha256.Sum256([]byte(p.Contenido))
		resultado[k] = ports.SnapshotGobernadoFormalizacion{Referencia: ref, Version: 1, HuellaSHA256: hex.EncodeToString(h[:])}
	}
	return resultado, nil
}

func (p publicacionesPropuestaDesarrollo) admite(s ports.SolicitudPropuestaFormalizacion) bool {
	return len(p) == 4 && s.Validar() == nil && s.VersionEsperada == 6 && len(s.Anexos) == 0 &&
		s.TipoFormalizacion == p["tipo_formalizacion"] && s.Plantilla == p["plantilla"] &&
		s.PoliticaFirma == p["politica_firma"] && s.PlanFirma == p["plan_firma"]
}
