package bootstrap

import (
	"context"
	"crypto/sha256"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	personal "vec-diputacion-granada/internal/modules/personal/domain"
)

// El registro B2 de Personal se integra como Cronos y Dietas en el gobierno V3
// único de desarrollo: vec-server es el único publicador y deriva una clave
// HMAC por audiencia de consumo, con dominio y prefijo propios, bajo la misma
// raíz y la misma audiencia de atestación. vec-interno sólo lee ese gobierno;
// nunca publica raíz, configuración ni claves B2.

// DescriptorCapacidadPersonalB2V3 es la declaración pública de una capacidad
// B2: el nombre del campo en `capacidades` del inventario privado de
// vec-interno, la audiencia de consumo, el dominio de derivación y el prefijo
// del identificador de clave. No contiene ni permite obtener secretos.
type DescriptorCapacidadPersonalB2V3 struct {
	Capacidad, Audiencia, Dominio, Prefijo string
}

// DescriptoresCapacidadPersonalB2V3Desarrollo devuelve los ocho descriptores
// en el orden de las capacidades de personal_b2_v3.json (ficha, vacantes,
// alta, hecho, catálogo consultar/publicar/retirar y empleados). Es la
// interfaz que reutiliza la herramienta de composición del material.
func DescriptoresCapacidadPersonalB2V3Desarrollo() [8]DescriptorCapacidadPersonalB2V3 {
	return [8]DescriptorCapacidadPersonalB2V3{
		{Capacidad: "ficha", Audiencia: personal.AudienciaFichaEmpleadoB2, Dominio: "vec.personal.registro-empleado.ficha.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:personal-b2-ficha:"},
		{Capacidad: "vacantes", Audiencia: personal.AudienciaVacantesB2, Dominio: "vec.personal.registro-empleado.vacantes.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:personal-b2-vacantes:"},
		{Capacidad: "alta", Audiencia: personal.AudienciaAltaEmpleadoB2, Dominio: "vec.personal.registro-empleado.alta.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:personal-b2-alta:"},
		{Capacidad: "hecho", Audiencia: personal.AudienciaHechoEmpleadoB2, Dominio: "vec.personal.registro-empleado.hecho.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:personal-b2-hecho:"},
		{Capacidad: "catalogo_consultar", Audiencia: personal.AudienciaConsultarCatalogoEmpleadoB2, Dominio: "vec.personal.registro-empleado.catalogo.consultar.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:personal-b2-catalogo-consultar:"},
		{Capacidad: "catalogo_publicar", Audiencia: personal.AudienciaPublicarCatalogoEmpleadoB2, Dominio: "vec.personal.registro-empleado.catalogo.publicar.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:personal-b2-catalogo-publicar:"},
		{Capacidad: "catalogo_retirar", Audiencia: personal.AudienciaRetirarCatalogoEmpleadoB2, Dominio: "vec.personal.registro-empleado.catalogo.retirar.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:personal-b2-catalogo-retirar:"},
		{Capacidad: "empleados", Audiencia: personal.AudienciaEmpleadosB2, Dominio: "vec.personal.registro-empleado.empleados.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:personal-b2-empleados:"},
	}
}

func descriptoresMaterialPersonalB2Desarrollo() []descriptorMaterialConsumidorV3Desarrollo {
	publicos := DescriptoresCapacidadPersonalB2V3Desarrollo()
	descriptores := make([]descriptorMaterialConsumidorV3Desarrollo, 0, len(publicos))
	for _, d := range publicos {
		descriptores = append(descriptores, descriptorMaterialConsumidorV3Desarrollo{
			Audiencia: d.Audiencia, Dominio: d.Dominio, Prefijo: d.Prefijo,
			ProveedorNominal: "proveedor-material-personal-b2-" + d.Capacidad,
		})
	}
	return descriptores
}

// CapacidadPublicadaPersonalB2V3 son las coordenadas públicas de una clave B2
// publicada. SHA256 es la huella del secreto que ya guarda el gobierno
// (huella_secreto_sha256); ni esta estructura ni su formato exponen la clave.
type CapacidadPublicadaPersonalB2V3 struct {
	DescriptorCapacidadPersonalB2V3
	ClaveID          string
	Version          uint64
	RevisionGobierno uint64
	OrdenPuntero     uint64
	HuellaGobierno   string
	SHA256           string
	EmisorID         string
	Desde, Hasta     time.Time
}

// ClaveCapacidadPersonalB2V3 añade a las coordenadas públicas el secreto
// derivado, sólo para que la herramienta de composición escriba el archivo
// privado de la capacidad. Su formato y su JSON nunca muestran el secreto.
type ClaveCapacidadPersonalB2V3 struct {
	CapacidadPublicadaPersonalB2V3
	secreto []byte
}

func (ClaveCapacidadPersonalB2V3) String() string     { return "[clave Personal B2 privada]" }
func (c ClaveCapacidadPersonalB2V3) GoString() string { return c.String() }
func (ClaveCapacidadPersonalB2V3) MarshalJSON() ([]byte, error) {
	return []byte(`"[clave Personal B2 privada]"`), nil
}

// CopiarSecreto devuelve una copia que el llamante debe borrar tras usarla.
func (c *ClaveCapacidadPersonalB2V3) CopiarSecreto() []byte {
	if c == nil {
		return nil
	}
	return append([]byte(nil), c.secreto...)
}

// Borrar sobrescribe el secreto retenido.
func (c *ClaveCapacidadPersonalB2V3) Borrar() {
	if c != nil {
		borrarBytes(c.secreto)
		c.secreto = nil
	}
}

