package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func catalogoSAEPrueba() CatalogoOfertaSAE {
	return CatalogoOfertaSAE{
		Referencia: "catalogo:sae:v1", Version: 1, HuellaSHA256: strings.Repeat("a", 64),
		Modalidades: []string{"interinidad"}, CriteriosValoracion: []string{"experiencia"},
		ResultadosValoracion: []string{"apto", "no_apto"}, ResultadosEntrevista: []string{"apto", "no_apto"},
		Plantillas: map[string]string{DocumentoSAESolicitud: "plantilla:solicitud:v1", DocumentoSAEActa: "plantilla:acta:v1", DocumentoSAEResolucion: "plantilla:resolucion:v1"},
	}
}

func ofertaSAEPrueba(t *testing.T) OfertaSAE {
	t.Helper()
	o, err := NuevaOfertaSAE("oferta-sae:001", DatosOfertaSAE{
		CategoriaRef: "categoria:operario", PuestoRef: "puesto:auxiliar", NumeroPlazas: 2,
		Modalidad: "interinidad", Duracion: "Hasta cobertura reglamentaria", Requisitos: "Requisitos según catálogo de la oferta",
		ExpedienteCTRef: "expediente:ct:001",
	}, catalogoSAEPrueba())
	if err != nil {
		t.Fatal(err)
	}
	return o
}

func cambioSAEPrueba(o OfertaSAE, accion, clave string) CambioOfertaSAE {
	return CambioOfertaSAE{Accion: accion, Clave: clave, ReciboRef: "recibo:" + clave,
		ActorRef: "persona:rrhh:001", VersionEsperada: o.Version,
		Instante: time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)}
}

func aplicarSAEPrueba(t *testing.T, o OfertaSAE, c CambioOfertaSAE) OfertaSAE {
	t.Helper()
	siguiente, recibo, reutilizado, err := o.Aplicar(c)
	if err != nil || recibo != c.ReciboRef || reutilizado || siguiente.Version != o.Version+1 {
		t.Fatalf("cambio %s: estado=%+v recibo=%q replay=%v error=%v", c.Accion, siguiente, recibo, reutilizado, err)
	}
	return siguiente
}

func TestOfertaSAERecorridoManualExternoYReplay(t *testing.T) {
	o := ofertaSAEPrueba(t)
	if o.Estado != EstadoSAEPreparada || o.Version != 1 || o.Catalogo.Version != 1 {
		t.Fatalf("alta: %+v", o)
	}
	envio := cambioSAEPrueba(o, AccionSAEEnviar, "clave-envio-0001")
	envio.NumeroSAE, envio.FechaEnvio = "SAE/2026/001", "2026-09-28"
	o = aplicarSAEPrueba(t, o, envio)
	if o.Estado != EstadoSAEEnviada || o.NumeroSAE != envio.NumeroSAE {
		t.Fatalf("envío manual: %+v", o)
	}
	candidato := CandidatoOfertaSAE{Referencia: "candidato:externo:001", NombreProtegidoRef: "dato:nombre:001", DocumentoProtegidoRef: "dato:documento:001", ContactoProtegidoRef: "dato:contacto:001"}
	registro := cambioSAEPrueba(o, AccionSAERegistrarCandidato, "clave-candidato-0001")
	registro.Candidato = &candidato // Sin empleado ni persona previa: es aspirante externo válido.
	o = aplicarSAEPrueba(t, o, registro)
	o = aplicarSAEPrueba(t, o, cambioSAEPrueba(o, AccionSAERecibirCandidatos, "clave-recepcion-0001"))
	o = aplicarSAEPrueba(t, o, cambioSAEPrueba(o, AccionSAEIniciarSeleccion, "clave-seleccion-0001"))
	valoracion := cambioSAEPrueba(o, AccionSAEValorar, "clave-valoracion-0001")
	valoracion.Valoracion = &ValoracionOfertaSAE{CandidatoRef: candidato.Referencia, Orden: 1, EntrevistaResultado: "apto", Criterios: []ResultadoCriterioSAE{{Criterio: "experiencia", Resultado: "apto"}}}
	o = aplicarSAEPrueba(t, o, valoracion)
	previa := o
	resolucion := cambioSAEPrueba(o, AccionSAEResolver, "clave-resolucion-0001")
	o = aplicarSAEPrueba(t, o, resolucion)
	if o.Estado != EstadoSAEResuelta || len(o.Actuaciones) != 6 || previa.Estado != EstadoSAEEnSeleccion {
		t.Fatalf("resolución o copia: %+v", o)
	}
	repetida, recibo, reutilizado, err := o.Aplicar(registro)
	if err != nil || !reutilizado || recibo != registro.ReciboRef || repetida.Version != o.Version || len(repetida.Candidatos) != 1 {
		t.Fatalf("replay: %q %v %v", recibo, reutilizado, err)
	}
	registro.Candidato = &CandidatoOfertaSAE{Referencia: "candidato:otro", NombreProtegidoRef: "dato:nombre:002", DocumentoProtegidoRef: "dato:documento:002", ContactoProtegidoRef: "dato:contacto:002"}
	if _, _, _, err = o.Aplicar(registro); !errors.Is(err, ErrOfertaSAEClaveReutilizada) {
		t.Fatalf("clave reutilizada: %v", err)
	}
}

