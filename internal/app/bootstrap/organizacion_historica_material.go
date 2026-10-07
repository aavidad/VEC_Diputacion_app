package bootstrap

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	personal "vec-diputacion-granada/internal/modules/personal/domain"
)

// DescriptorCapacidadOrganizacionHistoricaV3 contiene las coordenadas públicas
// de derivación de la consulta; no contiene material secreto ni concesiones.
type DescriptorCapacidadOrganizacionHistoricaV3 struct {
	Audiencia, Dominio, Prefijo string
}

func DescriptorCapacidadOrganizacionHistoricaV3Desarrollo() DescriptorCapacidadOrganizacionHistoricaV3 {
	return DescriptorCapacidadOrganizacionHistoricaV3{
		Audiencia: personal.AudienciaConsultaOrganizacionHistorica,
		Dominio:   "vec.personal.organizacion-historica.consulta.desarrollo.capacidad-v3",
		Prefijo:   "clave:capacidad:personal-organizacion-historica-consulta:",
	}
}

func descriptorMaterialOrganizacionHistorica() descriptorMaterialConsumidorV3Desarrollo {
	d := DescriptorCapacidadOrganizacionHistoricaV3Desarrollo()
	return descriptorMaterialConsumidorV3Desarrollo{
		Audiencia: d.Audiencia, Dominio: d.Dominio, Prefijo: d.Prefijo,
		ProveedorNominal: "proveedor-material-personal-organizacion-historica-consulta",
	}
}

// seleccionarMaterialOrganizacionHistorica valida esta extensión sin cambiar
// la selección ni el catálogo de las capacidades ya compuestas. Su error no
// se propaga al arranque de CT ni habilita un consumidor alternativo.
func seleccionarMaterialOrganizacionHistorica(cfg config.Config, previos []descriptorMaterialConsumidorV3Desarrollo) (catalogoMaterialAutorizacionComunDesarrollo, bool, error) {
	activo, err := cfg.OrganizacionHistoricaGobiernoDesarrolloActivo()
	if err != nil || !activo {
		return catalogoMaterialAutorizacionComunDesarrollo{}, false, err
	}
	descriptores := make([]descriptorMaterialConsumidorV3Desarrollo, 0, len(previos)+1)
	descriptores = append(descriptores, previos...)
	descriptores = append(descriptores, descriptorMaterialOrganizacionHistorica())
	catalogo, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo(descriptores)
	if err != nil {
		return catalogoMaterialAutorizacionComunDesarrollo{}, false, err
	}
	return catalogo, true, nil
}

// CapacidadPublicadaOrganizacionHistoricaV3 devuelve sólo la identidad y las
// huellas que conserva el gobierno central. La clave no sale del publicador.
type CapacidadPublicadaOrganizacionHistoricaV3 struct {
	DescriptorCapacidadOrganizacionHistoricaV3
	ClaveID                                 string
	Version, RevisionGobierno, OrdenPuntero uint64
	HuellaGobierno, SHA256, EmisorID        string
	Desde, Hasta                            time.Time
}

// ClaveCapacidadOrganizacionHistoricaV3 conserva la clave propia de OH sólo
// durante la preparación privada. No acredita gobierno ni concede acceso.
type ClaveCapacidadOrganizacionHistoricaV3 struct {
	CapacidadPublicadaOrganizacionHistoricaV3
	secreto []byte
}

func (ClaveCapacidadOrganizacionHistoricaV3) String() string {
	return "[clave Organización histórica privada]"
}
func (c ClaveCapacidadOrganizacionHistoricaV3) GoString() string { return c.String() }
func (ClaveCapacidadOrganizacionHistoricaV3) MarshalJSON() ([]byte, error) {
	return json.Marshal("[clave Organización histórica privada]")
}
func (c *ClaveCapacidadOrganizacionHistoricaV3) CopiarSecreto() []byte {
	if c == nil {
		return nil
	}
	return append([]byte(nil), c.secreto...)
}
func (c *ClaveCapacidadOrganizacionHistoricaV3) Borrar() {
	if c != nil {
		borrarBytes(c.secreto)
		c.secreto = nil
	}
}

func publicarMaterialOrganizacionHistorica(ctx context.Context, gobierno *pgxpool.Pool, material materialAtestacionContratacionTemporalDesarrollo, catalogo catalogoMaterialAutorizacionComunDesarrollo) (CapacidadPublicadaOrganizacionHistoricaV3, error) {
	return publicarMaterialOrganizacionHistoricaCon(material, catalogo, func(m *materialAtestacionContratacionTemporalDesarrollo) error {
		return publicarGobiernoAtestacionContratacionTemporalDesarrollo(ctx, gobierno, m)
	})
}

func publicarMaterialOrganizacionHistoricaCon(material materialAtestacionContratacionTemporalDesarrollo, catalogo catalogoMaterialAutorizacionComunDesarrollo, publicar func(*materialAtestacionContratacionTemporalDesarrollo) error) (CapacidadPublicadaOrganizacionHistoricaV3, error) {
	var vacia CapacidadPublicadaOrganizacionHistoricaV3
	d := descriptorMaterialOrganizacionHistorica()
	seleccionado, ok := catalogo.descriptorPara(d.Audiencia)
	if publicar == nil || !ok || seleccionado != d || !audienciaConsumoGobiernoPostgreSQLContratacionTemporalDesarrolloEsPropia(d.Audiencia) {
		return vacia, errGobiernoPostgreSQLContratacionTemporalDesarrolloIncoherente
	}
	derivado, err := derivarMaterialConsumidorV3Desarrollo(material, d)
	if err != nil {
		return vacia, err
	}
	defer borrarBytes(derivado.claveHMAC)
	if err := publicar(&derivado); err != nil {
		return vacia, err
	}
	return CapacidadPublicadaOrganizacionHistoricaV3{
		DescriptorCapacidadOrganizacionHistoricaV3: DescriptorCapacidadOrganizacionHistoricaV3Desarrollo(),
		ClaveID: derivado.claveHMACID, Version: derivado.claveHMACVersion,
		RevisionGobierno: derivado.claveHMACRevision, OrdenPuntero: derivado.claveHMACOrden,
		HuellaGobierno: derivado.claveHMACHuella, SHA256: derivado.claveHMACSecreto,
		EmisorID: derivado.emisorID, Desde: derivado.validaDesde, Hasta: derivado.validaHasta,
	}, nil
}