var errMaterialPersonalB2V3Desarrollo = errors.New("vec: material Personal B2 de desarrollo no disponible")

// DerivarClavesPersonalB2V3Desarrollo aplica la misma y única derivación que
// usa el publicador (derivarMaterialConsumidorV3Desarrollo) a partir de la
// clave base de capacidad CT (`clave:capacidad:ct:...`), que el llamante toma
// de su material privado existente. No consulta ni escribe el gobierno; sirve
// para cotejar el material privado contra lo publicado. Las coordenadas de
// versión, revisión y orden no dependen de la derivación: el llamante las
// toma del gobierno publicado.
func DerivarClavesPersonalB2V3Desarrollo(claveBase []byte, claveBaseID, emisorID string, desde, hasta time.Time) ([8]ClaveCapacidadPersonalB2V3, error) {
	var claves [8]ClaveCapacidadPersonalB2V3
	if len(claveBase) < sha256.Size || claveBaseID == "" || emisorID == "" || !desde.Before(hasta) {
		return claves, errMaterialPersonalB2V3Desarrollo
	}
	base := materialAtestacionContratacionTemporalDesarrollo{
		claveHMACID: claveBaseID, claveHMAC: claveBase, claveHMACVersion: 1, claveHMACRevision: 1,
		emisorID: emisorID, validaDesde: desde.UTC(), validaHasta: hasta.UTC(),
	}
	for i, d := range DescriptoresCapacidadPersonalB2V3Desarrollo() {
		derivado, err := derivarMaterialConsumidorV3Desarrollo(base, descriptorPersonalB2Interno(d))
		if err != nil {
			for j := range claves {
				claves[j].Borrar()
			}
			return [8]ClaveCapacidadPersonalB2V3{}, errMaterialPersonalB2V3Desarrollo
		}
		claves[i] = ClaveCapacidadPersonalB2V3{
			CapacidadPublicadaPersonalB2V3: CapacidadPublicadaPersonalB2V3{
				DescriptorCapacidadPersonalB2V3: d, ClaveID: derivado.claveHMACID,
				HuellaGobierno: derivado.claveHMACHuella, SHA256: derivado.claveHMACSecreto,
				EmisorID: derivado.emisorID, Desde: derivado.validaDesde, Hasta: derivado.validaHasta,
			},
			secreto: derivado.claveHMAC,
		}
	}
	return claves, nil
}

func descriptorPersonalB2Interno(d DescriptorCapacidadPersonalB2V3) descriptorMaterialConsumidorV3Desarrollo {
	return descriptorMaterialConsumidorV3Desarrollo{Audiencia: d.Audiencia, Dominio: d.Dominio, Prefijo: d.Prefijo,
		ProveedorNominal: "proveedor-material-personal-b2-" + d.Capacidad}
}

// publicarMaterialPersonalB2Desarrollo ejecuta la publicación de las ocho
// claves con el publicador existente (misma transacción serializada, mismo
// cerrojo consultivo y mismos actos `acto:ct:desarrollo:`). Es idempotente:
// una clave ya publicada se reconoce y conserva su versión y su puntero. Sólo
// devuelve coordenadas públicas; los secretos derivados se borran al salir.
func publicarMaterialPersonalB2Desarrollo(ctx context.Context, gobierno *pgxpool.Pool, material materialAtestacionContratacionTemporalDesarrollo, catalogo catalogoMaterialAutorizacionComunDesarrollo) ([8]CapacidadPublicadaPersonalB2V3, error) {
	return publicarMaterialPersonalB2ConDesarrollo(material, catalogo, func(m *materialAtestacionContratacionTemporalDesarrollo) error {
		return publicarGobiernoAtestacionContratacionTemporalDesarrollo(ctx, gobierno, m)
	})
}

func publicarMaterialPersonalB2ConDesarrollo(material materialAtestacionContratacionTemporalDesarrollo, catalogo catalogoMaterialAutorizacionComunDesarrollo, publicar func(*materialAtestacionContratacionTemporalDesarrollo) error) ([8]CapacidadPublicadaPersonalB2V3, error) {
	var publicadas [8]CapacidadPublicadaPersonalB2V3
	if publicar == nil {
		return publicadas, errGobiernoPostgreSQLContratacionTemporalDesarrolloIncoherente
	}
	for i, d := range DescriptoresCapacidadPersonalB2V3Desarrollo() {
		descriptor, ok := catalogo.descriptorPara(d.Audiencia)
		if !ok || descriptor != descriptorPersonalB2Interno(d) {
			return [8]CapacidadPublicadaPersonalB2V3{}, errGobiernoPostgreSQLContratacionTemporalDesarrolloIncoherente
		}
		derivado, err := derivarMaterialConsumidorV3Desarrollo(material, descriptor)
		if err != nil {
			return [8]CapacidadPublicadaPersonalB2V3{}, err
		}
		err = publicar(&derivado)
		borrarBytes(derivado.claveHMAC)
		if err != nil {
			return [8]CapacidadPublicadaPersonalB2V3{}, err
		}
		publicadas[i] = CapacidadPublicadaPersonalB2V3{
			DescriptorCapacidadPersonalB2V3: d, ClaveID: derivado.claveHMACID,
			Version: derivado.claveHMACVersion, RevisionGobierno: derivado.claveHMACRevision,
			OrdenPuntero: derivado.claveHMACOrden, HuellaGobierno: derivado.claveHMACHuella,
			SHA256: derivado.claveHMACSecreto, EmisorID: derivado.emisorID,
			Desde: derivado.validaDesde, Hasta: derivado.validaHasta,
		}
	}
	return publicadas, nil
}
