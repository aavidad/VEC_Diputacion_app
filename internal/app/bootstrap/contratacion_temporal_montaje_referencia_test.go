package bootstrap

// Referencias ficticias usadas por los contratos de montaje de CT.
const (
	organizacionAltaContratacionTemporalDesarrollo = "organizacion:desarrollo:dipgra"
	unidadCoberturaContratacionTemporalDesarrollo  = "unidad:desarrollo:rrhh"
)

var referenciaOrganizativaCTMontajePrueba = referenciaCT{
	organizacionRef: organizacionAltaContratacionTemporalDesarrollo,
	unidadRef:       unidadCoberturaContratacionTemporalDesarrollo,
}