func TestOfertaSAECierraSiSeleccionManualDeclaraDesierta(t *testing.T) {
	o := ofertaSAEPrueba(t)
	envio := cambioSAEPrueba(o, AccionSAEEnviar, "clave-envio-0002")
	envio.NumeroSAE, envio.FechaEnvio = "SAE/2026/002", "2026-09-28"
	o = aplicarSAEPrueba(t, o, envio)
	registro := cambioSAEPrueba(o, AccionSAERegistrarCandidato, "clave-candidato-0002")
	registro.Candidato = &CandidatoOfertaSAE{Referencia: "candidato:externo:002", NombreProtegidoRef: "dato:nombre:002", DocumentoProtegidoRef: "dato:documento:002", ContactoProtegidoRef: "dato:contacto:002"}
	o = aplicarSAEPrueba(t, o, registro)
	o = aplicarSAEPrueba(t, o, cambioSAEPrueba(o, AccionSAERecibirCandidatos, "clave-recepcion-0002"))
	o = aplicarSAEPrueba(t, o, cambioSAEPrueba(o, AccionSAEIniciarSeleccion, "clave-seleccion-0002"))
	valoracion := cambioSAEPrueba(o, AccionSAEValorar, "clave-valoracion-0002")
	valoracion.Valoracion = &ValoracionOfertaSAE{CandidatoRef: registro.Candidato.Referencia, Orden: 1, EntrevistaResultado: "no_apto", Criterios: []ResultadoCriterioSAE{{Criterio: "experiencia", Resultado: "no_apto"}}}
	o = aplicarSAEPrueba(t, o, valoracion)
	o = aplicarSAEPrueba(t, o, cambioSAEPrueba(o, AccionSAEDeclararDesierta, "clave-desierta-0002"))
	if o.Estado != EstadoSAEDesierta {
		t.Fatalf("cierre: %s", o.Estado)
	}
}

func TestOfertaSAEDeniegaReglasAusentesYVersionObsoleta(t *testing.T) {
	c := catalogoSAEPrueba()
	c.Modalidades = nil
	if _, err := NuevaOfertaSAE("oferta-sae:001", ofertaSAEPrueba(t).Datos, c); !errors.Is(err, ErrOfertaSAEInvalida) {
		t.Fatalf("catálogo vacío: %v", err)
	}
	o := ofertaSAEPrueba(t)
	if _, _, _, err := o.Aplicar(cambioSAEPrueba(o, AccionSAERecibirCandidatos, "clave-anticipada-0001")); !errors.Is(err, ErrOfertaSAETransicion) {
		t.Fatalf("salto de fase: %v", err)
	}
	envio := cambioSAEPrueba(o, AccionSAEEnviar, "clave-envio-0003")
	envio.NumeroSAE, envio.FechaEnvio = "SAE/2026/003", "2026-09-28"
	o = aplicarSAEPrueba(t, o, envio)
	registro := cambioSAEPrueba(o, AccionSAERegistrarCandidato, "clave-candidato-0003")
	registro.Candidato = &CandidatoOfertaSAE{Referencia: "candidato:externo:003", NombreProtegidoRef: "dato:nombre:003", DocumentoProtegidoRef: "dato:documento:003", ContactoProtegidoRef: "dato:contacto:003"}
	registro.VersionEsperada--
	if _, _, _, err := o.Aplicar(registro); !errors.Is(err, ErrOfertaSAEVersion) {
		t.Fatalf("versión obsoleta: %v", err)
	}
	registro.VersionEsperada = o.Version
	registro.Candidato.DocumentoProtegidoRef = ""
	if _, _, _, err := o.Aplicar(registro); !errors.Is(err, ErrOfertaSAETransicion) {
		t.Fatalf("documento sin protección: %v", err)
	}
	envio.NumeroSAE = "SAE/2026/003\nOtra línea"
	if _, _, _, err := ofertaSAEPrueba(t).Aplicar(envio); !errors.Is(err, ErrOfertaSAETransicion) {
		t.Fatalf("número con salto de línea: %v", err)
	}
}
