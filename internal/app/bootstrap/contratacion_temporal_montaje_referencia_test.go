package bootstrap

// Referencias ficticias usadas por los contratos de montaje de CT.
const (
	organizacionAltaContratacionTemporalDesarrollo = "organizacion:desarrollo:dipgra"
	unidadCoberturaContratacionTemporalDesarrollo  = "unidad:desarrollo:rrhh"
)

var referenciaOrganizativaCTMontajePrueba = referenciaCT{
	organizacionRef: organizacionAltaContratacionTemporalDesarrollo,
	unidadRef:       unidadCoberturaContratacionTemporalDesarrollo,
	fuenteUnidad: fuenteUnidadReferenciaCT{
		referencia:   "fuente:unidad:rrhh",
		version:      2,
		huellaSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	},
}
